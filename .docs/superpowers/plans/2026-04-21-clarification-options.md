# Clarification Options (Knowledge Gap)

**Date:** 2026-04-21
**Status:** Planned

## Problem

KnowledgeGap clarification cards currently only accept free-text answers. Many questions the generator produces are enumerable — e.g., "When do you need the bath salts by?" maps cleanly to `Today / This week / This month / Whenever`. Forcing the user to type prose instead of tapping a chip is slower and produces less structured data. Other clarification kinds (AmbiguousAction, TypeAssignment, ContextAssignment) already render chips; KnowledgeGap is the last remaining kind stuck on free-text.

## Goal

Let the knowledge-gap generator emit selectable options alongside free-text as a fallback, with a confidence signal indicating how confident it is that the options it produced are exhaustive. The frontend renders chips when confidence is high and options are present, and always keeps the free-text escape hatch available.

## Non-Goals

- No changes to other clarification kinds.
- No DB migration — `answer_options` is JSONB and schema-less.
- No changes to the clarification queue navigator, snooze, or dismiss flows.
- No change to how the note side-effect is created on resolve (selected option just pre-fills the text).

---

## Design Decisions

### Options + Confidence live on `KnowledgeGapOptions`

Mirror the shape of `TypeAssignmentOptions` (options array + confidence float). Two new fields:

```go
type KnowledgeGapOptions struct {
    GapCategory              string   `json:"gap_category"`
    RelatedEntityType        string   `json:"related_entity_type"`
    RelatedEntityID          string   `json:"related_entity_id"`
    SuggestedQuestion        string   `json:"suggested_question"`
    ExistingKnowledgeSummary string   `json:"existing_knowledge_summary"`
    Options                  []string `json:"options"`      // nil/empty = free-text only
    Confidence               float64  `json:"confidence"`   // 0 when no options
}
```

Zero values (nil options, 0 confidence) serialize back-compat with existing stored rows.

### Confidence threshold: 0.6

