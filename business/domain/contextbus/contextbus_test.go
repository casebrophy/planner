package contextbus_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/unitest"
	"github.com/casebrophy/planner/business/types/contextkind"
	"github.com/casebrophy/planner/business/types/debriefstatus"
	"github.com/casebrophy/planner/business/types/taskstatus"
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
	unitest.Run(t, listRules(db.BusDomain), "listRules")
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
					cmpopts.EquateComparable(debriefstatus.Status{}, contextkind.Kind{}),
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
				return cmp.Diff(gotResp, exp.(contextbus.Context), cmpopts.EquateApproxTime(time.Second), cmpopts.EquateComparable(debriefstatus.Status{}, contextkind.Kind{}))
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
				Kind:          contextkind.Project,
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
				return cmp.Diff(gotResp, expResp, cmpopts.EquateComparable(debriefstatus.Status{}, contextkind.Kind{}))
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
				Kind:          contexts[0].Kind,
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
				return cmp.Diff(gotResp, expResp, cmpopts.EquateApproxTime(time.Second), cmpopts.EquateComparable(debriefstatus.Status{}, contextkind.Kind{}))
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

func listRules(busDomain dbtest.BusDomain) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "create list context succeeds",
			ExpResp: contextbus.Active,
			ExcFunc: func(ctx context.Context) any {
				c, err := busDomain.Context.Create(ctx, contextbus.NewContext{
					Title: "My List",
					Kind:  contextkind.List,
				})
				if err != nil {
					return err
				}
				return c.Status
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
		{
			Name:    "pause list context fails",
			ExpResp: "error",
			ExcFunc: func(ctx context.Context) any {
				c, err := busDomain.Context.Create(ctx, contextbus.NewContext{
					Title: "Pauseable List?",
					Kind:  contextkind.List,
				})
				if err != nil {
					return err
				}
				paused := contextbus.Paused
				_, err = busDomain.Context.Update(ctx, c, contextbus.UpdateContext{
					Status: &paused,
				})
				if err != nil {
					return "error"
				}
				return "no error"
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
		{
			Name:    "close list context succeeds",
			ExpResp: contextbus.Closed,
			ExcFunc: func(ctx context.Context) any {
				c, err := busDomain.Context.Create(ctx, contextbus.NewContext{
					Title: "Closeable List",
					Kind:  contextkind.List,
				})
				if err != nil {
					return err
				}
				closed := contextbus.Closed
				updated, err := busDomain.Context.Update(ctx, c, contextbus.UpdateContext{
					Status: &closed,
				})
				if err != nil {
					return err
				}
				return updated.Status
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
		{
			Name:    "create list with area parent succeeds",
			ExpResp: "ok",
			ExcFunc: func(ctx context.Context) any {
				area, err := busDomain.Context.Create(ctx, contextbus.NewContext{
					Title: "My Area",
					Kind:  contextkind.Area,
				})
				if err != nil {
					return err
				}
				_, err = busDomain.Context.Create(ctx, contextbus.NewContext{
					Title:           "My List Under Area",
					Kind:            contextkind.List,
					ParentContextID: &area.ID,
				})
				if err != nil {
					return err
				}
				return "ok"
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
		{
			Name:    "create list with project parent fails",
			ExpResp: "error",
			ExcFunc: func(ctx context.Context) any {
				project, err := busDomain.Context.Create(ctx, contextbus.NewContext{
					Title: "My Project",
					Kind:  contextkind.Project,
				})
				if err != nil {
					return err
				}
				_, err = busDomain.Context.Create(ctx, contextbus.NewContext{
					Title:           "My List Under Project",
					Kind:            contextkind.List,
					ParentContextID: &project.ID,
				})
				if err != nil {
					return "error"
				}
				return "no error"
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
		{
			Name:    "reset list sets done tasks to open",
			ExpResp: "ok",
			ExcFunc: func(ctx context.Context) any {
				// Create a list context
				list, err := busDomain.Context.Create(ctx, contextbus.NewContext{
					Title: "Reset Test List",
					Kind:  contextkind.List,
				})
				if err != nil {
					return err
				}

				// Create one Done task and one Open task
				doneTask, err := busDomain.Task.Create(ctx, taskbus.NewTask{
					ContextID: &list.ID,
					Title:     "Done Item",
					Status:    taskstatus.Done,
				})
				if err != nil {
					return err
				}
				_, err = busDomain.Task.Create(ctx, taskbus.NewTask{
					ContextID: &list.ID,
					Title:     "Open Item",
					Status:    taskstatus.Open,
				})
				if err != nil {
					return err
				}

				// Reset the list
				if err := busDomain.Task.ResetByContext(ctx, list.ID); err != nil {
					return err
				}

				// Verify done task is now open
				refreshed, err := busDomain.Task.QueryByID(ctx, doneTask.ID)
				if err != nil {
					return err
				}
				if refreshed.Status != taskstatus.Open {
					return fmt.Errorf("expected open, got %s", refreshed.Status)
				}
				if refreshed.CompletedAt != nil {
					return fmt.Errorf("expected completed_at to be nil after reset")
				}
				return "ok"
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}
