package noteapp

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	noteBus *notebus.Business
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

	note, err := a.noteBus.Create(ctx, bnn)
	if err != nil {
		return errs.Newf(errs.Internal, "create: %s", err)
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
