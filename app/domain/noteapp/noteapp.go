package noteapp

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
	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	noteBus          *notebus.Business
	contextBus       *contextbus.Business
	clarificationBus *clarificationbus.Business
	extractor        extractor.Extractor
}

func (a *app) create(ctx context.Context, r *http.Request) web.Encoder {
	var input NewNote
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if input.Content == "" {
		return errs.Newf(errs.InvalidArgument, "content is required")
	}

	bnn, err := toBusNewNote(input)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if bnn.ContextID == nil && bnn.TaskID == nil {
		return errs.Newf(errs.InvalidArgument, "one of contextId or taskId is required")
	}

	note, err := a.noteBus.Create(ctx, bnn)
	if err != nil {
		return errs.Newf(errs.Internal, "create: %s", err)
	}

	if note.ContextID == nil && note.TaskID == nil {
		go a.asyncClassify(context.Background(), "note", note.ID, fmt.Sprintf("Note: %s", note.Content))
	}

	return toAppNote(note)
}

func (a *app) update(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "note_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	note, err := a.noteBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	var input UpdateNote
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	bun, err := toBusUpdateNote(input)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	updated, err := a.noteBus.Update(ctx, note, bun)
	if err != nil {
		return errs.Newf(errs.Internal, "update: %s", err)
	}

	return toAppNote(updated)
}

func (a *app) delete(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "note_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	note, err := a.noteBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	if err := a.noteBus.Delete(ctx, note); err != nil {
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

	notes, err := a.noteBus.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return errs.Newf(errs.Internal, "query: %s", err)
	}

	total, err := a.noteBus.Count(ctx, filter)
	if err != nil {
		return errs.Newf(errs.Internal, "count: %s", err)
	}

	return query.NewResult(toAppNotes(notes), total, pg.Number(), pg.RowsPerPage())
}

func (a *app) queryByID(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "note_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	note, err := a.noteBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	return toAppNote(note)
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

	extraction, err := a.extractor.ExtractText(ctx, text, ctxRefs)
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
		note, err := a.noteBus.QueryByID(ctx, entityID)
		if err != nil {
			return
		}
		a.noteBus.Update(ctx, note, notebus.UpdateNote{ContextID: &ctxID}) //nolint:errcheck
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
