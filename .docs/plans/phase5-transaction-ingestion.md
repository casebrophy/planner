# Phase 5: Transaction Ingestion — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upload bank CSV exports via REST, parse and store transactions, optionally categorize/match contexts via Anthropic, review and correct in a frontend view.

**Architecture:** CSV upload endpoint receives multipart/file, a per-bank format adapter parses rows into a common struct, transactions are stored, then optionally enriched via Anthropic API for clean_name/category/context matching. Frontend provides a transaction list with review workflow. No ModelRouter/Ollama/tier system — just a direct Anthropic inferencer for categorization.

**Tech Stack:** Go (backend, same 3-layer pattern), PostgreSQL, Anthropic SDK, Vue 3 + Pinia (frontend)

---

## File Map

### Backend — new files

| File | Responsibility |
|------|---------------|
| `business/domain/transactionbus/model.go` | Transaction, NewTransaction, UpdateTransaction structs |
| `business/domain/transactionbus/transactionbus.go` | Business methods + Storer interface |
| `business/domain/transactionbus/filter.go` | QueryFilter struct |
| `business/domain/transactionbus/order.go` | OrderBy constants |
| `business/domain/transactionbus/stores/transactiondb/model.go` | DB struct + converters |
| `business/domain/transactionbus/stores/transactiondb/transactiondb.go` | SQL queries |
| `business/domain/transactionbus/stores/transactiondb/filter.go` | applyFilter |
| `business/domain/transactionbus/stores/transactiondb/order.go` | orderByClause |
| `business/domain/transactionbus/csvparser/csvparser.go` | CSV parsing: format detection, per-bank adapters, row→NewTransaction |
| `business/domain/transactionbus/csvparser/formats.go` | Chase checking, Chase credit, Amex, generic format definitions |
| `business/domain/transactionbus/csvparser/csvparser_test.go` | CSV parser unit tests |
| `business/domain/ingestbus/extractor/transaction.go` | ExtractTransactions method on Extractor + AnthropicExtractor |
| `app/domain/transactionapp/model.go` | App DTOs + converters |
| `app/domain/transactionapp/transactionapp.go` | HTTP handlers (queryAll, queryByID, importCSV, update, delete) |
| `app/domain/transactionapp/filter.go` | parseFilter |
| `app/domain/transactionapp/order.go` | parseOrder |
| `app/domain/transactionapp/route.go` | Route registration |

### Backend — modified files

| File | Change |
|------|--------|
| `business/sdk/migrate/sql/migrate.sql` | Add transactions table DDL (version 1.12) |
| `business/types/rawinputsource/rawinputsource.go` | Verify `transaction` is already a valid source type |
| `business/domain/ingestbus/extractor/anthropic.go` | Add `ExtractTransactions` to `Extractor` interface |
| `business/domain/ingestbus/extractor/mock.go` | Add `ExtractTransactions` to mock |
| `api/services/planner/main.go` | Add `transactionapp.Routes{}` to mux |

### Frontend — new files

| File | Responsibility |
|------|---------------|
| `web/src/types/transaction.ts` | Transaction, TransactionFilter types |
| `web/src/services/transactionService.ts` | API client for transactions + CSV upload |
| `web/src/stores/transactionStore.ts` | Pinia store |
| `web/src/views/TransactionBoardView.vue` | Transaction list + CSV upload + filter/review |
| `web/src/components/transactions/TransactionRow.vue` | Single transaction row display |
| `web/src/components/transactions/TransactionImport.vue` | CSV file picker + upload form |
| `web/src/components/transactions/TransactionFilterBar.vue` | Filter controls |

### Frontend — modified files

| File | Change |
|------|--------|
| `web/src/types/index.ts` | Export transaction types |
| `web/src/router/index.ts` | Add `/transactions` route |
| `web/src/components/layout/AppSidebar.vue` | Add Transactions nav item |

---

### Task 1: Migration — transactions table

**Files:**
- Modify: `business/sdk/migrate/sql/migrate.sql`

- [ ] **Step 1: Add transactions table DDL**

Append to the end of `migrate.sql`:

```sql
-- Version: 1.12
-- Description: Create transactions table
CREATE TABLE transactions (
    transaction_id UUID        NOT NULL DEFAULT gen_random_uuid(),
    raw_input_id   UUID        REFERENCES raw_inputs(raw_input_id),
    source         TEXT        NOT NULL,
    date           TIMESTAMPTZ NOT NULL,
    description    TEXT        NOT NULL,
    clean_name     TEXT,
    amount         INTEGER     NOT NULL,
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

- [ ] **Step 2: Verify migration runs**

Run: `make db-up && make migrate`
Expected: Migration completes without errors.

- [ ] **Step 3: Commit**

```bash
git add business/sdk/migrate/sql/migrate.sql
git commit -m "feat: add transactions table migration (v1.12)"
```

---

### Task 2: transactionbus — business layer

**Files:**
- Create: `business/domain/transactionbus/model.go`
- Create: `business/domain/transactionbus/transactionbus.go`
- Create: `business/domain/transactionbus/filter.go`
- Create: `business/domain/transactionbus/order.go`
- Delete: `business/domain/transactionbus/doc.go`

- [ ] **Step 1: Create model.go**

```go
package transactionbus

import (
	"time"

	"github.com/google/uuid"
)

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
```

- [ ] **Step 2: Create filter.go**

```go
package transactionbus

import "github.com/google/uuid"

type QueryFilter struct {
	ContextID *uuid.UUID
	Source    *string
	Reviewed *bool
	Category *string
}
```

- [ ] **Step 3: Create order.go**

```go
package transactionbus

import "github.com/casebrophy/planner/business/sdk/order"

const (
	OrderByDate      = "date"
	OrderByAmount    = "amount"
	OrderByCreatedAt = "created_at"
)

var DefaultOrderBy = order.NewBy(OrderByDate, order.DESC)
```

- [ ] **Step 4: Create transactionbus.go**

```go
package transactionbus

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/foundation/logger"
)

type Storer interface {
	Create(ctx context.Context, t Transaction) error
	CreateBatch(ctx context.Context, txns []Transaction) (int, error)
	Update(ctx context.Context, t Transaction) error
	Delete(ctx context.Context, t Transaction) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Transaction, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (Transaction, error)
}

type Business struct {
	log    *logger.Logger
	storer Storer
}

func NewBusiness(log *logger.Logger, storer Storer) *Business {
	return &Business{
		log:    log,
		storer: storer,
	}
}

