package threadapp

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/foundation/sqldb"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	threadBus *threadbus.Business
}

func (a *app) addEntry(ctx context.Context, r *http.Request) web.Encoder {
	var input NewThreadEntry
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if input.SubjectType == "" {
		return errs.Newf(errs.InvalidArgument, "subjectType is required")
	}
	if input.SubjectID == "" {
		return errs.Newf(errs.InvalidArgument, "subjectId is required")
	}
	if input.Kind == "" {
		return errs.Newf(errs.InvalidArgument, "kind is required")
	}
	if input.Content == "" {
		return errs.Newf(errs.InvalidArgument, "content is required")
	}

	bne, err := toBusNewThreadEntry(input)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	entry, err := a.threadBus.AddEntry(ctx, bne)
	if err != nil {
		return errs.Newf(errs.Internal, "add entry: %s", err)
	}

	return toAppThreadEntry(entry)
}

func (a *app) queryThread(ctx context.Context, r *http.Request) web.Encoder {
	subjectType := web.Param(r, "subject_type")
	subjectID, err := uuid.Parse(web.Param(r, "subject_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	pg, err := page.Parse(r.URL.Query().Get("page"), r.URL.Query().Get("rows"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	entries, err := a.threadBus.QueryBySubject(ctx, subjectType, subjectID, pg)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query thread: %s", err)
	}

	total, err := a.threadBus.CountBySubject(ctx, subjectType, subjectID)
	if err != nil {
		return errs.Newf(errs.Internal, "count thread: %s", err)
	}

	return query.NewResult(toAppThreadEntries(entries), total, pg.Number(), pg.RowsPerPage())
}
