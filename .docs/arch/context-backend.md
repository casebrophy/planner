# Context Backend System

> Manages projects and areas as first-class contexts in the planner. Each context tracks its lifecycle (active/paused/closed), can store typed events with metadata, and triggers debrief workflows when closed. Coordinates with task and clarification domains for cross-cutting concerns like cascade dismissal and debrief card creation.

## Core Types

### Context
```go
type Context struct {
	ID            uuid.UUID
	Title         string
	Description   string
	Kind          contextkind.Kind
	Status        Status
	Summary       string
	LastEvent     *time.Time
	LastThreadAt  *time.Time
	DebriefStatus debriefstatus.Status
	Outcome       *contextoutcome.Outcome
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
```

### NewContext
```go
type NewContext struct {
	Title       string
	Description string
	Kind        contextkind.Kind
}
```

### UpdateContext
```go
type UpdateContext struct {
	Title         *string
	Description   *string
	Kind          *contextkind.Kind
	Status        *Status
	Summary       *string
	DebriefStatus *debriefstatus.Status
	Outcome       *contextoutcome.Outcome
}
```

### Event
```go
type Event struct {
	ID        uuid.UUID
	ContextID uuid.UUID
	Kind      string
	Content   string
	Metadata  *json.RawMessage
	SourceID  *uuid.UUID
	CreatedAt time.Time
}
```

### NewEvent
```go
type NewEvent struct {
	ContextID uuid.UUID
	Kind      string
	Content   string
	Metadata  *json.RawMessage
	SourceID  *uuid.UUID
}
```

### Status
```go
type Status int

const (
	Active Status = iota
	Paused
	Closed
)
```

### QueryFilter
```go
type QueryFilter struct {
	ID     *uuid.UUID
	Status *Status
	Kind   *contextkind.Kind
	Title  *string
}
```

### Storer Interface
```go
type Storer interface {
	Create(ctx context.Context, c Context) error
	Update(ctx context.Context, c Context) error
	Delete(ctx context.Context, c Context) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Context, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (Context, error)
	CreateEvent(ctx context.Context, e Event) error
	QueryEvents(ctx context.Context, contextID uuid.UUID, pg page.Page) ([]Event, error)
	CountEvents(ctx context.Context, contextID uuid.UUID) (int, error)
}
```

## File Map

### Business Layer
- `business/domain/contextbus/contextbus.go` — **NewBusiness()** — Initialize business with logger and storer
- `business/domain/contextbus/contextbus.go` — **Create()** — Create new context with default kind (project)
- `business/domain/contextbus/contextbus.go` — **Update()** — Update context fields; validate area restrictions
- `business/domain/contextbus/contextbus.go` — **Delete()** — Delete context
- `business/domain/contextbus/contextbus.go` — **Query()** — Query contexts with filter, order, pagination
- `business/domain/contextbus/contextbus.go` — **Count()** — Count filtered contexts
- `business/domain/contextbus/contextbus.go` — **QueryByID()** — Get context by ID
- `business/domain/contextbus/contextbus.go` — **AddEvent()** — Create event; update context last_event timestamp
- `business/domain/contextbus/contextbus.go` — **QueryEvents()** — Query events by context ID
- `business/domain/contextbus/contextbus.go` — **CountEvents()** — Count events for context

### Store Layer
- `business/domain/contextbus/stores/contextdb/contextdb.go` — **NewStore()** — Initialize DB store
- `business/domain/contextbus/stores/contextdb/contextdb.go` — **Create()** — INSERT context row
- `business/domain/contextbus/stores/contextdb/contextdb.go` — **Update()** — UPDATE context row
- `business/domain/contextbus/stores/contextdb/contextdb.go` — **Delete()** — DELETE context row
- `business/domain/contextbus/stores/contextdb/contextdb.go` — **Query()** — SELECT contexts with dynamic filter/order/pagination
- `business/domain/contextbus/stores/contextdb/contextdb.go` — **Count()** — COUNT contexts with filter
- `business/domain/contextbus/stores/contextdb/contextdb.go` — **QueryByID()** — SELECT single context by ID
- `business/domain/contextbus/stores/contextdb/contextdb.go` — **CreateEvent()** — INSERT event row
- `business/domain/contextbus/stores/contextdb/contextdb.go` — **QueryEvents()** — SELECT events by context ID
- `business/domain/contextbus/stores/contextdb/contextdb.go` — **CountEvents()** — COUNT events by context ID

