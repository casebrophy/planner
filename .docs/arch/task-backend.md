# Task Backend System

> The task domain is the primary work-tracking entity. It supports full CRUD, filtering by status/priority/context/due-date range, ordering by six fields, and pagination. Tasks may be optionally linked to a context via a nullable FK. Status transitions are managed in the business layer, including automatic `completed_at` stamping when status transitions to `done`. All nine routes (CRUD + 4 dependency routes) are protected by API-key auth.
>
> **New features (current):**
> - **Recurrence:** `RecurrenceRule` (optional RRULE string) and `RecurrenceParentID` (FK to parent task). When a recurring task transitions to `done`, `CreateNextRecurrence()` auto-generates the next instance via the recurrence engine.
> - **Task dependencies:** 2-table model (`tasks` + `task_dependencies`) with cycle detection. Completing an upstream task auto-unblocks downstream dependents. Auto-blocking: adding a dependency to an incomplete upstream task immediately blocks the downstream task if `open`.
> - **BlockedReason:** Explicit text reason for blocked status. Used to distinguish dependency-blocks from manual blocks. `reevaluateBlocked()` re-opens a task if blocked with no reason and all unmet dependencies are resolved.
> - **New status values:** `open`, `blocked`, `done`, `dismissed` (replaces `todo`, `in_progress`, `cancelled`). Migration updates CHECK constraint.
> - **Thread/debrief columns (migration v1.11):** `expected_update_days REAL`, `last_thread_at TIMESTAMPTZ`, `debrief_status TEXT` — wired through all three layers. `DebriefStatus` defaults to `pending` on creation. `LastThreadAt` is system-managed (not exposed in the update DTO).

---

## Core Types

### App Layer — `app/domain/taskapp/model.go`

```go
// Response DTO — returned by all read and write handlers.
type Task struct {
    ID                 string   `json:"id"`
    ContextID          *string  `json:"contextId,omitempty"`
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
}

// Request body for POST /api/v1/tasks.
type NewTask struct {
    Title          string  `json:"title"`
    Description    string  `json:"description"`
    ContextID      *string `json:"contextId"`
    Priority       string  `json:"priority"`
    Energy         string  `json:"energy"`
    DurationMin    *int    `json:"durationMin"`
    DueDate        *string `json:"dueDate"`
    RecurrenceRule *string `json:"recurrenceRule"`
}

// Request body for PUT /api/v1/tasks/{task_id}. All fields optional.
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
}
```

### Business Layer — `business/domain/taskbus/model.go`

```go
type Task struct {
    ID                 uuid.UUID
    ContextID          *uuid.UUID
    Title              string
    Description        string
    Status             taskstatus.Status
    Priority           taskpriority.Priority
    Energy             taskenergy.Energy
    DurationMin        *int
    DueDate            *time.Time
    ScheduledAt        *time.Time
    ExpectedUpdateDays *float64
    LastThreadAt       *time.Time
    DebriefStatus      debriefstatus.Status
    BlockedReason      string
    CreatedAt          time.Time
    UpdatedAt          time.Time
    CompletedAt        *time.Time
    RecurrenceRule     *string
    RecurrenceParentID *uuid.UUID
}

type NewTask struct {
    Title          string
    Description    string
    ContextID      *uuid.UUID
    Status         taskstatus.Status
    Priority       taskpriority.Priority
    Energy         taskenergy.Energy
    DurationMin    *int
    DueDate        *time.Time
    RecurrenceRule *string
}

type UpdateTask struct {
    Title              *string
    Description        *string
    ContextID          *uuid.UUID
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
}
```

### Business Layer — `business/domain/taskbus/filter.go`

```go
type QueryFilter struct {
    ID           *uuid.UUID
    Status       *taskstatus.Status
    Priority     *taskpriority.Priority
    ContextID    *uuid.UUID
    StartDueDate *time.Time
    EndDueDate   *time.Time
}
```

### Business Layer — `business/domain/taskbus/taskbus.go`

