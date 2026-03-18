# Task Backend System

The Task domain provides full CRUD operations for tasks with status, priority, and energy tracking. Tasks are organized hierarchically under contexts, with comprehensive filtering and sorting capabilities. The architecture follows a layered pattern: HTTP handlers → business logic core → database store, with thin translation layers between each tier.

## Core Types

### Task (Business Layer)
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
```

### NewTask (Business Layer)
```go
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
```

### UpdateTask (Business Layer)
```go
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

### Task (App Layer)
```go
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
```

### NewTask (App Layer)
```go
type NewTask struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	ContextID   *string `json:"contextId"`
	Priority    string  `json:"priority"`
	Energy      string  `json:"energy"`
	DurationMin *int    `json:"durationMin"`
	DueDate     *string `json:"dueDate"`
}
```

### UpdateTask (App Layer)
```go
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

### taskDB (Store Layer)
```go
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

### QueryFilter
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

### Storer Interface
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

### Status Enum
```go
type Status struct {
	value string
}

var (
	Todo       = Status{"todo"}
	InProgress = Status{"in_progress"}
	Done       = Status{"done"}
	Cancelled  = Status{"cancelled"}
)

// Functions:
// Parse(s string) (Status, error)
// MustParse(s string) Status
// (s Status) String() string
// (s Status) MarshalText() ([]byte, error)
// (s *Status) UnmarshalText(data []byte) error
// (s Status) EqualString(v string) bool
```

### Priority Enum
```go
type Priority struct {
	value string
}

var (
	Low    = Priority{"low"}
	Medium = Priority{"medium"}
	High   = Priority{"high"}
	Urgent = Priority{"urgent"}
)

// Functions:
// Parse(s string) (Priority, error)
// MustParse(s string) Priority
// (p Priority) String() string
// (p Priority) MarshalText() ([]byte, error)
// (p *Priority) UnmarshalText(data []byte) error
// (p Priority) EqualString(v string) bool
```

### Energy Enum
```go
type Energy struct {
	value string
}

var (
	Low    = Energy{"low"}
	Medium = Energy{"medium"}
	High   = Energy{"high"}
)

// Functions:
// Parse(s string) (Energy, error)
// MustParse(s string) Energy
// (e Energy) String() string
// (e Energy) MarshalText() ([]byte, error)
// (e *Energy) UnmarshalText(data []byte) error
// (e Energy) EqualString(v string) bool
```

## File Map

### Type Definitions
- **`business/types/taskstatus/taskstatus.go`** — Status enum (todo, in_progress, done, cancelled) with Parse/MustParse and text marshaling
- **`business/types/taskpriority/taskpriority.go`** — Priority enum (low, medium, high, urgent) with Parse/MustParse and text marshaling
- **`business/types/taskenergy/taskenergy.go`** — Energy enum (low, medium, high) with Parse/MustParse and text marshaling

### App Layer (HTTP Handlers)
- **`app/domain/taskapp/model.go`** — HTTP DTOs: Task, NewTask, UpdateTask with conversion functions (toAppTask, toBusNewTask, toBusUpdateTask)
- **`app/domain/taskapp/taskapp.go`** — Handler methods:
  - **create()** — POST /api/v1/tasks, validates title required, converts to business layer, creates task
  - **update()** — PUT /api/v1/tasks/{task_id}, fetches task by ID, applies updates, returns updated task
  - **delete()** — DELETE /api/v1/tasks/{task_id}, fetches task by ID, deletes via storer
  - **queryAll()** — GET /api/v1/tasks, supports pagination, filtering (status, priority, context_id, due_date range), sorting
  - **queryByID()** — GET /api/v1/tasks/{task_id}, fetches single task by UUID
- **`app/domain/taskapp/route.go`** — **Routes.Add()** — registers all five endpoints with Auth middleware, instantiates taskdb.Store and taskbus.Business
- **`app/domain/taskapp/filter.go`** — **parseFilter()** — parses query parameters (status, priority, context_id, start_due_date, end_due_date) into QueryFilter
- **`app/domain/taskapp/order.go`** — **parseOrder()** — maps request orderBy field names (id, title, status, priority, due_date, created_at) to taskbus constants

### Business Layer (Core Logic)
- **`business/domain/taskbus/model.go`** — Business models: Task, NewTask, UpdateTask
- **`business/domain/taskbus/taskbus.go`** — Business struct and methods:
  - **Create()** — generates UUID, sets CreatedAt/UpdatedAt, calls storer.Create
  - **Update()** — applies partial updates (including CompletedAt when status→Done), updates UpdatedAt, calls storer.Update
  - **Delete()** — calls storer.Delete
  - **Query()** — delegates to storer with filter/order/pagination
  - **Count()** — delegates to storer to count filtered tasks
  - **QueryByID()** — delegates to storer to fetch by UUID
- **`business/domain/taskbus/filter.go`** — QueryFilter struct for filtering by ID, Status, Priority, ContextID, due date range
- **`business/domain/taskbus/order.go`** — Order field constants (OrderByID, OrderByTitle, OrderByStatus, OrderByPriority, OrderByDueDate, OrderByCreatedAt) and DefaultOrderBy

