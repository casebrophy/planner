package contextapp

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/foundation/sqldb"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	contextBus *contextbus.Business
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
