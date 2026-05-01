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
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	log               *logger.Logger
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
		a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "answer nil")
		return
	}

	// Check for free_text override — triggers correction + cleanup + re-ingest.
	var freeTextAnswer struct {
		FreeText string `json:"free_text"`
	}
	if jsonErr := json.Unmarshal(*item.Answer, &freeTextAnswer); jsonErr == nil && freeTextAnswer.FreeText != "" {
		if item.SubjectType == "raw_input" {
			// Delete notes before tasks: notes may reference tasks via task_id with
			// ON DELETE SET NULL, which can violate the notes_has_target check constraint
			// if the note has no context_id. Deleting notes first avoids this.
			_ = a.noteBus.DeleteByRawInputUnconfirmed(ctx, item.SubjectID)
			_ = a.taskBus.DeleteByRawInputUnconfirmed(ctx, item.SubjectID)
			_ = a.eventBus.DeleteByRawInputUnconfirmed(ctx, item.SubjectID)
			ri, err := a.rawinputBus.QueryByID(ctx, item.SubjectID)
			if err == nil {
				correction := freeTextAnswer.FreeText
				_, _ = a.rawinputBus.Update(ctx, ri, rawinputbus.UpdateRawInput{UserCorrection: &correction})
				_, _ = a.rawinputBus.ResetForReprocess(ctx, item.SubjectID)
			}
		}
		return
	}

	switch item.Kind {
	case clarificationkind.ContextAssignment:
		// Answer should contain a context_id string to assign
		var answer struct {
			ContextID string `json:"context_id"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "unmarshal context_assignment answer", "error", err)
			return
		}
		contextID, err := uuid.Parse(answer.ContextID)
		if err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "parse context_id", "error", err)
			return
		}
		// Update the subject based on subject_type
		switch item.SubjectType {
		case "task":
			task, err := a.taskBus.QueryByID(ctx, item.SubjectID)
			if err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "query task for context_assignment", "error", err)
				return
			}
			if _, err := a.taskBus.Update(ctx, task, taskbus.UpdateTask{ContextID: &contextID}); err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "update task context", "error", err)
				return
			}
		case "note":
			note, err := a.noteBus.QueryByID(ctx, item.SubjectID)
			if err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "query note for context_assignment", "error", err)
				return
			}
			if _, err := a.noteBus.Update(ctx, note, notebus.UpdateNote{ContextID: &contextID}); err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "update note context", "error", err)
				return
			}
		case "event":
			event, err := a.eventBus.QueryByID(ctx, item.SubjectID)
			if err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "query event for context_assignment", "error", err)
				return
			}
			if _, err := a.eventBus.Update(ctx, event, eventbus.UpdateEvent{ContextID: &contextID}); err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "update event context", "error", err)
				return
			}
		case "email":
			email, err := a.emailBus.QueryByID(ctx, item.SubjectID)
			if err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "query email for context_assignment", "error", err)
				return
			}
			if _, err := a.emailBus.Update(ctx, email, emailbus.UpdateEmail{ContextID: &contextID}); err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "update email context", "error", err)
				return
			}
		}

	case clarificationkind.AmbiguousDeadline:
		var answer struct {
			DueDate string `json:"due_date"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "unmarshal ambiguous_deadline answer", "error", err)
			return
		}
		if answer.DueDate == "" {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "empty due_date")
			return
		}
		dueDate, err := time.Parse("2006-01-02", answer.DueDate)
		if err != nil {
			dueDate, err = time.Parse(time.RFC3339, answer.DueDate)
			if err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "parse due_date", "error", err)
				return
			}
		}
		if item.SubjectType != "task" {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "ambiguous_deadline only supports task subject")
			return
		}
		task, err := a.taskBus.QueryByID(ctx, item.SubjectID)
		if err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "query task for ambiguous_deadline", "error", err)
			return
		}
		if _, err := a.taskBus.Update(ctx, task, taskbus.UpdateTask{DueDate: &dueDate}); err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "update task due_date", "error", err)
			return
		}

	case clarificationkind.AmbiguousAction:
		var answer struct {
			Selected *int `json:"selected"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "unmarshal ambiguous_action answer", "error", err)
			return
		}
		if answer.Selected == nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "missing selected index")
			return
		}
		var opts clarificationbus.AmbiguousActionOptions
		if err := json.Unmarshal(item.AnswerOptions, &opts); err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "unmarshal ambiguous_action options", "error", err)
			return
		}
		if len(opts.Interpretations) == 0 {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "empty interpretations")
			return
		}
		if *answer.Selected < 0 || *answer.Selected >= len(opts.Interpretations) {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "selected out of range")
			return
		}
		nt := taskbus.NewTask{
			Title:    opts.Interpretations[*answer.Selected],
			Status:   taskstatus.Open,
			Priority: taskpriority.Medium,
			Energy:   taskenergy.Medium,
		}
		if _, err := a.taskBus.Create(ctx, nt); err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "create task", "error", err)
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
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "unmarshal new_context answer", "error", err)
			return
		}
		switch answer.Action {
		case "confirm":
			if answer.Title != "" || answer.Description != "" {
				c, err := a.contextBus.QueryByID(ctx, item.SubjectID)
				if err != nil {
					a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "query context", "error", err)
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
					a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "update context", "error", err)
					return
				}
			}
		case "merge":
			if answer.MergeTarget == "" {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "empty merge_target")
				return
			}
			c, err := a.contextBus.QueryByID(ctx, item.SubjectID)
			if err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "query context", "error", err)
				return
			}
			if err := a.contextBus.Delete(ctx, c); err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "delete context", "error", err)
				return
			}
		}

	case clarificationkind.InactivityPrompt:
		var answer struct {
			Action string `json:"action"`
			Note   string `json:"note"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "unmarshal inactivity_prompt answer", "error", err)
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
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "add thread entry", "error", err)
			return
		}

		switch answer.Action {
		case "completed":
			if item.SubjectType == "task" {
				task, err := a.taskBus.QueryByID(ctx, item.SubjectID)
				if err != nil {
					a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "query task", "error", err)
					return
				}
				done := taskstatus.Done
				if _, err := a.taskBus.Update(ctx, task, taskbus.UpdateTask{Status: &done}); err != nil {
					a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "update task status", "error", err)
					return
				}
			} else if item.SubjectType == "context" {
				c, err := a.contextBus.QueryByID(ctx, item.SubjectID)
				if err != nil {
					a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "query context", "error", err)
					return
				}
				closed := contextbus.Closed
				if _, err := a.contextBus.Update(ctx, c, contextbus.UpdateContext{Status: &closed}); err != nil {
					a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "update context status", "error", err)
					return
				}
			}
		}

	case clarificationkind.ContextDebrief:
		var answer struct {
			Response string `json:"response"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "unmarshal context_debrief answer", "error", err)
			return
		}
		if answer.Response == "" {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "empty response")
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
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "record debrief observation", "error", err)
			return
		}

		// Check if all debrief cards for this context are resolved
		kind := clarificationkind.ContextDebrief
		pending := clarificationstatus.Pending
		snoozed := clarificationstatus.Snoozed
		subjectType := "context"

		pendingCount, err := a.clarificationBus.Count(ctx, clarificationbus.QueryFilter{
			Kind: &kind, Status: &pending, SubjectType: &subjectType, SubjectID: &item.SubjectID,
		})
		if err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "query pending debrief count", "error", err)
			return
		}
		snoozedCount, err := a.clarificationBus.Count(ctx, clarificationbus.QueryFilter{
			Kind: &kind, Status: &snoozed, SubjectType: &subjectType, SubjectID: &item.SubjectID,
		})
		if err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "query snoozed debrief count", "error", err)
			return
		}

		if pendingCount == 0 && snoozedCount == 0 {
			c, err := a.contextBus.QueryByID(ctx, item.SubjectID)
			if err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "query context", "error", err)
				return
			}
			done := debriefstatus.Done
			if _, err := a.contextBus.Update(ctx, c, contextbus.UpdateContext{DebriefStatus: &done}); err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "update debrief_status", "error", err)
				return
			}
		}

	case clarificationkind.StaleTask:
		var answer struct {
			Status string `json:"status"`
			Note   string `json:"note"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "unmarshal stale_task answer", "error", err)
			return
		}
		if answer.Status == "" && answer.Note == "" {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "empty status and note")
			return
		}
		if answer.Note != "" {
			if _, err := a.threadBus.AddEntry(ctx, threadbus.NewThreadEntry{
				SubjectType: item.SubjectType,
				SubjectID:   item.SubjectID,
				Kind:        threadentrykind.Update,
				Content:     answer.Note,
				Source:      threadsource.System,
			}); err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "add thread entry", "error", err)
				// continue — thread entry failure should not block status update
			}
		}
		if answer.Status != "" {
			task, err := a.taskBus.QueryByID(ctx, item.SubjectID)
			if err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "query task", "error", err)
				return
			}
			status, err := taskstatus.Parse(answer.Status)
			if err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "parse task status", "error", err)
				return
			}
			if _, err := a.taskBus.Update(ctx, task, taskbus.UpdateTask{Status: &status}); err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "update task status", "error", err)
				return
			}
		}

	case clarificationkind.EntityLink:
		var answer struct {
			Confirmed bool `json:"confirmed"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "unmarshal entity_link answer", "error", err)
			return
		}
		if !answer.Confirmed {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "not confirmed")
			return
		}
		var opts clarificationbus.EntityLinkOptions
		if err := json.Unmarshal(item.AnswerOptions, &opts); err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "unmarshal entity_link options", "error", err)
			return
		}
		sourceID, err := uuid.Parse(opts.SourceID)
		if err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "parse source_id", "error", err)
			return
		}
		targetID, err := uuid.Parse(opts.TargetID)
		if err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "parse target_id", "error", err)
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
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "create entity_link", "error", err)
			return
		}

	case clarificationkind.TaskDebrief:
		var answer struct {
			Value string `json:"value"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "unmarshal task_debrief answer", "error", err)
			return
		}
		if answer.Value == "" || answer.Value == "skip" {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "empty or skip value")
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
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "record task_debrief observation", "error", err)
			return
		}

	case clarificationkind.WeeklyReview:
		var answer struct {
			SelectedTaskIDs []string `json:"selected_task_ids"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "unmarshal weekly_review answer", "error", err)
			return
		}
		if len(answer.SelectedTaskIDs) == 0 {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "empty selected_task_ids")
			return
		}
		for _, taskIDStr := range answer.SelectedTaskIDs {
			taskID, err := uuid.Parse(taskIDStr)
			if err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "parse weekly_review task_id", "error", err)
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
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "record weekly_review observation", "error", err)
				continue
			}
		}

	case clarificationkind.TypeAssignment:
		// Answer: {actual_type: "task"|"note"|"event"}
		var answer struct {
			ActualType string `json:"actual_type"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "unmarshal type_assignment answer", "error", err)
			return
		}
		if answer.ActualType == "" {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "empty actual_type")
			return
		}

		// Parse the options to get clause_text, predicted_type, and confidence for logging.
		var opts clarificationbus.TypeAssignmentOptions
		if err := json.Unmarshal(item.AnswerOptions, &opts); err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "unmarshal type_assignment options", "error", err)
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
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "record type_assignment correction", "error", err)
			}
		}

		// Clear the unconfirmed flag on the subject item.
		falseVal := false
		switch item.SubjectType {
		case "task":
			task, err := a.taskBus.QueryByID(ctx, item.SubjectID)
			if err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "query task for type_assignment", "error", err)
				return
			}
			if _, err := a.taskBus.Update(ctx, task, taskbus.UpdateTask{Unconfirmed: &falseVal}); err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "clear task unconfirmed", "error", err)
				return
			}
		case "note":
			note, err := a.noteBus.QueryByID(ctx, item.SubjectID)
			if err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "query note for type_assignment", "error", err)
				return
			}
			if _, err := a.noteBus.Update(ctx, note, notebus.UpdateNote{Unconfirmed: &falseVal}); err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "clear note unconfirmed", "error", err)
				return
			}
		case "event":
			event, err := a.eventBus.QueryByID(ctx, item.SubjectID)
			if err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "query event for type_assignment", "error", err)
				return
			}
			if _, err := a.eventBus.Update(ctx, event, eventbus.UpdateEvent{Unconfirmed: &falseVal}); err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "clear event unconfirmed", "error", err)
				return
			}
		}

	case clarificationkind.EventPrep:
		// Answer: {confirmed: true|false}
		// No automated side-effect — user acknowledges the suggestion and manually schedules tasks.
		// Future: could auto-add tasks to next plan generation.

	case clarificationkind.AmbiguousEntityMatch:
		// Answer: {choice: "use_existing" | "create_new"}
		// If user chose "use_existing", delete the unconfirmed duplicate extracted from the raw_input
		// If user chose "create_new", just mark resolved (the unconfirmed entity stays for confirmation)
		var answer struct {
			Choice string `json:"choice"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "unmarshal ambiguous_entity_match answer", "error", err)
			return
		}
		if answer.Choice == "" {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "empty choice")
			return
		}

		if answer.Choice == "use_existing" {
			// Parse options to get the entity type that was extracted
			var opts clarificationbus.AmbiguousEntityMatchOptions
			if err := json.Unmarshal(item.AnswerOptions, &opts); err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "unmarshal ambiguous_entity_match options", "error", err)
				return
			}

			// Delete the unconfirmed entity of this type that was created from the raw_input.
			// item.SubjectID is the raw_input ID; opts.CandidateType is the entity type ("task"/"event"/"note").
			switch opts.CandidateType {
			case "task":
				if err := a.taskBus.DeleteByRawInputUnconfirmed(ctx, item.SubjectID); err != nil {
					// Non-fatal: log and continue
					a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "delete unconfirmed task", "error", err)
				}
			case "event":
				if err := a.eventBus.DeleteByRawInputUnconfirmed(ctx, item.SubjectID); err != nil {
					// Non-fatal: log and continue
					a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "delete unconfirmed event", "error", err)
				}
			case "note":
				if err := a.noteBus.DeleteByRawInputUnconfirmed(ctx, item.SubjectID); err != nil {
					// Non-fatal: log and continue
					a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "delete unconfirmed note", "error", err)
				}
			}
		}

	case clarificationkind.KnowledgeGap:
		// Answer: {answer_text: string}, {selected_option: string}, or {dismissed: true}
		var answer struct {
			AnswerText     string `json:"answer_text"`
			SelectedOption string `json:"selected_option"`
			Dismissed      bool   `json:"dismissed"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "unmarshal knowledge_gap answer", "error", err)
			return
		}
		// Precedence: dismissed > selected_option > answer_text
		if answer.Dismissed {
			if _, err := a.clarificationBus.Dismiss(ctx, item); err != nil {
				a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "dismiss knowledge_gap", "error", err)
				return
			}
			return
		}
		// Determine the note content: selected_option takes precedence over answer_text
		noteContent := answer.SelectedOption
		if noteContent == "" {
			noteContent = answer.AnswerText
		}
		// If user provided answer (either selected_option or answer_text), create a note
		if noteContent == "" {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "empty knowledge_gap note content")
			return
		}
		// Create a note from the answer. If the subject is a task, set TaskID directly so
		// the foreign-key column reflects the link; otherwise rely on the entity_link row below.
		newNote := notebus.NewNote{
			Content: noteContent,
			Source:  "clarification",
		}
		if item.SubjectType == "task" {
			subjectID := item.SubjectID
			newNote.TaskID = &subjectID
		}
		note, err := a.noteBus.Create(ctx, newNote)
		if err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "create knowledge_gap note", "error", err)
			return
		}
		// Link the note back to the subject entity.
		if _, err := a.entityLinkBus.Create(ctx, entitylinkbus.NewEntityLink{
			SourceType: "note",
			SourceID:   note.ID,
			TargetType: item.SubjectType,
			TargetID:   item.SubjectID,
			Confidence: 1.0,
			Kind:       "knowledge_gap_answer",
		}); err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "create knowledge_gap entity_link", "error", err)
			return
		}

	case clarificationkind.VoiceReference:
		var answer struct {
			ResolvedText string `json:"resolved_text"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "unmarshal voice_reference answer", "error", err)
			return
		}
		if answer.ResolvedText == "" {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "empty resolved_text")
			return
		}
		// Parse options to get the original text
		var opts clarificationbus.VoiceReferenceOptions
		if err := json.Unmarshal(item.AnswerOptions, &opts); err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "unmarshal voice_reference options", "error", err)
			return
		}
		// Add thread entry to raw_input with format: "Voice reference '{originalText}' → resolved as '{resolvedText}'"
		threadContent := fmt.Sprintf("Voice reference %q → resolved as %q", opts.OriginalText, answer.ResolvedText)
		if _, err := a.threadBus.AddEntry(ctx, threadbus.NewThreadEntry{
			SubjectType: item.SubjectType,
			SubjectID:   item.SubjectID,
			Kind:        threadentrykind.Update,
			Content:     threadContent,
			Source:      threadsource.System,
		}); err != nil {
			a.log.Warn(ctx, "clarification.dispatch", "clarification_id", item.ID, "kind", item.Kind, "subject_type", item.SubjectType, "subject_id", item.SubjectID, "reason", "add voice_reference thread entry", "error", err)
			// continue — thread entry failure should not block resolution
		}

	}
}
