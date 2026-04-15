# Split Backend System

> Tracks how transactions are split among multiple parties. Each split record represents one person's share of a transaction, including their name, amount owed (in cents), optional Venmo handle, and settlement status.

## Core Types

### Business Model

```go
// Split represents one party's share of a transaction.
type Split struct {
    ID            uuid.UUID
    TransactionID uuid.UUID
    PartyName     string
    Amount        int       // cents
    VenmoHandle   *string
    Settled       bool
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

// NewSplit is the create input.
type NewSplit struct {
    TransactionID uuid.UUID
    PartyName     string
    Amount        int
    VenmoHandle   *string
}

// UpdateSplit is the update input (all fields optional).
type UpdateSplit struct {
    PartyName   *string
    Amount      *int
    VenmoHandle *string
    Settled     *bool
}

// QueryFilter narrows Split queries.
type QueryFilter struct {
    TransactionID *uuid.UUID
}
```

### DB Model

```go
// splitDB is the database representation (db:"column_name" tags).
type splitDB struct {
    ID            uuid.UUID `db:"split_id"`
    TransactionID uuid.UUID `db:"transaction_id"`
    PartyName     string    `db:"party_name"`
    Amount        int       `db:"amount"`
    VenmoHandle   *string   `db:"venmo_handle"`
    Settled       bool      `db:"settled"`
    CreatedAt     time.Time `db:"created_at"`
    UpdatedAt     time.Time `db:"updated_at"`
}

// Converters
func toDBSplit(s splitbus.Split) splitDB
func toBusSplit(s splitDB) splitbus.Split
func toBusSplits(ss []splitDB) []splitbus.Split
```

### App DTO

```go
// Split is the JSON response struct.
type Split struct {
    ID            string  `json:"id"`
    TransactionID string  `json:"transactionId"`
    PartyName     string  `json:"partyName"`
    Amount        int     `json:"amount"`
    VenmoHandle   *string `json:"venmoHandle,omitempty"`
    Settled       bool    `json:"settled"`
    CreatedAt     string  `json:"createdAt"`  // RFC3339
    UpdatedAt     string  `json:"updatedAt"`  // RFC3339
}

func (s Split) Encode() ([]byte, string, error)

// NewSplit is the JSON create request.
type NewSplit struct {
    TransactionID string  `json:"transactionId"`
    PartyName     string  `json:"partyName"`
    Amount        int     `json:"amount"`
    VenmoHandle   *string `json:"venmoHandle,omitempty"`
}

// UpdateSplit is the JSON update request (all fields optional).
type UpdateSplit struct {
    PartyName   *string `json:"partyName,omitempty"`
    Amount      *int    `json:"amount,omitempty"`
    VenmoHandle *string `json:"venmoHandle,omitempty"`
    Settled     *bool   `json:"settled,omitempty"`
}

// Converters
func toAppSplit(s splitbus.Split) Split
func toAppSplits(ss []splitbus.Split) []Split
```

### Storer Interface

```go
type Storer interface {
    Create(ctx context.Context, s Split) error
    Update(ctx context.Context, s Split) error
    Delete(ctx context.Context, s Split) error
    DeleteByTransaction(ctx context.Context, transactionID uuid.UUID) error
    Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Split, error)
    Count(ctx context.Context, filter QueryFilter) (int, error)
    QueryByID(ctx context.Context, id uuid.UUID) (Split, error)
}
```

### Ordering

```go
const (
    OrderByCreatedAt = "created_at"
)

var DefaultOrderBy = order.NewBy(OrderByCreatedAt, order.ASC)
```

## File Map

### Models

- `business/domain/splitbus/model.go` — **Split**, **NewSplit**, **UpdateSplit** structs; business layer types
- `business/domain/splitbus/stores/splitdb/model.go` — **splitDB**; DB struct + **toDBSplit()**, **toBusSplit()**, **toBusSplits()** converters
- `app/domain/splitapp/model.go` — **Split**, **NewSplit**, **UpdateSplit** app DTOs; **toAppSplit()**, **toAppSplits()** converters; **Encode()** JSON serialization

### Handlers

