# Transaction Backend System

> Financial transaction ledger captured from bank/payment CSV imports. Transactions are enriched post-import with category, clean name, and context mapping. Imported raw transactions link to rawinput ingestion records for traceability. Users can query by context, source, review status, and category; ordering by date (default DESC), amount, or created_at.

## Core Types

### App Layer

```go
type Transaction struct {
	ID          string  `json:"id"`
	RawInputID  *string `json:"rawInputId,omitempty"`
	Source      string  `json:"source"`
	Date        string  `json:"date"`
	Description string  `json:"description"`
	CleanName   *string `json:"cleanName,omitempty"`
	Amount      int     `json:"amount"`  // cents
	Category    *string `json:"category,omitempty"`
	ContextID   *string `json:"contextId,omitempty"`
	Notes       *string `json:"notes,omitempty"`
	Reviewed    bool    `json:"reviewed"`
	CreatedAt   string  `json:"createdAt"`
}

type UpdateTransaction struct {
	CleanName *string `json:"cleanName"`
	Category  *string `json:"category"`
	ContextID *string `json:"contextId"`
	Notes     *string `json:"notes"`
	Reviewed  *bool   `json:"reviewed"`
}

type ImportResult struct {
	Total    int `json:"total"`
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
}
```

### Business Layer

```go
type Transaction struct {
	ID          uuid.UUID
	RawInputID  *uuid.UUID
	Source      string
	Date        time.Time
	Description string
	CleanName   *string
	Amount      int
	Category    *string
	ContextID   *uuid.UUID
	Notes       *string
	Reviewed    bool
	CreatedAt   time.Time
}

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

type UpdateTransaction struct {
	CleanName *string
	Category  *string
	ContextID *uuid.UUID
	Notes     *string
	Reviewed  *bool
}

// Enricher enriches a transaction with AI-extracted metadata.
type Enricher interface {
	EnrichTransaction(ctx context.Context, txn Transaction) (TransactionEnrichment, error)
}

// TransactionEnrichment holds the AI-generated metadata for a transaction.
type TransactionEnrichment struct {
	CleanName          string
	Category           string
	SuggestedContextID *uuid.UUID
	ContextConfidence  float64
}

type QueryFilter struct {
	ContextID *uuid.UUID
	Source    *string
	Reviewed  *bool
	Category  *string
}

type Storer interface {
	Create(ctx context.Context, t Transaction) error
	CreateBatch(ctx context.Context, txns []Transaction) (int, error)
	Update(ctx context.Context, t Transaction) error
	Delete(ctx context.Context, t Transaction) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Transaction, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (Transaction, error)
}

const (
	OrderByDate      = "date"
	OrderByAmount    = "amount"
	OrderByCreatedAt = "created_at"
)

var DefaultOrderBy = order.NewBy(OrderByDate, order.DESC)
```

### Store Layer

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

## File Map

### App Layer (app/domain/transactionapp/)
- `transactionapp.go` — **queryAll/queryByID/update/delete/importCSV** handlers
- `model.go` — Transaction, UpdateTransaction, ImportResult, EnrichmentStatus DTOs + **toAppTransaction()**, **toAppTransactions()**, **toAppEnrichmentStatus()** converters
- `route.go` — **Routes.Add()** registers 6 endpoints with auth middleware
- `filter.go` — **parseFilter()** maps (context_id, source, reviewed, category) → QueryFilter
- `order.go` — **parseOrder()** maps (date, amount, created_at) → order constants

