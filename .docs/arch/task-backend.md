# Task Backend System

> Core task management domain supporting CRUD, filtering, ordering, status/priority/energy enums, due dates, recurrence rules, scheduling, duration estimation, and dependency-based blocking logic. Manages task dependencies (upstream/downstream) with cycle prevention and automatic blocking/unblocking transitions. When a task completes, dependent tasks are re-evaluated and recurrence rules generate the next instance.

## Core Types

### Business Layer

```go
type Task struct {
    ID                 uuid.UUID
    ContextID          *uuid.UUID
    RawInputID         *uuid.UUID
    Title              string
    Description        string
    Status             taskstatus.Status        // open, blocked, done, dismissed
    Priority           taskpriority.Priority    // low, medium, high, urgent
    Energy             taskenergy.Energy        // low, medium, high
    DurationMin        *int
    DueDate            *time.Time
    ScheduledAt        *time.Time
    ExpectedUpdateDays *float64
    LastThreadAt       *time.Time
    DebriefStatus      debriefstatus.Status     // pending, in_progress, complete
    BlockedReason      string
    Unconfirmed        bool
    CreatedAt          time.Time
    UpdatedAt          time.Time
    CompletedAt        *time.Time
    RecurrenceRule     *string                  // RRULE format
    RecurrenceParentID *uuid.UUID
    TrackOutcome       bool
}

type NewTask struct {
    Title              string
    Description        string
    ContextID          *uuid.UUID
    RawInputID         *uuid.UUID
    Status             taskstatus.Status
    Priority           taskpriority.Priority
    Energy             taskenergy.Energy
    DurationMin        *int
    DueDate            *time.Time
    RecurrenceRule     *string
    TrackOutcome       bool
    Unconfirmed        bool
}

type UpdateTask struct {
    Title              *string
    Description        *string
    ContextID          *uuid.UUID
    RawInputID         *uuid.UUID
    Status             *taskstatus.Status
    Priority           *taskpriority.Priority
    Energy             *taskenergy.Energy
    DurationMin        *int
    DueDate            *time.Time
    ScheduledAt        *time.Time
    ExpectedUpdateDays *float64
    BlockedReason      *string
    DebriefStatus      *debriefstatus.Status
    RecurrenceRule     *string
    TrackOutcome       *bool
    Unconfirmed        *bool
}
```

### Store Layer

```go
type taskDB struct {
    ID                 uuid.UUID  `db:"task_id"`
    ContextID          *uuid.UUID `db:"context_id"`
    RawInputID         *uuid.UUID `db:"raw_input_id"`
    Title              string     `db:"title"`
    Description        string     `db:"description"`
    Status             string     `db:"status"`
    Priority           string     `db:"priority"`
    Energy             string     `db:"energy"`
    DurationMin        *int       `db:"duration_min"`
    DueDate            *time.Time `db:"due_date"`
    ScheduledAt        *time.Time `db:"scheduled_at"`
    ExpectedUpdateDays *float64   `db:"expected_update_days"`
    LastThreadAt       *time.Time `db:"last_thread_at"`
    BlockedReason      string     `db:"blocked_reason"`
    DebriefStatus      string     `db:"debrief_status"`
    CreatedAt          time.Time  `db:"created_at"`
    UpdatedAt          time.Time  `db:"updated_at"`
    CompletedAt        *time.Time `db:"completed_at"`
    RecurrenceRule     *string    `db:"recurrence_rule"`
    RecurrenceParentID *uuid.UUID `db:"recurrence_parent_id"`
    TrackOutcome       bool       `db:"track_outcome"`
    Unconfirmed        bool       `db:"unconfirmed"`
}
```

### App Layer (HTTP)

