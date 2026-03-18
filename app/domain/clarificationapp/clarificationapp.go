package clarificationapp

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/clarificationstatus"
	"github.com/casebrophy/planner/foundation/sqldb"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	clarificationBus *clarificationbus.Business
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

	// TODO: Resolution dispatcher side-effects (Phase 3b step 8)
	// Maps kind + answer → side-effect (e.g. update context_id, create task, etc.)

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
