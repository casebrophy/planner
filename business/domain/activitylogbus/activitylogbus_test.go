package activitylogbus_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/activitylogbus"
	"github.com/casebrophy/planner/business/domain/activitylogbus/stores/activitylogdb"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/unitest"
)

func Test_ActivityLog(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_ActivityLog")

	alBus := activitylogbus.NewBusiness(db.Log, activitylogdb.NewStore(db.Log, db.DB))

	subjectID := uuid.New()
	logs, err := activitylogbus.TestSeedLogs(context.Background(), 2, "task", subjectID, alBus)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	unitest.Run(t, query(alBus, logs), "query")
	unitest.Run(t, queryBySubjects(alBus, logs, subjectID), "queryBySubjects")
	unitest.Run(t, create(alBus), "create")
	unitest.Run(t, count(alBus), "count")
	unitest.Run(t, streaks(alBus, subjectID), "streaks")
}

func query(alBus *activitylogbus.Business, logs []activitylogbus.Log) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "all",
			ExpResp: logs,
			ExcFunc: func(ctx context.Context) any {
				resp, err := alBus.Query(ctx, activitylogbus.QueryFilter{}, activitylogbus.DefaultOrderBy, page.New(1, 10))
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.([]activitylogbus.Log)
				if !exists {
					return "error occurred"
				}
				expResp := exp.([]activitylogbus.Log)
				return cmp.Diff(gotResp, expResp,
					cmpopts.EquateApproxTime(time.Second),
					cmpopts.SortSlices(func(a, b activitylogbus.Log) bool {
						return a.ID.String() < b.ID.String()
					}),
				)
			},
		},
	}
}

func queryBySubjects(alBus *activitylogbus.Business, logs []activitylogbus.Log, subjectID uuid.UUID) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "matching-subject",
			ExpResp: logs,
			ExcFunc: func(ctx context.Context) any {
				filter := activitylogbus.QueryBySubjectsFilter{
					SubjectType: "task",
					SubjectIDs:  []uuid.UUID{subjectID},
					From:        time.Now().Add(-24 * time.Hour),
					To:          time.Now().Add(24 * time.Hour),
				}
				resp, err := alBus.QueryBySubjects(ctx, filter)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.([]activitylogbus.Log)
				if !exists {
					return "error occurred"
				}
				expResp := exp.([]activitylogbus.Log)
				return cmp.Diff(gotResp, expResp,
					cmpopts.EquateApproxTime(time.Second),
					cmpopts.SortSlices(func(a, b activitylogbus.Log) bool {
						return a.ID.String() < b.ID.String()
					}),
				)
			},
		},
		{
			Name:    "no-matching-subject",
			ExpResp: []activitylogbus.Log{},
			ExcFunc: func(ctx context.Context) any {
				filter := activitylogbus.QueryBySubjectsFilter{
					SubjectType: "task",
					SubjectIDs:  []uuid.UUID{uuid.New()},
					From:        time.Now().Add(-24 * time.Hour),
					To:          time.Now().Add(24 * time.Hour),
				}
				resp, err := alBus.QueryBySubjects(ctx, filter)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.([]activitylogbus.Log)
				if !exists {
					return "error occurred"
				}
				expResp := exp.([]activitylogbus.Log)
				if len(gotResp) == 0 && len(expResp) == 0 {
					return ""
				}
				return cmp.Diff(gotResp, expResp)
			},
		},
	}
}

func create(alBus *activitylogbus.Business) []unitest.Table {
	val := "new-activity"
	return []unitest.Table{
		{
			Name: "basic",
			ExpResp: activitylogbus.Log{
				SubjectType: "note",
				SubjectID:   uuid.UUID{}, // will be set in CmpFunc
				Value:       &val,
			},
			ExcFunc: func(ctx context.Context) any {
				nl := activitylogbus.NewLog{
					SubjectType: "note",
					SubjectID:   uuid.New(),
					Value:       &val,
				}
				resp, err := alBus.Create(ctx, nl)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(activitylogbus.Log)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(activitylogbus.Log)
				expResp.ID = gotResp.ID
				expResp.SubjectID = gotResp.SubjectID
				expResp.LoggedAt = gotResp.LoggedAt
				return cmp.Diff(gotResp, expResp, cmpopts.EquateApproxTime(time.Second))
			},
		},
	}
}

func count(alBus *activitylogbus.Business) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "all",
			ExpResp: 2,
			ExcFunc: func(ctx context.Context) any {
				n, err := alBus.Count(ctx, activitylogbus.QueryFilter{})
				if err != nil {
					return err
				}
				return n
			},
			CmpFunc: func(got any, exp any) string {
				gotN, ok := got.(int)
				if !ok {
					return "error occurred"
				}
				if gotN < exp.(int) {
					return fmt.Sprintf("expected at least %d, got %d", exp.(int), gotN)
				}
				return ""
			},
		},
	}
}

func streaks(alBus *activitylogbus.Business, subjectID uuid.UUID) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "basic",
			ExpResp: nil,
			ExcFunc: func(ctx context.Context) any {
				info, err := alBus.QueryStreaks(ctx, "task", subjectID)
				if err != nil {
					return err
				}
				return info
			},
			CmpFunc: func(got any, exp any) string {
				info, ok := got.(activitylogbus.StreakInfo)
				if !ok {
					return "error occurred"
				}
				// We seeded 2 logs on the same day, so we expect at least 1 day streak
				if info.TotalCount < 2 {
					return fmt.Sprintf("expected TotalCount >= 2, got %d", info.TotalCount)
				}
				if info.Current < 1 {
					return fmt.Sprintf("expected Current >= 1, got %d", info.Current)
				}
				return ""
			},
		},
	}
}
