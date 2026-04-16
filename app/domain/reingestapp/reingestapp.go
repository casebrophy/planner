package reingestapp

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	log      *logger.Logger
	taskBus  *taskbus.Business
	noteBus  *notebus.Business
	eventBus *eventbus.Business
	riBus    *rawinputbus.Business
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
		return errs.Newf(errs.InvalidArgument, "task has no raw_input_id; cannot reingest")
	}

	skipClassify := task.ContextID != nil

	if !skipClassify {
		if err := a.taskBus.DeleteByRawInputUnconfirmed(ctx, *task.RawInputID); err != nil {
			return errs.Newf(errs.Internal, "delete unconfirmed: %s", err)
		}
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
		return errs.Newf(errs.InvalidArgument, "note has no raw_input_id; cannot reingest")
	}

	skipClassify := note.ContextID != nil || note.TaskID != nil

	if !skipClassify {
		if err := a.noteBus.DeleteByRawInputUnconfirmed(ctx, *note.RawInputID); err != nil {
			return errs.Newf(errs.Internal, "delete unconfirmed: %s", err)
		}
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
		return errs.Newf(errs.InvalidArgument, "event has no raw_input_id; cannot reingest")
	}

	skipClassify := event.ContextID != nil

	if !skipClassify {
		if err := a.eventBus.DeleteByRawInputUnconfirmed(ctx, *event.RawInputID); err != nil {
			return errs.Newf(errs.Internal, "delete unconfirmed: %s", err)
		}
	}

	if err := a.resetRawInput(ctx, *event.RawInputID, skipClassify); err != nil {
		return errs.Newf(errs.Internal, "reset raw input: %s", err)
	}

	return ReingestResponse{RawInputID: event.RawInputID.String(), SkipClassify: skipClassify, Enqueued: true}
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
