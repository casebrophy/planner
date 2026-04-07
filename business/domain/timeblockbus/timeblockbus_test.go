package timeblockbus_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/timeblockbus"
	"github.com/casebrophy/planner/business/domain/timeblockbus/stores/timeblockdb"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/unitest"
)

func Test_TimeBlock(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_TimeBlock")

	// TimeBlock requires a task (FK).
	tasks, err := taskbus.TestSeedTasks(context.Background(), 1, db.BusDomain.Task)
	if err != nil {
		t.Fatalf("Seeding tasks: %s", err)
	}

	tbBus := timeblockbus.NewBusiness(db.Log, timeblockdb.NewStore(db.Log, db.DB))

	blocks, err := timeblockbus.TestSeedTimeBlocks(context.Background(), 2, tasks[0].ID, tbBus)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	unitest.Run(t, query(tbBus, blocks), "query")
	unitest.Run(t, create(tbBus, tasks[0].ID), "create")
	unitest.Run(t, update(tbBus, blocks), "update")
	unitest.Run(t, delete(tbBus, blocks), "delete")
}

func query(tbBus *timeblockbus.Business, blocks []timeblockbus.TimeBlock) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "all",
			ExpResp: blocks,
			ExcFunc: func(ctx context.Context) any {
				resp, err := tbBus.Query(ctx, timeblockbus.QueryFilter{}, timeblockbus.DefaultOrderBy, page.New(1, 10))
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.([]timeblockbus.TimeBlock)
				if !exists {
					return "error occurred"
				}
				expResp := exp.([]timeblockbus.TimeBlock)
				return cmp.Diff(gotResp, expResp,
					cmpopts.EquateApproxTime(time.Second),
					cmpopts.SortSlices(func(a, b timeblockbus.TimeBlock) bool {
						return a.ID.String() < b.ID.String()
					}),
				)
			},
		},
		{
			Name:    "byid",
			ExpResp: blocks[0],
			ExcFunc: func(ctx context.Context) any {
				resp, err := tbBus.QueryByID(ctx, blocks[0].ID)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(timeblockbus.TimeBlock)
				if !exists {
					return "error occurred"
				}
				return cmp.Diff(gotResp, exp.(timeblockbus.TimeBlock), cmpopts.EquateApproxTime(time.Second))
			},
		},
	}
}

func create(tbBus *timeblockbus.Business, taskID uuid.UUID) []unitest.Table {
	startsAt := time.Now().Add(10 * time.Hour)
	endsAt := startsAt.Add(time.Hour)

	return []unitest.Table{
		{
			Name: "basic",
			ExpResp: timeblockbus.TimeBlock{
				TaskID:    taskID,
				StartsAt:  startsAt,
				EndsAt:    endsAt,
				Confirmed: false,
			},
			ExcFunc: func(ctx context.Context) any {
				ntb := timeblockbus.NewTimeBlock{
					TaskID:   taskID,
					StartsAt: startsAt,
					EndsAt:   endsAt,
				}
				resp, err := tbBus.Create(ctx, ntb)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(timeblockbus.TimeBlock)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(timeblockbus.TimeBlock)
				expResp.ID = gotResp.ID
				expResp.CreatedAt = gotResp.CreatedAt
				expResp.UpdatedAt = gotResp.UpdatedAt
				return cmp.Diff(gotResp, expResp, cmpopts.EquateApproxTime(time.Second))
			},
		},
	}
}

func update(tbBus *timeblockbus.Business, blocks []timeblockbus.TimeBlock) []unitest.Table {
	confirmed := true

	return []unitest.Table{
		{
			Name: "basic",
			ExpResp: timeblockbus.TimeBlock{
				ID:        blocks[0].ID,
				TaskID:    blocks[0].TaskID,
				StartsAt:  blocks[0].StartsAt,
				EndsAt:    blocks[0].EndsAt,
				Confirmed: true,
				CreatedAt: blocks[0].CreatedAt,
			},
			ExcFunc: func(ctx context.Context) any {
				utb := timeblockbus.UpdateTimeBlock{
					Confirmed: &confirmed,
				}
				resp, err := tbBus.Update(ctx, blocks[0], utb)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(timeblockbus.TimeBlock)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(timeblockbus.TimeBlock)
				expResp.UpdatedAt = gotResp.UpdatedAt
				return cmp.Diff(gotResp, expResp, cmpopts.EquateApproxTime(time.Second))
			},
		},
	}
}

func delete(tbBus *timeblockbus.Business, blocks []timeblockbus.TimeBlock) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "basic",
			ExpResp: nil,
			ExcFunc: func(ctx context.Context) any {
				if err := tbBus.Delete(ctx, blocks[1]); err != nil {
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
