package observationapp

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/observationbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	observationBus *observationbus.Business
}

func (a *app) record(ctx context.Context, r *http.Request) web.Encoder {
	var input NewObservation
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
	if len(input.Data) == 0 {
		return errs.Newf(errs.InvalidArgument, "data is required")
	}

	bno, err := toBusNewObservation(input)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	obs, err := a.observationBus.Record(ctx, bno)
	if err != nil {
		return errs.Newf(errs.Internal, "record: %s", err)
	}

	return toAppObservation(obs)
}

func (a *app) queryBySubject(ctx context.Context, r *http.Request) web.Encoder {
	subjectType := web.Param(r, "subject_type")
	subjectID, err := uuid.Parse(web.Param(r, "subject_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	pg, err := page.Parse(r.URL.Query().Get("page"), r.URL.Query().Get("rows"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	obs, err := a.observationBus.QueryBySubject(ctx, subjectType, subjectID, pg)
	if err != nil {
		return errs.Newf(errs.Internal, "query by subject: %s", err)
	}

	filter := observationbus.QueryFilter{
		SubjectType: &subjectType,
		SubjectID:   &subjectID,
	}
	total, err := a.observationBus.Count(ctx, filter)
	if err != nil {
		return errs.Newf(errs.Internal, "count: %s", err)
	}

	return query.NewResult(toAppObservations(obs), total, pg.Number(), pg.RowsPerPage())
}
