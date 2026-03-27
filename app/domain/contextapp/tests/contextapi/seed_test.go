package contextapi_test

import (
	"context"
	"fmt"

	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
)

type seedData struct {
	contexts []contextbus.Context
}

func insertSeedData(db *dbtest.Database) (seedData, error) {
	ctx := context.Background()

	contexts, err := contextbus.TestSeedContexts(ctx, 2, db.BusDomain.Context)
	if err != nil {
		return seedData{}, fmt.Errorf("seeding contexts: %w", err)
	}

	return seedData{contexts: contexts}, nil
}