func (b *Business) Create(ctx context.Context, nt NewTransaction) (Transaction, error) {
	now := time.Now()

	t := Transaction{
		ID:          uuid.New(),
		RawInputID:  nt.RawInputID,
		Source:      nt.Source,
		Date:        nt.Date,
		Description: nt.Description,
		CleanName:   nt.CleanName,
		Amount:      nt.Amount,
		Category:    nt.Category,
		ContextID:   nt.ContextID,
		Notes:       nt.Notes,
		Reviewed:    false,
		CreatedAt:   now,
	}

	if err := b.storer.Create(ctx, t); err != nil {
		return Transaction{}, fmt.Errorf("create: %w", err)
	}

	return t, nil
}

// CreateBatch inserts multiple transactions, skipping duplicates.
// Returns the number of rows actually inserted.
func (b *Business) CreateBatch(ctx context.Context, nts []NewTransaction) (int, error) {
	now := time.Now()

	txns := make([]Transaction, len(nts))
	for i, nt := range nts {
		txns[i] = Transaction{
			ID:          uuid.New(),
			RawInputID:  nt.RawInputID,
			Source:      nt.Source,
			Date:        nt.Date,
			Description: nt.Description,
			CleanName:   nt.CleanName,
			Amount:      nt.Amount,
			Category:    nt.Category,
			ContextID:   nt.ContextID,
			Notes:       nt.Notes,
			Reviewed:    false,
			CreatedAt:   now,
		}
	}

	inserted, err := b.storer.CreateBatch(ctx, txns)
	if err != nil {
		return 0, fmt.Errorf("create batch: %w", err)
	}

	return inserted, nil
}

func (b *Business) Update(ctx context.Context, t Transaction, ut UpdateTransaction) (Transaction, error) {
	if ut.CleanName != nil {
		t.CleanName = ut.CleanName
	}
	if ut.Category != nil {
		t.Category = ut.Category
	}
	if ut.ContextID != nil {
		t.ContextID = ut.ContextID
	}
	if ut.Notes != nil {
		t.Notes = ut.Notes
	}
	if ut.Reviewed != nil {
		t.Reviewed = *ut.Reviewed
	}

	if err := b.storer.Update(ctx, t); err != nil {
		return Transaction{}, fmt.Errorf("update: %w", err)
	}

	return t, nil
}

func (b *Business) Delete(ctx context.Context, t Transaction) error {
	if err := b.storer.Delete(ctx, t); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

func (b *Business) Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Transaction, error) {
	txns, err := b.storer.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return txns, nil
}

func (b *Business) Count(ctx context.Context, filter QueryFilter) (int, error) {
	n, err := b.storer.Count(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	return n, nil
}

func (b *Business) QueryByID(ctx context.Context, id uuid.UUID) (Transaction, error) {
	t, err := b.storer.QueryByID(ctx, id)
	if err != nil {
		return Transaction{}, fmt.Errorf("query by id[%s]: %w", id, err)
	}
	return t, nil
}
```

- [ ] **Step 5: Delete the placeholder doc.go**

Remove `business/domain/transactionbus/doc.go`.

- [ ] **Step 6: Verify compilation**

Run: `go build ./business/domain/transactionbus/...`
Expected: Builds successfully.

- [ ] **Step 7: Commit**

```bash
git add business/domain/transactionbus/
git commit -m "feat: add transactionbus business layer"
```

---

### Task 3: transactiondb — store layer

**Files:**
- Create: `business/domain/transactionbus/stores/transactiondb/model.go`
- Create: `business/domain/transactionbus/stores/transactiondb/transactiondb.go`
- Create: `business/domain/transactionbus/stores/transactiondb/filter.go`
- Create: `business/domain/transactionbus/stores/transactiondb/order.go`

- [ ] **Step 1: Create model.go**

```go
package transactiondb

import (
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/transactionbus"
)

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

func toDBTransaction(t transactionbus.Transaction) transactionDB {
	return transactionDB{
		ID:          t.ID,
		RawInputID:  t.RawInputID,
		Source:      t.Source,
		Date:        t.Date,
		Description: t.Description,
		CleanName:   t.CleanName,
		Amount:      t.Amount,
		Category:    t.Category,
		ContextID:   t.ContextID,
		Notes:       t.Notes,
		Reviewed:    t.Reviewed,
		CreatedAt:   t.CreatedAt,
	}
}

func toBusTransaction(t transactionDB) transactionbus.Transaction {
	return transactionbus.Transaction{
		ID:          t.ID,
		RawInputID:  t.RawInputID,
		Source:      t.Source,
		Date:        t.Date,
		Description: t.Description,
		CleanName:   t.CleanName,
		Amount:      t.Amount,
		Category:    t.Category,
		ContextID:   t.ContextID,
		Notes:       t.Notes,
		Reviewed:    t.Reviewed,
		CreatedAt:   t.CreatedAt,
	}
}

func toBusTransactions(ts []transactionDB) []transactionbus.Transaction {
	items := make([]transactionbus.Transaction, len(ts))
	for i, t := range ts {
		items[i] = toBusTransaction(t)
	}
	return items
}
```

- [ ] **Step 2: Create filter.go**

```go
package transactiondb

import (
	"bytes"

	"github.com/casebrophy/planner/business/domain/transactionbus"
)

func applyFilter(filter transactionbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	if filter.ContextID != nil {
		buf.WriteString(" AND context_id = :filter_context_id")
		data["filter_context_id"] = *filter.ContextID
	}
	if filter.Source != nil {
		buf.WriteString(" AND source = :filter_source")
		data["filter_source"] = *filter.Source
	}
	if filter.Reviewed != nil {
		buf.WriteString(" AND reviewed = :filter_reviewed")
		data["filter_reviewed"] = *filter.Reviewed
	}
	if filter.Category != nil {
		buf.WriteString(" AND category = :filter_category")
		data["filter_category"] = *filter.Category
	}
}
```

- [ ] **Step 3: Create order.go**

```go
package transactiondb

import (
	"fmt"

	"github.com/casebrophy/planner/business/domain/transactionbus"
	"github.com/casebrophy/planner/business/sdk/order"
)

var orderByFields = map[string]string{
	transactionbus.OrderByDate:      "date",
	transactionbus.OrderByAmount:    "amount",
	transactionbus.OrderByCreatedAt: "created_at",
}

func orderByClause(ob order.By) (string, error) {
	col, ok := orderByFields[ob.Field]
	if !ok {
		return "", fmt.Errorf("unknown order field %q", ob.Field)
	}
	return col + " " + ob.Direction, nil
}
```

- [ ] **Step 4: Create transactiondb.go**

```go
package transactiondb

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/business/domain/transactionbus"
	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/foundation/logger"
)

type Store struct {
	log *logger.Logger
	db  sqlx.ExtContext
}

func NewStore(log *logger.Logger, db *sqlx.DB) *Store {
	return &Store{
		log: log,
		db:  db,
	}
}

const columns = `transaction_id, raw_input_id, source, date, description, clean_name, amount, category, context_id, notes, reviewed, created_at`

