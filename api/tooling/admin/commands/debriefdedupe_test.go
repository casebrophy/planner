package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/clarificationstatus"
	"github.com/casebrophy/planner/business/types/taskpriority"
	"github.com/casebrophy/planner/business/types/taskstatus"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/google/uuid"
)

// createChildRecurringTask creates a child task whose recurrence_parent_id points
// at parentID. It does so by first creating the task via the business layer
// (which does not expose recurrence_parent_id) and then issuing a raw SQL UPDATE.
func createChildRecurringTask(
	ctx context.Context,
	t *testing.T,
	db *dbtest.Database,
	parentID uuid.UUID,
	rule string,
	title string,
) taskbus.Task {
	t.Helper()

	child, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title:          title,
		Description:    "recurring child",
		Status:         taskstatus.Open,
		Priority:       taskpriority.Medium,
		RecurrenceRule: &rule,
	})
	if err != nil {
		t.Fatalf("creating child task: %v", err)
	}

	_, err = db.DB.ExecContext(ctx,
		`UPDATE tasks SET recurrence_parent_id = $1, recurrence_rule = $2 WHERE task_id = $3`,
		parentID, rule, child.ID)
	if err != nil {
		t.Fatalf("setting recurrence_parent_id on child task: %v", err)
	}

	return child
}

func TestDebriefdedupe_DryRunNoOp(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "TestDebriefdedupe_DryRunNoOp")
	ctx := context.Background()
	log := logger.New(os.Stdout, logger.LevelError, "test")

	// Create a recurring parent task.
	rule := "FREQ=WEEKLY;BYDAY=MO"
	parent, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title:          "Weekly Task (parent)",
		Description:    "A recurring parent task",
		Status:         taskstatus.Open,
		Priority:       taskpriority.Medium,
		RecurrenceRule: &rule,
	})
	if err != nil {
		t.Fatalf("creating parent recurring task: %v", err)
	}

	// Create 3 sibling child tasks that share the same recurrence_parent_id.
	// Attach one task_debrief clarification per child (satisfies uq_clarification_dedup).
	for i := 0; i < 3; i++ {
		child := createChildRecurringTask(ctx, t, db, parent.ID, rule, fmt.Sprintf("Weekly Task Occurrence %d", i))

		_, err := db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
			Kind:               clarificationkind.TaskDebrief,
			SubjectType:        "task",
			SubjectID:          child.ID,
			SubjectDescription: "Weekly Task",
			Question:           "How did the task go?",
			AnswerOptions:      json.RawMessage(`[]`),
			PriorityScore:      0.5,
		})
		if err != nil {
			t.Fatalf("creating clarification: %v", err)
		}
		// Small delay to ensure distinct created_at times.
		time.Sleep(10 * time.Millisecond)
	}

	// Verify 3 pending clarifications exist.
	pending := clarificationstatus.Pending
	itemsBefore, err := db.BusDomain.Clarification.Query(ctx,
		clarificationbus.QueryFilter{Status: &pending},
		clarificationbus.DefaultOrderBy,
		page.New(1, 100))
	if err != nil {
		t.Fatalf("querying before dry-run: %v", err)
	}
	if len(itemsBefore) != 3 {
		t.Fatalf("expected 3 pending clarifications before dry-run, got %d", len(itemsBefore))
	}

	// Run with --dry-run.
	cmd := &DebriefdedupeCMD{}
	err = cmd.Run(ctx, log, db.DB, []string{"--dry-run", "--limit", "100"})
	if err != nil {
		t.Fatalf("dry-run command failed: %v", err)
	}

	// Verify nothing changed.
	itemsAfter, err := db.BusDomain.Clarification.Query(ctx,
		clarificationbus.QueryFilter{Status: &pending},
		clarificationbus.DefaultOrderBy,
		page.New(1, 100))
	if err != nil {
		t.Fatalf("querying after dry-run: %v", err)
	}
	if len(itemsAfter) != 3 {
		t.Fatalf("expected 3 pending clarifications after dry-run, got %d", len(itemsAfter))
	}
}

