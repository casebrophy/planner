# Thread Backend System

The Thread domain provides an append-only log of timestamped entries attached to any subject in the system. It uses a unified polymorphic model keyed on `subject_type` + `subject_id`, allowing both tasks and contexts to share a single `thread_entries` table without separate junction tables. Threads are write-once (no update or delete operations). The architecture follows the standard layered pattern: HTTP handlers → business logic core → database store.

## Core Types

### ThreadEntry (Business Layer)
```go
type ThreadEntry struct {
	ID             uuid.UUID
	SubjectType    string
	SubjectID      uuid.UUID
	Kind           threadentrykind.Kind
	Content        string
	Metadata       *json.RawMessage
	Source         threadsource.Source
	SourceID       *uuid.UUID
	Sentiment      *string
	RequiresAction bool
	CreatedAt      time.Time
}
```

### NewThreadEntry (Business Layer)
```go
type NewThreadEntry struct {
	SubjectType    string
	SubjectID      uuid.UUID
	Kind           threadentrykind.Kind
	Content        string
	Metadata       *json.RawMessage
	Source         threadsource.Source
	SourceID       *uuid.UUID
	Sentiment      *string
	RequiresAction bool
}
```

### ThreadEntry (App Layer)
```go
type ThreadEntry struct {
	ID             string          `json:"id"`
	SubjectType    string          `json:"subjectType"`
	SubjectID      string          `json:"subjectId"`
	Kind           string          `json:"kind"`
	Content        string          `json:"content"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	Source         string          `json:"source"`
	SourceID       *string         `json:"sourceId,omitempty"`
	Sentiment      *string         `json:"sentiment,omitempty"`
	RequiresAction bool            `json:"requiresAction"`
	CreatedAt      string          `json:"createdAt"`
}
```

### NewThreadEntry (App Layer)
```go
type NewThreadEntry struct {
	SubjectType    string          `json:"subjectType"`
	SubjectID      string          `json:"subjectId"`
	Kind           string          `json:"kind"`
	Content        string          `json:"content"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	Source         string          `json:"source"`
	SourceID       *string         `json:"sourceId"`
	Sentiment      *string         `json:"sentiment"`
	RequiresAction bool            `json:"requiresAction"`
}
```

### threadEntryDB (Store Layer)
```go
type threadEntryDB struct {
	ID             uuid.UUID        `db:"entry_id"`
	SubjectType    string           `db:"subject_type"`
	SubjectID      uuid.UUID        `db:"subject_id"`
	Kind           string           `db:"kind"`
	Content        string           `db:"content"`
	Metadata       *json.RawMessage `db:"metadata"`
	Source         string           `db:"source"`
	SourceID       *uuid.UUID       `db:"source_id"`
	Sentiment      *string          `db:"sentiment"`
	RequiresAction bool             `db:"requires_action"`
	CreatedAt      time.Time        `db:"created_at"`
}
```

### QueryFilter
```go
type QueryFilter struct {
	SubjectType    *string
	SubjectID      *uuid.UUID
	Kind           *threadentrykind.Kind
	RequiresAction *bool
}
```

### Storer Interface
```go
type Storer interface {
	Create(ctx context.Context, entry ThreadEntry) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]ThreadEntry, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (ThreadEntry, error)
}
```

### threadentrykind.Kind Enum
```go
type Kind struct {
	value string
}

var (
	Update          = Kind{"update"}
	Blocker         = Kind{"blocker"}
	BlockerResolved = Kind{"blocker_resolved"}
	Milestone       = Kind{"milestone"}
	ScopeChange     = Kind{"scope_change"}
	TimelineSlip    = Kind{"timeline_slip"}
	ExternalDep     = Kind{"external_dep"}
	Decision        = Kind{"decision"}
	Observation     = Kind{"observation"}
	Email           = Kind{"email"}
	Transaction     = Kind{"transaction"}
)

// Functions:
// Parse(s string) (Kind, error)
// MustParse(s string) Kind
// (k Kind) String() string
// (k Kind) MarshalText() ([]byte, error)
// (k *Kind) UnmarshalText(data []byte) error
// (k Kind) EqualString(v string) bool
```

### threadsource.Source Enum
```go
type Source struct {
	value string
}

var (
	User              = Source{"user"}
	Voice             = Source{"voice"}
	EmailSource       = Source{"email"}
	TransactionSource = Source{"transaction"}
	System            = Source{"system"}
	Claude            = Source{"claude"}
)

// Functions:
// Parse(s string) (Source, error)
// MustParse(s string) Source
// (s Source) String() string
// (s Source) MarshalText() ([]byte, error)
// (s *Source) UnmarshalText(data []byte) error
// (s Source) EqualString(v string) bool
```

## File Map

### Type Definitions
- **`business/types/threadentrykind/threadentrykind.go`** — Kind enum (update, blocker, blocker_resolved, milestone, scope_change, timeline_slip, external_dep, decision, observation, email, transaction) with Parse/MustParse and text marshaling
- **`business/types/threadsource/threadsource.go`** — Source enum (user, voice, email, transaction, system, claude) with Parse/MustParse and text marshaling

### App Layer (HTTP Handlers)
- **`app/domain/threadapp/model.go`** — HTTP DTOs: ThreadEntry, NewThreadEntry with conversion functions:
  - **toAppThreadEntry()** — business ThreadEntry → app ThreadEntry (formats UUIDs as strings, formats CreatedAt as RFC3339, dereferences Metadata pointer)
  - **toAppThreadEntries()** — slice converter
  - **toBusNewThreadEntry()** — app NewThreadEntry → business NewThreadEntry (parses SubjectID UUID, parses Kind enum, parses Source enum defaulting to "user", parses optional SourceID UUID)
- **`app/domain/threadapp/threadapp.go`** — Handler methods:
  - **addEntry()** — POST /api/v1/threads, validates subjectType/subjectId/kind/content required, converts to business layer, creates entry
  - **queryThread()** — GET /api/v1/threads/{subject_type}/{subject_id}, parses path params, paginates, calls QueryBySubject + CountBySubject, returns paginated result
- **`app/domain/threadapp/route.go`** — **Routes.Add()** — registers two endpoints with Auth middleware, instantiates threaddb.Store and threadbus.Business

### Business Layer (Core Logic)
- **`business/domain/threadbus/model.go`** — Business models: ThreadEntry, NewThreadEntry
- **`business/domain/threadbus/threadbus.go`** — Business struct and methods:
  - **NewBusiness()** — constructor taking logger and Storer
  - **AddEntry()** — generates UUID, stamps CreatedAt, assembles ThreadEntry, calls storer.Create
  - **QueryBySubject()** — builds QueryFilter with SubjectType+SubjectID, delegates to storer.Query with DefaultOrderBy
  - **CountBySubject()** — builds QueryFilter with SubjectType+SubjectID, delegates to storer.Count
  - **QueryByID()** — delegates to storer.QueryByID by UUID
  - **Query()** — general-purpose query with caller-supplied filter/orderBy/page (used by other domains, not directly exposed via HTTP)
  - **Count()** — general-purpose count with caller-supplied filter (used by other domains, not directly exposed via HTTP)
- **`business/domain/threadbus/filter.go`** — QueryFilter struct for filtering by SubjectType, SubjectID, Kind, RequiresAction
- **`business/domain/threadbus/order.go`** — Order field constant (OrderByCreatedAt = "created_at") and DefaultOrderBy (created_at DESC)
- **`business/domain/threadbus/doc.go`** — Package doc: "manages append-only thread entries for tasks and contexts"

### Store Layer (Database)
- **`business/domain/threadbus/stores/threaddb/model.go`** — threadEntryDB internal struct (all db tags), conversion functions:
  - **toDBThreadEntry()** — business ThreadEntry → threadEntryDB (converts Kind and Source enums to strings)
  - **toBusThreadEntry()** — threadEntryDB → business ThreadEntry (MustParses Kind and Source strings back to enums — panics on unknown value)
  - **toBusThreadEntries()** — slice converter
- **`business/domain/threadbus/stores/threaddb/threaddb.go`** — Store struct and methods:
  - **NewStore()** — constructor taking logger and sqlx.DB
  - **Create()** — INSERT all fields into thread_entries via named query
  - **Query()** — SELECT with WHERE 1=1 base, applies filter, ORDER BY, OFFSET/FETCH pagination
  - **Count()** — SELECT COUNT(*) with filter applied
  - **QueryByID()** — SELECT WHERE entry_id by UUID
- **`business/domain/threadbus/stores/threaddb/filter.go`** — **applyFilter()** — builds WHERE clauses for SubjectType, SubjectID, Kind, RequiresAction
- **`business/domain/threadbus/stores/threaddb/order.go`** — orderByFields mapping (OrderByCreatedAt → "created_at"), **orderByClause()** — validates and formats ORDER BY clause

## Database Schema

```sql
-- Version: 1.08
CREATE TABLE thread_entries (
    entry_id         UUID        NOT NULL DEFAULT gen_random_uuid(),
    subject_type     TEXT        NOT NULL CHECK (subject_type IN ('task', 'context')),
    subject_id       UUID        NOT NULL,
    kind             TEXT        NOT NULL CHECK (kind IN (
        'update', 'blocker', 'blocker_resolved', 'milestone',
        'scope_change', 'timeline_slip', 'external_dep',
        'decision', 'observation', 'email', 'transaction'
    )),
    content          TEXT        NOT NULL,
    metadata         JSONB,
    source           TEXT        NOT NULL DEFAULT 'user' CHECK (source IN ('user', 'voice', 'email', 'transaction', 'system', 'claude')),
    source_id        UUID,
    sentiment        TEXT        CHECK (sentiment IN ('positive', 'neutral', 'negative', 'mixed')),
    requires_action  BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (entry_id)
);
CREATE INDEX idx_thread_subject ON thread_entries(subject_type, subject_id, created_at DESC);
CREATE INDEX idx_thread_action ON thread_entries(requires_action) WHERE requires_action = TRUE;
```

## Impact Callouts

### ThreadEntry struct (business/domain/threadbus/model.go)
Adding or changing a field on ThreadEntry affects:
- `business/domain/threadbus/stores/threaddb/model.go` — toDBThreadEntry() and toBusThreadEntry() must be updated to map the field
- `app/domain/threadapp/model.go` — app ThreadEntry struct and toAppThreadEntry() must surface the field in JSON
- `business/domain/threadbus/stores/threaddb/threaddb.go` — SQL SELECT column list in Query() and QueryByID() must include the new column; INSERT in Create() must include the new bind param
- Database schema (migrate.sql) — must add the column with appropriate type and constraints

### NewThreadEntry struct (business/domain/threadbus/model.go)
Adding a field on NewThreadEntry affects:
- `app/domain/threadapp/model.go` — app NewThreadEntry and toBusNewThreadEntry() must parse and map the new field
- `business/domain/threadbus/threadbus.go` — AddEntry() must copy the new field into the assembled ThreadEntry

### Storer interface (business/domain/threadbus/threadbus.go)
Adding or changing a Storer method affects:
- `business/domain/threadbus/stores/threaddb/threaddb.go` — Store must implement the new method signature
- Any code constructing a mock or alternative Storer implementation

### QueryFilter struct (business/domain/threadbus/filter.go)
Adding a filter field affects:
- `business/domain/threadbus/stores/threaddb/filter.go` — applyFilter() must handle the new field with an appropriate WHERE clause and data binding
- Any caller building a QueryFilter manually (currently threadbus.QueryBySubject and threadbus.CountBySubject)
- No HTTP filter parsing exists today (queryThread uses only path params); adding HTTP-driven filtering would also require changes in threadapp

### Order constants (business/domain/threadbus/order.go)
Adding a new OrderBy constant affects:
- `business/domain/threadbus/stores/threaddb/order.go` — orderByFields map must include the new constant → SQL column name mapping
- Any app-layer order parsing if a parseOrder() function is added to threadapp in the future

### threadentrykind / threadsource enums (business/types/)
Adding or removing enum values affects:
- Database CHECK constraints in migrate.sql (kind and source columns)
- `business/domain/threadbus/stores/threaddb/model.go` — toBusThreadEntry() calls MustParse; an unknown DB value will panic at runtime
- toBusNewThreadEntry() in app layer calls Parse and returns InvalidArgument; callers of the API will see 400 for unrecognised values

### subject_type constraint (database only)
The DB CHECK constraint limits subject_type to ('task', 'context'). To attach threads to a new domain:
- The CHECK constraint in migrate.sql must be altered or replaced
- No Go code needs to change — SubjectType is passed through as a plain string

## Routes

| Method | Path | Handler | Notes |
|--------|------|---------|-------|
| POST | /api/v1/threads | addEntry | Body: NewThreadEntry; validates subjectType, subjectId, kind, content required; source defaults to "user" |
| GET | /api/v1/threads/{subject_type}/{subject_id} | queryThread | Path params parsed as string + UUID; supports `page` and `rows` query params for pagination; returns paginated result ordered by created_at DESC |

All endpoints require Auth middleware (X-API-Key header validation).

## Cross-Domain Dependencies

- **Task Domain** — Tasks are a valid subject_type ("task"); subject_id references task UUIDs, but there is no FK constraint enforcing this at the DB level
- **Context Domain** — Contexts are a valid subject_type ("context"); subject_id references context UUIDs, likewise with no FK constraint
- **Email Domain** — Ingest pipeline is expected to create thread entries with kind="email" and source="email"; source_id can reference the originating email record UUID
- **Page SDK** (`business/sdk/page`) — queryThread uses Page for pagination (Offset, RowsPerPage, Number)
- **Order SDK** (`business/sdk/order`) — Query uses order.By with Field constant and Direction; only OrderByCreatedAt is supported
- **sqldb utilities** (`foundation/sqldb`) — Store uses NamedExecContext, NamedQuerySlice, NamedQueryStruct helpers
- **Error handling** (`app/sdk/errs`) — Handlers return InvalidArgument for bad input, NotFound when ErrDBNotFound is returned, Internal for unexpected errors
- **HTTP web framework** (`foundation/web`) — Handlers implement web.Encoder pattern; app ThreadEntry implements Encode() directly
- **query SDK** (`app/sdk/query`) — queryThread returns query.NewResult (paginated envelope with total count)
