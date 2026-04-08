# Task Backend System

> Core task management domain supporting CRUD, filtering, ordering, status/priority/energy enums, due dates, recurrence rules, scheduling, duration estimation, and dependency-based blocking logic. Manages task dependencies (upstream/downstream) with cycle prevention and automatic blocking/unblocking transitions. When a task completes, dependent tasks are re-evaluated and recurrence rules generate the next instance.

## Core Types

### App Layer DTOs

```go
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

### Business Layer Types

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

type Dependency struct {
	TaskID      uuid.UUID
	DependsOnID uuid.UUID
	CreatedAt   time.Time
}

type QueryFilter struct {
	ID           *uuid.UUID
	Status       *taskstatus.Status
	Priority     *taskpriority.Priority
	ContextID    *uuid.UUID
	StartDueDate *time.Time
	EndDueDate   *time.Time
}

const (
	OrderByID        = "task_id"
	OrderByTitle     = "title"
	OrderByStatus    = "status"
	OrderByPriority  = "priority"
	OrderByDueDate   = "due_date"
	OrderByCreatedAt = "created_at"
)

var DefaultOrderBy = order.NewBy(OrderByCreatedAt, order.DESC)
```

### Storer Interfaces

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
```

### Store Layer

```go
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

## File Map

### App Layer (app/domain/taskapp/)
- `taskapp.go` — **create/update/delete/queryAll/queryByID** handler methods
- `model.go` — Task, NewTask, UpdateTask DTOs + **toAppTask()**, **toBusNewTask()**, **toBusUpdateTask()** converters
- `route.go` — **Routes.Add()** registers 9 endpoints; instantiates Store + DependencyStore + Business
- `filter.go` — **parseFilter()** parses (status, priority, context_id, start_due_date, end_due_date) → QueryFilter
- `order.go` — orderByFields map; **parseOrder()** parses (id, title, status, priority, due_date, created_at)
- `dependency.go` — **addDependency/removeDependency/queryDependencies/queryDependents** handlers

### Business Layer (business/domain/taskbus/)
- `taskbus.go` — **Create/Update/Delete/Query/Count/QueryByID/DismissTasksByContext**; **CreateNextRecurrence()** on completion; **UnblockDependents()** on task done
- `model.go` — Task, NewTask, UpdateTask, Dependency domain types
- `dependency.go` — DependencyStorer interface; **AddDependency()** with cycle prevention + auto-block; **RemoveDependency()** + reevaluateBlocked(); **QueryDependencies/QueryDependents/UnblockDependents/reevaluateBlocked**
- `filter.go` — QueryFilter struct (ID, Status, Priority, ContextID, StartDueDate, EndDueDate)
- `order.go` — 6 OrderBy constants; DefaultOrderBy = created_at DESC

### Store Layer (business/domain/taskbus/stores/taskdb/)
- `taskdb.go` — Implements Storer: **Create/Update/Delete/Query/Count/QueryByID/DismissTasksByContext**
- `model.go` — taskDB struct with db tags; **toDBTask()**, **toBusTask()** converters
- `dependency.go` — DependencyStore struct; implements DependencyStorer (5 methods)
- `filter.go` — **applyFilter()** WHERE clauses from QueryFilter
- `order.go` — orderByFields map; **orderByClause()** translates constants → SQL columns

## Impact Callouts

### ⚠ Task struct (business/domain/taskbus/model.go)
Adding/removing fields requires:
- `taskapp/model.go` — toAppTask() converter (JSON tags)
- `taskdb/model.go` — taskDB struct + toDBTask()/toBusTask() converters (db tags)
- `taskdb/taskdb.go` — INSERT/UPDATE/SELECT SQL column lists
- Migration SQL required

### ⚠ Status/Priority/Energy Enums (business/types/)
Changing enum values affects:
- `taskapp/model.go` — toBusNewTask()/toBusUpdateTask() use Parse()/MustParse()
- `taskdb/model.go` — toBusTask() uses MustParse(); toDBTask() uses .String()
- Database CHECK constraints in tasks table

### ⚠ QueryFilter (business/domain/taskbus/filter.go)
Adding filter fields requires:
- `taskapp/filter.go` — parseFilter() new query param
- `taskdb/filter.go` — applyFilter() new WHERE clause
- Both must stay in sync

### ⚠ Recurrence (business/domain/taskbus/taskbus.go)
On task completion (Status=Done), CreateNextRecurrence() parses RecurrenceRule, computes NextOccurrence(), creates new task with same core fields (Status=Open, linked via RecurrenceParentID). Error logged but doesn't prevent status update.

### ⚠ Blocking Logic (business/domain/taskbus/dependency.go)
- **AddDependency**: auto-blocks downstream task if upstream is not Done; prevents cycles
- **RemoveDependency**: triggers reevaluateBlocked() to unblock if no other unmet dependencies
- **UnblockDependents**: called on task completion to cascade transitions

### ⚠ DependencyStorer interface (business/domain/taskbus/dependency.go)
Adding methods requires:
- `taskdb/dependency.go` — implementation
- Business methods in `dependency.go` that call the storer

## Routes

| Method | Path | Handler |
|--------|------|---------|
| GET | /api/v1/tasks | queryAll — filter, order, pagination |
| GET | /api/v1/tasks/{task_id} | queryByID |
| POST | /api/v1/tasks | create — title required; priority/energy default Medium |
| PUT | /api/v1/tasks/{task_id} | update — auto-sets CompletedAt if status→Done; triggers recurrence/unblocking |
| DELETE | /api/v1/tasks/{task_id} | delete |
| POST | /api/v1/tasks/{task_id}/dependencies/{depends_on_id} | addDependency — cycle prevention; auto-blocks downstream |
| DELETE | /api/v1/tasks/{task_id}/dependencies/{depends_on_id} | removeDependency — reevaluates blocking |
| GET | /api/v1/tasks/{task_id}/dependencies | queryDependencies — upstream tasks |
| GET | /api/v1/tasks/{task_id}/dependents | queryDependents — downstream tasks |

All routes require `X-API-Key` header (mid.Auth middleware).

## Cross-Domain Dependencies

- **contextbus** — Task.ContextID links to context; DismissTasksByContext() called on context close
- **taskstatus, taskpriority, taskenergy, debriefstatus** (business/types/) — typed enums; Parse()/String() in converters
- **recurrence** (business/types/) — CreateNextRecurrence() uses recurrence.Parse() and NextOccurrence()
- **sqldb** (business/sdk/sqldb) — NamedExecContext, NamedQuerySlice, NamedQueryStruct
