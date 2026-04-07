package observationbus_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/observationbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/unitest"
	"github.com/casebrophy/planner/business/types/observationkind"
)

func Test_Observation(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Observation")

	subjectID := uuid.New()

	obs, err := observationbus.TestSeedObservations(context.Background(), "task", subjectID, 2, db.BusDomain.Observation)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	unitest.Run(t, queryBySubject(db.BusDomain, obs, subjectID), "queryBySubject")
	unitest.Run(t, record(db.BusDomain, subjectID), "record")
	unitest.Run(t, queryByKind(db.BusDomain, obs), "queryByKind")
	unitest.Run(t, count(db.BusDomain, obs, subjectID), "count")
}

func record(busDomain dbtest.BusDomain, subjectID uuid.UUID) []unitest.Table {
	return []unitest.Table{
		{
			Name: "basic",
			ExpResp: observationbus.Observation{
				SubjectType: "task",
				SubjectID:   subjectID,
				Kind:        observationkind.Lesson,
				Data:        json.RawMessage(`{"key": "val"}`),
				Source:      "user",
				Confidence:  0.9,
				Weight:      1.0,
			},
			ExcFunc: func(ctx context.Context) any {
				no := observationbus.NewObservation{
					SubjectType: "task",
					SubjectID:   subjectID,
					Kind:        observationkind.Lesson,
					Data:        json.RawMessage(`{"key": "val"}`),
					Source:      "user",
					Confidence:  0.9,
					Weight:      1.0,
				}
				resp, err := busDomain.Observation.Record(ctx, no)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(observationbus.Observation)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(observationbus.Observation)
				expResp.ID = gotResp.ID
				expResp.CreatedAt = gotResp.CreatedAt
				return cmp.Diff(gotResp, expResp, cmpopts.EquateComparable(observationkind.Kind{}))
			},
		},
	}
}

func queryByKind(busDomain dbtest.BusDomain, obs []observationbus.Observation) []unitest.Table {
	// Note: the record test runs before this and adds one more lesson observation,
	// so the total number of lesson observations will be len(obs)+1.
	// We verify that all seeded observations are present in the results.
	return []unitest.Table{
		{
			Name:    "lesson",
			ExpResp: obs,
			ExcFunc: func(ctx context.Context) any {
				resp, err := busDomain.Observation.QueryByKind(ctx, observationkind.Lesson, page.New(1, 10))
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.([]observationbus.Observation)
				if !exists {
					return "error occurred"
				}
				expResp := exp.([]observationbus.Observation)
				// Build a set of returned IDs.
				gotIDs := make(map[uuid.UUID]bool, len(gotResp))
				for _, o := range gotResp {
					gotIDs[o.ID] = true
				}
				// Verify every seeded observation is present.
				for _, o := range expResp {
					if !gotIDs[o.ID] {
						return cmp.Diff(gotResp, expResp,
							cmpopts.EquateApproxTime(time.Second),
							cmpopts.EquateComparable(observationkind.Kind{}),
						)
					}
				}
				return ""
			},
		},
	}
}

func count(busDomain dbtest.BusDomain, obs []observationbus.Observation, subjectID uuid.UUID) []unitest.Table {
	// Note: the record test runs before this and adds one more observation with the same subjectID,
	// so the expected count is len(obs)+1.
	return []unitest.Table{
		{
			Name:    "by-subject",
			ExpResp: len(obs) + 1,
			ExcFunc: func(ctx context.Context) any {
				subjectType := obs[0].SubjectType
				filter := observationbus.QueryFilter{
					SubjectType: &subjectType,
					SubjectID:   &subjectID,
				}
				n, err := busDomain.Observation.Count(ctx, filter)
				if err != nil {
					return err
				}
				return n
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}

func queryBySubject(busDomain dbtest.BusDomain, obs []observationbus.Observation, subjectID uuid.UUID) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "all",
			ExpResp: obs,
			ExcFunc: func(ctx context.Context) any {
				resp, err := busDomain.Observation.QueryBySubject(ctx, "task", subjectID, page.New(1, 10))
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.([]observationbus.Observation)
				if !exists {
					return "error occurred"
				}
				expResp := exp.([]observationbus.Observation)
				return cmp.Diff(gotResp, expResp,
					cmpopts.EquateApproxTime(time.Second),
					cmpopts.EquateComparable(observationkind.Kind{}),
					cmpopts.SortSlices(func(a, b observationbus.Observation) bool {
						return a.ID.String() < b.ID.String()
					}),
				)
			},
		},
	}
}