```go
type Task struct {
    ID                 string   `json:"id"`
    ContextID          *string  `json:"contextId,omitempty"`
    RawInputID         *string  `json:"rawInputId,omitempty"`
    Title              string   `json:"title"`
    Description        string   `json:"description"`
    Status             string   `json:"status"`
    Priority           string   `json:"priority"`
    Energy             string   `json:"energy"`
    DurationMin        *int     `json:"durationMin,omitempty"`
    DueDate            *string  `json:"dueDate,omitempty"`
    ScheduledAt        *string  `json:"scheduledAt,omitempty"`
    ExpectedUpdateDays *float64 `json:"expectedUpdateDays,omitempty"`
    LastThreadAt       *string  `json:"lastThreadAt,omitempty"`
    BlockedReason      string   `json:"blockedReason,omitempty"`
    DebriefStatus      string   `json:"debriefStatus"`
    CreatedAt          string   `json:"createdAt"`
    UpdatedAt          string   `json:"updatedAt"`
    CompletedAt        *string  `json:"completedAt,omitempty"`
    RecurrenceRule     *string  `json:"recurrenceRule,omitempty"`
    RecurrenceParentID *string  `json:"recurrenceParentId,omitempty"`
    TrackOutcome       bool     `json:"trackOutcome"`
    Unconfirmed        bool     `json:"unconfirmed"`
}

type NewTask struct {
    Title              string  `json:"title"`
    Description        string  `json:"description"`
    ContextID          *string `json:"contextId"`
    RawInputID         *string `json:"rawInputId"`
    Priority           string  `json:"priority"`
    Energy             string  `json:"energy"`
    DurationMin        *int    `json:"durationMin"`
    DueDate            *string `json:"dueDate"`
    RecurrenceRule     *string `json:"recurrenceRule"`
}

type UpdateTask struct {
    Title              *string  `json:"title"`
    Description        *string  `json:"description"`
    ContextID          *string  `json:"contextId"`
    Status             *string  `json:"status"`
    Priority           *string  `json:"priority"`
    Energy             *string  `json:"energy"`
    DurationMin        *int     `json:"durationMin"`
    DueDate            *string  `json:"dueDate"`
    ScheduledAt        *string  `json:"scheduledAt"`
    ExpectedUpdateDays *float64 `json:"expectedUpdateDays"`
    BlockedReason      *string  `json:"blockedReason"`
    DebriefStatus      *string  `json:"debriefStatus"`
    RecurrenceRule     *string  `json:"recurrenceRule"`
    TrackOutcome       *bool    `json:"trackOutcome,omitempty"`
}
```

## Storer Interface

```go
type Storer interface {
    Create(ctx context.Context, task Task) error
    Update(ctx context.Context, task Task) error
    Delete(ctx context.Context, task Task) error
    DeleteBatch(ctx context.Context, ids []uuid.UUID) error
    Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Task, error)
    Count(ctx context.Context, filter QueryFilter) (int, error)
    QueryByID(ctx context.Context, id uuid.UUID) (Task, error)
    DismissTasksByContext(ctx context.Context, contextID uuid.UUID) (int, error)
    DeleteByRawInputUnconfirmed(ctx context.Context, rawInputID uuid.UUID) error
    ResetByContext(ctx context.Context, contextID uuid.UUID) error
}
```

## Business Methods

Core business logic in `business/domain/taskbus/taskbus.go`:

