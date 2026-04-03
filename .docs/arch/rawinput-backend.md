# RawInput Backend System

> Raw inputs are the entry point for all user-provided content (voice, email). Each raw_input record stores the original text and tracks async processing state through a retry-aware pipeline. The background IngestWorker polls for pending items, dispatches them to the ingest pipeline, and applies exponential-backoff retry logic. Failed items can be manually reset via the reprocess endpoint.

## Core Types

### Business Model (`business/domain/rawinputbus/model.go`)

```go
type RawInput struct {
    ID          uuid.UUID
    SourceType  rawinputsource.Source
    Status      rawinputstatus.Status
    RawContent  string
    ProcessedAt *time.Time
    Error       *string
    RetryCount  int
    NextRetryAt *time.Time
    MaxRetries  int
    CreatedAt   time.Time
}

type NewRawInput struct {
    SourceType rawinputsource.Source
    RawContent string
}

type UpdateRawInput struct {
    Status      *rawinputstatus.Status
    ProcessedAt *time.Time
    Error       *string
    RetryCount  *int
    NextRetryAt *time.Time
}
```

### Storer Interface (`business/domain/rawinputbus/rawinputbus.go`)

```go
type Storer interface {
    Create(ctx context.Context, ri RawInput) error
    Update(ctx context.Context, ri RawInput) error
    Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]RawInput, error)
    Count(ctx context.Context, filter QueryFilter) (int, error)
    QueryByID(ctx context.Context, id uuid.UUID) (RawInput, error)
    QueryRetryable(ctx context.Context, limit int) ([]RawInput, error)
    ResetForReprocess(ctx context.Context, id uuid.UUID) (RawInput, error)
}
```

### Query Filter (`business/domain/rawinputbus/filter.go`)

```go
type QueryFilter struct {
    Status     *rawinputstatus.Status
    SourceType *rawinputsource.Source
}
```

### DB Struct (`business/domain/rawinputbus/stores/rawinputdb/model.go`)

```go
type rawInputDB struct {
    ID          uuid.UUID  `db:"raw_input_id"`
    SourceType  string     `db:"source_type"`
    Status      string     `db:"status"`
    RawContent  string     `db:"raw_content"`
    ProcessedAt *time.Time `db:"processed_at"`
    Error       *string    `db:"error"`
    RetryCount  int        `db:"retry_count"`
    NextRetryAt *time.Time `db:"next_retry_at"`
    MaxRetries  int        `db:"max_retries"`
    CreatedAt   time.Time  `db:"created_at"`
}
```

### App DTO (`app/domain/rawinputapp/model.go`)

```go
type RawInput struct {
    ID          string  `json:"id"`
    SourceType  string  `json:"sourceType"`
    Status      string  `json:"status"`
    RawContent  string  `json:"rawContent"`
    ProcessedAt *string `json:"processedAt,omitempty"`
    Error       *string `json:"error,omitempty"`
    RetryCount  int     `json:"retryCount"`
    NextRetryAt *string `json:"nextRetryAt,omitempty"`
    MaxRetries  int     `json:"maxRetries"`
    CreatedAt   string  `json:"createdAt"`
}
```

### Worker Interfaces (`business/sdk/worker/ingestworker.go`)

```go
type RawInputQueuer interface {
    QueryRetryable(ctx context.Context, limit int) ([]rawinputbus.RawInput, error)
    MarkForRetry(ctx context.Context, ri rawinputbus.RawInput, errMsg string) (rawinputbus.RawInput, error)
    MarkFailed(ctx context.Context, ri rawinputbus.RawInput, errMsg string) (rawinputbus.RawInput, error)
}

type RawInputProcessor interface {
    ProcessRawInputByID(ctx context.Context, id uuid.UUID) error
}
```

## File Map

### Business Layer

- `business/domain/rawinputbus/model.go` — `RawInput`, `NewRawInput`, `UpdateRawInput`
- `business/domain/rawinputbus/rawinputbus.go` — **NewBusiness()**, **Create()**, **Update()**, **MarkProcessing()**, **MarkProcessed()**, **MarkFailed()**, **MarkForRetry()**, **ComputeBackoff()**, **QueryRetryable()**, **ResetForReprocess()**, **RecoverStuck()**, **Query()**, **Count()**, **QueryByID()**; defines `Storer` interface
- `business/domain/rawinputbus/filter.go` — `QueryFilter` struct
- `business/domain/rawinputbus/order.go` — `OrderByCreatedAt`, `OrderByStatus`, `DefaultOrderBy`

### Store Layer

