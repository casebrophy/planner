package contextapi_test

import (
	"net/http"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/casebrophy/planner/app/domain/contextapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/contextbus"
)

func query200(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "all",
			URL:        "/api/v1/contexts?page=1&rows=10",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodGet,
			StatusCode: http.StatusOK,
			GotResp:    &query.Result[contextapp.Context]{},
			ExpResp: &query.Result[contextapp.Context]{
				Total:       len(sd.contexts),
				Page:        1,
				RowsPerPage: 10,
				Items:       toAppContexts(sd.contexts),
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*query.Result[contextapp.Context])
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*query.Result[contextapp.Context])

				clearTimes := func(items []contextapp.Context) {
					for i := range items {
						items[i].CreatedAt = ""
						items[i].UpdatedAt = ""
					}
				}
				clearTimes(gotResp.Items)
				clearTimes(expResp.Items)

				return cmp.Diff(gotResp, expResp,
					cmpopts.SortSlices(func(a, b contextapp.Context) bool {
						return a.ID < b.ID
					}),
				)
			},
		},
	}
}

func toAppContexts(contexts []contextbus.Context) []contextapp.Context {
	appContexts := make([]contextapp.Context, len(contexts))
	for i, c := range contexts {
		appContexts[i] = contextapp.Context{
			ID:            c.ID.String(),
			Title:         c.Title,
			Description:   c.Description,
			Status:        c.Status.String(),
			Summary:       c.Summary,
			DebriefStatus: c.DebriefStatus.String(),
		}
	}
	return appContexts
}
