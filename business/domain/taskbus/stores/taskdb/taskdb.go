package taskdb

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/casebrophy/planner/business/domain/taskbus"
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

func (s *Store) Create(ctx context.Context, task taskbus.Task) error {
	const q = `
	INSERT INTO tasks
		(task_id, context_id, raw_input_id, title, description, status, priority, energy, duration_min, due_date, scheduled_at, expected_update_days, last_thread_at, debrief_status, blocked_reason, created_at, updated_at, completed_at, recurrence_rule, recurrence_parent_id, track_outcome, unconfirmed)
	VALUES
		(:task_id, :context_id, :raw_input_id, :title, :description, :status, :priority, :energy, :duration_min, :due_date, :scheduled_at, :expected_update_days, :last_thread_at, :debrief_status, :blocked_reason, :created_at, :updated_at, :completed_at, :recurrence_rule, :recurrence_parent_id, :track_outcome, :unconfirmed)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBTask(task)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Update(ctx context.Context, task taskbus.Task) error {
	const q = `
	UPDATE tasks SET
		context_id = :context_id,
		raw_input_id = :raw_input_id,
		title = :title,
		description = :description,
		status = :status,
		priority = :priority,
		energy = :energy,
		duration_min = :duration_min,
		due_date = :due_date,
		scheduled_at = :scheduled_at,
		expected_update_days = :expected_update_days,
		last_thread_at = :last_thread_at,
		debrief_status = :debrief_status,
		blocked_reason = :blocked_reason,
		updated_at = :updated_at,
		completed_at = :completed_at,
		recurrence_rule = :recurrence_rule,
		recurrence_parent_id = :recurrence_parent_id,
		track_outcome = :track_outcome,
		unconfirmed = :unconfirmed
	WHERE
		task_id = :task_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBTask(task)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Delete(ctx context.Context, task taskbus.Task) error {
	data := struct {
		ID uuid.UUID `db:"task_id"`
	}{
		ID: task.ID,
	}

	const q = `DELETE FROM tasks WHERE task_id = :task_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) DeleteBatch(ctx context.Context, ids []uuid.UUID) error {
	const q = `DELETE FROM tasks WHERE task_id = ANY($1)`

	if _, err := s.db.ExecContext(ctx, q, pq.Array(ids)); err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	return nil
}

func (s *Store) DeleteByRawInputUnconfirmed(ctx context.Context, rawInputID uuid.UUID) error {
	data := struct {
		RawInputID uuid.UUID `db:"raw_input_id"`
	}{
		RawInputID: rawInputID,
	}

	const q = `DELETE FROM tasks WHERE raw_input_id = :raw_input_id AND unconfirmed = true`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Query(ctx context.Context, filter taskbus.QueryFilter, orderBy order.By, pg page.Page) ([]taskbus.Task, error) {
	data := map[string]any{
		"offset":        pg.Offset(),
		"rows_per_page": pg.RowsPerPage(),
	}

	var buf bytes.Buffer
	buf.WriteString(`SELECT task_id, context_id, raw_input_id, title, description, status, priority, energy, duration_min, due_date, scheduled_at, expected_update_days, last_thread_at, debrief_status, blocked_reason, created_at, updated_at, completed_at, recurrence_rule, recurrence_parent_id, track_outcome, unconfirmed FROM tasks WHERE 1=1`)

	applyFilter(filter, data, &buf)

	orderClause, err := orderByClause(orderBy)
	if err != nil {
		return nil, err
	}

	buf.WriteString(fmt.Sprintf(" ORDER BY %s OFFSET :offset ROWS FETCH NEXT :rows_per_page ROWS ONLY", orderClause))

	dbTasks, err := sqldb.NamedQuerySlice[taskDB](ctx, s.log, s.db, buf.String(), data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return toBusTasks(dbTasks), nil
}

func (s *Store) Count(ctx context.Context, filter taskbus.QueryFilter) (int, error) {
	data := map[string]any{}

	var buf bytes.Buffer
	buf.WriteString(`SELECT COUNT(*) FROM tasks WHERE 1=1`)

	applyFilter(filter, data, &buf)

	var count struct {
		Count int `db:"count"`
	}
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, buf.String(), data, &count); err != nil {
		return 0, fmt.Errorf("namedquerystruct: %w", err)
	}

	return count.Count, nil
}

func (s *Store) QueryByID(ctx context.Context, id uuid.UUID) (taskbus.Task, error) {
	data := struct {
		ID uuid.UUID `db:"task_id"`
	}{
		ID: id,
	}

	const q = `SELECT task_id, context_id, raw_input_id, title, description, status, priority, energy, duration_min, due_date, scheduled_at, expected_update_days, last_thread_at, debrief_status, blocked_reason, created_at, updated_at, completed_at, recurrence_rule, recurrence_parent_id, track_outcome, unconfirmed FROM tasks WHERE task_id = :task_id`

	var t taskDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &t); err != nil {
		return taskbus.Task{}, fmt.Errorf("namedquerystruct: %w", err)
	}

	return toBusTask(t), nil
}

func (s *Store) DismissTasksByContext(ctx context.Context, contextID uuid.UUID) (int, error) {
	const q = `
	UPDATE tasks
	SET status = 'dismissed', updated_at = :updated_at
	WHERE context_id = :context_id AND status IN ('open', 'blocked')`

	data := struct {
		ContextID uuid.UUID `db:"context_id"`
		UpdatedAt time.Time `db:"updated_at"`
	}{
		ContextID: contextID,
		UpdatedAt: time.Now().UTC(),
	}

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return 0, fmt.Errorf("dismissing tasks by context: %w", err)
	}

	return 0, nil
}

func (s *Store) ResetByContext(ctx context.Context, contextID uuid.UUID) error {
	const q = `
	UPDATE tasks
	SET status = 'open', completed_at = NULL, updated_at = :updated_at
	WHERE context_id = :context_id AND status = 'done'`

	data := struct {
		ContextID uuid.UUID `db:"context_id"`
		UpdatedAt time.Time `db:"updated_at"`
	}{
		ContextID: contextID,
		UpdatedAt: time.Now().UTC(),
	}

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("reset by context: %w", err)
	}

	return nil
}
