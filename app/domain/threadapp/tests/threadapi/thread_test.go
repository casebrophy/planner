package threadapi_test

import (
	"testing"

	"github.com/casebrophy/planner/app/sdk/apitest"
)

func Test_Thread(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "Test_Thread")

	sd, err := insertSeedData(test.DB)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	test.Run(t, create200(sd), "create-200")
	test.Run(t, create400(sd), "create-400")
	test.Run(t, create401(sd), "create-401")
	test.Run(t, query200(sd), "query-200")
	test.Run(t, query401(sd), "query-401")
}
