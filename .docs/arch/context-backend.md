# Context Backend System

> Manages user projects and ongoing areas (contexts) with lifecycle tracking, event threading, and debrief flows. Contexts are long-lived organizational units that group related work—projects are time-bounded and can be closed (triggering debrief and task cascade), while areas are permanent and cannot be closed.

## Core Types

### Business Layer Types

```go
type Context struct {
	ID            uuid.UUID
	Title         string
	Description   string
	Kind          contextkind.Kind      // "project" or "area"
	Status        Status                // Active, Paused, Closed
	Summary       string
	LastEvent     *time.Time
	LastThreadAt  *time.Time
	DebriefStatus debriefstatus.Status  // Pending, Completed, Dismissed
	Outcome       *contextoutcome.Outcome
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type NewContext struct {
	Title       string
	Description string
	Kind        contextkind.Kind
}

type UpdateContext struct {
	Title         *string
	Description   *string
	Kind          *contextkind.Kind
	Status        *Status
	Summary       *string
	DebriefStatus *debriefstatus.Status
	Outcome       *contextoutcome.Outcome
}

type Event struct {
	ID        uuid.UUID
	ContextID uuid.UUID
	Kind      string
	Content   string
	Metadata  *json.RawMessage
	SourceID  *uuid.UUID
	CreatedAt time.Time
}

type NewEvent struct {
	ContextID uuid.UUID
	Kind      string
	Content   string
	Metadata  *json.RawMessage
	SourceID  *uuid.UUID
}

type Status int

const (
	Active Status = iota // 0
	Paused Status = 1    // 1
	Closed Status = 2    // 2
)

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

### App Layer DTOs

```go
type Context struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	Status        string  `json:"status"`
	Kind          string  `json:"kind"`
	Summary       string  `json:"summary"`
	LastEvent     *string `json:"lastEvent,omitempty"`
	LastThreadAt  *string `json:"lastThreadAt,omitempty"`
	DebriefStatus string  `json:"debriefStatus"`
	Outcome       *string `json:"outcome,omitempty"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
}

type NewContext struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
}

type UpdateContext struct {
	Title         *string `json:"title"`
	Description   *string `json:"description"`
	Status        *string `json:"status"`
	Kind          *string `json:"kind"`
	Summary       *string `json:"summary"`
	DebriefStatus *string `json:"debriefStatus"`
	Outcome       *string `json:"outcome"`
}

type Event struct {
	ID        string          `json:"id"`
	ContextID string          `json:"contextId"`
	Kind      string          `json:"kind"`
	Content   string          `json:"content"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	SourceID  *string         `json:"sourceId,omitempty"`
	CreatedAt string          `json:"createdAt"`
}

type NewEvent struct {
	Kind     string          `json:"kind"`
	Content  string          `json:"content"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
	SourceID *string         `json:"sourceId"`
}
```

### Store Layer Models

```go
type contextDB struct {
	ID            uuid.UUID  `db:"context_id"`
	Title         string     `db:"title"`
	Description   string     `db:"description"`
	Kind          string     `db:"kind"`
	Status        string     `db:"status"`
	Summary       string     `db:"summary"`
	LastEvent     *time.Time `db:"last_event"`
	LastThreadAt  *time.Time `db:"last_thread_at"`
	DebriefStatus string     `db:"debrief_status"`
	Outcome       *string    `db:"outcome"`
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"`
}

