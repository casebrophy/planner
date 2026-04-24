package reclassifybus

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/business/domain/classificationcorrectionbus"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/types/taskenergy"
	"github.com/casebrophy/planner/business/types/taskpriority"
	"github.com/casebrophy/planner/business/types/taskstatus"
	"github.com/casebrophy/planner/foundation/logger"
)

type Bus struct {
	log  *logger.Logger
	task *taskbus.Business
	note *notebus.Business
	corr *classificationcorrectionbus.Business
	db   *sqlx.DB
}

func NewBusiness(log *logger.Logger, task *taskbus.Business, note *notebus.Business, corr *classificationcorrectionbus.Business, db *sqlx.DB) *Bus {
	return &Bus{
		log:  log,
		task: task,
		note: note,
		corr: corr,
		db:   db,
	}
}

// TaskToNote converts a task to a note. Fails if task has recurrence, dependents, or is scheduled in today's plan.
// TODO: feedback loop to ingest prompt tuning (see .docs/superpowers/plans/2026-04-22-user-reclassification.md)
func (b *Bus) TaskToNote(ctx context.Context, taskID uuid.UUID) (notebus.Note, error) {
	// Fetch the task
	t, err := b.task.QueryByID(ctx, taskID)
	if err != nil {
		return notebus.Note{}, fmt.Errorf("fetch task: %w", err)
	}

	// Preflight checks
	if err := b.preflightTaskToNote(ctx, t); err != nil {
		return notebus.Note{}, err
	}

	// Start transaction
	tx, err := b.db.BeginTxx(ctx, nil)
	if err != nil {
		return notebus.Note{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Create note with combined title+description as content
	content := t.Title
	if t.Description != "" {
		content = t.Title + "\n\n" + t.Description
	}

	newNote := notebus.Note{
		ID:        uuid.New(),
		ContextID: t.ContextID,
		TaskID:    nil,
		Content:   content,
		Source:    "reclassified_from_task",
		RawInputID: t.RawInputID,
		Unconfirmed: t.Unconfirmed,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}

	if err := b.note.CreateWithTx(ctx, tx, newNote); err != nil {
		return notebus.Note{}, fmt.Errorf("create note: %w", err)
	}

	// Copy task_tags to note_tags
	if err := b.copyTaskTagsToNoteTags(ctx, tx, t.ID, newNote.ID); err != nil {
		return notebus.Note{}, fmt.Errorf("copy tags: %w", err)
	}

	// Record correction
	correction := classificationcorrectionbus.NewCorrection{
		ClauseText:    t.Title,
		PredictedType: "task",
		Confidence:    0,
		ActualType:    "note",
		Source:        "correction_applied",
	}
	if _, err := b.corr.RecordWithTx(ctx, tx, correction); err != nil {
		return notebus.Note{}, fmt.Errorf("record correction: %w", err)
	}

	// Delete task with tx
	if err := b.task.DeleteWithTx(ctx, tx, t); err != nil {
		return notebus.Note{}, fmt.Errorf("delete task: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return notebus.Note{}, fmt.Errorf("commit tx: %w", err)
	}

	return newNote, nil
}

// NoteToTask converts a note to a task. Creates task with first line as title, remainder as description.
func (b *Bus) NoteToTask(ctx context.Context, noteID uuid.UUID) (taskbus.Task, error) {
	// Fetch the note
	n, err := b.note.QueryByID(ctx, noteID)
	if err != nil {
		return taskbus.Task{}, fmt.Errorf("fetch note: %w", err)
	}

	// Start transaction
	tx, err := b.db.BeginTxx(ctx, nil)
	if err != nil {
		return taskbus.Task{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Split content into title and description
	title := n.Content
	description := ""
	if idx := strings.Index(n.Content, "\n"); idx != -1 {
		title = strings.TrimSpace(n.Content[:idx])
		remainder := strings.TrimSpace(n.Content[idx+1:])
		if remainder != "" {
			description = remainder
		}
	}

	// Create task with defaults: Open, Medium priority, Medium energy, Unconfirmed=true
	newTask := taskbus.Task{
		ID:          uuid.New(),
		ContextID:   n.ContextID,
		RawInputID:  n.RawInputID,
		Title:       title,
		Description: description,
		Status:      taskstatus.Open,
		Priority:    taskpriority.Medium,
		Energy:      taskenergy.Medium,
		Unconfirmed: true,
		CreatedAt:   n.CreatedAt,
		UpdatedAt:   n.UpdatedAt,
	}

	if err := b.task.CreateWithTx(ctx, tx, newTask); err != nil {
		return taskbus.Task{}, fmt.Errorf("create task: %w", err)
	}

	// Copy note_tags to task_tags
	if err := b.copyNoteTagsToTaskTags(ctx, tx, n.ID, newTask.ID); err != nil {
		return taskbus.Task{}, fmt.Errorf("copy tags: %w", err)
	}

	// Record correction
	correction := classificationcorrectionbus.NewCorrection{
		ClauseText:    n.Content,
		PredictedType: "note",
		Confidence:    0,
		ActualType:    "task",
		Source:        "correction_applied",
	}
	if _, err := b.corr.RecordWithTx(ctx, tx, correction); err != nil {
		return taskbus.Task{}, fmt.Errorf("record correction: %w", err)
	}

	// Delete note with tx
	if err := b.note.DeleteWithTx(ctx, tx, n); err != nil {
		return taskbus.Task{}, fmt.Errorf("delete note: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return taskbus.Task{}, fmt.Errorf("commit tx: %w", err)
	}

	return newTask, nil
}

// preflightTaskToNote checks if the task can be converted to a note.
func (b *Bus) preflightTaskToNote(ctx context.Context, t taskbus.Task) error {
	// Refuse recurring tasks
	if t.RecurrenceRule != nil {
		return errors.New("cannot convert recurring tasks to notes")
	}

	// Refuse tasks with recurrence children
	var count int
	err := b.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM tasks WHERE recurrence_parent_id = $1", t.ID)
	if err != nil {
		return fmt.Errorf("check recurrence children: %w", err)
	}
	if count > 0 {
		return errors.New("cannot convert tasks with recurrence children to notes")
	}

	// Refuse tasks with dependents
	var dependentIDs []string
	err = b.db.SelectContext(ctx, &dependentIDs, "SELECT task_id FROM task_dependencies WHERE depends_on_id = $1 ORDER BY task_id", t.ID)
	if err != nil {
		return fmt.Errorf("check dependents: %w", err)
	}
	if len(dependentIDs) > 0 {
		return fmt.Errorf("cannot convert task with dependents to note (tasks: %s)", strings.Join(dependentIDs, ", "))
	}

	// Refuse if in today's or future daily plan
	now := time.Now()
	today := now.Truncate(24 * time.Hour)

	var inPlan bool
	err = b.db.GetContext(ctx, &inPlan,
		"SELECT EXISTS(SELECT 1 FROM daily_plan_entries WHERE task_id = $1 AND plan_date >= $2)",
		t.ID, today)
	if err != nil {
		return fmt.Errorf("check daily plan: %w", err)
	}
	if inPlan {
		return errors.New("cannot convert task scheduled in today's plan or later")
	}

	return nil
}

// copyTaskTagsToNoteTags copies all tags from task_tags to note_tags.
func (b *Bus) copyTaskTagsToNoteTags(ctx context.Context, tx *sqlx.Tx, taskID, noteID uuid.UUID) error {
	const q = `
		INSERT INTO note_tags (note_id, tag_id)
		SELECT $1, tag_id FROM task_tags WHERE task_id = $2
		ON CONFLICT DO NOTHING`

	if _, err := tx.ExecContext(ctx, q, noteID, taskID); err != nil {
		return fmt.Errorf("copy tags: %w", err)
	}

	return nil
}

// copyNoteTagsToTaskTags copies all tags from note_tags to task_tags.
func (b *Bus) copyNoteTagsToTaskTags(ctx context.Context, tx *sqlx.Tx, noteID, taskID uuid.UUID) error {
	const q = `
		INSERT INTO task_tags (task_id, tag_id)
		SELECT $1, tag_id FROM note_tags WHERE note_id = $2
		ON CONFLICT DO NOTHING`

	if _, err := tx.ExecContext(ctx, q, taskID, noteID); err != nil {
		return fmt.Errorf("copy tags: %w", err)
	}

	return nil
}