- `business/domain/rawinputbus/stores/rawinputdb/model.go` — `rawInputDB` struct, **toDBRawInput()**, **toBusRawInput()**, **toBusRawInputs()**
- `business/domain/rawinputbus/stores/rawinputdb/rawinputdb.go` — **NewStore()**, **Create()**, **Update()**, **Query()**, **Count()**, **QueryByID()**, **QueryRetryable()**, **ResetForReprocess()**
- `business/domain/rawinputbus/stores/rawinputdb/filter.go` — **applyFilter()** — builds `AND status = :filter_status` / `AND source_type = :filter_source_type`
- `business/domain/rawinputbus/stores/rawinputdb/order.go` — **orderByClause()** — maps `OrderByCreatedAt`→`created_at`, `OrderByStatus`→`status`

### App Layer

- `app/domain/rawinputapp/model.go` — `RawInput` DTO, **toAppRawInput()**, **toAppRawInputs()**, **Encode()**
- `app/domain/rawinputapp/rawinputapp.go` — **queryAll()**, **queryByID()**, **reprocess()**
- `app/domain/rawinputapp/route.go` — **Routes.Add()** — wires store → business → handlers, registers 3 routes with auth
- `app/domain/rawinputapp/filter.go` — **parseFilter()** — parses `?status=` and `?source_type=`
- `app/domain/rawinputapp/order.go` — **parseOrder()** — parses `?orderBy=created_at|status`

### Background Worker

- `business/sdk/worker/ingestworker.go` — **NewIngestWorker()**, **Run()**, **ProcessBatch()** — polls every 30s, dispatches items in goroutines with 3-min timeout; handles retry vs terminal failure

## Database Schema

```sql
CREATE TABLE raw_inputs (
    raw_input_id  UUID        NOT NULL DEFAULT gen_random_uuid(),
    source_type   TEXT        NOT NULL,
    status        TEXT        NOT NULL DEFAULT 'pending',  -- pending|processing|processed|failed
    raw_content   TEXT        NOT NULL,
    processed_at  TIMESTAMPTZ NULL,
    error         TEXT        NULL,
    retry_count   INT         NOT NULL DEFAULT 0,
    next_retry_at TIMESTAMPTZ NULL,     -- NULL = eligible immediately; future = waiting for backoff
    max_retries   INT         NOT NULL DEFAULT 5,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (raw_input_id)
);
CREATE INDEX idx_raw_inputs_status ON raw_inputs(status, created_at);
CREATE INDEX idx_raw_inputs_retryable ON raw_inputs(created_at)
    WHERE status = 'pending';
```

**Status state machine:**
- `pending` — waiting for processing (also the retry-waiting state with future `next_retry_at`)
- `processing` — claimed by worker goroutine
- `processed` — terminal success
- `failed` — terminal failure (retry_count >= max_retries)

## Impact Callouts

### ⚠ RawInput business model (`business/domain/rawinputbus/model.go`)

Changing this struct shape affects:
- `business/domain/rawinputbus/rawinputbus.go` — all method parameters and return types
- `business/domain/rawinputbus/stores/rawinputdb/model.go` — `toDBRawInput()` / `toBusRawInput()` field mapping
- `business/domain/rawinputbus/stores/rawinputdb/rawinputdb.go` — INSERT / UPDATE / SELECT column lists in SQL
- `app/domain/rawinputapp/model.go` — `toAppRawInput()` field mapping, app DTO struct
- `business/sdk/worker/ingestworker.go` — uses `RawInput.RetryCount` and `RawInput.MaxRetries` for retry logic
- Migration required if DB column added/removed

### ⚠ NewRawInput (`business/domain/rawinputbus/model.go`)

Changing this struct affects:
- `business/domain/rawinputbus/rawinputbus.go` — `Create()` converts `NewRawInput` → `RawInput` with defaults (MaxRetries=5)
- `business/domain/ingestbus/ingestbus.go` — `EnqueueEmail()` and `EnqueueText()` build `NewRawInput` to call `rawinputbus.Create()`

### ⚠ UpdateRawInput (`business/domain/rawinputbus/model.go`)

Changing this struct affects:
- `business/domain/rawinputbus/rawinputbus.go` — `Update()` applies partial fields using nil-check pattern; **does NOT support clearing NextRetryAt to NULL** — use `ResetForReprocess` instead
- `business/domain/rawinputbus/stores/rawinputdb/rawinputdb.go` — UPDATE SQL SET clause (excludes max_retries — immutable after create)

### ⚠ Storer interface (`business/domain/rawinputbus/rawinputbus.go`)

