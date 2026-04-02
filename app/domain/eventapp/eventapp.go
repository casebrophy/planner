package eventapp

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	eventBus *eventbus.Business
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
