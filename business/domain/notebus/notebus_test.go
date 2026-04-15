package notebus_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/unitest"
)

func Test_Note(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Note")

	ctx := context.Background()

	contexts, err := contextbus.TestSeedContexts(ctx, 1, db.BusDomain.Context)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	notes, err := notebus.TestSeedNotes(ctx, 2, db.BusDomain.Note, contexts[0].ID)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	unitest.Run(t, query(db.BusDomain, notes), "query")
	unitest.Run(t, create(db.BusDomain, notes, contexts[0].ID), "create")
	unitest.Run(t, update(db.BusDomain, notes), "update")
	unitest.Run(t, delete(db.BusDomain, notes), "delete")
}

func Test_Note_DeleteByRawInputUnconfirmed(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Note_DeleteByRawInputUnconfirmed")
	ctx := context.Background()

	ri, err := rawinputbus.TestSeedRawInputs(ctx, 1, db.BusDomain.RawInput)
	if err != nil {
		t.Fatalf("seeding raw input: %s", err)
	}
	rawInputID := ri[0].ID

	task, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title:      "target task",
		RawInputID: &rawInputID,
	})
	if err != nil {
		t.Fatalf("creating task: %s", err)
	}

	unconfirmedNote, err := db.BusDomain.Note.Create(ctx, notebus.NewNote{
		Content:     "Unconfirmed note",
		Source:      "voice",
		RawInputID:  &rawInputID,
		TaskID:      &task.ID,
		Unconfirmed: true,
	})
	if err != nil {
		t.Fatalf("creating unconfirmed note: %s", err)
	}

	confirmedNote, err := db.BusDomain.Note.Create(ctx, notebus.NewNote{
		Content:     "Confirmed note",
		Source:      "voice",
		RawInputID:  &rawInputID,
		TaskID:      &task.ID,
		Unconfirmed: false,
	})
	if err != nil {
		t.Fatalf("creating confirmed note: %s", err)
	}

	if err := db.BusDomain.Note.DeleteByRawInputUnconfirmed(ctx, rawInputID); err != nil {
		t.Fatalf("DeleteByRawInputUnconfirmed: %s", err)
	}

	_, err = db.BusDomain.Note.QueryByID(ctx, unconfirmedNote.ID)
	if err == nil {
		t.Fatal("expected unconfirmed note to be deleted, but it still exists")
	}

	got, err := db.BusDomain.Note.QueryByID(ctx, confirmedNote.ID)
	if err != nil {
		t.Fatalf("confirmed note should survive: %s", err)
	}
	if got.ID != confirmedNote.ID {
		t.Fatalf("expected confirmed note ID %s, got %s", confirmedNote.ID, got.ID)
	}
}

func query(busDomain dbtest.BusDomain, notes []notebus.Note) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "all",
			ExpResp: notes,
			ExcFunc: func(ctx context.Context) any {
				resp, err := busDomain.Note.Query(ctx, notebus.QueryFilter{}, notebus.DefaultOrderBy, page.New(1, 10))
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.([]notebus.Note)
				if !exists {
					return "error occurred"
				}
				expResp := exp.([]notebus.Note)
				return cmp.Diff(gotResp, expResp,
					cmpopts.EquateApproxTime(time.Second),
					cmpopts.SortSlices(func(a, b notebus.Note) bool {
						return a.ID.String() < b.ID.String()
					}),
				)
			},
		},
		{
			Name:    "byid",
			ExpResp: notes[0],
			ExcFunc: func(ctx context.Context) any {
				resp, err := busDomain.Note.QueryByID(ctx, notes[0].ID)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(notebus.Note)
				if !exists {
					return "error occurred"
				}
				return cmp.Diff(gotResp, exp.(notebus.Note), cmpopts.EquateApproxTime(time.Second))
			},
		},
	}
}

func create(busDomain dbtest.BusDomain, _ []notebus.Note, contextID uuid.UUID) []unitest.Table {
	return []unitest.Table{
		{
			Name: "basic",
			ExpResp: notebus.Note{
				ContextID: &contextID,
				Content:   "New Note Content",
				Source:    "manual",
			},
			ExcFunc: func(ctx context.Context) any {
				nn := notebus.NewNote{
					ContextID: &contextID,
					Content:   "New Note Content",
					Source:    "manual",
				}
				resp, err := busDomain.Note.Create(ctx, nn)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(notebus.Note)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(notebus.Note)
				expResp.ID = gotResp.ID
				expResp.CreatedAt = gotResp.CreatedAt
				expResp.UpdatedAt = gotResp.UpdatedAt
				return cmp.Diff(gotResp, expResp, cmpopts.EquateApproxTime(time.Second))
			},
		},
	}
}

func update(busDomain dbtest.BusDomain, notes []notebus.Note) []unitest.Table {
	newContent := "Updated Content"

	return []unitest.Table{
		{
			Name: "basic",
			ExpResp: notebus.Note{
				ID:         notes[0].ID,
				ContextID:  notes[0].ContextID,
				TaskID:     notes[0].TaskID,
				Content:    "Updated Content",
				Source:     notes[0].Source,
				RawInputID: notes[0].RawInputID,
				CreatedAt:  notes[0].CreatedAt,
			},
			ExcFunc: func(ctx context.Context) any {
				un := notebus.UpdateNote{
					Content: &newContent,
				}
				resp, err := busDomain.Note.Update(ctx, notes[0], un)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(notebus.Note)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(notebus.Note)
				expResp.UpdatedAt = gotResp.UpdatedAt
				return cmp.Diff(gotResp, expResp, cmpopts.EquateApproxTime(time.Second))
			},
		},
	}
}

func delete(busDomain dbtest.BusDomain, notes []notebus.Note) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "basic",
			ExpResp: nil,
			ExcFunc: func(ctx context.Context) any {
				if err := busDomain.Note.Delete(ctx, notes[1]); err != nil {
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
