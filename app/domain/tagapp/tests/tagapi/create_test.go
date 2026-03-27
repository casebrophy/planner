package tagapi_test

import (
	"net/http"

	"github.com/google/go-cmp/cmp"

	"github.com/casebrophy/planner/app/domain/tagapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/app/sdk/errs"
)

func create200(_ seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "basic",
			URL:        "/api/v1/tags",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPost,
			StatusCode: http.StatusOK,
			Input:      &tagapp.NewTag{Name: "newtag"},
			GotResp:    &tagapp.Tag{},
			ExpResp:    &tagapp.Tag{Name: "newtag"},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*tagapp.Tag)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*tagapp.Tag)
				expResp.ID = gotResp.ID
				return cmp.Diff(gotResp, expResp)
			},
		},
	}
}

func create400(_ seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "missing-name",
			URL:        "/api/v1/tags",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPost,
			StatusCode: http.StatusBadRequest,
			Input:      &tagapp.NewTag{},
			GotResp:    &errs.Error{},
			ExpResp:    &errs.Error{Code: errs.InvalidArgument, Message: "name is required"},
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
			URL:        "/api/v1/tags",
			Token:      "",
			Method:     http.MethodPost,
			StatusCode: http.StatusUnauthorized,
			GotResp:    &errs.Error{},
			ExpResp:    &errs.Error{Code: errs.Unauthenticated, Message: "missing api key"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}
