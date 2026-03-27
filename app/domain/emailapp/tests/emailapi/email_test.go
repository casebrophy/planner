package emailapi_test

import (
	"testing"

	"github.com/casebrophy/planner/app/sdk/apitest"
)

func Test_Email(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "Test_Email")

	sd, err := insertSeedData(test.DB)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	test.Run(t, query200(sd), "query-200")
	test.Run(t, query401(sd), "query-401")
}
