package reclassifybus_test

import (
	"context"
	"errors"
	"testing"

	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/reclassifybus"
	"github.com/casebrophy/planner/business/domain/tagbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/business/types/rawinputsource"
	"github.com/casebrophy/planner/business/types/taskenergy"
	"github.com/casebrophy/planner/business/types/taskpriority"
	"github.com/casebrophy/planner/business/types/taskstatus"
	"github.com/google/uuid"
)

func TestTaskToNote_HappyPath(t *testing.T) {
	t.Parallel()

	dbt := dbtest.New(t, "task_to_note_happy")
	ctx := context.Background()

	// Create a tag
	tag, err := dbt.BusDomain.Tag.Create(ctx, tagbus.NewTag{Name: "test-tag"})
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}

	// Create raw_input to satisfy task.raw_input_id FK
	ri, err := dbt.BusDomain.RawInput.Create(ctx, rawinputbus.NewRawInput{
		SourceType: rawinputsource.Voice,
		RawContent: "Task Title Task Description",
	})
	if err != nil {
		t.Fatalf("create raw input: %v", err)
	}
	rawInputID := ri.ID

	// Create task with tags and raw_input
	task, err := dbt.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title:       "Task Title",
		Description: "Task Description",
		Status:      taskstatus.Open,
		Priority:    taskpriority.Medium,
		Energy:      taskenergy.Medium,
		RawInputID:  &rawInputID,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	// Add tag to task
	if _, err := dbt.DB.ExecContext(ctx, "INSERT INTO task_tags (task_id, tag_id) VALUES ($1, $2)", task.ID, tag.ID); err != nil {
		t.Fatalf("insert task_tag: %v", err)
	}

	// Create reclassifybus
	reclassifyBus := reclassifybus.NewBusiness(dbt.Log, dbt.BusDomain.Task, dbt.BusDomain.Note, dbt.BusDomain.ClassificationCorrection, dbt.DB)

	// Convert task to note
	note, err := reclassifyBus.TaskToNote(ctx, task.ID)
	if err != nil {
		t.Fatalf("convert task to note: %v", err)
	}

	// Verify note exists
	if note.ID == uuid.Nil {
		t.Error("note ID is nil")
	}
	if note.Content != "Task Title\n\nTask Description" {
		t.Errorf("note content = %q, want %q", note.Content, "Task Title\n\nTask Description")
	}
	if note.Source != "reclassified_from_task" {
		t.Errorf("note source = %q, want %q", note.Source, "reclassified_from_task")
	}
	if note.RawInputID == nil || *note.RawInputID != rawInputID {
		t.Error("raw input ID not preserved")
	}

	// Verify task is deleted
	_, err = dbt.BusDomain.Task.QueryByID(ctx, task.ID)
	if !errors.Is(err, sqldb.ErrDBNotFound) {
		t.Errorf("task should be deleted, got: %v", err)
	}

	// Verify tags are copied
	var tagCount int
	if err := dbt.DB.GetContext(ctx, &tagCount, "SELECT COUNT(*) FROM note_tags WHERE note_id = $1", note.ID); err != nil {
		t.Fatalf("count note_tags: %v", err)
	}
	if tagCount != 1 {
		t.Errorf("note_tags count = %d, want 1", tagCount)
	}

	// Verify correction is recorded
	corrs, err := dbt.BusDomain.ClassificationCorrection.QueryBySource(ctx, "correction_applied", page.New(1, 100))
	if err != nil {
		t.Fatalf("query corrections: %v", err)
	}
	if len(corrs) != 1 {
		t.Errorf("corrections count = %d, want 1", len(corrs))
	}
	if corrs[0].PredictedType != "task" || corrs[0].ActualType != "note" {
		t.Error("correction types incorrect")
	}
}

func TestTaskToNote_RefusesRecurring(t *testing.T) {
	t.Parallel()

	dbt := dbtest.New(t, "task_to_note_recurring")
	ctx := context.Background()

	// Create recurring task
	rrule := "FREQ=DAILY"
	task, err := dbt.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title:          "Recurring Task",
		Status:         taskstatus.Open,
		Priority:       taskpriority.Medium,
		Energy:         taskenergy.Medium,
		RecurrenceRule: &rrule,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	// Create reclassifybus
	reclassifyBus := reclassifybus.NewBusiness(dbt.Log, dbt.BusDomain.Task, dbt.BusDomain.Note, dbt.BusDomain.ClassificationCorrection, dbt.DB)

	// Should refuse
	_, err = reclassifyBus.TaskToNote(ctx, task.ID)
	if err == nil {
		t.Error("should refuse recurring task")
	}

	// Task should still exist
	_, err = dbt.BusDomain.Task.QueryByID(ctx, task.ID)
	if err != nil {
		t.Errorf("task should still exist: %v", err)
	}
}

