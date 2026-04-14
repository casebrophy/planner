# Context Backend System

Contexts organize work into hierarchical containers: **Projects** and **Areas**. Each context has a lifecycle (Active → Paused/Closed), tracks debrief metadata (Pending/Done/Skipped), and can nest via `parent_context_id`. Contexts maintain both historical event timestamps (`last_event`, legacy) and thread activity timestamps (`last_thread_at`). Default ordering is by `last_event DESC`, prioritizing recently active contexts.

## Core Types

### Business Domain (business/domain/contextbus/)

```go
type Context struct {
    ID              uuid.UUID                        // Unique identifier
    Title           string                           // Display name
    Description     string                           // Full description
    Kind            contextkind.Kind                 // "project" or "area"
    Status          Status                           // Active=0, Paused=1, Closed=2
    Summary         string                           // Debrief summary (written on close)
    LastEvent       *time.Time                       // Legacy event timestamp (no longer written)
    LastThreadAt    *time.Time                       // Latest thread entry timestamp
    DebriefStatus   debriefstatus.Status             // Pending, Done, or Skipped
    Outcome         *contextoutcome.Outcome          // went_well, mixed, difficult, ongoing_issues
    ParentContextID *uuid.UUID                       // Hierarchical nesting
    CreatedAt       time.Time                        // Creation timestamp
    UpdatedAt       time.Time                        // Last modification timestamp
}

type NewContext struct {
    Title           string
    Description     string
    Kind            contextkind.Kind                 // Defaults to Project if empty
    ParentContextID *uuid.UUID
}

type UpdateContext struct {
    Title           *string
    Description     *string
    Kind            *contextkind.Kind
    Status          *Status                          // Cannot Paused/Closed if Kind=Area
    Summary         *string
    DebriefStatus   *debriefstatus.Status
    Outcome         *contextoutcome.Outcome
    ParentContextID *uuid.UUID
}

type Status int                                       // Active, Paused, Closed
func (s Status) String() string                       // Returns "active", "paused", "closed"
func Parse(s string) (Status, error)                  // Parses string → Status
func MustParse(s string) Status                       // Parse without error return

type QueryFilter struct {
    ID              *uuid.UUID
    Status          *Status
    Kind            *contextkind.Kind
    Title           *string                          // ILIKE substring match
    ParentContextID *uuid.UUID
}

// Order constants and defaults
const (
    OrderByID        = "context_id"
    OrderByTitle     = "title"
    OrderByStatus    = "status"
    OrderByLastEvent = "last_event"
    OrderByCreatedAt = "created_at"
)
var DefaultOrderBy = order.NewBy(OrderByLastEvent, order.DESC)
```

### Enum Types (business/types/)

**contextkind.Kind** (`business/types/contextkind/`)
```go
var (
    Project = Kind{"project"}  // Default; can be closed/paused
    Area    = Kind{"area"}     // Organizational unit; always active (never closed/paused)
    List    = Kind{"list"}     // Checklist container; can close, cannot pause; parent must be Area
)
```

**contextoutcome.Outcome** (`business/types/contextoutcome/`)
```go
var (
    WentWell      = Outcome{"went_well"}
    Mixed         = Outcome{"mixed"}
    Difficult     = Outcome{"difficult"}
    OngoingIssues = Outcome{"ongoing_issues"}
)
```

**debriefstatus.Status** (`business/types/debriefstatus/`)
```go
var (
    Pending = Status{"pending"}  // Default on create; triggered on close
    Done    = Status{"done"}     // User completed debrief
    Skipped = Status{"skipped"}  // User skipped debrief
)
```

### Storer Interface (business/domain/contextbus/contextbus.go)

```go
type Storer interface {
    // Lifecycle
    Create(ctx context.Context, c Context) error
    Update(ctx context.Context, c Context) error
    Delete(ctx context.Context, c Context) error
    
    // Query
    Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Context, error)
    Count(ctx context.Context, filter QueryFilter) (int, error)
    QueryByID(ctx context.Context, id uuid.UUID) (Context, error)
}
```

### App Layer Types (app/domain/contextapp/)