### Store Layer (Database)
- **`business/domain/taskbus/stores/taskdb/model.go`** — taskDB internal struct (all db tags), conversion functions:
  - **toDBTask()** — Business Task → taskDB (converts enums to strings)
  - **toBusTask()** — taskDB → Business Task (parses string enums)
  - **toBusTasks()** — slice converter
- **`business/domain/taskbus/stores/taskdb/taskdb.go`** — Store struct and methods:
  - **NewStore()** — constructor taking logger and sqlx.DB
  - **Create()** — INSERT with all fields via named query
  - **Update()** — UPDATE all writable fields WHERE task_id via named query
  - **Delete()** — DELETE WHERE task_id via named query
  - **Query()** — SELECT with WHERE 1=1 base, applies filter, ORDER BY, OFFSET/FETCH pagination
  - **Count()** — SELECT COUNT(*) with filter applied
  - **QueryByID()** — SELECT WHERE task_id by UUID
- **`business/domain/taskbus/stores/taskdb/filter.go`** — **applyFilter()** — builds WHERE clauses for ID, Status, Priority, ContextID, StartDueDate, EndDueDate
- **`business/domain/taskbus/stores/taskdb/order.go`** — orderByFields mapping (business constants → SQL column names), **orderByClause()** — validates and formats ORDER BY clause

## Database Schema

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

CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_context ON tasks(context_id);
CREATE INDEX idx_tasks_due ON tasks(due_date) WHERE due_date IS NOT NULL;
CREATE INDEX idx_tasks_priority ON tasks(priority);
```

## Impact Callouts

### ⚠ Task struct (business/domain/taskbus/model.go)
Changing the Task struct shape affects:
- `business/domain/taskbus/stores/taskdb/model.go` — toDBTask() and toBusTask() converters must be updated
- `app/domain/taskapp/model.go` — App Task struct and toAppTask() converter must be updated
- `business/domain/taskbus/stores/taskdb/taskdb.go` — SQL SELECT/INSERT/UPDATE statements reference all fields
- Database schema (migrate.sql) — must add/remove columns and update indexes

### ⚠ Storer interface (business/domain/taskbus/taskbus.go)
Adding or changing a method affects:
- `business/domain/taskbus/stores/taskdb/taskdb.go` — Store must implement the new method signature
- `business/domain/taskbus/taskbus.go` — Business methods delegate to storer, must call new methods
- All handler methods in `app/domain/taskapp/taskapp.go` that use the storer indirectly

### ⚠ QueryFilter struct (business/domain/taskbus/filter.go)
Adding a filter field affects:
- `business/domain/taskbus/stores/taskdb/filter.go` — applyFilter() must handle new field with WHERE clause
- `app/domain/taskapp/filter.go` — parseFilter() must parse new query parameter and populate filter field
- Handler queryAll() in `app/domain/taskapp/taskapp.go` — no change needed (passes filter through)

### ⚠ Order constants (business/domain/taskbus/order.go)
Adding a new OrderBy constant affects:
- `business/domain/taskbus/stores/taskdb/order.go` — orderByFields map must include new field → SQL column name mapping
- `app/domain/taskapp/order.go` — orderByFields map must include new request field name → business constant mapping
- Handler queryAll() — no change needed (passes orderBy through)

### ⚠ Status/Priority/Energy enums (business/types/)
Changing enum values affects:
- Database CHECK constraints in migrate.sql must be updated
- All code parsing strings to enums (toBusTask, toBusUpdateTask) — validates against enum set
- All validation in handlers and conversion functions

## Routes

| Method | Path | Handler | Notes |
|--------|------|---------|-------|
| GET | /api/v1/tasks | queryAll | Supports query params: `page`, `rows`, `status`, `priority`, `context_id`, `start_due_date`, `end_due_date`, `orderBy` |
| GET | /api/v1/tasks/{task_id} | queryByID | Fetches single task by UUID |
| POST | /api/v1/tasks | create | Body: NewTask; validates title required; defaults Priority=medium, Energy=medium, Status=todo |
| PUT | /api/v1/tasks/{task_id} | update | Body: UpdateTask; all fields optional; sets CompletedAt if Status→Done |
| DELETE | /api/v1/tasks/{task_id} | delete | Removes task record |

All endpoints require Auth middleware (API key validation).

## Cross-Domain Dependencies

- **Context Domain** — Tasks have optional ContextID foreign key to contexts table; cascade NULL on context deletion
- **Page SDK** (`business/sdk/page`) — queryAll uses Page struct for pagination (Offset, RowsPerPage, Number)
- **Order SDK** (`business/sdk/order`) — Query uses order.By struct with Field (constant) and Direction (ASC/DESC)
- **sqldb utilities** (`foundation/sqldb`) — Store uses NamedExecContext, NamedQuerySlice, NamedQueryStruct helpers
- **Error handling** (`app/sdk/errs`) — Handlers return specific error codes (InvalidArgument, NotFound, Internal)
- **HTTP web framework** (`foundation/web`) — Handlers implement web.Encoder pattern for responses
