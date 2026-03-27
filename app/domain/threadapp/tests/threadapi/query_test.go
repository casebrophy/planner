package threadapi_test

import (
	"fmt"
	"net/http"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/casebrophy/planner/app/domain/threadapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/threadbus"
)

func query200(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "by-subject",
			URL:        fmt.Sprintf("/api/v1/threads/task/%s?page=1&rows=10", sd.subjectID),
			Token:      apitest.TestAPIKey,
			Method:     http.MethodGet,
			StatusCode: http.StatusOK,
			GotResp:    &query.Result[threadapp.ThreadEntry]{},
			ExpResp: &query.Result[threadapp.ThreadEntry]{
				Total:       len(sd.entries),
				Page:        1,
				RowsPerPage: 10,
				Items:       toAppThreadEntries(sd.entries),
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*query.Result[threadapp.ThreadEntry])
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*query.Result[threadapp.ThreadEntry])

				clearTimes := func(items []threadapp.ThreadEntry) {
					for i := range items {
						items[i].CreatedAt = ""
					}
				}
				clearTimes(gotResp.Items)
				clearTimes(expResp.Items)

				return cmp.Diff(gotResp, expResp,
					cmpopts.SortSlices(func(a, b threadapp.ThreadEntry) bool {
						return a.ID < b.ID
					}),
				)
			},
		},
	}
}

func query401(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "no-key",
			URL:        fmt.Sprintf("/api/v1/threads/task/%s?page=1&rows=10", sd.subjectID),
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

// toAppThreadEntries converts business thread entries to app-layer thread entries for comparison.
func toAppThreadEntries(entries []threadbus.ThreadEntry) []threadapp.ThreadEntry {
	appEntries := make([]threadapp.ThreadEntry, len(entries))
	for i, e := range entries {
		appEntries[i] = threadapp.ThreadEntry{
			ID:             e.ID.String(),
			SubjectType:    e.SubjectType,
			SubjectID:      e.SubjectID.String(),
			Kind:           e.Kind.String(),
			Content:        e.Content,
			Source:         e.Source.String(),
			RequiresAction: e.RequiresAction,
		}
	}
	return appEntries
}
