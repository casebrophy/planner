package activitylogapi_test

import (
	"testing"

	"github.com/casebrophy/planner/app/sdk/apitest"
)

func Test_ActivityLog(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "Test_ActivityLog")

	sd, err := insertSeedData(test.DB)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	test.Run(t, bulk200(sd), "bulk-200")
	test.Run(t, bulk400(sd), "bulk-400")
	test.Run(t, bulk401(sd), "bulk-401")
}