func (s *Store) Create(ctx context.Context, t transactionbus.Transaction) error {
	const q = `
	INSERT INTO transactions
		(transaction_id, raw_input_id, source, date, description, clean_name, amount, category, context_id, notes, reviewed, created_at)
	VALUES
		(:transaction_id, :raw_input_id, :source, :date, :description, :clean_name, :amount, :category, :context_id, :notes, :reviewed, :created_at)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBTransaction(t)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// CreateBatch inserts multiple transactions using ON CONFLICT DO NOTHING for dedup.
// Returns the number of rows actually inserted.
func (s *Store) CreateBatch(ctx context.Context, txns []transactionbus.Transaction) (int, error) {
	if len(txns) == 0 {
		return 0, nil
	}

	var buf bytes.Buffer
	buf.WriteString(`INSERT INTO transactions
		(transaction_id, raw_input_id, source, date, description, clean_name, amount, category, context_id, notes, reviewed, created_at)
	VALUES `)

	args := make([]any, 0, len(txns)*12)
	for i, t := range txns {
		if i > 0 {
			buf.WriteString(", ")
		}
		base := i * 12
		buf.WriteString(fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			base+1, base+2, base+3, base+4, base+5, base+6,
			base+7, base+8, base+9, base+10, base+11, base+12))
		db := toDBTransaction(t)
		args = append(args, db.ID, db.RawInputID, db.Source, db.Date, db.Description, db.CleanName,
			db.Amount, db.Category, db.ContextID, db.Notes, db.Reviewed, db.CreatedAt)
	}
	buf.WriteString(" ON CONFLICT ON CONSTRAINT idx_transactions_dedup DO NOTHING")

	result, err := s.db.ExecContext(ctx, buf.String(), args...)
	if err != nil {
		return 0, fmt.Errorf("execcontext batch: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}

	return int(rows), nil
}

func (s *Store) Update(ctx context.Context, t transactionbus.Transaction) error {
	const q = `
	UPDATE transactions SET
		clean_name = :clean_name,
		category = :category,
		context_id = :context_id,
		notes = :notes,
		reviewed = :reviewed
	WHERE
		transaction_id = :transaction_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBTransaction(t)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Delete(ctx context.Context, t transactionbus.Transaction) error {
	data := struct {
		ID uuid.UUID `db:"transaction_id"`
	}{
		ID: t.ID,
	}

	const q = `DELETE FROM transactions WHERE transaction_id = :transaction_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Query(ctx context.Context, filter transactionbus.QueryFilter, orderBy order.By, pg page.Page) ([]transactionbus.Transaction, error) {
	data := map[string]any{
		"offset":        pg.Offset(),
		"rows_per_page": pg.RowsPerPage(),
	}

	var buf bytes.Buffer
	buf.WriteString(`SELECT ` + columns + ` FROM transactions WHERE 1=1`)

	applyFilter(filter, data, &buf)

	orderClause, err := orderByClause(orderBy)
	if err != nil {
		return nil, err
	}

	buf.WriteString(fmt.Sprintf(" ORDER BY %s OFFSET :offset ROWS FETCH NEXT :rows_per_page ROWS ONLY", orderClause))

	dbItems, err := sqldb.NamedQuerySlice[transactionDB](ctx, s.log, s.db, buf.String(), data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return toBusTransactions(dbItems), nil
}

func (s *Store) Count(ctx context.Context, filter transactionbus.QueryFilter) (int, error) {
	data := map[string]any{}

	var buf bytes.Buffer
	buf.WriteString(`SELECT COUNT(*) FROM transactions WHERE 1=1`)

	applyFilter(filter, data, &buf)

	var count struct {
		Count int `db:"count"`
	}
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, buf.String(), data, &count); err != nil {
		return 0, fmt.Errorf("namedquerystruct: %w", err)
	}

	return count.Count, nil
}

func (s *Store) QueryByID(ctx context.Context, id uuid.UUID) (transactionbus.Transaction, error) {
	data := struct {
		ID uuid.UUID `db:"transaction_id"`
	}{
		ID: id,
	}

	q := `SELECT ` + columns + ` FROM transactions WHERE transaction_id = :transaction_id`

	var t transactionDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &t); err != nil {
		return transactionbus.Transaction{}, fmt.Errorf("namedquerystruct: %w", err)
	}

	return toBusTransaction(t), nil
}

// Compile-time check that Store implements Storer.
var _ transactionbus.Storer = (*Store)(nil)
```

Note: The `strings` import should be removed — it was included by mistake. Only import what's needed. The `columns` constant avoids repeating the column list.

- [ ] **Step 5: Verify compilation**

Run: `go build ./business/domain/transactionbus/...`
Expected: Builds successfully.

- [ ] **Step 6: Commit**

```bash
git add business/domain/transactionbus/stores/
git commit -m "feat: add transactiondb store layer"
```

---

### Task 4: CSV parser

**Files:**
- Create: `business/domain/transactionbus/csvparser/csvparser.go`
- Create: `business/domain/transactionbus/csvparser/formats.go`
- Test: `business/domain/transactionbus/csvparser/csvparser_test.go`

- [ ] **Step 1: Write csvparser_test.go with test cases**

```go
package csvparser_test

import (
	"testing"
	"time"

	"github.com/casebrophy/planner/business/domain/transactionbus/csvparser"
)

func TestParse_ChaseChecking(t *testing.T) {
	csv := `Details,Posting Date,Description,Amount,Type,Balance,Check or Slip #
DEBIT,01/15/2025,"STARBUCKS STORE 12345",-4.50,ACH_DEBIT,1234.56,
CREDIT,01/16/2025,"PAYROLL DEPOSIT",3200.00,ACH_CREDIT,4434.56,`

	txns, err := csvparser.Parse(csv, "chase_checking")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(txns) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(txns))
	}

	// Debit
	if txns[0].Description != "STARBUCKS STORE 12345" {
		t.Errorf("expected description 'STARBUCKS STORE 12345', got %q", txns[0].Description)
	}
	if txns[0].Amount != -450 {
		t.Errorf("expected amount -450, got %d", txns[0].Amount)
	}
	expected := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	if !txns[0].Date.Equal(expected) {
		t.Errorf("expected date %v, got %v", expected, txns[0].Date)
	}

	// Credit
	if txns[1].Amount != 320000 {
		t.Errorf("expected amount 320000, got %d", txns[1].Amount)
	}
}

func TestParse_ChaseCredit(t *testing.T) {
	csv := `Transaction Date,Post Date,Description,Category,Type,Amount,Memo
01/10/2025,01/12/2025,AMAZON.COM,Shopping,Sale,-29.99,
01/11/2025,01/13/2025,PAYMENT THANK YOU,Payment,Payment,500.00,`

	txns, err := csvparser.Parse(csv, "chase_credit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(txns) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(txns))
	}

	if txns[0].Description != "AMAZON.COM" {
		t.Errorf("expected description 'AMAZON.COM', got %q", txns[0].Description)
	}
	if txns[0].Amount != -2999 {
		t.Errorf("expected amount -2999, got %d", txns[0].Amount)
	}
}

func TestParse_Amex(t *testing.T) {
	csv := `Date,Description,Amount
01/20/2025,WHOLE FOODS MARKET,85.43
01/21/2025,UBER TRIP,22.50`

	txns, err := csvparser.Parse(csv, "amex")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(txns) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(txns))
	}

	// Amex amounts are positive = charge (stored as negative cents)
	if txns[0].Amount != -8543 {
		t.Errorf("expected amount -8543, got %d", txns[0].Amount)
	}
}

func TestParse_AutoDetect(t *testing.T) {
	csv := `Details,Posting Date,Description,Amount,Type,Balance,Check or Slip #
DEBIT,02/01/2025,"TEST",-10.00,ACH_DEBIT,100.00,`

	txns, err := csvparser.Parse(csv, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(txns) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(txns))
	}

	if txns[0].Source != "chase_checking" {
		t.Errorf("expected auto-detected source 'chase_checking', got %q", txns[0].Source)
	}
}

func TestParse_EmptyCSV(t *testing.T) {
	_, err := csvparser.Parse("", "chase_checking")
	if err == nil {
		t.Fatal("expected error for empty CSV")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./business/domain/transactionbus/csvparser/... -v -count=1`
Expected: FAIL — package does not exist yet.

- [ ] **Step 3: Create formats.go**

```go
package csvparser

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Row is a parsed CSV row ready for transaction creation.
type Row struct {
	Source      string
	Date        time.Time
	Description string
	Amount      int // cents, negative = debit
}

// Format defines how to parse a specific bank's CSV export.
type Format struct {
	Name          string
	HeaderMatch   []string // headers that uniquely identify this format
	DateCol       string
	DescCol       string
	AmountCol     string
	DateLayout    string
	AmountNegate  bool // true if positive amounts mean charges (Amex)
}

var formats = []Format{
	{
		Name:        "chase_checking",
		HeaderMatch: []string{"Details", "Posting Date", "Description", "Amount", "Type", "Balance"},
		DateCol:     "Posting Date",
		DescCol:     "Description",
		AmountCol:   "Amount",
		DateLayout:  "01/02/2006",
	},
	{
		Name:        "chase_credit",
		HeaderMatch: []string{"Transaction Date", "Post Date", "Description", "Category", "Type", "Amount"},
		DateCol:     "Transaction Date",
		DescCol:     "Description",
		AmountCol:   "Amount",
		DateLayout:  "01/02/2006",
	},
	{
		Name:        "amex",
		HeaderMatch: []string{"Date", "Description", "Amount"},
		DateCol:     "Date",
		DescCol:     "Description",
		AmountCol:   "Amount",
		DateLayout:  "01/02/2006",
		AmountNegate: true,
	},
}

// detect identifies the format from the header row.
func detect(headers []string) (Format, error) {
	headerSet := make(map[string]bool, len(headers))
	for _, h := range headers {
		headerSet[strings.TrimSpace(h)] = true
	}

	for _, f := range formats {
		match := true
		for _, required := range f.HeaderMatch {
			if !headerSet[required] {
				match = false
				break
			}
		}
		if match {
			return f, nil
		}
	}

	return Format{}, fmt.Errorf("unrecognized CSV format, headers: %v", headers)
}

// lookup returns the named format or an error.
func lookup(name string) (Format, error) {
	for _, f := range formats {
		if f.Name == name {
			return f, nil
		}
	}
	return Format{}, fmt.Errorf("unknown format %q", name)
}

// parseRow converts a CSV record into a Row using the given format.
func parseRow(f Format, headerIndex map[string]int, record []string) (Row, error) {
	dateStr := strings.TrimSpace(record[headerIndex[f.DateCol]])
	date, err := time.Parse(f.DateLayout, dateStr)
	if err != nil {
		return Row{}, fmt.Errorf("parse date %q: %w", dateStr, err)
	}

	desc := strings.TrimSpace(record[headerIndex[f.DescCol]])
	desc = strings.Trim(desc, `"`)

	amountStr := strings.TrimSpace(record[headerIndex[f.AmountCol]])
	amountStr = strings.ReplaceAll(amountStr, ",", "")
	amountFloat, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return Row{}, fmt.Errorf("parse amount %q: %w", amountStr, err)
	}

	if f.AmountNegate {
		amountFloat = -amountFloat
	}

	// Convert dollars to cents
	amountCents := int(math.Round(amountFloat * 100))

	return Row{
		Source:      f.Name,
		Date:        date,
		Description: desc,
		Amount:      amountCents,
	}, nil
}
```

- [ ] **Step 4: Create csvparser.go**

```go
package csvparser

