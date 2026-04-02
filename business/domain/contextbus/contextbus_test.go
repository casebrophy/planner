package contextbus_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/unitest"
	"github.com/casebrophy/planner/business/types/debriefstatus"
)

func Test_Context(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Context")

	contexts, err := contextbus.TestSeedContexts(context.Background(), 2, db.BusDomain.Context)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	unitest.Run(t, query(db.BusDomain, contexts), "query")
	unitest.Run(t, create(db.BusDomain, contexts), "create")
	unitest.Run(t, update(db.BusDomain, contexts), "update")
	unitest.Run(t, delete(db.BusDomain, contexts), "delete")
}

func query(busDomain dbtest.BusDomain, contexts []contextbus.Context) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "all",
			ExpResp: contexts,
			ExcFunc: func(ctx context.Context) any {
				resp, err := busDomain.Context.Query(ctx, contextbus.QueryFilter{}, contextbus.DefaultOrderBy, page.New(1, 10))
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.([]contextbus.Context)
				if !exists {
					return "error occurred"
				}
				expResp := exp.([]contextbus.Context)
				return cmp.Diff(gotResp, expResp,
					cmpopts.EquateApproxTime(time.Second),
					cmpopts.EquateComparable(debriefstatus.Status{}),
					cmpopts.SortSlices(func(a, b contextbus.Context) bool {
						return a.ID.String() < b.ID.String()
					}),
				)
			},
		},
		{
			Name:    "byid",
			ExpResp: contexts[0],
			ExcFunc: func(ctx context.Context) any {
				resp, err := busDomain.Context.QueryByID(ctx, contexts[0].ID)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(contextbus.Context)
				if !exists {
					return "error occurred"
				}
				return cmp.Diff(gotResp, exp.(contextbus.Context), cmpopts.EquateApproxTime(time.Second), cmpopts.EquateComparable(debriefstatus.Status{}))
			},
		},
	}
}

func create(busDomain dbtest.BusDomain, _ []contextbus.Context) []unitest.Table {
	return []unitest.Table{
		{
			Name: "basic",
			ExpResp: contextbus.Context{
				Title:         "New Context",
				Description:   "New Description",
				Status:        contextbus.Active,
				DebriefStatus: debriefstatus.Pending,
			},
			ExcFunc: func(ctx context.Context) any {
				nc := contextbus.NewContext{
					Title:       "New Context",
					Description: "New Description",
				}
				resp, err := busDomain.Context.Create(ctx, nc)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(contextbus.Context)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(contextbus.Context)
				expResp.ID = gotResp.ID
				expResp.CreatedAt = gotResp.CreatedAt
				expResp.UpdatedAt = gotResp.UpdatedAt
				return cmp.Diff(gotResp, expResp, cmpopts.EquateComparable(debriefstatus.Status{}))
			},
		},
	}
}

func update(busDomain dbtest.BusDomain, contexts []contextbus.Context) []unitest.Table {
	newTitle := "Updated Context"

	return []unitest.Table{
		{
			Name: "basic",
			ExpResp: contextbus.Context{
				ID:            contexts[0].ID,
				Title:         "Updated Context",
				Description:   contexts[0].Description,
				Status:        contexts[0].Status,
				Summary:       contexts[0].Summary,
				DebriefStatus: contexts[0].DebriefStatus,
				CreatedAt:     contexts[0].CreatedAt,
			},
			ExcFunc: func(ctx context.Context) any {
				uc := contextbus.UpdateContext{
					Title: &newTitle,
				}
				resp, err := busDomain.Context.Update(ctx, contexts[0], uc)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(contextbus.Context)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(contextbus.Context)
				expResp.UpdatedAt = gotResp.UpdatedAt
				return cmp.Diff(gotResp, expResp, cmpopts.EquateApproxTime(time.Second), cmpopts.EquateComparable(debriefstatus.Status{}))
			},
		},
	}
}

func delete(busDomain dbtest.BusDomain, contexts []contextbus.Context) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "basic",
			ExpResp: nil,
			ExcFunc: func(ctx context.Context) any {
				if err := busDomain.Context.Delete(ctx, contexts[1]); err != nil {
					return err
				}
				return nil
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}
