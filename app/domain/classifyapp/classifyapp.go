package classifyapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/taskstatus"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	log              *logger.Logger
	taskBus          *taskbus.Business
	noteBus          *notebus.Business
	eventBus         *eventbus.Business
	contextBus       *contextbus.Business
	clarificationBus *clarificationbus.Business
	extractor        extractor.Extractor
}

// classify routes to the correct entity classifier based on entity_type query param.
// Default is "task" for backward compatibility.
// POST /api/v1/tasks/classify and POST /api/v1/classify both route here.
func (a *app) classify(ctx context.Context, r *http.Request) web.Encoder {
	entityType := r.URL.Query().Get("entity_type")
	if entityType == "" {
		entityType = "task"
	}

	switch entityType {
	case "task":
		return a.classifyTasks(ctx)
	case "note":
		return a.classifyNotes(ctx)
	case "event":
		return a.classifyEvents(ctx)
	default:
		return errs.Newf(errs.InvalidArgument, "unsupported entity_type %q (use task, note, or event)", entityType)
	}
}

func (a *app) classifyTasks(ctx context.Context) web.Encoder {
	openStatus := taskstatus.Open
	tasks, err := a.taskBus.Query(ctx, taskbus.QueryFilter{Status: &openStatus}, taskbus.DefaultOrderBy, page.New(1, 200))
	if err != nil {
		return errs.Newf(errs.Internal, "query tasks: %s", err)
	}

	var unlinked []taskbus.Task
	for _, t := range tasks {
		if t.ContextID == nil {
			unlinked = append(unlinked, t)
		}
	}

	if len(unlinked) == 0 {
		return ClassifyAccepted{Message: "No unlinked tasks to classify", UnlinkedCount: 0}
	}

	ctxRefs, encErr := a.fetchContextRefs(ctx)
	if encErr != nil {
		return encErr
	}

	n := len(unlinked)
	go func() {
		bgCtx := context.Background()
		for _, task := range unlinked {
			a.classifyEntity(bgCtx, "task", task.ID, fmt.Sprintf("Task: %s\nDescription: %s", task.Title, task.Description), ctxRefs)
		}
	}()

	return ClassifyAccepted{Message: fmt.Sprintf("Classification started for %d tasks", n), UnlinkedCount: n}
}

func (a *app) classifyNotes(ctx context.Context) web.Encoder {
	notes, err := a.noteBus.Query(ctx, notebus.QueryFilter{}, notebus.DefaultOrderBy, page.New(1, 200))
	if err != nil {
		return errs.Newf(errs.Internal, "query notes: %s", err)
	}

	var unlinked []notebus.Note
	for _, n := range notes {
		if n.ContextID == nil {
			unlinked = append(unlinked, n)
		}
	}

	if len(unlinked) == 0 {
		return ClassifyAccepted{Message: "No unlinked notes to classify", UnlinkedCount: 0}
	}

	ctxRefs, encErr := a.fetchContextRefs(ctx)
	if encErr != nil {
		return encErr
	}

	n := len(unlinked)
	go func() {
		bgCtx := context.Background()
		for _, note := range unlinked {
			a.classifyEntity(bgCtx, "note", note.ID, fmt.Sprintf("Note: %s", note.Content), ctxRefs)
		}
	}()

	return ClassifyAccepted{Message: fmt.Sprintf("Classification started for %d notes", n), UnlinkedCount: n}
}

func (a *app) classifyEvents(ctx context.Context) web.Encoder {
	events, err := a.eventBus.Query(ctx, eventbus.QueryFilter{}, eventbus.DefaultOrderBy, page.New(1, 200))
	if err != nil {
		return errs.Newf(errs.Internal, "query events: %s", err)
	}

	var unlinked []eventbus.Event
	for _, e := range events {
		if e.ContextID == nil {
			unlinked = append(unlinked, e)
		}
	}

	if len(unlinked) == 0 {
		return ClassifyAccepted{Message: "No unlinked events to classify", UnlinkedCount: 0}
	}

	ctxRefs, encErr := a.fetchContextRefs(ctx)
	if encErr != nil {
		return encErr
	}

	n := len(unlinked)
	go func() {
		bgCtx := context.Background()
		for _, event := range unlinked {
			a.classifyEntity(bgCtx, "event", event.ID, fmt.Sprintf("Event: %s\nDescription: %s", event.Title, event.Description), ctxRefs)
		}
	}()

	return ClassifyAccepted{Message: fmt.Sprintf("Classification started for %d events", n), UnlinkedCount: n}
}

