package tagapp

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/tagbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	tagBus *tagbus.Business
}

func (a *app) create(ctx context.Context, r *http.Request) web.Encoder {
	var input NewTag
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if input.Name == "" {
		return errs.Newf(errs.InvalidArgument, "name is required")
	}

	bt := toBusNewTag(input)

	tag, err := a.tagBus.Create(ctx, bt)
	if err != nil {
		return errs.Newf(errs.Internal, "create: %s", err)
	}

	return toAppTag(tag)
}

func (a *app) delete(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "tag_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	tag := tagbus.Tag{ID: id}

	if err := a.tagBus.Delete(ctx, tag); err != nil {
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

	tags, err := a.tagBus.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return errs.Newf(errs.Internal, "query: %s", err)
	}

	total, err := a.tagBus.Count(ctx, filter)
	if err != nil {
		return errs.Newf(errs.Internal, "count: %s", err)
	}

	return query.NewResult(toAppTags(tags), total, pg.Number(), pg.RowsPerPage())
}

func (a *app) addToTask(ctx context.Context, r *http.Request) web.Encoder {
	taskID, err := uuid.Parse(web.Param(r, "task_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	tagID, err := uuid.Parse(web.Param(r, "tag_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if err := a.tagBus.AddToTask(ctx, taskID, tagID); err != nil {
		return errs.Newf(errs.Internal, "add to task: %s", err)
	}

	return web.NoResponse{}
}

func (a *app) removeFromTask(ctx context.Context, r *http.Request) web.Encoder {
	taskID, err := uuid.Parse(web.Param(r, "task_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	tagID, err := uuid.Parse(web.Param(r, "tag_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if err := a.tagBus.RemoveFromTask(ctx, taskID, tagID); err != nil {
		return errs.Newf(errs.Internal, "remove from task: %s", err)
	}

	return web.NoResponse{}
}

func (a *app) addToContext(ctx context.Context, r *http.Request) web.Encoder {
	contextID, err := uuid.Parse(web.Param(r, "context_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	tagID, err := uuid.Parse(web.Param(r, "tag_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if err := a.tagBus.AddToContext(ctx, contextID, tagID); err != nil {
		return errs.Newf(errs.Internal, "add to context: %s", err)
	}

	return web.NoResponse{}
}

func (a *app) removeFromContext(ctx context.Context, r *http.Request) web.Encoder {
	contextID, err := uuid.Parse(web.Param(r, "context_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	tagID, err := uuid.Parse(web.Param(r, "tag_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if err := a.tagBus.RemoveFromContext(ctx, contextID, tagID); err != nil {
		return errs.Newf(errs.Internal, "remove from context: %s", err)
	}

	return web.NoResponse{}
}

func (a *app) queryByTask(ctx context.Context, r *http.Request) web.Encoder {
	taskID, err := uuid.Parse(web.Param(r, "task_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	tags, err := a.tagBus.QueryByTask(ctx, taskID)
	if err != nil {
		return errs.Newf(errs.Internal, "query by task: %s", err)
	}

	return query.NewResult(toAppTags(tags), len(tags), 1, len(tags))
}

func (a *app) queryByContext(ctx context.Context, r *http.Request) web.Encoder {
	contextID, err := uuid.Parse(web.Param(r, "context_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	tags, err := a.tagBus.QueryByContext(ctx, contextID)
	if err != nil {
		return errs.Newf(errs.Internal, "query by context: %s", err)
	}

	return query.NewResult(toAppTags(tags), len(tags), 1, len(tags))
}