```go
type Context struct {
    ID              string  `json:"id"`
    Title           string  `json:"title"`
    Description     string  `json:"description"`
    Status          string  `json:"status"`
    Kind            string  `json:"kind"`
    Summary         string  `json:"summary"`
    LastEvent       *string `json:"lastEvent,omitempty"`
    LastThreadAt    *string `json:"lastThreadAt,omitempty"`
    DebriefStatus   string  `json:"debriefStatus"`
    Outcome         *string `json:"outcome,omitempty"`
    ParentContextID *string `json:"parentContextId,omitempty"`
    CreatedAt       string  `json:"createdAt"`                 // RFC3339 format
    UpdatedAt       string  `json:"updatedAt"`                 // RFC3339 format
}

type NewContext struct {
    Title           string  `json:"title"`
    Description     string  `json:"description"`
    Kind            string  `json:"kind"`
    ParentContextID *string `json:"parentContextId,omitempty"`
}

type UpdateContext struct {
    Title           *string `json:"title"`
    Description     *string `json:"description"`
    Status          *string `json:"status"`
    Kind            *string `json:"kind"`
    Summary         *string `json:"summary"`
    DebriefStatus   *string `json:"debriefStatus"`
    Outcome         *string `json:"outcome"`
    ParentContextID *string `json:"parentContextId,omitempty"`
}
```

### Store Layer Models (business/domain/contextbus/stores/contextdb/)

```go
type contextDB struct {
    ID              uuid.UUID  `db:"context_id"`
    Title           string     `db:"title"`
    Description     string     `db:"description"`
    Kind            string     `db:"kind"`
    Status          string     `db:"status"`
    Summary         string     `db:"summary"`
    LastEvent       *time.Time `db:"last_event"`
    LastThreadAt    *time.Time `db:"last_thread_at"`
    DebriefStatus   string     `db:"debrief_status"`
    Outcome         *string    `db:"outcome"`
    ParentContextID *uuid.UUID `db:"parent_context_id"`
    CreatedAt       time.Time  `db:"created_at"`
    UpdatedAt       time.Time  `db:"updated_at"`
}
```

## File Map

### Models
- `business/domain/contextbus/model.go` — Context, NewContext, UpdateContext, Status (Active/Paused/Closed)
- `business/domain/contextbus/filter.go` — QueryFilter struct
- `business/domain/contextbus/order.go` — Order constants (OrderByID, OrderByTitle, etc.) and DefaultOrderBy
- `business/types/contextkind/contextkind.go` — Kind enum (Project, Area)
- `business/types/contextoutcome/contextoutcome.go` — Outcome enum (WentWell, Mixed, Difficult, OngoingIssues)
- `business/types/debriefstatus/debriefstatus.go` — Status enum (Pending, Done, Skipped)

### Business Layer
- `business/domain/contextbus/contextbus.go` — Business struct, Storer interface, Create/Update/Delete/Query/Count/QueryByID methods

### Store Layer
- `business/domain/contextbus/stores/contextdb/model.go` — contextDB struct (db-tagged), toDBContext/toBusContext/toBusContexts converters
- `business/domain/contextbus/stores/contextdb/contextdb.go` — Store struct, Create/Update/Delete/Query/Count/QueryByID implementations (SQL: INSERT/UPDATE/DELETE/SELECT)
- `business/domain/contextbus/stores/contextdb/filter.go` — applyFilter() builds WHERE clauses (ID, Status, Kind, Title ILIKE, ParentContextID)
- `business/domain/contextbus/stores/contextdb/order.go` — orderByFields map, orderByClause() builds ORDER BY clause

### App Layer
- `app/domain/contextapp/model.go` — Context/NewContext/UpdateContext DTOs, toAppContext/toAppContexts/toBusNewContext/toBusUpdateContext converters
- `app/domain/contextapp/contextapp.go` — app struct, create/update/delete/queryAll/queryByID handlers, triggerDebriefFlow() method
- `app/domain/contextapp/route.go` — Routes.Add() wires up handlers, instantiates business + dependencies
- `app/domain/contextapp/filter.go` — parseFilter() maps query params to QueryFilter
- `app/domain/contextapp/order.go` — parseOrder() maps request field names to order constants

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

### ⚠ New Filter Field (business/domain/contextbus/filter.go)

Changing affects:
1. `business/domain/contextbus/filter.go` — add field to QueryFilter struct
2. `business/domain/contextbus/stores/contextdb/filter.go` — add handling in applyFilter()
3. `app/domain/contextapp/filter.go` — add query param parsing in parseFilter()

### ⚠ New Order Field (business/domain/contextbus/order.go)

Changing affects:
1. `business/domain/contextbus/order.go` — add OrderBy constant
2. `business/domain/contextbus/stores/contextdb/order.go` — add to orderByFields map (SQL column name)
3. `app/domain/contextapp/order.go` — add to orderByFields map (request field name)

### ⚠ Kind Status Constraints (business/domain/contextbus/contextbus.go)

