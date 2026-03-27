package inactivitybus_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/unitest"
)

func Test_Inactivity(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Inactivity")

	unitest.Run(t, checkAll(db), "check-all")
}

// checkAll tests that CheckAll runs without error on an empty database
// (no stale tasks or contexts exist).
func checkAll(db *dbtest.Database) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "no-stale-items",
			ExpResp: error(nil),
			ExcFunc: func(ctx context.Context) any {
				return db.BusDomain.Inactivity.CheckAll(ctx)
			},
			CmpFunc: func(got any, exp any) string {
				if got != nil {
					return fmt.Sprintf("expected nil error, got: %v", got)
				}
				return ""
			},
		},
	}
}
