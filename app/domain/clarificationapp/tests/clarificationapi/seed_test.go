package clarificationapi_test

import (
	"context"
	"fmt"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
)

type seedData struct {
	items []clarificationbus.ClarificationItem
}

func insertSeedData(db *dbtest.Database) (seedData, error) {
	ctx := context.Background()

	items, err := clarificationbus.TestSeedClarifications(ctx, 3, db.BusDomain.Clarification)
	if err != nil {
		return seedData{}, fmt.Errorf("seeding clarifications: %w", err)
	}

	return seedData{items: items}, nil
}