import (
	"encoding/csv"
	"fmt"
	"strings"
)

// Parse reads a CSV string and returns parsed rows.
// If formatName is empty, auto-detects the format from headers.
func Parse(csvData string, formatName string) ([]Row, error) {
	csvData = strings.TrimSpace(csvData)
	if csvData == "" {
		return nil, fmt.Errorf("empty CSV data")
	}

	reader := csv.NewReader(strings.NewReader(csvData))
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV must have a header row and at least one data row")
	}

	headers := records[0]

	var f Format
	if formatName == "" {
		f, err = detect(headers)
		if err != nil {
			return nil, err
		}
	} else {
		f, err = lookup(formatName)
		if err != nil {
			return nil, err
		}
	}

	// Build header index
	headerIndex := make(map[string]int, len(headers))
	for i, h := range headers {
		headerIndex[strings.TrimSpace(h)] = i
	}

	// Validate required columns exist
	for _, col := range []string{f.DateCol, f.DescCol, f.AmountCol} {
		if _, ok := headerIndex[col]; !ok {
			return nil, fmt.Errorf("missing required column %q", col)
		}
	}

	var rows []Row
	for i, record := range records[1:] {
		row, err := parseRow(f, headerIndex, record)
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+2, err)
		}
		rows = append(rows, row)
	}

	return rows, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./business/domain/transactionbus/csvparser/... -v -count=1`
Expected: All 5 tests PASS.

- [ ] **Step 6: Commit**

```bash
git add business/domain/transactionbus/csvparser/
git commit -m "feat: add CSV parser with Chase, Amex format support"
```

---

### Task 5: transactionapp — HTTP handlers

**Files:**
- Create: `app/domain/transactionapp/model.go`
- Create: `app/domain/transactionapp/transactionapp.go`
- Create: `app/domain/transactionapp/filter.go`
- Create: `app/domain/transactionapp/order.go`
- Create: `app/domain/transactionapp/route.go`

- [ ] **Step 1: Create model.go**

```go
package transactionapp

