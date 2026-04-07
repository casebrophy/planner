package observationapi_test

import (
	"encoding/json"
	"net/http"

	"github.com/google/go-cmp/cmp"

	"github.com/casebrophy/planner/app/domain/observationapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/app/sdk/errs"
)

func record200(sd seedData) []apitest.Table {
	conf := float32(0.9)
	weight := float32(1.0)

	return []apitest.Table{
		{
			Name:       "basic",
			URL:        "/api/v1/observations",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPost,
			StatusCode: http.StatusOK,
			Input: &observationapp.NewObservation{
				SubjectType: "task",
				SubjectID:   sd.subjectID.String(),
				Kind:        "lesson",
				Data:        json.RawMessage(`{"key":"value"}`),
				Source:      "user",
				Confidence:  &conf,
				Weight:      &weight,
			},
			GotResp: &observationapp.Observation{},
			ExpResp: &observationapp.Observation{
				SubjectType: "task",
				SubjectID:   sd.subjectID.String(),
				Kind:        "lesson",
				Data:        json.RawMessage(`{"key":"value"}`),
				Source:      "user",
				Confidence:  0.9,
				Weight:      1.0,
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*observationapp.Observation)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*observationapp.Observation)
				expResp.ID = gotResp.ID
				expResp.CreatedAt = gotResp.CreatedAt
				return cmp.Diff(gotResp, expResp)
			},
		},
		{
			Name:       "defaults",
			URL:        "/api/v1/observations",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPost,
			StatusCode: http.StatusOK,
			Input: &observationapp.NewObservation{
				SubjectType: "task",
				SubjectID:   sd.subjectID.String(),
				Kind:        "lesson",
				Data:        json.RawMessage(`{"key":"value"}`),
			},
			GotResp: &observationapp.Observation{},
			ExpResp: &observationapp.Observation{
				SubjectType: "task",
				SubjectID:   sd.subjectID.String(),
				Kind:        "lesson",
				Data:        json.RawMessage(`{"key":"value"}`),
				Source:      "user",
				Confidence:  1.0,
				Weight:      1.0,
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*observationapp.Observation)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*observationapp.Observation)
				expResp.ID = gotResp.ID
				expResp.CreatedAt = gotResp.CreatedAt
				return cmp.Diff(gotResp, expResp)
			},
		},
	}
}

func record400(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "missing-kind",
			URL:        "/api/v1/observations",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPost,
			StatusCode: http.StatusBadRequest,
			Input: &observationapp.NewObservation{
				SubjectType: "task",
				SubjectID:   sd.subjectID.String(),
				Data:        json.RawMessage(`{"key":"value"}`),
				Source:      "user",
			},
			GotResp: &errs.Error{},
			ExpResp: &errs.Error{Code: errs.InvalidArgument, Message: "kind is required"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
		{
			Name:       "missing-subject-type",
			URL:        "/api/v1/observations",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPost,
			StatusCode: http.StatusBadRequest,
			Input: &observationapp.NewObservation{
				SubjectID: sd.subjectID.String(),
				Kind:      "lesson",
				Data:      json.RawMessage(`{"key":"value"}`),
			},
			GotResp: &errs.Error{},
			ExpResp: &errs.Error{Code: errs.InvalidArgument, Message: "subjectType is required"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
		{
			Name:       "missing-subject-id",
			URL:        "/api/v1/observations",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPost,
			StatusCode: http.StatusBadRequest,
			Input: &observationapp.NewObservation{
				SubjectType: "task",
				Kind:        "lesson",
				Data:        json.RawMessage(`{"key":"value"}`),
			},
			GotResp: &errs.Error{},
			ExpResp: &errs.Error{Code: errs.InvalidArgument, Message: "subjectId is required"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
		{
			Name:       "missing-data",
			URL:        "/api/v1/observations",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPost,
			StatusCode: http.StatusBadRequest,
			Input: map[string]any{
				"subjectType": "task",
				"subjectId":   sd.subjectID.String(),
				"kind":        "lesson",
			},
			GotResp: &errs.Error{},
			ExpResp: &errs.Error{Code: errs.InvalidArgument, Message: "data is required"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
		{
			Name:       "invalid-kind",
			URL:        "/api/v1/observations",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPost,
			StatusCode: http.StatusBadRequest,
			Input: &observationapp.NewObservation{
				SubjectType: "task",
				SubjectID:   sd.subjectID.String(),
				Kind:        "bogus",
				Data:        json.RawMessage(`{"key":"value"}`),
			},
			GotResp: &errs.Error{},
			ExpResp: &errs.Error{Code: errs.InvalidArgument},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*errs.Error)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*errs.Error)
				if gotResp.Code != expResp.Code {
					return cmp.Diff(gotResp.Code, expResp.Code)
				}
				return ""
			},
		},
		{
			Name:       "invalid-subject-id",
			URL:        "/api/v1/observations",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPost,
			StatusCode: http.StatusBadRequest,
			Input: &observationapp.NewObservation{
				SubjectType: "task",
				SubjectID:   "not-a-uuid",
				Kind:        "lesson",
				Data:        json.RawMessage(`{"key":"value"}`),
			},
			GotResp: &errs.Error{},
			ExpResp: &errs.Error{Code: errs.InvalidArgument},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*errs.Error)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*errs.Error)
				if gotResp.Code != expResp.Code {
					return cmp.Diff(gotResp.Code, expResp.Code)
				}
				return ""
			},
		},
	}
}

func record401(_ seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "no-key",
			URL:        "/api/v1/observations",
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
