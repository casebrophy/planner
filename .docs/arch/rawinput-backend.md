# RawInput Backend System

The RawInput domain is the ingestion entry point for all external data entering the planner. It stores raw content from various sources (email, transaction, voice, file) in an unprocessed form, tracks pipeline processing status, and exposes read-only + reprocess endpoints. Write access (Create/Update) is internal — used by domain components such as the email ingester — not exposed via HTTP POST/PUT. The architecture follows the standard layered pattern: HTTP handlers → business logic core → database store.

## Core Types

### RawInput (Business Layer)
```go
type RawInput struct {
    ID          uuid.UUID
    SourceType  rawinputsource.Source
    Status      rawinputstatus.Status
    RawContent  string
    ProcessedAt *time.Time
    Error       *string
    CreatedAt   time.Time
}
```

### NewRawInput (Business Layer)
```go
type NewRawInput struct {
    SourceType rawinputsource.Source
    RawContent string
}
```

### UpdateRawInput (Business Layer)
```go
type UpdateRawInput struct {
    Status      *rawinputstatus.Status
    ProcessedAt *time.Time
    Error       *string
}
```

### RawInput (App Layer)
```go
type RawInput struct {
    ID          string  `json:"id"`
    SourceType  string  `json:"sourceType"`
    Status      string  `json:"status"`
    RawContent  string  `json:"rawContent"`
    ProcessedAt *string `json:"processedAt,omitempty"`
    Error       *string `json:"error,omitempty"`
    CreatedAt   string  `json:"createdAt"`
}
```

### rawInputDB (Store Layer)
```go
type rawInputDB struct {
    ID          uuid.UUID  `db:"raw_input_id"`
    SourceType  string     `db:"source_type"`
    Status      string     `db:"status"`
    RawContent  string     `db:"raw_content"`
    ProcessedAt *time.Time `db:"processed_at"`
    Error       *string    `db:"error"`
    CreatedAt   time.Time  `db:"created_at"`
}
```

### QueryFilter
```go
type QueryFilter struct {
    Status     *rawinputstatus.Status
    SourceType *rawinputsource.Source
}
```

### Storer Interface
```go
type Storer interface {
    Create(ctx context.Context, ri RawInput) error
    Update(ctx context.Context, ri RawInput) error
    Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]RawInput, error)
    Count(ctx context.Context, filter QueryFilter) (int, error)
    QueryByID(ctx context.Context, id uuid.UUID) (RawInput, error)
}
```

### Source Enum (`business/types/rawinputsource`)
```go
type Source struct {
    value string
}

var (
    Email       = Source{"email"}
    Transaction = Source{"transaction"}
    Voice       = Source{"voice"}
    File        = Source{"file"}
)

// Functions:
// Parse(s string) (Source, error)
// MustParse(s string) Source
// (s Source) String() string
// (s Source) MarshalText() ([]byte, error)
// (s *Source) UnmarshalText(data []byte) error
// (s Source) EqualString(v string) bool
```

### Status Enum (`business/types/rawinputstatus`)
```go
type Status struct {
    value string
}

var (
    Pending    = Status{"pending"}
    Processing = Status{"processing"}
    Processed  = Status{"processed"}
    Failed     = Status{"failed"}
)

// Functions:
// Parse(s string) (Status, error)
// MustParse(s string) Status
// (s Status) String() string
// (s Status) MarshalText() ([]byte, error)
// (s *Status) UnmarshalText(data []byte) error
// (s Status) EqualString(v string) bool
```

## File Map

### Type Definitions
- **`business/types/rawinputsource/rawinputsource.go`** — Source enum (email, transaction, voice, file) with Parse/MustParse and text marshaling
- **`business/types/rawinputstatus/rawinputstatus.go`** — Status enum (pending, processing, processed, failed) with Parse/MustParse and text marshaling

### App Layer (HTTP Handlers)
- **`app/domain/rawinputapp/model.go`** — HTTP DTO: RawInput with `Encode()` method and conversion functions `toAppRawInput()`, `toAppRawInputs()`; no inbound DTOs (no HTTP create/update)
- **`app/domain/rawinputapp/rawinputapp.go`** — Handler methods:
  - **queryAll()** — GET /api/v1/raw-inputs, supports pagination, filtering (status, source_type), sorting; calls `rawInputBus.Query` then `rawInputBus.Count`
  - **queryByID()** — GET /api/v1/raw-inputs/{raw_input_id}, fetches single record by UUID; returns 404 on `sqldb.ErrDBNotFound`
  - **reprocess()** — POST /api/v1/raw-inputs/{raw_input_id}/reprocess, fetches record by UUID then calls `rawInputBus.MarkProcessing` to reset status to `processing`; returns updated record
