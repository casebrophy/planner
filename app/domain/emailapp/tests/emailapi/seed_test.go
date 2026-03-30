package emailapi_test

import (
	"context"
	"fmt"

	"github.com/casebrophy/planner/business/domain/emailbus"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
)

type seedData struct {
	emails []emailbus.Email
}

func insertSeedData(db *dbtest.Database) (seedData, error) {
	ctx := context.Background()

	rawInputs, err := rawinputbus.TestSeedRawInputs(ctx, 1, db.BusDomain.RawInput)
	if err != nil {
		return seedData{}, fmt.Errorf("seeding raw inputs: %w", err)
	}

	emails, err := emailbus.TestSeedEmails(ctx, 2, rawInputs[0].ID, db.BusDomain.Email)
	if err != nil {
		return seedData{}, fmt.Errorf("seeding emails: %w", err)
	}

	return seedData{emails: emails}, nil
}
