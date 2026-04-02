package taskdb

import (
	"context"
	"fmt"

	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DependencyStore struct {
	log *logger.Logger
	db  sqlx.ExtContext
}

func NewDependencyStore(log *logger.Logger, db *sqlx.DB) *DependencyStore {
	return &DependencyStore{log: log, db: db}
}

func (s *DependencyStore) AddDependency(ctx context.Context, dep taskbus.Dependency) error {
	const q = `
	INSERT INTO task_dependencies (task_id, depends_on_id, created_at)
	VALUES (:task_id, :depends_on_id, :created_at)`

	data := struct {
		TaskID      uuid.UUID `db:"task_id"`
		DependsOnID uuid.UUID `db:"depends_on_id"`
		CreatedAt   string    `db:"created_at"`
	}{
		TaskID:      dep.TaskID,
		DependsOnID: dep.DependsOnID,
		CreatedAt:   dep.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("inserting dependency: %w", err)
	}
	return nil
}

func (s *DependencyStore) RemoveDependency(ctx context.Context, taskID, dependsOnID uuid.UUID) error {
	const q = `
	DELETE FROM task_dependencies
	WHERE task_id = :task_id AND depends_on_id = :depends_on_id`

	data := struct {
		TaskID      uuid.UUID `db:"task_id"`
		DependsOnID uuid.UUID `db:"depends_on_id"`
	}{TaskID: taskID, DependsOnID: dependsOnID}

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("deleting dependency: %w", err)
	}
	return nil
}

func (s *DependencyStore) QueryDependencies(ctx context.Context, taskID uuid.UUID) ([]taskbus.Task, error) {
	const q = `
	SELECT t.task_id, t.context_id, t.title, t.description, t.status, t.priority,
	       t.energy, t.duration_min, t.due_date, t.scheduled_at,
	       t.expected_update_days, t.last_thread_at, t.debrief_status,
	       t.blocked_reason, t.created_at, t.updated_at, t.completed_at
	FROM tasks t
	INNER JOIN task_dependencies td ON t.task_id = td.depends_on_id
	WHERE td.task_id = :task_id
	ORDER BY t.created_at`

	data := struct {
		TaskID uuid.UUID `db:"task_id"`
	}{TaskID: taskID}

	rows, err := sqldb.NamedQuerySlice[taskDB](ctx, s.log, s.db, q, data)
	if err != nil {
		return nil, fmt.Errorf("querying dependencies: %w", err)
	}
	return toBusTasks(rows), nil
}

func (s *DependencyStore) QueryDependents(ctx context.Context, taskID uuid.UUID) ([]taskbus.Task, error) {
	const q = `
	SELECT t.task_id, t.context_id, t.title, t.description, t.status, t.priority,
	       t.energy, t.duration_min, t.due_date, t.scheduled_at,
	       t.expected_update_days, t.last_thread_at, t.debrief_status,
	       t.blocked_reason, t.created_at, t.updated_at, t.completed_at
	FROM tasks t
	INNER JOIN task_dependencies td ON t.task_id = td.task_id
	WHERE td.depends_on_id = :depends_on_id
	ORDER BY t.created_at`

	data := struct {
		DependsOnID uuid.UUID `db:"depends_on_id"`
	}{DependsOnID: taskID}

	rows, err := sqldb.NamedQuerySlice[taskDB](ctx, s.log, s.db, q, data)
	if err != nil {
		return nil, fmt.Errorf("querying dependents: %w", err)
	}
	return toBusTasks(rows), nil
}

func (s *DependencyStore) HasUnmetDependencies(ctx context.Context, taskID uuid.UUID) (bool, error) {
	const q = `
	SELECT COUNT(*) AS count
	FROM task_dependencies td
	INNER JOIN tasks t ON t.task_id = td.depends_on_id
	WHERE td.task_id = :task_id AND t.status != 'done'`

	data := struct {
		TaskID uuid.UUID `db:"task_id"`
	}{TaskID: taskID}

	var result struct {
		Count int `db:"count"`
	}
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &result); err != nil {
		return false, fmt.Errorf("checking unmet dependencies: %w", err)
	}
	return result.Count > 0, nil
}
