package reingestapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/business/types/clarificationstatus"
	"github.com/casebrophy/planner/business/types/rawinputsource"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	log      *logger.Logger
	taskBus  *taskbus.Business
	noteBus  *notebus.Business
	eventBus *eventbus.Business
	riBus    *rawinputbus.Business
	clarBus  *clarificationbus.Business
}

func (a *app) reingestTask(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "task_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	task, err := a.taskBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query task: %s", err)
	}

	if task.RawInputID == nil {
		rawInputID, err := a.synthesizeRawInputForTask(ctx, task)
		if err != nil {
			return errs.Newf(errs.Internal, "synthesize raw_input: %s", err)
		}
		task.RawInputID = &rawInputID
	}

	skipClassify := task.ContextID != nil

	if !skipClassify {
		if err := a.taskBus.DeleteByRawInputUnconfirmed(ctx, *task.RawInputID); err != nil {
			return errs.Newf(errs.Internal, "delete unconfirmed: %s", err)
		}
	}

	if err := a.dismissStaleClarifications(ctx, *task.RawInputID, "task", task.ID); err != nil {
		return errs.Newf(errs.Internal, "dismiss stale clarifications: %s", err)
	}

	if err := a.resetRawInput(ctx, *task.RawInputID, skipClassify); err != nil {
		return errs.Newf(errs.Internal, "reset raw input: %s", err)
	}

	return ReingestResponse{RawInputID: task.RawInputID.String(), SkipClassify: skipClassify, Enqueued: true}
}

func (a *app) reingestNote(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "note_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	note, err := a.noteBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query note: %s", err)
	}

	if note.RawInputID == nil {
		rawInputID, err := a.synthesizeRawInputForNote(ctx, note)
		if err != nil {
			return errs.Newf(errs.Internal, "synthesize raw_input: %s", err)
		}
		note.RawInputID = &rawInputID
	}

	skipClassify := note.ContextID != nil || note.TaskID != nil

	if !skipClassify {
		if err := a.noteBus.DeleteByRawInputUnconfirmed(ctx, *note.RawInputID); err != nil {
			return errs.Newf(errs.Internal, "delete unconfirmed: %s", err)
		}
	}

	if err := a.dismissStaleClarifications(ctx, *note.RawInputID, "note", note.ID); err != nil {
		return errs.Newf(errs.Internal, "dismiss stale clarifications: %s", err)
	}

	if err := a.resetRawInput(ctx, *note.RawInputID, skipClassify); err != nil {
		return errs.Newf(errs.Internal, "reset raw input: %s", err)
	}

	return ReingestResponse{RawInputID: note.RawInputID.String(), SkipClassify: skipClassify, Enqueued: true}
}

func (a *app) reingestEvent(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "event_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	event, err := a.eventBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query event: %s", err)
	}

	if event.RawInputID == nil {
		rawInputID, err := a.synthesizeRawInputForEvent(ctx, event)
		if err != nil {
			return errs.Newf(errs.Internal, "synthesize raw_input: %s", err)
		}
		event.RawInputID = &rawInputID
	}

	skipClassify := event.ContextID != nil

	if !skipClassify {
		if err := a.eventBus.DeleteByRawInputUnconfirmed(ctx, *event.RawInputID); err != nil {
			return errs.Newf(errs.Internal, "delete unconfirmed: %s", err)
		}
	}

	if err := a.dismissStaleClarifications(ctx, *event.RawInputID, "event", event.ID); err != nil {
		return errs.Newf(errs.Internal, "dismiss stale clarifications: %s", err)
	}

	if err := a.resetRawInput(ctx, *event.RawInputID, skipClassify); err != nil {
		return errs.Newf(errs.Internal, "reset raw input: %s", err)
	}

	return ReingestResponse{RawInputID: event.RawInputID.String(), SkipClassify: skipClassify, Enqueued: true}
}

// dismissStaleClarifications finds and dismisses clarifications tied to a raw_input
// or entity that are still pending/snoozed. Clarifications become stale when reingest
// reprocesses an entity, so they should be dismissed to avoid duplication.
func (a *app) dismissStaleClarifications(ctx context.Context, rawInputID uuid.UUID, entityType string, entityID uuid.UUID) error {
	pg := page.New(1, 1000)

	// Dismiss clarifications tied to the raw_input
	rawInputType := "raw_input"
	pending := clarificationstatus.Pending
	snoozed := clarificationstatus.Snoozed

	for _, status := range []*clarificationstatus.Status{&pending, &snoozed} {
		rawInputClarifications, err := a.clarBus.Query(ctx, clarificationbus.QueryFilter{
			Status:      status,
			SubjectType: &rawInputType,
			SubjectID:   &rawInputID,
		}, clarificationbus.DefaultOrderBy, pg)
		if err != nil {
			return fmt.Errorf("query raw_input clarifications: %w", err)
		}
		for _, clar := range rawInputClarifications {
			if _, err := a.clarBus.Dismiss(ctx, clar); err != nil {
				a.log.Warn(ctx, "failed to dismiss raw_input clarification", "clarification_id", clar.ID, "error", err)
			}
		}
	}

	// Dismiss clarifications tied to the entity
	for _, status := range []*clarificationstatus.Status{&pending, &snoozed} {
		entityClarifications, err := a.clarBus.Query(ctx, clarificationbus.QueryFilter{
			Status:      status,
			SubjectType: &entityType,
			SubjectID:   &entityID,
		}, clarificationbus.DefaultOrderBy, pg)
		if err != nil {
			return fmt.Errorf("query entity clarifications: %w", err)
		}
		for _, clar := range entityClarifications {
			if _, err := a.clarBus.Dismiss(ctx, clar); err != nil {
				a.log.Warn(ctx, "failed to dismiss entity clarification", "clarification_id", clar.ID, "error", err)
			}
		}
	}

	return nil
}

