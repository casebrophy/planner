package activitylogapp

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/activitylogbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	logBus *activitylogbus.Business
}

func (a *app) create(ctx context.Context, r *http.Request) web.Encoder {
	var input NewActivityLog
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if input.SubjectType == "" {
		return errs.Newf(errs.InvalidArgument, "subjectType is required")
	}

	if input.SubjectID == "" {
		return errs.Newf(errs.InvalidArgument, "subjectId is required")
	}

	subjectID, err := uuid.Parse(input.SubjectID)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	nl := activitylogbus.NewLog{
		SubjectType: input.SubjectType,
		SubjectID:   subjectID,
		Value:       input.Value,
	}

	l, err := a.logBus.Create(ctx, nl)
	if err != nil {
		return errs.Newf(errs.Internal, "create: %s", err)
	}

	return toAppLog(l)
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

	logs, err := a.logBus.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return errs.Newf(errs.Internal, "query: %s", err)
	}

	total, err := a.logBus.Count(ctx, filter)
	if err != nil {
		return errs.Newf(errs.Internal, "count: %s", err)
	}

	return query.NewResult(toAppLogs(logs), total, pg.Number(), pg.RowsPerPage())
}

func (a *app) queryBulk(ctx context.Context, r *http.Request) web.Encoder {
	subjectType := r.URL.Query().Get("subject_type")
	if subjectType == "" {
		return errs.New(errs.InvalidArgument, fmt.Errorf("subject_type is required"))
	}

	idsParam := r.URL.Query().Get("subject_ids")
	if idsParam == "" {
		return errs.New(errs.InvalidArgument, fmt.Errorf("subject_ids is required"))
	}

	var subjectIDs []uuid.UUID
	for _, raw := range strings.Split(idsParam, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		id, err := uuid.Parse(raw)
		if err != nil {
			return errs.New(errs.InvalidArgument, fmt.Errorf("invalid subject_id %q: %w", raw, err))
		}
		subjectIDs = append(subjectIDs, id)
	}

	if len(subjectIDs) == 0 {
		return errs.New(errs.InvalidArgument, fmt.Errorf("subject_ids must contain at least one valid UUID"))
	}

	fromStr := r.URL.Query().Get("from")
	if fromStr == "" {
		return errs.New(errs.InvalidArgument, fmt.Errorf("from is required"))
	}
	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		return errs.New(errs.InvalidArgument, fmt.Errorf("invalid from date: %w", err))
	}

	toStr := r.URL.Query().Get("to")
	if toStr == "" {
		return errs.New(errs.InvalidArgument, fmt.Errorf("to is required"))
	}
	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		return errs.New(errs.InvalidArgument, fmt.Errorf("invalid to date: %w", err))
	}

	filter := activitylogbus.QueryBySubjectsFilter{
		SubjectType: subjectType,
		SubjectIDs:  subjectIDs,
		From:        from,
		To:          to,
	}

	logs, err := a.logBus.QueryBySubjects(ctx, filter)
	if err != nil {
		return errs.New(errs.Internal, err)
	}

	result := make(map[string][]ActivityLog)
	for _, l := range logs {
		key := l.SubjectID.String()
		result[key] = append(result[key], toAppLog(l))
	}

	return BulkLogsResponse{Items: result}
}

func (a *app) streaks(ctx context.Context, r *http.Request) web.Encoder {
	subjectType := web.Param(r, "subject_type")
	subjectID, err := uuid.Parse(web.Param(r, "subject_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	info, err := a.logBus.QueryStreaks(ctx, subjectType, subjectID)
	if err != nil {
		return errs.Newf(errs.Internal, "streaks: %s", err)
	}

	return toAppStreaks(info)
}
