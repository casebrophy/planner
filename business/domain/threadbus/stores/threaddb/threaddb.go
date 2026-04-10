package threaddb

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/business/domain/threadbus"
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

func (s *Store) Create(ctx context.Context, entry threadbus.ThreadEntry) error {
	const q = `
	INSERT INTO thread_entries
		(entry_id, subject_type, subject_id, kind, content, metadata, source, source_id, sentiment, requires_action, created_at)
	VALUES
		(:entry_id, :subject_type, :subject_id, :kind, :content, :metadata, :source, :source_id, :sentiment, :requires_action, :created_at)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBThreadEntry(entry)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	// Update last_thread_at on the parent subject so staleness detection
	// sees the new activity timestamp.
	if err := s.touchSubject(ctx, entry.SubjectType, entry.SubjectID, entry.CreatedAt); err != nil {
		s.log.Error(ctx, "threaddb", "msg", "failed to touch subject last_thread_at", "error", err,
			"subject_type", entry.SubjectType, "subject_id", entry.SubjectID)
	}

	return nil
}

// touchSubject updates last_thread_at on the parent task or context.
func (s *Store) touchSubject(ctx context.Context, subjectType string, subjectID uuid.UUID, at time.Time) error {
	var q string
	switch subjectType {
	case "task":
		q = `UPDATE tasks SET last_thread_at = :at WHERE task_id = :subject_id`
	case "context":
		q = `UPDATE contexts SET last_thread_at = :at WHERE context_id = :subject_id`
	default:
		return nil
	}

	data := struct {
		At        time.Time `db:"at"`
		SubjectID uuid.UUID `db:"subject_id"`
	}{
		At:        at,
		SubjectID: subjectID,
	}

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("touch %s last_thread_at: %w", subjectType, err)
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
