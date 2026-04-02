package dailyplanapi_test

import (
	"net/http"

	"github.com/google/go-cmp/cmp"

	"github.com/casebrophy/planner/app/domain/dailyplanapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/app/sdk/errs"
)

func getPlan200() []apitest.Table {
	return []apitest.Table{
		{
			Name:       "empty",
			URL:        "/api/v1/daily-plan",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodGet,
			StatusCode: http.StatusOK,
			GotResp:    &dailyplanapp.DailyPlan{},
			ExpResp:    &dailyplanapp.DailyPlan{Items: []dailyplanapp.DailyPlanItem{}},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*dailyplanapp.DailyPlan)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*dailyplanapp.DailyPlan)
				// Just verify it returns 200 with a plan structure
				// ID and timestamps will vary, so we only check basic fields
				return cmp.Diff(gotResp.Items, expResp.Items)
			},
		},
	}
}

func getPlan401() []apitest.Table {
	return []apitest.Table{
		{
			Name:       "no-key",
			URL:        "/api/v1/daily-plan",
			Token:      "",
			Method:     http.MethodGet,
			StatusCode: http.StatusUnauthorized,
			GotResp:    &errs.Error{},
			ExpResp:    &errs.Error{Code: errs.Unauthenticated, Message: "missing api key"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
		{
			Name:       "wrong-key",
			URL:        "/api/v1/daily-plan",
			Token:      "wrong-key",
			Method:     http.MethodGet,
			StatusCode: http.StatusUnauthorized,
			GotResp:    &errs.Error{},
			ExpResp:    &errs.Error{Code: errs.Unauthenticated, Message: "invalid api key"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}
