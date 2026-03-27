package threadbus_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/unitest"
	"github.com/casebrophy/planner/business/types/threadentrykind"
	"github.com/casebrophy/planner/business/types/threadsource"
)

func Test_Thread(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Thread")

	subjectID := uuid.New()

	entries, err := threadbus.TestSeedThreadEntries(context.Background(), "task", subjectID, 2, db.BusDomain.Thread)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	unitest.Run(t, create(db.BusDomain, subjectID), "create")
	unitest.Run(t, queryBySubject(db.BusDomain, entries, subjectID), "queryBySubject")
}

func create(busDomain dbtest.BusDomain, subjectID uuid.UUID) []unitest.Table {
	return []unitest.Table{
		{
			Name: "basic",
			ExpResp: threadbus.ThreadEntry{
				SubjectType: "task",
				SubjectID:   subjectID,
				Kind:        threadentrykind.Note,
				Content:     "new thread entry content",
				Source:      threadsource.User,
			},
			ExcFunc: func(ctx context.Context) any {
				ne := threadbus.NewThreadEntry{
					SubjectType: "task",
					SubjectID:   subjectID,
					Kind:        threadentrykind.Note,
					Content:     "new thread entry content",
					Source:      threadsource.User,
				}
				resp, err := busDomain.Thread.AddEntry(ctx, ne)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(threadbus.ThreadEntry)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(threadbus.ThreadEntry)
				expResp.ID = gotResp.ID
				expResp.CreatedAt = gotResp.CreatedAt
				return cmp.Diff(gotResp, expResp)
			},
		},
	}
}

func queryBySubject(busDomain dbtest.BusDomain, entries []threadbus.ThreadEntry, subjectID uuid.UUID) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "all",
			ExpResp: entries,
			ExcFunc: func(ctx context.Context) any {
				resp, err := busDomain.Thread.QueryBySubject(ctx, "task", subjectID, page.MustParse("1", "10"))
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.([]threadbus.ThreadEntry)
				if !exists {
					return "error occurred"
				}
				expResp := exp.([]threadbus.ThreadEntry)
				return cmp.Diff(gotResp, expResp,
					cmpopts.EquateApproxTime(time.Second),
					cmpopts.SortSlices(func(a, b threadbus.ThreadEntry) bool {
						return a.ID.String() < b.ID.String()
					}),
				)
			},
		},
	}
}
