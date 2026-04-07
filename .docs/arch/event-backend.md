# Event Backend System

> The event domain represents calendar events—time-bound occurrences with optional location and context linkage. It supports full CRUD, filtering by context ID and date range (DateFrom/DateTo), ordering by start time or creation time, and pagination. Events may be optionally linked to a context via a nullable FK. All five routes are protected by API-key auth.

---

## Core Types

### Business Layer — `business/domain/eventbus/model.go`

```go
// Event represents a calendar event with start/end times and optional context linkage.
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
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewEvent is the input DTO for creating an event. RawInputID is system-managed (not exposed to API).
type NewEvent struct {
	ContextID   *uuid.UUID
	Title       string
	Description string
	Location    *string
	StartsAt    time.Time
	EndsAt      time.Time
	AllDay      bool
	RawInputID  *uuid.UUID
}

// UpdateEvent is the input DTO for updating an event. All fields are optional.
type UpdateEvent struct {
	ContextID   *uuid.UUID
	Title       *string
	Description *string
	Location    *string
	StartsAt    *time.Time
	EndsAt      *time.Time
	AllDay      *bool
}
```

### Store Layer — `business/domain/eventbus/stores/eventdb/model.go`

```go
// eventDB is the raw database row struct with SQL tags.
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
}
```

Converters:
- `toDBEvent(eventbus.Event) eventDB` — business → store layer
- `toBusEvent(eventDB) eventbus.Event` — store → business layer
- `toBusEvents([]eventDB) []eventbus.Event` — batch conversion

### App Layer — `app/domain/eventapp/model.go`

```go
// Event is the JSON response DTO returned by all read and write handlers.
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
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

// NewEvent is the request body for POST /api/v1/events.
type NewEvent struct {
	ContextID   *string `json:"contextId"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Location    *string `json:"location"`
	StartsAt    string  `json:"startsAt"`
	EndsAt      string  `json:"endsAt"`
	AllDay      bool    `json:"allDay"`
}

// UpdateEvent is the request body for PUT /api/v1/events/{event_id}. All fields optional.
type UpdateEvent struct {
	ContextID   *string `json:"contextId"`
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Location    *string `json:"location"`
	StartsAt    *string `json:"startsAt"`
	EndsAt      *string `json:"endsAt"`
	AllDay      *bool   `json:"allDay"`
}

// Encode implements web.Encoder (JSON serialization).
func (e Event) Encode() ([]byte, string, error) {
	data, err := json.Marshal(e)
	return data, "application/json", err
}
```

Converters:
- `toAppEvent(eventbus.Event) Event` — business → app layer (timestamps to RFC3339, UUIDs to strings)
- `toAppEvents([]eventbus.Event) []Event` — batch conversion
- `toBusNewEvent(NewEvent) (eventbus.NewEvent, error)` — app → business layer (parses timestamps, UUIDs)
- `toBusUpdateEvent(UpdateEvent) (eventbus.UpdateEvent, error)` — app → business layer

### Filter & Order Types

**`business/domain/eventbus/filter.go`:**
```go
type QueryFilter struct {
	ContextID *uuid.UUID
	DateFrom  *time.Time
	DateTo    *time.Time
}
```

**`business/domain/eventbus/order.go`:**
```go
const (
	OrderByStartsAt  = "starts_at"
	OrderByCreatedAt = "created_at"
)