- **`app/domain/rawinputapp/route.go`** — **Routes.Add()** — registers three endpoints with Auth middleware, instantiates `rawinputdb.Store` and `rawinputbus.Business`
- **`app/domain/rawinputapp/filter.go`** — **parseFilter()** — parses query parameters (`status`, `source_type`) into `rawinputbus.QueryFilter`
- **`app/domain/rawinputapp/order.go`** — **parseOrder()** — maps request orderBy field names (`created_at`, `status`) to `rawinputbus` constants; default: `created_at DESC`

### Business Layer (Core Logic)
- **`business/domain/rawinputbus/model.go`** — Business models: RawInput, NewRawInput, UpdateRawInput
- **`business/domain/rawinputbus/rawinputbus.go`** — Business struct, Storer interface, and methods:
  - **Create()** — generates UUID, sets Status=Pending, sets CreatedAt, calls storer.Create; called by internal ingesters only
  - **Update()** — applies partial updates (Status, ProcessedAt, Error), calls storer.Update
  - **MarkProcessing()** — convenience wrapper: sets Status=Processing via Update
  - **MarkProcessed()** — convenience wrapper: sets Status=Processed + ProcessedAt=now via Update
  - **MarkFailed()** — convenience wrapper: sets Status=Failed + Error message via Update
  - **Query()** — delegates to storer with filter/order/pagination
  - **Count()** — delegates to storer to count filtered records
  - **QueryByID()** — delegates to storer to fetch by UUID
- **`business/domain/rawinputbus/filter.go`** — QueryFilter struct: optional Status and SourceType filters
- **`business/domain/rawinputbus/order.go`** — Order field constants (OrderByCreatedAt = `"created_at"`, OrderByStatus = `"status"`) and DefaultOrderBy (`created_at DESC`)

### Store Layer (Database)
- **`business/domain/rawinputbus/stores/rawinputdb/model.go`** — `rawInputDB` internal struct (all db tags), conversion functions:
  - **toDBRawInput()** — business RawInput → rawInputDB (converts enum types to strings via `.String()`)
  - **toBusRawInput()** — rawInputDB → business RawInput (parses string enums via `MustParse`)
  - **toBusRawInputs()** — slice converter
- **`business/domain/rawinputbus/stores/rawinputdb/rawinputdb.go`** — Store struct and methods:
  - **NewStore()** — constructor taking logger and `*sqlx.DB`
  - **Create()** — INSERT all seven columns via named query
  - **Update()** — UPDATE status, processed_at, error WHERE raw_input_id via named query; source_type and raw_content are immutable after creation
  - **Query()** — SELECT with WHERE 1=1 base, applies filter, ORDER BY clause, OFFSET/FETCH pagination
  - **Count()** — SELECT COUNT(*) with same filter applied
  - **QueryByID()** — SELECT WHERE raw_input_id by UUID; returns `sqldb.ErrDBNotFound` (= `sql.ErrNoRows`) when not found
- **`business/domain/rawinputbus/stores/rawinputdb/filter.go`** — **applyFilter()** — appends AND clauses for Status (`status = :filter_status`) and SourceType (`source_type = :filter_source_type`)
- **`business/domain/rawinputbus/stores/rawinputdb/order.go`** — orderByFields map (`rawinputbus.OrderByCreatedAt` → `"created_at"`, `rawinputbus.OrderByStatus` → `"status"`), **orderByClause()** — validates field and returns `"column direction"` string

## Database Schema

```sql
-- Version: 1.05
CREATE TABLE raw_inputs (
    raw_input_id  UUID        NOT NULL DEFAULT gen_random_uuid(),
    source_type   TEXT        NOT NULL CHECK (source_type IN ('email', 'transaction', 'voice', 'file')),
    status        TEXT        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'processed', 'failed')),
    raw_content   TEXT        NOT NULL,
    processed_at  TIMESTAMPTZ,
    error         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (raw_input_id)
);

CREATE INDEX idx_raw_inputs_status ON raw_inputs(status, created_at);
```

Note: `source_type` and `raw_content` are set at creation and never updated. The UPDATE statement in the store only writes `status`, `processed_at`, and `error`.

## Impact Callouts

### RawInput struct (business/domain/rawinputbus/model.go)
Adding or renaming a field on `RawInput` affects:
- `business/domain/rawinputbus/stores/rawinputdb/model.go` — `toDBRawInput()` and `toBusRawInput()` converters must be updated
- `business/domain/rawinputbus/stores/rawinputdb/rawinputdb.go` — INSERT and SELECT column lists must match; UPDATE only touches mutable columns
- `app/domain/rawinputapp/model.go` — App `RawInput` DTO and `toAppRawInput()` must be updated
- Database schema (migrate.sql) — must add/remove columns; UPDATE query in store is a subset of columns, so verify immutable columns are excluded

