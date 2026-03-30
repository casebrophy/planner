package threadapi_test

import (
	"net/http"

	"github.com/google/go-cmp/cmp"

	"github.com/casebrophy/planner/app/domain/threadapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/app/sdk/errs"
)

func create200(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "basic",
			URL:        "/api/v1/threads",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPost,
			StatusCode: http.StatusOK,
			Input: &threadapp.NewThreadEntry{
				SubjectType: "task",
				SubjectID:   sd.subjectID.String(),
				Kind:        "update",
				Content:     "test thread entry",
				Source:      "user",
			},
			GotResp: &threadapp.ThreadEntry{},
			ExpResp: &threadapp.ThreadEntry{
				SubjectType: "task",
				SubjectID:   sd.subjectID.String(),
				Kind:        "update",
				Content:     "test thread entry",
				Source:      "user",
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*threadapp.ThreadEntry)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*threadapp.ThreadEntry)
				expResp.ID = gotResp.ID
				expResp.CreatedAt = gotResp.CreatedAt
				return cmp.Diff(gotResp, expResp)
			},
		},
	}
}

func create400(_ seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "missing-kind",
			URL:        "/api/v1/threads",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPost,
			StatusCode: http.StatusBadRequest,
			Input: &threadapp.NewThreadEntry{
				SubjectType: "task",
				SubjectID:   "00000000-0000-0000-0000-000000000001",
				Content:     "some content",
				Source:      "user",
			},
			GotResp: &errs.Error{},
			ExpResp: &errs.Error{Code: errs.InvalidArgument, Message: "kind is required"},
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
			URL:        "/api/v1/threads",
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
