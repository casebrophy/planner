# Task Backend System

> The task domain is the primary work-tracking entity. It supports full CRUD, filtering by status/priority/context/due-date range, ordering by six fields, and pagination. Tasks may be optionally linked to a context via a nullable FK. Status transitions are managed in the business layer, including automatic `completed_at` stamping when status transitions to `done`. All five routes are protected by API-key auth.
>
> **Pending schema additions (not yet in code or migration):** `expected_update_days REAL`, `last_thread_at TIMESTAMPTZ`, `debrief_status TEXT` — once added to the DB these must propagate through all three layers (see Impact Callouts below for the full cascade).

---

## Core Types

### App Layer — `app/domain/taskapp/model.go`

```go
// Response DTO — returned by all read and write handlers.
type Task struct {
    ID          string  `json:"id"`
    ContextID   *string `json:"contextId,omitempty"`
    Title       string  `json:"title"`
    Description string  `json:"description"`
    Status      string  `json:"status"`
    Priority    string  `json:"priority"`
    Energy      string  `json:"energy"`
    DurationMin *int    `json:"durationMin,omitempty"`
    DueDate     *string `json:"dueDate,omitempty"`
    ScheduledAt *string `json:"scheduledAt,omitempty"`
    CreatedAt   string  `json:"createdAt"`
    UpdatedAt   string  `json:"updatedAt"`
    CompletedAt *string `json:"completedAt,omitempty"`
}

// Request body for POST /api/v1/tasks.
type NewTask struct {
    Title       string  `json:"title"`
    Description string  `json:"description"`
    ContextID   *string `json:"contextId"`
    Priority    string  `json:"priority"`
    Energy      string  `json:"energy"`
    DurationMin *int    `json:"durationMin"`
    DueDate     *string `json:"dueDate"`
}

// Request body for PUT /api/v1/tasks/{task_id}. All fields optional.
type UpdateTask struct {
    Title       *string `json:"title"`
    Description *string `json:"description"`
    ContextID   *string `json:"contextId"`
    Status      *string `json:"status"`
    Priority    *string `json:"priority"`
    Energy      *string `json:"energy"`
    DurationMin *int    `json:"durationMin"`
    DueDate     *string `json:"dueDate"`
    ScheduledAt *string `json:"scheduledAt"`
}
```

### Business Layer — `business/domain/taskbus/model.go`

```go
type Task struct {
    ID          uuid.UUID
    ContextID   *uuid.UUID
    Title       string
    Description string
    Status      taskstatus.Status
    Priority    taskpriority.Priority
    Energy      taskenergy.Energy
    DurationMin *int
    DueDate     *time.Time
    ScheduledAt *time.Time
    CreatedAt   time.Time
    UpdatedAt   time.Time
    CompletedAt *time.Time
}

type NewTask struct {
    Title       string
    Description string
    ContextID   *uuid.UUID
    Status      taskstatus.Status
    Priority    taskpriority.Priority
    Energy      taskenergy.Energy
    DurationMin *int
    DueDate     *time.Time
}

type UpdateTask struct {
    Title       *string
    Description *string
    ContextID   *uuid.UUID
    Status      *taskstatus.Status
    Priority    *taskpriority.Priority
    Energy      *taskenergy.Energy
    DurationMin *int
    DueDate     *time.Time
    ScheduledAt *time.Time
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
}
```

### Store Layer — `business/domain/taskbus/stores/taskdb/model.go`

```go
// Internal struct used only within taskdb. Maps to the tasks table via sqlx db tags.
// Enums are stored as strings; converters handle the typed↔string translation.
type taskDB struct {
    ID          uuid.UUID  `db:"task_id"`
    ContextID   *uuid.UUID `db:"context_id"`
    Title       string     `db:"title"`
    Description string     `db:"description"`
    Status      string     `db:"status"`
    Priority    string     `db:"priority"`
    Energy      string     `db:"energy"`
    DurationMin *int       `db:"duration_min"`
    DueDate     *time.Time `db:"due_date"`
    ScheduledAt *time.Time `db:"scheduled_at"`
    CreatedAt   time.Time  `db:"created_at"`
    UpdatedAt   time.Time  `db:"updated_at"`
    CompletedAt *time.Time `db:"completed_at"`
}
```

### Enum Types