### Business Layer (business/domain/transactionbus/)
- `transactionbus.go` — **Create/CreateBatch/Update/Delete/Query/Count/QueryByID/EnrichmentStatus**; adds UUID + timestamps at create; **WithEnricher()** + **enrichBatch()** for async AI enrichment (2min timeout, max 3 concurrent via semaphore); atomic counters track pending/active/done/failed
- `model.go` — Transaction, NewTransaction, UpdateTransaction domain types; **Enricher** interface; **TransactionEnrichment** struct
- `enricher.go` — **ExtractorEnricher** adapter wraps extractor.Extractor for AI enrichment (CleanName, Category, SuggestedContextID)
- `filter.go` — QueryFilter struct (ContextID, Source, Reviewed, Category)
- `order.go` — OrderByDate, OrderByAmount, OrderByCreatedAt constants; DefaultOrderBy = date DESC

### Store Layer (business/domain/transactionbus/stores/transactiondb/)
- `transactiondb.go` — SQL implementation; **CreateBatch** uses bulk insert with dedup constraint
- `model.go` — transactionDB struct + **toDBTransaction()**, **toBusTransaction()** converters
- `filter.go` — **applyFilter()** WHERE clauses for ContextID, Source, Reviewed, Category
- `order.go` — orderByFields map; **orderByClause()** → SQL column names

### Sub-package
- `business/domain/transactionbus/csvparser/` — bank-specific CSV parsing formats for import

## Impact Callouts

### ⚠ Transaction struct (business/domain/transactionbus/model.go)
Changing domain type affects:
- `transactionapp/model.go` — app DTO + toAppTransaction() converter
- `transactiondb/model.go` — transactionDB struct + converters
- `transactiondb/transactiondb.go` — INSERT/UPDATE/SELECT SQL column lists
- Migration SQL required

### ⚠ QueryFilter (business/domain/transactionbus/filter.go)
Adding filter fields requires:
- `transactiondb/filter.go` — applyFilter() new WHERE clause
- `transactionapp/filter.go` — parseFilter() new query param

### ⚠ Order constants (business/domain/transactionbus/order.go)
Adding order fields requires:
- `transactiondb/order.go` — new entry in orderByFields map (const → SQL column)
- `transactionapp/order.go` — new entry in orderByFields map (request field → const)

### ⚠ Storer interface (business/domain/transactionbus/transactionbus.go)
Adding methods requires:
- `transactiondb/transactiondb.go` — implementation

### ⚠ CreateBatch deduplication
CreateBatch uses a unique constraint on (source, date, description, amount) to skip duplicates. Returns count of actually inserted rows. Skipped rows are not an error condition. If Enricher is set, spawns async enrichBatch() goroutine post-insert.

### ⚠ Enricher interface (business/domain/transactionbus/model.go)
Enrichment is optional and driven by external configuration (cfg.Extractor + cfg.OllamaEnabled). ExtractorEnricher adapts the ingestbus Extractor interface. enrichBatch():
- Bounded by 2-minute timeout context and semaphore (max 3 concurrent goroutines); additional batches block until a slot opens (or timeout expires)
- Skips transactions already having CleanName + Category
- Only applies updates if enrichment values are non-empty
- Only sets ContextID if confidence >= 0.7

## Routes

| Method | Path | Handler |
|--------|------|---------|
| GET | /api/v1/transactions | queryAll — filters: context_id, source, reviewed, category; orderBy: date, amount, created_at |
| GET | /api/v1/transactions/{transaction_id} | queryByID — 404 if not found |
| PUT | /api/v1/transactions/{transaction_id} | update — partial: cleanName, category, contextId, notes, reviewed |
| DELETE | /api/v1/transactions/{transaction_id} | delete — 204 on success |
| POST | /api/v1/transactions/import | importCSV — multipart form (file, format); bulk insert; returns ImportResult |
| GET | /api/v1/transactions/enrichment-status | enrichmentStatus — returns active/pending/done/failed counts + enabled flag |

All routes require `X-API-Key` header authentication.

## Cross-Domain Dependencies

- **rawinput** — Transaction.RawInputID links to raw_inputs for ingestion traceability
- **context** — Transaction.ContextID optionally links to contexts for user-assigned context mapping
- **csvparser** — business/domain/transactionbus/csvparser/ implements bank-specific CSV format parsers
