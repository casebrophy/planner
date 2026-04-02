package eventapi_test

import (
	"net/http"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/casebrophy/planner/app/domain/eventapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/app/sdk/query"
)

func query200() []apitest.Table {
	return []apitest.Table{
		{
			Name:       "all",
			URL:        "/api/v1/events?page=1&rows=10",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodGet,
			StatusCode: http.StatusOK,
			GotResp:    &query.Result[eventapp.Event]{},
			ExpResp: &query.Result[eventapp.Event]{
				Total:       0,
				Page:        1,
				RowsPerPage: 10,
				Items:       []eventapp.Event{},
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*query.Result[eventapp.Event])
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*query.Result[eventapp.Event])

				// Clear time fields for deterministic comparison.
				clearTimes := func(items []eventapp.Event) {
					for i := range items {
						items[i].CreatedAt = ""
						items[i].UpdatedAt = ""
					}
				}
				clearTimes(gotResp.Items)
				clearTimes(expResp.Items)

				return cmp.Diff(gotResp, expResp,
					cmpopts.SortSlices(func(a, b eventapp.Event) bool {
						return a.ID < b.ID
					}),
				)
			},
		},
	}
}
