# Ingestbus Classification Improvements

**Goal:** raise accuracy of type classification (task/event/note) and context assignment during ingestion, and make `contextkind.List` reachable from the ingestion path. Two tracks, sequenced: (1) build an eval harness to measure before we change anything, (2) tune prompts/schema/heuristic against the harness.

**Root cause analysis (from codebase review):**

1. `business/domain/ingestbus/extractor/prompt.go` has per-type prompts that say "This IS a task/event/note. Do not reclassify." The heuristic `classify.Classify()` is therefore authoritative, and any miscall locks Claude in. Example failure: "get rid of cooking oil" — the heuristic's obligation-verb regex doesn't include "get rid of", so it falls through to `NoteType`, and `buildNoteExtractionPrompt` tells Claude it IS a note.
2. `contextkind.{Project,Area,List}` is defined but never mentioned in any extractor prompt. Claude can suggest new contexts by title only — no `kind` field in the output schema — so every suggested context defaults to whatever the backend assumes on write (likely `Project` or `Area`). `List` is functionally dead on the ingestion path.
3. `suggested_context_id` / `context_confidence` come from Claude but there's no guidance on what makes a good match. No active-context descriptions or kinds are exposed in the prompt — just whatever JSON is serialized.
4. No eval fixtures, no accuracy metric, no regression test. Every prompt change today is a guess.

---

## Track 1 — Eval harness

### 1.1 Fixture format

Create `business/domain/ingestbus/eval/testdata/classification/*.json`. One file per case. Schema:

```json
{
  "name": "cooking-oil-disposal",
  "source": "voice",
  "input_text": "I need to get rid of the old cooking oil under the sink",
  "active_contexts": [
    {"id": "ctx-01", "title": "Kitchen cleanout", "kind": "list"},
    {"id": "ctx-02", "title": "Home maintenance", "kind": "area"}
  ],
  "now": "2026-04-22T10:00:00Z",
  "timezone": "America/Chicago",
  "expected": {
    "primary_type": "task",
    "title_contains": ["cooking oil"],
    "context_id": "ctx-01",
    "context_kind": "list",
    "min_action_items": 1,
    "max_action_items": 1,
    "forbid_notes": true,
    "forbid_events": true
  }
}
```

`expected` fields are all optional — only assert on what's specified. `context_id` can also be `"new"` to mean "should have suggested a new context" with optional `new_context_kind`.

### 1.2 Seed fixtures (10–15 cases)

Mine failures from the user's actual `raw_inputs` table and from the bugs they described. At minimum include:

- **task-miscalled-as-note (imperative):** "get rid of cooking oil" — current bug
- **task-miscalled-as-note (verb form):** "clean out the garage this weekend"
- **task-as-note (colloquial):** "I should probably book the oil change"
- **list-item-assignment:** an item that belongs in an existing list-kind context
- **list-kind-new-context:** multi-item input that should produce a new list-kind context (e.g. "things to pack for the trip: passport, chargers, meds")
- **event-clear:** "dentist at 2pm Thursday" — positive regression test
- **note-clear:** "my PT's phone number is 555-1234" — positive regression test
- **context-ambiguous:** input that could fit two contexts; assert `context_confidence < 0.7`
- **multi-item-task:** "buy milk, call mom, email bob" — assert 3 action_items
- **task-vs-event-boundary:** "call dentist Thursday" (task with date) vs "dentist Thursday at 2pm" (event)
- **receipt / transaction paths:** 1–2 cases so we don't regress those

Each fixture gets a one-line comment in the JSON describing why it's included.

### 1.3 Harness package

```
business/domain/ingestbus/eval/
  eval.go          # Fixture type, loader, runner
  eval_test.go     # TestClassificationEval — gated by EVAL_SIDECAR_URL env var
  metrics.go       # Accuracy computation
  metrics_test.go  # Unit tests on the metrics functions with synthetic results
  testdata/classification/*.json
```

- `eval.Run(ctx, extractor, fixtures)` — runs each fixture through `extractor.ExtractText` (the same interface the real ingest pipeline uses), captures result + escalation path + latency.
- `eval.Score(results)` returns `Metrics{TypeAccuracy, ContextAccuracy, ListAssignmentRate, EscalationRate, PerFixture []FixtureResult}`.
- `eval_test.go` skips with `t.Skip` if `EVAL_SIDECAR_URL` is empty. This keeps it out of `make test`.
- Output: human-readable summary to stdout + JSONL to `.tmp/eval-<timestamp>.jsonl` for diffing.

### 1.4 Makefile target

```
eval-classification:
	EVAL_SIDECAR_URL=$${EVAL_SIDECAR_URL:-http://localhost:8081} \
	go test ./business/domain/ingestbus/eval/... -run TestClassificationEval -count=1 -v
```

Don't wire this into `make test` or CI — it hits a real LLM and is non-deterministic. Run it manually before/after prompt changes.

### 1.5 Baseline capture