### NewRawInput / UpdateRawInput structs (business/domain/rawinputbus/model.go)
- `NewRawInput` changes affect `Business.Create()` — field assignments in `rawinputbus.go`
- `UpdateRawInput` changes affect `Business.Update()`, `MarkProcessing()`, `MarkProcessed()`, `MarkFailed()` — all convenience wrappers construct `UpdateRawInput` inline

### Storer interface (business/domain/rawinputbus/rawinputbus.go)
Adding or changing a method affects:
- `business/domain/rawinputbus/stores/rawinputdb/rawinputdb.go` — Store must implement the new method
- Any internal caller of `rawinputbus.Business` (e.g., ingest pipeline, email ingester) that relies on the delegated method

### QueryFilter struct (business/domain/rawinputbus/filter.go)
Adding a filter field affects:
- `business/domain/rawinputbus/stores/rawinputdb/filter.go` — `applyFilter()` must add a new AND clause
- `app/domain/rawinputapp/filter.go` — `parseFilter()` must parse the new query parameter and set the filter field

### Order constants (business/domain/rawinputbus/order.go)
Adding a new OrderBy constant affects:
- `business/domain/rawinputbus/stores/rawinputdb/order.go` — `orderByFields` map must include the new constant → SQL column name mapping
- `app/domain/rawinputapp/order.go` — `orderByFields` map must include the new request field name → business constant mapping

### Source / Status enums (business/types/)
Adding or renaming an enum value affects:
- Database CHECK constraints in migrate.sql — must add the new value to the `IN (...)` list
- `toBusRawInput()` in store/model.go — calls `MustParse` (panics on unrecognized string); DB must only ever contain valid values
- `parseFilter()` in app layer — calls `rawinputstatus.Parse` / `rawinputsource.Parse`; clients passing invalid values get a 400

### reprocess endpoint behavior
`reprocess` does not reset a record to `pending`; it sets status to `processing`. Any pipeline consumer watching for `pending` records will not pick up a reprocessed record unless the ingest pipeline also polls `processing`. If the intent changes to reset to `pending`, update `app/domain/rawinputapp/rawinputapp.go` to call `MarkPending` (which would need to be added to `rawinputbus.go`).

## Routes

| Method | Path | Handler | Notes |
|--------|------|---------|-------|
| GET | /api/v1/raw-inputs | queryAll | Query params: `page`, `rows`, `status`, `source_type`, `orderBy` (created_at, status); default order: created_at DESC |
| GET | /api/v1/raw-inputs/{raw_input_id} | queryByID | Fetches single record by UUID; 404 if not found |
| POST | /api/v1/raw-inputs/{raw_input_id}/reprocess | reprocess | Sets status=processing on the given record; 404 if not found |

All endpoints require Auth middleware (API key via `X-API-Key` header). There is no HTTP endpoint for creating or updating raw inputs — those operations are internal only.

## Cross-Domain Dependencies

- **Email Domain** — the `emails` table has a `raw_input_id` foreign key referencing `raw_inputs(raw_input_id)`; deleting a raw input without cascading would orphan email records. The email ingester calls `rawinputbus.Business.Create()` and then `MarkProcessed()` / `MarkFailed()` during pipeline execution.
- **Ingest Pipeline** (`business/domain/ingestbus/`) — consumes `rawinputbus.Business` to create records and update pipeline status; relies on all three status-transition convenience methods (`MarkProcessing`, `MarkProcessed`, `MarkFailed`)
- **Page SDK** (`business/sdk/page`) — queryAll uses Page struct for pagination (Offset, RowsPerPage, Number)
- **Order SDK** (`business/sdk/order`) — Query uses `order.By` struct with Field constant and Direction (ASC/DESC)
- **sqldb utilities** (`foundation/sqldb`) — Store uses `NamedExecContext`, `NamedQuerySlice`, `NamedQueryStruct` helpers; `ErrDBNotFound` is returned by `QueryByID` on miss and must be checked in handlers
- **Error handling** (`app/sdk/errs`) — Handlers return `errs.NotFound` (404) on `sqldb.ErrDBNotFound`, `errs.InvalidArgument` (400) on bad UUID or invalid enum, `errs.Internal` (500) on unexpected store errors
- **HTTP web framework** (`foundation/web`) — Handlers implement `web.HandlerFunc` and return `web.Encoder` for responses
