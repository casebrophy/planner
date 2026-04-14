# Clarification Free-Text Override → Re-ingest with Cleanup

**Problem:** When raw input is misinterpreted (e.g., "treat leather on blundstones" → "treat leather on blood stones"), all AI-generated suggestions are wrong and the user has no way to provide the correct interpretation.

**Solution:** Add a free-text input to clarifications. When resolved with free-text, clean up unconfirmed entities from the bad first pass, write the correction to the raw input, and re-trigger the ingest pipeline. The extractor gives the correction more weight than the original content.

## Design Decisions

- **`user_correction` column on `raw_inputs`** — separate field (not appended to `raw_content`) so the extractor can weight it higher
- **`raw_input_id` FK on `tasks`** — events and notes already have this; tasks are missing it. Needed so we can find all entities spawned from a raw input for cleanup.
- **Cleanup before re-ingest** — on free-text resolve, delete all `unconfirmed` entities (tasks, events, notes) linked to the source raw input. Only unconfirmed ones — if the user already manually edited/confirmed an entity, it survives.
- **Re-ingest on resolve** — after cleanup, `dispatchResolution` writes the correction to the raw input and calls `ResetForReprocess()`. The async pipeline picks it up and re-extracts with the correction in the prompt.
- **Extractor prompt injection** — `ExtractText` and `ExtractEmail` gain an optional `userCorrection string` param. When non-empty, the prompt template prepends it as a high-priority instruction: "The user has corrected this input: <correction>. Use this as the authoritative interpretation."
- **No new clarification kind** — free-text override works for any existing kind. The `free_text` key in the answer payload triggers the re-ingest path in `dispatchResolution`.
- **Infinite loop guard** — subsequent corrections overwrite `user_correction`, don't stack. If re-ingest generates another clarification, the user can correct again but only one correction is active at a time.
- **Pattern precedent** — `voice_reference` kind already uses a text input; `ResetForReprocess()` already exists; events/notes already have `raw_input_id`.

## Task 1: Migration — `user_correction` on raw_inputs + `raw_input_id` on tasks

