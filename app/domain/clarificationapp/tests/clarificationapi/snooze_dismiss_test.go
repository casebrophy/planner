package clarificationapi_test

import (
	"fmt"
	"net/http"

	"github.com/google/go-cmp/cmp"

	"github.com/casebrophy/planner/app/domain/clarificationapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
)

func snooze200(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "basic",
			URL:        fmt.Sprintf("/api/v1/clarifications/%s/snooze", sd.items[1].ID),
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPost,
			StatusCode: http.StatusOK,
			Input:      &clarificationapp.SnoozeInput{Hours: 24},
			GotResp:    &clarificationapp.ClarificationItem{},
			ExpResp:    &clarificationapp.ClarificationItem{Status: "snoozed"},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*clarificationapp.ClarificationItem)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*clarificationapp.ClarificationItem)
				if gotResp.Status != expResp.Status {
					return cmp.Diff(gotResp.Status, expResp.Status)
				}
				return ""
			},
		},
	}
}

func dismiss200(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "basic",
			URL:        fmt.Sprintf("/api/v1/clarifications/%s/dismiss", sd.items[2].ID),
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPost,
			StatusCode: http.StatusOK,
			GotResp:    &clarificationapp.ClarificationItem{},
			ExpResp:    &clarificationapp.ClarificationItem{Status: "dismissed"},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*clarificationapp.ClarificationItem)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*clarificationapp.ClarificationItem)
				if gotResp.Status != expResp.Status {
					return cmp.Diff(gotResp.Status, expResp.Status)
				}
				return ""
			},
		},
	}
}
