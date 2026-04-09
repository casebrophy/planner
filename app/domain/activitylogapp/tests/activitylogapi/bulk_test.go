package activitylogapi_test

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/casebrophy/planner/app/domain/activitylogapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/app/sdk/errs"
)

func bulk200(sd seedData) []apitest.Table {
	from := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	to := time.Now().Add(24 * time.Hour).Format(time.RFC3339)

	return []apitest.Table{
		{
			Name: "by-subject",
			URL: fmt.Sprintf("/api/v1/activity-logs/bulk?subject_type=task&subject_ids=%s,%s&from=%s&to=%s",
				sd.subjectID1.String(), sd.subjectID2.String(), from, to),
			Token:      apitest.TestAPIKey,
			Method:     http.MethodGet,
			StatusCode: http.StatusOK,
			GotResp:    &activitylogapp.BulkLogsResponse{},
			ExpResp: &activitylogapp.BulkLogsResponse{
				Items: map[string][]activitylogapp.ActivityLog{
					sd.subjectID1.String(): {
						{
							SubjectType: "task",
							SubjectID:   sd.subjectID1.String(),
							Value:       sd.logs[0].Value,
						},
						{
							SubjectType: "task",
							SubjectID:   sd.subjectID1.String(),
							Value:       sd.logs[1].Value,
						},
					},
					sd.subjectID2.String(): {
						{
							SubjectType: "task",
							SubjectID:   sd.subjectID2.String(),
							Value:       sd.logs[2].Value,
						},
					},
				},
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*activitylogapp.BulkLogsResponse)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*activitylogapp.BulkLogsResponse)

				// Clear dynamic fields (ID, LoggedAt) for comparison.
				clearDynamic := func(items map[string][]activitylogapp.ActivityLog) {
					for k := range items {
						for i := range items[k] {
							items[k][i].ID = ""
							items[k][i].LoggedAt = ""
						}
					}
				}
				clearDynamic(gotResp.Items)
				clearDynamic(expResp.Items)

				return cmp.Diff(gotResp, expResp,
					cmpopts.SortSlices(func(a, b activitylogapp.ActivityLog) bool {
						aVal := ""
						if a.Value != nil {
							aVal = *a.Value
						}
						bVal := ""
						if b.Value != nil {
							bVal = *b.Value
						}
						return aVal < bVal
					}),
				)
			},
		},
		{
			Name: "single-subject",
			URL: fmt.Sprintf("/api/v1/activity-logs/bulk?subject_type=task&subject_ids=%s&from=%s&to=%s",
				sd.subjectID1.String(), from, to),
			Token:      apitest.TestAPIKey,
			Method:     http.MethodGet,
			StatusCode: http.StatusOK,
			GotResp:    &activitylogapp.BulkLogsResponse{},
			ExpResp: &activitylogapp.BulkLogsResponse{
				Items: map[string][]activitylogapp.ActivityLog{
					sd.subjectID1.String(): {
						{
							SubjectType: "task",
							SubjectID:   sd.subjectID1.String(),
							Value:       sd.logs[0].Value,
						},
						{
							SubjectType: "task",
							SubjectID:   sd.subjectID1.String(),
							Value:       sd.logs[1].Value,
						},
					},
				},
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*activitylogapp.BulkLogsResponse)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*activitylogapp.BulkLogsResponse)

				clearDynamic := func(items map[string][]activitylogapp.ActivityLog) {
					for k := range items {
						for i := range items[k] {
							items[k][i].ID = ""
							items[k][i].LoggedAt = ""
						}
					}
				}
				clearDynamic(gotResp.Items)
				clearDynamic(expResp.Items)

				return cmp.Diff(gotResp, expResp,
					cmpopts.SortSlices(func(a, b activitylogapp.ActivityLog) bool {
						aVal := ""
						if a.Value != nil {
							aVal = *a.Value
						}
						bVal := ""
						if b.Value != nil {
							bVal = *b.Value
						}
						return aVal < bVal
					}),
				)
			},
		},
	}
}

