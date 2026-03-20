package contextapp

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
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/debriefstatus"
	"github.com/casebrophy/planner/foundation/sqldb"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	contextBus       *contextbus.Business
	clarificationBus *clarificationbus.Business
}

func (a *app) create(ctx context.Context, r *http.Request) web.Encoder {
	var input NewContext
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if input.Title == "" {
		return errs.Newf(errs.InvalidArgument, "title is required")
	}

	bc := toBusNewContext(input)

	c, err := a.contextBus.Create(ctx, bc)
	if err != nil {
		return errs.Newf(errs.Internal, "create: %s", err)
	}

	return toAppContext(c)
}

func (a *app) update(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "context_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	c, err := a.contextBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	previousStatus := c.Status

	var input UpdateContext
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	buc, err := toBusUpdateContext(input)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	updated, err := a.contextBus.Update(ctx, c, buc)
	if err != nil {
		return errs.Newf(errs.Internal, "update: %s", err)
	}

	// If status transitioned to closed, trigger debrief flow
	if previousStatus != contextbus.Closed && updated.Status == contextbus.Closed {
		a.triggerDebriefFlow(ctx, updated)
	}

	return toAppContext(updated)
}

func (a *app) delete(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "context_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	c, err := a.contextBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	if err := a.contextBus.Delete(ctx, c); err != nil {
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

	contexts, err := a.contextBus.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return errs.Newf(errs.Internal, "query: %s", err)
	}

	total, err := a.contextBus.Count(ctx, filter)
	if err != nil {
		return errs.Newf(errs.Internal, "count: %s", err)
	}

	return query.NewResult(toAppContexts(contexts), total, pg.Number(), pg.RowsPerPage())
}

func (a *app) queryByID(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "context_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	c, err := a.contextBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	return toAppContext(c)
}

func (a *app) addEvent(ctx context.Context, r *http.Request) web.Encoder {
	contextID, err := uuid.Parse(web.Param(r, "context_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	var input NewEvent
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if input.Kind == "" {
		return errs.Newf(errs.InvalidArgument, "kind is required")
	}
	if input.Content == "" {
		return errs.Newf(errs.InvalidArgument, "content is required")
	}

	bne, err := toBusNewEvent(input, contextID)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	event, err := a.contextBus.AddEvent(ctx, bne)
	if err != nil {
		return errs.Newf(errs.Internal, "add event: %s", err)
	}

	return toAppEvent(event)
}

func (a *app) queryEvents(ctx context.Context, r *http.Request) web.Encoder {
	contextID, err := uuid.Parse(web.Param(r, "context_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	pg, err := page.Parse(r.URL.Query().Get("page"), r.URL.Query().Get("rows"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	events, err := a.contextBus.QueryEvents(ctx, contextID, pg)
	if err != nil {
		return errs.Newf(errs.Internal, "query events: %s", err)
	}

	total, err := a.contextBus.CountEvents(ctx, contextID)
	if err != nil {
		return errs.Newf(errs.Internal, "count events: %s", err)
	}

	return query.NewResult(toAppEvents(events), total, pg.Number(), pg.RowsPerPage())
}

// triggerDebriefFlow sets debrief_status to pending and creates 3 pre-snoozed
// context_debrief clarification cards (snoozed 24h).
func (a *app) triggerDebriefFlow(ctx context.Context, c contextbus.Context) {
	// Set debrief_status to pending
	pending := debriefstatus.Pending
	if _, err := a.contextBus.Update(ctx, c, contextbus.UpdateContext{
		DebriefStatus: &pending,
	}); err != nil {
		// Log but don't fail the update response
		return
	}

	snoozedUntil := time.Now().Add(24 * time.Hour)

	debriefQuestions := []string{
		fmt.Sprintf("Context '%s' is now closed. What went well?", c.Title),
		fmt.Sprintf("Context '%s' is now closed. What could have gone better?", c.Title),
		fmt.Sprintf("Context '%s' is now closed. Any lessons learned or patterns to note?", c.Title),
	}

	for _, question := range debriefQuestions {
		optionsJSON, _ := json.Marshal(map[string]any{
			"type":       "context_debrief",
			"context_id": c.ID.String(),
		})

		until := snoozedUntil
		if _, err := a.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
			Kind:          clarificationkind.ContextDebrief,
			SubjectType:   "context",
			SubjectID:     c.ID,
			Question:      question,
			AnswerOptions: json.RawMessage(optionsJSON),
			SnoozedUntil:  &until,
		}); err != nil {
			// Log but continue creating remaining cards
			continue
		}
	}
}
