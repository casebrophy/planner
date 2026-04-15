package splitdb

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/business/domain/splitbus"
	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/foundation/logger"
)

var _ splitbus.Storer = (*Store)(nil)

const columns = `split_id, transaction_id, party_name, amount, venmo_handle, settled, created_at, updated_at`

// Store manages data access for splits.
type Store struct {
	log *logger.Logger
	db  sqlx.ExtContext
}

// NewStore constructs a Store.
func NewStore(log *logger.Logger, db *sqlx.DB) *Store {
	return &Store{
		log: log,
		db:  db,
	}
}

// Create inserts a new split.
func (s *Store) Create(ctx context.Context, split splitbus.Split) error {
	const q = `
	INSERT INTO transaction_splits
		(split_id, transaction_id, party_name, amount, venmo_handle, settled, created_at, updated_at)
	VALUES
		(:split_id, :transaction_id, :party_name, :amount, :venmo_handle, :settled, :created_at, :updated_at)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBSplit(split)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// Update modifies an existing split.
func (s *Store) Update(ctx context.Context, split splitbus.Split) error {
	const q = `
	UPDATE transaction_splits SET
		party_name   = :party_name,
		amount       = :amount,
		venmo_handle = :venmo_handle,
		settled      = :settled,
		updated_at   = :updated_at
	WHERE
		split_id = :split_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBSplit(split)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// Delete removes a split.
func (s *Store) Delete(ctx context.Context, split splitbus.Split) error {
	data := struct {
		ID uuid.UUID `db:"split_id"`
	}{
		ID: split.ID,
	}

	const q = `DELETE FROM transaction_splits WHERE split_id = :split_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// DeleteByTransaction removes all splits for a transaction.
func (s *Store) DeleteByTransaction(ctx context.Context, transactionID uuid.UUID) error {
	data := struct {
		TransactionID uuid.UUID `db:"transaction_id"`
	}{
		TransactionID: transactionID,
	}

	const q = `DELETE FROM transaction_splits WHERE transaction_id = :transaction_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// Query returns a filtered, ordered, paginated slice of splits.
func (s *Store) Query(ctx context.Context, filter splitbus.QueryFilter, orderBy order.By, pg page.Page) ([]splitbus.Split, error) {
	data := map[string]any{
		"offset":        pg.Offset(),
		"rows_per_page": pg.RowsPerPage(),
	}

	var buf bytes.Buffer
	buf.WriteString("SELECT " + columns + " FROM transaction_splits WHERE 1=1")

	applyFilter(filter, data, &buf)

	orderClause, err := orderByClause(orderBy)
	if err != nil {
		return nil, err
	}

	buf.WriteString(fmt.Sprintf(" ORDER BY %s OFFSET :offset ROWS FETCH NEXT :rows_per_page ROWS ONLY", orderClause))

	dbItems, err := sqldb.NamedQuerySlice[splitDB](ctx, s.log, s.db, buf.String(), data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return toBusSplits(dbItems), nil
}

// Count returns the number of splits matching the filter.
func (s *Store) Count(ctx context.Context, filter splitbus.QueryFilter) (int, error) {
	data := map[string]any{}

	var buf bytes.Buffer
	buf.WriteString(`SELECT COUNT(*) FROM transaction_splits WHERE 1=1`)

	applyFilter(filter, data, &buf)

	var count struct {
		Count int `db:"count"`
	}
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, buf.String(), data, &count); err != nil {
		return 0, fmt.Errorf("namedquerystruct: %w", err)
	}

	return count.Count, nil
}

// QueryByID returns the split with the given ID.
func (s *Store) QueryByID(ctx context.Context, id uuid.UUID) (splitbus.Split, error) {
	data := struct {
		ID uuid.UUID `db:"split_id"`
	}{
		ID: id,
	}

	const q = `SELECT ` + columns + ` FROM transaction_splits WHERE split_id = :split_id`

	var sp splitDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &sp); err != nil {
		return splitbus.Split{}, fmt.Errorf("namedquerystruct: %w", err)
	}

	return toBusSplit(sp), nil
}
