package entitylinkbus_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/entitylinkbus"
	"github.com/casebrophy/planner/business/domain/entitylinkbus/stores/entitylinkdb"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/unitest"
)

func Test_EntityLink(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_EntityLink")

	store := entitylinkdb.NewStore(db.Log, db.DB)
	bus := entitylinkbus.NewBusiness(db.Log, store)

	unitest.Run(t, testCreate(bus), "create")
	unitest.Run(t, testQueryBySource(bus), "queryBySource")
	unitest.Run(t, testQueryByTarget(bus), "queryByTarget")
	unitest.Run(t, testDelete(bus), "delete")
}

func testCreate(bus *entitylinkbus.Business) []unitest.Table {
	sourceID := uuid.New()
	targetID := uuid.New()

	return []unitest.Table{
		{
			Name: "basic",
			ExpResp: entitylinkbus.EntityLink{
				SourceType: "task",
				SourceID:   sourceID,
				TargetType: "note",
				TargetID:   targetID,
				Confidence: 1.0,
				Kind:       "manual",
			},
			ExcFunc: func(ctx context.Context) any {
				nl := entitylinkbus.NewEntityLink{
					SourceType: "task",
					SourceID:   sourceID,
					TargetType: "note",
					TargetID:   targetID,
				}
				resp, err := bus.Create(ctx, nl)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, ok := got.(entitylinkbus.EntityLink)
				if !ok {
					return "error occurred"
				}
				expResp := exp.(entitylinkbus.EntityLink)
				expResp.ID = gotResp.ID
				expResp.CreatedAt = gotResp.CreatedAt
				return cmp.Diff(gotResp, expResp, cmpopts.EquateApproxTime(time.Second))
			},
		},
	}
}

func testQueryBySource(bus *entitylinkbus.Business) []unitest.Table {
	sourceID := uuid.New()
	targetID := uuid.New()

	// Pre-create a link so we have something to query.
	link, _ := bus.Create(context.Background(), entitylinkbus.NewEntityLink{
		SourceType: "event",
		SourceID:   sourceID,
		TargetType: "task",
		TargetID:   targetID,
		Kind:       "ai_suggested",
		Confidence: 0.85,
	})

	return []unitest.Table{
		{
			Name:    "found",
			ExpResp: []entitylinkbus.EntityLink{link},
			ExcFunc: func(ctx context.Context) any {
				// QueryByEntity calls QueryBySource + QueryByTarget; use the
				// store directly through the business layer to verify source path.
				resp, err := bus.QueryByEntity(ctx, "event", sourceID)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, ok := got.([]entitylinkbus.EntityLink)
				if !ok {
					return "error occurred"
				}
				expResp := exp.([]entitylinkbus.EntityLink)
				return cmp.Diff(gotResp, expResp, cmpopts.EquateApproxTime(time.Second))
			},
		},
	}
}

func testQueryByTarget(bus *entitylinkbus.Business) []unitest.Table {
	sourceID := uuid.New()
	targetID := uuid.New()

	link, _ := bus.Create(context.Background(), entitylinkbus.NewEntityLink{
		SourceType: "note",
		SourceID:   sourceID,
		TargetType: "task",
		TargetID:   targetID,
	})

	return []unitest.Table{
		{
			Name:    "found",
			ExpResp: []entitylinkbus.EntityLink{link},
			ExcFunc: func(ctx context.Context) any {
				// Query from the target entity's perspective.
				resp, err := bus.QueryByEntity(ctx, "task", targetID)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, ok := got.([]entitylinkbus.EntityLink)
				if !ok {
					return "error occurred"
				}
				expResp := exp.([]entitylinkbus.EntityLink)
				return cmp.Diff(gotResp, expResp, cmpopts.EquateApproxTime(time.Second))
			},
		},
	}
}

func testDelete(bus *entitylinkbus.Business) []unitest.Table {
	sourceID := uuid.New()
	targetID := uuid.New()

	link, _ := bus.Create(context.Background(), entitylinkbus.NewEntityLink{
		SourceType: "task",
		SourceID:   sourceID,
		TargetType: "event",
		TargetID:   targetID,
	})

	return []unitest.Table{
		{
			Name:    "basic",
			ExpResp: nil,
			ExcFunc: func(ctx context.Context) any {
				if err := bus.Delete(ctx, link.ID); err != nil {
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
