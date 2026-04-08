package contextdb

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/business/domain/contextbus"
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

// Context operations

func (s *Store) Create(ctx context.Context, c contextbus.Context) error {
	const q = `
	INSERT INTO contexts
		(context_id, title, description, kind, status, summary, last_event, last_thread_at, debrief_status, outcome, parent_context_id, created_at, updated_at)
	VALUES
		(:context_id, :title, :description, :kind, :status, :summary, :last_event, :last_thread_at, :debrief_status, :outcome, :parent_context_id, :created_at, :updated_at)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBContext(c)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Update(ctx context.Context, c contextbus.Context) error {
	const q = `
	UPDATE contexts SET
		title = :title,
		description = :description,
		kind = :kind,
		status = :status,
		summary = :summary,
		last_event = :last_event,
		last_thread_at = :last_thread_at,
		debrief_status = :debrief_status,
		outcome = :outcome,
		parent_context_id = :parent_context_id,
		updated_at = :updated_at
	WHERE
		context_id = :context_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBContext(c)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Delete(ctx context.Context, c contextbus.Context) error {
	data := struct {
		ID uuid.UUID `db:"context_id"`
	}{
		ID: c.ID,
	}

	const q = `DELETE FROM contexts WHERE context_id = :context_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Query(ctx context.Context, filter contextbus.QueryFilter, orderBy order.By, pg page.Page) ([]contextbus.Context, error) {
	data := map[string]any{
		"offset":        pg.Offset(),
		"rows_per_page": pg.RowsPerPage(),
	}

	var buf bytes.Buffer
	buf.WriteString(`SELECT context_id, title, description, kind, status, summary, last_event, last_thread_at, debrief_status, outcome, parent_context_id, created_at, updated_at FROM contexts WHERE 1=1`)

	applyFilter(filter, data, &buf)

	orderClause, err := orderByClause(orderBy)
	if err != nil {
		return nil, err
	}

	buf.WriteString(fmt.Sprintf(" ORDER BY %s OFFSET :offset ROWS FETCH NEXT :rows_per_page ROWS ONLY", orderClause))

	dbContexts, err := sqldb.NamedQuerySlice[contextDB](ctx, s.log, s.db, buf.String(), data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return toBusContexts(dbContexts), nil
}

func (s *Store) Count(ctx context.Context, filter contextbus.QueryFilter) (int, error) {
	data := map[string]any{}

	var buf bytes.Buffer
	buf.WriteString(`SELECT COUNT(*) FROM contexts WHERE 1=1`)

	applyFilter(filter, data, &buf)

	var count struct {
		Count int `db:"count"`
	}
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, buf.String(), data, &count); err != nil {
		return 0, fmt.Errorf("namedquerystruct: %w", err)
	}

	return count.Count, nil
}

func (s *Store) QueryByID(ctx context.Context, id uuid.UUID) (contextbus.Context, error) {
	data := struct {
		ID uuid.UUID `db:"context_id"`
	}{
		ID: id,
	}

	const q = `SELECT context_id, title, description, kind, status, summary, last_event, last_thread_at, debrief_status, outcome, parent_context_id, created_at, updated_at FROM contexts WHERE context_id = :context_id`

	var c contextDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &c); err != nil {
		return contextbus.Context{}, fmt.Errorf("namedquerystruct: %w", err)
	}

	return toBusContext(c), nil
}

