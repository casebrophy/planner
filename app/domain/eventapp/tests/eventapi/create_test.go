package eventapi_test

import (
	"net/http"

	"github.com/google/go-cmp/cmp"

	"github.com/casebrophy/planner/app/domain/eventapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/app/sdk/errs"
)

func create200() []apitest.Table {
	return []apitest.Table{
		{
			Name:       "basic",
			URL:        "/api/v1/events",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPost,
			StatusCode: http.StatusOK,
			Input: &eventapp.NewEvent{
				Title:    "Dentist Appointment",
				StartsAt: "2026-04-05T14:00:00Z",
				EndsAt:   "2026-04-05T15:00:00Z",
				Location: ptrString("123 Main St"),
			},
			GotResp: &eventapp.Event{},
			ExpResp: &eventapp.Event{
				Title:       "Dentist Appointment",
				Description: "",
				StartsAt:    "2026-04-05T14:00:00Z",
				EndsAt:      "2026-04-05T15:00:00Z",
				AllDay:      false,
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*eventapp.Event)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*eventapp.Event)
				expResp.ID = gotResp.ID
				expResp.CreatedAt = gotResp.CreatedAt
				expResp.UpdatedAt = gotResp.UpdatedAt
				expResp.Location = gotResp.Location
				expResp.RawInputID = gotResp.RawInputID
				return cmp.Diff(gotResp, expResp)
			},
		},
	}
}

func create400() []apitest.Table {
	return []apitest.Table{
		{
			Name:       "missing-title",
			URL:        "/api/v1/events",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPost,
			StatusCode: http.StatusBadRequest,
			Input: &eventapp.NewEvent{
				StartsAt: "2026-04-05T14:00:00Z",
				EndsAt:   "2026-04-05T15:00:00Z",
			},
			GotResp: &errs.Error{},
			ExpResp: &errs.Error{Code: errs.InvalidArgument, Message: "title is required"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
		{
			Name:       "missing-starts-at",
			URL:        "/api/v1/events",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPost,
			StatusCode: http.StatusBadRequest,
			Input: &eventapp.NewEvent{
				Title:  "Test Event",
				EndsAt: "2026-04-05T15:00:00Z",
			},
			GotResp: &errs.Error{},
			ExpResp: &errs.Error{Code: errs.InvalidArgument, Message: "startsAt is required"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
		{
			Name:       "missing-ends-at",
			URL:        "/api/v1/events",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPost,
			StatusCode: http.StatusBadRequest,
			Input: &eventapp.NewEvent{
				Title:    "Test Event",
				StartsAt: "2026-04-05T14:00:00Z",
			},
			GotResp: &errs.Error{},
			ExpResp: &errs.Error{Code: errs.InvalidArgument, Message: "endsAt is required"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}

func create401() []apitest.Table {
	return []apitest.Table{
		{
			Name:       "no-key",
			URL:        "/api/v1/events",
			Token:      "",
			Method:     http.MethodPost,
			StatusCode: http.StatusUnauthorized,
			Input: &eventapp.NewEvent{
				Title:    "Test Event",
				StartsAt: "2026-04-05T14:00:00Z",
				EndsAt:   "2026-04-05T15:00:00Z",
			},
			GotResp: &errs.Error{},
			ExpResp: &errs.Error{Code: errs.Unauthenticated, Message: "missing api key"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
		{
			Name:       "wrong-key",
			URL:        "/api/v1/events",
			Token:      "wrong-key",
			Method:     http.MethodPost,
			StatusCode: http.StatusUnauthorized,
			Input: &eventapp.NewEvent{
				Title:    "Test Event",
				StartsAt: "2026-04-05T14:00:00Z",
				EndsAt:   "2026-04-05T15:00:00Z",
			},
			GotResp: &errs.Error{},
			ExpResp: &errs.Error{Code: errs.Unauthenticated, Message: "invalid api key"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}

func ptrString(s string) *string {
	return &s
}