- `app/domain/splitapp/splitapp.go` — HTTP handlers
  - **create()** — `POST /api/v1/splits`, accepts JSON **NewSplit**, calls **Business.Create()**, returns **Split** JSON
  - **queryByTransaction()** — `GET /api/v1/transactions/{transaction_id}/splits`, queries splits by transaction ID with pagination + ordering, returns paginated **Split** array
  - **update()** — `PUT /api/v1/splits/{split_id}`, retrieves split by ID, patches with **UpdateSplit** input, calls **Business.Update()**, returns updated **Split**
  - **delete()** — `DELETE /api/v1/splits/{split_id}`, retrieves split by ID, calls **Business.Delete()**, returns no content

### Core Business

- `business/domain/splitbus/splitbus.go` — **Business** type + methods
  - **NewBusiness()** — constructs Business
  - **Create()** — generates UUID + timestamps, validates, calls **Storer.Create()**
  - **Update()** — patches fields from **UpdateSplit**, updates **UpdatedAt**, calls **Storer.Update()**
  - **Delete()** — calls **Storer.Delete()**
  - **DeleteByTransaction()** — calls **Storer.DeleteByTransaction()**, used when a transaction is deleted
  - **Query()** — pass-through to **Storer.Query()**
  - **Count()** — pass-through to **Storer.Count()**
  - **QueryByID()** — retrieves single split, wraps `sqldb.ErrDBNotFound` → app-layer **NotFound** error
- `business/domain/splitbus/filter.go` — **QueryFilter** struct definition
- `business/domain/splitbus/order.go` — **OrderByCreatedAt** constant, **DefaultOrderBy** default

### Store

- `business/domain/splitbus/stores/splitdb/splitdb.go` — **Store** type + implements **Storer** interface
  - **NewStore()** — constructs Store
  - **Create()** — INSERT into `transaction_splits`, uses named parameters
  - **Update()** — UPDATE `transaction_splits` by `split_id`
  - **Delete()** — DELETE by `split_id`
  - **DeleteByTransaction()** — DELETE all splits for a transaction ID (cascades when parent transaction deleted)
  - **Query()** — SELECT with dynamic WHERE clause from **applyFilter()**, ORDER BY via **orderByClause()**, pagination via `OFFSET...ROWS FETCH`
  - **Count()** — COUNT(*) with same filter
  - **QueryByID()** — SELECT single split by `split_id`, wraps missing rows as `sqldb.ErrDBNotFound`
- `business/domain/splitbus/stores/splitdb/model.go` — DB layer converters
- `business/domain/splitbus/stores/splitdb/filter.go` — **applyFilter()** — builds WHERE clause from **QueryFilter**; currently only **TransactionID** filter
- `business/domain/splitbus/stores/splitdb/order.go` — **orderByClause()** — maps **OrderBy.Field** to SQL column; currently only `created_at`

### Routing & Wiring

- `app/domain/splitapp/route.go` — **Routes.Add()** — instantiates **splitdb.Store**, **splitbus.Business**, and **app**, registers 4 handlers with auth middleware
- `app/domain/splitapp/order.go` — **parseOrder()** — maps request query param `?orderBy=created_at` to **order.By** using **orderByFields** map

## Impact Callouts

### ⚠ Split Struct (business/domain/splitbus/model.go)

Changing this struct shape affects **all 3 layers and requires database migration**:

- `app/domain/splitapp/model.go` — JSON binding in **NewSplit** request, response serialization in **Split** output
- `app/domain/splitapp/splitapp.go` — **create()** unpacks JSON into **NewSplit**, passes to **Business.Create()**; **update()** unpacks **UpdateSplit**, passes to **Business.Update()**; **queryByTransaction()** and **delete()** receive **Split** from business layer
- `business/domain/splitbus/splitbus.go` — **Create()** constructs full **Split** with UUID + timestamps; **Update()** modifies fields from **UpdateSplit** and updates **UpdatedAt**; **QueryByID()** returns **Split** wrapped in error handling
- `business/domain/splitbus/stores/splitdb/model.go` — **splitDB** struct with `db:""` tags maps to database columns; **toDBSplit()** and **toBusSplit()** converters bridge layers
- `business/domain/splitbus/stores/splitdb/splitdb.go` — **Create()** INSERT maps all fields to `transaction_splits` columns; **Update()** maps subset of fields; **Query()** SELECT and **Scan()** map all columns; **QueryByID()** SELECT all columns
- `business/sdk/migrate/sql/migrate.sql` — `CREATE TABLE transaction_splits` with column definitions; adding/removing fields requires new migration

