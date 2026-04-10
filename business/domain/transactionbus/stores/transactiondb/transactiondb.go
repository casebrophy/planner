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

var _ transactionbus.Storer = (*Store)(nil)

const columns = `transaction_id, raw_input_id, source, date, description, clean_name, amount, category, context_id, notes, reviewed, created_at`

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

func (s *Store) CreateBatch(ctx context.Context, txns []transactionbus.Transaction) (int, error) {
	if len(txns) == 0 {
		return 0, nil
	}

	const colCount = 12
	valuePlaceholders := make([]string, len(txns))
	args := make([]any, 0, len(txns)*colCount)

	for i, t := range txns {
		base := i * colCount
		valuePlaceholders[i] = fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			base+1, base+2, base+3, base+4, base+5, base+6,
			base+7, base+8, base+9, base+10, base+11, base+12,
		)
		args = append(args,
			t.ID,
			t.RawInputID,
			t.Source,
			t.Date,
			t.Description,
			t.CleanName,
			t.Amount,
			t.Category,
			t.ContextID,
			t.Notes,
			t.Reviewed,
			t.CreatedAt,
		)
	}

	q := fmt.Sprintf(`
	INSERT INTO transactions
		(transaction_id, raw_input_id, source, date, description, clean_name, amount, category, context_id, notes, reviewed, created_at)
	VALUES
		%s
	ON CONFLICT (source, date, description, amount) DO NOTHING`,
		strings.Join(valuePlaceholders, ", "),
	)

	result, err := s.db.ExecContext(ctx, q, args...)
	if err != nil {
		return 0, fmt.Errorf("execcontext: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}

	return int(n), nil
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
	buf.WriteString("SELECT " + columns + " FROM transactions WHERE 1=1")

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

	const q = `SELECT ` + columns + ` FROM transactions WHERE transaction_id = :transaction_id`

	var t transactionDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &t); err != nil {
		return transactionbus.Transaction{}, fmt.Errorf("namedquerystruct: %w", err)
	}

	return toBusTransaction(t), nil
}
