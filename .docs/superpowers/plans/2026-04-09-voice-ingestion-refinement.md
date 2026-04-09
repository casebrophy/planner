# Voice Ingestion Pipeline Refinement

## Problem

The voice ingestion pipeline degrades over time — it misclassifies tasks as notes, puts action-oriented utterances into the wrong bucket, and produces irrelevant output. The root causes are:

1. No pre-processing: Claude sees raw transcription noise (fillers, run-ons, compound intents)
2. No task-first bias: tasks, events, and notes are treated as equally likely
3. Rule-based classification is missing: Claude decides type AND extracts fields in one pass — coupling two decisions that should be separate
4. No correction feedback loop: misclassifications have no path back to improving future accuracy
5. Reprocess is overloaded: it conflates "retry a failed pipeline" with "fix a wrong classification"

---

## Design

### Two-stage pipeline (replaces current single-stage)

```
raw voice text
      ↓
[1] Cleanup: strip fillers, split into clauses
      ↓
[2] Rule-based classifier (per clause)
      ├── high confidence → type set, proceed
      └── low confidence → optimistic creation (most likely type) + unconfirmed flag + clarification card
      ↓
[3] Claude extraction (always, per clause)
      - Receives type hint: "this is a task"
      - Focused prompt per type
      - Still extracts all structured fields
      ↓
[4] Create item(s) with unconfirmed=true if low confidence
```

### Correction feedback loop

Two sources of labeled training data:

| Source | When | Logged as |
|--------|------|-----------|
| Clarification answered | User resolves low-confidence card | `clarification_answered` |
| Type correction | User promotes/demotes a created item | `correction_applied` |

All corrections stored in `classification_corrections`:
```
(clause_text, predicted_type, confidence, actual_type, source, created_at)
```

This table is the foundation for future few-shot injection and eventual logistic regression.

### Reprocess split

**Reprocess** (existing) — retry failed pipeline only
- Valid only when raw input status is `failed` or `pending`
- If items were already created → no-op, return error
- Never touches correction records

**Correct** (new) — fix wrong type on a created item
- Promote note→task, task→note, etc.
- Creates new item, deletes old item (preserving raw_input_id linkage)
- Logs to `classification_corrections` with `source=correction_applied`
- Clears `unconfirmed` flag on the new item

---

## Files to Touch

### New
| File | Layer | Purpose |
|------|-------|---------|
| `business/domain/ingestbus/classify/classifier.go` | Business | Rule-based classifier: obligation verbs, temporal anchors, reference patterns → type + confidence |
| `business/domain/ingestbus/classify/classifier_test.go` | Business | Table-driven tests: clear tasks, clear notes, clear events, ambiguous cases |
| `business/domain/ingestbus/cleanup/cleanup.go` | Business | Strip fillers, split compound input into clauses |
| `business/domain/ingestbus/cleanup/cleanup_test.go` | Business | Filler removal, clause splitting, edge cases |
| `business/domain/classificationcorrectionbus/` | Business | New domain: CRUD for correction log |
| `business/sdk/migrate/sql/<next>.sql` | Migration | `classification_corrections` table + `unconfirmed` column on tasks/notes/events |
| `app/domain/correctionapp/` | App | HTTP handlers for type correction (promote/demote) |

### Modified
| File | Layer | Why |
|------|-------|-----|
| `business/domain/ingestbus/extractor/prompt.go` | Business | Type-aware prompts per clause type; task-first framing; expanded examples; negative examples |
| `business/domain/ingestbus/extractor/claudecli.go` | Business | Pass type hint into prompt; tighten escalation heuristic; add `notes` to required schema |
| `business/domain/ingestbus/ingestbus.go` | Business | Wire cleanup → classifier → Claude extraction pipeline; replace single ExtractText call |
| `business/domain/rawinputbus/rawinputbus.go` | Business | Guard Reprocess: return error if items already created |
| `business/domain/taskbus/model.go` | Business | Add `Unconfirmed bool` field |
| `business/domain/notebus/model.go` | Business | Add `Unconfirmed bool` field |
| `business/domain/eventbus/model.go` | Business | Add `Unconfirmed bool` field |
| `business/types/clarificationkind/` | Types | Add `TypeAssignment` kind |
| `business/domain/clarificationbus/options.go` | Business | Add `TypeAssignmentOptions` struct |
| `app/domain/clarificationapp/` | App | Handle `TypeAssignment` clarification resolution → log correction |
| `api/services/planner/main.go` | Wire | Wire correctionbus into ingestbus and correctionapp |