```go
type Storer interface {
    Create(ctx context.Context, task Task) error
    Update(ctx context.Context, task Task) error
    Delete(ctx context.Context, task Task) error
    Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Task, error)
    Count(ctx context.Context, filter QueryFilter) (int, error)
    QueryByID(ctx context.Context, id uuid.UUID) (Task, error)
    DismissTasksByContext(ctx context.Context, contextID uuid.UUID) (int, error)
}

type DependencyStorer interface {
    AddDependency(ctx context.Context, dep Dependency) error
    RemoveDependency(ctx context.Context, taskID, dependsOnID uuid.UUID) error
    QueryDependencies(ctx context.Context, taskID uuid.UUID) ([]Task, error)
    QueryDependents(ctx context.Context, taskID uuid.UUID) ([]Task, error)
    HasUnmetDependencies(ctx context.Context, taskID uuid.UUID) (bool, error)
}

type Dependency struct {
    TaskID      uuid.UUID
    DependsOnID uuid.UUID
    CreatedAt   time.Time
}
```

### Store Layer — `business/domain/taskbus/stores/taskdb/model.go`

```go
// Internal struct used only within taskdb. Maps to the tasks table via sqlx db tags.
// Enums are stored as strings; converters handle the typed↔string translation.
type taskDB struct {
    ID                 uuid.UUID  `db:"task_id"`
    ContextID          *uuid.UUID `db:"context_id"`
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
}
```

### Enum Types

`business/types/taskstatus/` — values: `open`, `blocked`, `done`, `dismissed` (migrated from `todo`, `in_progress`, `cancelled`)
`business/types/taskpriority/` — values: `low`, `medium`, `high`, `urgent`
`business/types/taskenergy/` — values: `low`, `medium`, `high`
`business/types/debriefstatus/` — values: `pending`, `done`, `skipped`

All enums expose `Parse(s string) (T, error)`, `MustParse(s string) T`, `String() string`, and text marshaling.

### Database Schema

#### `tasks` table (migration current)

```sql
CREATE TABLE tasks (
    task_id            UUID        NOT NULL DEFAULT gen_random_uuid(),
    context_id         UUID        REFERENCES contexts(context_id) ON DELETE SET NULL,
    title              TEXT        NOT NULL,
    description        TEXT        NOT NULL DEFAULT '',
    status             TEXT        NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'blocked', 'done', 'dismissed')),
    priority           TEXT        NOT NULL DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    energy             TEXT        NOT NULL DEFAULT 'medium' CHECK (energy IN ('low', 'medium', 'high')),
    duration_min       INTEGER,
    due_date           TIMESTAMPTZ,
    scheduled_at       TIMESTAMPTZ,
    expected_update_days REAL,
    last_thread_at     TIMESTAMPTZ,
    debrief_status     TEXT        NOT NULL DEFAULT 'pending' CHECK (debrief_status IN ('pending', 'done', 'skipped')),
    blocked_reason     TEXT        NOT NULL DEFAULT '',
    recurrence_rule    TEXT,
    recurrence_parent_id UUID      REFERENCES tasks(task_id) ON DELETE SET NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at       TIMESTAMPTZ,
    PRIMARY KEY (task_id)
);
CREATE INDEX idx_tasks_status        ON tasks(status);
CREATE INDEX idx_tasks_context       ON tasks(context_id);
CREATE INDEX idx_tasks_due           ON tasks(due_date) WHERE due_date IS NOT NULL;
CREATE INDEX idx_tasks_priority      ON tasks(priority);
CREATE INDEX idx_tasks_recurrence    ON tasks(recurrence_parent_id) WHERE recurrence_parent_id IS NOT NULL;
```

#### `task_dependencies` table (new)

```sql
CREATE TABLE task_dependencies (
    task_id       UUID        NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    depends_on_id UUID        NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (task_id, depends_on_id),
    CONSTRAINT no_self_dep CHECK (task_id != depends_on_id)
);
CREATE INDEX idx_task_deps_upstream ON task_dependencies(depends_on_id);
```