Adding/changing a method affects:
- `business/domain/rawinputbus/stores/rawinputdb/rawinputdb.go` — must implement the method
- `business/sdk/worker/ingestworker.go` — `RawInputQueuer` is a subset of Storer; if adding new retry methods, update the interface
- Any mock in tests (`app/domain/rawinputapp/tests/`, `business/domain/rawinputbus/stores/rawinputdb/*_test.go`)

### ⚠ RawInputQueuer / RawInputProcessor interfaces (`business/sdk/worker/ingestworker.go`)

These are worker-specific subsets of Storer. Changing them affects:
- `business/sdk/worker/ingestworker_test.go` — mock implementations of both interfaces
- `api/services/planner/main.go` — wires `riBus` (rawinputbus.Business) and `igBus` (ingestbus.Business) to IngestWorker

### ⚠ App DTO RawInput (`app/domain/rawinputapp/model.go`)

Changing JSON field names or adding/removing fields affects:
- Frontend TypeScript type `RawInput` in `api/services/frontend/web/src/types/rawinput.ts`
- Frontend store `rawinputStore.ts` and service `rawinputService.ts`
- Test fixtures in `app/domain/rawinputapp/tests/rawinputapi/query_test.go` — `toAppRawInputs()` must include all new fields

### ⚠ QueryFilter (`business/domain/rawinputbus/filter.go`)

Adding a new filter field cascades:
- `business/domain/rawinputbus/stores/rawinputdb/filter.go` — add `applyFilter` branch
- `app/domain/rawinputapp/filter.go` — add `parseFilter` query param parsing
- Frontend service `list()` method in `rawinputService.ts` — expose as optional param

### ⚠ RetryCount / NextRetryAt / MaxRetries fields

These fields power the retry state machine. Do not modify without updating:
- `business/domain/rawinputbus/rawinputbus.go` — `MarkForRetry()`, `ComputeBackoff()`, `ResetForReprocess()`
- `business/domain/rawinputbus/stores/rawinputdb/rawinputdb.go` — `QueryRetryable()` WHERE clause, `ResetForReprocess()` UPDATE
- `business/sdk/worker/ingestworker.go` — `ri.RetryCount + 1 >= ri.MaxRetries` terminal check
- Migration SQL if column semantics change

## Routes

| Method | Path | Handler | Auth |
|--------|------|---------|------|
| GET | `/api/v1/raw-inputs` | `queryAll` | X-API-Key |
| GET | `/api/v1/raw-inputs/{raw_input_id}` | `queryByID` | X-API-Key |
| POST | `/api/v1/raw-inputs/{raw_input_id}/reprocess` | `reprocess` | X-API-Key |

## Cross-Domain Dependencies

- **ingestbus** (`business/domain/ingestbus/ingestbus.go`) — `EnqueueEmail()`, `EnqueueText()` create raw_input records; `ProcessRawInputByID()` implements `RawInputProcessor` interface consumed by IngestWorker
- **voiceingestapp** (`app/domain/voiceingestapp/`) — HTTP handler that calls `ingestbus.EnqueueText()` and returns `rawInputId` immediately (async)
- **emailbus** (`business/domain/emailbus/`) — SMTP server calls `ingestbus.EnqueueEmail()` for incoming emails
- **emails table** — has FK `raw_input_id → raw_inputs(raw_input_id)`; email records link back to their originating raw_input
- **main.go** (`api/services/planner/main.go`) — instantiates `IngestWorker` and runs it as a goroutine on `jobCtx`; also wires `rawinputapp.Routes` into the mux

## Async Processing Flow

```
Voice/Email arrives
    ↓
voiceingestapp.ingest() or emailbus SMTP handler
    ↓
ingestbus.EnqueueText() / EnqueueEmail()
    ↓
rawinputbus.Create() → INSERT status=pending, max_retries=5
    ↓ (returns rawInputId immediately to caller)

--- IngestWorker goroutine (every 30s) ---
    ↓
rawinputbus.QueryRetryable(ctx, 20)
    WHERE status='pending' AND (next_retry_at IS NULL OR next_retry_at <= NOW())
    ↓ per item (goroutine, 3-min timeout)
ingestbus.ProcessRawInputByID(ctx, id)
    → rawinputbus.MarkProcessing()
    → processTextInput() / processRawInput()
    → rawinputbus.MarkProcessed()
    ↓
[ERROR + retryCount+1 < maxRetries] → MarkForRetry() (exponential backoff: 2^n min, cap 30min)
[ERROR + retryCount+1 >= maxRetries] → MarkFailed() (terminal)

--- Manual recovery ---
POST /api/v1/raw-inputs/{id}/reprocess
    ↓
rawinputbus.ResetForReprocess() → status=pending, retry_count=0, next_retry_at=NULL, error=NULL
```
