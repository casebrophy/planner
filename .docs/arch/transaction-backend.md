# Transaction Backend System

The Transaction domain provides CRUD access to financial transaction records, plus a CSV import endpoint for bulk ingestion from bank exports. Transactions are optionally linked to a raw_input and a context. The architecture follows the standard layered pattern: HTTP handlers → business logic core → database store, with a csvparser sub-package for format-aware CSV parsing.

## Core Types

### Transaction (Business Layer)
```go
type Transaction struct {
	ID          uuid.UUID
	RawInputID  *uuid.UUID
	Source      string
	Date        time.Time
	Description string
	CleanName   *string
	Amount      int       // cents, negative = debit
	Category    *string
	ContextID   *uuid.UUID
	Notes       *string
	Reviewed    bool
	CreatedAt   time.Time
}
```

### NewTransaction (Business Layer)
```go
type NewTransaction struct {
	RawInputID  *uuid.UUID
	Source      string
	Date        time.Time
	Description string
	CleanName   *string
	Amount      int
	Category    *string
	ContextID   *uuid.UUID
	Notes       *string
}
```

### UpdateTransaction (Business Layer)
```go
type UpdateTransaction struct {
	CleanName *string
	Category  *string
	ContextID *uuid.UUID
	Notes     *string
	Reviewed  *bool
}
```

### Transaction (App Layer)
```go
type Transaction struct {
	ID          string  `json:"id"`
	RawInputID  *string `json:"rawInputId,omitempty"`
	Source      string  `json:"source"`
	Date        string  `json:"date"`
	Description string  `json:"description"`
	CleanName   *string `json:"cleanName,omitempty"`
	Amount      int     `json:"amount"`
	Category    *string `json:"category,omitempty"`
	ContextID   *string `json:"contextId,omitempty"`
	Notes       *string `json:"notes,omitempty"`
	Reviewed    bool    `json:"reviewed"`
	CreatedAt   string  `json:"createdAt"`
}
```

### transactionDB (Store Layer)
```go
type transactionDB struct {
	ID          uuid.UUID  `db:"transaction_id"`
	RawInputID  *uuid.UUID `db:"raw_input_id"`
	Source      string     `db:"source"`
	Date        time.Time  `db:"date"`
	Description string     `db:"description"`
	CleanName   *string    `db:"clean_name"`
	Amount      int        `db:"amount"`
	Category    *string    `db:"category"`
	ContextID   *uuid.UUID `db:"context_id"`
	Notes       *string    `db:"notes"`
	Reviewed    bool       `db:"reviewed"`
	CreatedAt   time.Time  `db:"created_at"`
}
```

### QueryFilter
```go
type QueryFilter struct {
	ContextID *uuid.UUID
	Source    *string
	Reviewed  *bool
	Category  *string
}
```

### Storer Interface
```go
type Storer interface {
	Create(ctx context.Context, t Transaction) error
	CreateBatch(ctx context.Context, txns []Transaction) (int, error)
	Update(ctx context.Context, t Transaction) error
	Delete(ctx context.Context, t Transaction) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Transaction, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (Transaction, error)
}
```

## File Map

### App Layer (HTTP Handlers)
- **`app/domain/transactionapp/model.go`** — HTTP DTOs: Transaction (response), UpdateTransaction (request body), ImportResult (import response) with conversion functions:
  - **toAppTransaction()** — transactionbus.Transaction → app Transaction (UUID fields to strings, times to RFC3339)
  - **toAppTransactions()** — slice converter
  - **Transaction.Encode()** — implements web.Encoder via json.Marshal
  - **ImportResult.Encode()** — implements web.Encoder via json.Marshal
- **`app/domain/transactionapp/transactionapp.go`** — Handler methods:
  - **queryAll()** — GET /api/v1/transactions, parses page/rows, filter, orderBy; calls transactionBus.Query + transactionBus.Count; returns paginated result
  - **queryByID()** — GET /api/v1/transactions/{transaction_id}, parses UUID from path param; checks sqldb.ErrDBNotFound → errs.NotFound
  - **update()** — PUT /api/v1/transactions/{transaction_id}, looks up existing record, decodes UpdateTransaction body, converts ContextID string → uuid.UUID, calls transactionBus.Update; returns updated transaction
  - **delete()** — DELETE /api/v1/transactions/{transaction_id}, looks up existing record, calls transactionBus.Delete; returns web.NoResponse{}
  - **importCSV()** — POST /api/v1/transactions/import, reads multipart form file (max 10 MB), reads optional `format` field, delegates to csvparser.Parse, bulk-creates via transactionBus.CreateBatch; returns ImportResult with total/imported/skipped counts