---

## File Map

### App Layer (`app/domain/taskapp/`)

- `taskapp.go` — **create()**, **update()**, **delete()**, **queryAll()**, **queryByID()** — HTTP handlers; decode request, call business layer, return app DTO or `errs.Error`; `update` and `delete` check `sqldb.ErrDBNotFound` and return 404
- `dependency.go` — **addDependency()**, **removeDependency()**, **queryDependencies()**, **queryDependents()** — dependency management handlers; `addDependency` enforces cycle detection and auto-blocking logic via business layer; returns 204 No Content on success
- `model.go` — **toAppTask()**, **toAppTasks()**, **toBusNewTask()**, **toBusUpdateTask()** — type converters between app and business layers; all time fields formatted as RFC3339 strings; `Task.Encode()` implements `web.Encoder`; handles UUID and string marshaling for `RecurrenceRule` and `RecurrenceParentID`
- `filter.go` — **parseFilter()** — maps query params (`status`, `priority`, `context_id`, `start_due_date`, `end_due_date`) to `taskbus.QueryFilter`; returns error on invalid enum or UUID values
- `order.go` — **parseOrder()** — maps `orderBy` query param string to `order.By` via `orderByFields` map; falls back to `taskbus.DefaultOrderBy` (`created_at DESC`)
- `route.go` — **Routes.Add()** — instantiates `taskdb.NewStore`, `taskdb.NewDependencyStore`, and `taskbus.NewBusiness`; registers nine endpoints (CRUD + 4 dependency routes) with `mid.Auth` middleware

### Business Layer (`business/domain/taskbus/`)

- `taskbus.go` — **NewBusiness()**, **Create()**, **Update()**, **Delete()**, **Query()**, **Count()**, **QueryByID()**, **DismissTasksByContext()** — domain logic; `Create` generates UUID, sets `CreatedAt`/`UpdatedAt`, and defaults `DebriefStatus` to `pending`; `Update` merges patch fields, auto-sets `CompletedAt` on first transition to `done`, triggers `UnblockDependents()`, and creates next recurrence if applicable; defines `Storer` interface
- `dependency.go` — **AddDependency()**, **RemoveDependency()**, **QueryDependencies()**, **QueryDependents()**, **UnblockDependents()**, **reevaluateBlocked()**, **CreateNextRecurrence()** — dependency and recurrence logic; `AddDependency` checks for cycles and auto-blocks downstream if upstream incomplete; `reevaluateBlocked()` re-opens tasks with no explicit `BlockedReason` if all unmet dependencies resolve; `CreateNextRecurrence()` parses the recurrence rule and generates the next task instance; defines `DependencyStorer` interface
- `model.go` — `Task`, `NewTask`, `UpdateTask`, `Dependency` — domain structs with strongly-typed enum fields
- `filter.go` — `QueryFilter` — shared filter struct consumed by both business Query/Count and store applyFilter
- `order.go` — order field constants and `DefaultOrderBy`

### Store Layer (`business/domain/taskbus/stores/taskdb/`)

- `taskdb.go` — **NewStore()**, **Create()**, **Update()**, **Delete()**, **Query()**, **Count()**, **QueryByID()**, **DismissTasksByContext()** — SQL implementations using `foundation/sqldb` helpers; `Query` builds dynamic SQL via string buffer + `applyFilter` + `orderByClause` + OFFSET/FETCH pagination; `Create` and `Update` include `blocked_reason`, `recurrence_rule`, `recurrence_parent_id` columns
- `dependency.go` — **NewDependencyStore()**, **AddDependency()**, **RemoveDependency()**, **QueryDependencies()**, **QueryDependents()**, **HasUnmetDependencies()** — SQL implementations for the `task_dependencies` table; `QueryDependencies` returns tasks that the given task depends on; `QueryDependents` returns tasks that depend on the given task (reverse); `HasUnmetDependencies` checks if any upstream task is not `done`
- `model.go` — `taskDB` (unexported), **toDBTask()**, **toBusTask()**, **toBusTasks()** — sqlx-tagged struct; enums serialized to strings in `toDBTask`, parsed back via `MustParse` in `toBusTask`; handles UUID/string conversion for `RecurrenceParentID`
- `filter.go` — **applyFilter()** — appends `AND` clauses to query buffer for each non-nil filter field; uses named params in `data` map
- `order.go` — `orderByFields` map (business constant → SQL column name); **orderByClause()** — returns `"column direction"` or error on unknown field

