package taskdb

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

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
		(task_id, context_id, title, description, status, priority, energy, duration_min, due_date, scheduled_at, expected_update_days, last_thread_at, blocked_reason, debrief_status, created_at, updated_at, completed_at)
	VALUES
		(:task_id, :context_id, :title, :description, :status, :priority, :energy, :duration_min, :due_date, :scheduled_at, :expected_update_days, :last_thread_at, :blocked_reason, :debrief_status, :created_at, :updated_at, :completed_at)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBTask(task)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Update(ctx context.Context, task taskbus.Task) error {
	const q = `
	UPDATE tasks SET
		context_id = :context_id,
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
		blocked_reason = :blocked_reason,
		debrief_status = :debrief_status,
		updated_at = :updated_at,
		completed_at = :completed_at
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

func (s *Store) Query(ctx context.Context, filter taskbus.QueryFilter, orderBy order.By, pg page.Page) ([]taskbus.Task, error) {
	data := map[string]any{
		"offset":        pg.Offset(),
		"rows_per_page": pg.RowsPerPage(),
	}

	var buf bytes.Buffer
	buf.WriteString(`SELECT task_id, context_id, title, description, status, priority, energy, duration_min, due_date, scheduled_at, expected_update_days, last_thread_at, blocked_reason, debrief_status, created_at, updated_at, completed_at FROM tasks WHERE 1=1`)

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

	const q = `SELECT task_id, context_id, title, description, status, priority, energy, duration_min, due_date, scheduled_at, expected_update_days, last_thread_at, blocked_reason, debrief_status, created_at, updated_at, completed_at FROM tasks WHERE task_id = :task_id`

	var t taskDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &t); err != nil {
		return taskbus.Task{}, fmt.Errorf("namedquerystruct: %w", err)
	}

	return toBusTask(t), nil
}

func (s *Store) AddDependency(ctx context.Context, taskID uuid.UUID, dependsOnID uuid.UUID) error {
	data := struct {
		TaskID      uuid.UUID `db:"task_id"`
		DependsOnID uuid.UUID `db:"depends_on_id"`
	}{
		TaskID:      taskID,
		DependsOnID: dependsOnID,
	}

	const q = `
	INSERT INTO task_dependencies (task_id, depends_on_id)
	VALUES (:task_id, :depends_on_id)
	ON CONFLICT DO NOTHING`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) RemoveDependency(ctx context.Context, taskID uuid.UUID, dependsOnID uuid.UUID) error {
	data := struct {
		TaskID      uuid.UUID `db:"task_id"`
		DependsOnID uuid.UUID `db:"depends_on_id"`
	}{
		TaskID:      taskID,
		DependsOnID: dependsOnID,
	}

	const q = `DELETE FROM task_dependencies WHERE task_id = :task_id AND depends_on_id = :depends_on_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) QueryDependencies(ctx context.Context, taskID uuid.UUID) ([]taskbus.Task, error) {
	data := struct {
		TaskID uuid.UUID `db:"task_id"`
	}{
		TaskID: taskID,
	}

	const q = `
	SELECT t.task_id, t.context_id, t.title, t.description, t.status, t.priority, t.energy,
		t.duration_min, t.due_date, t.scheduled_at, t.expected_update_days, t.last_thread_at,
		t.blocked_reason, t.debrief_status, t.created_at, t.updated_at, t.completed_at
	FROM tasks t
	INNER JOIN task_dependencies td ON t.task_id = td.depends_on_id
	WHERE td.task_id = :task_id`

	dbTasks, err := sqldb.NamedQuerySlice[taskDB](ctx, s.log, s.db, q, data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return toBusTasks(dbTasks), nil
}

func (s *Store) QueryDependents(ctx context.Context, taskID uuid.UUID) ([]taskbus.Task, error) {
	data := struct {
		TaskID uuid.UUID `db:"task_id"`
	}{
		TaskID: taskID,
	}

	const q = `
	SELECT t.task_id, t.context_id, t.title, t.description, t.status, t.priority, t.energy,
		t.duration_min, t.due_date, t.scheduled_at, t.expected_update_days, t.last_thread_at,
		t.blocked_reason, t.debrief_status, t.created_at, t.updated_at, t.completed_at
	FROM tasks t
	INNER JOIN task_dependencies td ON t.task_id = td.task_id
	WHERE td.depends_on_id = :task_id`

	dbTasks, err := sqldb.NamedQuerySlice[taskDB](ctx, s.log, s.db, q, data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return toBusTasks(dbTasks), nil
}
