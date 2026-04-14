package notedb

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/business/domain/notebus"
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

func (s *Store) Create(ctx context.Context, note notebus.Note) error {
	const q = `
	INSERT INTO notes
		(note_id, context_id, task_id, content, source, raw_input_id, created_at, updated_at, unconfirmed)
	VALUES
		(:note_id, :context_id, :task_id, :content, :source, :raw_input_id, :created_at, :updated_at, :unconfirmed)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBNote(note)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Update(ctx context.Context, note notebus.Note) error {
	const q = `
	UPDATE notes SET
		context_id = :context_id,
		task_id = :task_id,
		content = :content,
		source = :source,
		updated_at = :updated_at,
		unconfirmed = :unconfirmed
	WHERE
		note_id = :note_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBNote(note)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Delete(ctx context.Context, note notebus.Note) error {
	data := struct {
		ID uuid.UUID `db:"note_id"`
	}{
		ID: note.ID,
	}

	const q = `DELETE FROM notes WHERE note_id = :note_id`

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

	const q = `DELETE FROM notes WHERE raw_input_id = :raw_input_id AND unconfirmed = true`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Query(ctx context.Context, filter notebus.QueryFilter, orderBy order.By, pg page.Page) ([]notebus.Note, error) {
	data := map[string]any{
		"offset":        pg.Offset(),
		"rows_per_page": pg.RowsPerPage(),
	}

	var buf bytes.Buffer
	buf.WriteString(`SELECT note_id, context_id, task_id, content, source, raw_input_id, created_at, updated_at FROM notes WHERE 1=1`)

	applyFilter(filter, data, &buf)

	orderClause, err := orderByClause(orderBy)
	if err != nil {
		return nil, err
	}

	buf.WriteString(fmt.Sprintf(" ORDER BY %s OFFSET :offset ROWS FETCH NEXT :rows_per_page ROWS ONLY", orderClause))

	dbNotes, err := sqldb.NamedQuerySlice[noteDB](ctx, s.log, s.db, buf.String(), data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return toBusNotes(dbNotes), nil
}

func (s *Store) Count(ctx context.Context, filter notebus.QueryFilter) (int, error) {
	data := map[string]any{}

	var buf bytes.Buffer
	buf.WriteString(`SELECT COUNT(*) FROM notes WHERE 1=1`)

	applyFilter(filter, data, &buf)

	var count struct {
		Count int `db:"count"`
	}
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, buf.String(), data, &count); err != nil {
		return 0, fmt.Errorf("namedquerystruct: %w", err)
	}

	return count.Count, nil
}

func (s *Store) QueryByID(ctx context.Context, id uuid.UUID) (notebus.Note, error) {
	data := struct {
		ID uuid.UUID `db:"note_id"`
	}{
		ID: id,
	}

	const q = `SELECT note_id, context_id, task_id, content, source, raw_input_id, created_at, updated_at FROM notes WHERE note_id = :note_id`

	var n noteDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &n); err != nil {
		return notebus.Note{}, fmt.Errorf("namedquerystruct: %w", err)
	}

	return toBusNote(n), nil
}
