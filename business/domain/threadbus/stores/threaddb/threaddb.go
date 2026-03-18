package threaddb

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/business/domain/threadbus"
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

func (s *Store) Create(ctx context.Context, entry threadbus.ThreadEntry) error {
	const q = `
	INSERT INTO thread_entries
		(entry_id, subject_type, subject_id, kind, content, metadata, source, source_id, sentiment, requires_action, created_at)
	VALUES
		(:entry_id, :subject_type, :subject_id, :kind, :content, :metadata, :source, :source_id, :sentiment, :requires_action, :created_at)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBThreadEntry(entry)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Query(ctx context.Context, filter threadbus.QueryFilter, orderBy order.By, pg page.Page) ([]threadbus.ThreadEntry, error) {
	data := map[string]any{
		"offset":        pg.Offset(),
		"rows_per_page": pg.RowsPerPage(),
	}

	var buf bytes.Buffer
	buf.WriteString(`SELECT entry_id, subject_type, subject_id, kind, content, metadata, source, source_id, sentiment, requires_action, created_at FROM thread_entries WHERE 1=1`)

	applyFilter(filter, data, &buf)

	orderClause, err := orderByClause(orderBy)
	if err != nil {
		return nil, err
	}

	buf.WriteString(fmt.Sprintf(" ORDER BY %s OFFSET :offset ROWS FETCH NEXT :rows_per_page ROWS ONLY", orderClause))

	dbEntries, err := sqldb.NamedQuerySlice[threadEntryDB](ctx, s.log, s.db, buf.String(), data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return toBusThreadEntries(dbEntries), nil
}

func (s *Store) Count(ctx context.Context, filter threadbus.QueryFilter) (int, error) {
	data := map[string]any{}

	var buf bytes.Buffer
	buf.WriteString(`SELECT COUNT(*) FROM thread_entries WHERE 1=1`)

	applyFilter(filter, data, &buf)

	var count struct {
		Count int `db:"count"`
	}
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, buf.String(), data, &count); err != nil {
		return 0, fmt.Errorf("namedquerystruct: %w", err)
	}

	return count.Count, nil
}

func (s *Store) QueryByID(ctx context.Context, id uuid.UUID) (threadbus.ThreadEntry, error) {
	data := struct {
		ID uuid.UUID `db:"entry_id"`
	}{
		ID: id,
	}

	const q = `SELECT entry_id, subject_type, subject_id, kind, content, metadata, source, source_id, sentiment, requires_action, created_at FROM thread_entries WHERE entry_id = :entry_id`

	var e threadEntryDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &e); err != nil {
		return threadbus.ThreadEntry{}, fmt.Errorf("namedquerystruct: %w", err)
	}

	return toBusThreadEntry(e), nil
}
