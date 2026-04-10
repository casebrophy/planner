package transactionapp

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/transactionbus"
	"github.com/casebrophy/planner/business/domain/transactionbus/csvparser"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	transactionBus *transactionbus.Business
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

	txns, err := a.transactionBus.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return errs.Newf(errs.Internal, "query: %s", err)
	}

	total, err := a.transactionBus.Count(ctx, filter)
	if err != nil {
		return errs.Newf(errs.Internal, "count: %s", err)
	}

	return query.NewResult(toAppTransactions(txns), total, pg.Number(), pg.RowsPerPage())
}

func (a *app) queryByID(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "transaction_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	txn, err := a.transactionBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	return toAppTransaction(txn)
}

func (a *app) update(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "transaction_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	txn, err := a.transactionBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	var input UpdateTransaction
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	but := transactionbus.UpdateTransaction{
		CleanName: input.CleanName,
		Category:  input.Category,
		Notes:     input.Notes,
		Reviewed:  input.Reviewed,
	}

	if input.ContextID != nil {
		cid, err := uuid.Parse(*input.ContextID)
		if err != nil {
			return errs.New(errs.InvalidArgument, err)
		}
		but.ContextID = &cid
	}

	updated, err := a.transactionBus.Update(ctx, txn, but)
	if err != nil {
		return errs.Newf(errs.Internal, "update: %s", err)
	}

	return toAppTransaction(updated)
}

func (a *app) delete(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "transaction_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	txn, err := a.transactionBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query by id: %s", err)
	}

	if err := a.transactionBus.Delete(ctx, txn); err != nil {
		return errs.Newf(errs.Internal, "delete: %s", err)
	}

	return web.NoResponse{}
}

func (a *app) importCSV(ctx context.Context, r *http.Request) web.Encoder {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return errs.Newf(errs.Internal, "read file: %s", err)
	}

	format := r.FormValue("format")

	rows, err := csvparser.Parse(string(data), format)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	nts := make([]transactionbus.NewTransaction, len(rows))
	for i, row := range rows {
		nts[i] = transactionbus.NewTransaction{
			Source:      row.Source,
			Date:        row.Date,
			Description: row.Description,
			Amount:      row.Amount,
		}
	}

	inserted, err := a.transactionBus.CreateBatch(ctx, nts)
	if err != nil {
		return errs.Newf(errs.Internal, "create batch: %s", err)
	}

	return ImportResult{
		Total:    len(rows),
		Imported: inserted,
		Skipped:  len(rows) - inserted,
	}
}

func (a *app) enrichmentStatus(ctx context.Context, r *http.Request) web.Encoder {
	return toAppEnrichmentStatus(a.transactionBus.EnrichmentStatus())
}
