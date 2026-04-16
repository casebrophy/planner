package reingestapi_test

import (
	"context"
	"fmt"
	"time"

	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/types/rawinputsource"
	"github.com/casebrophy/planner/business/types/rawinputstatus"
)

type seedData struct {
	ctx           contextbus.Context
	linkedTask    taskbus.Task
	unlinkedTask  taskbus.Task
	noRITask      taskbus.Task
	linkedNote    notebus.Note
	linkedEvent   eventbus.Event
	unlinkedEvent eventbus.Event
}

func insertSeedData(db *dbtest.Database) (seedData, error) {
	ctx := context.Background()

	// Create a context for linking entities.
	ctxItem, err := db.BusDomain.Context.Create(ctx, contextbus.NewContext{
		Title: "Test Context",
	})
	if err != nil {
		return seedData{}, fmt.Errorf("seeding context: %w", err)
	}

	// --- Tasks ---

	// linkedTask: has ContextID + RawInputID
	processed := rawinputstatus.Processed
	riLinkedTask, err := db.BusDomain.RawInput.Create(ctx, rawinputbus.NewRawInput{
		SourceType:       rawinputsource.Manual,
		RawContent:       "linked task raw input",
		SourceEntityKind: "task",
		Status:           &processed,
	})
	if err != nil {
		return seedData{}, fmt.Errorf("seeding ri for linked task: %w", err)
	}
	linkedTask, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title:      "Linked Task",
		ContextID:  &ctxItem.ID,
		RawInputID: &riLinkedTask.ID,
	})
	if err != nil {
		return seedData{}, fmt.Errorf("seeding linked task: %w", err)
	}
	_ = db.BusDomain.RawInput.UpdateSourceEntity(ctx, riLinkedTask.ID, linkedTask.ID, "task")

	// unlinkedTask: no ContextID, has RawInputID
	riUnlinkedTask, err := db.BusDomain.RawInput.Create(ctx, rawinputbus.NewRawInput{
		SourceType:       rawinputsource.Manual,
		RawContent:       "unlinked task raw input",
		SourceEntityKind: "task",
		Status:           &processed,
	})
	if err != nil {
		return seedData{}, fmt.Errorf("seeding ri for unlinked task: %w", err)
	}
	unlinkedTask, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title:      "Unlinked Task",
		RawInputID: &riUnlinkedTask.ID,
	})
	if err != nil {
		return seedData{}, fmt.Errorf("seeding unlinked task: %w", err)
	}
	_ = db.BusDomain.RawInput.UpdateSourceEntity(ctx, riUnlinkedTask.ID, unlinkedTask.ID, "task")

	// noRITask: no RawInputID at all
	noRITask, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title: "No RI Task",
	})
	if err != nil {
		return seedData{}, fmt.Errorf("seeding no-ri task: %w", err)
	}

	// --- Notes ---

	// linkedNote: notes always have context_id or task_id (DB constraint), so always skipClassify=true
	riLinkedNote, err := db.BusDomain.RawInput.Create(ctx, rawinputbus.NewRawInput{
		SourceType:       rawinputsource.Manual,
		RawContent:       "linked note raw input",
		SourceEntityKind: "note",
		Status:           &processed,
	})
	if err != nil {
		return seedData{}, fmt.Errorf("seeding ri for linked note: %w", err)
	}
	linkedNote, err := db.BusDomain.Note.Create(ctx, notebus.NewNote{
		ContextID:  &ctxItem.ID,
		Content:    "Linked Note Content",
		Source:     "manual",
		RawInputID: &riLinkedNote.ID,
	})
	if err != nil {
		return seedData{}, fmt.Errorf("seeding linked note: %w", err)
	}
	_ = db.BusDomain.RawInput.UpdateSourceEntity(ctx, riLinkedNote.ID, linkedNote.ID, "note")

	// --- Events ---

	now := time.Now()

	// linkedEvent: has ContextID + RawInputID
	riLinkedEvent, err := db.BusDomain.RawInput.Create(ctx, rawinputbus.NewRawInput{
		SourceType:       rawinputsource.Manual,
		RawContent:       "linked event raw input",
		SourceEntityKind: "event",
		Status:           &processed,
	})
	if err != nil {
		return seedData{}, fmt.Errorf("seeding ri for linked event: %w", err)
	}
	linkedEvent, err := db.BusDomain.Event.Create(ctx, eventbus.NewEvent{
		Title:      "Linked Event",
		ContextID:  &ctxItem.ID,
		StartsAt:   now,
		EndsAt:     now.Add(time.Hour),
		RawInputID: &riLinkedEvent.ID,
	})
	if err != nil {
		return seedData{}, fmt.Errorf("seeding linked event: %w", err)
	}
	_ = db.BusDomain.RawInput.UpdateSourceEntity(ctx, riLinkedEvent.ID, linkedEvent.ID, "event")

	// unlinkedEvent: no ContextID, has RawInputID
	riUnlinkedEvent, err := db.BusDomain.RawInput.Create(ctx, rawinputbus.NewRawInput{
		SourceType:       rawinputsource.Manual,
		RawContent:       "unlinked event raw input",
		SourceEntityKind: "event",
		Status:           &processed,
	})
	if err != nil {
		return seedData{}, fmt.Errorf("seeding ri for unlinked event: %w", err)
	}
	unlinkedEvent, err := db.BusDomain.Event.Create(ctx, eventbus.NewEvent{
		Title:      "Unlinked Event",
		StartsAt:   now,
		EndsAt:     now.Add(time.Hour),
		RawInputID: &riUnlinkedEvent.ID,
	})
	if err != nil {
		return seedData{}, fmt.Errorf("seeding unlinked event: %w", err)
	}
	_ = db.BusDomain.RawInput.UpdateSourceEntity(ctx, riUnlinkedEvent.ID, unlinkedEvent.ID, "event")

	return seedData{
		ctx:           ctxItem,
		linkedTask:    linkedTask,
		unlinkedTask:  unlinkedTask,
		noRITask:      noRITask,
		linkedNote:    linkedNote,
		linkedEvent:   linkedEvent,
		unlinkedEvent: unlinkedEvent,
	}, nil
}