**Files to modify:**
- `business/sdk/migrate/sql/migrate.sql`:
  - Add `user_correction TEXT` column to `raw_inputs` table (nullable)
  - Add `raw_input_id UUID REFERENCES raw_inputs(raw_input_id)` column to `tasks` table (nullable — existing tasks won't have it)

## Task 2: Task model — Add `RawInputID` field

**Files to modify:**
- `business/domain/taskbus/model.go` — add `RawInputID *uuid.UUID` to `Task`, `NewTask`, `UpdateTask`
- `business/domain/taskbus/stores/taskdb/model.go` — add `RawInputID *string` to DB struct + `toDBTask`/`toBusTask` converters
- `business/domain/taskbus/stores/taskdb/taskdb.go` — include `raw_input_id` in INSERT/UPDATE SQL
- `business/domain/ingestbus/ingestbus.go` — pass `ri.ID` as `RawInputID` when creating tasks (events/notes already do this)

**Tests:**
- Store test: create task with raw_input_id, query back, assert field persists

## Task 3: Raw Input model — Add `UserCorrection` field

**Files to modify:**
- `business/domain/rawinputbus/model.go` — add `UserCorrection *string` to `RawInput`, `UpdateRawInput`
- `business/domain/rawinputbus/stores/rawinputdb/model.go` — add `UserCorrection *string` to DB struct + converters
- `business/domain/rawinputbus/rawinputbus.go` — ensure `Update()` can set `UserCorrection`

**Tests:**
- Store test: create raw input, update with user_correction, query back, assert field persists

## Task 4: Extractor — Accept and weight `userCorrection`

**Files to modify:**
- `business/domain/ingestbus/extractor/model.go` — add `UserCorrection string` param to `ExtractText()` and `ExtractEmail()` interfaces
- `business/domain/ingestbus/extractor/prompt.go` — when `userCorrection` is non-empty, prepend to the extraction prompt: "IMPORTANT — The user has provided a correction for this input: '<correction>'. Treat this as the authoritative interpretation. The original content may contain transcription errors."
- `business/domain/ingestbus/ingestbus.go` — pass `ri.UserCorrection` through to extractor calls

**Tests:**
- Unit test: verify prompt includes correction text when provided
- Unit test: verify prompt unchanged when correction is empty

## Task 5: Cleanup — Delete unconfirmed entities before re-ingest

**Files to modify:**
- `business/domain/taskbus/taskbus.go` — add `DeleteByRawInputUnconfirmed(ctx, rawInputID)` method to Storer interface + business layer
- `business/domain/taskbus/stores/taskdb/taskdb.go` — implement: `DELETE FROM tasks WHERE raw_input_id = $1 AND unconfirmed = true`
- `business/domain/eventbus/eventbus.go` — add `DeleteByRawInputUnconfirmed(ctx, rawInputID)` (same pattern)
- `business/domain/eventbus/stores/eventdb/eventdb.go` — implement SQL
- `business/domain/notebus/notebus.go` — add `DeleteByRawInputUnconfirmed(ctx, rawInputID)` (same pattern)
- `business/domain/notebus/stores/notedb/notedb.go` — implement SQL

**Tests:**
- Store test per domain: create 2 entities from same raw_input_id (one confirmed, one unconfirmed), call DeleteByRawInputUnconfirmed, assert only unconfirmed deleted

## Task 6: dispatchResolution — Correction + cleanup + re-ingest

**Files to modify:**
- `app/domain/clarificationapp/clarificationapp.go` — in `dispatchResolution()`:
  1. Before the kind switch, check if answer contains `"free_text"` key
  2. If yes AND subject_type is `raw_input`:
     a. Call `taskBus.DeleteByRawInputUnconfirmed(ctx, subjectID)`
     b. Call `eventBus.DeleteByRawInputUnconfirmed(ctx, subjectID)`
     c. Call `noteBus.DeleteByRawInputUnconfirmed(ctx, subjectID)`
     d. Write `free_text` value to raw input's `user_correction` via `rawInputBus.Update()`
     e. Call `rawInputBus.ResetForReprocess()`
  3. If yes AND subject_type is NOT `raw_input`: store answer as-is, no side-effects
- `app/domain/clarificationapp/clarificationapp.go` — inject `rawInputBus`, `taskBus`, `eventBus`, `noteBus` dependencies
- `app/domain/clarificationapp/route.go` — wire all bus dependencies

**Tests:**
- API test: resolve clarification (subject_type=raw_input) with `{"free_text": "treat leather on shoes"}` → assert:
  - Unconfirmed entities from that raw input are deleted
  - Confirmed entities from that raw input survive
  - Raw input `user_correction` is set
  - Raw input status reset to pending
- API test: resolve non-raw-input clarification with `{"free_text": "..."}` → assert no crash, status=resolved

## Task 7: Frontend — Free-text override input in ClarificationCard

**Files to modify:**
- `api/services/frontend/web/src/components/clarifications/ClarificationCard.vue`

**Implementation:**
1. Add reactive refs: `freeTextOverride = ref('')`, `showFreeTextInput = ref(false)`
2. Below suggestions, add a "None of these? Type your own" text link
3. When toggled, show text input + "Submit" button
4. On submit: `resolveWithValue({ free_text: freeTextOverride.value })`
5. Disable submit when input empty
6. Reset state when navigating between clarifications (watch `currentIndex`)
7. Follow `voice_reference` text input pattern

**Tests:**
- Vitest: toggle override, type value, submit → assert `emit('resolve', { free_text: 'value' })`
- Vitest: submit disabled when input empty

## Ordering

```
Task 1 (migration: user_correction + tasks.raw_input_id)
  ├→ Task 2 (task model: RawInputID) — depends on 1
  │    └→ Task 5 (cleanup methods) — depends on 2
  │         └→ Task 6 (dispatchResolution) — depends on 3, 4, 5
  └→ Task 3 (raw input model: UserCorrection) — depends on 1
       └→ Task 4 (extractor: weight correction) — depends on 3
            └→ Task 6 (dispatchResolution) — depends on 3, 4, 5
Task 7 (frontend) — independent, parallel with all backend tasks
```

## Risks

- **Medium risk: extractor prompt quality** — the correction needs to actually override the misinterpretation. Prompt wording matters. Test with real examples (voice transcription errors).
- **Low risk: confirmed entity divergence** — if the user confirmed a bad entity before correcting via free-text, it survives cleanup and the re-ingest creates a correct duplicate. Acceptable — user can manually delete the stale one.
- **Low risk: infinite loop** — re-ingest generates another clarification → user corrects → another re-ingest. Guard: `user_correction` overwrites (doesn't stack). Pipeline should also skip clarification generation when `user_correction` is set (the user already told us what they meant).
- **Low risk: orphaned clarifications** — the original clarification items from the first ingest pass still exist after cleanup. They're resolved, so they won't show in the queue, but they reference deleted entities. Acceptable — resolved clarifications are historical records.
