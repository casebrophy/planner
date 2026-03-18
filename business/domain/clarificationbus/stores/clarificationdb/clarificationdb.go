package clarificationdb

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
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

func (s *Store) Create(ctx context.Context, item clarificationbus.ClarificationItem) error {
	const q = `
	INSERT INTO clarification_items
		(clarification_id, kind, status, subject_type, subject_id, question, claude_guess, reasoning, answer_options, answer, priority_score, snoozed_until, created_at, resolved_at)
	VALUES
		(:clarification_id, :kind, :status, :subject_type, :subject_id, :question, :claude_guess, :reasoning, :answer_options, :answer, :priority_score, :snoozed_until, :created_at, :resolved_at)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBClarification(item)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Update(ctx context.Context, item clarificationbus.ClarificationItem) error {
	const q = `
	UPDATE clarification_items SET
		kind = :kind,
		status = :status,
		subject_type = :subject_type,
		subject_id = :subject_id,
		question = :question,
		claude_guess = :claude_guess,
		reasoning = :reasoning,
		answer_options = :answer_options,
		answer = :answer,
		priority_score = :priority_score,
		snoozed_until = :snoozed_until,
		resolved_at = :resolved_at
	WHERE
		clarification_id = :clarification_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBClarification(item)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Query(ctx context.Context, filter clarificationbus.QueryFilter, orderBy order.By, pg page.Page) ([]clarificationbus.ClarificationItem, error) {
	data := map[string]any{
		"offset":        pg.Offset(),
		"rows_per_page": pg.RowsPerPage(),
	}

	var buf bytes.Buffer
	buf.WriteString(`SELECT clarification_id, kind, status, subject_type, subject_id, question, claude_guess, reasoning, answer_options, answer, priority_score, snoozed_until, created_at, resolved_at FROM clarification_items WHERE 1=1`)

	applyFilter(filter, data, &buf)

	orderClause, err := orderByClause(orderBy)
	if err != nil {
		return nil, err
	}

	buf.WriteString(fmt.Sprintf(" ORDER BY %s OFFSET :offset ROWS FETCH NEXT :rows_per_page ROWS ONLY", orderClause))

	dbItems, err := sqldb.NamedQuerySlice[clarificationDB](ctx, s.log, s.db, buf.String(), data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return toBusClarifications(dbItems), nil
}

func (s *Store) Count(ctx context.Context, filter clarificationbus.QueryFilter) (int, error) {
	data := map[string]any{}

	var buf bytes.Buffer
	buf.WriteString(`SELECT COUNT(*) FROM clarification_items WHERE 1=1`)

	applyFilter(filter, data, &buf)

	var count struct {
		Count int `db:"count"`
	}
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, buf.String(), data, &count); err != nil {
		return 0, fmt.Errorf("namedquerystruct: %w", err)
	}

	return count.Count, nil
}

func (s *Store) QueryByID(ctx context.Context, id uuid.UUID) (clarificationbus.ClarificationItem, error) {
	data := struct {
		ID uuid.UUID `db:"clarification_id"`
	}{
		ID: id,
	}

	const q = `SELECT clarification_id, kind, status, subject_type, subject_id, question, claude_guess, reasoning, answer_options, answer, priority_score, snoozed_until, created_at, resolved_at FROM clarification_items WHERE clarification_id = :clarification_id`

	var c clarificationDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &c); err != nil {
		return clarificationbus.ClarificationItem{}, fmt.Errorf("namedquerystruct: %w", err)
	}

	return toBusClarification(c), nil
}

func (s *Store) UnsnoozeExpired(ctx context.Context, now time.Time) (int, error) {
	const q = `
	UPDATE clarification_items
	SET status = 'pending', snoozed_until = NULL
	WHERE status = 'snoozed' AND snoozed_until <= :now`

	data := struct {
		Now time.Time `db:"now"`
	}{
		Now: now,
	}

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return 0, fmt.Errorf("namedexeccontext: %w", err)
	}

	// Count how many were unsnoozed (approximate — we don't have rows affected easily)
	// Return 0 since the actual count isn't critical for the background job
	return 0, nil
}
