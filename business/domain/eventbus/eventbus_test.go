package eventbus_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/eventbus/stores/eventdb"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/unitest"
)

func Test_Event(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Event")

	store := eventdb.NewStore(db.Log, db.DB)
	bus := eventbus.NewBusiness(db.Log, store)

	unitest.Run(t, createAndQuery(bus), "create-and-query")
	unitest.Run(t, updateEvent(bus), "update")
	unitest.Run(t, deleteEvent(bus), "delete")
}

func createAndQuery(bus *eventbus.Business) []unitest.Table {
	return []unitest.Table{
		{
			Name: "basic-create",
			ExpResp: eventbus.Event{
				Title:    "Dentist Appointment",
				StartsAt: time.Time{},
				EndsAt:   time.Time{},
				AllDay:   false,
			},
			ExcFunc: func(ctx context.Context) any {
				startsAt := time.Now().Add(24 * time.Hour)
				endsAt := time.Now().Add(25 * time.Hour)

				ne := eventbus.NewEvent{
					Title:    "Dentist Appointment",
					StartsAt: startsAt,
					EndsAt:   endsAt,
					Location: ptrString("123 Main St"),
				}
				resp, err := bus.Create(ctx, ne)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(eventbus.Event)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(eventbus.Event)
				expResp.ID = gotResp.ID
				expResp.CreatedAt = gotResp.CreatedAt
				expResp.UpdatedAt = gotResp.UpdatedAt
				expResp.StartsAt = gotResp.StartsAt
				expResp.EndsAt = gotResp.EndsAt
				expResp.Location = gotResp.Location
				return cmp.Diff(gotResp, expResp,
					cmpopts.EquateApproxTime(time.Second),
				)
			},
		},
		{
			Name: "query-all",
			ExpResp: 1,
			ExcFunc: func(ctx context.Context) any {
				resp, err := bus.Query(ctx, eventbus.QueryFilter{}, eventbus.DefaultOrderBy, page.New(1, 10))
				if err != nil {
					return err
				}
				return len(resp)
			},
			CmpFunc: func(got any, exp any) string {
				gotCount, exists := got.(int)
				if !exists {
					return "error occurred"
				}
				expCount := exp.(int)
				if gotCount < expCount {
					return cmp.Diff(gotCount, expCount)
				}
				return ""
			},
		},
		{
			Name: "query-by-id",
			ExpResp: eventbus.Event{
				Title: "Dentist Appointment",
			},
			ExcFunc: func(ctx context.Context) any {
				startsAt := time.Now().Add(24 * time.Hour)
				endsAt := time.Now().Add(25 * time.Hour)

				ne := eventbus.NewEvent{
					Title:    "Dentist Appointment",
					StartsAt: startsAt,
					EndsAt:   endsAt,
					Location: ptrString("123 Main St"),
				}
				created, err := bus.Create(ctx, ne)
				if err != nil {
					return err
				}

				resp, err := bus.QueryByID(ctx, created.ID)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(eventbus.Event)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(eventbus.Event)
				return cmp.Diff(gotResp.Title, expResp.Title)
			},
		},
	}
}

func updateEvent(bus *eventbus.Business) []unitest.Table {
	return []unitest.Table{
		{
			Name: "basic-update",
			ExpResp: eventbus.Event{
				Title: "Updated Dentist Appointment",
			},
			ExcFunc: func(ctx context.Context) any {
				startsAt := time.Now().Add(24 * time.Hour)
				endsAt := time.Now().Add(25 * time.Hour)

				ne := eventbus.NewEvent{
					Title:    "Dentist Appointment",
					StartsAt: startsAt,
					EndsAt:   endsAt,
					Location: ptrString("123 Main St"),
				}
				created, err := bus.Create(ctx, ne)
				if err != nil {
					return err
				}

				newTitle := "Updated Dentist Appointment"
				ue := eventbus.UpdateEvent{
					Title: &newTitle,
				}
				resp, err := bus.Update(ctx, created, ue)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(eventbus.Event)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(eventbus.Event)
				return cmp.Diff(gotResp.Title, expResp.Title)
			},
		},
	}
}

func deleteEvent(bus *eventbus.Business) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "basic-delete",
			ExpResp: nil,
			ExcFunc: func(ctx context.Context) any {
				startsAt := time.Now().Add(24 * time.Hour)
				endsAt := time.Now().Add(25 * time.Hour)

				ne := eventbus.NewEvent{
					Title:    "Dentist Appointment",
					StartsAt: startsAt,
					EndsAt:   endsAt,
					Location: ptrString("123 Main St"),
				}
				created, err := bus.Create(ctx, ne)
				if err != nil {
					return err
				}

				if err := bus.Delete(ctx, created); err != nil {
					return err
				}

				_, err = bus.QueryByID(ctx, created.ID)
				if err == nil {
					return "expected error for deleted event, got nil"
				}

				return nil
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}

func ptrString(s string) *string {
	return &s
}

func Test_Event_DeleteByRawInputUnconfirmed(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Event_DeleteByRawInputUnconfirmed")
	ctx := context.Background()

	// Create a raw input to associate events with.
	ri, err := rawinputbus.TestSeedRawInputs(ctx, 1, db.BusDomain.RawInput)
	if err != nil {
		t.Fatalf("seeding raw input: %s", err)
	}
	rawInputID := ri[0].ID

	startsAt := time.Now().Add(24 * time.Hour)
	endsAt := time.Now().Add(25 * time.Hour)

	// Create an unconfirmed event linked to the raw input.
	unconfirmedEvent, err := db.BusDomain.Event.Create(ctx, eventbus.NewEvent{
		Title:       "Unconfirmed Event",
		StartsAt:    startsAt,
		EndsAt:      endsAt,
		RawInputID:  &rawInputID,
		Unconfirmed: true,
	})
	if err != nil {
		t.Fatalf("creating unconfirmed event: %s", err)
	}

	// Create a confirmed event linked to the same raw input.
	confirmedEvent, err := db.BusDomain.Event.Create(ctx, eventbus.NewEvent{
		Title:       "Confirmed Event",
		StartsAt:    startsAt,
		EndsAt:      endsAt,
		RawInputID:  &rawInputID,
		Unconfirmed: false,
	})
	if err != nil {
		t.Fatalf("creating confirmed event: %s", err)
	}

	// Call cleanup.
	if err := db.BusDomain.Event.DeleteByRawInputUnconfirmed(ctx, rawInputID); err != nil {
		t.Fatalf("DeleteByRawInputUnconfirmed: %s", err)
	}

	// Unconfirmed event should be gone.
	_, err = db.BusDomain.Event.QueryByID(ctx, unconfirmedEvent.ID)
	if err == nil {
		t.Fatal("expected unconfirmed event to be deleted, but it still exists")
	}

	// Confirmed event should survive.
	got, err := db.BusDomain.Event.QueryByID(ctx, confirmedEvent.ID)
	if err != nil {
		t.Fatalf("confirmed event should survive: %s", err)
	}
	if got.ID != confirmedEvent.ID {
		t.Fatalf("expected confirmed event ID %s, got %s", confirmedEvent.ID, got.ID)
	}
}