**Key constraint:** Foreign key `FOREIGN KEY (transaction_id) REFERENCES transactions(transaction_id) ON DELETE CASCADE` means **DeleteByTransaction()** is triggered by parent transaction deletion.

### ⚠ Storer Interface (business/domain/splitbus/splitbus.go)

Adding/removing a method affects:

- `business/domain/splitbus/stores/splitdb/splitdb.go` — **must implement** the new method
- `business/domain/splitbus/splitbus.go` — **Business** calls the method via receiver; if method is added, **Business** must be updated to call it
- `app/domain/splitapp/splitapp.go` — if the new method is used in a handler, handler code must be added or modified

### ⚠ QueryFilter (business/domain/splitbus/filter.go)

Adding a filter field affects:

- `business/domain/splitbus/stores/splitdb/filter.go` — **applyFilter()** must add a new `if filter.XYZ != nil` clause to build the WHERE condition
- `app/domain/splitapp/splitapp.go` — handler must parse the new query parameter and populate the filter field before calling **Business.Query()**
- `business/domain/splitbus/order.go` — existing **orderByFields** map only has `"created_at"`; new filter does not add fields here (filters ≠ order fields)

### ⚠ OrderByCreatedAt Constant (business/domain/splitbus/order.go)

Changing or adding order fields affects:

- `business/domain/splitbus/stores/splitdb/order.go` — **orderByFields** map must add the new field → SQL column mapping
- `app/domain/splitapp/order.go` — **orderByFields** map must list the request param names that map to business layer constants

## Routes

| Method | Path | Handler | Notes |
|--------|------|---------|-------|
| POST | /api/v1/splits | **create()** | Create a new split; requires auth |
| GET | /api/v1/transactions/{transaction_id}/splits | **queryByTransaction()** | List splits for a transaction with pagination/ordering; requires auth |
| PUT | /api/v1/splits/{split_id} | **update()** | Update split fields (party name, amount, Venmo handle, settled status); requires auth |
| DELETE | /api/v1/splits/{split_id} | **delete()** | Remove a split; requires auth |

All routes enforce `X-API-Key` auth via **mid.Auth()** middleware.

## Cross-Domain Dependencies

- **transactions**: Foreign key constraint `transaction_splits.transaction_id → transactions.transaction_id` with `ON DELETE CASCADE` means deleting a transaction deletes all its splits. The **DeleteByTransaction()** method is designed to be called when a parent transaction is deleted.
- **foundation/web**: Handler signature uses **web.Encoder** return type and **web.Param()**, **web.Decode()** utilities
- **foundation/logger**: Logging via **logger.Logger** in Store and Business constructors
- **business/sdk/page**: Pagination via **page.Parse()** and **page.Page** type in handlers and store
- **business/sdk/order**: Ordering via **order.By** and **order.NewBy()** in business and handlers
- **app/sdk/errs**: Error code mapping (NotFound, InvalidArgument, Internal) → HTTP status codes
- **app/sdk/query**: **query.NewResult()** wraps paginated results with total count
- **business/sdk/sqldb**: Database utilities **NamedExecContext()**, **NamedQuerySlice()**, **NamedQueryStruct()**, and **ErrDBNotFound** sentinel

## Database Schema

```sql
CREATE TABLE transaction_splits (
    split_id       UUID        NOT NULL DEFAULT gen_random_uuid(),
    transaction_id UUID        NOT NULL,
    party_name     TEXT        NOT NULL,
    amount         INTEGER     NOT NULL,  -- cents
    venmo_handle   TEXT,
    settled        BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (split_id),
    FOREIGN KEY (transaction_id) REFERENCES transactions(transaction_id) ON DELETE CASCADE
);

CREATE INDEX idx_transaction_splits_transaction_id ON transaction_splits(transaction_id);
```

**Note:** Foreign key ON DELETE CASCADE ensures splits are removed when their parent transaction is deleted.

## Notes

- **Transaction context required**: All Storer methods accept a `context.Context` parameter for cancellation and timeout support.
- **Settled flag**: Tracks whether this party's portion of the transaction has been settled (e.g., payment made).
- **Venmo integration**: Optional `venmo_handle` field stores recipient's Venmo handle for potential payment automation or notifications.
- **Amount in cents**: All monetary values are stored as integers (cents) to avoid floating-point precision issues.
