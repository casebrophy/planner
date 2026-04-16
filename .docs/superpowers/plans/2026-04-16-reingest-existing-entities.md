# Reingest Existing Entities — Implementation Plan

**Date:** 2026-04-16
**Feature:** Unified entity pipeline + reingest endpoint

## Overview

Today, entities enter the system through two divergent paths: the ingest pipeline (email/voice → rawinputbus → classify → entity creation → embeddings + knowledge gaps) and direct API creation (taskapp/noteapp/eventapp → entity row, with no downstream side effects). This causes manually-created entities to miss out on embeddings, knowledge gap detection, and clarification generation. It also means there is no way to replay the side-effect pipeline on existing entities when classifiers change or new gap heuristics ship.

This feature unifies both paths through `rawinputbus`: every entity has a backing raw_input row (`Source=Manual` for API-created ones). A new reingest endpoint re-queues an existing entity's raw_input for reprocessing, picking up new classifications and filling retroactive knowledge gaps. Clarifications become idempotent via Upsert so reprocessing never duplicates cards, and user answers on existing cards are preserved. Reingest runs asynchronously via the existing `IngestWorker` (30s poll), so the API returns immediately with a `queued` response.

## Design decisions

1. **Clarification Upsert preserves user answers.** ON CONFLICT DO UPDATE overwrites only AI-generated fields (`question`, `claude_guess`, `reasoning`, `priority_score`). `answer`, `status`, `resolved_at`, `snoozed_until` are never touched — user intent is sacred.
2. **Unified pipeline via rawinputbus.** Manual entity creation synthesizes a raw_input with `Source=Manual` and `SkipClassify=true`, then links the entity via `source_entity_id`. All post-create side effects (embeddings, gaps, clarifications) run through the same ingest pipeline regardless of origin.
3. **Bulk reingest is async.** The bulk endpoint resets matched raw_inputs to `pending` and returns `{queued: N}`. The existing `IngestWorker` drains the queue on its normal 30s cadence — no inline processing, no blocking.
4. **Reingest-mode flag suppresses the `unconfirmed=true` flip.** During normal ingest, entity resolution sets `unconfirmed=true` so `DeleteByRawInputUnconfirmed` can clean up on error. On reingest of an already-confirmed entity, that flip would risk deletion — `reingest_mode=true` on the raw_input suppresses it.
5. **Never force classify on linked entities.** If `raw_input.source_entity_id IS NOT NULL`, reingest runs embeddings + knowledge gap detection only. Classify stays user-intent-protected; linked entities skip it. Unlinked/unconfirmed entities still get the full pipeline.

## Architecture

Three paths now converge at the same post-create side-effect block:

```
          +--------------------+
Ingested  | email/voice/http   |--[rawinputbus.Create]--+
          +--------------------+                         |
                                                         v
          +--------------------+                 +-----------------+
Manual    | taskapp/noteapp/   |--[synthesize]->| rawinputbus row |
          | eventapp .Create   |                | Source=Manual   |
          +--------------------+                | SkipClassify=T  |
                                                | source_entity_* |
          +--------------------+                +--------+--------+
Reingest  | reingestapp        |--[reset]---->           |
          | POST /reingest     |   ReingestMode=T        |
          +--------------------+                         |
                                                         v
                                           +---------------------------+
                                           | IngestWorker (30s poll)   |
                                           | ProcessRawInputByID       |
                                           +-------------+-------------+
                                                         |
                  +---------------------+----------------+-----------+
                  | skip_classify=true?                              |
                  |                                                  |
                  v                                                  v
        +-----------------------+                       +---------------------------+
        | Load entity by        |                       | Classify (LLM)            |
        | source_entity_*,      |                       | Create entity row         |
        | skip classify+create  |                       | (unless reingest_mode)    |
        +-----------+-----------+                       +-------------+-------------+
                    |                                                 |
                    +-------------------+-----------------------------+
                                        |
                                        v
                        +----------------------------------+
                        | Embeddings (DeleteBySource +     |
                        | regenerate)                      |
                        | Knowledge gap Detect (async)     |
                        | Clarification Upsert (idempotent)|
                        +----------------------------------+
```

Two branching signals control behavior inside `ProcessRawInputByID`:

- `skip_classify` → "an entity already exists for this raw_input; load it, skip classify + create."
- `reingest_mode` → "do not flip unconfirmed=true during entity resolution; this entity was already confirmed."

Both default to `false`, preserving exact legacy ingest behavior.

---

## Phase 1: Clarification Upsert foundation

**Goal:** Make clarification creation idempotent on `(kind, subject_type, subject_id)` while preserving any user-provided answer state.

### Files touched

