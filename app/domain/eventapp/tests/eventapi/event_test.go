package eventapi_test

import (
	"testing"

	"github.com/casebrophy/planner/app/sdk/apitest"
)

func Test_Event(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "Test_Event")

	test.Run(t, query200(), "query-200")
	test.Run(t, create200(), "create-200")
	test.Run(t, create400(), "create-400")
	test.Run(t, create401(), "create-401")
}
