package correctionapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/business/domain/classificationcorrectionbus"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/business/types/taskenergy"
	"github.com/casebrophy/planner/business/types/taskpriority"
	"github.com/casebrophy/planner/business/types/taskstatus"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	taskBus       *taskbus.Business
	noteBus       *notebus.Business
	eventBus      *eventbus.Business
	correctionBus *classificationcorrectionbus.Business
}

func (a *app) correct(ctx context.Context, r *http.Request) web.Encoder {
	var body CorrectionRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if body.ItemType == body.NewType {
		return errs.New(errs.InvalidArgument, fmt.Errorf("item_type and new_type must differ"))
	}

	validTypes := map[string]bool{"task": true, "note": true, "event": true}
	if !validTypes[body.ItemType] {
		return errs.New(errs.InvalidArgument, fmt.Errorf("invalid item_type %q", body.ItemType))
	}
	if !validTypes[body.NewType] {
		return errs.New(errs.InvalidArgument, fmt.Errorf("invalid new_type %q", body.NewType))
	}

	itemID, err := uuid.Parse(body.ItemID)
	if err != nil {
		return errs.New(errs.InvalidArgument, fmt.Errorf("invalid item_id: %w", err))
	}

	now := time.Now()
	var newID uuid.UUID
	var clauseText string

	switch body.ItemType {
	case "task":
		task, err := a.taskBus.QueryByID(ctx, itemID)
		if err != nil {
			if errors.Is(err, sqldb.ErrDBNotFound) {
				return errs.New(errs.NotFound, err)
			}
			return errs.New(errs.Internal, err)
		}
		clauseText = task.Title

		switch body.NewType {
		case "note":
			content := task.Title
			if strings.TrimSpace(task.Description) != "" {
				content = task.Title + ": " + task.Description
			}
			note, err := a.noteBus.Create(ctx, notebus.NewNote{
				Content:     content,
				Source:      "correction",
				ContextID:   task.ContextID,
				RawInputID:  nil,
				Unconfirmed: false,
			})
			if err != nil {
				return errs.New(errs.Internal, err)
			}
			newID = note.ID

		case "event":
			event, err := a.eventBus.Create(ctx, eventbus.NewEvent{
				Title:       task.Title,
				Description: task.Description,
				ContextID:   task.ContextID,
				StartsAt:    now.Add(1 * time.Hour),
				EndsAt:      now.Add(2 * time.Hour),
				AllDay:      false,
				RawInputID:  nil,
				Unconfirmed: false,
			})
			if err != nil {
				return errs.New(errs.Internal, err)
			}
			newID = event.ID
		}

		if err := a.taskBus.Delete(ctx, task); err != nil {
			return errs.New(errs.Internal, err)
		}

	case "note":
		note, err := a.noteBus.QueryByID(ctx, itemID)
		if err != nil {
			if errors.Is(err, sqldb.ErrDBNotFound) {
				return errs.New(errs.NotFound, err)
			}
			return errs.New(errs.Internal, err)
		}
		n := len(note.Content)
		if n > 200 {
			n = 200
		}
		clauseText = note.Content[:n]

		switch body.NewType {
		case "task":
			title := truncate(note.Content, 100)
			task, err := a.taskBus.Create(ctx, taskbus.NewTask{
				Title:       title,
				Description: "",
				ContextID:   note.ContextID,
				Status:      taskstatus.MustParse("todo"),
				Priority:    taskpriority.MustParse("medium"),
				Energy:      taskenergy.MustParse("medium"),
				Unconfirmed: false,
			})
			if err != nil {
				return errs.New(errs.Internal, err)
			}
			newID = task.ID

		case "event":
			title := truncate(note.Content, 100)
			event, err := a.eventBus.Create(ctx, eventbus.NewEvent{
				Title:       title,
				Description: "",
				ContextID:   note.ContextID,
				StartsAt:    now.Add(1 * time.Hour),
				EndsAt:      now.Add(2 * time.Hour),
				AllDay:      false,
				RawInputID:  nil,
				Unconfirmed: false,
			})
			if err != nil {
				return errs.New(errs.Internal, err)
			}
			newID = event.ID
		}

		if err := a.noteBus.Delete(ctx, note); err != nil {
			return errs.New(errs.Internal, err)
		}

	case "event":
		event, err := a.eventBus.QueryByID(ctx, itemID)
		if err != nil {
			if errors.Is(err, sqldb.ErrDBNotFound) {
				return errs.New(errs.NotFound, err)
			}
			return errs.New(errs.Internal, err)
		}
		clauseText = event.Title

		switch body.NewType {
		case "task":
			task, err := a.taskBus.Create(ctx, taskbus.NewTask{
				Title:       event.Title,
				Description: event.Description,
				ContextID:   event.ContextID,
				Status:      taskstatus.MustParse("todo"),
				Priority:    taskpriority.MustParse("medium"),
				Energy:      taskenergy.MustParse("medium"),
				Unconfirmed: false,
			})
			if err != nil {
				return errs.New(errs.Internal, err)
			}
			newID = task.ID

		case "note":
			content := event.Title
			if strings.TrimSpace(event.Description) != "" {
				content = event.Title + ": " + event.Description
			}
			note, err := a.noteBus.Create(ctx, notebus.NewNote{
				Content:     content,
				Source:      "correction",
				ContextID:   event.ContextID,
				RawInputID:  nil,
				Unconfirmed: false,
			})
			if err != nil {
				return errs.New(errs.Internal, err)
			}
			newID = note.ID
		}

		if err := a.eventBus.Delete(ctx, event); err != nil {
			return errs.New(errs.Internal, err)
		}
	}

	if _, err := a.correctionBus.Record(ctx, classificationcorrectionbus.NewCorrection{
		ClauseText:    clauseText,
		PredictedType: body.ItemType,
		Confidence:    0.0,
		ActualType:    body.NewType,
		Source:        "correction_applied",
	}); err != nil {
		// Non-fatal: log but don't fail the request
		_ = err
	}

	return CorrectionResult{
		ID:   newID.String(),
		Type: body.NewType,
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
