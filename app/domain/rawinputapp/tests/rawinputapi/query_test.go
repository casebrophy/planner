package rawinputapi_test

import (
	"net/http"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/casebrophy/planner/app/domain/rawinputapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
)

func query200(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "all",
			URL:        "/api/v1/raw-inputs?page=1&rows=10",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodGet,
			StatusCode: http.StatusOK,
			GotResp:    &query.Result[rawinputapp.RawInput]{},
			ExpResp: &query.Result[rawinputapp.RawInput]{
				Total:       len(sd.rawInputs),
				Page:        1,
				RowsPerPage: 10,
				Items:       toAppRawInputs(sd.rawInputs),
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*query.Result[rawinputapp.RawInput])
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*query.Result[rawinputapp.RawInput])

				clearTimes := func(items []rawinputapp.RawInput) {
					for i := range items {
						items[i].CreatedAt = ""
					}
				}
				clearTimes(gotResp.Items)
				clearTimes(expResp.Items)

				return cmp.Diff(gotResp, expResp,
					cmpopts.SortSlices(func(a, b rawinputapp.RawInput) bool {
						return a.ID < b.ID
					}),
				)
			},
		},
	}
}

func query401(_ seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "no-key",
			URL:        "/api/v1/raw-inputs?page=1&rows=10",
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

// toAppRawInputs converts business raw inputs to app-layer raw inputs for comparison.
func toAppRawInputs(ris []rawinputbus.RawInput) []rawinputapp.RawInput {
	items := make([]rawinputapp.RawInput, len(ris))
	for i, ri := range ris {
		items[i] = rawinputapp.RawInput{
			ID:         ri.ID.String(),
			SourceType: ri.SourceType.String(),
			Status:     ri.Status.String(),
			RawContent: ri.RawContent,
			RetryCount: ri.RetryCount,
			MaxRetries: ri.MaxRetries,
		}
	}
	return items
}
