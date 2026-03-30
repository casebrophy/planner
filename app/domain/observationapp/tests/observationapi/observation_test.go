package observationapi_test

import (
	"testing"

	"github.com/casebrophy/planner/app/sdk/apitest"
)

func Test_Observation(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "Test_Observation")

	sd, err := insertSeedData(test.DB)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	test.Run(t, query200(sd), "query-200")
	test.Run(t, query401(sd), "query-401")
	test.Run(t, record200(sd), "record-200")
	test.Run(t, record400(sd), "record-400")
	test.Run(t, record401(sd), "record-401")
}
