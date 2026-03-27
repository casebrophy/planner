package observationapi_test

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/observationbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
)

type seedData struct {
	observations []observationbus.Observation
	subjectID    uuid.UUID
}

func insertSeedData(db *dbtest.Database) (seedData, error) {
	ctx := context.Background()

	subjectID := uuid.New()

	observations, err := observationbus.TestSeedObservations(ctx, "task", subjectID, 2, db.BusDomain.Observation)
	if err != nil {
		return seedData{}, fmt.Errorf("seeding observations: %w", err)
	}

	return seedData{observations: observations, subjectID: subjectID}, nil
}
