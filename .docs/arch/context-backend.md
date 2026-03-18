# Context Backend System

The Context domain manages top-level groupings of work (projects, initiatives, areas of life) with associated events and metadata. Contexts have a lifecycle (Active → Paused → Closed) and can emit events (notes, status changes, etc.) to build an audit trail and track progress.

## Core Types

### Status Enum

```go
type Status int

const (
    Active Status = iota  // "active"
    Paused                 // "paused"
    Closed                 // "closed"
)

// String() returns: "active" | "paused" | "closed" | "unknown"
// Parse(s string) (Status, error) — parses string to Status
// MustParse(s string) Status — panics on invalid input
```

### Business Domain Models

#### Context
```go
type Context struct {
    ID          uuid.UUID
    Title       string
    Description string
    Status      Status        // Active, Paused, or Closed
    Summary     string        // Optional high-level summary
    LastEvent   *time.Time    // Timestamp of most recent event (nullable)
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

#### NewContext (Creation Input)
```go
type NewContext struct {
    Title       string
    Description string
}
```

#### UpdateContext (Update Input)
```go
type UpdateContext struct {
    Title       *string  // Nullable — if nil, field not updated
    Description *string
    Status      *Status
    Summary     *string
}
```

#### Event
```go
type Event struct {
    ID        uuid.UUID
    ContextID uuid.UUID
    Kind      string              // Event type/category (e.g., "note", "status_change")
    Content   string              // Event description or message
    Metadata  *json.RawMessage    // Optional structured data (nullable)
    SourceID  *uuid.UUID          // Optional reference to originating entity (nullable)
    CreatedAt time.Time
}
```

#### NewEvent (Event Creation Input)
```go
type NewEvent struct {
    ContextID uuid.UUID
    Kind      string
    Content   string
    Metadata  *json.RawMessage    // Nullable
    SourceID  *uuid.UUID          // Nullable
}
```

### Query Filter
```go
type QueryFilter struct {
    ID     *uuid.UUID  // Filter by context ID (exact match)
    Status *Status     // Filter by status (exact match)
    Title  *string     // Filter by title (ILIKE substring match)
}
```

### Order By Constants
```go
const (
    OrderByID        = "context_id"
    OrderByTitle     = "title"
    OrderByStatus    = "status"
    OrderByLastEvent = "last_event"
    OrderByCreatedAt = "created_at"
)