var DefaultOrderBy = order.NewBy(OrderByStartsAt, order.ASC)
```

---

## File Map

### App (Handlers) — `app/domain/eventapp/`

- **`eventapp.go`** — `queryAll(ctx, r) web.Encoder` (GET /api/v1/events) — parses pagination/filter/order, queries events, counts total, returns paginated result
- **`eventapp.go`** — `queryByID(ctx, r) web.Encoder` (GET /api/v1/events/{event_id}) — fetches single event by UUID, returns 404 if not found
- **`eventapp.go`** — `create(ctx, r) web.Encoder` (POST /api/v1/events) — validates title/startsAt/endsAt required, converts app DTO to business DTO, creates event, returns response DTO; if `ContextID == nil` after creation, fires `asyncClassify` goroutine
- **`eventapp.go`** — `update(ctx, r) web.Encoder` (PUT /api/v1/events/{event_id}) — fetches current event, decodes partial update, applies changes, returns updated response DTO
- **`eventapp.go`** — `delete(ctx, r) web.Encoder` (DELETE /api/v1/events/{event_id}) — fetches event, deletes it, returns 204 NoResponse
- **`eventapp.go`** — `asyncClassify(ctx, entityType, entityID, text)` — background goroutine; queries active contexts, calls extractor.ExtractText, then either directly links the event (confidence >= 0.7) or creates a clarification card
- **`route.go`** — `Routes.Add(a *web.App, cfg mux.Config)` — instantiates eventdb.Store and eventbus.Business; also wires contextbus, clarificationbus, and extractor (Claude/Ollama failover) for auto-classification; registers all five routes with API-key auth middleware
- **`filter.go`** — `parseFilter(r *http.Request) (eventbus.QueryFilter, error)` — maps query params (context_id, date_from, date_to) to business QueryFilter
- **`order.go`** — `parseOrder(r *http.Request) (order.By, error)` — maps orderBy query param to business order constant, defaults to starts_at ASC

### Business (Core Logic) — `business/domain/eventbus/`

- **`eventbus.go`** — `Business` struct holds storer + logger
- **`eventbus.go`** — `NewBusiness(log, storer) *Business` — constructor
- **`eventbus.go`** — `Create(ctx, NewEvent) (Event, error)` — generates UUID + timestamps, inserts via storer
- **`eventbus.go`** — `Update(ctx, event Event, UpdateEvent) (Event, error)` — applies optional field updates (null-safe), sets UpdatedAt, persists via storer
- **`eventbus.go`** — `Delete(ctx, event Event) error` — deletes via storer
- **`eventbus.go`** — `Query(ctx, filter QueryFilter, orderBy order.By, page page.Page) ([]Event, error)` — delegates to storer
- **`eventbus.go`** — `Count(ctx, filter QueryFilter) (int, error)` — counts matching events via storer
- **`eventbus.go`** — `QueryByID(ctx, id uuid.UUID) (Event, error)` — fetches single event via storer
- **`eventbus.go`** — `Storer` interface defines contract: Create, Update, Delete, Query, Count, QueryByID

### Store (Database) — `business/domain/eventbus/stores/eventdb/`

- **`eventdb.go`** — `Store` struct holds logger + sqlx.ExtContext
- **`eventdb.go`** — `NewStore(log, db) *Store` — constructor
- **`eventdb.go`** — `Create(ctx, event) error` — named parameter insert into events table
- **`eventdb.go`** — `Update(ctx, event) error` — named parameter update, WHERE event_id = :event_id
- **`eventdb.go`** — `Delete(ctx, event) error` — named parameter delete by event_id
- **`eventdb.go`** — `Query(ctx, filter, orderBy, page) ([]Event, error)` — builds WHERE clause via applyFilter, appends ORDER BY and pagination, executes SELECT
- **`eventdb.go`** — `Count(ctx, filter) (int, error)` — builds WHERE clause, executes COUNT(*)
- **`eventdb.go`** — `QueryByID(ctx, id) (Event, error)` — fetches single row by event_id
- **`filter.go`** — `applyFilter(filter, data, buf)` — appends AND clauses for ContextID (=), DateFrom (starts_at >=), DateTo (ends_at <=)
- **`order.go`** — `orderByClause(ob order.By) (string, error)` — maps business field constant to SQL column (starts_at or created_at) + direction

---

## Impact Callouts

### ⚠ Event struct (`business/domain/eventbus/model.go`)

**Used by:**
- `business/domain/eventbus/eventbus.go` — all five business methods (Create, Update, Delete, Query, Count, QueryByID) accept or return Event
- `business/domain/eventbus/stores/eventdb/model.go` — toDBEvent() and toBusEvent() converters
- `app/domain/eventapp/model.go` — toAppEvent() converter, toAppEvents() batch
- `app/domain/eventapp/eventapp.go` — all five handlers work with Event (via Business methods)

**If modified:** Update converters in model.go files (eventdb + eventapp), update handler validation, update SQL INSERT/UPDATE/SELECT clauses.

### ⚠ NewEvent / UpdateEvent structs

**Used by:**
- `app/domain/eventapp/model.go` — toBusNewEvent() and toBusUpdateEvent() converters
- `app/domain/eventapp/eventapp.go` — create() and update() handlers decode request bodies into these types
- `business/domain/eventbus/eventbus.go` — Create() and Update() methods accept these as parameters

**If modified:** Update validation in handlers (create/update), update converters (timestamp/UUID parsing), update business layer method signatures.

### ⚠ Storer interface (`business/domain/eventbus/eventbus.go`)

**Implemented by:**
- `business/domain/eventbus/stores/eventdb/eventdb.go` — Store struct implements all six methods

**If modified:** Add method to eventdb.Store, update Business methods that call storer, update handlers that depend on new storer capability.

### ⚠ QueryFilter struct (`business/domain/eventbus/filter.go`)

**Used by:**
- `app/domain/eventapp/filter.go` — parseFilter() populates from query params (context_id, date_from, date_to)
- `business/domain/eventbus/eventbus.go` — Query() and Count() methods accept QueryFilter
- `business/domain/eventbus/stores/eventdb/filter.go` — applyFilter() builds WHERE clauses from QueryFilter fields

**If modified:** Add query param parsing in eventapp/filter.go, add WHERE clause logic in eventdb/filter.go, both directions.

### ⚠ Order constants (`business/domain/eventbus/order.go`)

**Used by:**
- `app/domain/eventapp/order.go` — orderByFields map translates request field names to business constants
- `business/domain/eventbus/stores/eventdb/order.go` — orderByFields map translates business constants to SQL columns
- `business/domain/eventbus/eventbus.go` — Query() method accepts order.By (uses constants)

**If modified:** Add constant, add to eventdb orderByFields (→ SQL column), add to eventapp orderByFields (← request field name), update DefaultOrderBy if needed.

---

## Routes

| Method | Path | Handler | Auth |
|--------|------|---------|------|
| GET | /api/v1/events | queryAll | API key |
| GET | /api/v1/events/{event_id} | queryByID | API key |
| POST | /api/v1/events | create | API key |
| PUT | /api/v1/events/{event_id} | update | API key |
| DELETE | /api/v1/events/{event_id} | delete | API key |

All routes registered in `app/domain/eventapp/route.go` → `Routes.Add()`. Auth middleware applied via `mid.Auth(cfg.APIKey)`.

---

## Cross-Domain Dependencies

**Imports from shared infrastructure:**
- `github.com/casebrophy/planner/business/sdk/order` — order parsing/constant types
- `github.com/casebrophy/planner/business/sdk/page` — pagination logic
- `github.com/casebrophy/planner/business/sdk/sqldb` — NamedExecContext, NamedQuerySlice, NamedQueryStruct helpers
- `github.com/casebrophy/planner/foundation/logger` — structured logging
- `github.com/casebrophy/planner/foundation/web` — HTTP framework (App, Handle, Encoder, Param)
- `github.com/casebrophy/planner/app/sdk/errs` — error codes (InvalidArgument, NotFound, Internal)
- `github.com/casebrophy/planner/app/sdk/query` — paginated result wrapper
- `github.com/casebrophy/planner/app/sdk/mid` — middleware (Auth)
- `github.com/casebrophy/planner/app/sdk/mux` — mux config (Log, DB, APIKey)

**Auto-classify pipeline (noteapp/eventapp pattern):**
- `asyncClassify()` fires in a goroutine after event creation when `ContextID == nil`
- Queries active contexts via `contextbus`, calls `extractor.ExtractText()`, applies 0.7 confidence threshold
- High confidence → direct `eventbus.Update()` with matched context
- Low confidence → creates `clarificationbus.ClarificationItem` of kind `ContextAssignment`
- Additional dependencies: `business/domain/contextbus`, `business/domain/clarificationbus`, `business/domain/ingestbus/extractor`, `business/types/clarificationkind`

---

## Database Table

```sql
CREATE TABLE events (
    event_id UUID PRIMARY KEY,
    context_id UUID NULLABLE,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    location TEXT NULLABLE,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    all_day BOOLEAN NOT NULL,
    raw_input_id UUID NULLABLE,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Indexes for common query patterns
CREATE INDEX idx_events_context_id ON events(context_id);
CREATE INDEX idx_events_starts_at ON events(starts_at);
CREATE INDEX idx_events_created_at ON events(created_at);
```

---

## Query Examples

**List all events (default sort by start time):**
```
GET /api/v1/events?page=1&rows=20
```

**Filter by context and date range:**
```
GET /api/v1/events?context_id=<uuid>&date_from=2026-04-01T00:00:00Z&date_to=2026-04-30T23:59:59Z&page=1&rows=20
```

**Sort by creation time (descending):**
```
GET /api/v1/events?orderBy=created_at&page=1&rows=20
```

**Fetch single event:**
```
GET /api/v1/events/<event_id>
```

**Create event:**
```
POST /api/v1/events
{
  "title": "Team Standup",
  "description": "Daily sync",
  "startsAt": "2026-04-01T10:00:00Z",
  "endsAt": "2026-04-01T10:30:00Z",
  "allDay": false,
  "contextId": "<uuid or null>",
  "location": "Conference Room A"
}
```

**Update event (partial):**
```
PUT /api/v1/events/<event_id>
{
  "title": "Team Standup (Rescheduled)",
  "startsAt": "2026-04-01T14:00:00Z"
}
```

**Delete event:**
```
DELETE /api/v1/events/<event_id>
```
