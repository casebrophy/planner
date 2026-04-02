package voiceingestapp

import (
	"context"
	"net/http"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/business/domain/ingestbus"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	ingestBus *ingestbus.Business
}

func (a *app) ingest(ctx context.Context, r *http.Request) web.Encoder {
	var req ingestRequest
	if err := web.Decode(r, &req); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if req.Text == "" {
		return errs.Newf(errs.InvalidArgument, "text is required")
	}

	result, err := a.ingestBus.ProcessText(ctx, req.Text)
	if err != nil {
		return errs.Newf(errs.Internal, "process text: %s", err)
	}

	// Convert UUIDs to strings
	taskIDStrs := make([]string, len(result.TaskIDs))
	for i, id := range result.TaskIDs {
		taskIDStrs[i] = id.String()
	}

	eventIDStrs := make([]string, len(result.EventIDs))
	for i, id := range result.EventIDs {
		eventIDStrs[i] = id.String()
	}

	return ingestResponse{
		TaskIDs:  taskIDStrs,
		EventIDs: eventIDStrs,
	}
}
