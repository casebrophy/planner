package eventdb

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/business/domain/eventbus"
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

func (s *Store) Create(ctx context.Context, event eventbus.Event) error {
	const q = `
	INSERT INTO events
		(event_id, context_id, title, description, location, starts_at, ends_at, all_day, raw_input_id, created_at, updated_at, unconfirmed)
	VALUES
		(:event_id, :context_id, :title, :description, :location, :starts_at, :ends_at, :all_day, :raw_input_id, :created_at, :updated_at, :unconfirmed)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBEvent(event)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Update(ctx context.Context, event eventbus.Event) error {
	const q = `
	UPDATE events SET
		context_id = :context_id,
		title = :title,
		description = :description,
		location = :location,
		starts_at = :starts_at,
		ends_at = :ends_at,
		all_day = :all_day,
		updated_at = :updated_at,
		unconfirmed = :unconfirmed
	WHERE
		event_id = :event_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBEvent(event)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Delete(ctx context.Context, event eventbus.Event) error {
	data := struct {
		ID uuid.UUID `db:"event_id"`
	}{
		ID: event.ID,
	}

	const q = `DELETE FROM events WHERE event_id = :event_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) DeleteByRawInputUnconfirmed(ctx context.Context, rawInputID uuid.UUID) error {
	data := struct {
		RawInputID uuid.UUID `db:"raw_input_id"`
	}{
		RawInputID: rawInputID,
	}

	const q = `DELETE FROM events WHERE raw_input_id = :raw_input_id AND unconfirmed = true`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Query(ctx context.Context, filter eventbus.QueryFilter, orderBy order.By, pg page.Page) ([]eventbus.Event, error) {
	data := map[string]any{
		"offset":        pg.Offset(),
		"rows_per_page": pg.RowsPerPage(),
	}

	var buf bytes.Buffer
	buf.WriteString(`SELECT event_id, context_id, title, description, location, starts_at, ends_at, all_day, raw_input_id, created_at, updated_at, unconfirmed FROM events WHERE 1=1`)

	applyFilter(filter, data, &buf)

	orderClause, err := orderByClause(orderBy)
	if err != nil {
		return nil, err
	}

	buf.WriteString(fmt.Sprintf(" ORDER BY %s OFFSET :offset ROWS FETCH NEXT :rows_per_page ROWS ONLY", orderClause))

	dbEvents, err := sqldb.NamedQuerySlice[eventDB](ctx, s.log, s.db, buf.String(), data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return toBusEvents(dbEvents), nil
}

func (s *Store) Count(ctx context.Context, filter eventbus.QueryFilter) (int, error) {
	data := map[string]any{}

	var buf bytes.Buffer
	buf.WriteString(`SELECT COUNT(*) FROM events WHERE 1=1`)

	applyFilter(filter, data, &buf)

	var count struct {
		Count int `db:"count"`
	}
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, buf.String(), data, &count); err != nil {
		return 0, fmt.Errorf("namedquerystruct: %w", err)
	}

	return count.Count, nil
}

func (s *Store) QueryByID(ctx context.Context, id uuid.UUID) (eventbus.Event, error) {
	data := struct {
		ID uuid.UUID `db:"event_id"`
	}{
		ID: id,
	}

	const q = `SELECT event_id, context_id, title, description, location, starts_at, ends_at, all_day, raw_input_id, created_at, updated_at, unconfirmed FROM events WHERE event_id = :event_id`

	var e eventDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &e); err != nil {
		return eventbus.Event{}, fmt.Errorf("namedquerystruct: %w", err)
	}

	return toBusEvent(e), nil
}
