package clarificationapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/go-cmp/cmp"

	"github.com/casebrophy/planner/app/domain/clarificationapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/app/sdk/errs"
)

func resolve200(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "basic",
			URL:        fmt.Sprintf("/api/v1/clarifications/%s/resolve", sd.items[0].ID),
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPost,
			StatusCode: http.StatusOK,
			Input:      &clarificationapp.ResolveInput{Answer: json.RawMessage(`"yes"`)},
			GotResp:    &clarificationapp.ClarificationItem{},
			ExpResp:    &clarificationapp.ClarificationItem{Status: "resolved"},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*clarificationapp.ClarificationItem)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*clarificationapp.ClarificationItem)
				// Only verify status transition.
				if gotResp.Status != expResp.Status {
					return cmp.Diff(gotResp.Status, expResp.Status)
				}
				return ""
			},
		},
	}
}

func resolve401(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "no-key",
			URL:        fmt.Sprintf("/api/v1/clarifications/%s/resolve", sd.items[0].ID),
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
