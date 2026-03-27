package tagapi_test

import (
	"context"
	"fmt"

	"github.com/casebrophy/planner/business/domain/tagbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
)

type seedData struct {
	tags []tagbus.Tag
}

func insertSeedData(db *dbtest.Database) (seedData, error) {
	ctx := context.Background()

	tags, err := tagbus.TestSeedTags(ctx, 2, db.BusDomain.Tag)
	if err != nil {
		return seedData{}, fmt.Errorf("seeding tags: %w", err)
	}

	return seedData{tags: tags}, nil
}