- CREATE `business/sdk/migrate/sql/<next_number>_clarification_upsert.sql`
- MODIFY `business/domain/clarificationbus/clarificationbus.go`
- MODIFY `business/domain/clarificationbus/stores/clarificationdb/clarificationdb.go`
- MODIFY `app/domain/classifyapp/classifyapp.go`
- MODIFY `business/domain/knowledgegapbus/knowledgegapbus.go`
- MODIFY `business/domain/ingestbus/ingestbus.go` (clarification call sites only — the ingest-flow changes land in Phase 4)
- MODIFY `business/domain/clarificationbus/stores/clarificationdb/clarificationdb_test.go`

### Implementation steps

1. **Write migration `<next_number>_clarification_upsert.sql`.** Inspect the highest-numbered file in `business/sdk/migrate/sql/` and increment. Contents:
   - Deduplicate any existing rows that would violate the new unique constraint. Keep the row with the most informative state (prefer rows where `answer IS NOT NULL`, then most recent `created_at`). Delete the rest. Use a CTE with `ROW_NUMBER() OVER (PARTITION BY kind, subject_type, subject_id ORDER BY (answer IS NOT NULL) DESC, created_at DESC)`.
   - Add the constraint: `ALTER TABLE clarifications ADD CONSTRAINT uq_clarification_dedup UNIQUE (kind, subject_type, subject_id);`.
   - Include a SQL comment explaining that if a new ClarificationKind is added later that should NOT be deduped, either it uses a partial index or the app layer bypasses Upsert.

2. **Add `Upsert` to the Storer interface** in `business/domain/clarificationbus/clarificationbus.go`. Signature mirrors `Create`: `Upsert(ctx context.Context, nc NewClarification) (Clarification, error)`.

3. **Add `Upsert` on the Business struct** in the same file. Accepts a `NewClarification`, validates inputs the same way `Create` does, calls `s.storer.Upsert(ctx, nc)`, returns the resulting `Clarification`. No retry loop needed — the DB handles conflict resolution atomically.

4. **Implement `Upsert` in the store** (`stores/clarificationdb/clarificationdb.go`). Build the INSERT from the existing `Create` SQL, append:
   - `ON CONFLICT ON CONSTRAINT uq_clarification_dedup DO UPDATE SET`
   - `question = EXCLUDED.question,`
   - `claude_guess = EXCLUDED.claude_guess,`
   - `reasoning = EXCLUDED.reasoning,`
   - `priority_score = EXCLUDED.priority_score,`
   - `updated_at = NOW()`
   - (Explicitly do NOT include `answer`, `status`, `resolved_at`, `snoozed_until`, `answer_options`, `subject_description` — the last two are stable per-kind, no need to rewrite.)
   - Append `RETURNING *` and scan into the DB struct, then convert via `toBusClarification`.

5. **Swap Create → Upsert at all clarification-creation call sites.**
   - `app/domain/classifyapp/classifyapp.go`: replace `clarificationBus.Create(...)` with `.Upsert(...)`.
   - `business/domain/knowledgegapbus/knowledgegapbus.go`: remove the Count-then-Create dedup guard; call `Upsert` directly.
   - `business/domain/ingestbus/ingestbus.go`: any `clarificationBus.Create(...)` sites become `Upsert`. (Other ingest-flow changes land in Phase 4.)

6. **Update the store test** (`stores/clarificationdb/clarificationdb_test.go`): add the two new tests described below.

### Tests for this phase

- `TestClarificationUpsert_Idempotency` (dbtest): Upsert a clarification. Upsert again with identical inputs. Assert exactly one row exists in the table for the dedup key.
- `TestClarificationUpsert_PreservesAnswer` (dbtest): Upsert. Update the row directly setting `answer='yes'`, `status='resolved'`, `resolved_at=NOW()`. Upsert again with a different `question`/`claude_guess`/`reasoning`/`priority_score`. Reload and assert the four AI fields changed, and `answer`, `status`, `resolved_at` are unchanged.
- `TestClarificationUpsert_RespectsDedupKey` (dbtest): Upsert a row with kind=A. Upsert another row with kind=B, same subject — assert two distinct rows exist (different kinds → different dedup keys).

### Definition of done

- Migration applies cleanly against a seeded dev DB (dedup step leaves exactly the expected rows).
- `make test` passes for clarificationbus store tests.
- Grep confirms zero remaining `clarificationBus.Create` call sites (all switched to Upsert).

---

## Phase 2: RawInput schema extensions + Manual source type

**Goal:** Extend `raw_inputs` with the four new columns and add `Manual` to the source enum, plumbed through business, store, and type layers.

### Files touched

