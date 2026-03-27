package rawinputapi_test

import (
	"context"
	"fmt"

	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
)

type seedData struct {
	rawInputs []rawinputbus.RawInput
}

func insertSeedData(db *dbtest.Database) (seedData, error) {
	ctx := context.Background()

	rawInputs, err := rawinputbus.TestSeedRawInputs(ctx, 2, db.BusDomain.RawInput)
	if err != nil {
		return seedData{}, fmt.Errorf("seeding raw inputs: %w", err)
	}

	return seedData{rawInputs: rawInputs}, nil
}
