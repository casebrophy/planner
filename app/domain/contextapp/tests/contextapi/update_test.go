package contextapi_test

import (
	"fmt"
	"net/http"

	"github.com/google/go-cmp/cmp"

	"github.com/casebrophy/planner/app/domain/contextapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/app/sdk/errs"
)

func update200(sd seedData) []apitest.Table {
	newTitle := "Updated Context"

	return []apitest.Table{
		{
			Name:       "basic",
			URL:        fmt.Sprintf("/api/v1/contexts/%s", sd.contexts[0].ID),
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPut,
			StatusCode: http.StatusOK,
			Input: &contextapp.UpdateContext{
				Title: &newTitle,
			},
			GotResp: &contextapp.Context{},
			ExpResp: &contextapp.Context{
				ID:            sd.contexts[0].ID.String(),
				Title:         "Updated Context",
				Description:   sd.contexts[0].Description,
				Status:        sd.contexts[0].Status.String(),
				Summary:       sd.contexts[0].Summary,
				DebriefStatus: sd.contexts[0].DebriefStatus.String(),
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*contextapp.Context)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*contextapp.Context)
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
			URL:        fmt.Sprintf("/api/v1/contexts/%s", sd.contexts[0].ID),
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPut,
			StatusCode: http.StatusBadRequest,
			Input: &contextapp.UpdateContext{
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
			URL:        fmt.Sprintf("/api/v1/contexts/%s", sd.contexts[0].ID),
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