---

## Cascade Rules

- New `unconfirmed` field on task/note/event → DB column (migration) + DB struct + converters + app DTO (3 domains, all layers)
- New `clarificationkind.TypeAssignment` → `business/types/clarificationkind/` + migration CHECK constraint + frontend `ClarificationCard` handler
- New `TypeAssignmentOptions` → `clarificationbus/options.go` + clarificationapp resolution handler + frontend deserializer
- New `classification_corrections` table → full new domain (`classificationcorrectionbus`) or lightweight store-only if no business logic needed
- `Reprocess` guard → `rawinputbus` + any callers that currently call Reprocess expecting it to fix classification

---

## Classifier Design (rule-based)

Input: single clause string
Output: `(type, confidence float64)`

```
Obligation verbs → task (high confidence):
  "need to", "have to", "should", "must", "want to", "remember to",
  "don't forget", "make sure", "pick up", "call", "email", "text",
  "schedule", "book", "buy", "send", "finish", "complete"

Temporal anchor (no obligation verb) → event (high confidence):
  time expression present (e.g. "at 2pm", "Tuesday", "next Monday")
  + no obligation verb

Pure reference → note (high confidence):
  phone number pattern, address, name + "is" + fact, "the best X is Y"

Mixed / unclear → ambiguous (low confidence):
  obligation verb + temporal anchor → could be task or event
  vague reference + possible action → could be note or task
```

Confidence thresholds:
- ≥ 0.75 → high confidence, proceed
- < 0.75 → low confidence, optimistic create + clarification card

Starting threshold at 0.75 (matches existing context confidence cutoff pattern).

---

## Prompt Refactor

Replace single generic prompt with three type-specific prompts:

**Task prompt:** "This clause has been classified as a task. Extract: title, description, priority (low/medium/high/urgent), deadline if present."

**Event prompt:** "This clause has been classified as an event. Extract: title, starts_at (UTC ISO 8601), ends_at (estimate 1hr if unclear), location, all_day, description."

**Note prompt:** "This clause has been classified as a note — reference information with no implied action. Extract: content, suggested_tags (1-3)."

All prompts include: current time, user timezone, active contexts for context matching.

---

## Test Approach

- `classify/classifier_test.go` — table-driven: 20+ cases covering clear tasks, events, notes, and ambiguous
- `cleanup/cleanup_test.go` — filler removal, clause splitting on "and", "also", "oh and", punctuation
- `ingestbus_test.go` — add regression cases: compound input splits correctly, low-confidence creates unconfirmed item + clarification card
- `correctionapp` — API test: correction creates new item, deletes old, logs to corrections table
- Reprocess guard test: reprocess on already-processed raw input returns error

---

## Build Order

1. `cleanup` package (no dependencies, easy to test)
2. `classify` package (no dependencies, easy to test)
3. Migration: `unconfirmed` columns + `classification_corrections` table + `type_assignment` clarification kind
4. `classificationcorrectionbus` domain (store + business)
5. Prompt refactor (type-aware prompts, type hint parameter)
6. Wire cleanup → classify → Claude into `processTextInput()`
7. `unconfirmed` flag on task/note/event models + converters + DTOs
8. `TypeAssignment` clarification kind + options struct + clarification resolution handler
9. Reprocess guard in `rawinputbus`
10. `correctionapp` — promote/demote endpoint
11. Frontend: unconfirmed badge + promote/demote action + TypeAssignment clarification card

---

## Callouts

- **Clause splitting is load-bearing** — the unit of correction logging is a clause, not a full voice input. If splitting is skipped or non-deterministic, the correction data is noisy.
- **Ollama fallback won't receive type hint** — it has its own extraction path. Acceptable since Ollama is last resort.
- **`unconfirmed` migration touches 3 domain models** — coordinate all three in a single migration to avoid partial state.
- **Reprocess guard must be enforced at the business layer** — not just the HTTP layer, since Reprocess is also called internally.
- **Confidence threshold (0.75) is a starting guess** — instrument it early so you can tune based on real correction data.
- **Frontend ClarificationCard needs a new branch** for `type_assignment` kind — don't forget this or the clarification will render blank.