type eventDB struct {
	ID        uuid.UUID        `db:"event_id"`
	ContextID uuid.UUID        `db:"context_id"`
	Kind      string           `db:"kind"`
	Content   string           `db:"content"`
	Metadata  *json.RawMessage `db:"metadata"`
	SourceID  *uuid.UUID       `db:"source_id"`
	CreatedAt time.Time        `db:"created_at"`
}
```

## File Map

### App Layer (app/domain/contextapp/)
- `contextapp.go` — **create()** POST /api/v1/contexts; **update()** PUT with debrief flow on close, cascade task dismissal for projects; **delete()** DELETE; **queryAll()** GET with filter/order/page; **queryByID()** GET single; **addEvent()** POST /{id}/events; **queryEvents()** GET /{id}/events; **triggerDebriefFlow()** creates 3 pre-snoozed clarification cards (24h) on project close
- `model.go` — **toAppContext()**, **toAppContexts()**, **toBusNewContext()**, **toBusUpdateContext()**, **toAppEvent()**, **toAppEvents()**, **toBusNewEvent()** converters
- `route.go` — **Routes.Add()** wires store → business → handlers, registers 7 endpoints with auth middleware
- `filter.go` — **parseFilter()** parses query params (status, kind, title) → QueryFilter
- `order.go` — **parseOrder()** maps request orderBy field names; defaults to last_event DESC

### Business Layer (business/domain/contextbus/)
- `contextbus.go` — **Create()** defaults status=Active, debriefStatus=Pending; **Update()** enforces area invariant (areas cannot close/pause); **Delete/Query/Count/QueryByID** delegate to storer; **AddEvent()** creates Event + updates context.LastEvent and UpdatedAt; **QueryEvents/CountEvents** delegate to storer
- `model.go` — Context, NewContext, UpdateContext, Event, NewEvent, Status types with Parse/MustParse/String
- `filter.go` — QueryFilter struct (ID, Status, Kind, Title)
- `order.go` — OrderByID, OrderByTitle, OrderByStatus, OrderByLastEvent, OrderByCreatedAt; DefaultOrderBy = last_event DESC

### Store Layer (business/domain/contextbus/stores/contextdb/)
- `contextdb.go` — **Create/Update/Delete/Query/Count/QueryByID** for contexts; **CreateEvent/QueryEvents/CountEvents** for events
- `model.go` — contextDB and eventDB structs; **toDBContext()**, **toBusContext()**, **toBusContexts()**, **toDBEvent()**, **toBusEvent()**, **toBusEvents()** converters
- `filter.go` — **applyFilter()** WHERE clauses (ID exact, Status/Kind equality, Title ILIKE)
- `order.go` — orderByFields map; **orderByClause()** translates constants to SQL column names

## Impact Callouts

### ⚠ Context Struct (business/domain/contextbus/model.go)
Adding/changing fields affects:
- `contextapp/model.go` — app Context DTO + toAppContext() converter
- `contextdb/model.go` — contextDB struct db tags + converters
- `contextdb/contextdb.go` — INSERT/UPDATE/SELECT SQL column lists
- Migration SQL required for new/removed DB columns

### ⚠ UpdateContext Struct (business/domain/contextbus/model.go)
Changing pointer fields affects:
- `contextapp/model.go` — app UpdateContext DTO + toBusUpdateContext() converter
- `contextbus.go` — Update() method must apply new field

### ⚠ Status Enum (business/domain/contextbus/model.go)
Adding new statuses affects:
- `model.go` — Parse(), MustParse(), String() methods
- `contextdb/filter.go` — applyFilter() status comparison
- `contextapp/filter.go` — parseFilter() must handle new status value
- Database CHECK constraint on contexts.status column

### ⚠ Storer Interface (business/domain/contextbus/contextbus.go)
Adding/changing methods affects:
- `contextdb/contextdb.go` — must implement new method

### ⚠ Area Invariant (business/domain/contextbus/contextbus.go)
Area contexts (kind="area") cannot transition to Closed or Paused. Update() enforces this. Frontend must prevent UI from allowing close/pause on areas.

### ⚠ Task Cascade (contextapp/contextapp.go)
When a project transitions to Closed, update() calls taskBus.DismissTasksByContext(). Only projects cascade (areas are permanent). Errors are logged but don't fail the context update.

### ⚠ Debrief Flow (contextapp/contextapp.go)
triggerDebriefFlow() depends on clarificationbus and taskbus. Creates 3 hardcoded clarification cards (outcome, challenge, lesson) with 24h snooze. If either bus is nil, flow silently skips.

### ⚠ Event Struct (business/domain/contextbus/model.go)
Adding fields affects:
- `contextapp/model.go` — app Event DTO + toAppEvent() converter
- `contextdb/model.go` — eventDB struct + converters
- `contextdb/contextdb.go` — CreateEvent/QueryEvents SQL

## Routes

| Method | Path | Handler |
|--------|------|---------|
| GET | /api/v1/contexts | queryAll — filter/order/pagination |
| POST | /api/v1/contexts | create — title required, defaults to project |
| GET | /api/v1/contexts/{context_id} | queryByID |
| PUT | /api/v1/contexts/{context_id} | update — triggers debrief/cascade on close |
| DELETE | /api/v1/contexts/{context_id} | delete — hard delete |
| POST | /api/v1/contexts/{context_id}/events | addEvent — kind/content required |
| GET | /api/v1/contexts/{context_id}/events | queryEvents — paginated, DESC by created_at |

All routes require `X-API-Key` header (mid.Auth middleware).

## Cross-Domain Dependencies

- **taskbus** — DismissTasksByContext(contextID) called on project close
- **clarificationbus** — creates 3 context_debrief clarification items on project close
- **contextkind** — business/types/contextkind; "project" and "area" control lifecycle rules
- **contextoutcome** — business/types/contextoutcome; success/failure/neutral/inconclusive
- **debriefstatus** — business/types/debriefstatus; Pending/Completed/Dismissed; defaults to Pending on create