`business/types/taskstatus/` — values: `todo`, `in_progress`, `done`, `cancelled`
`business/types/taskpriority/` — values: `low`, `medium`, `high`, `urgent`
`business/types/taskenergy/` — values: `low`, `medium`, `high`

All enums expose `Parse(s string) (T, error)`, `MustParse(s string) T`, `String() string`, and text marshaling.

### Database Schema — `business/sdk/migrate/sql/migrate.sql` (version 1.03)

```sql
CREATE TABLE tasks (
    task_id       UUID        NOT NULL DEFAULT gen_random_uuid(),
    context_id    UUID        REFERENCES contexts(context_id) ON DELETE SET NULL,
    title         TEXT        NOT NULL,
    description   TEXT        NOT NULL DEFAULT '',
    status        TEXT        NOT NULL DEFAULT 'todo' CHECK (status IN ('todo', 'in_progress', 'done', 'cancelled')),
    priority      TEXT        NOT NULL DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    energy        TEXT        NOT NULL DEFAULT 'medium' CHECK (energy IN ('low', 'medium', 'high')),
    duration_min  INTEGER,
    due_date      TIMESTAMPTZ,
    scheduled_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at  TIMESTAMPTZ,
    PRIMARY KEY (task_id)
);
CREATE INDEX idx_tasks_status   ON tasks(status);
CREATE INDEX idx_tasks_context  ON tasks(context_id);
CREATE INDEX idx_tasks_due      ON tasks(due_date) WHERE due_date IS NOT NULL;
CREATE INDEX idx_tasks_priority ON tasks(priority);
```

---

## File Map

### App Layer (`app/domain/taskapp/`)

- `taskapp.go` — **create()**, **update()**, **delete()**, **queryAll()**, **queryByID()** — HTTP handlers; decode request, call business layer, return app DTO or `errs.Error`; `update` and `delete` check `sqldb.ErrDBNotFound` and return 404
- `model.go` — **toAppTask()**, **toAppTasks()**, **toBusNewTask()**, **toBusUpdateTask()** — type converters between app and business layers; all time fields formatted as RFC3339 strings; `Task.Encode()` implements `web.Encoder`
- `filter.go` — **parseFilter()** — maps query params (`status`, `priority`, `context_id`, `start_due_date`, `end_due_date`) to `taskbus.QueryFilter`; returns error on invalid enum or UUID values
- `order.go` — **parseOrder()** — maps `orderBy` query param string to `order.By` via `orderByFields` map; falls back to `taskbus.DefaultOrderBy` (`created_at DESC`)
- `route.go` — **Routes.Add()** — instantiates `taskdb.NewStore` and `taskbus.NewBusiness`; registers five endpoints with `mid.Auth` middleware

### Business Layer (`business/domain/taskbus/`)

- `taskbus.go` — **NewBusiness()**, **Create()**, **Update()**, **Delete()**, **Query()**, **Count()**, **QueryByID()** — domain logic; `Create` generates UUID and sets `CreatedAt`/`UpdatedAt`; `Update` merges patch fields and auto-sets `CompletedAt` on first transition to `done`; defines `Storer` interface
- `model.go` — `Task`, `NewTask`, `UpdateTask` — domain structs with strongly-typed enum fields
- `filter.go` — `QueryFilter` — shared filter struct consumed by both business Query/Count and store applyFilter
- `order.go` — order field constants and `DefaultOrderBy`

### Store Layer (`business/domain/taskbus/stores/taskdb/`)

- `taskdb.go` — **NewStore()**, **Create()**, **Update()**, **Delete()**, **Query()**, **Count()**, **QueryByID()** — SQL implementations using `foundation/sqldb` helpers; `Query` builds dynamic SQL via string buffer + `applyFilter` + `orderByClause` + OFFSET/FETCH pagination
- `model.go` — `taskDB` (unexported), **toDBTask()**, **toBusTask()**, **toBusTasks()** — sqlx-tagged struct; enums serialized to strings in `toDBTask`, parsed back via `MustParse` in `toBusTask`
- `filter.go` — **applyFilter()** — appends `AND` clauses to query buffer for each non-nil filter field; uses named params in `data` map
- `order.go` — `orderByFields` map (business constant → SQL column name); **orderByClause()** — returns `"column direction"` or error on unknown field

