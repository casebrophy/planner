package noteapi_test

import (
	"context"
	"fmt"

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

	// Create an unlinked note (no task_id).
	unlinked, err := db.BusDomain.Note.Create(ctx, notebus.NewNote{
		Content: "Unlinked note",
		Source:  "manual",
	})
	if err != nil {
		return seedData{}, fmt.Errorf("seeding unlinked note: %w", err)
	}
	notes = append(notes, unlinked)

	return seedData{tasks: tasks, notes: notes}, nil
}
