# Context Backend Architecture

The Context domain manages top-level groupings of work with two classifications: **Projects** (closable, trigger debrief and task cascade-dismiss on close) and **Areas** (perpetually active, reusable). Contexts have a lifecycle (Active → Paused → Closed), emit events for audit trails, and support debrief workflows.

## Core Types

### Business Layer Models (`business/domain/contextbus/model.go`)

```go
type Context struct {
    ID            uuid.UUID
    Title         string
    Description   string
    Kind          contextkind.Kind         // "project" or "area"
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
    Kind        contextkind.Kind          // Defaults to Project if zero-value
}

type UpdateContext struct {
    Title         *string                  // nil = do not update
    Description   *string
    Kind          *contextkind.Kind        // Can change project ↔ area (subject to validation)
    Status        *Status
    Summary       *string
    DebriefStatus *debriefstatus.Status
    Outcome       *contextoutcome.Outcome
}

type Status int

const (
    Active Status = iota  // "active"
    Paused                 // "paused"
    Closed                 // "closed"
)

// Implements: String(), Parse(s string), MustParse(s string)

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

### Kind Enum (`business/types/contextkind/contextkind.go`)

```go
type Kind struct {
    value string
}

var (
    Project = Kind{"project"}  // Closable, triggers debrief, cascades task dismiss
    Area    = Kind{"area"}     // Always active, reusable, cannot close/pause
)

// Implements: Parse(s string), MustParse(s string), String(), MarshalText(), UnmarshalText(), EqualString(v string)
```

**Validation rule** (enforced in `contextbus.Update()`):
- If Kind == Area, Status must remain Active. Attempting to set Status to Paused or Closed returns error.

### Query Filter (`business/domain/contextbus/filter.go`)

```go
type QueryFilter struct {
    ID     *uuid.UUID        // Exact match on context_id
    Status *Status           // Exact match on status
    Kind   *contextkind.Kind // Exact match on kind ("project" | "area")
    Title  *string           // Case-insensitive substring match (ILIKE)
}
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
    Kind          string     `db:"kind"`              // "project" or "area" (string stored in DB)
    Status        string     `db:"status"`            // "active", "paused", or "closed"
    Summary       string     `db:"summary"`
    LastEvent     *time.Time `db:"last_event"`
    LastThreadAt  *time.Time `db:"last_thread_at"`
    DebriefStatus string     `db:"debrief_status"`    // "pending", "done", "skipped"
    Outcome       *string    `db:"outcome"`           // nullable, enum string
    CreatedAt     time.Time  `db:"created_at"`
    UpdatedAt     time.Time  `db:"updated_at"`
}

type eventDB struct {
    ID        uuid.UUID        `db:"event_id"`
    ContextID uuid.UUID        `db:"context_id"`
    Kind      string           `db:"kind"`            // Event type (e.g., "status_change", "note")
    Content   string           `db:"content"`
    Metadata  *json.RawMessage `db:"metadata"`        // Optional structured data
    SourceID  *uuid.UUID       `db:"source_id"`       // Optional reference (FK to task, etc.)
    CreatedAt time.Time        `db:"created_at"`
}
```

**Conversion functions**: 
- `toDBContext()` — converts contextkind.Kind to string via `.String()`
- `toBusContext()` — converts string back to contextkind.Kind via `contextkind.MustParse()`
- `toBusContexts()`, `toDBEvent()`, `toBusEvent()`, `toBusEvents()`

### App Layer Models (`app/domain/contextapp/model.go`)

```go
type Context struct {
    ID            string  `json:"id"`
    Title         string  `json:"title"`
    Description   string  `json:"description"`
    Kind          string  `json:"kind"`                    // "project" | "area"
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
    Kind        string `json:"kind"`            // "project" | "area" (defaults to "project" in toBusNewContext)
}