var DefaultOrderBy = order.NewBy(OrderByLastEvent, order.DESC)  // Most recently updated first
```

### Storer Interface (Store Contract)

```go
type Storer interface {
    // Context operations
    Create(ctx context.Context, c Context) error
    Update(ctx context.Context, c Context) error
    Delete(ctx context.Context, c Context) error
    Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Context, error)
    Count(ctx context.Context, filter QueryFilter) (int, error)
    QueryByID(ctx context.Context, id uuid.UUID) (Context, error)

    // Event operations
    CreateEvent(ctx context.Context, e Event) error
    QueryEvents(ctx context.Context, contextID uuid.UUID, pg page.Page) ([]Event, error)
    CountEvents(ctx context.Context, contextID uuid.UUID) (int, error)
}
```

### HTTP API Models (App/Handler Layer)

#### Context (HTTP Response)
```go
type Context struct {
    ID          string  `json:"id"`                      // UUID string
    Title       string  `json:"title"`
    Description string  `json:"description"`
    Status      string  `json:"status"`                  // "active" | "paused" | "closed"
    Summary     string  `json:"summary"`
    LastEvent   *string `json:"lastEvent,omitempty"`     // RFC3339 timestamp (nullable)
    CreatedAt   string  `json:"createdAt"`               // RFC3339 timestamp
    UpdatedAt   string  `json:"updatedAt"`               // RFC3339 timestamp
}
```

#### NewContext (HTTP Request)
```go
type NewContext struct {
    Title       string `json:"title"`           // Required
    Description string `json:"description"`
}
```

#### UpdateContext (HTTP Request)
```go
type UpdateContext struct {
    Title       *string `json:"title"`          // Optional — nullable
    Description *string `json:"description"`
    Status      *string `json:"status"`
    Summary     *string `json:"summary"`
}
```

#### Event (HTTP Response)
```go
type Event struct {
    ID        string          `json:"id"`
    ContextID string          `json:"contextId"`
    Kind      string          `json:"kind"`
    Content   string          `json:"content"`
    Metadata  json.RawMessage `json:"metadata,omitempty"`    // Nullable
    SourceID  *string         `json:"sourceId,omitempty"`    // UUID string (nullable)
    CreatedAt string          `json:"createdAt"`             // RFC3339 timestamp
}
```

#### NewEvent (HTTP Request)
```go
type NewEvent struct {
    Kind     string          `json:"kind"`           // Required
    Content  string          `json:"content"`        // Required
    Metadata json.RawMessage `json:"metadata,omitempty"`
    SourceID *string         `json:"sourceId"`       // UUID string (nullable)
}
```

## File Map

### Models / Types

- **`business/domain/contextbus/model.go`** — Defines core business domain models:
  - `Context`, `NewContext`, `UpdateContext` (context management)
  - `Event`, `NewEvent` (event/audit trail)
  - `Status` enum (Active=0, Paused=1, Closed=2)
  - `Parse()`, `MustParse()`, `String()` for Status conversion
  - Error type: `ErrInvalidStatus`

- **`business/domain/contextbus/filter.go`** — Query filtering:
  - `QueryFilter` struct (ID, Status, Title filters)

- **`business/domain/contextbus/order.go`** — Sorting:
  - `OrderByID`, `OrderByTitle`, `OrderByStatus`, `OrderByLastEvent`, `OrderByCreatedAt` constants
  - `DefaultOrderBy = order.NewBy(OrderByLastEvent, order.DESC)`

- **`business/domain/contextbus/stores/contextdb/model.go`** — Database-level models:
  - `contextDB` (internal DB struct with `db:` tags: context_id, title, description, status, summary, last_event, created_at, updated_at)
  - `eventDB` (internal DB struct with `db:` tags: event_id, context_id, kind, content, metadata, source_id, created_at)
  - Conversion functions: `toDBContext()`, `toBusContext()`, `toBusContexts()`, `toDBEvent()`, `toBusEvent()`, `toBusEvents()`

- **`app/domain/contextapp/model.go`** — HTTP API models:
  - `Context`, `NewContext`, `UpdateContext` (with `json:` tags)
  - `Event`, `NewEvent` (with `json:` tags)
  - Conversion functions: `toAppContext()`, `toAppContexts()`, `toBusNewContext()`, `toBusUpdateContext()`, `toAppEvent()`, `toAppEvents()`, `toBusNewEvent()`

- **`business/types/contextstatus/contextstatus.go`** (Alternative Status implementation)
  - `Status` struct with string backing (not currently used by contextbus; different from contextbus.Status)

### App (Handlers)

- **`app/domain/contextapp/contextapp.go`** — HTTP handlers with validation:
  - **`create(ctx, *http.Request) web.Encoder`** — POST /api/v1/contexts
    - Validates: `title` is required
    - Calls `contextBus.Create()`, returns `Context` response
  - **`update(ctx, *http.Request) web.Encoder`** — PUT /api/v1/contexts/{context_id}
    - Queries existing context (404 if not found)
    - Validates status enum if provided
    - Calls `contextBus.Update()`, returns updated `Context`
  - **`delete(ctx, *http.Request) web.Encoder`** — DELETE /api/v1/contexts/{context_id}
    - Queries existing context (404 if not found)
    - Calls `contextBus.Delete()`, returns 204 No Content
  - **`queryAll(ctx, *http.Request) web.Encoder`** — GET /api/v1/contexts (with pagination)
    - Parses: `page`, `rows`, `status`, `title` query params; `orderBy`
    - Calls `contextBus.Query()` and `contextBus.Count()`
    - Returns paginated result with total count
  - **`queryByID(ctx, *http.Request) web.Encoder`** — GET /api/v1/contexts/{context_id}
    - Calls `contextBus.QueryByID()`, returns `Context` (404 if not found)
  - **`addEvent(ctx, *http.Request) web.Encoder`** — POST /api/v1/contexts/{context_id}/events
    - Validates: `kind` and `content` are required
    - Calls `contextBus.AddEvent()`, returns `Event` response
  - **`queryEvents(ctx, *http.Request) web.Encoder`** — GET /api/v1/contexts/{context_id}/events (with pagination)
    - Parses: `page`, `rows` query params
    - Calls `contextBus.QueryEvents()` and `contextBus.CountEvents()`
    - Returns paginated result with total count

- **`app/domain/contextapp/route.go`** — Route registration:
  - **`Routes.Add(a *web.App, cfg mux.Config)`** — Registers all context endpoints
    - Instantiates `contextdb.NewStore()` with logger and DB
    - Instantiates `contextbus.NewBusiness()` with logger and store
    - Instantiates handler `app` struct
    - Applies auth middleware to all routes
    - Maps HTTP methods and paths to handler methods

- **`app/domain/contextapp/filter.go`** — Query parameter parsing:
  - **`parseFilter(*http.Request) (contextbus.QueryFilter, error)`** — Extracts status, title filters from query string

- **`app/domain/contextapp/order.go`** — Order parameter parsing:
  - **`parseOrder(*http.Request) (order.By, error)`** — Maps query param `orderBy` to valid field; defaults to OrderByLastEvent DESC

### Business (Core)

- **`business/domain/contextbus/contextbus.go`** — Business logic layer:
  - **`Business` struct** — Holds logger and Storer reference
  - **`NewBusiness(log, storer) *Business`** — Constructor
  - **`Create(ctx, NewContext) (Context, error)`** — Creates new context with:
    - Generated UUID, timestamp (CreatedAt, UpdatedAt = now)
    - Default Status = Active
    - Calls `storer.Create()`
  - **`Update(ctx, Context, UpdateContext) (Context, error)`** — Updates context:
    - Applies partial updates (Title, Description, Status, Summary) if provided
    - Sets UpdatedAt = now
    - Calls `storer.Update()`
  - **`Delete(ctx, Context) error`** — Calls `storer.Delete()`
  - **`Query(ctx, QueryFilter, order.By, page.Page) ([]Context, error)`** — Calls `storer.Query()`
  - **`Count(ctx, QueryFilter) (int, error)`** — Calls `storer.Count()`
  - **`QueryByID(ctx, uuid.UUID) (Context, error)`** — Calls `storer.QueryByID()`
  - **`AddEvent(ctx, NewEvent) (Event, error)`** — Creates event and updates context:
    - Generates UUID, sets CreatedAt = now
    - Calls `storer.CreateEvent()`
    - Queries context via `storer.QueryByID()` (propagates error if not found)
    - Updates context: LastEvent = now, UpdatedAt = now
    - Calls `storer.Update()` on context
  - **`QueryEvents(ctx, uuid.UUID, page.Page) ([]Event, error)`** — Calls `storer.QueryEvents()`
  - **`CountEvents(ctx, uuid.UUID) (int, error)`** — Calls `storer.CountEvents()`

### Store

- **`business/domain/contextbus/stores/contextdb/contextdb.go`** — Database operations:
  - **`Store` struct** — Holds logger and sqlx.ExtContext (db connection)
  - **`NewStore(log, db) *Store`** — Constructor

  **Context operations:**
  - **`Create(ctx, Context) error`** — INSERT into `contexts` table; uses named parameters (`:context_id`, `:title`, `:description`, `:status`, `:summary`, `:last_event`, `:created_at`, `:updated_at`)
  - **`Update(ctx, Context) error`** — UPDATE `contexts` set all fields where `context_id = :context_id`
  - **`Delete(ctx, Context) error`** — DELETE from `contexts` where `context_id = :context_id`
  - **`Query(ctx, QueryFilter, order.By, page.Page) ([]Context, error)`** — SELECT with dynamic filter clauses and ORDER BY + OFFSET/FETCH NEXT pagination
  - **`Count(ctx, QueryFilter) (int, error)`** — SELECT COUNT(*) with dynamic filter clauses
  - **`QueryByID(ctx, uuid.UUID) (Context, error)`** — SELECT single context by `context_id`; returns `sqldb.ErrDBNotFound` if not found

  **Event operations:**
  - **`CreateEvent(ctx, Event) error`** — INSERT into `context_events` table; uses named parameters (`:event_id`, `:context_id`, `:kind`, `:content`, `:metadata`, `:source_id`, `:created_at`)
  - **`QueryEvents(ctx, uuid.UUID, page.Page) ([]Event, error)`** — SELECT from `context_events` WHERE `context_id = :context_id` ORDER BY `created_at DESC` with pagination
  - **`CountEvents(ctx, uuid.UUID) (int, error)`** — SELECT COUNT(*) from `context_events` WHERE `context_id = :context_id`

- **`business/domain/contextbus/stores/contextdb/filter.go`** — Dynamic filter building:
  - **`applyFilter(QueryFilter, map[string]any, *bytes.Buffer)`** — Appends WHERE clauses:
    - ID filter: `AND context_id = :id`
    - Status filter: `AND status = :filter_status` (converted to string via `Status.String()`)
    - Title filter: `AND title ILIKE :filter_title` (case-insensitive substring with `%` wildcards)

- **`business/domain/contextbus/stores/contextdb/order.go`** — Order clause building:
  - **`orderByFields` map** — Maps business field names to DB column names (e.g., `contextbus.OrderByID` → `"context_id"`)
  - **`orderByClause(order.By) (string, error)`** — Builds ORDER BY clause; returns error for unknown fields

## Database Schema

### Tables

**contexts**
```sql
CREATE TABLE contexts (
    context_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title         TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active', 'paused', 'closed')),
    summary       TEXT NOT NULL DEFAULT '',
    last_event    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**context_events**
```sql
CREATE TABLE context_events (
    event_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    context_id    UUID NOT NULL REFERENCES contexts(context_id) ON DELETE CASCADE,
    kind          TEXT NOT NULL,
    content       TEXT NOT NULL,
    metadata      JSONB,
    source_id     UUID,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_context_events_context ON context_events(context_id, created_at DESC);
```

## Routes

| Method | Path | Handler | Auth | Notes |
|--------|------|---------|------|-------|
| GET | `/api/v1/contexts` | `queryAll` | Required | Query params: `page`, `rows`, `status`, `title`, `orderBy` |
| GET | `/api/v1/contexts/{context_id}` | `queryByID` | Required | Returns 404 if not found |
| POST | `/api/v1/contexts` | `create` | Required | Body: `{title, description}` (title required) |
| PUT | `/api/v1/contexts/{context_id}` | `update` | Required | Body: `{title?, description?, status?, summary?}` (all optional) |
| DELETE | `/api/v1/contexts/{context_id}` | `delete` | Required | Returns 204 No Content |
| POST | `/api/v1/contexts/{context_id}/events` | `addEvent` | Required | Body: `{kind, content, metadata?, sourceId?}` (kind, content required) |
| GET | `/api/v1/contexts/{context_id}/events` | `queryEvents` | Required | Query params: `page`, `rows` |

## Impact Callouts

### ⚠ Context Struct (`business/domain/contextbus/model.go`)
Changing this struct shape affects:
- `app/domain/contextapp/model.go` — `toAppContext()`, `toAppContexts()` conversion functions must map all fields
- `business/domain/contextbus/stores/contextdb/model.go` — `toDBContext()`, `toBusContext()` conversion functions must handle new fields
- SQL schema in `business/sdk/migrate/sql/migrate.sql` — `contexts` table columns must be added/modified
- Handler validation in `app/domain/contextapp/contextapp.go` — Update, Create handlers may need new validation rules
- All SELECT/INSERT/UPDATE queries in `contextdb.go` must be updated to reference new columns

### ⚠ Status Enum (`business/domain/contextbus/model.go`)
Changing status values or adding new statuses affects:
- `Parse()` and `String()` methods — must handle all enum values
- Database constraint in `business/sdk/migrate/sql/migrate.sql` — CHECK (status IN (...))
- `app/domain/contextapp/model.go` — `toBusUpdateContext()` status parsing
- HTTP API contract — clients must accept new status strings
- Filter logic in `business/domain/contextbus/stores/contextdb/filter.go`

### ⚠ Event Struct (`business/domain/contextbus/model.go`)
Changing this struct shape affects:
- `app/domain/contextapp/model.go` — `toAppEvent()`, `toAppEvents()` conversion functions
- `business/domain/contextbus/stores/contextdb/model.go` — `toDBEvent()`, `toBusEvent()`, `toBusEvents()` conversion functions
- SQL schema in `business/sdk/migrate/sql/migrate.sql` — `context_events` table columns
- All SELECT/INSERT queries in `contextdb.go` must be updated
- `contextBus.AddEvent()` — event initialization logic may need updates

### ⚠ Storer Interface (`business/domain/contextbus/contextbus.go`)
Adding or changing a method affects:
- `business/domain/contextbus/stores/contextdb/contextdb.go` — must implement the full interface (Go compiler enforces this)
- `app/domain/contextapp/contextapp.go` — any new handler needing new store capabilities must call the new method
- Migration/test setup code that instantiates the store

**Methods and implementations:**
- `Create(ctx context.Context, c Context) error` — implemented by `Store.Create()` (INSERT query)
- `Update(ctx context.Context, c Context) error` — implemented by `Store.Update()` (UPDATE query)
- `Delete(ctx context.Context, c Context) error` — implemented by `Store.Delete()` (DELETE query)
- `Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Context, error)` — implemented by `Store.Query()` (SELECT with dynamic filters)
- `Count(ctx context.Context, filter QueryFilter) (int, error)` — implemented by `Store.Count()` (SELECT COUNT(*))
- `QueryByID(ctx context.Context, id uuid.UUID) (Context, error)` — implemented by `Store.QueryByID()` (SELECT single)
- `CreateEvent(ctx context.Context, e Event) error` — implemented by `Store.CreateEvent()` (INSERT event)
- `QueryEvents(ctx context.Context, contextID uuid.UUID, pg page.Page) ([]Event, error)` — implemented by `Store.QueryEvents()` (SELECT events)
- `CountEvents(ctx context.Context, contextID uuid.UUID) (int, error)` — implemented by `Store.CountEvents()` (SELECT COUNT(*) events)

### ⚠ Query/Update Flow in contextBus
The `AddEvent()` method performs a two-step store operation:
1. **`storer.CreateEvent(ctx, event)`** — Inserts the event
2. **`storer.QueryByID(ctx, contextID)`** — Fetches the context
3. **`storer.Update(ctx, context)`** — Updates `context.LastEvent` and `context.UpdatedAt`

If any step fails, the error propagates. If step 2 fails (context not found), it returns "query context for event update: %w". This design assumes:
- The context exists when `AddEvent()` is called (no validation in handler)
- The context is updated atomically with the event (but as two separate queries — not truly atomic)

Changing this requires updating:
- `contextbus.AddEvent()` logic
- Any handler calling `contextBus.AddEvent()` to validate context exists beforehand (currently handled by app/domain/contextapp/contextapp.go, but no pre-check)

### ⚠ FilterAndOrderBy Constants
Changing or removing order/filter fields affects:
- `app/domain/contextapp/order.go` — `orderByFields` map must be updated to parse query param to business constant
- `business/domain/contextbus/stores/contextdb/order.go` — `orderByFields` map must map business constant to DB column name
- HTTP API contract — clients use the field names in `orderBy` query parameter
- Filter parsing in `app/domain/contextapp/filter.go` — new filters may need new parsing logic

## Cross-Domain Dependencies

- **On tasks domain** — `context_id` FK in tasks table; context can be deleted (SET NULL on tasks)
- **On tags domain** — `context_tags` junction table; context can be deleted (CASCADE)
- **On logger** — All business and store layers accept `*logger.Logger` for structured logging
- **On sqldb** — Store uses sqldb utility functions (`NamedExecContext`, `NamedQuerySlice`, `NamedQueryStruct`) and `sqldb.ErrDBNotFound` error type
- **On web framework** — Handlers use `web.Encode` response, `web.Decode` request, `web.Param` path extraction, `web.NoResponse` for 204 responses
- **On order & page SDKs** — Business layer uses `order.By` and `page.Page` for pagination and sorting abstractions
- **On middleware** — Route registration uses `mid.Auth(cfg.APIKey)` to enforce API key authentication on all endpoints
