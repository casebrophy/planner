# TimeBlock Backend System

> Time blocks represent scheduled time slots allocated to tasks. Each block has a start/end time, confirmed status, and belongs to exactly one task. Blocks support filtering by task ID, date range, and confirmation status, plus ordering by start time or creation date.

## Core Types

### App Layer

```go
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

### Business Layer

```go
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

type QueryFilter struct {
	TaskID    *uuid.UUID
	DateFrom  *time.Time
	DateTo    *time.Time
	Confirmed *bool
}

const (
	OrderByStartsAt  = "starts_at"
	OrderByCreatedAt = "created_at"
)

var DefaultOrderBy = order.NewBy(OrderByStartsAt, order.ASC)

type Storer interface {
	Create(ctx context.Context, block TimeBlock) error
	Update(ctx context.Context, block TimeBlock) error
	Delete(ctx context.Context, block TimeBlock) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]TimeBlock, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (TimeBlock, error)
}
```

### Store Layer

```go
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

### App Layer (app/domain/timeblockapp/)
- `timeblockapp.go` — **create/update/delete/queryAll/queryByID** handlers
- `model.go` — TimeBlock, NewTimeBlock, UpdateTimeBlock DTOs + **toAppTimeBlock()**, **toBusNewTimeBlock()**, **toBusUpdateTimeBlock()** converters
- `route.go` — **Routes.Add()** wires Store → Business → Handlers with auth middleware
- `filter.go` — **parseFilter()** maps (task_id, date_from, date_to, confirmed) → QueryFilter
- `order.go` — **parseOrder()** maps (starts_at, created_at) → order constants via orderByFields map

### Business Layer (business/domain/timeblockbus/)
- `timeblockbus.go` — **Create/Update/Delete/Query/Count/QueryByID** + Storer interface
- `model.go` — TimeBlock, NewTimeBlock, UpdateTimeBlock domain types
- `filter.go` — QueryFilter struct (TaskID, DateFrom, DateTo, Confirmed)
- `order.go` — OrderByStartsAt, OrderByCreatedAt; DefaultOrderBy = starts_at ASC

### Store Layer (business/domain/timeblockbus/stores/timeblockdb/)
- `timeblockdb.go` — SQL implementation using sqldb helpers
- `model.go` — timeBlockDB struct + **toDBTimeBlock()**, **toBusTimeBlock()**, **toBusTimeBlocks()** converters
- `filter.go` — **applyFilter()** WHERE clauses: task_id =, starts_at >=, ends_at <=, confirmed =
- `order.go` — orderByFields map; **orderByClause()** → SQL column names

## Impact Callouts

### ⚠ TimeBlock struct (business/domain/timeblockbus/model.go)
Adding/removing fields requires:
- `timeblockapp/model.go` — app DTO + toAppTimeBlock() converter
- `timeblockdb/model.go` — timeBlockDB struct + toDBTimeBlock/toBusTimeBlock converters
- `timeblockdb/timeblockdb.go` — INSERT/UPDATE/SELECT SQL
- Migration SQL for schema changes

### ⚠ QueryFilter (business/domain/timeblockbus/filter.go)
Adding filter fields requires:
- `timeblockapp/filter.go` — parseFilter() new query param
- `timeblockdb/filter.go` — applyFilter() new WHERE clause

### ⚠ Order Constants (business/domain/timeblockbus/order.go)
Adding order fields requires:
- `timeblockdb/order.go` — add to orderByFields map (const → SQL column)
- `timeblockapp/order.go` — add to orderByFields map (request field → const)

## Routes

| Method | Path | Handler |
|--------|------|---------|
| GET | /api/v1/time-blocks | queryAll — filter by task_id, date_from, date_to, confirmed; pagination |
| GET | /api/v1/time-blocks/{block_id} | queryByID |
| POST | /api/v1/time-blocks | create — taskId, startsAt, endsAt required |
| PUT | /api/v1/time-blocks/{block_id} | update — startsAt, endsAt, confirmed all optional |
| DELETE | /api/v1/time-blocks/{block_id} | delete |

All routes require `X-API-Key` header (auth middleware).

## Cross-Domain Dependencies

- **Task domain** — TimeBlock.TaskID references Task.ID; no reverse dependency (Task does not own TimeBlock collection). Deleting a task does NOT cascade delete its time blocks — caller is responsible.
- **business/sdk/page** — pagination via page.Page
- **business/sdk/order** — ordering via order.By
- **business/sdk/sqldb** — NamedExecContext, NamedQuerySlice, NamedQueryStruct helpers