// fetchContextRefs builds the active context list for the extractor.
func (a *app) fetchContextRefs(ctx context.Context) ([]extractor.ContextRef, web.Encoder) {
	activeStatus := contextbus.Active
	contexts, err := a.contextBus.Query(ctx, contextbus.QueryFilter{Status: &activeStatus}, contextbus.DefaultOrderBy, page.New(1, 50))
	if err != nil {
		return nil, errs.Newf(errs.Internal, "query contexts: %s", err)
	}

	refs := make([]extractor.ContextRef, len(contexts))
	for i, c := range contexts {
		refs[i] = extractor.ContextRef{ID: c.ID.String(), Title: c.Title}
	}
	return refs, nil
}

// classifyEntity extracts a context suggestion for one entity and either updates directly
// (confidence >= 0.7) or creates a clarification card (confidence < 0.7).
// Must be called in a background goroutine.
func (a *app) classifyEntity(ctx context.Context, entityType string, entityID uuid.UUID, text string, ctxRefs []extractor.ContextRef) {
	extraction, err := a.extractor.ExtractText(ctx, text, ctxRefs, "")
	if err != nil {
		a.log.Error(ctx, "classifyEntity: extractor failed", "entityType", entityType, "entityID", entityID, "error", err)
		return
	}

	if extraction.SuggestedContextID == nil || *extraction.SuggestedContextID == "" {
		return
	}

	ctxID, err := uuid.Parse(*extraction.SuggestedContextID)
	if err != nil {
		a.log.Error(ctx, "classifyEntity: failed to parse suggested context ID", "entityType", entityType, "entityID", entityID, "suggestedContextID", *extraction.SuggestedContextID, "error", err)
		return
	}

	if _, err := a.contextBus.QueryByID(ctx, ctxID); err != nil {
		a.log.Error(ctx, "classifyEntity: context lookup failed", "entityType", entityType, "entityID", entityID, "contextID", ctxID, "error", err)
		return
	}

	if extraction.ContextConfidence >= 0.7 {
		switch entityType {
		case "task":
			task, err := a.taskBus.QueryByID(ctx, entityID)
			if err != nil {
				a.log.Error(ctx, "classifyEntity: task lookup failed", "entityID", entityID, "error", err)
				return
			}
			a.taskBus.Update(ctx, task, taskbus.UpdateTask{ContextID: &ctxID}) //nolint:errcheck
		case "note":
			note, err := a.noteBus.QueryByID(ctx, entityID)
			if err != nil {
				a.log.Error(ctx, "classifyEntity: note lookup failed", "entityID", entityID, "error", err)
				return
			}
			a.noteBus.Update(ctx, note, notebus.UpdateNote{ContextID: &ctxID}) //nolint:errcheck
		case "event":
			event, err := a.eventBus.QueryByID(ctx, entityID)
			if err != nil {
				a.log.Error(ctx, "classifyEntity: event lookup failed", "entityID", entityID, "error", err)
				return
			}
			a.eventBus.Update(ctx, event, eventbus.UpdateEvent{ContextID: &ctxID}) //nolint:errcheck
		}
	} else {
		busCtxRefs := make([]clarificationbus.ContextRef, len(ctxRefs))
		for i, r := range ctxRefs {
			busCtxRefs[i] = clarificationbus.ContextRef{ID: r.ID, Title: r.Title}
		}
		optJSON, _ := json.Marshal(clarificationbus.ContextAssignmentOptions{
			SuggestedContext:  ctxID.String(),
			Confidence:        extraction.ContextConfidence,
			AvailableContexts: busCtxRefs,
		})
		guess, _ := json.Marshal(map[string]string{"context_id": ctxID.String()})
		guessRaw := json.RawMessage(guess)
		reasoning := fmt.Sprintf("AI matched %s to context with %.0f%% confidence", entityType, extraction.ContextConfidence*100)

		a.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{ //nolint:errcheck
			Kind:          clarificationkind.ContextAssignment,
			SubjectType:   entityType,
			SubjectID:     entityID,
			Question:      fmt.Sprintf("Which context does this %s belong to?", entityType),
			ClaudeGuess:   &guessRaw,
			Reasoning:     &reasoning,
			AnswerOptions: json.RawMessage(optJSON),
		})
	}
}