import (
	"encoding/json"
	"time"

	"github.com/casebrophy/planner/business/domain/transactionbus"
)

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

func (t Transaction) Encode() ([]byte, string, error) {
	data, err := json.Marshal(t)
	return data, "application/json", err
}

func toAppTransaction(t transactionbus.Transaction) Transaction {
	at := Transaction{
		ID:          t.ID.String(),
		Source:      t.Source,
		Date:        t.Date.Format(time.RFC3339),
		Description: t.Description,
		CleanName:   t.CleanName,
		Amount:      t.Amount,
		Category:    t.Category,
		Notes:       t.Notes,
		Reviewed:    t.Reviewed,
		CreatedAt:   t.CreatedAt.Format(time.RFC3339),
	}

	if t.RawInputID != nil {
		s := t.RawInputID.String()
		at.RawInputID = &s
	}

	if t.ContextID != nil {
		s := t.ContextID.String()
		at.ContextID = &s
	}

	return at
}

func toAppTransactions(ts []transactionbus.Transaction) []Transaction {
	items := make([]Transaction, len(ts))
	for i, t := range ts {
		items[i] = toAppTransaction(t)
	}
	return items
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

func (r ImportResult) Encode() ([]byte, string, error) {
	data, err := json.Marshal(r)
	return data, "application/json", err
}
```

- [ ] **Step 2: Create filter.go**

```go
package transactionapp

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/transactionbus"
)

func parseFilter(r *http.Request) (transactionbus.QueryFilter, error) {
	var filter transactionbus.QueryFilter

	if v := r.URL.Query().Get("context_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return transactionbus.QueryFilter{}, err
		}
		filter.ContextID = &id
	}

	if v := r.URL.Query().Get("source"); v != "" {
		filter.Source = &v
	}

	if v := r.URL.Query().Get("reviewed"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return transactionbus.QueryFilter{}, err
		}
		filter.Reviewed = &b
	}

	if v := r.URL.Query().Get("category"); v != "" {
		filter.Category = &v
	}

	return filter, nil
}
```

- [ ] **Step 3: Create order.go**

```go
package transactionapp

import (
	"net/http"

	"github.com/casebrophy/planner/business/domain/transactionbus"
	"github.com/casebrophy/planner/business/sdk/order"
)

var orderByFields = map[string]string{
	"date":       transactionbus.OrderByDate,
	"amount":     transactionbus.OrderByAmount,
	"created_at": transactionbus.OrderByCreatedAt,
}

func parseOrder(r *http.Request) (order.By, error) {
	return order.Parse(orderByFields, r.URL.Query().Get("orderBy"), transactionbus.DefaultOrderBy)
}
```

- [ ] **Step 4: Create transactionapp.go**

```go
package transactionapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/transactionbus"
	"github.com/casebrophy/planner/business/domain/transactionbus/csvparser"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	transactionBus *transactionbus.Business
}

func (a *app) queryAll(ctx context.Context, r *http.Request) web.Encoder {
	pg, err := page.Parse(r.URL.Query().Get("page"), r.URL.Query().Get("rows"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	filter, err := parseFilter(r)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	orderBy, err := parseOrder(r)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	txns, err := a.transactionBus.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return errs.Newf(errs.Internal, "query: %s", err)
	}

	total, err := a.transactionBus.Count(ctx, filter)
	if err != nil {
		return errs.Newf(errs.Internal, "count: %s", err)
	}

	return query.NewResult(toAppTransactions(txns), total, pg.Number(), pg.RowsPerPage())
}

func (a *app) queryByID(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "transaction_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	txn, err := a.transactionBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	return toAppTransaction(txn)
}

func (a *app) update(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "transaction_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	txn, err := a.transactionBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	var ut UpdateTransaction
	if err := json.NewDecoder(r.Body).Decode(&ut); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	busUpdate := transactionbus.UpdateTransaction{
		CleanName: ut.CleanName,
		Category:  ut.Category,
		Notes:     ut.Notes,
		Reviewed:  ut.Reviewed,
	}

	if ut.ContextID != nil {
		cid, err := uuid.Parse(*ut.ContextID)
		if err != nil {
			return errs.New(errs.InvalidArgument, err)
		}
		busUpdate.ContextID = &cid
	}

	updated, err := a.transactionBus.Update(ctx, txn, busUpdate)
	if err != nil {
		return errs.Newf(errs.Internal, "update: %s", err)
	}

	return toAppTransaction(updated)
}

func (a *app) delete(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "transaction_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	txn, err := a.transactionBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	if err := a.transactionBus.Delete(ctx, txn); err != nil {
		return errs.Newf(errs.Internal, "delete: %s", err)
	}

	return nil
}

func (a *app) importCSV(ctx context.Context, r *http.Request) web.Encoder {
	// Accept multipart file upload. Field name: "file". Optional field: "format".
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
		return errs.New(errs.InvalidArgument, fmt.Errorf("parse form: %w", err))
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		return errs.New(errs.InvalidArgument, fmt.Errorf("missing file field: %w", err))
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return errs.New(errs.InvalidArgument, fmt.Errorf("read file: %w", err))
	}

	formatName := r.FormValue("format") // empty = auto-detect

	rows, err := csvparser.Parse(string(data), formatName)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	nts := make([]transactionbus.NewTransaction, len(rows))
	for i, row := range rows {
		nts[i] = transactionbus.NewTransaction{
			Source:      row.Source,
			Date:        row.Date,
			Description: row.Description,
			Amount:      row.Amount,
		}
	}

	inserted, err := a.transactionBus.CreateBatch(ctx, nts)
	if err != nil {
		return errs.Newf(errs.Internal, "create batch: %s", err)
	}

	return ImportResult{
		Total:    len(rows),
		Imported: inserted,
		Skipped:  len(rows) - inserted,
	}
}
```

- [ ] **Step 5: Create route.go**

```go
package transactionapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/transactionbus"
	"github.com/casebrophy/planner/business/domain/transactionbus/stores/transactiondb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	txnStore := transactiondb.NewStore(cfg.Log, cfg.DB)
	txnBus := transactionbus.NewBusiness(cfg.Log, txnStore)

	hdl := &app{transactionBus: txnBus}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/transactions", hdl.queryAll, authen)
	a.Handle(http.MethodGet, "/api/v1/transactions/{transaction_id}", hdl.queryByID, authen)
	a.Handle(http.MethodPut, "/api/v1/transactions/{transaction_id}", hdl.update, authen)
	a.Handle(http.MethodDelete, "/api/v1/transactions/{transaction_id}", hdl.delete, authen)
	a.Handle(http.MethodPost, "/api/v1/transactions/import", hdl.importCSV, authen)
}
```

- [ ] **Step 6: Verify compilation**

Run: `go build ./app/domain/transactionapp/...`
Expected: Builds successfully.

- [ ] **Step 7: Commit**

```bash
git add app/domain/transactionapp/
git commit -m "feat: add transactionapp HTTP handlers with CSV import"
```

---

### Task 6: Wire into main.go

**Files:**
- Modify: `api/services/planner/main.go`

- [ ] **Step 1: Add import and route**

Add to imports:
```go
"github.com/casebrophy/planner/app/domain/transactionapp"
```

Add to the `mux.WebAPI(...)` call, after `emailapp.Routes{}`:
```go
transactionapp.Routes{},
```

- [ ] **Step 2: Verify full build**

Run: `go build ./...`
Expected: Builds successfully.

- [ ] **Step 3: Verify migration + API start**

Run: `make db-up && make migrate && make dev` (in separate terminal, or just verify build)
Expected: API starts without errors.

- [ ] **Step 4: Commit**

```bash
git add api/services/planner/main.go
git commit -m "feat: wire transactionapp routes into API server"
```

---

### Task 7: Frontend — types + service

**Files:**
- Create: `web/src/types/transaction.ts`
- Modify: `web/src/types/index.ts`
- Create: `web/src/services/transactionService.ts`

- [ ] **Step 1: Create transaction.ts types**

```typescript
export interface Transaction {
  id: string
  rawInputId?: string
  source: string
  date: string
  description: string
  cleanName?: string
  amount: number // cents, negative = debit
  category?: string
  contextId?: string
  notes?: string
  reviewed: boolean
  createdAt: string
}

