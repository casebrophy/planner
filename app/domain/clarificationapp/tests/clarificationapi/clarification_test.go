package clarificationapi_test

import (
	"testing"

	"github.com/casebrophy/planner/app/sdk/apitest"
)

func Test_Clarification(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "Test_Clarification")

	sd, err := insertSeedData(test.DB)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	test.Run(t, query200(sd), "query-200")
	test.Run(t, query401(sd), "query-401")
	test.Run(t, resolve200(sd), "resolve-200")
	test.Run(t, resolve401(sd), "resolve-401")
	test.Run(t, snooze200(sd), "snooze-200")
	test.Run(t, dismiss200(sd), "dismiss-200")
}
