package rawinputapp

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/ingestbus"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	rawInputBus *rawinputbus.Business
	ingestBus   *ingestbus.Business
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

	ris, err := a.rawInputBus.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return errs.Newf(errs.Internal, "query: %s", err)
	}

	total, err := a.rawInputBus.Count(ctx, filter)
	if err != nil {
		return errs.Newf(errs.Internal, "count: %s", err)
	}

	return query.NewResult(toAppRawInputs(ris), total, pg.Number(), pg.RowsPerPage())
}

func (a *app) queryByID(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "raw_input_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	ri, err := a.rawInputBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	return toAppRawInput(ri)
}

func (a *app) reprocess(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "raw_input_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	// Verify the raw input exists before running the pipeline.
	if _, err := a.rawInputBus.QueryByID(ctx, id); err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	if err := a.ingestBus.Reprocess(ctx, id); err != nil {
		return errs.Newf(errs.Internal, "reprocess: %s", err)
	}

	// Re-query to return the final state of the record.
	updated, err := a.rawInputBus.QueryByID(ctx, id)
	if err != nil {
		return errs.Newf(errs.Internal, "query updated: %s", err)
	}

	return toAppRawInput(updated)
}
