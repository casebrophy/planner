# Context Backend Architecture

The Context domain manages top-level groupings of work (projects, initiatives, areas of life) with associated events and metadata. Contexts have a lifecycle (Active → Paused → Closed) and can emit events (notes, status changes, etc.) to build an audit trail and track progress.

## Core Types

### Status Enum (`business/domain/contextbus/model.go`)

```go
type Status int

const (
    Active Status = iota  // "active"
    Paused                 // "paused"
    Closed                 // "closed"
)

// String() string             — returns "active" | "paused" | "closed" | "unknown"
// Parse(s string) (Status, error)
// MustParse(s string) Status
```

Error type: `ErrInvalidStatus = StatusError("invalid status")`

### Business Layer Models (`business/domain/contextbus/model.go`)

```go
type Context struct {
    ID            uuid.UUID
    Title         string
    Description   string
    Status        Status                   // Active, Paused, or Closed
    Summary       string                   // Optional high-level summary
    LastEvent     *time.Time               // Timestamp of most recent event (nullable)
    LastThreadAt  *time.Time               // Most recent thread entry (system-managed)
    DebriefStatus debriefstatus.Status     // pending, done, skipped
    Outcome       *contextoutcome.Outcome  // went_well, mixed, difficult, ongoing_issues (nullable)
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

type NewContext struct {
    Title       string
    Description string
}

type UpdateContext struct {
    Title         *string                  // nil = do not update
    Description   *string
    Status        *Status
    Summary       *string
    DebriefStatus *debriefstatus.Status
    Outcome       *contextoutcome.Outcome
}

type Event struct {
    ID        uuid.UUID
    ContextID uuid.UUID
    Kind      string           // Event type/category (e.g., "note", "status_change")
    Content   string           // Event body text
    Metadata  *json.RawMessage // Optional structured data (nullable)
    SourceID  *uuid.UUID       // Optional reference to originating entity (nullable)
    CreatedAt time.Time
}

type NewEvent struct {
    ContextID uuid.UUID
    Kind      string
    Content   string
    Metadata  *json.RawMessage // nullable
    SourceID  *uuid.UUID       // nullable
}
```

### Query Filter (`business/domain/contextbus/filter.go`)

```go
type QueryFilter struct {
    ID     *uuid.UUID  // Exact match on context_id
    Status *Status     // Exact match on status
    Title  *string     // Case-insensitive substring match (ILIKE)
}
```

### Order By Constants (`business/domain/contextbus/order.go`)

```go
const (
    OrderByID        = "context_id"
    OrderByTitle     = "title"
    OrderByStatus    = "status"
    OrderByLastEvent = "last_event"
    OrderByCreatedAt = "created_at"
)

var DefaultOrderBy = order.NewBy(OrderByLastEvent, order.DESC)
```

### Storer Interface (`business/domain/contextbus/contextbus.go`)

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

### Store Layer Models (`business/domain/contextbus/stores/contextdb/model.go`)

```go
type contextDB struct {
    ID            uuid.UUID  `db:"context_id"`
    Title         string     `db:"title"`
    Description   string     `db:"description"`
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

Conversion functions: `toDBContext()`, `toBusContext()`, `toBusContexts()`, `toDBEvent()`, `toBusEvent()`, `toBusEvents()`

### App Layer Models (`app/domain/contextapp/model.go`)

```go
type Context struct {
    ID            string  `json:"id"`
    Title         string  `json:"title"`
    Description   string  `json:"description"`
    Status        string  `json:"status"`                  // "active" | "paused" | "closed"
    Summary       string  `json:"summary"`
    LastEvent     *string `json:"lastEvent,omitempty"`     // RFC3339 timestamp, nullable
    LastThreadAt  *string `json:"lastThreadAt,omitempty"`  // RFC3339 timestamp, nullable
    DebriefStatus string  `json:"debriefStatus"`           // "pending" | "done" | "skipped"
    Outcome       *string `json:"outcome,omitempty"`       // "went_well" | "mixed" | "difficult" | "ongoing_issues"
    CreatedAt     string  `json:"createdAt"`               // RFC3339 timestamp
    UpdatedAt     string  `json:"updatedAt"`               // RFC3339 timestamp
}

type NewContext struct {
    Title       string `json:"title"`           // Required
    Description string `json:"description"`
}

type UpdateContext struct {
    Title         *string `json:"title"`
    Description   *string `json:"description"`
    Status        *string `json:"status"`
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
    SourceID  *string         `json:"sourceId,omitempty"`   // UUID string, nullable
    CreatedAt string          `json:"createdAt"`             // RFC3339 timestamp
}

