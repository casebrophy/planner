package timeblockapp

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/timeblockbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	timeBlockBus *timeblockbus.Business
}

func (a *app) create(ctx context.Context, r *http.Request) web.Encoder {
	var input NewTimeBlock
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if input.TaskID == "" {
		return errs.Newf(errs.InvalidArgument, "taskId is required")
	}

	if input.StartsAt == "" {
		return errs.Newf(errs.InvalidArgument, "startsAt is required")
	}

	if input.EndsAt == "" {
		return errs.Newf(errs.InvalidArgument, "endsAt is required")
	}

	bntb, err := toBusNewTimeBlock(input)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	block, err := a.timeBlockBus.Create(ctx, bntb)
	if err != nil {
		return errs.Newf(errs.Internal, "create: %s", err)
	}

	return toAppTimeBlock(block)
}

func (a *app) update(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "block_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	block, err := a.timeBlockBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	var input UpdateTimeBlock
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	butb, err := toBusUpdateTimeBlock(input)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	updated, err := a.timeBlockBus.Update(ctx, block, butb)
	if err != nil {
		return errs.Newf(errs.Internal, "update: %s", err)
	}

	return toAppTimeBlock(updated)
}

func (a *app) delete(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "block_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	block, err := a.timeBlockBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	if err := a.timeBlockBus.Delete(ctx, block); err != nil {
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

	blocks, err := a.timeBlockBus.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return errs.Newf(errs.Internal, "query: %s", err)
	}

	total, err := a.timeBlockBus.Count(ctx, filter)
	if err != nil {
		return errs.Newf(errs.Internal, "count: %s", err)
	}

	return query.NewResult(toAppTimeBlocks(blocks), total, pg.Number(), pg.RowsPerPage())
}

func (a *app) queryByID(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "block_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	block, err := a.timeBlockBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	return toAppTimeBlock(block)
}
