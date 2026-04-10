package clarificationapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/classificationcorrectionbus"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/emailbus"
	"github.com/casebrophy/planner/business/domain/entitylinkbus"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/observationbus"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/clarificationstatus"
	"github.com/casebrophy/planner/business/types/debriefstatus"
	"github.com/casebrophy/planner/business/types/observationkind"
	"github.com/casebrophy/planner/business/types/taskenergy"
	"github.com/casebrophy/planner/business/types/taskpriority"
	"github.com/casebrophy/planner/business/types/taskstatus"
	"github.com/casebrophy/planner/business/types/threadentrykind"
	"github.com/casebrophy/planner/business/types/threadsource"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	clarificationBus  *clarificationbus.Business
	taskBus           *taskbus.Business
	noteBus           *notebus.Business
	eventBus          *eventbus.Business
	contextBus        *contextbus.Business
	emailBus          *emailbus.Business
	observationBus    *observationbus.Business
	rawinputBus       *rawinputbus.Business
	threadBus         *threadbus.Business
	entityLinkBus     *entitylinkbus.Business
	correctionBus     *classificationcorrectionbus.Business
}

func (a *app) queryQueue(ctx context.Context, r *http.Request) web.Encoder {
	pg, err := page.Parse(r.URL.Query().Get("page"), r.URL.Query().Get("rows"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	filter, err := parseFilter(r)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	// Default to pending if no status filter
	if filter.Status == nil {
		pending := clarificationstatus.Pending
		filter.Status = &pending
	}

	orderBy, err := parseOrder(r)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	items, err := a.clarificationBus.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return errs.Newf(errs.Internal, "query: %s", err)
	}

	total, err := a.clarificationBus.Count(ctx, filter)
	if err != nil {
		return errs.Newf(errs.Internal, "count: %s", err)
	}

	return query.NewResult(toAppClarifications(items), total, pg.Number(), pg.RowsPerPage())
}

func (a *app) queryByID(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	item, err := a.clarificationBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	return toAppClarification(item)
}

func (a *app) resolve(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	item, err := a.clarificationBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	var input ResolveInput
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if len(input.Answer) == 0 {
		return errs.Newf(errs.InvalidArgument, "answer is required")
	}

	rc := clarificationbus.ResolveClarificationItem{
		Answer: input.Answer,
	}

	resolved, err := a.clarificationBus.Resolve(ctx, item, rc)
	if err != nil {
		return errs.Newf(errs.Internal, "resolve: %s", err)
	}

	// Resolution dispatcher: map kind + answer → side-effect
	a.dispatchResolution(ctx, resolved)

	return toAppClarification(resolved)
}

func (a *app) snooze(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	item, err := a.clarificationBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	var input SnoozeInput
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	hours := 24
	if input.Hours > 0 {
		hours = input.Hours
	}

	until := time.Now().Add(time.Duration(hours) * time.Hour)

	snoozed, err := a.clarificationBus.Snooze(ctx, item, until)
	if err != nil {
		return errs.Newf(errs.Internal, "snooze: %s", err)
	}

	return toAppClarification(snoozed)
}

func (a *app) dismiss(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	item, err := a.clarificationBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	dismissed, err := a.clarificationBus.Dismiss(ctx, item)
	if err != nil {
		return errs.Newf(errs.Internal, "dismiss: %s", err)
	}

	return toAppClarification(dismissed)
}

func (a *app) countPending(ctx context.Context, r *http.Request) web.Encoder {
	pending := clarificationstatus.Pending
	filter := clarificationbus.QueryFilter{
		Status: &pending,
	}

	n, err := a.clarificationBus.Count(ctx, filter)
	if err != nil {
		return errs.Newf(errs.Internal, "count: %s", err)
	}

	return CountResponse{Count: n}
}

// dispatchResolution performs side-effects based on the resolved clarification's kind and answer.
// Errors are logged but do not fail the resolve response.
func (a *app) dispatchResolution(ctx context.Context, item clarificationbus.ClarificationItem) {
	if item.Answer == nil {
		return
	}

	switch item.Kind {
	case clarificationkind.ContextAssignment:
		// Answer should contain a context_id string to assign
		var answer struct {
			ContextID string `json:"context_id"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil {
			return
		}
		contextID, err := uuid.Parse(answer.ContextID)
		if err != nil {
			return
		}
		// Update the subject based on subject_type
		switch item.SubjectType {
		case "task":
			task, err := a.taskBus.QueryByID(ctx, item.SubjectID)
			if err != nil {
				return
			}
			if _, err := a.taskBus.Update(ctx, task, taskbus.UpdateTask{ContextID: &contextID}); err != nil {
				return
			}
		case "note":
			note, err := a.noteBus.QueryByID(ctx, item.SubjectID)
			if err != nil {
				return
			}
			if _, err := a.noteBus.Update(ctx, note, notebus.UpdateNote{ContextID: &contextID}); err != nil {
				return
			}
		case "event":
			event, err := a.eventBus.QueryByID(ctx, item.SubjectID)
			if err != nil {
				return
			}
			if _, err := a.eventBus.Update(ctx, event, eventbus.UpdateEvent{ContextID: &contextID}); err != nil {
				return
			}
		case "email":
			email, err := a.emailBus.QueryByID(ctx, item.SubjectID)
			if err != nil {
				return
			}
			if _, err := a.emailBus.Update(ctx, email, emailbus.UpdateEmail{ContextID: &contextID}); err != nil {
				return
			}
		}

	case clarificationkind.AmbiguousDeadline:
		var answer struct {
			DueDate string `json:"due_date"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil || answer.DueDate == "" {
			return
		}
		dueDate, err := time.Parse("2006-01-02", answer.DueDate)
		if err != nil {
			dueDate, err = time.Parse(time.RFC3339, answer.DueDate)
			if err != nil {
				return
			}
		}
		_ = dueDate

	case clarificationkind.AmbiguousAction:
		var answer struct {
			IsTask      bool   `json:"is_task"`
			Title       string `json:"title"`
			Description string `json:"description"`
			ContextID   string `json:"context_id"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil {
			return
		}
		if !answer.IsTask {
			return
		}
		nt := taskbus.NewTask{
			Title:       answer.Title,
			Description: answer.Description,
			Status:      taskstatus.Open,
			Priority:    taskpriority.Medium,
			Energy:      taskenergy.Medium,
		}
		if answer.ContextID != "" {
			ctxID, err := uuid.Parse(answer.ContextID)
			if err == nil {
				nt.ContextID = &ctxID
			}
		}
		if _, err := a.taskBus.Create(ctx, nt); err != nil {
			return
		}

	case clarificationkind.NewContext:
		var answer struct {
			Action      string `json:"action"`
			Title       string `json:"title"`
			Description string `json:"description"`
			MergeTarget string `json:"merge_target_id"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil {
			return
		}
		switch answer.Action {
		case "confirm":
			if answer.Title != "" || answer.Description != "" {
				c, err := a.contextBus.QueryByID(ctx, item.SubjectID)
				if err != nil {
					return
				}
				upd := contextbus.UpdateContext{}
				if answer.Title != "" {
					upd.Title = &answer.Title
				}
				if answer.Description != "" {
					upd.Description = &answer.Description
				}
				if _, err := a.contextBus.Update(ctx, c, upd); err != nil {
					return
				}
			}
		case "merge":
			if answer.MergeTarget == "" {
				return
			}
			c, err := a.contextBus.QueryByID(ctx, item.SubjectID)
			if err != nil {
				return
			}
			if err := a.contextBus.Delete(ctx, c); err != nil {
				return
			}
		}

	case clarificationkind.InactivityPrompt:
		var answer struct {
			Action string `json:"action"`
			Note   string `json:"note"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil {
			return
		}

		content := answer.Note
		if content == "" {
			content = fmt.Sprintf("Inactivity check resolved: %s", answer.Action)
		}
		entryKind := threadentrykind.Update
		source := threadsource.System
		if _, err := a.threadBus.AddEntry(ctx, threadbus.NewThreadEntry{
			SubjectType: item.SubjectType,
			SubjectID:   item.SubjectID,
			Kind:        entryKind,
			Content:     content,
			Source:      source,
		}); err != nil {
			return
		}

		switch answer.Action {
		case "completed":
			if item.SubjectType == "task" {
				task, err := a.taskBus.QueryByID(ctx, item.SubjectID)
				if err != nil {
					return
				}
				done := taskstatus.Done
				if _, err := a.taskBus.Update(ctx, task, taskbus.UpdateTask{Status: &done}); err != nil {
					return
				}
			} else if item.SubjectType == "context" {
				c, err := a.contextBus.QueryByID(ctx, item.SubjectID)
				if err != nil {
					return
				}
				closed := contextbus.Closed
				if _, err := a.contextBus.Update(ctx, c, contextbus.UpdateContext{Status: &closed}); err != nil {
					return
				}
			}
		}

	case clarificationkind.ContextDebrief:
		var answer struct {
			Response string `json:"response"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil || answer.Response == "" {
			return
		}
		obsData, _ := json.Marshal(map[string]string{
			"response": answer.Response,
			"question": item.Question,
		})
		if _, err := a.observationBus.Record(ctx, observationbus.NewObservation{
			SubjectType: item.SubjectType,
			SubjectID:   item.SubjectID,
			Kind:        observationkind.Debrief,
			Data:        json.RawMessage(obsData),
			Source:      "user",
			Confidence:  1.0,
			Weight:      2.0,
		}); err != nil {
			return
		}

		// Check if all debrief cards for this context are resolved
		kind := clarificationkind.ContextDebrief
		pending := clarificationstatus.Pending
		snoozed := clarificationstatus.Snoozed
		subjectType := "context"

		pendingCount, _ := a.clarificationBus.Count(ctx, clarificationbus.QueryFilter{
			Kind: &kind, Status: &pending, SubjectType: &subjectType, SubjectID: &item.SubjectID,
		})
		snoozedCount, _ := a.clarificationBus.Count(ctx, clarificationbus.QueryFilter{
			Kind: &kind, Status: &snoozed, SubjectType: &subjectType, SubjectID: &item.SubjectID,
		})

		if pendingCount == 0 && snoozedCount == 0 {
			c, err := a.contextBus.QueryByID(ctx, item.SubjectID)
			if err != nil {
				return
			}
			done := debriefstatus.Done
			if _, err := a.contextBus.Update(ctx, c, contextbus.UpdateContext{DebriefStatus: &done}); err != nil {
				return
			}
		}

	case clarificationkind.StaleTask:
		// Answer may contain a new status
		var answer struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil || answer.Status == "" {
			return
		}
		task, err := a.taskBus.QueryByID(ctx, item.SubjectID)
		if err != nil {
			return
		}
		status, err := taskstatus.Parse(answer.Status)
		if err != nil {
			return
		}
		if _, err := a.taskBus.Update(ctx, task, taskbus.UpdateTask{Status: &status}); err != nil {
			return
		}

	case clarificationkind.EntityLink:
		var answer struct {
			Confirmed bool `json:"confirmed"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil || !answer.Confirmed {
			return
		}
		var opts clarificationbus.EntityLinkOptions
		if err := json.Unmarshal(item.AnswerOptions, &opts); err != nil {
			return
		}
		sourceID, err := uuid.Parse(opts.SourceID)
		if err != nil {
			return
		}
		targetID, err := uuid.Parse(opts.TargetID)
		if err != nil {
			return
		}
		if _, err := a.entityLinkBus.Create(ctx, entitylinkbus.NewEntityLink{
			SourceType: opts.SourceType,
			SourceID:   sourceID,
			TargetType: opts.TargetType,
			TargetID:   targetID,
			Confidence: opts.Confidence,
			Kind:       "ai_suggested",
		}); err != nil {
			return
		}

	case clarificationkind.TaskDebrief:
		var answer struct {
			Value string `json:"value"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil || answer.Value == "" || answer.Value == "skip" {
			return
		}
		obsData, _ := json.Marshal(map[string]string{
			"importance": answer.Value,
			"question":   item.Question,
		})
		if _, err := a.observationBus.Record(ctx, observationbus.NewObservation{
			SubjectType: item.SubjectType,
			SubjectID:   item.SubjectID,
			Kind:        observationkind.Debrief,
			Data:        json.RawMessage(obsData),
			Source:      "user",
			Confidence:  1.0,
			Weight:      2.0,
		}); err != nil {
			return
		}

	case clarificationkind.WeeklyReview:
		var answer struct {
			SelectedTaskIDs []string `json:"selected_task_ids"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil || len(answer.SelectedTaskIDs) == 0 {
			return
		}
		for _, taskIDStr := range answer.SelectedTaskIDs {
			taskID, err := uuid.Parse(taskIDStr)
			if err != nil {
				continue
			}
			obsData, _ := json.Marshal(map[string]string{
				"importance": "high",
				"source":     "weekly_review",
			})
			if _, err := a.observationBus.Record(ctx, observationbus.NewObservation{
				SubjectType: "task",
				SubjectID:   taskID,
				Kind:        observationkind.Debrief,
				Data:        json.RawMessage(obsData),
				Source:      "user",
				Confidence:  1.0,
				Weight:      3.0,
			}); err != nil {
				continue
			}
		}

	case clarificationkind.TypeAssignment:
		// Answer: {actual_type: "task"|"note"|"event"}
		var answer struct {
			ActualType string `json:"actual_type"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil || answer.ActualType == "" {
			return
		}

		// Parse the options to get clause_text, predicted_type, and confidence for logging.
		var opts clarificationbus.TypeAssignmentOptions
		if err := json.Unmarshal(item.AnswerOptions, &opts); err != nil {
			return
		}

		// Log the correction.
		if a.correctionBus != nil {
			if _, err := a.correctionBus.Record(ctx, classificationcorrectionbus.NewCorrection{
				ClauseText:    opts.ClauseText,
				PredictedType: opts.PredictedType,
				Confidence:    opts.Confidence,
				ActualType:    answer.ActualType,
				Source:        "clarification_answered",
			}); err != nil {
				// Non-fatal: log and continue to clear unconfirmed flag.
				_ = err
			}
		}

		// Clear the unconfirmed flag on the subject item.
		falseVal := false
		switch item.SubjectType {
		case "task":
			task, err := a.taskBus.QueryByID(ctx, item.SubjectID)
			if err != nil {
				return
			}
			if _, err := a.taskBus.Update(ctx, task, taskbus.UpdateTask{Unconfirmed: &falseVal}); err != nil {
				return
			}
		case "note":
			note, err := a.noteBus.QueryByID(ctx, item.SubjectID)
			if err != nil {
				return
			}
			if _, err := a.noteBus.Update(ctx, note, notebus.UpdateNote{Unconfirmed: &falseVal}); err != nil {
				return
			}
		case "event":
			event, err := a.eventBus.QueryByID(ctx, item.SubjectID)
			if err != nil {
				return
			}
			if _, err := a.eventBus.Update(ctx, event, eventbus.UpdateEvent{Unconfirmed: &falseVal}); err != nil {
				return
			}
		}
	}
}
