# Task Backend System

> The Task domain manages the complete lifecycle of user tasks including CRUD operations, dependency tracking, recurrence scheduling, and automatic status management. Tasks can be organized within contexts, prioritized, scheduled, and linked as dependencies—with automatic blocking/unblocking logic based on upstream completion. When a task completes, dependent tasks are automatically re-evaluated and any configured recurrence rules generate the next instance.

## Core Types

### Task
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
```

### NewTask
```go
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
```

### UpdateTask
```go
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

### Dependency
```go
type Dependency struct {
	TaskID      uuid.UUID
	DependsOnID uuid.UUID
	CreatedAt   time.Time
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
	DismissTasksByContext(ctx context.Context, contextID uuid.UUID) (int, error)
}
```

### DependencyStorer Interface
```go
type DependencyStorer interface {
	AddDependency(ctx context.Context, dep Dependency) error
	RemoveDependency(ctx context.Context, taskID, dependsOnID uuid.UUID) error
	QueryDependencies(ctx context.Context, taskID uuid.UUID) ([]Task, error)
	QueryDependents(ctx context.Context, taskID uuid.UUID) ([]Task, error)
	HasUnmetDependencies(ctx context.Context, taskID uuid.UUID) (bool, error)
}
```

## File Map

### Business Layer
- `business/domain/taskbus/taskbus.go` — **NewBusiness()** — creates a Business instance with Storer and DependencyStorer
- `business/domain/taskbus/taskbus.go` — **Create()** — creates a new task with given NewTask input, assigns UUID and timestamps
- `business/domain/taskbus/taskbus.go` — **Update()** — patches a task, sets CompletedAt if status changes to Done, triggers UnblockDependents and CreateNextRecurrence on completion
- `business/domain/taskbus/taskbus.go` — **Delete()** — removes a task from storage
- `business/domain/taskbus/taskbus.go` — **Query()** — retrieves paginated filtered/ordered task list
- `business/domain/taskbus/taskbus.go` — **Count()** — returns count of tasks matching filter
- `business/domain/taskbus/taskbus.go` — **QueryByID()** — fetches single task by ID
- `business/domain/taskbus/taskbus.go` — **DismissTasksByContext()** — marks all open/blocked tasks in a context as dismissed
- `business/domain/taskbus/taskbus.go` — **CreateNextRecurrence()** — generates next recurring task instance from completed task
- `business/domain/taskbus/dependency.go` — **AddDependency()** — creates dependency edge, validates no cycles, auto-blocks downstream task if upstream not done
- `business/domain/taskbus/dependency.go` — **RemoveDependency()** — removes dependency and re-evaluates blocked status
- `business/domain/taskbus/dependency.go` — **QueryDependencies()** — retrieves upstream tasks (what this task depends on)
- `business/domain/taskbus/dependency.go` — **QueryDependents()** — retrieves downstream tasks (what depends on this task)
- `business/domain/taskbus/dependency.go` — **UnblockDependents()** — recursively unblocks dependent tasks after upstream completes

### Store Layer
- `business/domain/taskbus/stores/taskdb/taskdb.go` — **NewStore()** — creates Store instance
- `business/domain/taskbus/stores/taskdb/taskdb.go` — **Create()** — INSERT task into tasks table
- `business/domain/taskbus/stores/taskdb/taskdb.go` — **Update()** — UPDATE task record by task_id
- `business/domain/taskbus/stores/taskdb/taskdb.go` — **Delete()** — DELETE task by ID
- `business/domain/taskbus/stores/taskdb/taskdb.go` — **Query()** — SELECT with filter, orderBy, and pagination
- `business/domain/taskbus/stores/taskdb/taskdb.go` — **Count()** — SELECT COUNT with filter
- `business/domain/taskbus/stores/taskdb/taskdb.go` — **QueryByID()** — SELECT single task by task_id
- `business/domain/taskbus/stores/taskdb/taskdb.go` — **DismissTasksByContext()** — UPDATE status to dismissed for open/blocked tasks in context
- `business/domain/taskbus/stores/taskdb/dependency.go` — **NewDependencyStore()** — creates DependencyStore instance
- `business/domain/taskbus/stores/taskdb/dependency.go` — **AddDependency()** — INSERT into task_dependencies table
- `business/domain/taskbus/stores/taskdb/dependency.go` — **RemoveDependency()** — DELETE from task_dependencies
- `business/domain/taskbus/stores/taskdb/dependency.go` — **QueryDependencies()** — SELECT tasks where td.task_id = :id (upstream)
- `business/domain/taskbus/stores/taskdb/dependency.go` — **QueryDependents()** — SELECT tasks where td.depends_on_id = :id (downstream)
- `business/domain/taskbus/stores/taskdb/dependency.go` — **HasUnmetDependencies()** — COUNT non-done upstream dependencies