- **`app/domain/transactionapp/route.go`** — **Routes.Add()** — registers all five endpoints with Auth middleware; instantiates transactiondb.Store and transactionbus.Business
- **`app/domain/transactionapp/filter.go`** — **parseFilter()** — parses query parameters into QueryFilter:
  - `context_id` → uuid.UUID → filter.ContextID
  - `source` → string → filter.Source
  - `reviewed` → bool (strconv.ParseBool) → filter.Reviewed
  - `category` → string → filter.Category
- **`app/domain/transactionapp/order.go`** — **parseOrder()** — maps request orderBy field names to transactionbus constants:
  - `"date"` → `transactionbus.OrderByDate`
  - `"amount"` → `transactionbus.OrderByAmount`
  - `"created_at"` → `transactionbus.OrderByCreatedAt`

### Business Layer (Core Logic)
- **`business/domain/transactionbus/model.go`** — Business models: Transaction, NewTransaction, UpdateTransaction
- **`business/domain/transactionbus/transactionbus.go`** — Business struct, Storer interface definition, and methods:
  - **NewBusiness()** — constructor taking logger and Storer
  - **Create()** — generates UUID, sets Reviewed=false and CreatedAt to now, calls storer.Create
  - **CreateBatch()** — generates UUIDs for each, sets Reviewed=false and CreatedAt to now, calls storer.CreateBatch; returns count of inserted rows (dedup skips are not counted)
  - **Update()** — applies partial update (only non-nil fields from UpdateTransaction), calls storer.Update; returns updated Transaction
  - **Delete()** — calls storer.Delete
  - **Query()** — delegates to storer with filter/order/pagination
  - **Count()** — delegates to storer to count filtered transactions
  - **QueryByID()** — delegates to storer by UUID
- **`business/domain/transactionbus/filter.go`** — QueryFilter struct (ContextID, Source, Reviewed, Category)
- **`business/domain/transactionbus/order.go`** — Order field constants and DefaultOrderBy:
  - `OrderByDate = "date"` (DefaultOrderBy: DESC)
  - `OrderByAmount = "amount"`
  - `OrderByCreatedAt = "created_at"`

### Store Layer (Database)
- **`business/domain/transactionbus/stores/transactiondb/model.go`** — transactionDB internal struct (all db tags), conversion functions:
  - **toDBTransaction()** — transactionbus.Transaction → transactionDB
  - **toBusTransaction()** — transactionDB → transactionbus.Transaction
  - **toBusTransactions()** — slice converter
- **`business/domain/transactionbus/stores/transactiondb/transactiondb.go`** — Store struct and methods:
  - **NewStore()** — constructor taking logger and *sqlx.DB
  - **Create()** — INSERT INTO transactions with all 12 columns via named query
  - **CreateBatch()** — bulk INSERT with positional placeholders; uses `ON CONFLICT ON CONSTRAINT idx_transactions_dedup DO NOTHING` to silently skip duplicates; returns rows affected count
  - **Update()** — UPDATE transactions SET `clean_name, category, context_id, notes, reviewed` WHERE `transaction_id = :transaction_id`
  - **Delete()** — DELETE FROM transactions WHERE `transaction_id = :transaction_id`
  - **Query()** — SELECT all columns with WHERE 1=1 base, applies applyFilter, ORDER BY clause, OFFSET/FETCH pagination
  - **Count()** — SELECT COUNT(*) FROM transactions with same filter applied
  - **QueryByID()** — SELECT all columns WHERE `transaction_id = :transaction_id`
- **`business/domain/transactionbus/stores/transactiondb/filter.go`** — **applyFilter()** — builds WHERE clauses:
  - `filter.ContextID` → `AND context_id = :filter_context_id` (exact match)
  - `filter.Source` → `AND source = :filter_source` (exact match)
  - `filter.Reviewed` → `AND reviewed = :filter_reviewed` (exact match)
  - `filter.Category` → `AND category = :filter_category` (exact match)