type UpdateContext struct {
    Title         *string `json:"title"`
    Description   *string `json:"description"`
    Kind          *string `json:"kind"`         // Can change context kind (project ↔ area)
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

**Conversion functions**:
- `toAppContext()` — Context to HTTP DTO; converts Kind.String() and UUID to strings, timestamps to RFC3339
- `toBusNewContext()` — NewContext to business model; parses Kind string via `contextkind.Parse()`, defaults to Project if empty
- `toBusUpdateContext()` — UpdateContext to business patch; parses Kind/Status/DebriefStatus/Outcome strings
- `toAppEvent()`, `toAppEvents()`, `toBusNewEvent()`

## File Map

### App Layer

| File | Responsibility |
|------|---------------|
| `app/domain/contextapp/contextapp.go` | HTTP handlers + triggerDebriefFlow logic |
| `app/domain/contextapp/model.go` | HTTP DTOs + converters |
| `app/domain/contextapp/filter.go` | `parseFilter()` → QueryFilter (status, kind, title) |
| `app/domain/contextapp/route.go` | Routes.Add(), dependency injection (contextBus, clarificationBus, taskBus) |

**Handler methods in `contextapp.go`:**
- `create()` — validates title required; calls `contextBus.Create()` (defaults Kind to Project)
- `update()` — fetches context (404 if missing); applies patches; **if status transitions to Closed**, calls `triggerDebriefFlow()` and `taskBus.DismissTasksByContext()` (projects only)
- `delete()` — fetches context (404 if missing); calls `contextBus.Delete()` → 204 No Content
- `queryAll()` — parses page/rows/status/kind/title; calls Query + Count; returns paginated result
- `queryByID()` — fetches by UUID path param (404 if missing)
- `addEvent()` — validates kind + content required; calls `contextBus.AddEvent()`
- `queryEvents()` — parses page/rows; calls QueryEvents + CountEvents; returns paginated result

**Key: `triggerDebriefFlow(ctx, c)`** (lines 227–281):
- Fires when status transitions to Closed (any context)
- Sets `DebriefStatus` to Pending
- Creates 3 clarification items (Kind=ContextDebrief) snoozed 24h: outcome, challenge, lesson
- Errors are logged, do not fail the response

**Cascade dismiss** (lines 92–97 in `update()`):
- Fires when status transitions to Closed AND Kind == Project
- Calls `taskBus.DismissTasksByContext(ctx, contextID)` to mark open/blocked tasks as dismissed
- Errors are logged, do not fail the response

**Filter parsing in `filter.go`:**
- `status` query param → `contextbus.Parse()` → `QueryFilter.Status`
- `kind` query param → `contextkind.Parse()` → `QueryFilter.Kind` (NEW)
- `title` query param → `QueryFilter.Title`

### Business Layer

| File | Responsibility |
|------|---------------|
| `business/domain/contextbus/contextbus.go` | Business struct, Storer interface, all CRUD + event methods |
| `business/domain/contextbus/model.go` | Context, NewContext, UpdateContext, Event, NewEvent, Status enum |
| `business/domain/contextbus/filter.go` | QueryFilter struct (with Kind field) |
| `business/types/contextkind/contextkind.go` | Kind enum (Project, Area) + Parse/String |

**Business methods in `contextbus.go`:**
- `Create(ctx, NewContext) (Context, error)` — generates UUID; defaults Kind to Project if zero-value; sets Status=Active; calls `storer.Create()`
- `Update(ctx, Context, UpdateContext) (Context, error)` — applies patches; **validates**: if Kind==Area, Status must be Active (returns error if Paused/Closed); calls `storer.Update()`
- `Delete(ctx, Context) error` — calls `storer.Delete()`
- `Query()`, `Count()`, `QueryByID()` — delegate to storer
- `AddEvent(ctx, NewEvent) (Event, error)` — creates event, fetches context, updates `LastEvent` + `UpdatedAt` (two-step, not atomic)
- `QueryEvents()`, `CountEvents()` — delegate to storer

### Store Layer

| File | Responsibility |
|------|---------------|
| `business/domain/contextbus/stores/contextdb/contextdb.go` | Store struct + all SQL operations (CRUD) |
| `business/domain/contextbus/stores/contextdb/model.go` | contextDB, eventDB structs + converters |
| `business/domain/contextbus/stores/contextdb/filter.go` | applyFilter() — builds WHERE clauses |

**SQL operations in `contextdb.go`:**
- `Create()` — INSERT INTO contexts (context_id, title, description, **kind**, status, summary, last_event, last_thread_at, debrief_status, outcome, created_at, updated_at)
- `Update()` — UPDATE contexts SET title, description, **kind**, status, summary, last_event, last_thread_at, debrief_status, outcome, updated_at WHERE context_id
- `Delete()` — DELETE FROM contexts WHERE context_id
- `Query()` — SELECT all columns WHERE 1=1 + applyFilter + ORDER BY + OFFSET/FETCH
- `Count()` — SELECT COUNT(*) WHERE 1=1 + applyFilter
- `QueryByID()` — SELECT WHERE context_id (404 → sqldb.ErrDBNotFound)
- `CreateEvent()` — INSERT INTO context_events (event_id, context_id, kind, content, metadata, source_id, created_at)
- `QueryEvents()` — SELECT FROM context_events WHERE context_id ORDER BY created_at DESC with pagination
- `CountEvents()` — SELECT COUNT(*) FROM context_events WHERE context_id

**Filter clauses built by `applyFilter()`** (in `filter.go`):

| Filter field | SQL clause |
|-------------|-----------|
| `QueryFilter.ID` | `AND context_id = :id` |
| `QueryFilter.Status` | `AND status = :filter_status` (via Status.String()) |
| `QueryFilter.Kind` | `AND kind = :filter_kind` (via Kind.String()) |
| `QueryFilter.Title` | `AND title ILIKE :filter_title` (wrapped in %...%) |

Note: The `filter.go` file has a duplicate Kind filter clause (lines 18–20 and 26–28). This should be cleaned up in a refactor.

## Database Schema

### `contexts` table

```sql
CREATE TABLE contexts (
    context_id      UUID        NOT NULL DEFAULT gen_random_uuid(),
    title           TEXT        NOT NULL,
    description     TEXT        NOT NULL DEFAULT '',
    kind            TEXT        NOT NULL DEFAULT 'project'
                    CHECK (kind IN ('project', 'area')),
    status          TEXT        NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active', 'paused', 'closed')),
    summary         TEXT        NOT NULL DEFAULT '',
    last_event      TIMESTAMPTZ,
    last_thread_at  TIMESTAMPTZ,
    debrief_status  TEXT        NOT NULL DEFAULT 'pending'
                    CHECK (debrief_status IN ('pending', 'done', 'skipped')),
    outcome         TEXT
                    CHECK (outcome IS NULL OR outcome IN ('went_well', 'mixed', 'difficult', 'ongoing_issues')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
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

## Routes

All endpoints require the `X-API-Key` header (enforced by `mid.Auth` middleware).

| Method | Path | Handler | Body / Query Params | Notes |
|--------|------|---------|---------------------|-------|
| GET | `/api/v1/contexts` | `queryAll` | `page`, `rows`, `status`, `kind`, `title` | Filterable by status (active/paused/closed), **kind (project/area)**, title; paginated |
| GET | `/api/v1/contexts/{context_id}` | `queryByID` | — | 404 if not found |
| POST | `/api/v1/contexts` | `create` | `{title (req), description?, kind?}` | Creates context; kind defaults to "project" if omitted |
| PUT | `/api/v1/contexts/{context_id}` | `update` | `{title?, description?, status?, kind?, summary?, debriefStatus?, outcome?}` | 404 if missing; **if status→closed**, triggers debrief + cascade-dismiss (projects only) |
| DELETE | `/api/v1/contexts/{context_id}` | `delete` | — | 404 if not found; returns 204 No Content |
| POST | `/api/v1/contexts/{context_id}/events` | `addEvent` | `{kind (req), content (req), metadata?, sourceId?}` | Returns created event; also updates context `last_event` |
| GET | `/api/v1/contexts/{context_id}/events` | `queryEvents` | `page`, `rows` | Returns paginated list; ordered by `created_at DESC` |

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
