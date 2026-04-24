# Reclassify Backend System

> Bidirectional reclassification domain enabling conversion between tasks and notes. Converts task → note (combining title+description as content, preserving context/tags/raw_input, recording classification correction) and note → task (splitting first line as title, remainder as description, defaulting to Open/Medium/Medium, marking as Unconfirmed). Includes preflight validation: refuses recurring tasks, tasks with dependents, tasks scheduled in today's plan. Manages transaction boundaries and tag migration between domains.

## Core Types

### Business Layer

```go
type Bus struct {
    log  *logger.Logger
    task *taskbus.Business
    note *notebus.Business
    corr *classificationcorrectionbus.Business
    db   *sqlx.DB
}

// TaskToNote input/output
// Input: taskID uuid.UUID
// Output: notebus.Note
// Error conditions: not found, recurring, has dependents, in today's plan

// NoteToTask input/output
// Input: noteID uuid.UUID
// Output: taskbus.Task
// Error conditions: not found
```

### Store Layer Storer Additions

Cross-domain Storer interface extensions (no new store domain; reclassifybus composes existing stores):

- **taskbus.Storer**: uses existing Create, Delete, QueryByID, DeleteWithTx
- **notebus.Storer**: uses existing Create, Delete, QueryByID, CreateWithTx
- **classificationcorrectionbus.Storer**: uses existing CreateWithTx for recording reclassification event

### App Layer (HTTP)

```go
type convertTaskToNoteRequest
// Path: /api/v1/tasks/{task_id}/convert-to-note
// Response: noteapp.Note (from notebus.Note)

type convertNoteToTaskRequest
// Path: /api/v1/notes/{note_id}/convert-to-task
// Response: taskapp.Task (from taskbus.Task)
```

## Business Methods

Core logic in `business/domain/reclassifybus/reclassifybus.go`:

- `TaskToNote(ctx context.Context, taskID uuid.UUID) (notebus.Note, error)` — converts task to note with validation, transaction boundary, tag copying, correction recording
- `NoteToTask(ctx context.Context, noteID uuid.UUID) (taskbus.Task, error)` — converts note to task with transaction boundary, tag copying, correction recording
- `preflightTaskToNote(ctx context.Context, t taskbus.Task) error` — validation: refuses recurring, tasks with recurrence children, tasks with dependents, tasks scheduled in today's or future daily plan
- `copyTaskTagsToNoteTags(ctx context.Context, tx *sqlx.Tx, taskID, noteID uuid.UUID) error` — SQL INSERT into note_tags SELECT from task_tags
- `copyNoteTagsToTaskTags(ctx context.Context, tx *sqlx.Tx, noteID, taskID uuid.UUID) error` — SQL INSERT into task_tags SELECT from note_tags

## File Map

### Core Business Logic

- **Business**: `business/domain/reclassifybus/reclassifybus.go` (258 lines) — Bus struct, TaskToNote(), NoteToTask(), preflight validation, tag copying helpers
- **Models**: `business/domain/reclassifybus/model.go` (empty file, no types defined in this domain)
- **Tests**: `business/domain/reclassifybus/reclassifybus_test.go` (287 lines) — TestTaskToNote_HappyPath, TestTaskToNote_RefusesRecurring, TestTaskToNote_RefusesWithDependents, TestNoteToTask_HappyPath, TestNoteToTask_SingleLineContent

### HTTP Handlers

- `app/domain/reclassifyapp/reclassifyapp.go` (97 lines) — app struct with `log *logger.Logger`, `reclassifyBus *reclassifybus.Bus` fields; handlers convertTaskToNote(), convertNoteToTask(); error mapping via mapError() (NotFound, InvalidArgument, AlreadyExists, Internal)
- `app/domain/reclassifyapp/route.go` (39 lines) — Routes.Add() wires taskBus (via taskdb store + dependency store), noteBus, corrBus, reclassifyBus; registers POST /api/v1/tasks/{task_id}/convert-to-note and POST /api/v1/notes/{note_id}/convert-to-task with auth middleware
- **Tests**: `app/domain/reclassifyapp/reclassifyapp_test.go` (224 lines) — TestConvertTaskToNote, TestConvertNoteToTask, TestConvertTaskToNoteWithRecurrence, TestConvertTaskNotFound

## Routes

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| POST | `/api/v1/tasks/{task_id}/convert-to-note` | convertTaskToNote | Convert task to note; validates preflight checks; returns noteapp.Note |
| POST | `/api/v1/notes/{note_id}/convert-to-task` | convertNoteToTask | Convert note to task; returns taskapp.Task with defaults (Open, Medium priority, Medium energy, Unconfirmed=true) |

All routes require `X-API-Key` auth header (mid.Auth middleware).

