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
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/business/domain/classificationcorrectionbus"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/business/types/debriefstatus"
	"github.com/casebrophy/planner/business/types/taskenergy"
	"github.com/casebrophy/planner/business/types/taskpriority"
	"github.com/casebrophy/planner/business/types/taskstatus"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	db            *sqlx.DB
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

	validTypes := map[string]bool{"task": true, "note": true, "event": true}
	if !validTypes[body.ItemType] {
		return errs.New(errs.InvalidArgument, fmt.Errorf("invalid item_type %q", body.ItemType))
	}
	if !validTypes[body.NewType] {
		return errs.New(errs.InvalidArgument, fmt.Errorf("invalid new_type %q", body.NewType))
	}
	if body.ItemType == body.NewType {
		return errs.New(errs.InvalidArgument, fmt.Errorf("new_type must differ from item_type"))
	}

	itemID, err := uuid.Parse(body.ItemID)
	if err != nil {
		return errs.New(errs.InvalidArgument, fmt.Errorf("invalid item_id: %w", err))
	}

	now := time.Now()
	var newID uuid.UUID
	var clauseText string

	// Fetch source item OUTSIDE the transaction so we can map ErrDBNotFound→404 cleanly.
	var srcTask taskbus.Task
	var srcNote notebus.Note
	var srcEvent eventbus.Event

	switch body.ItemType {
	case "task":
		srcTask, err = a.taskBus.QueryByID(ctx, itemID)
		if err != nil {
			if errors.Is(err, sqldb.ErrDBNotFound) {
				return errs.New(errs.NotFound, err)
			}
			return errs.New(errs.Internal, err)
		}
		clauseText = srcTask.Title

	case "note":
		srcNote, err = a.noteBus.QueryByID(ctx, itemID)
		if err != nil {
			if errors.Is(err, sqldb.ErrDBNotFound) {
				return errs.New(errs.NotFound, err)
			}
			return errs.New(errs.Internal, err)
		}
		n := len(srcNote.Content)
		if n > 200 {
			n = 200
		}
		clauseText = srcNote.Content[:n]

	case "event":
		srcEvent, err = a.eventBus.QueryByID(ctx, itemID)
		if err != nil {
			if errors.Is(err, sqldb.ErrDBNotFound) {
				return errs.New(errs.NotFound, err)
			}
			return errs.New(errs.Internal, err)
		}
		clauseText = srcEvent.Title
	}

	tx, err := a.db.BeginTxx(ctx, nil)
	if err != nil {
		return errs.New(errs.Internal, fmt.Errorf("begin tx: %w", err))
	}
	defer func() { _ = tx.Rollback() }()

	// Lineage preservation (planner-bztz, 2026-04-29): corrections are semantic reclassifications,
	// not new captures, so we preserve RawInputID and CreatedAt from the source. UpdatedAt resets
	// to now (the item was just modified). Unconfirmed is forced false — the correction itself is
	// user confirmation. Tags are copied for task↔note paths only (no event_tags table). Inline
	// struct construction (vs bus Create) is required to participate in the caller's tx; defaults
	// must mirror taskbus/notebus/eventbus Create paths.
	switch body.ItemType {
	case "task":
		switch body.NewType {
		case "note":
			content := srcTask.Title
			if strings.TrimSpace(srcTask.Description) != "" {
				content = srcTask.Title + ": " + srcTask.Description
			}
			newNote := notebus.Note{
				ID:          uuid.New(),
				ContextID:   srcTask.ContextID,
				TaskID:      nil,
				Content:     content,
				Source:      "correction",
				RawInputID:  srcTask.RawInputID,
				Unconfirmed: false,
				CreatedAt:   srcTask.CreatedAt,
				UpdatedAt:   now,
			}
			if err := a.noteBus.CreateWithTx(ctx, tx, newNote); err != nil {
				return errs.New(errs.Internal, err)
			}
			newID = newNote.ID

			// Copy tags from source task to new note
			if err := copyTaskTagsToNoteTags(ctx, tx, srcTask.ID, newNote.ID); err != nil {
				return errs.New(errs.Internal, err)
			}

		case "event":
			newEvent := eventbus.Event{
				ID:          uuid.New(),
				ContextID:   srcTask.ContextID,
				Title:       srcTask.Title,
				Description: srcTask.Description,
				Location:    nil,
				StartsAt:    now.Add(1 * time.Hour),
				EndsAt:      now.Add(2 * time.Hour),
				AllDay:      false,
				RawInputID:  srcTask.RawInputID,
				Unconfirmed: false,
				CreatedAt:   srcTask.CreatedAt,
				UpdatedAt:   now,
			}
			if err := a.eventBus.CreateWithTx(ctx, tx, newEvent); err != nil {
				return errs.New(errs.Internal, err)
			}
			newID = newEvent.ID
		}

		if err := a.taskBus.DeleteWithTx(ctx, tx, srcTask); err != nil {
			return errs.New(errs.Internal, err)
		}

	case "note":
		switch body.NewType {
		case "task":
			title := truncate(srcNote.Content, 100)
			newTask := taskbus.Task{
				ID:            uuid.New(),
				ContextID:     srcNote.ContextID,
				RawInputID:    srcNote.RawInputID,
				Title:         title,
				Description:   "",
				Status:        taskstatus.Open,
				Priority:      taskpriority.Medium,
				Energy:        taskenergy.Medium,
				DebriefStatus: debriefstatus.Pending,
				Unconfirmed:   false,
				CreatedAt:     srcNote.CreatedAt,
				UpdatedAt:     now,
			}
			if err := a.taskBus.CreateWithTx(ctx, tx, newTask); err != nil {
				return errs.New(errs.Internal, err)
			}
			newID = newTask.ID

			// Copy tags from source note to new task
			if err := copyNoteTagsToTaskTags(ctx, tx, srcNote.ID, newTask.ID); err != nil {
				return errs.New(errs.Internal, err)
			}

		case "event":
			title := truncate(srcNote.Content, 100)
			newEvent := eventbus.Event{
				ID:          uuid.New(),
				ContextID:   srcNote.ContextID,
				Title:       title,
				Description: "",
				Location:    nil,
				StartsAt:    now.Add(1 * time.Hour),
				EndsAt:      now.Add(2 * time.Hour),
				AllDay:      false,
				RawInputID:  srcNote.RawInputID,
				Unconfirmed: false,
				CreatedAt:   srcNote.CreatedAt,
				UpdatedAt:   now,
			}
			if err := a.eventBus.CreateWithTx(ctx, tx, newEvent); err != nil {
				return errs.New(errs.Internal, err)
			}
			newID = newEvent.ID
		}

		if err := a.noteBus.DeleteWithTx(ctx, tx, srcNote); err != nil {
			return errs.New(errs.Internal, err)
		}

	case "event":
		switch body.NewType {
		case "task":
			newTask := taskbus.Task{
				ID:            uuid.New(),
				ContextID:     srcEvent.ContextID,
				RawInputID:    srcEvent.RawInputID,
				Title:         srcEvent.Title,
				Description:   srcEvent.Description,
				Status:        taskstatus.Open,
				Priority:      taskpriority.Medium,
				Energy:        taskenergy.Medium,
				DebriefStatus: debriefstatus.Pending,
				Unconfirmed:   false,
				CreatedAt:     srcEvent.CreatedAt,
				UpdatedAt:     now,
			}
			if err := a.taskBus.CreateWithTx(ctx, tx, newTask); err != nil {
				return errs.New(errs.Internal, err)
			}
			newID = newTask.ID

		case "note":
			content := srcEvent.Title
			if strings.TrimSpace(srcEvent.Description) != "" {
				content = srcEvent.Title + ": " + srcEvent.Description
			}
			newNote := notebus.Note{
				ID:          uuid.New(),
				ContextID:   srcEvent.ContextID,
				TaskID:      nil,
				Content:     content,
				Source:      "correction",
				RawInputID:  srcEvent.RawInputID,
				Unconfirmed: false,
				CreatedAt:   srcEvent.CreatedAt,
				UpdatedAt:   now,
			}
			if err := a.noteBus.CreateWithTx(ctx, tx, newNote); err != nil {
				return errs.New(errs.Internal, err)
			}
			newID = newNote.ID
		}

		if err := a.eventBus.DeleteWithTx(ctx, tx, srcEvent); err != nil {
			return errs.New(errs.Internal, err)
		}
	}

	// Record correction WITHIN the transaction; failure is now FATAL (was silently swallowed).
	if _, err := a.correctionBus.RecordWithTx(ctx, tx, classificationcorrectionbus.NewCorrection{
		ClauseText:    clauseText,
		PredictedType: body.ItemType,
		Confidence:    0.0,
		ActualType:    body.NewType,
		Source:        "correction_applied",
	}); err != nil {
		return errs.New(errs.Internal, err)
	}

	if err := tx.Commit(); err != nil {
		return errs.New(errs.Internal, fmt.Errorf("commit tx: %w", err))
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

func copyTaskTagsToNoteTags(ctx context.Context, tx *sqlx.Tx, taskID, noteID uuid.UUID) error {
	const q = `
		INSERT INTO note_tags (note_id, tag_id)
		SELECT $1, tag_id FROM task_tags WHERE task_id = $2
		ON CONFLICT DO NOTHING`
	if _, err := tx.ExecContext(ctx, q, noteID, taskID); err != nil {
		return fmt.Errorf("copy tags: %w", err)
	}
	return nil
}

func copyNoteTagsToTaskTags(ctx context.Context, tx *sqlx.Tx, noteID, taskID uuid.UUID) error {
	const q = `
		INSERT INTO task_tags (task_id, tag_id)
		SELECT $1, tag_id FROM note_tags WHERE note_id = $2
		ON CONFLICT DO NOTHING`
	if _, err := tx.ExecContext(ctx, q, taskID, noteID); err != nil {
		return fmt.Errorf("copy tags: %w", err)
	}
	return nil
}