- CREATE `business/sdk/migrate/sql/<next_number>_rawinput_reingest_fields.sql`
- MODIFY `business/types/rawinputsource/rawinputsource.go`
- MODIFY `business/domain/rawinputbus/model.go`
- MODIFY `business/domain/rawinputbus/stores/rawinputdb/model.go`
- MODIFY `business/domain/rawinputbus/stores/rawinputdb/rawinputdb.go`
- MODIFY `business/domain/rawinputbus/rawinputbus_test.go` (or equivalent dbtest file)

### Implementation steps

1. **Write migration `<next_number>_rawinput_reingest_fields.sql`:**
   - `ALTER TABLE raw_inputs ADD COLUMN source_entity_id UUID NULL;`
   - `ALTER TABLE raw_inputs ADD COLUMN source_entity_kind TEXT NULL;`
   - `ALTER TABLE raw_inputs ADD COLUMN skip_classify BOOLEAN NOT NULL DEFAULT FALSE;`
   - `ALTER TABLE raw_inputs ADD COLUMN reingest_mode BOOLEAN NOT NULL DEFAULT FALSE;`
   - Drop-and-recreate the `raw_inputs_source_check` CHECK constraint to include `'manual'` alongside existing values (email, voice, etc.). Find exact name by inspecting the current schema; if CHECK is anonymous, use `ALTER TABLE ... ADD CONSTRAINT raw_inputs_source_check CHECK (source IN (...))` after dropping the old one.
   - Add a partial index: `CREATE INDEX IF NOT EXISTS idx_raw_inputs_source_entity ON raw_inputs (source_entity_kind, source_entity_id) WHERE source_entity_id IS NOT NULL;` — supports lookup during reingest and bulk queries.

2. **Add `Manual` to `business/types/rawinputsource/rawinputsource.go`:**
   - Add the `Manual` constant following the existing pattern for `Email`, `Voice`, etc.
   - Add to the `Parse`/`MustParse` switch and to the `values` slice.
   - Check whether the type has a validation test and extend it.

3. **Extend business model** (`business/domain/rawinputbus/model.go`):
   - Add fields to `RawInput`: `SourceEntityID *uuid.UUID`, `SourceEntityKind string`, `SkipClassify bool`, `ReingestMode bool`.
   - Add the same fields to `NewRawInput` and `UpdateRawInput` (as `*T` for UpdateRawInput to allow partial updates).
   - Update constructors that fill defaults.
   - `ResetForReprocess` already zeros retry state — audit it to ensure it does NOT clear `SourceEntityID`/`SourceEntityKind`/`SkipClassify`/`ReingestMode`. On reingest those fields must survive the reset. Extend `ResetForReprocess` (or add a sibling `ResetForReingest`) to set `ReingestMode=true`; see Phase 5 for caller.

4. **Extend DB struct** (`business/domain/rawinputbus/stores/rawinputdb/model.go`):
   - Add struct fields with `db:"source_entity_id"`, `db:"source_entity_kind"`, `db:"skip_classify"`, `db:"reingest_mode"` tags. Use `uuid.NullUUID` and `sql.NullString` for nullable fields.
   - Update `toDBRawInput` / `toBusRawInput` converters to translate both directions, including pointer↔NullUUID mapping for `SourceEntityID`.

5. **Update SQL in `rawinputdb.go`:**
   - INSERT: add the four columns and `:source_entity_id, :source_entity_kind, :skip_classify, :reingest_mode` to the VALUES list.
   - UPDATE (full-row): add the four columns to the SET list.
   - SELECT * queries: explicit column lists (if any) must include the new columns; `SELECT *` patterns need no change.
   - `ResetForReprocess` store method: only touches status/retry columns — no change.
   - Add a new store method `UpdateSourceEntity(ctx, id uuid.UUID, entityID uuid.UUID, entityKind string) error` used by manual-create flow to backfill the link after the entity row is written.

6. **Update existing rawinputbus dbtests** to round-trip the new fields and verify defaults (`SkipClassify=false`, `ReingestMode=false`, `SourceEntityID=nil`, `SourceEntityKind=""` for pre-existing flows).

### Tests for this phase

- `TestRawInput_NewFields_RoundTrip` (dbtest): create a `NewRawInput` with all four fields set, fetch by ID, assert parity.
- `TestRawInput_DefaultsPreserveLegacy` (dbtest): create a raw_input via the legacy path (no new fields), assert all four default correctly.
- `TestRawInputSource_ManualEnum` (unit): parse/format round-trip for `Manual`.
- `TestUpdateSourceEntity` (dbtest): create raw_input without link, call `UpdateSourceEntity`, reload, assert link is set.

### Definition of done

- Migration applies cleanly; existing raw_inputs get the defaults.
- `make test` passes across `business/domain/rawinputbus/...` and `business/types/rawinputsource/...`.
- No existing ingest path behavior changes (all defaults preserve pre-migration flow).

---

## Phase 3: Unified manual-entity creation pipeline