- **`business/domain/transactionbus/stores/transactiondb/order.go`** — orderByFields map (business constants → SQL column names), **orderByClause()** — validates field exists and formats `column direction` string

### CSV Parser
- **`business/domain/transactionbus/csvparser/formats.go`** — Format struct, Row struct, registered formats, and parsing helpers:
  - **Row** — parsed output: Source, Date, Description, Amount (cents, negative = debit)
  - **Format** — per-bank descriptor: Name, HeaderMatch (unique header fingerprint), DateCol, DescCol, AmountCol, DateLayout, AmountNegate
  - **detect()** — auto-detects format by checking all registered HeaderMatch sets against actual CSV headers
  - **lookup()** — finds a format by name for explicit override
  - **parseRow()** — parses a single CSV record: date (time.Parse with layout), description (trim quotes), amount (float → cents via math.Round×100, negate if AmountNegate=true)
- **`business/domain/transactionbus/csvparser/csvparser.go`** — **Parse()** — top-level entry point:
  - Reads all CSV records; requires at least 1 header row + 1 data row
  - If `formatName` is empty, calls detect(); otherwise calls lookup()
  - Validates required columns exist in header; iterates data rows calling parseRow()
  - Returns `[]Row` or a descriptive error including the row number on parse failure
- **`business/domain/transactionbus/csvparser/csvparser_test.go`** — Unit tests for Parse() covering all three formats and auto-detection

#### Supported CSV Formats

| Format Name | Bank | Key Headers | Amount Sign |
|---|---|---|---|
| `chase_checking` | Chase Checking | Details, Posting Date, Description, Amount, Type, Balance | Negative = debit |
| `chase_credit` | Chase Credit | Transaction Date, Post Date, Description, Category, Type, Amount | Negative = debit |
| `amex` | American Express | Date, Description, Amount | Positive = charge (negated internally) |

Auto-detection matches by checking that all `HeaderMatch` headers are present in the CSV. Explicit format override via the `format` form field in the import request bypasses detection.

## Database Schema

```sql
CREATE TABLE transactions (
    transaction_id UUID        NOT NULL DEFAULT gen_random_uuid(),
    raw_input_id   UUID        REFERENCES raw_inputs(raw_input_id),
    source         TEXT        NOT NULL,
    date           TIMESTAMPTZ NOT NULL,
    description    TEXT        NOT NULL,
    clean_name     TEXT,
    amount         INTEGER     NOT NULL,  -- cents, negative = debit
    category       TEXT,
    context_id     UUID        REFERENCES contexts(context_id) ON DELETE SET NULL,
    notes          TEXT,
    reviewed       BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (transaction_id)
);

CREATE INDEX idx_transactions_date ON transactions(date DESC);
CREATE INDEX idx_transactions_context ON transactions(context_id);
CREATE INDEX idx_transactions_reviewed ON transactions(reviewed, created_at);
CREATE UNIQUE INDEX idx_transactions_dedup ON transactions(source, date, description, amount);
```

Note: `idx_transactions_dedup` is a unique index on `(source, date, description, amount)`. CreateBatch uses `ON CONFLICT ON CONSTRAINT idx_transactions_dedup DO NOTHING` to silently skip duplicate rows when re-importing the same CSV.

## Impact Callouts

### Transaction struct (business/domain/transactionbus/model.go)
Adding or removing a field affects:
- `business/domain/transactionbus/stores/transactiondb/model.go` — toDBTransaction() and toBusTransaction() must include the new field; transactionDB struct must add the corresponding `db:` tag
- `app/domain/transactionapp/model.go` — app Transaction struct and toAppTransaction() must include the new field with its `json:` tag
- `business/domain/transactionbus/stores/transactiondb/transactiondb.go` — SQL INSERT column list and VALUES named params in Create() and CreateBatch() must be updated; SELECT column list in Query() and QueryByID() must be updated; UPDATE SET clause in Update() must be updated if the field is mutable
- `business/sdk/migrate/sql/migrate.sql` — must add/remove columns and update constraints or indexes

