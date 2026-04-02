package clarificationbus_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/unitest"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/clarificationstatus"
)

func Test_Clarification(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Clarification")

	items, err := clarificationbus.TestSeedClarifications(context.Background(), 2, db.BusDomain.Clarification)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	unitest.Run(t, query(db.BusDomain, items), "query")
	unitest.Run(t, create(db.BusDomain, items), "create")
	unitest.Run(t, resolve(db.BusDomain, items), "resolve")
	unitest.Run(t, snooze(db.BusDomain, items), "snooze")
}

func query(busDomain dbtest.BusDomain, items []clarificationbus.ClarificationItem) []unitest.Table {
	pendingStatus := clarificationstatus.Pending

	return []unitest.Table{
		{
			Name:    "pending",
			ExpResp: items,
			ExcFunc: func(ctx context.Context) any {
				filter := clarificationbus.QueryFilter{
					Status: &pendingStatus,
				}
				resp, err := busDomain.Clarification.Query(ctx, filter, clarificationbus.DefaultOrderBy, page.New(1, 10))
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.([]clarificationbus.ClarificationItem)
				if !exists {
					return "error occurred"
				}
				expResp := exp.([]clarificationbus.ClarificationItem)
				return cmp.Diff(gotResp, expResp,
					cmpopts.EquateApproxTime(time.Second),
					cmpopts.EquateComparable(clarificationkind.Kind{}, clarificationstatus.Status{}),
					cmpopts.SortSlices(func(a, b clarificationbus.ClarificationItem) bool {
						return a.ID.String() < b.ID.String()
					}),
				)
			},
		},
	}
}

func create(busDomain dbtest.BusDomain, _ []clarificationbus.ClarificationItem) []unitest.Table {
	return []unitest.Table{
		{
			Name: "basic",
			ExpResp: clarificationbus.ClarificationItem{
				Kind:          clarificationkind.NewContext,
				Status:        clarificationstatus.Pending,
				SubjectType:   "context",
				Question:      "Should this be a new context?",
				AnswerOptions: json.RawMessage(`["yes","no"]`),
				PriorityScore: 0.9,
			},
			ExcFunc: func(ctx context.Context) any {
				ni := clarificationbus.NewClarificationItem{
					Kind:          clarificationkind.NewContext,
					SubjectType:   "context",
					SubjectID:     uuid.New(),
					Question:      "Should this be a new context?",
					AnswerOptions: json.RawMessage(`["yes","no"]`),
					PriorityScore: 0.9,
				}
				resp, err := busDomain.Clarification.Create(ctx, ni)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(clarificationbus.ClarificationItem)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(clarificationbus.ClarificationItem)
				expResp.ID = gotResp.ID
				expResp.SubjectID = gotResp.SubjectID
				expResp.CreatedAt = gotResp.CreatedAt
				expResp.PriorityScore = gotResp.PriorityScore
				return cmp.Diff(gotResp, expResp, cmpopts.EquateComparable(clarificationkind.Kind{}, clarificationstatus.Status{}))
			},
		},
	}
}

func resolve(busDomain dbtest.BusDomain, items []clarificationbus.ClarificationItem) []unitest.Table {
	return []unitest.Table{
		{
			Name: "basic",
			ExpResp: clarificationbus.ClarificationItem{
				Status: clarificationstatus.Resolved,
			},
			ExcFunc: func(ctx context.Context) any {
				rc := clarificationbus.ResolveClarificationItem{
					Answer: json.RawMessage(`"yes"`),
				}
				resp, err := busDomain.Clarification.Resolve(ctx, items[0], rc)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(clarificationbus.ClarificationItem)
				if !exists {
					return "error occurred"
				}
				// Only verify the status transition occurred correctly.
				if gotResp.Status != clarificationstatus.Resolved {
					return cmp.Diff(gotResp.Status.String(), clarificationstatus.Resolved.String())
				}
				return ""
			},
		},
	}
}

func snooze(busDomain dbtest.BusDomain, items []clarificationbus.ClarificationItem) []unitest.Table {
	return []unitest.Table{
		{
			Name: "basic",
			ExpResp: clarificationbus.ClarificationItem{
				Status: clarificationstatus.Snoozed,
			},
			ExcFunc: func(ctx context.Context) any {
				until := time.Now().Add(24 * time.Hour)
				resp, err := busDomain.Clarification.Snooze(ctx, items[1], until)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(clarificationbus.ClarificationItem)
				if !exists {
					return "error occurred"
				}
				// Only verify the status transition occurred correctly.
				if gotResp.Status != clarificationstatus.Snoozed {
					return cmp.Diff(gotResp.Status.String(), clarificationstatus.Snoozed.String())
				}
				return ""
			},
		},
	}
}
