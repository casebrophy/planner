package taskapi_test

import (
	"context"
	"fmt"

	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
)

type seedData struct {
	tasks []taskbus.Task
}

func insertSeedData(db *dbtest.Database) (seedData, error) {
	ctx := context.Background()

	tasks, err := taskbus.TestSeedTasks(ctx, 2, db.BusDomain.Task)
	if err != nil {
		return seedData{}, fmt.Errorf("seeding tasks: %w", err)
	}

	return seedData{tasks: tasks}, nil
}
