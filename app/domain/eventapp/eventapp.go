package eventapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/embeddingbus"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/foundation/web"
)

func truncateForDesc(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

type app struct {
	eventBus         *eventbus.Business
	contextBus       *contextbus.Business
	clarificationBus *clarificationbus.Business
	embeddingBus     *embeddingbus.Business
	extractor        extractor.Extractor
}

func (a *app) create(ctx context.Context, r *http.Request) web.Encoder {
	var input NewEvent
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if input.Title == "" {
		return errs.Newf(errs.InvalidArgument, "title is required")
	}

	if input.StartsAt == "" {
		return errs.Newf(errs.InvalidArgument, "startsAt is required")
	}

	if input.EndsAt == "" {
		return errs.Newf(errs.InvalidArgument, "endsAt is required")
	}

	bne, err := toBusNewEvent(input)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	event, err := a.eventBus.Create(ctx, bne)
	if err != nil {
		return errs.Newf(errs.Internal, "create: %s", err)
	}

	if event.ContextID == nil {
		go a.asyncClassify(context.Background(), "event", event.ID, fmt.Sprintf("Event: %s\nDescription: %s", event.Title, event.Description))
	}

	if a.embeddingBus != nil {
		go func(id uuid.UUID, title, desc string) {
			content := fmt.Sprintf("Event: %s\nDescription: %s", title, desc)
			a.embeddingBus.EmbedAndStore(context.Background(), "event", id, content)
		}(event.ID, event.Title, event.Description)
	}

	return toAppEvent(event)
}

func (a *app) update(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "event_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	event, err := a.eventBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	var input UpdateEvent
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	bue, err := toBusUpdateEvent(input)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	updated, err := a.eventBus.Update(ctx, event, bue)
	if err != nil {
		return errs.Newf(errs.Internal, "update: %s", err)
	}

	return toAppEvent(updated)
}

func (a *app) delete(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "event_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	event, err := a.eventBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	if err := a.eventBus.Delete(ctx, event); err != nil {
		return errs.Newf(errs.Internal, "delete: %s", err)
	}

	return web.NoResponse{}
}

func (a *app) queryAll(ctx context.Context, r *http.Request) web.Encoder {
	pg, err := page.Parse(r.URL.Query().Get("page"), r.URL.Query().Get("rows"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	filter, err := parseFilter(r)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	orderBy, err := parseOrder(r)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	events, err := a.eventBus.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return errs.Newf(errs.Internal, "query: %s", err)
	}

	total, err := a.eventBus.Count(ctx, filter)
	if err != nil {
		return errs.Newf(errs.Internal, "count: %s", err)
	}

	return query.NewResult(toAppEvents(events), total, pg.Number(), pg.RowsPerPage())
}

func (a *app) queryByID(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "event_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	event, err := a.eventBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	return toAppEvent(event)
}

func (a *app) asyncClassify(ctx context.Context, entityType string, entityID uuid.UUID, text string) {
	if a.extractor == nil {
		return
	}
	activeStatus := contextbus.Active
	contexts, err := a.contextBus.Query(ctx, contextbus.QueryFilter{Status: &activeStatus}, contextbus.DefaultOrderBy, page.New(1, 50))
	if err != nil {
		return
	}

	ctxRefs := make([]extractor.ContextRef, len(contexts))
	for i, c := range contexts {
		ctxRefs[i] = extractor.ContextRef{ID: c.ID.String(), Title: c.Title}
	}

	extraction, err := a.extractor.ExtractText(ctx, text, "", ctxRefs, "")
	if err != nil {
		return
	}

	if extraction.SuggestedContextID == nil || *extraction.SuggestedContextID == "" {
		return
	}

	ctxID, err := uuid.Parse(*extraction.SuggestedContextID)
	if err != nil {
		return
	}

	if _, err := a.contextBus.QueryByID(ctx, ctxID); err != nil {
		return
	}

	if extraction.ContextConfidence >= 0.7 {
		event, err := a.eventBus.QueryByID(ctx, entityID)
		if err != nil {
			return
		}
		a.eventBus.Update(ctx, event, eventbus.UpdateEvent{ContextID: &ctxID}) //nolint:errcheck
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
			Kind:               clarificationkind.ContextAssignment,
			SubjectType:        entityType,
			SubjectID:          entityID,
			SubjectDescription: fmt.Sprintf("Event: %s", truncateForDesc(text, 120)),
			Question:           fmt.Sprintf("Which context does this %s belong to?", entityType),
			ClaudeGuess:        &guessRaw,
			Reasoning:          &reasoning,
			AnswerOptions:      json.RawMessage(optJSON),
		})
	}
}