func bulk400(sd seedData) []apitest.Table {
	from := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	to := time.Now().Add(24 * time.Hour).Format(time.RFC3339)

	return []apitest.Table{
		{
			Name: "missing-subject-type",
			URL: fmt.Sprintf("/api/v1/activity-logs/bulk?subject_ids=%s&from=%s&to=%s",
				sd.subjectID1.String(), from, to),
			Token:      apitest.TestAPIKey,
			Method:     http.MethodGet,
			StatusCode: http.StatusBadRequest,
			GotResp:    &errs.Error{},
			ExpResp:    &errs.Error{Code: errs.InvalidArgument, Message: "subject_type is required"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
		{
			Name: "missing-subject-ids",
			URL: fmt.Sprintf("/api/v1/activity-logs/bulk?subject_type=task&from=%s&to=%s",
				from, to),
			Token:      apitest.TestAPIKey,
			Method:     http.MethodGet,
			StatusCode: http.StatusBadRequest,
			GotResp:    &errs.Error{},
			ExpResp:    &errs.Error{Code: errs.InvalidArgument, Message: "subject_ids is required"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
		{
			Name: "invalid-subject-id",
			URL: fmt.Sprintf("/api/v1/activity-logs/bulk?subject_type=task&subject_ids=not-a-uuid&from=%s&to=%s",
				from, to),
			Token:      apitest.TestAPIKey,
			Method:     http.MethodGet,
			StatusCode: http.StatusBadRequest,
			GotResp:    &errs.Error{},
			ExpResp:    &errs.Error{Code: errs.InvalidArgument},
			CmpFunc: func(got any, exp any) string {
				gotResp := got.(*errs.Error)
				expResp := exp.(*errs.Error)
				if gotResp.Code != expResp.Code {
					return fmt.Sprintf("code: got %d, exp %d", gotResp.Code, expResp.Code)
				}
				return ""
			},
		},
		{
			Name: "missing-from",
			URL: fmt.Sprintf("/api/v1/activity-logs/bulk?subject_type=task&subject_ids=%s&to=%s",
				sd.subjectID1.String(), to),
			Token:      apitest.TestAPIKey,
			Method:     http.MethodGet,
			StatusCode: http.StatusBadRequest,
			GotResp:    &errs.Error{},
			ExpResp:    &errs.Error{Code: errs.InvalidArgument, Message: "from is required"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
		{
			Name: "missing-to",
			URL: fmt.Sprintf("/api/v1/activity-logs/bulk?subject_type=task&subject_ids=%s&from=%s",
				sd.subjectID1.String(), from),
			Token:      apitest.TestAPIKey,
			Method:     http.MethodGet,
			StatusCode: http.StatusBadRequest,
			GotResp:    &errs.Error{},
			ExpResp:    &errs.Error{Code: errs.InvalidArgument, Message: "to is required"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}

func bulk401(sd seedData) []apitest.Table {
	from := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	to := time.Now().Add(24 * time.Hour).Format(time.RFC3339)

	return []apitest.Table{
		{
			Name: "no-key",
			URL: fmt.Sprintf("/api/v1/activity-logs/bulk?subject_type=task&subject_ids=%s&from=%s&to=%s",
				sd.subjectID1.String(), from, to),
			Token:      "",
			Method:     http.MethodGet,
			StatusCode: http.StatusUnauthorized,
			GotResp:    &errs.Error{},
			ExpResp:    &errs.Error{Code: errs.Unauthenticated, Message: "missing api key"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
		{
			Name: "wrong-key",
			URL: fmt.Sprintf("/api/v1/activity-logs/bulk?subject_type=task&subject_ids=%s&from=%s&to=%s",
				sd.subjectID1.String(), from, to),
			Token:      "wrong-key",
			Method:     http.MethodGet,
			StatusCode: http.StatusUnauthorized,
			GotResp:    &errs.Error{},
			ExpResp:    &errs.Error{Code: errs.Unauthenticated, Message: "invalid api key"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}