// resetRawInput resets a raw_input for reingest. If skipClassify is true, uses
// ResetForReingest (sets skip_classify=TRUE, reingest_mode=TRUE). Otherwise uses
// ResetForReprocess and explicitly sets reingest_mode=TRUE to preserve confirmed state.
func (a *app) resetRawInput(ctx context.Context, rawInputID uuid.UUID, skipClassify bool) error {
	if skipClassify {
		_, err := a.riBus.ResetForReingest(ctx, rawInputID)
		return err
	}

	ri, err := a.riBus.ResetForReprocess(ctx, rawInputID)
	if err != nil {
		return err
	}
	trueVal := true
	_, err = a.riBus.Update(ctx, ri, rawinputbus.UpdateRawInput{ReingestMode: &trueVal})
	return err
}

// synthesizeRawInputForTask creates a raw_input from task content and updates the task.
func (a *app) synthesizeRawInputForTask(ctx context.Context, task taskbus.Task) (uuid.UUID, error) {
	rawContent := buildTaskContent(task)
	ri, err := a.riBus.Create(ctx, rawinputbus.NewRawInput{
		SourceType:       rawinputsource.Manual,
		RawContent:       rawContent,
		SourceEntityID:   &task.ID,
		SourceEntityKind: "task",
	})
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("create raw_input: %w", err)
	}

	_, err = a.taskBus.Update(ctx, task, taskbus.UpdateTask{RawInputID: &ri.ID})
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("update task with raw_input_id: %w", err)
	}

	return ri.ID, nil
}

// synthesizeRawInputForNote creates a raw_input from note content and updates the note.
func (a *app) synthesizeRawInputForNote(ctx context.Context, note notebus.Note) (uuid.UUID, error) {
	ri, err := a.riBus.Create(ctx, rawinputbus.NewRawInput{
		SourceType:       rawinputsource.Manual,
		RawContent:       note.Content,
		SourceEntityID:   &note.ID,
		SourceEntityKind: "note",
	})
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("create raw_input: %w", err)
	}

	_, err = a.noteBus.Update(ctx, note, notebus.UpdateNote{RawInputID: &ri.ID})
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("update note with raw_input_id: %w", err)
	}

	return ri.ID, nil
}

// synthesizeRawInputForEvent creates a raw_input from event content and updates the event.
func (a *app) synthesizeRawInputForEvent(ctx context.Context, event eventbus.Event) (uuid.UUID, error) {
	rawContent := buildEventContent(event)
	ri, err := a.riBus.Create(ctx, rawinputbus.NewRawInput{
		SourceType:       rawinputsource.Manual,
		RawContent:       rawContent,
		SourceEntityID:   &event.ID,
		SourceEntityKind: "event",
	})
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("create raw_input: %w", err)
	}

	_, err = a.eventBus.Update(ctx, event, eventbus.UpdateEvent{RawInputID: &ri.ID})
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("update event with raw_input_id: %w", err)
	}

	return ri.ID, nil
}

// buildTaskContent combines task title and description into raw content.
func buildTaskContent(task taskbus.Task) string {
	var parts []string
	if task.Title != "" {
		parts = append(parts, task.Title)
	}
	if task.Description != "" {
		parts = append(parts, task.Description)
	}
	return strings.Join(parts, "\n\n")
}

// buildEventContent combines event title and description into raw content.
func buildEventContent(event eventbus.Event) string {
	var parts []string
	if event.Title != "" {
		parts = append(parts, event.Title)
	}
	if event.Description != "" {
		parts = append(parts, event.Description)
	}
	return strings.Join(parts, "\n\n")
}