The knowledge-gap detector already uses `ConfidenceThreshold = 0.6` for gap creation. Reuse the same threshold for "render chips" — define as a named constant in the Vue component. Below threshold → free-text only (even if the model returned options, we don't trust them enough to surface).

### Answer payload extends additively

Today: `{answer_text: string}` or `{dismissed: true}`.
Add: `{selected_option: string}`.

Precedence in `dispatchResolution()`: `dismissed` → `selected_option` (treat as `answer_text`, same note side-effect) → `answer_text`. No new bus methods; no new note fields.

### Prompt asks Claude to self-assess enumerability

Extend `BuildGapAnalysisPrompt` JSON schema per gap:

```json
{
  "category": "...",
  "question": "...",
  "reasoning": "...",
  "confidence": 0.8,
  "related_ids": [...],
  "options": ["Today", "This week", "This month", "Whenever"],
  "options_confidence": 0.85
}
```

Prompt guidance: "If the question has a small, well-defined set of likely answers (2–4), list them in `options` and rate your confidence in `options_confidence`. If the answer is open-ended (descriptions, reasons, long-form details), set `options` to `[]` and `options_confidence` to `0`."

### Frontend: chips + persistent free-text

The existing "None of these? Type your own" escape hatch stays visible. Chips appear above the textarea when the gate passes. Selecting a chip emits `resolve({ selected_option: value })` immediately — no separate submit button (matches the TypeAssignment chip UX).

---

## Files to Touch

**Backend**

| File | Change |
|------|--------|
| `business/domain/clarificationbus/options.go` | Add `Options` + `Confidence` to `KnowledgeGapOptions` |
| `business/domain/knowledgegapbus/model.go` | Add `Options []string` + `OptionsConfidence float64` to `GapCandidate` |
| `business/domain/ingestbus/extractor/prompt.go` | Extend `BuildGapAnalysisPrompt` JSON schema + guidance text |
| `business/domain/ingestbus/extractor/*.go` | Update the response-unmarshal struct used by `AnalyzeGaps` to include the two new fields |
| `business/domain/knowledgegapbus/knowledgegapbus.go` | Pass `gap.Options` + `gap.OptionsConfidence` into `KnowledgeGapOptions` at `Detect()` |
| `app/domain/clarificationapp/clarificationapp.go` | Extend KnowledgeGap `dispatchResolution` branch to handle `selected_option` |

**Frontend**

| File | Change |
|------|--------|
| `api/services/frontend/web/src/types/generated/clarification-options.ts` | Regenerate via tygo (do NOT hand-edit) |
| `api/services/frontend/web/src/components/clarifications/ClarificationCard.vue` | Add chip-render branch to `knowledge_gap` case; emit `{selected_option}` on chip click |

**Tests**

| File | Change |
|------|--------|
| `business/domain/knowledgegapbus/*_test.go` | Cover options populated vs empty paths |
| `app/domain/clarificationapp/*_test.go` (or apitest) | Cover `selected_option` resolve path |
| `api/services/frontend/web/src/__tests__/components/clarifications/ClarificationCard.test.ts` | Chip render, free-text fallback, click emits correct payload |

---

## Implementation Sequence

### Task 1 — Backend struct + extractor response

Worker: haiku.
Scope:
1. Add `Options []string` and `Confidence float64` json-tagged fields to `KnowledgeGapOptions` in `business/domain/clarificationbus/options.go`.
2. Add `Options []string` and `OptionsConfidence float64` to `GapCandidate` in `business/domain/knowledgegapbus/model.go`.
3. Update the extractor response struct in `business/domain/ingestbus/extractor/` that unmarshals the AI response into `GapCandidate` — add the two fields with matching json tags (`options`, `options_confidence`).
4. Wire in `knowledgegapbus.Detect()` (or `DetectWithOptions`) to copy `gap.Options` → `KnowledgeGapOptions.Options` and `gap.OptionsConfidence` → `KnowledgeGapOptions.Confidence` when constructing the clarification.

Verify: `make lint && make test`.

### Task 2 — Prompt engineering

Worker: haiku.
**Depends on Task 1.** Also coordinate with planner-ztbk (same file); if planner-ztbk is still in-progress when this runs, rebase carefully.

Scope:
1. In `BuildGapAnalysisPrompt` (`business/domain/ingestbus/extractor/prompt.go`), extend the JSON schema example and the output contract to include `options` and `options_confidence` fields.
2. Add a short paragraph of prompt guidance describing when to emit options vs empty: enumerable / small-set questions → 2–4 options with a confidence; open-ended → `[]` + `0`.
3. Keep the existing confidence filtering (≥0.5 for gap creation) separate from `options_confidence`.

Verify: `make lint && make test`. No unit test for the prompt text itself; any existing integration test that mocks the extractor response must still pass.

### Task 3 — App-layer resolve path

Worker: haiku.
**Depends on Task 1.** Can run in parallel with Task 2.

Scope:
1. In `app/domain/clarificationapp/clarificationapp.go` `dispatchResolution()` KnowledgeGap case: extend the answer struct to include `SelectedOption string` json-tagged `selected_option`.
2. Precedence: `dismissed` wins; then if `selected_option != ""`, treat it as `answer_text` for the existing note-creation side-effect; then `answer_text`. Do NOT add a new bus method or a new note field — the option string becomes the note content.

Verify: `make lint && make test`.

### Task 4 — Regenerate frontend types

Worker: haiku.
**Depends on Task 1 merged.**

Scope:
1. Find the tygo invocation (Makefile target or `package.json` script — e.g., `make gen-types` or `npm run gen-types`) and run it.
2. Confirm `api/services/frontend/web/src/types/generated/clarification-options.ts` now includes `Options` and `Confidence` fields on `KnowledgeGapOptions`.

Verify: `make frontend-lint && make frontend-build` (build is required per user preference — type regressions must fail loudly).

### Task 5 — Frontend chip rendering

Worker: haiku.
**Depends on Task 4.**

Scope:
1. In `ClarificationCard.vue`, extend the `knowledge_gap` branch:
   - Compute/read `answerOptions.options` and `answerOptions.confidence` from the already-computed `knowledgeGapOptions` ref.
   - Define a `CHIPS_THRESHOLD = 0.6` constant in the component.
   - Render chips when `options?.length > 0 && confidence >= CHIPS_THRESHOLD`. Follow the `type_assignment` branch for styling (emerald selected / gray unselected, tailwind classes).
   - Chip click → local `selectedOption` ref, then emit `resolve({ selected_option: value })` immediately (matches TypeAssignment UX).
   - Keep the existing free-text textarea below the chips; the "None of these? Type your own" affordance stays.
2. If `options` is empty or confidence below threshold, render exactly today's free-text-only UI.

Verify: `make frontend-lint && make frontend-build && make frontend-test`.

### Task 6 — Tests

Worker: haiku.
**Depends on Tasks 3 and 5.**

Scope:
1. Backend: add a `knowledgegapbus` test exercising `Detect` with a mocked `GapAnalyzer` returning a `GapCandidate` with options populated — assert the resulting clarification's `AnswerOptions` JSON contains `options` and `confidence`. Add a second test with empty options — assert zero-value propagation.
2. Backend: add an apitest or clarificationapp test posting `{selected_option: "This week"}` to resolve a KnowledgeGap clarification. Assert the note is created with that content and the clarification moves to resolved. Keep a second test for the existing `{answer_text: "..."}` path to prove no regression.
3. Frontend: extend `ClarificationCard.test.ts` with three cases:
   - chip render when options populated + confidence ≥ 0.6
   - free-text only when options empty
   - free-text only when confidence below threshold
   - chip click emits `resolve` with `{ selected_option: <value> }`

Verify: `make test && make frontend-test`.

---

## Verification

- `make lint && make test && make frontend-lint && make frontend-build && make frontend-test` all green.
- Manual smoke: seed a task, wait for knowledge-gap detection to run (or trigger it), check the clarification queue in the frontend — for an enumerable question, chips should render; for an open-ended one, the UI should match today's.
- Regression: resolve an existing pre-feature KnowledgeGap clarification (one whose stored `answer_options` JSON lacks the new fields) — ensure it falls back to free-text cleanly.

## Risks

1. **tygo regeneration forgotten.** The frontend build will still succeed if the generator is run but types will be stale if it isn't. Mitigation: Task 4 requires `make frontend-build` and Task 5 depends on it.
2. **Prompt quality.** Claude may over-enumerate (produce options for genuinely open-ended questions) or under-enumerate. The confidence gate + free-text fallback protect the user experience; adjust threshold post-launch if needed.
3. **Conflict with planner-ztbk.** Both edit `BuildGapAnalysisPrompt`. Sequence: land planner-ztbk first, then Task 2. If parallel is unavoidable, rebase carefully.

## Open Questions

- Should chip selection go through a confirmation step, or auto-submit on click? Plan currently says auto-submit (matches TypeAssignment UX). If the user wants a confirmation step, Task 5 gains a small scope bump.
- Should we persist `options_confidence` separately in the clarification record (for future analysis of gating accuracy), or is storing it inside `answer_options` JSONB sufficient? Plan says JSONB only — revisit if we want to analyze gating quality later.