- `Create(ctx, nt NewTask) (Task, error)` — creates a new task, generates UUID, sets timestamps, defaults DebriefStatus to "pending"
- `Update(ctx, task Task, ut UpdateTask) (Task, error)` — patches fields, sets CompletedAt when moving to "done", triggers UnblockDependents(), CreateNextRecurrence() if needed
- `Delete(ctx, task Task) error` — hard delete
- `DeleteBatch(ctx, ids []uuid.UUID) error` — delete multiple tasks by ID
- `DeleteByRawInputUnconfirmed(ctx, rawInputID uuid.UUID) error` — clean up unconfirmed ingested tasks before reingest
- `Query(ctx, filter, orderBy, pg) ([]Task, error)` — paginated filtered query
- `Count(ctx, filter) (int, error)` — count matching tasks
- `QueryByID(ctx, id uuid.UUID) (Task, error)` — fetch single task
- `DismissTasksByContext(ctx, contextID uuid.UUID) (int, error)` — dismiss all open/blocked tasks for a context
- `ResetByContext(ctx, contextID uuid.UUID) error` — reset all done tasks in a context back to open
- `CreateNextRecurrence(ctx, task Task) (Task, error)` — creates next occurrence from a completed recurring task
- `UnblockDependents(ctx, taskID uuid.UUID) error` — called when a task is marked done; unblocks dependent tasks
- `AddDependency(ctx, taskID uuid.UUID, dependsOnID uuid.UUID) error` — adds blocking relationship
- `RemoveDependency(ctx, taskID uuid.UUID, dependsOnID uuid.UUID) error` — removes blocking relationship
- `QueryDependencies(ctx, taskID uuid.UUID) ([]uuid.UUID, error)` — tasks this task depends on
- `QueryDependents(ctx, taskID uuid.UUID) ([]uuid.UUID, error)` — tasks blocked by this task

## File Map

### Models

- **Business**: `business/domain/taskbus/model.go` (70 lines) — Task, NewTask, UpdateTask definitions
- **Store**: `business/domain/taskbus/stores/taskdb/model.go` (100 lines) — taskDB struct with db tags, toDBTask(), toBusTask(), toBusTasks() converters
- **App**: `app/domain/taskapp/model.go` (287 lines) — Task, NewTask, UpdateTask request/response DTOs; toAppTask(), toBusNewTask(), toBusUpdateTask() converters; DeleteBatchRequest struct

### Core Business Logic

- `business/domain/taskbus/taskbus.go` (249 lines) — Business struct, Storer interface, all business methods
- `business/domain/taskbus/taskbus_test.go` (383 lines) — unit tests for Create, Update, recurrence logic

### Store Implementation

- `business/domain/taskbus/stores/taskdb/taskdb.go` (222 lines) — SQL implementation of Storer interface (Create, Update, Delete, DeleteBatch, Query, Count, QueryByID, DismissTasksByContext, DeleteByRawInputUnconfirmed, ResetByContext)
- `business/domain/taskbus/stores/taskdb/filter.go` (52 lines) — applyFilter() helper, WHERE clause construction
- `business/domain/taskbus/stores/taskdb/order.go` (25 lines) — orderByFields map, orderByClause() for sorting

### Handlers

- `app/domain/taskapp/taskapp.go` (249 lines) — HTTP handler struct with `log *logger.Logger` field; HTTP handler methods:
  - `create(ctx, r)` — POST /api/v1/tasks; synthesizes Manual raw_input (Status=Processed, SkipClassify=true) before task creation, links raw_input back via UpdateSourceEntity; spawns goroutines for thread entry, embeddings, knowledge gap detection
  - `update(ctx, r)` — PUT /api/v1/tasks/{task_id}; fires thread entry (Update/Milestone), debrief on completion
  - `delete(ctx, r)` — DELETE /api/v1/tasks/{task_id}; single task delete
  - `deleteBatch(ctx, r)` — DELETE /api/v1/tasks/batch; batch delete
  - `queryAll(ctx, r)` — GET /api/v1/tasks; paginated list with filters, ordering
  - `queryByID(ctx, r)` — GET /api/v1/tasks/{task_id}; fetch single
  - `addDependency(ctx, r)` — POST /api/v1/tasks/{task_id}/dependencies/{depends_on_id}
  - `removeDependency(ctx, r)` — DELETE /api/v1/tasks/{task_id}/dependencies/{depends_on_id}
  - `queryDependencies(ctx, r)` — GET /api/v1/tasks/{task_id}/dependencies
  - `queryDependents(ctx, r)` — GET /api/v1/tasks/{task_id}/dependents

### Routes & Wiring