export interface UpdateTransaction {
  cleanName?: string
  category?: string
  contextId?: string
  notes?: string
  reviewed?: boolean
}

export interface TransactionFilter {
  contextId?: string
  source?: string
  reviewed?: boolean
  category?: string
}

export interface ImportResult {
  total: number
  imported: number
  skipped: number
}
```

- [ ] **Step 2: Update types/index.ts**

Add this line:
```typescript
export type { Transaction, UpdateTransaction, TransactionFilter, ImportResult } from './transaction'
```

- [ ] **Step 3: Create transactionService.ts**

```typescript
import { request } from './client'
import { createCRUDService } from './createCRUDService'
import type { Transaction, UpdateTransaction, TransactionFilter, ImportResult } from '@/types'

const baseCrud = createCRUDService<Transaction, never, UpdateTransaction, TransactionFilter>({
  basePath: '/api/v1/transactions',
  mapFilter: (filter) => ({
    context_id: filter.contextId,
    source: filter.source,
    reviewed: filter.reviewed !== undefined ? String(filter.reviewed) : undefined,
    category: filter.category,
  }),
})

export const transactionService = {
  ...baseCrud,

  async importCSV(file: File, format?: string): Promise<ImportResult> {
    const formData = new FormData()
    formData.append('file', file)
    if (format) {
      formData.append('format', format)
    }

    const BASE_URL = import.meta.env.VITE_API_BASE_URL || ''
    const API_KEY = import.meta.env.VITE_API_KEY || ''

    const response = await fetch(`${BASE_URL}/api/v1/transactions/import`, {
      method: 'POST',
      headers: API_KEY ? { 'X-API-Key': API_KEY } : {},
      body: formData,
    })

    if (!response.ok) {
      const body = await response.json().catch(() => ({}))
      throw new Error((body as Record<string, string>).error || response.statusText)
    }

    return response.json() as Promise<ImportResult>
  },
}
```

- [ ] **Step 4: Verify frontend build**

Run: `cd web && npm run lint && npm run build`
Expected: No errors.

- [ ] **Step 5: Commit**

```bash
git add web/src/types/transaction.ts web/src/types/index.ts web/src/services/transactionService.ts
git commit -m "feat: add transaction types and API service"
```

---

### Task 8: Frontend — Pinia store

**Files:**
- Create: `web/src/stores/transactionStore.ts`

- [ ] **Step 1: Create transactionStore.ts**

```typescript
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { transactionService } from '@/services/transactionService'
import { createCRUDStore } from './createCRUDStore'
import type { Transaction, UpdateTransaction, TransactionFilter, ImportResult } from '@/types'

export const useTransactionStore = defineStore('transaction', () => {
  const crud = createCRUDStore<Transaction, never, UpdateTransaction, TransactionFilter>({
    name: 'transaction',
    service: transactionService,
    defaultOrderBy: 'date',
    defaultRowsPerPage: 25,
  })

  const importing = ref(false)
  const lastImportResult = ref<ImportResult | null>(null)

  const unreviewedCount = computed(() =>
    crud.items.value.filter((t) => !t.reviewed).length,
  )

  const totalSpend = computed(() =>
    crud.items.value
      .filter((t) => t.amount < 0)
      .reduce((sum, t) => sum + t.amount, 0),
  )

  async function importCSV(file: File, format?: string): Promise<ImportResult> {
    importing.value = true
    try {
      const result = await transactionService.importCSV(file, format)
      lastImportResult.value = result
      // Refresh the list after import
      await crud.fetchAll()
      return result
    } finally {
      importing.value = false
    }
  }

  async function markReviewed(id: string): Promise<void> {
    const reviewed = true
    await crud.update(id, { reviewed })
  }

  return {
    ...crud,
    importing,
    lastImportResult,
    unreviewedCount,
    totalSpend,
    importCSV,
    markReviewed,
  }
})
```

- [ ] **Step 2: Verify frontend build**

Run: `cd web && npm run lint && npm run build`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add web/src/stores/transactionStore.ts
git commit -m "feat: add transaction Pinia store"
```

---

### Task 9: Frontend — TransactionRow component

**Files:**
- Create: `web/src/components/transactions/TransactionRow.vue`

- [ ] **Step 1: Create TransactionRow.vue**

