package taskapi_test

import (
	"net/http"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/casebrophy/planner/app/domain/taskapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/taskbus"
)

func query200(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "all",
			URL:        "/api/v1/tasks?page=1&rows=10",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodGet,
			StatusCode: http.StatusOK,
			GotResp:    &query.Result[taskapp.Task]{},
			ExpResp: &query.Result[taskapp.Task]{
				Total:       len(sd.tasks),
				Page:        1,
				RowsPerPage: 10,
				Items:       toAppTasks(sd.tasks),
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*query.Result[taskapp.Task])
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*query.Result[taskapp.Task])

				// Clear time fields for deterministic comparison.
				clearTimes := func(items []taskapp.Task) {
					for i := range items {
						items[i].CreatedAt = ""
						items[i].UpdatedAt = ""
					}
				}
				clearTimes(gotResp.Items)
				clearTimes(expResp.Items)

				return cmp.Diff(gotResp, expResp,
					cmpopts.SortSlices(func(a, b taskapp.Task) bool {
						return a.ID < b.ID
					}),
				)
			},
		},
	}
}

// toAppTasks converts business tasks to app-layer tasks for test comparison.
// Only sets deterministic fields (no timestamps).
func toAppTasks(tasks []taskbus.Task) []taskapp.Task {
	appTasks := make([]taskapp.Task, len(tasks))
	for i, t := range tasks {
		appTasks[i] = taskapp.Task{
			ID:            t.ID.String(),
			Title:         t.Title,
			Description:   t.Description,
			Status:        t.Status.String(),
			Priority:      t.Priority.String(),
			Energy:        t.Energy.String(),
			DebriefStatus: t.DebriefStatus.String(),
		}
	}
	return appTasks
}
