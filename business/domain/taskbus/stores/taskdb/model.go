package taskdb

import (
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/types/taskenergy"
	"github.com/casebrophy/planner/business/types/taskpriority"
	"github.com/casebrophy/planner/business/types/taskstatus"
)

type taskDB struct {
	ID          uuid.UUID  `db:"task_id"`
	ContextID   *uuid.UUID `db:"context_id"`
	Title       string     `db:"title"`
	Description string     `db:"description"`
	Status      string     `db:"status"`
	Priority    string     `db:"priority"`
	Energy      string     `db:"energy"`
	DurationMin *int       `db:"duration_min"`
	DueDate     *time.Time `db:"due_date"`
	ScheduledAt *time.Time `db:"scheduled_at"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
	CompletedAt *time.Time `db:"completed_at"`
}

func toDBTask(t taskbus.Task) taskDB {
	return taskDB{
		ID:          t.ID,
		ContextID:   t.ContextID,
		Title:       t.Title,
		Description: t.Description,
		Status:      t.Status.String(),
		Priority:    t.Priority.String(),
		Energy:      t.Energy.String(),
		DurationMin: t.DurationMin,
		DueDate:     t.DueDate,
		ScheduledAt: t.ScheduledAt,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		CompletedAt: t.CompletedAt,
	}
}

func toBusTask(t taskDB) taskbus.Task {
	return taskbus.Task{
		ID:          t.ID,
		ContextID:   t.ContextID,
		Title:       t.Title,
		Description: t.Description,
		Status:      taskstatus.MustParse(t.Status),
		Priority:    taskpriority.MustParse(t.Priority),
		Energy:      taskenergy.MustParse(t.Energy),
		DurationMin: t.DurationMin,
		DueDate:     t.DueDate,
		ScheduledAt: t.ScheduledAt,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		CompletedAt: t.CompletedAt,
	}
}

func toBusTasks(ts []taskDB) []taskbus.Task {
	tasks := make([]taskbus.Task, len(ts))
	for i, t := range ts {
		tasks[i] = toBusTask(t)
	}
	return tasks
}