## Transaction Boundaries

Both methods use explicit transaction management with rollback on error:

**TaskToNote transaction steps:**
1. Fetch task (outside tx)
2. Preflight validation (outside tx)
3. BEGIN transaction
4. notebus.CreateWithTx() — inserts new note
5. copyTaskTagsToNoteTags() — INSERT note_tags SELECT task_tags (ON CONFLICT DO NOTHING)
6. corrBus.RecordWithTx() — record correction (predicted_type="task", actual_type="note")
7. taskbus.DeleteWithTx() — hard delete task
8. COMMIT
9. Return created note

**NoteToTask transaction steps:**
1. Fetch note (outside tx)
2. BEGIN transaction
3. taskbus.CreateWithTx() — inserts new task (title from first line, description from remainder, Status=Open, Priority=Medium, Energy=Medium, Unconfirmed=true)
4. copyNoteTagsToTaskTags() — INSERT task_tags SELECT note_tags (ON CONFLICT DO NOTHING)
5. corrBus.RecordWithTx() — record correction (predicted_type="note", actual_type="task")
6. notebus.DeleteWithTx() — hard delete note
7. COMMIT
8. Return created task

## Impact Callouts

### ⚠ TaskToNote validation — strict preflight gates

**Refused conditions:**
1. `RecurrenceRule != nil` — cannot convert recurring tasks
2. Recurrence children exist (SELECT COUNT(*) FROM tasks WHERE recurrence_parent_id = $1) — cannot convert parents of recurrence chains
3. Dependent tasks exist (SELECT task_id FROM task_dependencies WHERE depends_on_id = $1) — cannot convert blocked-by tasks
4. Task scheduled in today's or future daily plan (SELECT EXISTS(SELECT 1 FROM daily_plan_entries WHERE task_id = $1 AND plan_date >= NOW()::date)) — scheduling conflict

**Error mapping (app layer mapError()):**
- "fetch task" or "fetch note" containing "no rows" → 404 NotFound
- "cannot convert recurring" or "cannot convert" + "dependents" or "cannot convert" + "recurrence children" → 400 InvalidArgument
- "cannot convert task scheduled in today's plan" → 409 AlreadyExists
- Default → 500 Internal

### ⚠ Content transformation — task title+description → note content

**TaskToNote:**
- content = task.Title
- if task.Description != "" then content += "\n\n" + task.Description
- Preserves task.ContextID, task.RawInputID, task.Unconfirmed
- Sets note.Source = "reclassified_from_task"
- Sets note.TaskID = nil

**NoteToTask:**
- Splits note.Content on first newline
- title = first line (trimmed)
- description = remainder (trimmed); empty if no remainder
- Defaults: Status=Open, Priority=Medium, Energy=Medium, Unconfirmed=true
- Preserves note.ContextID, note.RawInputID
- Sets task.TaskID = nil (note is deleted, not linked)

### ⚠ Tag migration — bidirectional copy with conflict handling

Both directions use `ON CONFLICT DO NOTHING` to handle tags already present:

```sql
INSERT INTO note_tags (note_id, tag_id)
SELECT $1, tag_id FROM task_tags WHERE task_id = $2
ON CONFLICT DO NOTHING
```

- Tags are copied, not moved
- Both original and converted entity are deleted, so duplicate tags won't occur
- If tag deletion cascades occur, tag_id FK constraint errors will bubble up

### ⚠ Classification correction recording

Every conversion records a correction entry via classificationcorrectionbus.RecordWithTx():

**TaskToNote:**
```go
NewCorrection{
    ClauseText:    task.Title,
    PredictedType: "task",
    Confidence:    0,
    ActualType:    "note",
    Source:        "correction_applied",
}
```

**NoteToTask:**
```go
NewCorrection{
    ClauseText:    note.Content,
    PredictedType: "note",
    Confidence:    0,
    ActualType:    "task",
    Source:        "correction_applied",
}
```

- Confidence always 0 (manual override, not ML-driven)
- Source = "correction_applied" enables filtering corrections by application vs. user feedback
- Failures in correction recording abort the transaction

### ⚠ RawInputID preservation

Both conversions preserve task.RawInputID / note.RawInputID:
- If task came from ingestion (RawInputID set), the new note retains that RawInputID
- If note came from ingestion (RawInputID set), the new task retains that RawInputID
- Enables tracking conversions back to source ingestion events
- Important for reingest pipeline (Phase 7) to link reclassified entities to original raw inputs

### ⚠ Unconfirmed field propagation

- TaskToNote: preserves task.Unconfirmed → note.Unconfirmed
- NoteToTask: sets new task.Unconfirmed = true (always, regardless of note.Unconfirmed)
  - Rationale: notes are less structured; converting to task marks for review
  - Follows pattern from notebus.NoteToTask (Phase 5 notes)