**Goal:** Refactor `taskapp`, `noteapp`, `eventapp` Create handlers to synthesize a raw_input first, create the entity, link the raw_input, then enqueue async processing — matching the ingested-entity flow.

### Files touched

- MODIFY `app/domain/taskapp/taskapp.go`
- MODIFY `app/domain/noteapp/noteapp.go` (or equivalent — match existing layout)
- MODIFY `app/domain/eventapp/eventapp.go` (or equivalent)
- MODIFY `app/domain/taskapp/route.go` (wire in rawinputBus dep if not already present)
- MODIFY `app/domain/noteapp/route.go`
- MODIFY `app/domain/eventapp/route.go`
- MODIFY `api/services/planner/main.go` (ensure rawinputBus is injected into task/note/event apps)
- MODIFY existing taskapp/noteapp/eventapp tests

### Implementation steps

1. **Dependency injection.** Confirm `rawinputBus` is reachable from each entity app. If `Routes.Add()` signatures for taskapp/noteapp/eventapp don't currently take `*rawinputbus.Business`, add it. Update the call site in `api/services/planner/main.go`.

2. **Refactor the Create handler** (taskapp as the reference; noteapp/eventapp follow the same structure):
   1. Parse + validate request body (unchanged).
   2. Build a `rawinputbus.NewRawInput` with:
      - `Source = rawinputsource.Manual`
      - `Content` = a canonical serialization of the incoming request (JSON-encode the payload; this gives a reingest-able textual representation).
      - `Status = rawinputstatus.Processed` (NOT `pending` — we already processed it via the direct create; reingest moves it back to pending later).
      - `SkipClassify = true`
      - `ReingestMode = false` (first-time processing; nothing to suppress)
      - `SourceEntityID = nil` (populated in step 4)
      - `SourceEntityKind = "task"` / `"note"` / `"event"` as appropriate.
   3. Call `rawinputBus.Create(ctx, newRI)` to get `ri`.
   4. Call `taskBus.Create(ctx, nt)` to get `tsk`. Persist `tsk.RawInputID = ri.ID` on the task row — either set this on `NewTask` before creating, OR follow up with an update. (Confirm the existing NewTask struct already supports RawInputID; if not, include that as a fix inside this phase.)
   5. Call `rawinputBus.UpdateSourceEntity(ctx, ri.ID, tsk.ID, "task")` to backfill the link on the raw_input side.
   6. Fire the async side-effect pipeline. Two options; pick one and document:
      - **Option A (preferred, simpler):** reset the raw_input to `pending` immediately so IngestWorker picks it up on its next tick. Users wait up to 30s for embeddings/gaps.
      - **Option B (lower-latency):** call `ingestBus.ProcessRawInputByID(context.Background(), ri.ID)` in a goroutine. Use `context.Background()` (NOT the request context — it cancels on response). Log errors; do not block the response.
      Phase 3 uses Option A for simplicity and to reuse the IngestWorker's retry/error handling. (Option B is a follow-up if 30s latency is user-visible.)
   7. Return the entity response (unchanged shape).

3. **Noteapp + Eventapp.** Apply the identical structure. Adjust the `SourceEntityKind` string constant per entity.

4. **Error handling.** If step 2.3 fails, return `errs.Internal`. If step 2.4 fails, attempt to delete the raw_input to avoid orphan pending rows (best-effort; log if the delete fails). If step 2.5 fails, the entity is created but unlinked — log and continue; Phase 7 backfill handles this edge case.

5. **Update existing taskapp/noteapp/eventapp API tests** to assert that a raw_input exists after Create with `Source=Manual`, `SkipClassify=true`, `SourceEntityID` matching the created entity.

### Tests for this phase

- `TestManualTaskCreate_ProducesRawInput` (apitest): POST /api/v1/tasks, fetch raw_inputs by source_entity_id, assert `Source=Manual`, `SkipClassify=true`.
- `TestManualNoteCreate_ProducesRawInput` (apitest): same for notes.
- `TestManualEventCreate_ProducesRawInput` (apitest): same for events.
- `TestManualTaskCreate_LinksBidirectionally` (apitest): assert `task.raw_input_id == raw_input.id` AND `raw_input.source_entity_id == task.id`.

### Definition of done

- `make test` passes for taskapp / noteapp / eventapp / rawinputbus.
- New manual entities visible in the raw_inputs table with Source=Manual and bidirectional links.
- Existing ingest flow untouched (regression tests still pass).

---

## Phase 4: Ingestbus skip_classify + reingest_mode branches

**Goal:** Teach `ProcessRawInputByID` to respect the two new flags — short-circuit classify + create when `SkipClassify=true`, and suppress `unconfirmed=true` when `ReingestMode=true`.

### Files touched

- MODIFY `business/domain/ingestbus/ingestbus.go`
- MODIFY `business/domain/ingestbus/ingestbus_test.go` (or dbtest equivalent)

