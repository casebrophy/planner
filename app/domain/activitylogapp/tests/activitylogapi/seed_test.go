package activitylogapi_test

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/activitylogbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
)

type seedData struct {
	logs       []activitylogbus.Log
	subjectID1 uuid.UUID
	subjectID2 uuid.UUID
}

func insertSeedData(db *dbtest.Database) (seedData, error) {
	ctx := context.Background()

	subjectID1 := uuid.New()
	subjectID2 := uuid.New()

	logs1, err := activitylogbus.TestSeedLogs(ctx, 2, "task", subjectID1, db.BusDomain.ActivityLog)
	if err != nil {
		return seedData{}, fmt.Errorf("seeding activity logs for subject 1: %w", err)
	}

	logs2, err := activitylogbus.TestSeedLogs(ctx, 1, "task", subjectID2, db.BusDomain.ActivityLog)
	if err != nil {
		return seedData{}, fmt.Errorf("seeding activity logs for subject 2: %w", err)
	}

	allLogs := append(logs1, logs2...)

	return seedData{
		logs:       allLogs,
		subjectID1: subjectID1,
		subjectID2: subjectID2,
	}, nil
}