- `app/domain/taskapp/route.go` (57 lines) — Routes.Add() wires taskBus, threadBus, debriefBus, embeddingBus, gapBus, rawinputBus into handler constructor with `cfg.Log`; registers all endpoints with auth and activity logging middlewares
- `app/domain/taskapp/filter.go` (83 lines) — parseFilter() converts query params (status, contextID, priority, etc.) to QueryFilter
- `app/domain/taskapp/order.go` (21 lines) — parseOrder() converts request fields to business order constants

### Dependencies (Task Blocking)

- `business/domain/taskbus/dependency.go` (132 lines) — DependencyStorer interface, Business methods for Add/Remove/Query
- `business/domain/taskbus/stores/taskdb/dependency.go` (122 lines) — SQL implementation (task_dependencies table)
- `app/domain/taskapp/dependency.go` (84 lines) — handlers for dependency CRUD

### Tests

- `app/domain/taskapp/tests/taskapi/` — integration tests:
  - `create_test.go` — task creation, defaults, validation
  - `update_test.go` — field patches, status transitions, completion side-effects
  - `delete_test.go` — single and batch delete
  - `query_test.go` — filtering, pagination, ordering
  - `task_test.go` — general CRUD
  - `seed_test.go` — seed fixtures
  - `rawinput_test.go` — TestManualTaskCreate_ProducesRawInput: asserts raw_input row is created and linked on manual task creation (Phase 3)

## Database Schema

```sql
CREATE TABLE tasks (
    task_id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    context_id           UUID REFERENCES contexts(context_id) ON DELETE SET NULL,
    raw_input_id         UUID REFERENCES raw_inputs(raw_input_id),
    title                TEXT NOT NULL,
    description          TEXT DEFAULT '',
    status               TEXT CHECK (status IN ('open', 'blocked', 'done', 'dismissed')),
    priority             TEXT CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    energy               TEXT CHECK (energy IN ('low', 'medium', 'high')),
    duration_min         INTEGER,
    due_date             TIMESTAMPTZ,
    scheduled_at         TIMESTAMPTZ,
    expected_update_days REAL,
    last_thread_at       TIMESTAMPTZ,
    debrief_status       TEXT CHECK (debrief_status IN ('pending', 'in_progress', 'complete')),
    blocked_reason       TEXT DEFAULT '',
    created_at           TIMESTAMPTZ DEFAULT NOW(),
    updated_at           TIMESTAMPTZ DEFAULT NOW(),
    completed_at         TIMESTAMPTZ,
    recurrence_rule      TEXT,
    recurrence_parent_id UUID REFERENCES tasks(task_id),
    track_outcome        BOOLEAN DEFAULT false,
    unconfirmed          BOOLEAN DEFAULT false
);

CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_context ON tasks(context_id);
CREATE INDEX idx_tasks_due ON tasks(due_date) WHERE due_date IS NOT NULL;
CREATE INDEX idx_tasks_priority ON tasks(priority);

CREATE TABLE task_dependencies (
    task_id       UUID NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    depends_on_id UUID NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (task_id, depends_on_id),
    CHECK (task_id != depends_on_id)
);
CREATE INDEX idx_task_deps_depends_on ON task_dependencies(depends_on_id);
```

