# RawInput Backend System

> Raw inputs are the entry point for all user-provided content (voice, email). Each raw_input record stores the original text and tracks async processing state through a retry-aware pipeline. The background IngestWorker polls for pending items, dispatches them to the ingest pipeline, and applies exponential-backoff retry logic. Failed items can be manually reset via the reprocess endpoint.

## Core Types

### Business Layer

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
	Result      json.RawMessage  // structured extraction result on success
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
	Result      *json.RawMessage
}

type QueryFilter struct {
	Status     *rawinputstatus.Status
	SourceType *rawinputsource.Source
}

const (
	OrderByCreatedAt = "created_at"
	OrderByStatus    = "status"
)

var DefaultOrderBy = order.NewBy(OrderByCreatedAt, order.DESC)
```

### Enum Types

**rawinputstatus.Status**: `Pending`, `Processing`, `Processed`, `Failed`

**rawinputsource.Source**: `Email`, `Transaction`, `Voice`, `File`

### Storer Interface

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

### App Layer DTO

```go
type RawInput struct {
	ID          string          `json:"id"`
	SourceType  string          `json:"sourceType"`
	Status      string          `json:"status"`
	RawContent  string          `json:"rawContent"`
	ProcessedAt *string         `json:"processedAt,omitempty"`
	Error       *string         `json:"error,omitempty"`
	RetryCount  int             `json:"retryCount"`
	NextRetryAt *string         `json:"nextRetryAt,omitempty"`
	MaxRetries  int             `json:"maxRetries"`
	Result      json.RawMessage `json:"result,omitempty"`
	CreatedAt   string          `json:"createdAt"`
}
```

### Store Layer

```go
type rawInputDB struct {
	ID          uuid.UUID        `db:"raw_input_id"`
	SourceType  string           `db:"source_type"`
	Status      string           `db:"status"`
	RawContent  string           `db:"raw_content"`
	ProcessedAt *time.Time       `db:"processed_at"`
	Error       *string          `db:"error"`
	RetryCount  int              `db:"retry_count"`
	NextRetryAt *time.Time       `db:"next_retry_at"`
	MaxRetries  int              `db:"max_retries"`
	Result      *json.RawMessage `db:"result"`
	CreatedAt   time.Time        `db:"created_at"`
}
```

## File Map

### App Layer (app/domain/rawinputapp/)
- `rawinputapp.go` — **queryAll()**, **queryByID()**, **reprocess()** handler methods
- `model.go` — RawInput DTO + **toAppRawInput()**, **toAppRawInputs()** converters
- `route.go` — **Routes.Add()** wires Store → Business → Handlers with auth; registers 3 routes
- `filter.go` — **parseFilter()** parses ?status= and ?source_type= query params
- `order.go` — **parseOrder()** parses ?orderBy=created_at|status

### Business Layer (business/domain/rawinputbus/)
- `rawinputbus.go` — **Create()** with MaxRetries=5 default; **Update()** partial patch; **MarkProcessing/MarkProcessed/MarkFailed/MarkForRetry()** status transitions; **ComputeBackoff()** exponential backoff (2^n min, cap 30min); **QueryRetryable()**, **ResetForReprocess()**, **RecoverStuck()**, **Query/Count/QueryByID**
- `model.go` — RawInput, NewRawInput, UpdateRawInput types
- `filter.go` — QueryFilter struct (Status, SourceType)
- `order.go` — OrderByCreatedAt, OrderByStatus constants; DefaultOrderBy = created_at DESC

### Store Layer (business/domain/rawinputbus/stores/rawinputdb/)
- `rawinputdb.go` — **Create/Update/Query/Count/QueryByID/QueryRetryable/ResetForReprocess** SQL methods
- `model.go` — rawInputDB struct + **toDBRawInput()**, **toBusRawInput()**, **toBusRawInputs()** converters
- `filter.go` — **applyFilter()** WHERE clauses for Status, SourceType
- `order.go` — orderByFields map; **orderByClause()** maps constants → SQL columns

## Impact Callouts

### ⚠ RawInput business model (business/domain/rawinputbus/model.go)
Changing this struct affects:
- `rawinputbus.go` — all method parameters and return types
- `rawinputdb/model.go` — toDBRawInput/toBusRawInput field mapping
- `rawinputdb/rawinputdb.go` — INSERT/UPDATE/SELECT SQL column lists
- `rawinputapp/model.go` — toAppRawInput() field mapping
- `business/sdk/worker/ingestworker.go` — uses RetryCount and MaxRetries for retry logic
- Migration SQL required for DB column changes

### ⚠ UpdateRawInput (business/domain/rawinputbus/model.go)
Update() applies partial fields using nil-check pattern. Does NOT support clearing NextRetryAt to NULL — use ResetForReprocess() instead. max_retries is immutable after create.

### ⚠ Result field (business/domain/rawinputbus/model.go)
Stores JSON result of successful processing (extracted tasks, events, notes):
- `rawinputbus.go` — Update() and UpdateRawInput allow Result to be set
- `ingestbus/ingestbus.go` — ProcessRawInputByID() populates result after extraction
- `rawinputapp/model.go` — toAppRawInput() includes result in response (omitted if null)

### ⚠ Storer interface (business/domain/rawinputbus/rawinputbus.go)
Adding/changing methods affects:
- `rawinputdb/rawinputdb.go` — must implement
- `business/sdk/worker/ingestworker.go` — RawInputQueuer is a subset of Storer

### ⚠ RetryCount / NextRetryAt / MaxRetries
These power the retry state machine. Changing them requires updates to:
- `rawinputbus.go` — MarkForRetry(), ComputeBackoff(), ResetForReprocess()
- `rawinputdb/rawinputdb.go` — QueryRetryable() WHERE clause, ResetForReprocess() UPDATE
- `business/sdk/worker/ingestworker.go` — terminal check: RetryCount+1 >= MaxRetries

### ⚠ ResetForReprocess() guard (business/domain/rawinputbus/rawinputbus.go)
Reprocess is guarded: only allowed when status is `failed` or `pending`. Prevents duplicate item creation when retrying failed processing.
Affects:
- `rawinputapp/rawinputapp.go` — **reprocess()** handler: error handling for invalid state (returns InvalidArgument if not failed/pending)
- Callers that invoke ResetForReprocess — must expect error if item already processed/processing

## Status State Machine

```
pending → processing → processed (terminal success)
                     → failed     (terminal, RetryCount >= MaxRetries)
