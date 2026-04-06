package contextapi_test

import (
	"net/http"

	"github.com/google/go-cmp/cmp"

	"github.com/casebrophy/planner/app/domain/contextapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/app/sdk/errs"
)

func create200(_ seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "basic",
			URL:        "/api/v1/contexts",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPost,
			StatusCode: http.StatusOK,
			Input: &contextapp.NewContext{
				Title:       "Test Context",
				Description: "Test Description",
			},
			GotResp: &contextapp.Context{},
			ExpResp: &contextapp.Context{
				Title:         "Test Context",
				Description:   "Test Description",
				Status:        "active",
				Kind:          "project",
				DebriefStatus: "pending",
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*contextapp.Context)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*contextapp.Context)
				expResp.ID = gotResp.ID
				expResp.CreatedAt = gotResp.CreatedAt
				expResp.UpdatedAt = gotResp.UpdatedAt
				return cmp.Diff(gotResp, expResp)
			},
		},
	}
}

func create400(_ seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "missing-title",
			URL:        "/api/v1/contexts",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPost,
			StatusCode: http.StatusBadRequest,
			Input:      &contextapp.NewContext{},
			GotResp:    &errs.Error{},
			ExpResp:    &errs.Error{Code: errs.InvalidArgument, Message: "title is required"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}

func create401(_ seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "no-key",
			URL:        "/api/v1/contexts",
			Token:      "",
			Method:     http.MethodPost,
			StatusCode: http.StatusUnauthorized,
			GotResp:    &errs.Error{},
			ExpResp:    &errs.Error{Code: errs.Unauthenticated, Message: "missing api key"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
		{
			Name:       "wrong-key",
			URL:        "/api/v1/contexts",
			Token:      "wrong-key",
			Method:     http.MethodPost,
			StatusCode: http.StatusUnauthorized,
			GotResp:    &errs.Error{},
			ExpResp:    &errs.Error{Code: errs.Unauthenticated, Message: "invalid api key"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}