func TestTaskToNote_RefusesWithDependents(t *testing.T) {
	t.Parallel()

	dbt := dbtest.New(t, "task_to_note_dependents")
	ctx := context.Background()

	// Create two tasks
	task1, err := dbt.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title:    "Task 1",
		Status:   taskstatus.Open,
		Priority: taskpriority.Medium,
		Energy:   taskenergy.Medium,
	})
	if err != nil {
		t.Fatalf("create task1: %v", err)
	}

	task2, err := dbt.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title:    "Task 2",
		Status:   taskstatus.Open,
		Priority: taskpriority.Medium,
		Energy:   taskenergy.Medium,
	})
	if err != nil {
		t.Fatalf("create task2: %v", err)
	}

	// Make task2 depend on task1
	if _, err := dbt.DB.ExecContext(ctx, "INSERT INTO task_dependencies (task_id, depends_on_id) VALUES ($1, $2)", task2.ID, task1.ID); err != nil {
		t.Fatalf("create dependency: %v", err)
	}

	// Create reclassifybus
	reclassifyBus := reclassifybus.NewBusiness(dbt.Log, dbt.BusDomain.Task, dbt.BusDomain.Note, dbt.BusDomain.ClassificationCorrection, dbt.DB)

	// Should refuse to convert task1
	_, err = reclassifyBus.TaskToNote(ctx, task1.ID)
	if err == nil {
		t.Error("should refuse task with dependents")
	}
}

func TestNoteToTask_HappyPath(t *testing.T) {
	t.Parallel()

	dbt := dbtest.New(t, "note_to_task_happy")
	ctx := context.Background()

	// Create a tag
	tag, err := dbt.BusDomain.Tag.Create(ctx, tagbus.NewTag{Name: "test-tag"})
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}

	// Create raw_input to satisfy note.raw_input_id FK
	ri, err := dbt.BusDomain.RawInput.Create(ctx, rawinputbus.NewRawInput{
		SourceType: rawinputsource.Voice,
		RawContent: "Note Title Note Description",
	})
	if err != nil {
		t.Fatalf("create raw input: %v", err)
	}
	rawInputID := ri.ID

	// Create note with tags
	note, err := dbt.BusDomain.Note.Create(ctx, notebus.NewNote{
		RawInputID: &rawInputID,
		Content:    "Note Title\n\nNote Description",
		Source:     "manual",
	})
	if err != nil {
		t.Fatalf("create note: %v", err)
	}

	// Add tag to note
	if _, err := dbt.DB.ExecContext(ctx, "INSERT INTO note_tags (note_id, tag_id) VALUES ($1, $2)", note.ID, tag.ID); err != nil {
		t.Fatalf("insert note_tag: %v", err)
	}

	// Create reclassifybus
	reclassifyBus := reclassifybus.NewBusiness(dbt.Log, dbt.BusDomain.Task, dbt.BusDomain.Note, dbt.BusDomain.ClassificationCorrection, dbt.DB)

	// Convert note to task
	task, err := reclassifyBus.NoteToTask(ctx, note.ID)
	if err != nil {
		t.Fatalf("convert note to task: %v", err)
	}

	// Verify task
	if task.ID == uuid.Nil {
		t.Error("task ID is nil")
	}
	if task.Title != "Note Title" {
		t.Errorf("task title = %q, want %q", task.Title, "Note Title")
	}
	if task.Description != "Note Description" {
		t.Errorf("task description = %q, want %q", task.Description, "Note Description")
	}
	if task.Status != taskstatus.Open {
		t.Errorf("task status = %v, want %v", task.Status, taskstatus.Open)
	}
	if task.Priority != taskpriority.Medium {
		t.Errorf("task priority = %v, want %v", task.Priority, taskpriority.Medium)
	}
	if task.Energy != taskenergy.Medium {
		t.Errorf("task energy = %v, want %v", task.Energy, taskenergy.Medium)
	}
	if !task.Unconfirmed {
		t.Error("task should be unconfirmed")
	}

	// Verify note is deleted
	_, err = dbt.BusDomain.Note.QueryByID(ctx, note.ID)
	if !errors.Is(err, sqldb.ErrDBNotFound) {
		t.Errorf("note should be deleted, got: %v", err)
	}

	// Verify tags are copied
	var tagCount int
	if err := dbt.DB.GetContext(ctx, &tagCount, "SELECT COUNT(*) FROM task_tags WHERE task_id = $1", task.ID); err != nil {
		t.Fatalf("count task_tags: %v", err)
	}
	if tagCount != 1 {
		t.Errorf("task_tags count = %d, want 1", tagCount)
	}
}

func TestNoteToTask_SingleLineContent(t *testing.T) {
	t.Parallel()

	dbt := dbtest.New(t, "note_to_task_single_line")
	ctx := context.Background()

	// Create single-line note
	note, err := dbt.BusDomain.Note.Create(ctx, notebus.NewNote{
		Content: "Single Line Title",
		Source:  "manual",
	})
	if err != nil {
		t.Fatalf("create note: %v", err)
	}

	// Create reclassifybus
	reclassifyBus := reclassifybus.NewBusiness(dbt.Log, dbt.BusDomain.Task, dbt.BusDomain.Note, dbt.BusDomain.ClassificationCorrection, dbt.DB)

	// Convert
	task, err := reclassifyBus.NoteToTask(ctx, note.ID)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}

	if task.Title != "Single Line Title" {
		t.Errorf("task title = %q, want %q", task.Title, "Single Line Title")
	}
	if task.Description != "" {
		t.Errorf("task description = %q, want empty", task.Description)
	}
}
