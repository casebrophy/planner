package classificationcorrectionbus_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/casebrophy/planner/business/domain/classificationcorrectionbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/unitest"
)

func Test_ClassificationCorrection(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_ClassificationCorrection")

	corrs, err := classificationcorrectionbus.TestSeedCorrections(context.Background(), 2, db.BusDomain.ClassificationCorrection)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	unitest.Run(t, record(db.BusDomain), "record")
	unitest.Run(t, queryBySource(db.BusDomain, corrs), "queryBySource")
	unitest.Run(t, count(db.BusDomain, corrs), "count")
}

func record(busDomain dbtest.BusDomain) []unitest.Table {
	return []unitest.Table{
		{
			Name: "basic",
			ExpResp: classificationcorrectionbus.Correction{
				ClauseText:    "test clause",
				PredictedType: "task",
				Confidence:    0.9,
				ActualType:    "context",
				Source:        "clarification_answered",
			},
			ExcFunc: func(ctx context.Context) any {
				nc := classificationcorrectionbus.NewCorrection{
					ClauseText:    "test clause",
					PredictedType: "task",
					Confidence:    0.9,
					ActualType:    "context",
					Source:        "clarification_answered",
				}
				resp, err := busDomain.ClassificationCorrection.Record(ctx, nc)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(classificationcorrectionbus.Correction)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(classificationcorrectionbus.Correction)
				expResp.ID = gotResp.ID
				expResp.CreatedAt = gotResp.CreatedAt
				return cmp.Diff(gotResp, expResp)
			},
		},
	}
}

func queryBySource(busDomain dbtest.BusDomain, corrs []classificationcorrectionbus.Correction) []unitest.Table {
	// Note: the record test runs before this and adds one more correction,
	// so the total number of corrections will be len(corrs)+1.
	// We verify that all seeded corrections are present in the results.
	return []unitest.Table{
		{
			Name:    "clarification_answered",
			ExpResp: corrs,
			ExcFunc: func(ctx context.Context) any {
				resp, err := busDomain.ClassificationCorrection.QueryBySource(ctx, "clarification_answered", page.New(1, 10))
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.([]classificationcorrectionbus.Correction)
				if !exists {
					return "error occurred"
				}
				expResp := exp.([]classificationcorrectionbus.Correction)
				// Build a set of returned IDs.
				gotIDs := make(map[string]bool, len(gotResp))
				for _, c := range gotResp {
					gotIDs[c.ID.String()] = true
				}
				// Verify every seeded correction is present.
				for _, c := range expResp {
					if !gotIDs[c.ID.String()] {
						return cmp.Diff(gotResp, expResp,
							cmpopts.EquateApproxTime(time.Second),
						)
					}
				}
				return ""
			},
		},
	}
}

func count(busDomain dbtest.BusDomain, corrs []classificationcorrectionbus.Correction) []unitest.Table {
	// Note: the record test runs before this and adds one more correction,
	// so the expected count is len(corrs)+1.
	return []unitest.Table{
		{
			Name:    "by-source",
			ExpResp: len(corrs) + 1,
			ExcFunc: func(ctx context.Context) any {
				source := "clarification_answered"
				filter := classificationcorrectionbus.QueryFilter{
					Source: &source,
				}
				n, err := busDomain.ClassificationCorrection.Count(ctx, filter)
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