pending ← (snoozed)  ← MarkForRetry() + exponential backoff NextRetryAt
pending ← ResetForReprocess() (manual reset, only allowed from failed/pending, RetryCount=0, error=nil)
processed ✗ ResetForReprocess() — guard blocks reprocessing of already-processed items
processing ✗ ResetForReprocess() — guard blocks reprocessing of items currently being processed
```

## Async Processing Flow

```
Voice/Email arrives → EnqueueText()/EnqueueEmail() → rawinputbus.Create() → status=pending

IngestWorker (every 30s):
  QueryRetryable(20) — WHERE status='pending' AND (next_retry_at IS NULL OR <= NOW())
  → per item: MarkProcessing() → ProcessRawInputByID() → MarkProcessed()+Result
  → on error: MarkForRetry() (backoff) or MarkFailed() (terminal)

Manual: POST /raw-inputs/{id}/reprocess → ResetForReprocess()
```

## Routes

| Method | Path | Handler |
|--------|------|---------|
| GET | /api/v1/raw-inputs | queryAll — filters: status, source_type; orderBy: created_at, status |
| GET | /api/v1/raw-inputs/{raw_input_id} | queryByID |
| POST | /api/v1/raw-inputs/{raw_input_id}/reprocess | reprocess — resets to pending, clears retry state |

All routes require `X-API-Key` header authentication.

## Cross-Domain Dependencies

- **ingestbus** — EnqueueEmail()/EnqueueText() create raw_input records; ProcessRawInputByID() implements RawInputProcessor interface for IngestWorker
- **voiceingestapp** — HTTP handler calls ingestbus.EnqueueText(); returns rawInputId immediately (async)
- **emailbus** — SMTP server calls ingestbus.EnqueueEmail() for incoming emails
- **emails table** — has FK raw_input_id → raw_inputs(raw_input_id)
- **main.go** — instantiates IngestWorker, runs as goroutine; wires rawinputapp.Routes
