package observationapi_test

import (
	"fmt"
	"net/http"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/casebrophy/planner/app/domain/observationapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/observationbus"
)

func query200(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "by-subject",
			URL:        fmt.Sprintf("/api/v1/observations/task/%s?page=1&rows=10", sd.subjectID),
			Token:      apitest.TestAPIKey,
			Method:     http.MethodGet,
			StatusCode: http.StatusOK,
			GotResp:    &query.Result[observationapp.Observation]{},
			ExpResp: &query.Result[observationapp.Observation]{
				Total:       len(sd.observations),
				Page:        1,
				RowsPerPage: 10,
				Items:       toAppObservations(sd.observations),
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*query.Result[observationapp.Observation])
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*query.Result[observationapp.Observation])

				clearTimes := func(items []observationapp.Observation) {
					for i := range items {
						items[i].CreatedAt = ""
					}
				}
				clearTimes(gotResp.Items)
				clearTimes(expResp.Items)

				return cmp.Diff(gotResp, expResp,
					cmpopts.SortSlices(func(a, b observationapp.Observation) bool {
						return a.ID < b.ID
					}),
				)
			},
		},
	}
}

func query401(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "no-key",
			URL:        fmt.Sprintf("/api/v1/observations/task/%s?page=1&rows=10", sd.subjectID),
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

// toAppObservations converts business observations to app-layer observations for comparison.
func toAppObservations(obs []observationbus.Observation) []observationapp.Observation {
	items := make([]observationapp.Observation, len(obs))
	for i, o := range obs {
		items[i] = observationapp.Observation{
			ID:          o.ID.String(),
			SubjectType: o.SubjectType,
			SubjectID:   o.SubjectID.String(),
			Kind:        o.Kind.String(),
			Data:        o.Data,
			Source:      o.Source,
			Confidence:  o.Confidence,
			Weight:      o.Weight,
		}
	}
	return items
}
