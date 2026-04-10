# Event Backend System

> Events are calendar items with start/end times, optional locations, and optional context associations. The event domain supports CRUD operations, filtering by context and date range, and asynchronous context classification via Claude Code extraction when events are created without explicit context assignment.

## Core Types

### Business Layer

```go
type Event struct {
	ID          uuid.UUID
	ContextID   *uuid.UUID
	Title       string
	Description string
	Location    *string
	StartsAt    time.Time
	EndsAt      time.Time
	AllDay      bool
	RawInputID  *uuid.UUID
	Unconfirmed bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type NewEvent struct {
	ContextID   *uuid.UUID
	Title       string
	Description string
	Location    *string
	StartsAt    time.Time
	EndsAt      time.Time
	AllDay      bool
	RawInputID  *uuid.UUID
	Unconfirmed bool
}

type UpdateEvent struct {
	ContextID   *uuid.UUID
	Title       *string
	Description *string
	Location    *string
	StartsAt    *time.Time
	EndsAt      *time.Time
	AllDay      *bool
	Unconfirmed *bool
}

type QueryFilter struct {
	ContextID *uuid.UUID
	DateFrom  *time.Time
	DateTo    *time.Time
}

const (
	OrderByStartsAt  = "starts_at"
	OrderByCreatedAt = "created_at"
)

var DefaultOrderBy = order.NewBy(OrderByStartsAt, order.ASC)

type Storer interface {
	Create(ctx context.Context, event Event) error
	Update(ctx context.Context, event Event) error
	Delete(ctx context.Context, event Event) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Event, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (Event, error)
}
```

### App Layer DTOs

```go
type Event struct {
	ID          string  `json:"id"`
	ContextID   *string `json:"contextId,omitempty"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Location    *string `json:"location,omitempty"`
	StartsAt    string  `json:"startsAt"`
	EndsAt      string  `json:"endsAt"`
	AllDay      bool    `json:"allDay"`
	RawInputID  *string `json:"rawInputId,omitempty"`
	Unconfirmed bool    `json:"unconfirmed"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

type NewEvent struct {
	ContextID   *string `json:"contextId"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Location    *string `json:"location"`
	StartsAt    string  `json:"startsAt"`
	EndsAt      string  `json:"endsAt"`
	AllDay      bool    `json:"allDay"`
}

type UpdateEvent struct {
	ContextID   *string `json:"contextId"`
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Location    *string `json:"location"`
	StartsAt    *string `json:"startsAt"`
	EndsAt      *string `json:"endsAt"`
	AllDay      *bool   `json:"allDay"`
}
```

### Store Layer

```go
type eventDB struct {
	ID          uuid.UUID  `db:"event_id"`
	ContextID   *uuid.UUID `db:"context_id"`
	Title       string     `db:"title"`
	Description string     `db:"description"`
	Location    *string    `db:"location"`
	StartsAt    time.Time  `db:"starts_at"`
	EndsAt      time.Time  `db:"ends_at"`
	AllDay      bool       `db:"all_day"`
	RawInputID  *uuid.UUID `db:"raw_input_id"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
	Unconfirmed bool       `db:"unconfirmed"`
}
```

## File Map

### App Layer (app/domain/eventapp/)
- `eventapp.go` — **create()** POST handler; triggers async classification via extractor if contextId omitted; fires async **EmbedAndStore()** to embeddingBus for vector storage; **update()** PATCH; **delete()** DELETE; **queryAll()** GET with filter/order/page; **queryByID()** GET single
- `model.go` — App DTOs + **toAppEvent()**, **toAppEvents()**, **toBusNewEvent()**, **toBusUpdateEvent()** converters (string ↔ time parsing)
- `route.go` — **Routes.Add()** registers 5 endpoints; wires extractor for async classification; wires embeddingBus from cfg.EmbeddingBus
- `filter.go` — **parseFilter()** maps (context_id, date_from, date_to) → QueryFilter
- `order.go` — **parseOrder()** maps (starts_at, created_at) → business constants; defaults to OrderByStartsAt

### Business Layer (business/domain/eventbus/)
- `eventbus.go` — **Create()** uuid.New() + timestamps; **Update()** applies patches; **Delete/Query/Count/QueryByID** delegate to storer
- `model.go` — Event, NewEvent, UpdateEvent domain types
- `filter.go` — QueryFilter struct (ContextID, DateFrom, DateTo)
- `order.go` — OrderByStartsAt, OrderByCreatedAt; DefaultOrderBy = starts_at ASC

### Store Layer (business/domain/eventbus/stores/eventdb/)
- `eventdb.go` — **Create/Update/Delete** via NamedExecContext; **Query** uses applyFilter + orderByClause; **Count** aggregates; **QueryByID** fetches single row
- `model.go` — eventDB struct + **toDBEvent()**, **toBusEvent()**, **toBusEvents()** converters
- `filter.go` — **applyFilter()** WHERE clauses: context_id =, starts_at >= :date_from, ends_at <= :date_to
- `order.go` — orderByFields map; **orderByClause()** returns SQL fragment

## Impact Callouts

### ⚠ Event struct (eventbus.Event, eventdb.eventDB)
Changing shape affects all layers:
- `eventapp/model.go` — app DTO + toAppEvent() converter
- `eventdb/model.go` — eventDB struct + toDBEvent/toBusEvent converters
- `eventdb/eventdb.go` — CREATE/UPDATE/SELECT column lists
- Migration SQL required

### ⚠ QueryFilter (eventbus/filter.go)
Adding filter fields requires:
- `eventapp/filter.go` — new query param extraction
- `eventdb/filter.go` — new WHERE clause logic

### ⚠ Order Fields (eventbus/order.go)
Adding order fields requires:
- `eventbus/order.go` — new constant
- `eventapp/order.go` — new entry in orderByFields map
- `eventdb/order.go` — new entry in orderByFields map (const → SQL column)

### ⚠ Async Context Classification (eventapp/eventapp.go)
create() triggers async context assignment for uncontexted events:
- **contextbus** — fetches active contexts for extractor reference list; validates suggested context
- **clarificationbus** — creates clarification when confidence < 0.7
- **ingestbus.extractor** — Claude Code or Ollama text extraction
- Auto-assigns context when confidence >= 0.7; creates clarification when < 0.7

### ⚠ Async Vector Embedding (eventapp/eventapp.go)
create() fires async goroutine calling embeddingBus.EmbedAndStore() for RAG:
- **embeddingbus** — vector generation + pgvector storage for event title + description
- Uses entity type "event" and event ID for reference
- Best-effort (errors logged but not returned)

## Routes

| Method | Path | Handler |
|--------|------|---------|
| GET | /api/v1/events | queryAll — context_id, date_from, date_to filters; orderBy; pagination |
| GET | /api/v1/events/{event_id} | queryByID — returns 404 if not found |
| POST | /api/v1/events | create — title, startsAt, endsAt required; triggers async classification if contextId omitted |
| PUT | /api/v1/events/{event_id} | update — all fields optional; updates updatedAt |
| DELETE | /api/v1/events/{event_id} | delete — returns 204 |

All routes require `X-API-Key` header (auth middleware).

## Cross-Domain Dependencies

- **contextbus** — async classification queries active contexts; validates suggested context ID
- **clarificationbus** — creates clarification items for low-confidence (< 0.7) context suggestions
- **ingestbus.extractor** — Claude Code or Ollama-based text extraction for async context inference
- **embeddingbus** — generates vectors and stores in pgvector for event content (title + description) on create
