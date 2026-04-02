package timeblockdb

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/business/domain/timeblockbus"
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

func (s *Store) Create(ctx context.Context, block timeblockbus.TimeBlock) error {
	const q = `
	INSERT INTO time_blocks
		(block_id, task_id, starts_at, ends_at, confirmed, created_at, updated_at)
	VALUES
		(:block_id, :task_id, :starts_at, :ends_at, :confirmed, :created_at, :updated_at)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBTimeBlock(block)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Update(ctx context.Context, block timeblockbus.TimeBlock) error {
	const q = `
	UPDATE time_blocks SET
		starts_at = :starts_at,
		ends_at = :ends_at,
		confirmed = :confirmed,
		updated_at = :updated_at
	WHERE
		block_id = :block_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBTimeBlock(block)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Delete(ctx context.Context, block timeblockbus.TimeBlock) error {
	data := struct {
		ID uuid.UUID `db:"block_id"`
	}{
		ID: block.ID,
	}

	const q = `DELETE FROM time_blocks WHERE block_id = :block_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Query(ctx context.Context, filter timeblockbus.QueryFilter, orderBy order.By, pg page.Page) ([]timeblockbus.TimeBlock, error) {
	data := map[string]any{
		"offset":        pg.Offset(),
		"rows_per_page": pg.RowsPerPage(),
	}

	var buf bytes.Buffer
	buf.WriteString(`SELECT block_id, task_id, starts_at, ends_at, confirmed, created_at, updated_at FROM time_blocks WHERE 1=1`)

	applyFilter(filter, data, &buf)

	orderClause, err := orderByClause(orderBy)
	if err != nil {
		return nil, err
	}

	buf.WriteString(fmt.Sprintf(" ORDER BY %s OFFSET :offset ROWS FETCH NEXT :rows_per_page ROWS ONLY", orderClause))

	dbBlocks, err := sqldb.NamedQuerySlice[timeBlockDB](ctx, s.log, s.db, buf.String(), data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return toBusTimeBlocks(dbBlocks), nil
}

func (s *Store) Count(ctx context.Context, filter timeblockbus.QueryFilter) (int, error) {
	data := map[string]any{}

	var buf bytes.Buffer
	buf.WriteString(`SELECT COUNT(*) FROM time_blocks WHERE 1=1`)

	applyFilter(filter, data, &buf)

	var count struct {
		Count int `db:"count"`
	}
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, buf.String(), data, &count); err != nil {
		return 0, fmt.Errorf("namedquerystruct: %w", err)
	}

	return count.Count, nil
}

func (s *Store) QueryByID(ctx context.Context, id uuid.UUID) (timeblockbus.TimeBlock, error) {
	data := struct {
		ID uuid.UUID `db:"block_id"`
	}{
		ID: id,
	}

	const q = `SELECT block_id, task_id, starts_at, ends_at, confirmed, created_at, updated_at FROM time_blocks WHERE block_id = :block_id`

	var tb timeBlockDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &tb); err != nil {
		return timeblockbus.TimeBlock{}, fmt.Errorf("namedquerystruct: %w", err)
	}

	return toBusTimeBlock(tb), nil
}