### Implementation steps

1. **Load the raw_input at the top of ProcessRawInputByID** (already happens). Capture the new fields into local vars for readability: `skipClassify := ri.SkipClassify`, `reingestMode := ri.ReingestMode`.

2. **Add the skip_classify branch.** Directly after raw_input load, before classify:
   - If `skipClassify == true AND ri.SourceEntityID != nil`:
     - Dispatch on `ri.SourceEntityKind`:
       - `"task"` → `taskBus.QueryByID(ctx, *ri.SourceEntityID)`
       - `"note"` → `noteBus.QueryByID(...)`
       - `"event"` → `eventBus.QueryByID(...)`
     - Handle `errors.Is(err, sqldb.ErrDBNotFound)` → return `errs.NotFound` (entity was deleted out from under us; caller should handle gracefully).
     - Skip classify + entity-create entirely; jump to the shared side-effect block (step 4).
   - If `skipClassify == true AND ri.SourceEntityID == nil`:
     - This is an illegal state (skip_classify implies a linked entity). Log warn and fall through to full pipeline as a defensive fallback.

3. **Add the reingest_mode guard around unconfirmed flip.** Find the site in `ingestbus.go` that sets `unconfirmed=true` on the entity during resolution. Wrap: `if !reingestMode { ... set unconfirmed = true ... }`. This preserves the existing DeleteByRawInputUnconfirmed cleanup for fresh ingests while preventing reingest from flagging already-confirmed entities for deletion.

4. **Shared side-effect block (runs for all three paths).** Ensure this block executes regardless of which branch was taken:
   - `embeddingBus.DeleteBySource(ctx, sourceKind, entityID)` — wipe old embeddings.
   - Regenerate embeddings from the entity's current text.
   - Fire knowledge gap detection in a goroutine with `context.Background()` (existing pattern).
   - Clarification generation via Upsert.
   - Mark raw_input as `processed` on success.

5. **Audit all return paths for `MarkFailed` coverage.** Per gotcha #3, `ProcessRawInputByID` returns errors without calling `MarkFailed` — this stays. The reingest endpoint (Phase 5) is responsible for calling `MarkFailed` on error.

### Tests for this phase

- `TestProcessRawInput_SkipClassifyLoadsEntity` (dbtest + fake bus stack): seed task + raw_input with `SkipClassify=true, SourceEntityID=taskID, SourceEntityKind="task"`. Call ProcessRawInputByID. Assert classify NOT invoked (mock/spy), entity NOT re-created, embeddings called.
- `TestProcessRawInput_SkipClassifyMissingEntity` (dbtest): raw_input with `SkipClassify=true` but `SourceEntityID` pointing to a deleted task → returns `errs.NotFound`.
- `TestProcessRawInput_ReingestModeSuppressesUnconfirmed` (dbtest): seed confirmed task + raw_input with `ReingestMode=true`. Process. Assert task.unconfirmed is still false.
- `TestProcessRawInput_NonReingestSetsUnconfirmed` (dbtest): regression — without the flag, unconfirmed flip still happens for fresh ingests.
- `TestProcessRawInput_DefensiveFallback` (dbtest): `SkipClassify=true`, `SourceEntityID=nil` → logs warn, runs full pipeline, completes without crash.

### Definition of done

- `make test` passes across ingestbus.
- Spies/fakes confirm classify is skipped when expected and runs when expected.
- Existing async-ingest apitests pass unchanged.

---

## Phase 5: Reingest bus methods + single-entity API endpoints

**Goal:** Add `Reingest(ctx, id)` methods on taskbus/notebus/eventbus, plus `POST /api/v1/{entity}/{id}/reingest` endpoints.

### Files touched

- MODIFY `business/domain/taskbus/taskbus.go`
- MODIFY `business/domain/notebus/notebus.go`
- MODIFY `business/domain/eventbus/eventbus.go`
- CREATE `app/domain/reingestapp/reingestapp.go`
- CREATE `app/domain/reingestapp/route.go`
- CREATE `app/domain/reingestapp/reingestapp_test.go`
- MODIFY `api/services/planner/main.go`

### Implementation steps