Status transitions are constrained by kind. Update() enforces these rules in order:
```go
// Area contexts cannot be closed or paused.
if c.Kind == contextkind.Area && (c.Status == Closed || c.Status == Paused) {
    return Context{}, errors.New("area contexts cannot be closed or paused")
}
// List contexts cannot be paused (but can be closed).
if c.Kind == contextkind.List && c.Status == Paused {
    return Context{}, errors.New("list contexts cannot be paused")
}
```
Also enforced: list contexts must have an area parent (validated via `validateListParent()` in both Create and Update).

Changing these constraints affects any UI that presents status options based on Kind.

### ⚠ Debrief Flow Trigger (app/domain/contextapp/contextapp.go, line 106)

Transitioning to Closed status triggers `triggerDebriefFlow()` — **except for list contexts**:
```go
if previousStatus != contextbus.Closed && updated.Status == contextbus.Closed {
    if updated.Kind != contextkind.List {
        a.triggerDebriefFlow(ctx, updated)
    }
    ...
}
```
- Sets DebriefStatus → Pending
- Creates 3 clarification cards (context_debrief type) snoozed 24h
- Depends on clarificationbus.Create() being wired into Routes
- List contexts skip this flow entirely (no debrief on list close)

Changes to debrief questions/structure live in `triggerDebriefFlow()`.

### ⚠ Task Cascade on Context Close (app/domain/contextapp/contextapp.go, line 111)

When a Project transitions to Closed:
- Calls `taskBus.DismissTasksByContext(ctx, updated.ID)`
- Only Projects cascade (defensive check `updated.Kind == contextkind.Project`)
- Depends on taskbus being wired into Routes

## Routes

| Method | Path | Handler | Authenticated |
|--------|------|---------|---|
| POST | /api/v1/contexts | create | Yes |
| GET | /api/v1/contexts | queryAll | Yes |
| GET | /api/v1/contexts/{context_id} | queryByID | Yes |
| PUT | /api/v1/contexts/{context_id} | update | Yes |
| DELETE | /api/v1/contexts/{context_id} | delete | Yes |

**create** — Validates title (required), defaults Kind → Project, Status → Active, DebriefStatus → Pending. Fires async thread entry on success.

**queryAll** — Paginated query with optional filters (status, kind, title, parent_context_id) and ordering. Returns query.Result with total count.

**queryByID** — Fetch single context; 404 if not found.

**update** — Fetch context, apply UpdateContext mutations. On status transition to Closed: triggers debrief flow, dismisses open/blocked tasks (Projects only), fires async thread entry with milestone kind.

**delete** — Soft-delete (DELETE FROM contexts); no cascade logic (tasks remain, threads remain).

## Cross-Domain Dependencies

### Inbound: Who uses Context?

- **taskbus** — reads contexts for hierarchy; `task.context_id` FK; dismissed by context close via DismissTasksByContext()
- **threadbus** — reads contexts; contexts add thread entries (AddEntry with SubjectType="context")
- **clarificationbus** — creates clarification cards on debrief trigger (Kind=ContextDebrief)
- **inactivitybus** — reads `contexts.last_event` (legacy) in COALESCE(last_event, last_thread_at, updated_at) for inactivity detection
- **ingestbus** — reads contexts for matching during ingestion; does NOT write context_events (table removed in migration 1.25)

### Outbound: What Context calls

- **threadbus.AddEntry()** — async goroutine on create (Update kind) and update (Update/Milestone kinds)
- **clarificationbus.Create()** — on debrief trigger (3 cards, snoozed 24h)
- **taskbus.DismissTasksByContext()** — on Project close, bulk-dismiss open/blocked tasks

### Database

- Table: `contexts` (created in migration 1.01)
- Columns: context_id (uuid), title, description, kind (text), status (text), summary, last_event (timestamp, legacy), last_thread_at (timestamp), debrief_status (text), outcome (text), parent_context_id (fk), created_at, updated_at
- Removed: `context_events` table (dropped in migration 1.25)
- CHECK constraints: kind in ('project', 'area', 'list'), status in ('active', 'paused', 'closed'), debrief_status in ('pending', 'done', 'skipped'), outcome in ('went_well', 'mixed', 'difficult', 'ongoing_issues')

### Default Ordering

Default is `order.NewBy(OrderByLastEvent, order.DESC)` — newest event activity first. Since LastEvent is legacy and no longer written, queries effectively fall back to LastThreadAt (via COALESCE in UI logic or inactivitybus). Frontend should handle NULL ordering gracefully.