### ⚠ Cross-store Storer method additions

reclassifybus does NOT add new Storer methods to taskbus/notebus. Instead, it uses existing methods:

- **taskbus.Storer**: Create, Delete, QueryByID, DeleteWithTx (added in Phase 5 for transaction support)
- **notebus.Storer**: Create, Delete, QueryByID, CreateWithTx
- **classificationcorrectionbus.Storer**: Create, CreateWithTx (used for correction recording)

If any of these methods were missing (e.g., CreateWithTx on taskbus/notebus), reclassifybus.TaskToNote/NoteToTask would fail at compile time.

### ⚠ Null/pointer field handling

Fields handled carefully across conversions:

**TaskToNote preserves:**
- task.ContextID (*uuid.UUID, may be nil) → note.ContextID
- task.RawInputID (*uuid.UUID, may be nil) → note.RawInputID
- task.Unconfirmed (bool) → note.Unconfirmed

**NoteToTask preserves:**
- note.ContextID (*uuid.UUID, may be nil) → task.ContextID
- note.RawInputID (*uuid.UUID, may be nil) → task.RawInputID
- Always sets task.Unconfirmed = true (not nil; bool is non-nullable)

**Discarded fields:**
- TaskToNote discards: task.Status, Priority, Energy, DurationMin, DueDate, ScheduledAt, ExpectedUpdateDays, BlockedReason, DebriefStatus, CompletedAt, RecurrenceRule, RecurrenceParentID, TrackOutcome
- NoteToTask creates new task with defaults; note.Source, note.TaskID discarded

## Cross-Domain Dependencies

- **taskbus**: TaskToNote reads task, validates preflight, deletes task; NoteToTask creates task with defaults
- **notebus**: TaskToNote creates note; NoteToTask reads note, deletes note
- **classificationcorrectionbus**: Records both conversions as correction events (feedback loop for classifier tuning)
- **task_dependencies**: checked during TaskToNote preflight (SELECT COUNT(*) query)
- **daily_plan_entries**: checked during TaskToNote preflight (SELECT EXISTS query)
- **task_tags / note_tags**: migrated via SQL INSERT ... SELECT ON CONFLICT DO NOTHING
- **raw_inputs**: RawInputID preserved across both conversions (FK, may be NULL)

## Database Schema (Read-Only for reclassifybus)

reclassifybus does not define its own tables. It operates on:

- **tasks** (task_id FK preserved via RawInputID)
- **notes** (note_id FK preserved via RawInputID)
- **task_tags** (migrated to note_tags via copyTaskTagsToNoteTags)
- **note_tags** (migrated to task_tags via copyNoteTagsToTaskTags)
- **classification_corrections** (new rows inserted for each conversion)
- **task_dependencies** (checked, not modified)
- **daily_plan_entries** (checked, not modified)

## No Separate Wiring

reclassifybus is wired once in `api/services/planner/main.go` at the app layer:

```go
// Fetch existing stores/buses
taskStore := taskdb.NewStore(cfg.Log, cfg.DB)
depStore := taskdb.NewDependencyStore(cfg.Log, cfg.DB)
taskBus := taskbus.NewBusiness(cfg.Log, taskStore, depStore)

noteStore := notedb.NewStore(cfg.Log, cfg.DB)
noteBus := notebus.NewBusiness(cfg.Log, noteStore)

corrStore := correctiondb.NewStore(cfg.Log, cfg.DB)
corrBus := classificationcorrectionbus.NewBusiness(cfg.Log, corrStore)

// Wire reclassifybus
reclassifyBus := reclassifybus.NewBusiness(cfg.Log, taskBus, noteBus, corrBus, cfg.DB)

// App handler
hdl := &app{log: cfg.Log, reclassifyBus: reclassifyBus}
a.Handle(http.MethodPost, "/api/v1/tasks/{task_id}/convert-to-note", hdl.convertTaskToNote, authen)
a.Handle(http.MethodPost, "/api/v1/notes/{note_id}/convert-to-task", hdl.convertNoteToTask, authen)
```

No separate business layer wiring (no NewBusiness called in main.go except at app layer Routes.Add()).

## Enums

- **taskstatus**: open, blocked, done, dismissed (NoteToTask always defaults to "open")
- **taskpriority**: low, medium, high, urgent (NoteToTask always defaults to "medium")
- **taskenergy**: low, medium, high (NoteToTask always defaults to "medium")

## Updates

- **Phase 7 (2026-04-24)**: Initial implementation — bidirectional task↔note conversion with preflight validation, transaction boundaries, tag migration, classification correction recording.