---

## Impact Callouts

### ⚠ taskbus.Task (`business/domain/taskbus/model.go`)

Adding, removing, or renaming a field affects:

- `taskbus/taskbus.go` — `Create()` builds a `Task` literal from `NewTask` (must include new field); `Update()` merges `UpdateTask` onto `Task` (must handle new field); may trigger `CreateNextRecurrence()` or `UnblockDependents()` on status changes
- `taskbus/dependency.go` — `CreateNextRecurrence()` may read new fields when cloning a task; auto-blocking logic may check new status or blocker fields
- `taskdb/model.go` — `toDBTask()` maps every `Task` field to a `taskDB` field; `toBusTask()` maps back — both converters must be kept in sync
- `taskdb/taskdb.go` — SQL INSERT column list and `:named` params in `Create()`; UPDATE SET clause in `Update()`; SELECT column list in `Query()` and `QueryByID()` — all must include the new column
- `taskapp/model.go` — `toAppTask()` maps `taskbus.Task` → `app.Task`; add field to `app.Task` struct and converter; handle time/UUID formatting for response

### ⚠ taskDB (`business/domain/taskbus/stores/taskdb/model.go`)

Adding or removing a `db`-tagged field affects:

- `taskdb/taskdb.go` — INSERT column list (Create), UPDATE SET clause (Update), SELECT column list (Query, QueryByID) must exactly match the struct's db tags; sqlx will silently miss columns not in the SELECT list
- `toDBTask()` and `toBusTask()` converters in the same file — new fields must be mapped in both directions

### ⚠ Storer interface (`business/domain/taskbus/taskbus.go`)

Adding or changing a method signature affects:

- `taskdb/taskdb.go` — `*Store` must implement the new/changed method or the build fails
- Any future test doubles or mock implementations of `Storer`

### ⚠ QueryFilter (`business/domain/taskbus/filter.go`)

Adding a filter field affects:

- `taskdb/filter.go` — `applyFilter()` must add an `if` branch appending the SQL `AND` clause and setting the data map key
- `taskapp/filter.go` — `parseFilter()` must add parsing of the new query param and assignment to the filter field

### ⚠ Order constants (`business/domain/taskbus/order.go`)

Adding a new `OrderBy*` constant affects:

- `taskdb/order.go` — `orderByFields` map must add `constant → SQL column` mapping
- `taskapp/order.go` — `orderByFields` map must add `"request string" → constant` mapping

### ⚠ Enum values (`business/types/taskstatus`, `taskpriority`, `taskenergy`, `debriefstatus`)

Adding a new value affects:

- `business/sdk/migrate/sql/migrate.sql` — `CHECK` constraint on the `tasks` table (`status`, `energy`, `priority`) or `debrief_status` column must include the new value (requires ALTER TABLE or a new migration version)
- Converters `toBusTask()` and `toBusUpdateTask()` — `MustParse`/`Parse` will panic or error on unknown values until the enum is updated
- `taskbus/taskbus.go` — logic that checks for specific status values (e.g., `if Status == taskstatus.Done`) may need adjustment; auto-unblocking and recurrence creation depend on `Done` status specifically

### ⚠ Recurrence: `RecurrenceRule`, `RecurrenceParentID`

Recurrence fields are wired through all three layers and tied to business logic:

