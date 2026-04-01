package voiceingestapi_test

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/google/go-cmp/cmp"
)

func ingest401() []apitest.Table {
	return []apitest.Table{
		{
			Name:       "no-key",
			URL:        "/api/v1/ingest/voice",
			Token:      "",
			Method:     http.MethodPost,
			StatusCode: http.StatusUnauthorized,
			Input: map[string]string{
				"text": "remind me to wash the dishes",
			},
			GotResp: &errs.Error{},
			ExpResp: &errs.Error{Code: errs.Unauthenticated, Message: "missing api key"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}

func ingest400() []apitest.Table {
	return []apitest.Table{
		{
			Name:       "empty-text",
			URL:        "/api/v1/ingest/voice",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPost,
			StatusCode: http.StatusBadRequest,
			Input: map[string]string{
				"text": "",
			},
			GotResp: &errs.Error{},
			ExpResp: &errs.Error{Code: errs.InvalidArgument, Message: "text is required"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}