---

## Impact Callouts

### ⚠ taskbus.Task (`business/domain/taskbus/model.go`)

Adding, removing, or renaming a field affects:

- `taskbus/taskbus.go` — `Create()` builds a `Task` literal from `NewTask` (must include new field); `Update()` merges `UpdateTask` onto `Task` (must handle new field)
- `taskdb/model.go` — `toDBTask()` maps every `Task` field to a `taskDB` field; `toBusTask()` maps back — both converters must be kept in sync
- `taskdb/taskdb.go` — SQL INSERT column list and `:named` params in `Create()`; UPDATE SET clause in `Update()`; SELECT column list in `Query()` and `QueryByID()` — all must include the new column
- `taskapp/model.go` — `toAppTask()` maps `taskbus.Task` → `app.Task`; add field to `app.Task` struct and converter

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

### ⚠ Enum values (`business/types/taskstatus`, `taskpriority`, `taskenergy`)

Adding a new value affects:

- `business/sdk/migrate/sql/migrate.sql` — `CHECK` constraint on the `tasks` table must include the new value (requires ALTER TABLE or a new migration version)
- Converters `toBusTask()` and `toBusUpdateTask()` — `MustParse`/`Parse` will panic or error on unknown values until the enum is updated

### ⚠ Pending columns: `expected_update_days`, `last_thread_at`, `debrief_status`

These columns are planned but not yet in the schema or code. When added, the full cascade is:

1. `business/sdk/migrate/sql/migrate.sql` — add ALTER TABLE (new migration version)
2. `taskbus/model.go` — add fields to `Task`; add to `UpdateTask` (and `NewTask` if settable at creation)
3. `taskdb/model.go` — add `db`-tagged fields to `taskDB`; update `toDBTask()` and `toBusTask()`
4. `taskdb/taskdb.go` — add column to INSERT list in `Create()`; add to UPDATE SET in `Update()`; add to SELECT list in `Query()` and `QueryByID()`
5. `taskapp/model.go` — add fields to `app.Task` response DTO and `app.UpdateTask` request DTO; update `toAppTask()` and `toBusUpdateTask()`
6. `business/types/debriefstatus/` — implement enum type (directory already exists as untracked)

---

## Routes

| Method | Path | Handler | Auth |
|--------|------|---------|------|
| GET | `/api/v1/tasks` | `queryAll` | X-API-Key |
| GET | `/api/v1/tasks/{task_id}` | `queryByID` | X-API-Key |
| POST | `/api/v1/tasks` | `create` | X-API-Key |
| PUT | `/api/v1/tasks/{task_id}` | `update` | X-API-Key |
| DELETE | `/api/v1/tasks/{task_id}` | `delete` | X-API-Key |

Query params for `GET /api/v1/tasks`: `page`, `rows`, `orderBy` (id/title/status/priority/due_date/created_at), `status`, `priority`, `context_id`, `start_due_date`, `end_due_date`.

`create` defaults: `status=todo`, `priority=medium`, `energy=medium`. `title` is required.

---

## Cross-Domain Dependencies

- **contexts** — `tasks.context_id` FK references `contexts.context_id` ON DELETE SET NULL. `QueryFilter.ContextID` and the `context_id` query param filter tasks by context.
- **tags** — `task_tags` join table (migration 1.04) links tasks to tags. Tag assignment is a separate domain (`tagbus`/`tagdb`); `taskapp` has no awareness of tags.
- **threadbus** (planned) — `last_thread_at` column will track the most recent thread entry touching a task; no code dependency exists yet.
- **debriefstatus type** — `business/types/debriefstatus/` directory is present (untracked) and will be needed when `debrief_status` is wired into the task model.
- **page SDK** (`business/sdk/page`) — `queryAll` uses `page.Parse` and `page.Page` for OFFSET/FETCH pagination.
- **order SDK** (`business/sdk/order`) — `Query` uses `order.By{Field, Direction}`; field constants live in `taskbus/order.go`.
- **sqldb** (`foundation/sqldb`) — store uses `NamedExecContext`, `NamedQuerySlice`, `NamedQueryStruct`; returns `sqldb.ErrDBNotFound` (wraps `sql.ErrNoRows`) on missing rows — handlers must check this explicitly to return 404.