1. `taskbus/model.go` — `Task` and `NewTask` both have `RecurrenceRule` (*string); `CreateNextRecurrence()` in business layer reads these
2. `taskbus/dependency.go` — `CreateNextRecurrence()` parses the RRULE string, computes the next due date, and creates a new Task with `RecurrenceParentID` pointing to the original
3. `taskbus/taskbus.go` — `Update()` calls `CreateNextRecurrence()` if task transitions to `Done` and has a `RecurrenceRule`
4. `taskdb/model.go` — `taskDB` has `db`-tagged fields; converters handle UUID ↔ string conversion
5. `taskdb/taskdb.go` — all INSERT/UPDATE/SELECT SQL include the columns; indexes on `recurrence_parent_id` for querying instance chains
6. `taskapp/model.go` — response and update DTOs include both fields as pointers

**Cycle prevention:** Self-linking via `RecurrenceParentID` is prevented by the database (CHECK constraint `recurrence_parent_id != task_id`); query chains via `RecurrenceParentID` for task history.

### ⚠ Task dependencies: `task_dependencies` table, `Dependency` struct, `DependencyStorer` interface

Dependencies create a second table and parallel store layer:

1. `taskbus/model.go` — `Dependency` struct with `TaskID`, `DependsOnID`, `CreatedAt`
2. `taskbus/dependency.go` — all dependency methods; `AddDependency()` enforces no cycles (direct only, not transitive) and auto-blocks downstream if upstream incomplete; `RemoveDependency()` and `UnblockDependents()` re-evaluate blocked status
3. `taskbus/taskbus.go` — wires `DependencyStorer` into `NewBusiness()`; `Update()` calls `UnblockDependents()` when task transitions to `Done`
4. `taskdb/dependency.go` — SQL queries on `task_dependencies` table; `QueryDependencies` and `QueryDependents` fetch the Task rows via JOIN
5. `taskapp/dependency.go` — four handlers: `addDependency`, `removeDependency`, `queryDependencies`, `queryDependents`
6. `taskapp/route.go` — four additional routes for dependency management

**Cycle detection:** `AddDependency()` queries upstream dependencies via `QueryDependencies()` and checks if the downstream task already exists as a dependency of the upstream. This catches direct cycles only; transitive cycles are not explicitly prevented (design choice: assumes UI prevents multi-hop cycles).

### ⚠ Thread/debrief columns: `expected_update_days`, `last_thread_at`, `debrief_status`

These columns are wired through all three layers. When modifying them:

1. `taskbus/model.go` — `Task` struct has the fields; `UpdateTask` has `ExpectedUpdateDays` and `DebriefStatus` (not `LastThreadAt` — system-managed)
2. `taskdb/model.go` — `taskDB` has `db`-tagged fields; `toDBTask()` and `toBusTask()` handle conversion
3. `taskdb/taskdb.go` — all INSERT/UPDATE/SELECT SQL include the columns
4. `taskapp/model.go` — response DTO includes all three; update DTO includes `ExpectedUpdateDays` and `DebriefStatus`
5. `business/types/debriefstatus/` — enum type: `pending`, `done`, `skipped`

### ⚠ BlockedReason field

`BlockedReason` is a string field (not an enum) used to distinguish manual blocks from dependency-induced blocks:

1. `taskbus/model.go` — `Task` has `BlockedReason` (string); `UpdateTask` has optional `BlockedReason` pointer
2. `taskbus/dependency.go` — `reevaluateBlocked()` checks `if task.BlockedReason != ""` to protect manually-blocked tasks; if reason is empty and all dependencies resolve, the task is re-opened
3. `taskdb/model.go` and `taskdb/taskdb.go` — standard string column with default empty string
4. `taskapp/model.go` — response and update DTOs both include `BlockedReason` as string (response) or *string (update)

**Semantics:** If a task is blocked and has a non-empty `BlockedReason`, dependency resolution will NOT auto-unblock it (manual hold). If `BlockedReason` is empty and unmet dependencies exist, the task stays blocked. If `BlockedReason` is empty and all dependencies resolve, the task auto-unblocks to `Open`.

