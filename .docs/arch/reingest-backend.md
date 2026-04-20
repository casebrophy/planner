# Reingest Backend System

> The reingest domain enables users to reprocess existing tasks, notes, and events through the ingestion pipeline. Reingest supports both single-entity (by ID) and bulk operations (by entity type and filters). When a user triggers reingest, the associated raw_input is reset for reprocessing, and the IngestWorker pipeline handles the async extraction and classification. This enables rapid iteration on extraction prompts and classification logic without manual data re-entry.

## Core Concepts

**Reingest vs Reprocess:**
- **Reingest**: Full pipeline rerun with skip_classify flag based on entity's context linkage. Unlinked entities run full classification; linked entities skip to extraction. Set reingest_mode=TRUE to preserve confirmed state.
- **Reprocess**: Reset raw_input for retry, typically due to transient failures. Only valid for pending/failed items; blocked from processed items.

**Single-Entity Flow:**
1. Query task/note/event by ID
2. Check if entity has raw_input_id; if nil, synthesize one from entity content (lazy backfill)
3. Determine skip_classify flag based on context_id linkage
4. Delete unconfirmed entities if skip_classify=false
5. Reset raw_input (ResetForReingest or ResetForReprocess)
6. Return immediately; IngestWorker pipeline handles async processing

**Bulk Flow:**
1. Parse request: {entity_type, context_id?, date_range?}
2. Query all entities matching filters
3. For each entity, synthesize raw_input if needed (lazy backfill), then apply single-entity reingest logic
4. Return {queued: N} immediately
5. IngestWorker drains queue

## File Map

### Models
- `app/domain/reingestapp/model.go` — DTOs for reingest operations

### Handlers
- `app/domain/reingestapp/reingestapp.go` — **reingestTask()**, **reingestNote()**, **reingestEvent()**, **reingestBulk()** — Orchestrates single-entity and bulk reingest; queries entity by ID/filter, synthesizes raw_input if needed, determines skip_classify, dismisses stale clarifications, resets raw_input
- `app/domain/reingestapp/reingestapp.go` — **dismissStaleClarifications()** — Finds and dismisses pending/snoozed clarifications tied to raw_input or entity being reingested
- `app/domain/reingestapp/reingestapp.go` — **synthesizeRawInputForTask()**, **synthesizeRawInputForNote()**, **synthesizeRawInputForEvent()** — Creates raw_input from entity content (lazy backfill for pre-migration entities), updates entity with raw_input_id
- `app/domain/reingestapp/reingestapp.go` — **buildTaskContent()**, **buildEventContent()** — Helper functions to combine title and description into raw content

### Wiring
- `app/domain/reingestapp/route.go` — Dependency injection (taskBus, noteBus, eventBus, riBus, clarBus) and route registration

### Cross-Domain Coordination
- **taskbus.Business** — Query by ID, QueryByFilter, DeleteByRawInputUnconfirmed
- **notebus.Business** — Query by ID, QueryByFilter, DeleteByRawInputUnconfirmed
- **eventbus.Business** — Query by ID, QueryByFilter, DeleteByRawInputUnconfirmed
- **rawinputbus.Business** — ResetForReingest, ResetForReprocess, Update (for reingest_mode)
- **clarificationbus.Business** — Query (to find stale clarifications by subject_type/id), Dismiss (to mark as dismissed)

## Impact Callouts

### ⚠ Single-Entity Reingest Logic (app/domain/reingestapp/reingestapp.go, lines 36–156)
Reingest handlers synthesize raw_input when nil, then proceed with normal flow. Changes affect:
- **Synthesis logic** — if rawinputbus.Create, taskbus/notebus/eventbus.Update change signatures, synthesis breaks
- **Content extraction** — Task/Event use title+description; Note uses content; change in entity fields requires updating buildTaskContent/buildEventContent helpers
- **Skip_classify determination** — Currently uses context_id presence (task/event) or context_id || task_id (note); if rules change, all three handlers must be updated consistently
- **ResetForReingest vs ResetForReprocess** — Branching logic conditioned on skip_classify; if rawinputbus changes these methods, reingest fails (lines 209–222)
- **Stale clarification dismissal** — dismissStaleClarifications() queries for clarifications by subject_type ("raw_input", "task", "note", "event") and subject_id; if clarificationbus.Query or Dismiss interfaces change, dismissal breaks. Must use clarificationbus.DefaultOrderBy (line 175, 192). Dismissal happens before resetRawInput to ensure stale clarifications don't resurface in the ingest queue.

### ⚠ Bulk Reingest Logic (app/domain/reingestapp/reingestapp.go, lines 310–427)
Bulk reingest applies single-entity logic in a loop. Changes affect:
- **entityType switch** — Adding a new entity type requires a new case with corresponding query/reingest loop (lines 328–423), including dismissStaleClarifications call
- **Query methods** — queryTasksForBulkReingest/queryNotesForBulkReingest/queryEventsForBulkReingest all use page.New(1, 10000) (lines 429–462); if bulk operations exceed 10k items, pagination logic needed
- **Error handling** — Uses a.log.Warn for individual failures including dismissal failures; successful items still queue even if dismissal fails (lines 338, 346, 351, 355, etc.)
- **contextID filter** — Parses contextID from request; invalid UUID format returns 400 BadRequest (line 321)
- **Stale clarification dismissal** — Each entity in the bulk loop calls dismissStaleClarifications before resetRawInput (lines 350, 381, 412); dismissal failures are logged but don't block the reingest of that entity