## Routes

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/api/v1/tasks` | queryAll | List tasks (paginated, filterable, sortable) |
| GET | `/api/v1/tasks/{task_id}` | queryByID | Fetch single task |
| POST | `/api/v1/tasks` | create | Create task; synthesizes Manual raw_input, fires thread entry, embeddings, knowledge gap detection |
| PUT | `/api/v1/tasks/{task_id}` | update | Patch task; fires thread entry (Milestone on completion), debrief cards |
| DELETE | `/api/v1/tasks/{task_id}` | delete | Hard delete single task |
| DELETE | `/api/v1/tasks/batch` | deleteBatch | Batch delete by IDs |
| POST | `/api/v1/tasks/{task_id}/dependencies/{depends_on_id}` | addDependency | Add blocking dependency |
| DELETE | `/api/v1/tasks/{task_id}/dependencies/{depends_on_id}` | removeDependency | Remove blocking dependency |
| GET | `/api/v1/tasks/{task_id}/dependencies` | queryDependencies | List tasks this task depends on |
| GET | `/api/v1/tasks/{task_id}/dependents` | queryDependents | List tasks blocked by this task |

All routes require `X-API-Key` auth header (mid.Auth middleware). PUT route also triggers activity logging (mid.ActivityLog).

## Impact Callouts

### ⚠ Task struct — used across all 3 layers

**Business layer usage:**
- All business methods read/mutate Task
- CompletedAt set when status → "done"
- RecurrenceRule drives CreateNextRecurrence()
- DebriefStatus controls debrief card generation

**Store layer usage:**
- toDBTask() converts Task → taskDB (string enums)
- toBusTask() converts taskDB → Task (typed enums via MustParse)
- All SQL queries (INSERT/UPDATE) use task_id, context_id, raw_input_id, title, description, status, priority, energy, duration_min, due_date, scheduled_at, expected_update_days, last_thread_at, debrief_status, blocked_reason, created_at, updated_at, completed_at, recurrence_rule, recurrence_parent_id, track_outcome, unconfirmed

**App layer usage:**
- toAppTask() converts Task → JSON response (timestamps as RFC3339 strings)
- toBusNewTask() parses NewTask request → Task with defaults (Status=open, DebriefStatus=pending)
- toBusUpdateTask() parses UpdateTask → patches only provided fields

**Affected by schema changes:**
- New field requires: SQL migration → taskDB struct + tags → converters → Task struct → handlers → routes
- Removing field: reverse order (routes/handlers first, test before dropping column)

### ⚠ Storer interface — all methods must be implemented

Adding a new Storer method:
1. Add to `business/domain/taskbus/taskbus.go` Storer interface
2. Implement in `business/domain/taskbus/stores/taskdb/taskdb.go`
3. Add Business wrapper method if needed for validation/logging
4. Wire into route handler if exposed via HTTP

Changing method signature:
- All handlers calling the method must pass new parameters
- All tests using mockStore must update implementations
- SQL queries must be updated in taskdb.go

Example: DismissTasksByContext() returns (int, error) — the count is used by callers to know how many tasks were affected.

### ⚠ Task Status field — enum with strict values

Valid statuses: `open`, `blocked`, `done`, `dismissed`

**Database validation:**
- Enforced via CHECK constraint: `status IN ('open', 'blocked', 'done', 'dismissed')`
- INSERT/UPDATE with invalid status fails at DB level

**Type conversion:**
- Store layer: taskDB.Status string → taskbus.Task.Status via taskstatus.MustParse()
- App layer: UpdateTask.Status *string → taskbus.UpdateTask.Status *taskstatus.Status via taskstatus.Parse()

**Business logic dependencies:**
- Update() checks if status → "done" and auto-sets CompletedAt
- Update() triggers UnblockDependents() and CreateNextRecurrence() on "done"
- App handler fires thread Milestone entry on "done"
- App handler fires debrief card on "done"

**Affected by adding new status:**
- Add to taskstatus enum type (`business/types/taskstatus/`)
- Add to CHECK constraint in migration SQL
- Update Business logic for any new state transitions
- Update App handlers if new status affects side-effects

### ⚠ Filtering & Ordering

**Filter struct** in `business/domain/taskbus/filter.go`:
- Fields: Status, ContextID, Priority, Energy, HasRecurrence, TrackOutcome, Unconfirmed, etc.
- Passed from app parseFilter() → Business Query() → Store applyFilter()

**Order struct** in `business/domain/taskbus/order.go`:
- Constants like "status", "created_at", "due_date", "priority"
- Passed from app parseOrder() → Business Query() → Store orderByClause()

**Cross-layer tracing:**
- New filter field: add to business QueryFilter → app parseFilter() (parse from query param) → store applyFilter() (build WHERE clause)
- New order field: add to business order constants → app parseOrder() (request field name) → store orderByFields (SQL column name)

### ⚠ Recurrence system

**Fields:**
- Task.RecurrenceRule: RRULE string (e.g., "FREQ=DAILY")
- Task.RecurrenceParentID: UUID of parent task (for linking recurrence chains)

**Business logic:**
- CreateNextRecurrence() called when status → "done" and RecurrenceRule is set
- Uses recurrence.Parse() to compute NextOccurrence(from time.Time)
- Creates new Task with ID=uuid.New(), copies core fields (title, description, priority, energy, duration_min), sets DueDate=nextDue, RecurrenceParentID=&parent.ID

**Affected by changes:**
- Modifying RecurrenceRule format requires updating recurrence package
- Removing RecurrenceRule field requires removing CreateNextRecurrence() call and testing

### ⚠ Task Dependency system

**DependencyStorer interface** in `business/domain/taskbus/dependency.go`:
- `AddDependency(ctx, taskID, dependsOnID) error` — task_id depends on depends_on_id
- `RemoveDependency(ctx, taskID, dependsOnID) error`
- `QueryDependencies(ctx, taskID) ([]uuid.UUID, error)` — tasks this depends on
- `QueryDependents(ctx, taskID) ([]uuid.UUID, error)` — tasks depending on this
- `UnblockDependents(ctx, taskID) error` — called when taskID is marked done; sets dependent tasks back to "open" if they were "blocked"

**Database:**
- task_dependencies(task_id, depends_on_id, created_at) with CHECK(task_id != depends_on_id)
- ON DELETE CASCADE ensures cleanup when a task is deleted

**Cross-domain:**
- Blocks are stored separately from task status; status "blocked" is set by app logic when dependencies exist

### ⚠ RawInputID field — links to ingestion pipeline

Task.RawInputID references raw_inputs(raw_input_id):
- Set during task creation if task was generated from an ingested email/file/voice
- DeleteByRawInputUnconfirmed() cleans up unconfirmed tasks before reingest
- Used by ingestbus to track which tasks came from which raw input

### ⚠ raw_input synthesis on manual task creation (Phase 3)

Handler create() now synthesizes a raw_input row before calling taskBus.Create():
1. `rawinputBus.Create()` — creates raw_input with SourceType=Manual, Status=Processed, SkipClassify=true, SourceEntityKind="task"; sets bt.RawInputID before task insert
2. `rawinputBus.UpdateSourceEntity()` — links raw_input back to the created task ID (called synchronously after task creation)

If task creation fails after raw_input is created, the orphaned raw_input ID is logged. UpdateSourceEntity errors are logged but do not fail the request.

### ⚠ Embed-on-create and gap-detect-on-create

Handler create() spawns goroutines:
1. threadBus.AddEntry() — async thread entry logging
2. embeddingBus.EmbedAndStore() — async vector embedding with error logging to `a.log`
3. gapBus.Detect() — async knowledge gap detection with error logging to `a.log`

Async operations don't fail the create. Embedding and gap-detect errors are explicitly logged via `a.log.Error()` instead of being silently dropped.

## Cross-Domain Dependencies

- **contextbus**: Task.ContextID references contexts; task queries and mutations flow through context lifecycle
- **threadbus**: Task creation/update writes thread entries; threads track task history
- **debriefbus**: Task completion (status → "done") triggers debrief card generation
- **embeddingbus**: Task title+description embedded for semantic search
- **knowledgegapbus**: Task content analyzed for knowledge gaps (detected at create/update)
- **activitylogbus**: Task updates logged for activity dashboard (via mid.ActivityLog)
- **rawinputbus**: Manual task creation synthesizes a raw_input row (Phase 3); Task.RawInputID links to ingested or manual-capture content; used by IngestWorker during reingest (Phase 5)

## Enums

- **taskstatus**: open, blocked, done, dismissed
- **taskpriority**: low, medium, high, urgent
- **taskenergy**: low, medium, high
- **debriefstatus**: pending, in_progress, complete
