package dailyplanbus_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/casebrophy/planner/business/domain/dailyplanbus"
	"github.com/casebrophy/planner/business/domain/dailyplanbus/stores/dailyplandb"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/unitest"
	"github.com/casebrophy/planner/business/types/taskenergy"
	"github.com/casebrophy/planner/business/types/taskpriority"
	"github.com/casebrophy/planner/business/types/taskstatus"
)

func Test_DailyPlan(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_DailyPlan")

	store := dailyplandb.NewStore(db.Log, db.DB)
	bus := dailyplanbus.NewBusiness(db.Log, store)

	unitest.Run(t, createPlan(db, bus), "create-plan")
	unitest.Run(t, createAndQueryItems(db, bus), "create-and-query-items")
	unitest.Run(t, updateItem(db, bus), "update-item")
	unitest.Run(t, dismissItem(db, bus), "dismiss-item")
}

func createPlan(db *dbtest.Database, bus *dailyplanbus.Business) []unitest.Table {
	return []unitest.Table{
		{
			Name: "basic-create",
			ExpResp: dailyplanbus.DailyPlan{
				Generation: 1,
				ModelUsed:  "haiku",
			},
			ExcFunc: func(ctx context.Context) any {
				today := time.Now().Truncate(24 * time.Hour)
				plan, err := bus.Create(ctx, dailyplanbus.NewDailyPlan{
					PlanDate:   today,
					Generation: 1,
					ModelUsed:  "haiku",
				})
				if err != nil {
					return err
				}
				return plan
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(dailyplanbus.DailyPlan)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(dailyplanbus.DailyPlan)
				expResp.ID = gotResp.ID
				expResp.PlanDate = gotResp.PlanDate
				expResp.CreatedAt = gotResp.CreatedAt
				return cmp.Diff(gotResp, expResp,
					cmpopts.EquateApproxTime(time.Second),
				)
			},
		},
	}
}

func createAndQueryItems(db *dbtest.Database, bus *dailyplanbus.Business) []unitest.Table {
	return []unitest.Table{
		{
			Name: "basic-add-and-query",
			ExpResp: []dailyplanbus.DailyPlanItem{
				{
					Position:      1,
					GroupName:     "High Priority",
					GroupPosition: 1,
					Status:        "proposed",
					AIDurationMin: intPtr(30),
				},
			},
			ExcFunc: func(ctx context.Context) any {
				// Create a task first
				task, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
					Title:    "Test Task for Plan",
					Status:   taskstatus.Todo,
					Priority: taskpriority.Medium,
					Energy:   taskenergy.Medium,
				})
				if err != nil {
					return err
				}

				// Create a daily plan for today
				today := time.Now().Truncate(24 * time.Hour)
				plan, err := bus.Create(ctx, dailyplanbus.NewDailyPlan{
					PlanDate:   today,
					Generation: 1,
					ModelUsed:  "haiku",
				})
				if err != nil {
					return err
				}

				// Add an item to the plan
				item, err := bus.AddItem(ctx, dailyplanbus.NewDailyPlanItem{
					PlanID:        plan.ID,
					TaskID:        task.ID,
					Position:      1,
					GroupName:     "High Priority",
					GroupPosition: 1,
					AIDurationMin: intPtr(30),
				})
				if err != nil {
					return err
				}

				// Query the item by its ID
				items, err := bus.QueryItemByID(ctx, item.ID)
				if err != nil {
					return err
				}

				return []dailyplanbus.DailyPlanItem{items}
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.([]dailyplanbus.DailyPlanItem)
				if !exists {
					return "error occurred"
				}
				expResp := exp.([]dailyplanbus.DailyPlanItem)
				if len(gotResp) != len(expResp) {
					return "length mismatch"
				}
				// Normalize IDs and timestamps for comparison
				for i := range gotResp {
					expResp[i].ID = gotResp[i].ID
					expResp[i].PlanID = gotResp[i].PlanID
					expResp[i].TaskID = gotResp[i].TaskID
					expResp[i].CreatedAt = gotResp[i].CreatedAt
				}
				return cmp.Diff(gotResp, expResp,
					cmpopts.EquateApproxTime(time.Second),
				)
			},
		},
	}
}

