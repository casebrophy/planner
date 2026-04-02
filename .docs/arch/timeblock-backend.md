# TimeBlock Backend System

> Time-slotted task scheduling. A time block assigns a task to a specific start/end window, optionally confirmed. Blocks appear in the weekly calendar view alongside events. CRUD via REST API; no business logic beyond basic validation.

## Core Types

```go
// business/domain/timeblockbus/model.go
type TimeBlock struct {
    ID        uuid.UUID
    TaskID    uuid.UUID
    StartsAt  time.Time
    EndsAt    time.Time
    Confirmed bool
    CreatedAt time.Time
    UpdatedAt time.Time
}

type NewTimeBlock struct {
    TaskID   uuid.UUID
    StartsAt time.Time
    EndsAt   time.Time
}

type UpdateTimeBlock struct {
    StartsAt  *time.Time
    EndsAt    *time.Time
    Confirmed *bool
}
```

```go
// business/domain/timeblockbus/filter.go
type QueryFilter struct {
    TaskID    *uuid.UUID
    DateFrom  *time.Time
    DateTo    *time.Time
    Confirmed *bool
}
```

```go
// business/domain/timeblockbus/timeblockbus.go
type Storer interface {
    Create(ctx context.Context, block TimeBlock) error
    Update(ctx context.Context, block TimeBlock) error
    Delete(ctx context.Context, block TimeBlock) error
    Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]TimeBlock, error)
    Count(ctx context.Context, filter QueryFilter) (int, error)
    QueryByID(ctx context.Context, id uuid.UUID) (TimeBlock, error)
}
```

```go
// app/domain/timeblockapp/model.go
type TimeBlock struct {
    ID        string `json:"id"`
    TaskID    string `json:"taskId"`
    StartsAt  string `json:"startsAt"`
    EndsAt    string `json:"endsAt"`
    Confirmed bool   `json:"confirmed"`
    CreatedAt string `json:"createdAt"`
    UpdatedAt string `json:"updatedAt"`
}

type NewTimeBlock struct {
    TaskID   string `json:"taskId"`
    StartsAt string `json:"startsAt"`
    EndsAt   string `json:"endsAt"`
}

type UpdateTimeBlock struct {
    StartsAt  *string `json:"startsAt"`
    EndsAt    *string `json:"endsAt"`
    Confirmed *bool   `json:"confirmed"`
}
```

```go
// business/domain/timeblockbus/stores/timeblockdb/model.go
type timeBlockDB struct {
    ID        uuid.UUID `db:"block_id"`
    TaskID    uuid.UUID `db:"task_id"`
    StartsAt  time.Time `db:"starts_at"`
    EndsAt    time.Time `db:"ends_at"`
    Confirmed bool      `db:"confirmed"`
    CreatedAt time.Time `db:"created_at"`
    UpdatedAt time.Time `db:"updated_at"`
}
```

## File Map

### Models
- `business/domain/timeblockbus/model.go` — TimeBlock, NewTimeBlock, UpdateTimeBlock
- `business/domain/timeblockbus/filter.go` — QueryFilter (TaskID, DateFrom, DateTo, Confirmed)
- `business/domain/timeblockbus/order.go` — OrderByStartsAt, OrderByCreatedAt, DefaultOrderBy
- `business/domain/timeblockbus/stores/timeblockdb/model.go` — timeBlockDB + toDBTimeBlock/toBusTimeBlock converters
- `app/domain/timeblockapp/model.go` — app-layer DTOs + toAppTimeBlock/toBusNewTimeBlock/toBusUpdateTimeBlock converters

### Handlers
- `app/domain/timeblockapp/timeblockapp.go` — **create()** POST, **update()** PUT, **delete()** DELETE, **queryAll()** GET, **queryByID()** GET
- `app/domain/timeblockapp/route.go` — Routes.Add() wiring
- `app/domain/timeblockapp/filter.go` — parseFilter() from query params
- `app/domain/timeblockapp/order.go` — parseOrder() from query params

### Core
- `business/domain/timeblockbus/timeblockbus.go` — **Create()**, **Update()**, **Delete()**, **Query()**, **Count()**, **QueryByID()** — thin pass-through to Storer

### Store
- `business/domain/timeblockbus/stores/timeblockdb/timeblockdb.go` — **Create()** INSERT, **Update()** UPDATE, **Delete()** DELETE, **Query()** SELECT with filter+order+page, **Count()** SELECT COUNT, **QueryByID()** SELECT by PK
- `business/domain/timeblockbus/stores/timeblockdb/filter.go` — applyFilter() builds WHERE clauses
- `business/domain/timeblockbus/stores/timeblockdb/order.go` — orderByClause() maps field constants to SQL columns

## Impact Callouts

### ⚠ TimeBlock (business/domain/timeblockbus/model.go)
Changing this struct shape affects:
- `business/domain/timeblockbus/timeblockbus.go` — Create() constructs it, Update() mutates fields
- `business/domain/timeblockbus/stores/timeblockdb/model.go` — toDBTimeBlock/toBusTimeBlock field mapping
- `business/domain/timeblockbus/stores/timeblockdb/timeblockdb.go` — SQL INSERT/UPDATE/SELECT column lists
- `app/domain/timeblockapp/model.go` — toAppTimeBlock field mapping
- `app/domain/scheduleapp/scheduleapp.go` — reads TaskID, StartsAt, EndsAt, Confirmed to build ScheduleItem
- Migration required if DB column added/removed

### ⚠ Storer interface (business/domain/timeblockbus/timeblockbus.go)
Adding/changing a method affects:
- `business/domain/timeblockbus/stores/timeblockdb/timeblockdb.go` — must implement the method
- `business/domain/timeblockbus/timeblockbus.go` — calls the method
- `app/domain/timeblockapp/timeblockapp.go` — if new endpoint, must wire handler

### ⚠ QueryFilter (business/domain/timeblockbus/filter.go)
Adding a filter field affects:
- `business/domain/timeblockbus/stores/timeblockdb/filter.go` — applyFilter() must add WHERE clause
- `app/domain/timeblockapp/filter.go` — parseFilter() must parse query param
- `app/domain/scheduleapp/scheduleapp.go` — uses DateFrom/DateTo filters directly

## Routes

| Method | Path | Handler | Auth |
|--------|------|---------|------|
| GET | /api/v1/time-blocks | queryAll | API key |
| GET | /api/v1/time-blocks/{block_id} | queryByID | API key |
| POST | /api/v1/time-blocks | create | API key |
| PUT | /api/v1/time-blocks/{block_id} | update | API key |
| DELETE | /api/v1/time-blocks/{block_id} | delete | API key |

## Cross-Domain Dependencies

- **tasks** — `time_blocks.task_id` FK to `tasks.task_id` (CASCADE delete)
- **scheduleapp** — queries time blocks alongside events for merged schedule view
- **mcpapp** — potential future consumer for `create_time_block` MCP tool (not yet wired)
