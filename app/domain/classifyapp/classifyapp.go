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

	activeStatus := contextbus.Active
	contexts, err := a.contextBus.Query(ctx, contextbus.QueryFilter{Status: &activeStatus}, contextbus.DefaultOrderBy, page.New(1, 50))
	if err != nil {
		return errs.Newf(errs.Internal, "query contexts: %s", err)
	}

	ctxRefs := make([]extractor.ContextRef, len(contexts))
	for i, c := range contexts {
		ctxRefs[i] = extractor.ContextRef{ID: c.ID.String(), Title: c.Title}
	}

	n := len(unlinked)

	// Spawn goroutine for LLM calls — return immediately to avoid gateway timeout
	go func() {
		bgCtx := context.Background()
		for _, task := range unlinked {
			extraction, err := a.extractor.ExtractText(bgCtx, fmt.Sprintf("Task: %s\nDescription: %s", task.Title, task.Description), ctxRefs)
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

			if _, err := a.contextBus.QueryByID(bgCtx, ctxID); err != nil {
				continue
			}

			if extraction.ContextConfidence >= 0.7 {
				ut := taskbus.UpdateTask{ContextID: &ctxID}
				if _, err := a.taskBus.Update(bgCtx, task, ut); err != nil {
					continue
				}
			} else {
				busCtxRefs := make([]clarificationbus.ContextRef, len(ctxRefs))
				for i, r := range ctxRefs {
					busCtxRefs[i] = clarificationbus.ContextRef{ID: r.ID, Title: r.Title}
				}
				optionsJSON, _ := json.Marshal(clarificationbus.ContextAssignmentOptions{
					SuggestedContext:  ctxID.String(),
					Confidence:        extraction.ContextConfidence,
					AvailableContexts: busCtxRefs,
				})
				guess, _ := json.Marshal(map[string]string{"context_id": ctxID.String()})
				guessRaw := json.RawMessage(guess)
				reasoning := fmt.Sprintf("AI matched '%s' to context with %.0f%% confidence", task.Title, extraction.ContextConfidence*100)

				a.clarificationBus.Create(bgCtx, clarificationbus.NewClarificationItem{ //nolint:errcheck
					Kind:          clarificationkind.ContextAssignment,
					SubjectType:   "task",
					SubjectID:     task.ID,
					Question:      fmt.Sprintf("Which context does '%s' belong to?", task.Title),
					ClaudeGuess:   &guessRaw,
					Reasoning:     &reasoning,
					AnswerOptions: json.RawMessage(optionsJSON),
				})
			}
		}
	}()

	return ClassifyAccepted{
		Message:       fmt.Sprintf("Classification started for %d tasks", n),
		UnlinkedCount: n,
	}
}