type NewEvent struct {
    Kind     string          `json:"kind"`    // Required
    Content  string          `json:"content"` // Required
    Metadata json.RawMessage `json:"metadata,omitempty"`
    SourceID *string         `json:"sourceId"`
}
```

Conversion functions: `toAppContext()`, `toAppContexts()`, `toBusNewContext()`, `toBusUpdateContext()`, `toAppEvent()`, `toAppEvents()`, `toBusNewEvent()`

## File Map

### App Layer

| File | Responsibility |
|------|---------------|
| `app/domain/contextapp/contextapp.go` | HTTP handler methods |
| `app/domain/contextapp/model.go` | HTTP DTOs + toApp*/toBus* converters |
| `app/domain/contextapp/filter.go` | `parseFilter()` — maps query params to `QueryFilter` |
| `app/domain/contextapp/order.go` | `parseOrder()` — maps `orderBy` query param to business constant |
| `app/domain/contextapp/route.go` | `Routes.Add()` — registers endpoints, wires store + business + handler |

**Handler methods in `contextapp.go`:**
- `create(ctx, *http.Request) web.Encoder` — validates title required; calls `contextBus.Create()`
- `update(ctx, *http.Request) web.Encoder` — fetches existing context (404 if missing); validates status enum; calls `contextBus.Update()`
- `delete(ctx, *http.Request) web.Encoder` — fetches existing context (404 if missing); calls `contextBus.Delete()`; returns `web.NoResponse{}`
- `queryAll(ctx, *http.Request) web.Encoder` — parses `page`, `rows`, `status`, `title`, `orderBy`; calls `contextBus.Query()` + `Count()`; returns paginated result
- `queryByID(ctx, *http.Request) web.Encoder` — parses UUID path param; calls `contextBus.QueryByID()` (404 if missing)
- `addEvent(ctx, *http.Request) web.Encoder` — validates kind + content required; calls `contextBus.AddEvent()`
- `queryEvents(ctx, *http.Request) web.Encoder` — parses `page`, `rows`; calls `contextBus.QueryEvents()` + `CountEvents()`; returns paginated result

**Filter parsing in `filter.go`:**
- `status` query param → `contextbus.Parse()` → `QueryFilter.Status`
- `title` query param → `QueryFilter.Title`

**Order parsing in `order.go`:**

| Query param `orderBy` value | Business constant |
|-----------------------------|--------------------|
| `id` | `contextbus.OrderByID` |
| `title` | `contextbus.OrderByTitle` |
| `status` | `contextbus.OrderByStatus` |
| `last_event` | `contextbus.OrderByLastEvent` |
| `created_at` | `contextbus.OrderByCreatedAt` |

Default: `OrderByLastEvent DESC`

### Business Layer

| File | Responsibility |
|------|---------------|
| `business/domain/contextbus/contextbus.go` | `Business` struct, `Storer` interface, all business methods |
| `business/domain/contextbus/model.go` | `Context`, `NewContext`, `UpdateContext`, `Event`, `NewEvent`, `Status` enum |
| `business/domain/contextbus/filter.go` | `QueryFilter` struct |
| `business/domain/contextbus/order.go` | Order constants + `DefaultOrderBy` |

**Business methods in `contextbus.go`:**
- `NewBusiness(log *logger.Logger, storer Storer) *Business`
- `Create(ctx, NewContext) (Context, error)` — generates UUID; sets Status=Active, CreatedAt=now, UpdatedAt=now; calls `storer.Create()`
- `Update(ctx, Context, UpdateContext) (Context, error)` — applies partial updates (Title, Description, Status, Summary); sets UpdatedAt=now; calls `storer.Update()`
- `Delete(ctx, Context) error` — calls `storer.Delete()`
- `Query(ctx, QueryFilter, order.By, page.Page) ([]Context, error)` — delegates to `storer.Query()`
- `Count(ctx, QueryFilter) (int, error)` — delegates to `storer.Count()`
- `QueryByID(ctx, uuid.UUID) (Context, error)` — delegates to `storer.QueryByID()`
- `AddEvent(ctx, NewEvent) (Event, error)` — generates UUID, sets CreatedAt=now; calls `storer.CreateEvent()`; then fetches context via `storer.QueryByID()`, sets `context.LastEvent=now`, `context.UpdatedAt=now`, calls `storer.Update()` (two-step, not atomic)
- `QueryEvents(ctx, uuid.UUID, page.Page) ([]Event, error)` — delegates to `storer.QueryEvents()`
- `CountEvents(ctx, uuid.UUID) (int, error)` — delegates to `storer.CountEvents()`

### Store Layer

| File | Responsibility |
|------|---------------|
| `business/domain/contextbus/stores/contextdb/contextdb.go` | `Store` struct + all SQL operations |
| `business/domain/contextbus/stores/contextdb/model.go` | `contextDB`, `eventDB` structs + converters |
| `business/domain/contextbus/stores/contextdb/filter.go` | `applyFilter()` — builds WHERE clauses |
| `business/domain/contextbus/stores/contextdb/order.go` | `orderByFields` map + `orderByClause()` |

**SQL operations in `contextdb.go`:**
- `NewStore(log *logger.Logger, db *sqlx.DB) *Store`
- `Create()` — INSERT INTO contexts (context_id, title, description, status, summary, last_event, created_at, updated_at)
- `Update()` — UPDATE contexts SET title, description, status, summary, last_event, updated_at WHERE context_id
- `Delete()` — DELETE FROM contexts WHERE context_id
- `Query()` — SELECT context_id, title, description, status, summary, last_event, created_at, updated_at FROM contexts WHERE 1=1 + dynamic filter + ORDER BY + OFFSET/FETCH pagination
- `Count()` — SELECT COUNT(*) FROM contexts WHERE 1=1 + dynamic filter
- `QueryByID()` — SELECT single context WHERE context_id; propagates `sqldb.ErrDBNotFound` on miss
- `CreateEvent()` — INSERT INTO context_events (event_id, context_id, kind, content, metadata, source_id, created_at)
- `QueryEvents()` — SELECT from context_events WHERE context_id ORDER BY created_at DESC with pagination
- `CountEvents()` — SELECT COUNT(*) FROM context_events WHERE context_id

**Filter clauses built by `applyFilter()`:**

| Filter field | SQL clause |
|-------------|-----------|
| `QueryFilter.ID` | `AND context_id = :id` |
| `QueryFilter.Status` | `AND status = :filter_status` (via `Status.String()`) |
| `QueryFilter.Title` | `AND title ILIKE :filter_title` (wrapped in `%...%`) |

**Order column mapping in `orderByFields`:**

| Business constant | SQL column |
|------------------|-----------|
| `OrderByID` | `context_id` |
| `OrderByTitle` | `title` |
| `OrderByStatus` | `status` |
| `OrderByLastEvent` | `last_event` |
| `OrderByCreatedAt` | `created_at` |

## Database Schema

### `contexts` table

```sql
CREATE TABLE contexts (
    context_id    UUID        NOT NULL DEFAULT gen_random_uuid(),
    title         TEXT        NOT NULL,
    description   TEXT        NOT NULL DEFAULT '',
    status        TEXT        NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active', 'paused', 'closed')),
    summary       TEXT        NOT NULL DEFAULT '',
    last_event    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (context_id)
);
```

### `context_events` table

```sql
CREATE TABLE context_events (
    event_id      UUID        NOT NULL DEFAULT gen_random_uuid(),
    context_id    UUID        NOT NULL REFERENCES contexts(context_id) ON DELETE CASCADE,
    kind          TEXT        NOT NULL,
    content       TEXT        NOT NULL,
    metadata      JSONB,
    source_id     UUID,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id)
);
CREATE INDEX idx_context_events_context ON context_events(context_id, created_at DESC);
```

### Thread/debrief columns (migration v1.11)

The following columns are wired through all three layers:

| Column | Type | Values |
|--------|------|--------|
| `last_thread_at` | `TIMESTAMPTZ` | nullable; system-managed (not in update DTO) |
| `debrief_status` | `TEXT` | `'pending'`, `'done'`, `'skipped'` (default: `'pending'`); uses `debriefstatus` enum type |
| `outcome` | `TEXT` | `'went_well'`, `'mixed'`, `'difficult'`, `'ongoing_issues'`; uses `contextoutcome` enum type |

When modifying these:
- `business/domain/contextbus/model.go` — `Context` has all three fields; `UpdateContext` has `DebriefStatus` and `Outcome` (not `LastThreadAt`)
- `business/domain/contextbus/stores/contextdb/model.go` — `contextDB` has db-tagged fields; converters handle enum ↔ string
- `business/domain/contextbus/stores/contextdb/contextdb.go` — all INSERT/UPDATE/SELECT SQL include the columns
- `app/domain/contextapp/model.go` — response DTO includes all three; update DTO includes `DebriefStatus` and `Outcome`

## Routes

All endpoints require the `X-API-Key` header (enforced by `mid.Auth` middleware).

| Method | Path | Handler | Body / Query Params | Notes |
|--------|------|---------|---------------------|-------|
| GET | `/api/v1/contexts` | `queryAll` | `page`, `rows`, `status`, `title`, `orderBy` | Returns paginated list with total count |
| GET | `/api/v1/contexts/{context_id}` | `queryByID` | — | 404 if not found |
| POST | `/api/v1/contexts` | `create` | `{title (required), description}` | Returns created context |
| PUT | `/api/v1/contexts/{context_id}` | `update` | `{title?, description?, status?, summary?}` | 404 if not found; validates status enum |
| DELETE | `/api/v1/contexts/{context_id}` | `delete` | — | 404 if not found; returns 204 No Content |
| POST | `/api/v1/contexts/{context_id}/events` | `addEvent` | `{kind (required), content (required), metadata?, sourceId?}` | Returns created event; also updates context `last_event` |
| GET | `/api/v1/contexts/{context_id}/events` | `queryEvents` | `page`, `rows` | Returns paginated list with total count; ordered by `created_at DESC` |

## Impact Callouts

### Context struct (`business/domain/contextbus/model.go`)

Adding or removing a field from `Context` or `UpdateContext` breaks:
- `business/domain/contextbus/stores/contextdb/model.go` — `toDBContext()`, `toBusContext()` must map new field
- `business/domain/contextbus/stores/contextdb/contextdb.go` — SELECT, INSERT, and UPDATE SQL column lists must be updated
- `app/domain/contextapp/model.go` — `toAppContext()` and `toBusUpdateContext()` must handle new field
- `business/sdk/migrate/sql/migrate.sql` — schema must add/remove column

### Status enum (`business/domain/contextbus/model.go`)

Adding or renaming a status value breaks:
- `Parse()` and `String()` methods in `model.go`
- `business/sdk/migrate/sql/migrate.sql` — `CHECK (status IN (...))` constraint
- `app/domain/contextapp/model.go` — `toBusUpdateContext()` calls `contextbus.Parse()`
- `business/domain/contextbus/stores/contextdb/filter.go` — uses `Status.String()` in filter clause
- HTTP clients relying on the status string value in responses

### Event struct (`business/domain/contextbus/model.go`)

Adding or removing a field from `Event` or `NewEvent` breaks:
- `business/domain/contextbus/stores/contextdb/model.go` — `toDBEvent()`, `toBusEvent()` must map new field
- `business/domain/contextbus/stores/contextdb/contextdb.go` — SELECT and INSERT column lists
- `app/domain/contextapp/model.go` — `toAppEvent()` and `toBusNewEvent()` must handle new field
- `business/sdk/migrate/sql/migrate.sql` — `context_events` table schema

### Storer interface (`business/domain/contextbus/contextbus.go`)

Adding or changing a method signature breaks:
- `business/domain/contextbus/stores/contextdb/contextdb.go` — must implement the complete interface (Go compiler enforces)
- Any mock or test implementation of the interface

### QueryFilter struct (`business/domain/contextbus/filter.go`)

Adding a filter field requires updating all three layers:
- `business/domain/contextbus/stores/contextdb/filter.go` — `applyFilter()` must add WHERE clause
- `app/domain/contextapp/filter.go` — `parseFilter()` must parse new query parameter

### Order constants (`business/domain/contextbus/order.go`)

Adding a new `OrderBy` constant requires:
- `business/domain/contextbus/stores/contextdb/order.go` — add entry in `orderByFields` map to SQL column
- `app/domain/contextapp/order.go` — add entry in `orderByFields` map from query param string to constant

### AddEvent two-step update

`Business.AddEvent()` performs three sequential store calls (CreateEvent → QueryByID → Update) with no transaction. If the context is deleted between CreateEvent and QueryByID, the operation returns an error but the event row is already committed. Callers in the handler layer do not pre-validate context existence before calling AddEvent.

## Cross-Domain Dependencies

| Dependency | Nature |
|-----------|--------|
| **tasks** domain | `tasks.context_id` FK references `contexts.context_id`; ON DELETE SET NULL — context deletion nullifies task FK |
| **tags** domain | `context_tags` junction table references `contexts.context_id`; ON DELETE CASCADE — context deletion removes tag associations |
| `foundation/logger` | Both business and store layers accept `*logger.Logger` for structured logging |
| `foundation/sqldb` | Store uses `NamedExecContext`, `NamedQuerySlice`, `NamedQueryStruct`; `sqldb.ErrDBNotFound` propagated from `QueryByID()` |
| `foundation/web` | Handlers use `web.Decode()`, `web.Param()`, `web.NoResponse{}`, and return `web.Encoder` |
| `business/sdk/order` | Business layer uses `order.By` (Field + Direction); store maps constants to SQL column names |
| `business/sdk/page` | Business layer uses `page.Page` for Offset + RowsPerPage; parsed in handlers via `page.Parse()` |
| `app/sdk/errs` | Handlers return `errs.New(errs.NotFound, ...)`, `errs.New(errs.InvalidArgument, ...)`, `errs.Newf(errs.Internal, ...)` |
| `app/sdk/mid` | Route registration applies `mid.Auth(cfg.APIKey)` to all seven context endpoints |
| `app/sdk/query` | `queryAll` and `queryEvents` wrap results with `query.NewResult()` for paginated response envelope |
