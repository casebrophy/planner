package threadapi_test

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
)

type seedData struct {
	entries   []threadbus.ThreadEntry
	subjectID uuid.UUID
}

func insertSeedData(db *dbtest.Database) (seedData, error) {
	ctx := context.Background()

	subjectID := uuid.New()

	entries, err := threadbus.TestSeedThreadEntries(ctx, "task", subjectID, 2, db.BusDomain.Thread)
	if err != nil {
		return seedData{}, fmt.Errorf("seeding thread entries: %w", err)
	}

	return seedData{entries: entries, subjectID: subjectID}, nil
}
