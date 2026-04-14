package rawinputdb

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/business/domain/rawinputbus"
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

func (s *Store) Create(ctx context.Context, ri rawinputbus.RawInput) error {
	const q = `
	INSERT INTO raw_inputs
		(raw_input_id, source_type, status, raw_content, processed_at, error, retry_count, next_retry_at, max_retries, result, created_at, user_correction)
	VALUES
		(:raw_input_id, :source_type, :status, :raw_content, :processed_at, :error, :retry_count, :next_retry_at, :max_retries, :result, :created_at, :user_correction)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBRawInput(ri)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Update(ctx context.Context, ri rawinputbus.RawInput) error {
	const q = `
	UPDATE raw_inputs SET
		status = :status,
		processed_at = :processed_at,
		error = :error,
		retry_count = :retry_count,
		next_retry_at = :next_retry_at,
		result = :result,
		user_correction = :user_correction
	WHERE
		raw_input_id = :raw_input_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBRawInput(ri)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Query(ctx context.Context, filter rawinputbus.QueryFilter, orderBy order.By, pg page.Page) ([]rawinputbus.RawInput, error) {
	data := map[string]any{
		"offset":        pg.Offset(),
		"rows_per_page": pg.RowsPerPage(),
	}

	var buf bytes.Buffer
	buf.WriteString(`SELECT raw_input_id, source_type, status, raw_content, processed_at, error, retry_count, next_retry_at, max_retries, result, created_at, user_correction FROM raw_inputs WHERE 1=1`)

	applyFilter(filter, data, &buf)

	orderClause, err := orderByClause(orderBy)
	if err != nil {
		return nil, err
	}

	buf.WriteString(fmt.Sprintf(" ORDER BY %s OFFSET :offset ROWS FETCH NEXT :rows_per_page ROWS ONLY", orderClause))

	dbItems, err := sqldb.NamedQuerySlice[rawInputDB](ctx, s.log, s.db, buf.String(), data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return toBusRawInputs(dbItems), nil
}

func (s *Store) Count(ctx context.Context, filter rawinputbus.QueryFilter) (int, error) {
	data := map[string]any{}

	var buf bytes.Buffer
	buf.WriteString(`SELECT COUNT(*) FROM raw_inputs WHERE 1=1`)

	applyFilter(filter, data, &buf)

	var count struct {
		Count int `db:"count"`
	}
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, buf.String(), data, &count); err != nil {
		return 0, fmt.Errorf("namedquerystruct: %w", err)
	}

	return count.Count, nil
}

func (s *Store) QueryByID(ctx context.Context, id uuid.UUID) (rawinputbus.RawInput, error) {
	data := struct {
		ID uuid.UUID `db:"raw_input_id"`
	}{
		ID: id,
	}

	const q = `SELECT raw_input_id, source_type, status, raw_content, processed_at, error, retry_count, next_retry_at, max_retries, result, created_at, user_correction FROM raw_inputs WHERE raw_input_id = :raw_input_id`

	var ri rawInputDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &ri); err != nil {
		return rawinputbus.RawInput{}, fmt.Errorf("namedquerystruct: %w", err)
	}

	return toBusRawInput(ri), nil
}

func (s *Store) QueryRetryable(ctx context.Context, limit int) ([]rawinputbus.RawInput, error) {
	data := map[string]any{
		"limit": limit,
	}

	const q = `
	SELECT raw_input_id, source_type, status, raw_content, processed_at, error, retry_count, next_retry_at, max_retries, result, created_at, user_correction
	FROM raw_inputs
	WHERE status = 'pending'
	  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
	ORDER BY created_at ASC
	FETCH NEXT :limit ROWS ONLY`

	dbItems, err := sqldb.NamedQuerySlice[rawInputDB](ctx, s.log, s.db, q, data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}
	return toBusRawInputs(dbItems), nil
}

func (s *Store) ResetForReprocess(ctx context.Context, id uuid.UUID) (rawinputbus.RawInput, error) {
	data := struct {
		ID uuid.UUID `db:"raw_input_id"`
	}{ID: id}

	const q = `
	UPDATE raw_inputs
	SET status = 'pending', retry_count = 0, next_retry_at = NULL, error = NULL
	WHERE raw_input_id = :raw_input_id
	RETURNING raw_input_id, source_type, status, raw_content, processed_at, error, retry_count, next_retry_at, max_retries, result, created_at, user_correction`

	var ri rawInputDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &ri); err != nil {
		return rawinputbus.RawInput{}, fmt.Errorf("namedquerystruct: %w", err)
	}
	return toBusRawInput(ri), nil
}
