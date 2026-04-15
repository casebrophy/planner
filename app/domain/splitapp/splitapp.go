package splitapp

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/splitbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	splitBus *splitbus.Business
}

func (a *app) create(ctx context.Context, r *http.Request) web.Encoder {
	var input NewSplit
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	txnID, err := uuid.Parse(input.TransactionID)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	ns := splitbus.NewSplit{
		TransactionID: txnID,
		PartyName:     input.PartyName,
		Amount:        input.Amount,
		VenmoHandle:   input.VenmoHandle,
	}

	s, err := a.splitBus.Create(ctx, ns)
	if err != nil {
		return errs.Newf(errs.Internal, "create: %s", err)
	}

	return toAppSplit(s)
}

func (a *app) queryByTransaction(ctx context.Context, r *http.Request) web.Encoder {
	txnID, err := uuid.Parse(web.Param(r, "transaction_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	pg, err := page.Parse(r.URL.Query().Get("page"), r.URL.Query().Get("rows"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	orderBy, err := parseOrder(r)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	filter := splitbus.QueryFilter{TransactionID: &txnID}

	splits, err := a.splitBus.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return errs.Newf(errs.Internal, "query: %s", err)
	}

	total, err := a.splitBus.Count(ctx, filter)
	if err != nil {
		return errs.Newf(errs.Internal, "count: %s", err)
	}

	return query.NewResult(toAppSplits(splits), total, pg.Number(), pg.RowsPerPage())
}

func (a *app) update(ctx context.Context, r *http.Request) web.Encoder {
	splitID, err := uuid.Parse(web.Param(r, "split_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	s, err := a.splitBus.QueryByID(ctx, splitID)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	var input UpdateSplit
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	us := splitbus.UpdateSplit{
		PartyName:   input.PartyName,
		Amount:      input.Amount,
		VenmoHandle: input.VenmoHandle,
		Settled:     input.Settled,
	}

	updated, err := a.splitBus.Update(ctx, s, us)
	if err != nil {
		return errs.Newf(errs.Internal, "update: %s", err)
	}

	return toAppSplit(updated)
}

func (a *app) delete(ctx context.Context, r *http.Request) web.Encoder {
	splitID, err := uuid.Parse(web.Param(r, "split_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	s, err := a.splitBus.QueryByID(ctx, splitID)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	if err := a.splitBus.Delete(ctx, s); err != nil {
		return errs.Newf(errs.Internal, "delete: %s", err)
	}

	return web.NoResponse{}
}