### NewTransaction struct (business/domain/transactionbus/model.go)
Adding a field affects:
- `business/domain/transactionbus/transactionbus.go` — Create() and CreateBatch() must assign the new field when constructing the Transaction
- Caller code (e.g. importCSV handler or future ingestion pipeline) must populate the new field

### UpdateTransaction struct (business/domain/transactionbus/model.go)
Adding a field affects:
- `business/domain/transactionbus/transactionbus.go` — Update() must apply the new field to the Transaction before calling storer.Update
- `business/domain/transactionbus/stores/transactiondb/transactiondb.go` — SQL UPDATE SET clause must include the new column
- `app/domain/transactionapp/model.go` — app UpdateTransaction struct must expose the field; update() handler must map it to the bus type

### Storer interface (business/domain/transactionbus/transactionbus.go)
Adding or changing a method affects:
- `business/domain/transactionbus/stores/transactiondb/transactiondb.go` — Store struct must implement the new method with the matching signature
- Any mock storer used in tests must implement the new method
- Business methods in transactionbus.go that delegate to storer must call new methods as appropriate

### QueryFilter struct (business/domain/transactionbus/filter.go)
Adding a filter field affects:
- `business/domain/transactionbus/stores/transactiondb/filter.go` — applyFilter() must add a new conditional block with the WHERE clause fragment and named param key
- `app/domain/transactionapp/filter.go` — parseFilter() must parse the new query parameter from r.URL.Query() and set the filter field

### Order constants (business/domain/transactionbus/order.go)
Adding a new OrderBy constant affects:
- `business/domain/transactionbus/stores/transactiondb/order.go` — orderByFields map must add the constant → SQL column name entry
- `app/domain/transactionapp/order.go` — orderByFields map must add the request field string → business constant entry

### CSV Format (business/domain/transactionbus/csvparser/formats.go)
Adding a new bank format:
- Add a new Format entry to the `formats` slice in formats.go with unique `HeaderMatch` headers
- No other files need changes; detect() iterates all registered formats automatically
- Add test cases to csvparser_test.go covering the new format

## Routes

| Method | Path | Handler | Notes |
|--------|------|---------|-------|
| GET | /api/v1/transactions | queryAll | Query params: `page`, `rows`, `context_id`, `source`, `reviewed`, `category`, `orderBy` (date, amount, created_at); default order: date DESC |
| GET | /api/v1/transactions/{transaction_id} | queryByID | Fetches single transaction by UUID; returns 404 if not found |
| PUT | /api/v1/transactions/{transaction_id} | update | Partial update: cleanName, category, contextId, notes, reviewed; returns updated transaction |
| DELETE | /api/v1/transactions/{transaction_id} | delete | Deletes transaction; returns 204 No Content |
| POST | /api/v1/transactions/import | importCSV | Multipart form: `file` (CSV bytes), optional `format` (chase_checking, chase_credit, amex); returns ImportResult |

All endpoints require Auth middleware (X-API-Key header validation).

## Cross-Domain Dependencies

- **RawInput Domain** — Transactions have an optional foreign key `raw_input_id` referencing `raw_inputs(raw_input_id)`; set when a transaction originates from an ingested raw_input record (currently unused by the CSV import path, which sets RawInputID to nil)
- **Context Domain** — Transactions have an optional `context_id` foreign key to `contexts(context_id)` with ON DELETE SET NULL; UpdateTransaction allows linking or re-linking a transaction to a context after creation
- **Page SDK** (`business/sdk/page`) — queryAll uses page.Page for pagination (Offset, RowsPerPage, Number)
- **Order SDK** (`business/sdk/order`) — Query uses order.By with Field constant and Direction (ASC/DESC)
- **sqldb utilities** (`foundation/sqldb`) — Store uses NamedExecContext, NamedQuerySlice, NamedQueryStruct helpers; ErrDBNotFound (= sql.ErrNoRows) is returned by QueryByID when no row matches and must be checked by callers
- **Error handling** (`app/sdk/errs`) — queryByID, update, and delete check errors.Is(err, sqldb.ErrDBNotFound) and map to errs.NotFound (HTTP 404); other errors map to errs.Internal (HTTP 500)
- **HTTP web framework** (`foundation/web`) — Handlers implement web.HandlerFunc signature and return web.Encoder; Transaction.Encode() and ImportResult.Encode() satisfy the interface via json.Marshal