Run the harness once before Track 2 and save the raw output to `.docs/evals/2026-04-22-classification-baseline.md` (human-readable summary: pass rate, per-fixture outcomes, escalation rate, notable failure modes). This becomes the "before" we measure improvement against.

### 1.6 Tests for the harness itself

- `eval_test.go` — unit test with a mock extractor proves the runner correctly interprets all `expected.*` assertion fields (happy path + each failure mode).
- `metrics_test.go` — given a synthetic `[]FixtureResult`, metrics computation produces expected numbers.

---

## Track 2 — Prompt & schema tuning

Run after Track 1 is merged and baseline captured. Re-run eval after each change.

### 2.1 Soften the "do not reclassify" lock

In `prompt.go`, the per-type prompts (`buildTaskExtractionPrompt`, `buildEventExtractionPrompt`, `buildNoteExtractionPrompt`) currently say *"This IS a task. Do not reclassify."* Change to: *"A heuristic classifier suggested this is likely a task (confidence: {N}). If that looks wrong, override it — set `reclassified_as` in the output and extract as the correct type."*

Add a new field to the extraction schema:

```
"reclassified_as": "task|event|note|null"  // non-null only when overriding the hint
```

Teach `ingestbus.go:882` (where `typeHint` is passed in) to read `reclassified_as` from the response and route through the correct downstream path. If reclassified, don't create the original-type entity.

### 2.2 Pass heuristic confidence into the prompt

Current: we pass `string(cl.Type)` as `typeHint`. Change `BuildTextExtractionPrompt` to accept `typeHint` + `typeHintConfidence float64`. When confidence < 0.5, use `buildGenericTextExtractionPrompt` regardless of the type (let Claude decide fresh). When confidence >= 0.5, use the type-specific prompt but include the confidence number.

`ingestbus.go` must pass `cl.Confidence` alongside the type. Wire it through `extractor.ExtractText`'s interface.

### 2.3 Tighten the task heuristic

`business/domain/ingestbus/classify/classifier.go` — add imperative patterns that currently slip through:

- "get rid of X"
- "clean out / clean up X"
- "throw away / toss X"
- "need to [verb]"
- "should [verb]" (with caveat — often weak, boost only if combined with object)

Add a `classifier_test.go` case per pattern.

### 2.4 Teach extractor about context kinds

In `buildGenericTextExtractionPrompt` and each per-type prompt:

- When serializing `contextsJSON`, include each context's `kind` field (today we likely only include id+title — verify in `ingestbus.go` where contexts are marshaled and expand the struct if needed).
- Add a new output field: `suggested_new_context_kind: "project|area|list|null"` — non-null only when `suggest_new_context` is true.
- Add prompt guidance:
  > Contexts come in three kinds. **Project** = a goal with a clear end state (e.g. "Launch the blog"). **Area** = an ongoing responsibility with no end (e.g. "Home maintenance", "Personal finance"). **List** = a collection of related items that share a purpose but aren't sequenced work (e.g. "Shopping", "Books to read", "Movies to watch"). If this input is a single item that belongs to an obvious collection, prefer assigning to an existing **list**-kind context or suggesting a new one. Imperatives like "add X", "remember to buy Y", "we need more Z" are strong list-item signals.

- Wire `suggested_new_context_kind` through the extractor result type and into the downstream code that actually creates the context (grep for where `suggest_new_context` is consumed — likely `ingestbus.go` or `contextapp`).

### 2.5 Few-shot examples

In `buildGenericTextExtractionPrompt`, replace the single-line `Examples:` block with a short `## Examples` block containing 4–6 worked cases covering: imperative task, event with time, note, list-item assignment, multi-item expansion, ambiguous reclassification. Keep it under ~400 tokens — don't bloat every request.

### 2.6 Lower escalation threshold for type disagreement

`business/domain/ingestbus/extractor/router.go` (or wherever `shouldEscalate` is defined for text extraction): add a trigger — if `reclassified_as` is non-null, escalate one tier (haiku → sonnet). Rationale: the model corrected the heuristic, which is a signal of ambiguity; spend more on it.

### 2.7 Tests

- Extend `extractor/prompt_test.go` with assertions that new prompts contain the context-kind explanation, the reclassification instruction, and reference the confidence number.
- Update `ingestbus_test.go` for the new `typeHintConfidence` parameter path.
- Add a classifier test for each new imperative pattern.
- Mock-extractor test: when extractor returns `reclassified_as: "task"` while `typeHint` was `"note"`, `ingestbus` creates a task and not a note.
- Mock-extractor test: when extractor returns `suggested_new_context_kind: "list"`, a list-kind context is created.

### 2.8 Re-run eval + write delta

After Track 2 lands, re-run `make eval-classification` and write `.docs/evals/2026-04-22-classification-after.md` with before/after numbers and per-fixture deltas. If any baseline-passing fixture regressed, fix before merging.

---

## Files to touch

### Track 1 (new package, no modifications to existing code)

