package noteapi_test

import (
	"context"
	"fmt"

	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
)

type seedData struct {
	tasks []taskbus.Task
	notes []notebus.Note
}

func insertSeedData(db *dbtest.Database) (seedData, error) {
	ctx := context.Background()

	tasks, err := taskbus.TestSeedTasks(ctx, 2, db.BusDomain.Task)
	if err != nil {
		return seedData{}, fmt.Errorf("seeding tasks: %w", err)
	}

	contexts, err := contextbus.TestSeedContexts(ctx, 1, db.BusDomain.Context)
	if err != nil {
		return seedData{}, fmt.Errorf("seeding contexts: %w", err)
	}

	// Create notes linked to task[0].
	taskID := tasks[0].ID
	var notes []notebus.Note
	for i := 0; i < 2; i++ {
		n, err := db.BusDomain.Note.Create(ctx, notebus.NewNote{
			TaskID:  &taskID,
			Content: fmt.Sprintf("Note for task %d", i),
			Source:  "manual",
		})
		if err != nil {
			return seedData{}, fmt.Errorf("seeding note %d: %w", i, err)
		}
		notes = append(notes, n)
	}

	// Create a context-linked note (task_id filter test expects exactly 2 task notes).
	contextID := contexts[0].ID
	contextNote, err := db.BusDomain.Note.Create(ctx, notebus.NewNote{
		ContextID: &contextID,
		Content:   "Context note",
		Source:    "manual",
	})
	if err != nil {
		return seedData{}, fmt.Errorf("seeding context note: %w", err)
	}
	notes = append(notes, contextNote)

	return seedData{tasks: tasks, notes: notes}, nil
}