### App Layer (Handlers)
- `app/domain/contextapp/contextapp.go` — **create()** — POST /api/v1/contexts, validate title, create context
- `app/domain/contextapp/contextapp.go` — **update()** — PUT /api/v1/contexts/{context_id}, check status transition for debrief and cascade dismissal
- `app/domain/contextapp/contextapp.go` — **delete()** — DELETE /api/v1/contexts/{context_id}
- `app/domain/contextapp/contextapp.go` — **queryAll()** — GET /api/v1/contexts, filter/sort/paginate
- `app/domain/contextapp/contextapp.go` — **queryByID()** — GET /api/v1/contexts/{context_id}
- `app/domain/contextapp/contextapp.go` — **addEvent()** — POST /api/v1/contexts/{context_id}/events, validate kind and content
- `app/domain/contextapp/contextapp.go` — **queryEvents()** — GET /api/v1/contexts/{context_id}/events
- `app/domain/contextapp/contextapp.go` — **triggerDebriefFlow()** — Create 3 pre-snoozed context_debrief clarification cards when context closes

## Impact Callouts

### ⚠ Context (`business/domain/contextbus/model.go`)
Changing this struct shape affects:
- `app/domain/contextapp/model.go` — must update toAppContext() and toBusNewContext() conversion
- `business/domain/contextbus/stores/contextdb/model.go` — must update contextDB struct and conversion functions
- `app/domain/contextapp/contextapp.go` — handlers read/write Context fields
- Migration required if DB column added/removed

### ⚠ Event (`business/domain/contextbus/model.go`)
Changing struct shape affects:
- `app/domain/contextapp/model.go` — must update toAppEvent() conversion
- `business/domain/contextbus/stores/contextdb/model.go` — must update eventDB struct and conversion functions
- `app/domain/contextapp/contextapp.go` — addEvent() and queryEvents() handlers
- Migration required if DB column added/removed

### ⚠ Storer Interface (`business/domain/contextbus/contextbus.go`)
Adding/changing a method affects:
- `business/domain/contextbus/stores/contextdb/contextdb.go` — must implement all Storer methods

### ⚠ Status (`business/domain/contextbus/model.go`)
Changing enum values affects:
- `business/domain/contextbus/contextbus.go` — Update() validates area restrictions on Closed/Paused
- `app/domain/contextapp/contextapp.go` — triggerDebriefFlow() checks status transition Active→Closed
- `app/domain/contextapp/filter.go` — parseFilter() converts user input to Status
- `business/domain/contextbus/stores/contextdb/model.go` — toDBContext() serializes status string

## Routes

| Method | Path | Handler | Auth |
|--------|------|---------|------|
| GET | /api/v1/contexts | queryAll | Required |
| GET | /api/v1/contexts/{context_id} | queryByID | Required |
| POST | /api/v1/contexts | create | Required |
| PUT | /api/v1/contexts/{context_id} | update | Required |
| DELETE | /api/v1/contexts/{context_id} | delete | Required |
| POST | /api/v1/contexts/{context_id}/events | addEvent | Required |
| GET | /api/v1/contexts/{context_id}/events | queryEvents | Required |

## Cross-Domain Dependencies

- **taskbus** — `contextapp.update()` calls `taskBus.DismissTasksByContext()` to cascade-dismiss all open/blocked tasks when a project closes
- **clarificationbus** — `contextapp.triggerDebriefFlow()` calls `clarificationBus.Create()` to generate 3 debrief clarification cards when context closes
- **contextkind** — Type system for Kind field (Project, Area)
- **debriefstatus** — Type system for DebriefStatus field (Pending, etc.)
- **contextoutcome** — Type system for Outcome field (optional enum)