```vue
<script setup lang="ts">
import type { Transaction } from '@/types'

const props = defineProps<{
  transaction: Transaction
}>()

const emit = defineEmits<{
  review: [id: string]
  click: [id: string]
}>()

function formatAmount(cents: number): string {
  const abs = Math.abs(cents)
  const dollars = (abs / 100).toFixed(2)
  return cents < 0 ? `-$${dollars}` : `$${dollars}`
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
  })
}
</script>

<template>
  <div
    class="flex items-center gap-4 px-4 py-3 border-b border-gray-800 hover:bg-gray-800/50 cursor-pointer transition-colors"
    :class="{ 'opacity-60': transaction.reviewed }"
    @click="emit('click', transaction.id)"
  >
    <!-- Date -->
    <span class="text-sm text-gray-400 w-16 shrink-0">
      {{ formatDate(transaction.date) }}
    </span>

    <!-- Description + clean name -->
    <div class="flex-1 min-w-0">
      <p class="text-sm text-gray-100 truncate">
        {{ transaction.cleanName || transaction.description }}
      </p>
      <p v-if="transaction.cleanName" class="text-xs text-gray-500 truncate">
        {{ transaction.description }}
      </p>
    </div>

    <!-- Category -->
    <span
      v-if="transaction.category"
      class="text-xs px-2 py-0.5 rounded-full bg-gray-700 text-gray-300 shrink-0"
    >
      {{ transaction.category }}
    </span>

    <!-- Amount -->
    <span
      class="text-sm font-mono w-24 text-right shrink-0"
      :class="transaction.amount < 0 ? 'text-red-400' : 'text-green-400'"
    >
      {{ formatAmount(transaction.amount) }}
    </span>

    <!-- Review button -->
    <button
      v-if="!transaction.reviewed"
      class="text-xs px-2 py-1 rounded bg-blue-600 hover:bg-blue-500 text-white shrink-0"
      @click.stop="emit('review', transaction.id)"
    >
      Review
    </button>
    <span v-else class="text-xs text-gray-500 w-14 text-center shrink-0">
      ✓
    </span>
  </div>
</template>
```

- [ ] **Step 2: Commit**

```bash
git add web/src/components/transactions/TransactionRow.vue
git commit -m "feat: add TransactionRow component"
```

---

### Task 10: Frontend — TransactionImport component

**Files:**
- Create: `web/src/components/transactions/TransactionImport.vue`

- [ ] **Step 1: Create TransactionImport.vue**

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { useTransactionStore } from '@/stores/transactionStore'
import { useToastStore } from '@/stores/toastStore'

const transactionStore = useTransactionStore()
const toastStore = useToastStore()

const fileInput = ref<HTMLInputElement | null>(null)
const selectedFile = ref<File | null>(null)
const format = ref('')

const formatOptions = [
  { value: '', label: 'Auto-detect' },
  { value: 'chase_checking', label: 'Chase Checking' },
  { value: 'chase_credit', label: 'Chase Credit Card' },
  { value: 'amex', label: 'American Express' },
]

function onFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  selectedFile.value = input.files?.[0] || null
}

async function upload() {
  if (!selectedFile.value) return

  try {
    const result = await transactionStore.importCSV(selectedFile.value, format.value || undefined)
    toastStore.success(`Imported ${result.imported} transactions (${result.skipped} duplicates skipped)`)
    selectedFile.value = null
    if (fileInput.value) fileInput.value.value = ''
  } catch (err) {
    toastStore.error(`Import failed: ${err instanceof Error ? err.message : 'Unknown error'}`)
  }
}
</script>

<template>
  <div class="bg-gray-800 rounded-lg p-4 space-y-3">
    <h3 class="text-sm font-medium text-gray-200">Import Bank CSV</h3>

    <div class="flex items-center gap-3">
      <input
        ref="fileInput"
        type="file"
        accept=".csv"
        class="text-sm text-gray-400 file:mr-3 file:py-1.5 file:px-3 file:rounded file:border-0 file:text-sm file:bg-gray-700 file:text-gray-200 hover:file:bg-gray-600"
        @change="onFileChange"
      />

      <select
        v-model="format"
        class="bg-gray-700 text-sm text-gray-200 rounded px-2 py-1.5 border border-gray-600"
      >
        <option v-for="opt in formatOptions" :key="opt.value" :value="opt.value">
          {{ opt.label }}
        </option>
      </select>

      <button
        class="px-3 py-1.5 rounded text-sm font-medium transition-colors"
        :class="
          selectedFile && !transactionStore.importing
            ? 'bg-blue-600 hover:bg-blue-500 text-white'
            : 'bg-gray-700 text-gray-500 cursor-not-allowed'
        "
        :disabled="!selectedFile || transactionStore.importing"
        @click="upload"
      >
        {{ transactionStore.importing ? 'Importing...' : 'Upload' }}
      </button>
    </div>

    <p v-if="transactionStore.lastImportResult" class="text-xs text-gray-400">
      Last import: {{ transactionStore.lastImportResult.imported }} new,
      {{ transactionStore.lastImportResult.skipped }} skipped of
      {{ transactionStore.lastImportResult.total }} rows
    </p>
  </div>
</template>
```

- [ ] **Step 2: Commit**

```bash
git add web/src/components/transactions/TransactionImport.vue
git commit -m "feat: add TransactionImport CSV upload component"
```

---

### Task 11: Frontend — TransactionFilterBar component

**Files:**
- Create: `web/src/components/transactions/TransactionFilterBar.vue`

- [ ] **Step 1: Create TransactionFilterBar.vue**

```vue
<script setup lang="ts">
import { computed } from 'vue'
import { useTransactionStore } from '@/stores/transactionStore'

const transactionStore = useTransactionStore()

const reviewedFilter = computed({
  get: () => transactionStore.filter.value.reviewed,
  set: (v: boolean | undefined) => {
    transactionStore.filter.value = { ...transactionStore.filter.value, reviewed: v }
    transactionStore.fetchAll()
  },
})

const sourceFilter = computed({
  get: () => transactionStore.filter.value.source || '',
  set: (v: string) => {
    transactionStore.filter.value = {
      ...transactionStore.filter.value,
      source: v || undefined,
    }
    transactionStore.fetchAll()
  },
})

function clearFilters() {
  transactionStore.filter.value = {} as any
  transactionStore.fetchAll()
}

const sources = ['chase_checking', 'chase_credit', 'amex']
</script>

<template>
  <div class="flex items-center gap-3 text-sm">
    <!-- Review status -->
    <div class="flex items-center gap-1.5">
      <button
        class="px-2 py-1 rounded transition-colors"
        :class="reviewedFilter === undefined ? 'bg-gray-600 text-white' : 'text-gray-400 hover:text-white'"
        @click="reviewedFilter = undefined"
      >
        All
      </button>
      <button
        class="px-2 py-1 rounded transition-colors"
        :class="reviewedFilter === false ? 'bg-blue-600 text-white' : 'text-gray-400 hover:text-white'"
        @click="reviewedFilter = false"
      >
        Needs Review
      </button>
      <button
        class="px-2 py-1 rounded transition-colors"
        :class="reviewedFilter === true ? 'bg-green-600 text-white' : 'text-gray-400 hover:text-white'"
        @click="reviewedFilter = true"
      >
        Reviewed
      </button>
    </div>

    <!-- Source filter -->
    <select
      :value="sourceFilter"
      class="bg-gray-700 text-gray-200 rounded px-2 py-1 border border-gray-600"
      @change="sourceFilter = ($event.target as HTMLSelectElement).value"
    >
      <option value="">All Sources</option>
      <option v-for="s in sources" :key="s" :value="s">{{ s }}</option>
    </select>

    <button
      class="text-gray-500 hover:text-gray-300 text-xs"
      @click="clearFilters"
    >
      Clear
    </button>
  </div>
