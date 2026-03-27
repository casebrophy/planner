package tagapi_test

import (
	"net/http"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/casebrophy/planner/app/domain/tagapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/tagbus"
)

func query200(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "all",
			URL:        "/api/v1/tags?page=1&rows=10",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodGet,
			StatusCode: http.StatusOK,
			GotResp:    &query.Result[tagapp.Tag]{},
			ExpResp: &query.Result[tagapp.Tag]{
				Total:       len(sd.tags),
				Page:        1,
				RowsPerPage: 10,
				Items:       toAppTags(sd.tags),
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*query.Result[tagapp.Tag])
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*query.Result[tagapp.Tag])
				return cmp.Diff(gotResp, expResp,
					cmpopts.SortSlices(func(a, b tagapp.Tag) bool {
						return a.ID < b.ID
					}),
				)
			},
		},
	}
}

func toAppTags(tags []tagbus.Tag) []tagapp.Tag {
	appTags := make([]tagapp.Tag, len(tags))
	for i, t := range tags {
		appTags[i] = tagapp.Tag{
			ID:   t.ID.String(),
			Name: t.Name,
		}
	}
	return appTags
}
