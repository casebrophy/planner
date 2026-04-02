package dailyplanapi_test

import (
	"testing"

	"github.com/casebrophy/planner/app/sdk/apitest"
)

func Test_DailyPlan(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "Test_DailyPlan")

	test.Run(t, getPlan200(), "get-plan-200")
	test.Run(t, getPlan401(), "get-plan-401")
}