| Action | File | Why |
|--------|------|-----|
| CREATE | `business/domain/ingestbus/eval/eval.go` | Fixture type, loader, runner |
| CREATE | `business/domain/ingestbus/eval/eval_test.go` | `TestClassificationEval` + mock-extractor unit tests for assertions |
| CREATE | `business/domain/ingestbus/eval/metrics.go` | Score computation |
| CREATE | `business/domain/ingestbus/eval/metrics_test.go` | Synthetic unit tests for scoring |
| CREATE | `business/domain/ingestbus/eval/testdata/classification/*.json` | 10–15 seed fixtures |
| MODIFY | `Makefile` | Add `eval-classification` target |
| CREATE | `.docs/evals/2026-04-22-classification-baseline.md` | Captured baseline (after first run) |

### Track 2

| Action | File | Why |
|--------|------|-----|
| MODIFY | `business/domain/ingestbus/extractor/prompt.go` | Soften reclassification lock, add context-kind guidance, few-shot, confidence in hint |
| MODIFY | `business/domain/ingestbus/extractor/claudecli.go` | Add `reclassified_as` + `suggested_new_context_kind` fields to JSON schema; update response struct |
| MODIFY | `business/domain/ingestbus/extractor/model.go` | Add `typeHintConfidence` to `ExtractText` signature; extend result struct |
| MODIFY | `business/domain/ingestbus/extractor/router.go` | Escalate when `reclassified_as` is non-null |
| MODIFY | `business/domain/ingestbus/extractor/ollama.go`, `mock.go`, `failover.go` | Match new `ExtractText` signature |
| MODIFY | `business/domain/ingestbus/ingestbus.go` | Pass heuristic confidence to extractor; handle `reclassified_as` + `suggested_new_context_kind`; include context `kind` when serializing active contexts |
| MODIFY | `business/domain/ingestbus/classify/classifier.go` | Broader imperative patterns |
| MODIFY | `business/domain/ingestbus/classify/classifier_test.go` | New test cases |
| MODIFY | `business/domain/ingestbus/extractor/prompt_test.go` | Assert new prompt content |
| MODIFY | `business/domain/ingestbus/extractor/claudecli_prompt_test.go` | Cover new schema fields |
| MODIFY | `business/domain/ingestbus/ingestbus_test.go` | Reclassification + list-context flows |
| CREATE | `.docs/evals/2026-04-22-classification-after.md` | Post-change delta |

---

## Cascade rules that apply

- Changing the `Extractor` interface (new param, new result field): update **all** implementations — `claudecli.go`, `ollama.go`, `failover.go`, `mock.go` — and every caller. Compile will catch the first but tests are the real check.
- New `reclassified_as` field + downstream routing: `ingestbus.go` decides what entity to create; do not let both the original-type and the reclassified-type entity get created for the same clause.
- `suggested_new_context_kind` must flow to wherever contexts are auto-created. Find the site by grepping for `suggest_new_context` in `ingestbus.go`.
- No DB migration needed — `contextkind.List` already exists. Track 2 only changes how it gets populated.

---

## Gotchas

- `ingestbus.ProcessRawInputByID` does **not** call `MarkFailed` on error (per memory `async-ingest-design-...`). Reclassification must not throw errors — if the new type fails validation, log and fall back to the original hint rather than crashing the pipeline.
- `sqldb.ErrDBNotFound` must be checked explicitly at every store call (project convention).
- DB port is 5433 in local dev.
- Sidecar uses `--output-format json`, so `claudecli.runHTTP` unwraps two envelopes (per CLAUDE.md sidecar notes). Any schema additions must round-trip through both.
- The eval harness will consume real LLM tokens. Seed with ≤15 fixtures and don't run in CI.

---

## Sequencing / beads

Proposed issues:

1. **parent (feature):** "Improve ingestbus classification accuracy"
2. **child (task):** "Build classification eval harness" — Track 1 (§1.1–1.6)
3. **child (task):** "Capture baseline eval run" — depends on #2
4. **child (task):** "Prompt + schema: reclassification override + confidence pass-through" — Track 2 §2.1–2.2, §2.5, depends on #3
5. **child (task):** "Classifier: broaden imperative patterns" — §2.3, depends on #3 (can run parallel to #4 — different files)
6. **child (task):** "Prompt + schema: context-kind awareness (list-kind surfacing)" — §2.4, depends on #4 (same files)
7. **child (task):** "Escalation tweak + eval delta run" — §2.6 + §2.8, depends on #4, #5, #6

Each child's `--context` should link back to this plan + its relevant section heading.

---

## Open questions / deferred

- **Feedback loop from `classificationcorrectionbus`** — could feed recent user corrections as few-shot examples in the prompt. Deferred: not needed for initial accuracy gain; adds prompt-size + staleness concerns.
- **Per-user prompt customization** — out of scope; single-user app.
- **Persist eval runs to DB for trend tracking** — nice-to-have; JSONL + committed markdown is enough for now.
