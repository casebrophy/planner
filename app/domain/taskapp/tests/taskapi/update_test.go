package taskapi_test

import (
	"fmt"
	"net/http"

	"github.com/google/go-cmp/cmp"

	"github.com/casebrophy/planner/app/domain/taskapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/app/sdk/errs"
)

func update200(sd seedData) []apitest.Table {
	newStatus := "blocked"

	return []apitest.Table{
		{
			Name:       "basic",
			URL:        fmt.Sprintf("/api/v1/tasks/%s", sd.tasks[0].ID),
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPut,
			StatusCode: http.StatusOK,
			Input: &taskapp.UpdateTask{
				Status: &newStatus,
			},
			GotResp: &taskapp.Task{},
			ExpResp: &taskapp.Task{
				ID:            sd.tasks[0].ID.String(),
				Title:         sd.tasks[0].Title,
				Description:   sd.tasks[0].Description,
				Status:        "blocked",
				Priority:      sd.tasks[0].Priority.String(),
				Energy:        sd.tasks[0].Energy.String(),
				DebriefStatus: sd.tasks[0].DebriefStatus.String(),
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*taskapp.Task)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*taskapp.Task)
				expResp.CreatedAt = gotResp.CreatedAt
				expResp.UpdatedAt = gotResp.UpdatedAt
				return cmp.Diff(gotResp, expResp)
			},
		},
	}
}

func update400(sd seedData) []apitest.Table {
	badStatus := "not-a-status"

	return []apitest.Table{
		{
			Name:       "bad-status",
			URL:        fmt.Sprintf("/api/v1/tasks/%s", sd.tasks[0].ID),
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPut,
			StatusCode: http.StatusBadRequest,
			Input: &taskapp.UpdateTask{
				Status: &badStatus,
			},
			GotResp: &errs.Error{},
			ExpResp: &errs.Error{Code: errs.InvalidArgument},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*errs.Error)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*errs.Error)
				// Only compare the error code, not the message (implementation detail).
				expResp.Message = gotResp.Message
				return cmp.Diff(gotResp, expResp)
			},
		},
	}
}

func update401(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "no-key",
			URL:        fmt.Sprintf("/api/v1/tasks/%s", sd.tasks[0].ID),
			Token:      "",
			Method:     http.MethodPut,
			StatusCode: http.StatusUnauthorized,
			GotResp:    &errs.Error{},
			ExpResp:    &errs.Error{Code: errs.Unauthenticated, Message: "missing api key"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}