func (a *app) reingestBulk(ctx context.Context, r *http.Request) web.Encoder {
	var req BulkReingestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if req.EntityType == "" {
		return errs.New(errs.InvalidArgument, errors.New("entityType is required"))
	}

	if req.ContextID != "" {
		if _, err := uuid.Parse(req.ContextID); err != nil {
			return errs.New(errs.InvalidArgument, errors.New("invalid contextId format"))
		}
	}

	queued := 0

	switch req.EntityType {
	case "task":
		tasks, err := a.queryTasksForBulkReingest(ctx, req.ContextID)
		if err != nil {
			return errs.Newf(errs.Internal, "query tasks: %s", err)
		}
		for _, task := range tasks {
			if task.RawInputID == nil {
				rawInputID, err := a.synthesizeRawInputForTask(ctx, task)
				if err != nil {
					a.log.Warn(ctx, "failed to synthesize raw_input for task", "error", err)
					continue
				}
				task.RawInputID = &rawInputID
			}
			skipClassify := task.ContextID != nil
			if !skipClassify {
				if err := a.taskBus.DeleteByRawInputUnconfirmed(ctx, *task.RawInputID); err != nil {
					a.log.Warn(ctx, "failed to delete unconfirmed task", "error", err)
					continue
				}
			}
			if err := a.dismissStaleClarifications(ctx, *task.RawInputID, "task", task.ID); err != nil {
				a.log.Warn(ctx, "failed to dismiss stale clarifications for task", "error", err)
			}
			if err := a.resetRawInput(ctx, *task.RawInputID, skipClassify); err != nil {
				a.log.Warn(ctx, "failed to reset task raw_input", "error", err)
				continue
			}
			queued++
		}

	case "note":
		notes, err := a.queryNotesForBulkReingest(ctx, req.ContextID)
		if err != nil {
			return errs.Newf(errs.Internal, "query notes: %s", err)
		}
		for _, note := range notes {
			if note.RawInputID == nil {
				rawInputID, err := a.synthesizeRawInputForNote(ctx, note)
				if err != nil {
					a.log.Warn(ctx, "failed to synthesize raw_input for note", "error", err)
					continue
				}
				note.RawInputID = &rawInputID
			}
			skipClassify := note.ContextID != nil || note.TaskID != nil
			if !skipClassify {
				if err := a.noteBus.DeleteByRawInputUnconfirmed(ctx, *note.RawInputID); err != nil {
					a.log.Warn(ctx, "failed to delete unconfirmed note", "error", err)
					continue
				}
			}
			if err := a.dismissStaleClarifications(ctx, *note.RawInputID, "note", note.ID); err != nil {
				a.log.Warn(ctx, "failed to dismiss stale clarifications for note", "error", err)
			}
			if err := a.resetRawInput(ctx, *note.RawInputID, skipClassify); err != nil {
				a.log.Warn(ctx, "failed to reset note raw_input", "error", err)
				continue
			}
			queued++
		}

	case "event":
		events, err := a.queryEventsForBulkReingest(ctx, req.ContextID)
		if err != nil {
			return errs.Newf(errs.Internal, "query events: %s", err)
		}
		for _, event := range events {
			if event.RawInputID == nil {
				rawInputID, err := a.synthesizeRawInputForEvent(ctx, event)
				if err != nil {
					a.log.Warn(ctx, "failed to synthesize raw_input for event", "error", err)
					continue
				}
				event.RawInputID = &rawInputID
			}
			skipClassify := event.ContextID != nil
			if !skipClassify {
				if err := a.eventBus.DeleteByRawInputUnconfirmed(ctx, *event.RawInputID); err != nil {
					a.log.Warn(ctx, "failed to delete unconfirmed event", "error", err)
					continue
				}
			}
			if err := a.dismissStaleClarifications(ctx, *event.RawInputID, "event", event.ID); err != nil {
				a.log.Warn(ctx, "failed to dismiss stale clarifications for event", "error", err)
			}
			if err := a.resetRawInput(ctx, *event.RawInputID, skipClassify); err != nil {
				a.log.Warn(ctx, "failed to reset event raw_input", "error", err)
				continue
			}
			queued++
		}

	default:
		return errs.Newf(errs.InvalidArgument, "invalid entityType: %s (must be task, note, or event)", req.EntityType)
	}

	return BulkReingestResponse{Queued: queued}
}

func (a *app) queryTasksForBulkReingest(ctx context.Context, contextIDStr string) ([]taskbus.Task, error) {
	filter := taskbus.QueryFilter{}
	if contextIDStr != "" {
		contextID, err := uuid.Parse(contextIDStr)
		if err != nil {
			return nil, errors.New("invalid contextId format")
		}
		filter.ContextID = &contextID
	}
	return a.taskBus.Query(ctx, filter, taskbus.DefaultOrderBy, page.New(1, 10000))
}

func (a *app) queryNotesForBulkReingest(ctx context.Context, contextIDStr string) ([]notebus.Note, error) {
	filter := notebus.QueryFilter{}
	if contextIDStr != "" {
		contextID, err := uuid.Parse(contextIDStr)
		if err != nil {
			return nil, errors.New("invalid contextId format")
		}
		filter.ContextID = &contextID
	}
	return a.noteBus.Query(ctx, filter, notebus.DefaultOrderBy, page.New(1, 10000))
}

func (a *app) queryEventsForBulkReingest(ctx context.Context, contextIDStr string) ([]eventbus.Event, error) {
	filter := eventbus.QueryFilter{}
	if contextIDStr != "" {
		contextID, err := uuid.Parse(contextIDStr)
		if err != nil {
			return nil, errors.New("invalid contextId format")
		}
		filter.ContextID = &contextID
	}
	return a.eventBus.Query(ctx, filter, eventbus.DefaultOrderBy, page.New(1, 10000))
}
