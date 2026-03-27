package emailapi_test

import (
	"net/http"

	"github.com/google/go-cmp/cmp"

	"github.com/casebrophy/planner/app/domain/emailapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
)

func query200(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "all",
			URL:        "/api/v1/emails?page=1&rows=10",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodGet,
			StatusCode: http.StatusOK,
			GotResp:    &query.Result[emailapp.Email]{},
			ExpResp:    &query.Result[emailapp.Email]{},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*query.Result[emailapp.Email])
				if !exists {
					return "error occurred"
				}
				if gotResp.Total != len(sd.emails) {
					return cmp.Diff(gotResp.Total, len(sd.emails))
				}
				return ""
			},
		},
	}
}

func query401(_ seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "no-key",
			URL:        "/api/v1/emails?page=1&rows=10",
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