### ⚠ Response Types (app/domain/reingestapp/model.go)
ReingestResponse (single-entity) and BulkReingestResponse must match client expectations:
- Single-entity response includes rawInputId (string UUID), skipClassify (bool), enqueued (bool)
- Bulk response includes only queued (int) — no detail on which entities failed

### ⚠ Routes (app/domain/reingestapp/route.go)
Endpoint registration at lines 46–48:
- POST /api/v1/tasks/{task_id}/reingest → reingestTask
- POST /api/v1/notes/{note_id}/reingest → reingestNote
- POST /api/v1/events/{event_id}/reingest → reingestEvent
- POST /api/v1/reingest/bulk → reingestBulk

Adding new entity types requires new routes and handlers.

## Database Schema

No dedicated schema; reingest operates on existing tables:
- **tasks/notes/events** — Requires raw_input_id, context_id
- **raw_inputs** — Modified by ResetForReingest/ResetForReprocess (status=pending, skip_classify, reingest_mode)

## Routes

| Method | Path | Handler | Request | Response | Permission |
|--------|------|---------|---------|----------|------------|
| POST | /api/v1/tasks/{task_id}/reingest | reingestTask | — | ReingestResponse | API Key Auth |
| POST | /api/v1/notes/{note_id}/reingest | reingestNote | — | ReingestResponse | API Key Auth |
| POST | /api/v1/events/{event_id}/reingest | reingestEvent | — | ReingestResponse | API Key Auth |
| POST | /api/v1/reingest/bulk | reingestBulk | BulkReingestRequest | BulkReingestResponse | API Key Auth |

## Cross-Domain Dependencies

### Upstream (Required)
- **rawinputbus** — ResetForReingest, ResetForReprocess, Update (to set reingest_mode)
- **taskbus** — QueryByID, Query, DeleteByRawInputUnconfirmed
- **notebus** — QueryByID, Query, DeleteByRawInputUnconfirmed
- **eventbus** — QueryByID, Query, DeleteByRawInputUnconfirmed

### Downstream (Consumers)
- **IngestWorker** — Processes reingest queue asynchronously; watches raw_input status=pending + reingest_mode=true

## Data Flow

**Single-Entity Reingest:**
1. POST /api/v1/tasks/{id}/reingest
2. Fetch task by ID from taskbus
3. Extract raw_input_id from task
4. Determine skip_classify based on task.context_id
5. If skip_classify=false, delete unconfirmed task copies
6. Find and dismiss pending/snoozed clarifications tied to raw_input_id and task_id via dismissStaleClarifications()
7. Reset raw_input (calls ResetForReingest or ResetForReprocess)
8. Return {rawInputId, skipClassify, enqueued: true}
9. IngestWorker picks up raw_input.status=pending + reingest_mode=true and processes

**Bulk Reingest:**
1. POST /api/v1/reingest/bulk {entityType: "task", contextId: "uuid?"}
2. Parse entity_type and optional context_id filter
3. Query all tasks (optionally filtered by context_id)
4. For each task: apply single-entity reingest logic (including dismissStaleClarifications), count successes
5. Return {queued: N}
6. IngestWorker processes all requeued raw_inputs

## Testing

### Unit Tests
- `app/domain/reingestapp/tests/reingestapi/reingest_test.go` — API integration tests
- Test fixtures: linked/unlinked tasks/notes/events with and without raw_inputs
- Coverage: single-entity reingest with skip_classify variations, auth failure, 404 (no entity)
- Coverage: nil-raw_input synthesis and reingest success
- Coverage: bulk reingest by entity_type and context_id filters, queued count, invalid inputs
- Coverage: Test_ReingestDismissesStaleClarifications — verifies clarifications tied to raw_input and entity are dismissed on reingest

### Test Expectations
- Linked entities (with context_id) → skip_classify=true
- Unlinked entities (no context_id) → skip_classify=false (task/event); notes always linked (context_id or task_id required)
- Reingest sets status=pending, reingest_mode=true
- Nil raw_input_id entities synthesize raw_input with Manual source and proceed normally
- Single-entity returns ReingestResponse; bulk returns BulkReingestResponse
- Auth failure → 401 Unauthorized
- Invalid entity_id → 404 NotFound
- Entity without raw_input is synthesized and reingested successfully (200 OK)
- Clarifications with subject_type in (raw_input, task, note, event) matching the entity are dismissed before reingest completes

## Notes

- **Lazy Backfill:** Entities without raw_input_id (pre-migration) are handled defensively: reingest synthesizes a raw_input from entity content on-the-fly, updates the entity with the raw_input_id, and proceeds normally. This avoids eager migrations and handles backfill lazily per-entity during reingest.
- **Async Processing:** Reingest returns immediately; actual reprocessing happens asynchronously via IngestWorker. Client cannot poll for completion.
- **Unconfirmed Deletion:** If skip_classify=false, reingestapp deletes any unconfirmed (Tier 3) copies of the entity. Confirmed entities are preserved.
- **Pagination for Bulk:** Bulk queries use page.New(1, 10000); if a single context has >10k items, only the first 10k are reingested. Migration to streaming or multiple pages needed for enterprise-scale use.
- **Error Handling:** Bulk reingest logs individual failures with a.log.Warn but continues; client only sees total queued count, not which items failed.
- **No Date Range Filter Yet:** BulkReingestRequest.DateRange exists in model but is not yet implemented; bulk queries ignore it.
