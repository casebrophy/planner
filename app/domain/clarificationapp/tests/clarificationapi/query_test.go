package clarificationapi_test

import (
	"net/http"

	"github.com/google/go-cmp/cmp"

	"github.com/casebrophy/planner/app/domain/clarificationapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
)

func query200(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "pending-queue",
			URL:        "/api/v1/clarifications?page=1&rows=10",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodGet,
			StatusCode: http.StatusOK,
			GotResp:    &query.Result[clarificationapp.ClarificationItem]{},
			ExpResp:    &query.Result[clarificationapp.ClarificationItem]{},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*query.Result[clarificationapp.ClarificationItem])
				if !exists {
					return "error occurred"
				}
				// Verify we get the expected count of seeded items.
				if gotResp.Total != len(sd.items) {
					return cmp.Diff(gotResp.Total, len(sd.items))
				}
				return ""
			},
		},
	}
}

func query401(_ seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "no-key",
			URL:        "/api/v1/clarifications?page=1&rows=10",
			Token:      "",
			Method:     http.MethodGet,
			StatusCode: http.StatusUnauthorized,
			GotResp:    &errs.Error{},
			ExpResp:    &errs.Error{Code: errs.Unauthenticated, Message: "missing api key"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}
