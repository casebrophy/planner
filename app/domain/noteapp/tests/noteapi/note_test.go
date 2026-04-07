package noteapi_test

import (
	"testing"

	"github.com/casebrophy/planner/app/sdk/apitest"
)

func Test_Note(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "Test_Note")

	sd, err := insertSeedData(test.DB)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	test.Run(t, queryByTaskID200(sd), "query-by-task-id-200")
	test.Run(t, queryAll200(sd), "query-all-200")
}
