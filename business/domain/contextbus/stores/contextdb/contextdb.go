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
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/foundation/sqldb"
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
		(context_id, title, description, status, summary, last_event, last_thread_at, debrief_status, outcome, created_at, updated_at)
	VALUES
		(:context_id, :title, :description, :status, :summary, :last_event, :last_thread_at, :debrief_status, :outcome, :created_at, :updated_at)`

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
		status = :status,
		summary = :summary,
		last_event = :last_event,
		last_thread_at = :last_thread_at,
		debrief_status = :debrief_status,
		outcome = :outcome,
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
	buf.WriteString(`SELECT context_id, title, description, status, summary, last_event, last_thread_at, debrief_status, outcome, created_at, updated_at FROM contexts WHERE 1=1`)

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

	const q = `SELECT context_id, title, description, status, summary, last_event, last_thread_at, debrief_status, outcome, created_at, updated_at FROM contexts WHERE context_id = :context_id`

	var c contextDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &c); err != nil {
		return contextbus.Context{}, fmt.Errorf("namedquerystruct: %w", err)
	}

	return toBusContext(c), nil
}

// Event operations

func (s *Store) CreateEvent(ctx context.Context, e contextbus.Event) error {
	const q = `
	INSERT INTO context_events
		(event_id, context_id, kind, content, metadata, source_id, created_at)
	VALUES
		(:event_id, :context_id, :kind, :content, :metadata, :source_id, :created_at)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBEvent(e)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) QueryEvents(ctx context.Context, contextID uuid.UUID, pg page.Page) ([]contextbus.Event, error) {
	data := map[string]any{
		"context_id":    contextID,
		"offset":        pg.Offset(),
		"rows_per_page": pg.RowsPerPage(),
	}

	const q = `
	SELECT event_id, context_id, kind, content, metadata, source_id, created_at
	FROM context_events
	WHERE context_id = :context_id
	ORDER BY created_at DESC
	OFFSET :offset ROWS FETCH NEXT :rows_per_page ROWS ONLY`

	dbEvents, err := sqldb.NamedQuerySlice[eventDB](ctx, s.log, s.db, q, data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return toBusEvents(dbEvents), nil
}

func (s *Store) CountEvents(ctx context.Context, contextID uuid.UUID) (int, error) {
	data := struct {
		ContextID uuid.UUID `db:"context_id"`
	}{
		ContextID: contextID,
	}

	const q = `SELECT COUNT(*) FROM context_events WHERE context_id = :context_id`

	var count struct {
		Count int `db:"count"`
	}
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &count); err != nil {
		return 0, fmt.Errorf("namedquerystruct: %w", err)
	}

	return count.Count, nil
}