1. **Business method `taskBus.Reingest(ctx, taskID)`** (mirror in notebus/eventbus):
   1. `tsk, err := b.QueryByID(ctx, taskID)`. Check `sqldb.ErrDBNotFound` and return `errs.NotFound`.
   2. If `tsk.RawInputID == nil` (pre-migration entity):
      - Synthesize a raw_input with `Source=Manual`, `SourceEntityID=&taskID`, `SourceEntityKind="task"`, `SkipClassify=true`, `ReingestMode=true`, `Status=Pending`. Content = JSON encoding of the task's current state.
      - Persist. Update `tsk.RawInputID` to point at the new raw_input.
      - Return. (IngestWorker will pick it up.)
   3. Else (`RawInputID != nil`):
      - Call `rawinputBus.ResetForReingest(ctx, *tsk.RawInputID)` which: clears retry state + errors, sets `Status=Pending`, sets `SkipClassify=true`, sets `ReingestMode=true`.
      - (Add `ResetForReingest` to rawinputbus as a sibling of `ResetForReprocess`; shares implementation except for the two flags.)
      - Return.
   4. The business method does NOT invoke ProcessRawInputByID — IngestWorker drains the queue asynchronously.

2. **Optional: synchronous reingest.** If the single-entity endpoint should run inline (low-latency, small blast radius), dispatch `go ingestBus.ProcessRawInputByID(context.Background(), riID)` with explicit `MarkFailed` on error. Default: async via IngestWorker. Leave a code comment documenting the tradeoff.

3. **`reingestapp.go` handlers:**
   - `reingestTask`: parse `task_id` from URL, call `taskBus.Reingest`, return `{queued: 1, raw_input_id: <id>}`.
   - `reingestNote`: same for noteBus.
   - `reingestEvent`: same for eventBus.
   - `reingestBulk`: stubbed here; implemented in Phase 6.
   - Error mapping: NotFound → 404, InvalidArgument → 400, Internal → 500. Use `errs.New(errs.NotFound, err)` pattern.

4. **`reingestapp/route.go`:**
   - Follow `app/domain/rawinputapp/route.go` exactly. Apply auth middleware at `Routes.Add()` level.
   - Register:
     - `POST /api/v1/tasks/{task_id}/reingest`
     - `POST /api/v1/notes/{note_id}/reingest`
     - `POST /api/v1/events/{event_id}/reingest`
     - `POST /api/v1/reingest/bulk` (Phase 6)

5. **Wire in `api/services/planner/main.go`:**
   - Instantiate `reingestapp.Routes{...}` with all required bus deps (taskBus, noteBus, eventBus, rawinputBus, ingestBus, embeddingBus).
   - Call `reingestapp.Routes{}.Add(app, cfg)` where other app routes are registered.

6. **Response shape:** single-entity endpoints return JSON `{"queued": 1, "raw_input_id": "<uuid>", "status": "pending"}`. Bulk returns `{"queued": N, "entity_type": "...", "filter_matched": M}` (M may equal N; distinguish if any rows were skipped due to status=processing).

### Tests for this phase

- `TestReingestTask_DoesNotDuplicateCards` (apitest): create task (generates clarifications), reingest, reingest again → clarification count unchanged (Upsert guarantee from Phase 1).
- `TestReingestTask_LinkedEntitySkipsClassify` (apitest): create task + context, reingest → classify spy not invoked, embeddings + gaps ran.
- `TestReingestTask_UnlinkedRunsFullPipeline` (apitest): seed a raw_input without source_entity_id (legacy shape), ResetForReprocess, assert classify invoked.
- `TestReingestTask_PreservesConfirmedState` (apitest): confirmed task, reingest, assert `unconfirmed == false` after processing.
- `TestReingestTask_NotFound` (apitest): reingest on random UUID → 404.
- `TestReingestTask_NilRawInputID_Synthesizes` (apitest): seed task with RawInputID=nil, reingest → new raw_input created with Source=Manual, link established.
- `TestReingestNote` / `TestReingestEvent` (apitest): smoke tests for the other two entities.

### Definition of done

- `make test` passes across reingestapp + taskbus/notebus/eventbus.
- `POST /api/v1/tasks/{id}/reingest` works end-to-end on dev (hit with curl + X-API-Key).
- No duplicate clarification rows after repeated reingest calls.

---

## Phase 6: Bulk reingest endpoint

**Goal:** `POST /api/v1/reingest/bulk` with filter body — matches entities, resets their raw_inputs to pending, returns `{queued: N}`. Actual processing is async via IngestWorker.

### Files touched

- MODIFY `app/domain/reingestapp/reingestapp.go`
- MODIFY `app/domain/reingestapp/reingestapp_test.go`

### Implementation steps

1. **Request body schema:**
   ```
   {
     "entity_type": "task" | "note" | "event",
     "context_id": "<uuid>",      // optional
     "date_range": {              // optional
       "from": "2026-01-01T00:00:00Z",
       "to":   "2026-04-16T00:00:00Z"
     },
     "limit": 500                 // optional, default 1000, hard cap 5000
   }
   ```