---

## Routes

### CRUD Routes

| Method | Path | Handler | Auth | Returns |
|--------|------|---------|------|---------|
| GET | `/api/v1/tasks` | `queryAll` | X-API-Key | `{ tasks: Task[], total: int, page: int, rowsPerPage: int }` |
| GET | `/api/v1/tasks/{task_id}` | `queryByID` | X-API-Key | `Task` |
| POST | `/api/v1/tasks` | `create` | X-API-Key | `Task` (201 Created) |
| PUT | `/api/v1/tasks/{task_id}` | `update` | X-API-Key | `Task` |
| DELETE | `/api/v1/tasks/{task_id}` | `delete` | X-API-Key | 204 No Content |

### Dependency Routes

| Method | Path | Handler | Auth | Returns |
|--------|------|---------|------|---------|
| POST | `/api/v1/tasks/{task_id}/dependencies/{depends_on_id}` | `addDependency` | X-API-Key | 204 No Content |
| DELETE | `/api/v1/tasks/{task_id}/dependencies/{depends_on_id}` | `removeDependency` | X-API-Key | 204 No Content |
| GET | `/api/v1/tasks/{task_id}/dependencies` | `queryDependencies` | X-API-Key | `Task[]` (tasks that this task depends on) |
| GET | `/api/v1/tasks/{task_id}/dependents` | `queryDependents` | X-API-Key | `Task[]` (tasks that depend on this task) |

### Query Parameters

`GET /api/v1/tasks` supports:
- Pagination: `page` (number, default 1), `rows` (per page, default 20)
- Ordering: `orderBy` (id/title/status/priority/due_date/created_at, default created_at DESC)
- Filtering: `status` (open/blocked/done/dismissed), `priority` (low/medium/high/urgent), `context_id` (UUID), `start_due_date` (RFC3339), `end_due_date` (RFC3339)

### Defaults

`POST /api/v1/tasks` defaults: `status=open`, `priority=medium`, `energy=medium`. `title` is required. `description` defaults to empty string.

---

## Cross-Domain Dependencies

- **contexts** — `tasks.context_id` FK references `contexts.context_id` ON DELETE SET NULL. `QueryFilter.ContextID` and the `context_id` query param filter tasks by context. `DismissTasksByContext()` sets all open/blocked tasks in a context to `dismissed` when the context is deleted.
- **tags** — `task_tags` join table (migration 1.04) links tasks to tags. Tag assignment is a separate domain (`tagbus`/`tagdb`); `taskapp` has no awareness of tags.
- **threadbus** — `last_thread_at` column tracks the most recent thread entry touching a task; system-managed (not exposed in REST API).
- **debriefstatus type** — `business/types/debriefstatus/` enum: `pending`, `done`, `skipped`. Defaults to `pending` on task creation; updatable via REST API.
- **recurrence engine** — `business/types/recurrence` module provides `Parse(rule string) (Rule, error)` and `Rule.NextOccurrence(from time.Time) time.Time`. Called by `CreateNextRecurrence()` to compute next due date.
- **page SDK** (`business/sdk/page`) — `queryAll` uses `page.Parse` and `page.Page` for OFFSET/FETCH pagination.
- **order SDK** (`business/sdk/order`) — `Query` uses `order.By{Field, Direction}`; field constants live in `taskbus/order.go`.
- **sqldb** (`foundation/sqldb`) — store uses `NamedExecContext`, `NamedQuerySlice`, `NamedQueryStruct`; returns `sqldb.ErrDBNotFound` (wraps `sql.ErrNoRows`) on missing rows — handlers must check this explicitly to return 404.
- **logger** — `foundation/logger` is injected into both `Store` and `DependencyStore` for structured logging.
- **web framework** — `foundation/web.App`, `web.Encoder`, `web.Param()`, `web.Decode()` for HTTP handling; `app/sdk/mid.Auth()` for API key authentication; `app/sdk/errs` for error responses.