func updateItem(db *dbtest.Database, bus *dailyplanbus.Business) []unitest.Table {
	return []unitest.Table{
		{
			Name: "update-user-position-and-duration",
			ExpResp: dailyplanbus.DailyPlanItem{
				Status:          "proposed",
				UserPosition:    intPtr(5),
				UserDurationMin: intPtr(45),
			},
			ExcFunc: func(ctx context.Context) any {
				// Create prerequisites
				task, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
					Title:    "Update Test Task",
					Status:   taskstatus.Todo,
					Priority: taskpriority.Medium,
					Energy:   taskenergy.Medium,
				})
				if err != nil {
					return err
				}

				today := time.Now().Truncate(24 * time.Hour)
				plan, err := bus.Create(ctx, dailyplanbus.NewDailyPlan{
					PlanDate:   today,
					Generation: 1,
					ModelUsed:  "haiku",
				})
				if err != nil {
					return err
				}

				item, err := bus.AddItem(ctx, dailyplanbus.NewDailyPlanItem{
					PlanID:        plan.ID,
					TaskID:        task.ID,
					Position:      1,
					GroupName:     "Work",
					GroupPosition: 1,
					AIDurationMin: intPtr(30),
				})
				if err != nil {
					return err
				}

				// Update the item
				updated, err := bus.UpdateItem(ctx, item, dailyplanbus.UpdatePlanItem{
					UserPosition:    intPtr(5),
					UserDurationMin: intPtr(45),
				})
				if err != nil {
					return err
				}

				return updated
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(dailyplanbus.DailyPlanItem)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(dailyplanbus.DailyPlanItem)
				expResp.ID = gotResp.ID
				expResp.PlanID = gotResp.PlanID
				expResp.TaskID = gotResp.TaskID
				expResp.Position = gotResp.Position
				expResp.GroupName = gotResp.GroupName
				expResp.GroupPosition = gotResp.GroupPosition
				expResp.AIDurationMin = gotResp.AIDurationMin
				expResp.CreatedAt = gotResp.CreatedAt
				return cmp.Diff(gotResp, expResp,
					cmpopts.EquateApproxTime(time.Second),
				)
			},
		},
	}
}

func dismissItem(db *dbtest.Database, bus *dailyplanbus.Business) []unitest.Table {
	return []unitest.Table{
		{
			Name: "dismiss-with-reason-and-note",
			ExpResp: dailyplanbus.DailyPlanItem{
				Status:        "dismissed",
				DismissReason: strPtr("not_today"),
				DismissNote:   strPtr("busy with other tasks"),
			},
			ExcFunc: func(ctx context.Context) any {
				// Create prerequisites
				task, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
					Title:    "Dismiss Test Task",
					Status:   taskstatus.Todo,
					Priority: taskpriority.Medium,
					Energy:   taskenergy.Medium,
				})
				if err != nil {
					return err
				}

				today := time.Now().Truncate(24 * time.Hour)
				plan, err := bus.Create(ctx, dailyplanbus.NewDailyPlan{
					PlanDate:   today,
					Generation: 1,
					ModelUsed:  "haiku",
				})
				if err != nil {
					return err
				}

				item, err := bus.AddItem(ctx, dailyplanbus.NewDailyPlanItem{
					PlanID:        plan.ID,
					TaskID:        task.ID,
					Position:      1,
					GroupName:     "Backlog",
					GroupPosition: 1,
				})
				if err != nil {
					return err
				}

				// Dismiss the item
				updated, err := bus.UpdateItem(ctx, item, dailyplanbus.UpdatePlanItem{
					Status:        strPtr("dismissed"),
					DismissReason: strPtr("not_today"),
					DismissNote:   strPtr("busy with other tasks"),
				})
				if err != nil {
					return err
				}

				return updated
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(dailyplanbus.DailyPlanItem)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(dailyplanbus.DailyPlanItem)
				expResp.ID = gotResp.ID
				expResp.PlanID = gotResp.PlanID
				expResp.TaskID = gotResp.TaskID
				expResp.Position = gotResp.Position
				expResp.GroupName = gotResp.GroupName
				expResp.GroupPosition = gotResp.GroupPosition
				expResp.CreatedAt = gotResp.CreatedAt
				return cmp.Diff(gotResp, expResp,
					cmpopts.EquateApproxTime(time.Second),
				)
			},
		},
	}
}

func intPtr(i int) *int {
	return &i
}

func strPtr(s string) *string {
	return &s
}