2. **Handler flow (`reingestBulk`):**
   1. Parse + validate. `entity_type` required. Enforce the hard limit cap server-side (reject > 5000 with `errs.InvalidArgument`).
   2. Dispatch on entity_type:
      - `"task"` → build a `taskbus.QueryFilter` from `context_id` + `date_range` (start date ≤ `to`, created_at ≥ `from`). Call `taskBus.Query(ctx, filter, order, page)` with `page.Number=1, RowsPerPage=limit`.
      - Same pattern for note and event.
   3. For each returned entity, call the per-entity Reingest business method. Batch errors: continue on individual failures, accumulate count of successes vs. failures.
   4. Return `{"queued": successCount, "failed": failedCount, "entity_type": "task", "matched": totalMatched}`.

3. **Concurrency consideration:** calling Reingest sequentially for up to 5000 entities could exceed the request timeout. Two options:
   - **Sync loop with short per-op work (DB writes only, no ProcessRawInputByID inline):** Reingest only resets the raw_input status — a ~1-3ms operation. 5000 × 3ms = 15s, tight but acceptable if we cap at 1000 for the default. Document the tradeoff.
   - **Kick off a goroutine.** Return `{"queued": matched, "note": "processing in background"}` immediately, spin a goroutine that iterates and calls Reingest. Preferred for large batches — use `context.Background()` and structured logs for observability.
   Choose sync with a 1000-row default cap. Revisit if users hit the cap.

4. **Audit log.** Write a log line `bulk reingest start` and `bulk reingest done` with entity_type, counts, caller (if available via context). Use the existing structured logger.

5. **Idempotent semantics.** If a raw_input is already `status=processing`, skip it (ResetForReingest should refuse — match `ResetForReprocess` behavior). These rows count toward `skipped`, not `failed`.

### Tests for this phase

- `TestReingestBulk_EntityTypeFilter` (apitest): seed mix of tasks/notes/events, bulk with `entity_type=task` → only tasks reset.
- `TestReingestBulk_ContextFilter` (apitest): bulk with context_id → only matching tasks reset.
- `TestReingestBulk_DateRange` (apitest): bulk with date_range covering half the seeded set → only those reset.
- `TestReingestBulk_RespectsLimit` (apitest): request limit=10, seed 50, assert `queued ≤ 10`.
- `TestReingestBulk_RejectsOverLimit` (apitest): request limit=10000 → 400.
- `TestReingestBulk_SkipsProcessing` (apitest): seed raw_input with status=processing, bulk → included in `skipped`, not `queued`.

### Definition of done

- `make test` passes for reingestapp bulk tests.
- Manual curl against dev confirms queue count matches expected.
- Bulk response documents async nature (`"note": "processing via IngestWorker"` or similar).

---

## Phase 7: Backfill migration for pre-existing entities (optional)

**Goal:** One-time SQL to synthesize raw_inputs for entities with `raw_input_id IS NULL`, so that every row in tasks/notes/events has a backing raw_input going forward.

### Files touched

- CREATE `business/sdk/migrate/sql/<next_number>_backfill_manual_rawinputs.sql`

### Implementation steps

1. **Decide migration vs. lazy backfill.** Two options:
   - **Eager (this phase):** A single migration creates raw_inputs for every entity with nil RawInputID, then updates the entity rows to point at them.
   - **Lazy:** Phase 5's Reingest method already handles nil RawInputID by synthesizing at reingest time. No migration needed; entities without reingest never get a raw_input until someone triggers one.
   Pick eager only if we want a clean invariant ("every entity has a raw_input"). Lazy is cheaper and likely sufficient for now. **Recommended: lazy; mark this phase as "skip unless invariant is needed."**

2. **If eager is chosen:** SQL template:
   - For tasks: `INSERT INTO raw_inputs (id, source, content, status, skip_classify, reingest_mode, source_entity_id, source_entity_kind, created_at, updated_at) SELECT gen_random_uuid(), 'manual', '{"backfill": true, "task_id": "'||id||'"}', 'processed', true, false, id, 'task', created_at, NOW() FROM tasks WHERE raw_input_id IS NULL;`
   - Then: `UPDATE tasks SET raw_input_id = r.id FROM raw_inputs r WHERE r.source_entity_id = tasks.id AND r.source_entity_kind = 'task' AND tasks.raw_input_id IS NULL;`
   - Repeat for notes and events.

3. **Idempotency.** Since the migration framework runs each file once, idempotency matters only for re-runs in tests. Keep `WHERE raw_input_id IS NULL` so reruns are no-ops.

### Tests for this phase

- `TestBackfill_PreExistingTaskGetsRawInput` (dbtest): seed a task with `raw_input_id=NULL`. Run migration. Assert task now has a raw_input with Source=Manual, SkipClassify=true.
- `TestBackfill_DoesNotTouchLinkedTasks` (dbtest): seed tasks with existing raw_input_id. Run migration. Assert those raw_inputs untouched.

### Definition of done

