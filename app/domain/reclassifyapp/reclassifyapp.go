package reclassifyapp

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/domain/noteapp"
	"github.com/casebrophy/planner/app/domain/taskapp"
	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/business/domain/reclassifybus"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	log           *logger.Logger
	reclassifyBus *reclassifybus.Bus
}

func (a *app) convertTaskToNote(ctx context.Context, r *http.Request) web.Encoder {
	taskIDStr := r.PathValue("task_id")
	if taskIDStr == "" {
		return errs.Newf(errs.InvalidArgument, "task_id required")
	}

	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	note, err := a.reclassifyBus.TaskToNote(ctx, taskID)
	if err != nil {
		return a.mapError(err)
	}

	return noteapp.ToAppNote(note)
}

func (a *app) convertNoteToTask(ctx context.Context, r *http.Request) web.Encoder {
	noteIDStr := r.PathValue("note_id")
	if noteIDStr == "" {
		return errs.Newf(errs.InvalidArgument, "note_id required")
	}

	noteID, err := uuid.Parse(noteIDStr)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	task, err := a.reclassifyBus.NoteToTask(ctx, noteID)
	if err != nil {
		return a.mapError(err)
	}

	return taskapp.ToAppTask(task)
}

// mapError converts reclassifybus errors to appropriate HTTP status codes.
func (a *app) mapError(err error) *errs.Error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()

	// Not found errors
	if strings.Contains(errMsg, "fetch task") || strings.Contains(errMsg, "fetch note") {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		// Wrapped "no rows" error
		if strings.Contains(errMsg, "no rows") {
			return errs.New(errs.NotFound, err)
		}
	}

	// Invalid argument errors (preflight checks)
	if strings.Contains(errMsg, "cannot convert recurring") ||
		strings.Contains(errMsg, "cannot convert") && strings.Contains(errMsg, "dependents") ||
		strings.Contains(errMsg, "cannot convert") && strings.Contains(errMsg, "recurrence children") {
		return errs.New(errs.InvalidArgument, err)
	}

	// Conflict: task scheduled in today's plan
	if strings.Contains(errMsg, "cannot convert task scheduled in today's plan") {
		return errs.New(errs.AlreadyExists, err)
	}

	// Default to internal error
	return errs.New(errs.Internal, err)
}