### App Layer (Handlers)
- `app/domain/taskapp/taskapp.go` — **create()** — POST /api/v1/tasks, validates NewTask, calls Business.Create()
- `app/domain/taskapp/taskapp.go` — **update()** — PUT /api/v1/tasks/{task_id}, loads existing task, applies UpdateTask patch
- `app/domain/taskapp/taskapp.go` — **delete()** — DELETE /api/v1/tasks/{task_id}, verifies task exists
- `app/domain/taskapp/taskapp.go` — **queryAll()** — GET /api/v1/tasks, parses filter/orderBy/pagination
- `app/domain/taskapp/taskapp.go` — **queryByID()** — GET /api/v1/tasks/{task_id}
- `app/domain/taskapp/dependency.go` — **addDependency()** — POST /api/v1/tasks/{task_id}/dependencies/{depends_on_id}
- `app/domain/taskapp/dependency.go` — **removeDependency()** — DELETE /api/v1/tasks/{task_id}/dependencies/{depends_on_id}
- `app/domain/taskapp/dependency.go` — **queryDependencies()** — GET /api/v1/tasks/{task_id}/dependencies
- `app/domain/taskapp/dependency.go` — **queryDependents()** — GET /api/v1/tasks/{task_id}/dependents

## Impact Callouts

### ⚠ Task (`business/domain/taskbus/model.go`)
Changing this struct shape affects:
- `app/domain/taskapp/model.go` — must update toAppTask() and toBusUpdateTask() conversion functions
- `business/domain/taskbus/stores/taskdb/model.go` — must update dbTask struct and toDBTask()/toBusTask() conversions
- `business/domain/taskbus/stores/taskdb/taskdb.go` — SQL INSERT/UPDATE columns and Scan() field list
- Migration required if DB column added/removed

### ⚠ Storer Interface (`business/domain/taskbus/taskbus.go`)
Adding/changing a method affects:
- `business/domain/taskbus/stores/taskdb/taskdb.go` — must implement the new method on Store
- `app/domain/taskapp/route.go` — if new endpoint, must wire route and handler

### ⚠ DependencyStorer Interface (`business/domain/taskbus/dependency.go`)
Adding/changing a method affects:
- `business/domain/taskbus/stores/taskdb/dependency.go` — must implement the new method on DependencyStore
- `business/domain/taskbus/taskbus.go` — must call the new method if needed for business logic

### ⚠ Update() Business Logic (`business/domain/taskbus/taskbus.go`)
When task completes (status = Done):
- Automatically calls UnblockDependents() — unblocks all downstream tasks
- Automatically calls CreateNextRecurrence() if RecurrenceRule is set — generates next recurring instance
- Changing completion logic affects cascading task automation

### ⚠ DismissTasksByContext() (`business/domain/taskbus/taskbus.go` + `stores/taskdb/taskdb.go`)
Called cross-domain from contextapp when a project context is closed:
- `app/domain/contextapp/contextapp.go` — calls this on project close (not area close)
- `business/domain/taskbus/stores/taskdb/taskdb.go` — raw SQL UPDATE, filters by context_id and open/blocked statuses

## Routes

| Method | Path | Handler | Auth |
|--------|------|---------|------|
| GET | /api/v1/tasks | queryAll | Required |
| GET | /api/v1/tasks/{task_id} | queryByID | Required |
| POST | /api/v1/tasks | create | Required |
| PUT | /api/v1/tasks/{task_id} | update | Required |
| DELETE | /api/v1/tasks/{task_id} | delete | Required |
| POST | /api/v1/tasks/{task_id}/dependencies/{depends_on_id} | addDependency | Required |
| DELETE | /api/v1/tasks/{task_id}/dependencies/{depends_on_id} | removeDependency | Required |
| GET | /api/v1/tasks/{task_id}/dependencies | queryDependencies | Required |
| GET | /api/v1/tasks/{task_id}/dependents | queryDependents | Required |

## Cross-Domain Dependencies

- **contextbus** — tasks optionally scoped to a context via ContextID; DismissTasksByContext() called by contextapp on project close
- **taskstatus, taskpriority, taskenergy** — enum type packages for task state fields
- **debriefstatus** — enum type for DebriefStatus field
- **recurrence** — parses RecurrenceRule strings and computes next occurrence dates
- **order, page** — SDK packages for query ordering and pagination
- **sqldb** — database abstraction layer for SQL execution