- (If eager chosen) Migration applies cleanly; all entities have a non-null raw_input_id.
- (If lazy chosen) Phase 5 tests for `TestReingestTask_NilRawInputID_Synthesizes` cover the scenario; document the "lazy only" decision in `.docs/arch/rawinput-backend.md`.

---

## Testing strategy

| Phase | Test type | Key tests |
|-------|-----------|-----------|
| 1     | dbtest    | Upsert idempotency, answer preservation, dedup key respects kind |
| 2     | dbtest + unit | RawInput new field round-trip, defaults preserve legacy, Manual enum parse |
| 3     | apitest   | Manual create produces raw_input, bidirectional links |
| 4     | dbtest (ingestbus with fakes) | skip_classify branches, reingest_mode suppresses unconfirmed |
| 5     | apitest   | Reingest no-duplicate-cards, linked-entity-skips-classify, unlinked-runs-full, preserves-confirmed, not-found, nil-rawinput synthesis |
| 6     | apitest   | Bulk entity_type filter, context filter, date range, limit enforcement, skips-processing |
| 7     | dbtest    | Backfill correctness (only if eager) |

Mock/spy strategy for Phase 4: ingestbus tests already use fakes for classify/embedding buses. Extend the fakes to record invocation counts so "classify NOT invoked" is directly assertable.

All tests use real Postgres via `business/sdk/dbtest` on port 5433. `apitest` helpers build a test server per test.

## Rollout sequence

1. **Phase 1** (Clarification Upsert) — must complete first; all subsequent phases assume Upsert exists.
2. **Phase 2** (RawInput schema) — must complete before Phases 3 + 4.
3. **Phases 3 + 4 in parallel** — different files, no shared code paths. Phase 3 produces manual raw_inputs that rely on Phase 4's skip_classify branch, but testing each in isolation works via fakes.
4. **Phase 5** (single-entity reingest) — depends on 1, 2, 4.
5. **Phase 6** (bulk reingest) — extends Phase 5's handler file.
6. **Phase 7** (backfill) — optional, ship lazily unless invariant is needed. Can land any time after Phase 5.

Ship each phase as a separate commit with tests green. Update `.docs/arch/rawinput-backend.md`, `.docs/arch/clarification-backend.md`, and create `.docs/arch/reingest-backend.md` once the full chain is merged — not per-phase.

## Risks + mitigations

1. **Clarification dedup migration destroys data.** Rows violating the new UNIQUE constraint are deleted. *Mitigation:* the dedup step prefers rows with non-null answers; add a `SELECT COUNT(*)` warning emitted by the migration for audit; take a DB backup before running in prod (`make backup`).
2. **Manual create latency increases (30s wait for embeddings).** Phase 3 uses async IngestWorker pickup — users see a 30s gap before embeddings exist. *Mitigation:* ship Option A first, measure, switch to Option B (goroutine) if latency complaints surface.
3. **Bulk reingest overwhelms IngestWorker.** Resetting 1000 rows to pending means the worker must drain them on a single 30s tick. *Mitigation:* limit bulk default to 1000; IngestWorker already has concurrency caps (confirm and adjust if needed); monitor worker queue depth after first rollout.
4. **reingest_mode flag leaks.** If ResetForReingest forgets to clear reingest_mode on success, the flag sticks and breaks future fresh ingests on the same raw_input. *Mitigation:* ensure the "mark processed" step in ProcessRawInputByID clears reingest_mode (or tolerate it being set — it only affects unconfirmed flip, which is idempotent on already-confirmed entities).
5. **Pre-migration entities with nil RawInputID cause reingest 500s.** *Mitigation:* Phase 5 business methods defensively synthesize a raw_input when RawInputID is nil. Tests cover this. Phase 7 (eager backfill) is an extra safety net.

## Out of scope

- **New entity kinds beyond task/note/event.** The dispatch switch in ingestbus assumes these three. Adding a fourth (e.g., `contact`, `habit`) is a follow-up with explicit checklist updates across types/classifyapp/ingestbus/reingestapp.
- **Bulk-delete-via-reingest.** No "reingest and re-evaluate whether this entity should exist" semantic. Reingest never deletes.
- **UI surface for reingest.** No frontend button, no settings toggle — API-only in this feature. A later frontend phase can add a "reprocess" action on TaskDetailView/ContextDetailView.
- **Reingest on rawinputbus directly.** Users can already `POST /api/v1/rawinputs/{id}/reprocess` — that stays the raw-input-level operation. The new endpoints operate at the entity level.
- **Automatic reingest triggered by classifier/gap-detector version changes.** Manual invocation only. Auto-reingest on deploy is a separate, riskier feature.
- **Cross-entity reingest coordination.** If reingesting a task re-classifies it into a different context, we do NOT automatically reingest the old/new contexts' other tasks. That would be a much larger cascade and is explicitly deferred.
