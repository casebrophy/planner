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

	unitest.Run(t, record(db.BusDomain, subjectID), "record")
	unitest.Run(t, queryBySubject(db.BusDomain, obs, subjectID), "queryBySubject")
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
				Source:      "test-source",
				Confidence:  0.9,
				Weight:      1.0,
			},
			ExcFunc: func(ctx context.Context) any {
				no := observationbus.NewObservation{
					SubjectType: "task",
					SubjectID:   subjectID,
					Kind:        observationkind.Lesson,
					Data:        json.RawMessage(`{"key": "val"}`),
					Source:      "test-source",
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
				return cmp.Diff(gotResp, expResp)
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
				resp, err := busDomain.Observation.QueryBySubject(ctx, "task", subjectID, page.MustParse("1", "10"))
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
					cmpopts.SortSlices(func(a, b observationbus.Observation) bool {
						return a.ID.String() < b.ID.String()
					}),
				)
			},
		},
	}
}
