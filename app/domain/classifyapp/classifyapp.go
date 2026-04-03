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
	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/taskstatus"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	taskBus          *taskbus.Business
	contextBus       *contextbus.Business
	clarificationBus *clarificationbus.Business
	extractor        extractor.Extractor
}

func (a *app) classify(ctx context.Context, r *http.Request) web.Encoder {
	// Query all open tasks — we filter for no context in Go since QueryFilter lacks NoContext
	openStatus := taskstatus.Open
	filter := taskbus.QueryFilter{
		Status: &openStatus,
	}

	tasks, err := a.taskBus.Query(ctx, filter, taskbus.DefaultOrderBy, page.New(1, 200))
	if err != nil {
		return errs.Newf(errs.Internal, "query tasks: %s", err)
	}

	// Filter to only tasks that have no context assigned
	var unlinked []taskbus.Task
	for _, t := range tasks {
		if t.ContextID == nil {
			unlinked = append(unlinked, t)
		}
	}

	if len(unlinked) == 0 {
		return ClassifyResult{Classified: 0, ClarificationsCreated: 0}
	}

	// Get active contexts for matching
	activeStatus := contextbus.Active
	contexts, err := a.contextBus.Query(ctx, contextbus.QueryFilter{Status: &activeStatus}, contextbus.DefaultOrderBy, page.New(1, 50))
	if err != nil {
		return errs.Newf(errs.Internal, "query contexts: %s", err)
	}

	ctxRefs := make([]extractor.ContextRef, len(contexts))
	for i, c := range contexts {
		ctxRefs[i] = extractor.ContextRef{ID: c.ID.String(), Title: c.Title}
	}

	classified := 0
	clarCreated := 0

	for _, task := range unlinked {
		// Use the text extractor to classify the task against available contexts
		extraction, err := a.extractor.ExtractText(ctx, fmt.Sprintf("Task: %s\nDescription: %s", task.Title, task.Description), ctxRefs)
		if err != nil {
			continue
		}

		if extraction.SuggestedContextID == nil || *extraction.SuggestedContextID == "" {
			continue
		}

		ctxID, err := uuid.Parse(*extraction.SuggestedContextID)
		if err != nil {
			continue
		}

		// Verify the suggested context actually exists
		if _, err := a.contextBus.QueryByID(ctx, ctxID); err != nil {
			continue
		}

		if extraction.ContextConfidence >= 0.7 {
			// High confidence — assign directly
			ut := taskbus.UpdateTask{ContextID: &ctxID}
			if _, err := a.taskBus.Update(ctx, task, ut); err != nil {
				continue
			}
			classified++
		} else {
			// Low confidence — create a clarification card for user review
			optionsJSON, _ := json.Marshal(map[string]any{
				"type":               "context_assignment",
				"task_id":            task.ID.String(),
				"suggested_context":  ctxID.String(),
				"confidence":         extraction.ContextConfidence,
				"available_contexts": ctxRefs,
			})
			guess, _ := json.Marshal(map[string]string{
				"context_id": ctxID.String(),
			})
			guessRaw := json.RawMessage(guess)
			reasoning := fmt.Sprintf("AI matched '%s' to context with %.0f%% confidence", task.Title, extraction.ContextConfidence*100)

			if _, err := a.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
				Kind:          clarificationkind.ContextAssignment,
				SubjectType:   "task",
				SubjectID:     task.ID,
				Question:      fmt.Sprintf("Which context does '%s' belong to?", task.Title),
				ClaudeGuess:   &guessRaw,
				Reasoning:     &reasoning,
				AnswerOptions: json.RawMessage(optionsJSON),
			}); err == nil {
				clarCreated++
			}
		}
	}

	return ClassifyResult{Classified: classified, ClarificationsCreated: clarCreated}
}