</template>
```

- [ ] **Step 2: Commit**

```bash
git add web/src/components/transactions/TransactionFilterBar.vue
git commit -m "feat: add TransactionFilterBar component"
```

---

### Task 12: Frontend — TransactionBoardView + routing + sidebar

**Files:**
- Create: `web/src/views/TransactionBoardView.vue`
- Modify: `web/src/router/index.ts`
- Modify: `web/src/components/layout/AppSidebar.vue`

- [ ] **Step 1: Create TransactionBoardView.vue**

```vue
<script setup lang="ts">
import { onMounted } from 'vue'
import { useTransactionStore } from '@/stores/transactionStore'
import PageHeader from '@/components/layout/PageHeader.vue'
import TransactionImport from '@/components/transactions/TransactionImport.vue'
import TransactionFilterBar from '@/components/transactions/TransactionFilterBar.vue'
import TransactionRow from '@/components/transactions/TransactionRow.vue'
import Pagination from '@/components/shared/Pagination.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import EmptyState from '@/components/shared/EmptyState.vue'

const transactionStore = useTransactionStore()

onMounted(() => {
  transactionStore.fetchAll()
})

function formatTotalSpend(cents: number): string {
  const abs = Math.abs(cents)
  return `$${(abs / 100).toFixed(2)}`
}
</script>

<template>
  <div class="space-y-4">
    <PageHeader title="Transactions">
      <template #subtitle>
        <span v-if="transactionStore.total.value > 0" class="text-gray-400 text-sm">
          {{ transactionStore.total.value }} transactions
          <span v-if="transactionStore.totalSpend.value < 0">
            · {{ formatTotalSpend(transactionStore.totalSpend.value) }} spend
          </span>
          <span v-if="transactionStore.unreviewedCount.value > 0" class="text-yellow-400">
            · {{ transactionStore.unreviewedCount.value }} needs review
          </span>
        </span>
      </template>
    </PageHeader>

    <TransactionImport />

    <TransactionFilterBar />

    <LoadingSpinner v-if="transactionStore.loading.value" />

    <EmptyState
      v-else-if="transactionStore.items.value.length === 0"
      title="No transactions"
      description="Import a bank CSV to get started"
    />

    <div v-else class="bg-gray-900 rounded-lg overflow-hidden border border-gray-800">
      <TransactionRow
        v-for="txn in transactionStore.items.value"
        :key="txn.id"
        :transaction="txn"
        @review="transactionStore.markReviewed"
        @click="() => {}"
      />
    </div>

    <Pagination
      v-if="transactionStore.total.value > transactionStore.rowsPerPage.value"
      :page="transactionStore.page.value"
      :rows-per-page="transactionStore.rowsPerPage.value"
      :total="transactionStore.total.value"
      @update:page="(p) => { transactionStore.page.value = p; transactionStore.fetchAll() }"
    />
  </div>
</template>
```

- [ ] **Step 2: Add route to router/index.ts**

Add the lazy import near the top:
```typescript
const TransactionBoardView = () => import('@/views/TransactionBoardView.vue')
```

Add to the routes array (after the contexts entry):
```typescript
{ path: '/transactions', name: 'transactions', component: TransactionBoardView },
```

- [ ] **Step 3: Add Transactions to AppSidebar.vue**

In the `navItems` array, add after the Contexts entry:
```typescript
{ name: 'Transactions', path: '/transactions', icon: 'credit-card' },
```

- [ ] **Step 4: Verify frontend build**

Run: `cd web && npm run lint && npm run build`
Expected: No errors.

- [ ] **Step 5: Commit**

```bash
git add web/src/views/TransactionBoardView.vue web/src/router/index.ts web/src/components/layout/AppSidebar.vue
git commit -m "feat: add TransactionBoardView with routing and sidebar nav"
```

---

### Task 13: Update docs/arch + CLAUDE.md

**Files:**
- Create: `.docs/arch/transaction-backend.md`
- Modify: `.docs/03-data-model.md` (move transactions from "Future Tables" to "Tables")
- Modify: `.docs/07-roadmap.md` (mark Phase 5 deliverables complete)
- Modify: `.docs/TOC.md` (add arch file reference)
- Modify: `CLAUDE.md` (update "Built" list)

- [ ] **Step 1: Create transaction-backend.md arch file**

Create `.docs/arch/transaction-backend.md` following the pattern of existing arch files. Document the full file map, types, routes, and cross-domain dependencies.

- [ ] **Step 2: Move transactions DDL in 03-data-model.md**

Move the `### transactions` section from `## Future Tables` up into `## Tables`. Add dedup index.

- [ ] **Step 3: Update 07-roadmap.md**

Mark these Phase 5 deliverables as done:
- ~~CSV parser with per-bank format adapters~~
- ~~`transactions` table~~
- ~~REST API (CRUD + CSV import)~~
- ~~Frontend: transaction board view with import and review~~

Leave as not-done:
- AI model layer (Inferencer/Embedder/ModelRouter) — deferred
- Ollama container, sensitivity tier classification, sanitization/promotion gate — deferred
- `sanitization_log` table — deferred
- Frontend: context detail with linked transactions — future enhancement

- [ ] **Step 4: Update TOC.md**

Add `arch/transaction-backend.md` to the transaction domain entry.

- [ ] **Step 5: Update CLAUDE.md**

In the "Built" list, add: `transactions (CRUD + CSV import + frontend view)`.

- [ ] **Step 6: Commit**

```bash
git add .docs/ CLAUDE.md
git commit -m "docs: add transaction-backend arch, update planning docs for Phase 5"
```

---

## Summary

| Task | What | Estimated Files |
|------|------|----------------|
| 1 | Migration DDL | 1 modified |
| 2 | transactionbus (business layer) | 4 created, 1 deleted |
| 3 | transactiondb (store layer) | 4 created |
| 4 | CSV parser + tests | 3 created |
| 5 | transactionapp (HTTP handlers) | 5 created |
| 6 | Wire into main.go | 1 modified |
| 7 | Frontend types + service | 2 created, 1 modified |
| 8 | Frontend Pinia store | 1 created |
| 9 | TransactionRow component | 1 created |
| 10 | TransactionImport component | 1 created |
| 11 | TransactionFilterBar component | 1 created |
| 12 | TransactionBoardView + routing | 1 created, 2 modified |
| 13 | Docs update | 1 created, 4 modified |