func TestDebriefdedupe_RealRunDismisses(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "TestDebriefdedupe_RealRunDismisses")
	ctx := context.Background()
	log := logger.New(os.Stdout, logger.LevelError, "test")

	// Create a recurring parent task.
	rule := "FREQ=WEEKLY;BYDAY=MO"
	parent, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title:          "Weekly Task (parent)",
		Description:    "A recurring parent task",
		Status:         taskstatus.Open,
		Priority:       taskpriority.Medium,
		RecurrenceRule: &rule,
	})
	if err != nil {
		t.Fatalf("creating parent recurring task: %v", err)
	}

	// Create 3 sibling children + one task_debrief clarification each.
	var clarificationIDs []uuid.UUID
	for i := 0; i < 3; i++ {
		child := createChildRecurringTask(ctx, t, db, parent.ID, rule, fmt.Sprintf("Weekly Task Occurrence %d", i))

		item, err := db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
			Kind:               clarificationkind.TaskDebrief,
			SubjectType:        "task",
			SubjectID:          child.ID,
			SubjectDescription: "Weekly Task",
			Question:           "How did the task go?",
			AnswerOptions:      json.RawMessage(`[]`),
			PriorityScore:      0.5,
		})
		if err != nil {
			t.Fatalf("creating clarification: %v", err)
		}
		clarificationIDs = append(clarificationIDs, item.ID)
		time.Sleep(10 * time.Millisecond)
	}

	// Run without --dry-run (real run).
	cmd := &DebriefdedupeCMD{}
	err = cmd.Run(ctx, log, db.DB, []string{"--limit", "100"})
	if err != nil {
		t.Fatalf("real run command failed: %v", err)
	}

	// Verify the most recent clarification is still pending.
	pending := clarificationstatus.Pending
	itemsAfter, err := db.BusDomain.Clarification.Query(ctx,
		clarificationbus.QueryFilter{Status: &pending},
		clarificationbus.DefaultOrderBy,
		page.New(1, 100))
	if err != nil {
		t.Fatalf("querying pending after real run: %v", err)
	}
	if len(itemsAfter) != 1 {
		t.Fatalf("expected 1 pending clarification after real run, got %d", len(itemsAfter))
	}

	// Verify 2 were dismissed.
	dismissed := clarificationstatus.Dismissed
	dismissedItems, err := db.BusDomain.Clarification.Query(ctx,
		clarificationbus.QueryFilter{Status: &dismissed},
		clarificationbus.DefaultOrderBy,
		page.New(1, 100))
	if err != nil {
		t.Fatalf("querying dismissed after real run: %v", err)
	}
	if len(dismissedItems) != 2 {
		t.Fatalf("expected 2 dismissed clarifications after real run, got %d", len(dismissedItems))
	}
}

func TestDebriefdedupe_NonRecurringUntouched(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "TestDebriefdedupe_NonRecurringUntouched")
	ctx := context.Background()
	log := logger.New(os.Stdout, logger.LevelError, "test")

	// Create 3 standalone non-recurring tasks, one task_debrief clarification each.
	for i := 0; i < 3; i++ {
		task, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
			Title:       fmt.Sprintf("One-Time Task %d", i),
			Description: "A non-recurring task",
			Status:      taskstatus.Open,
			Priority:    taskpriority.Medium,
		})
		if err != nil {
			t.Fatalf("creating non-recurring task: %v", err)
		}

		_, err = db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
			Kind:               clarificationkind.TaskDebrief,
			SubjectType:        "task",
			SubjectID:          task.ID,
			SubjectDescription: "One-Time Task",
			Question:           "How did the task go?",
			AnswerOptions:      json.RawMessage(`[]`),
			PriorityScore:      0.5,
		})
		if err != nil {
			t.Fatalf("creating clarification: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Verify 3 pending clarifications exist.
	pending := clarificationstatus.Pending
	itemsBefore, err := db.BusDomain.Clarification.Query(ctx,
		clarificationbus.QueryFilter{Status: &pending},
		clarificationbus.DefaultOrderBy,
		page.New(1, 100))
	if err != nil {
		t.Fatalf("querying before command: %v", err)
	}
	if len(itemsBefore) != 3 {
		t.Fatalf("expected 3 pending clarifications before, got %d", len(itemsBefore))
	}

	// Run the command.
	cmd := &DebriefdedupeCMD{}
	err = cmd.Run(ctx, log, db.DB, []string{"--limit", "100"})
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	// Verify all 3 clarifications are still pending (non-recurring tasks are untouched).
	itemsAfter, err := db.BusDomain.Clarification.Query(ctx,
		clarificationbus.QueryFilter{Status: &pending},
		clarificationbus.DefaultOrderBy,
		page.New(1, 100))
	if err != nil {
		t.Fatalf("querying after command: %v", err)
	}
	if len(itemsAfter) != 3 {
		t.Fatalf("expected 3 pending clarifications after (non-recurring untouched), got %d", len(itemsAfter))
	}
}

func TestDebriefdedupe_EmptyCase(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "TestDebriefdedupe_EmptyCase")
	ctx := context.Background()
	log := logger.New(os.Stdout, logger.LevelError, "test")

	// Don't create any clarifications

	// Run the command (should complete without error)
	cmd := &DebriefdedupeCMD{}
	err := cmd.Run(ctx, log, db.DB, []string{"--limit", "100"})
	if err != nil {
		t.Fatalf("command failed on empty DB: %v", err)
	}

	// Verify no clarifications were created
	pending := clarificationstatus.Pending
	items, err := db.BusDomain.Clarification.Query(ctx,
		clarificationbus.QueryFilter{Status: &pending},
		clarificationbus.DefaultOrderBy,
		page.New(1, 100))
	if err != nil {
		t.Fatalf("querying clarifications: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 clarifications on empty DB, got %d", len(items))
	}
}
