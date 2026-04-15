package splitapi_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/domain/splitapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/splitbus"
	"github.com/casebrophy/planner/business/domain/splitbus/stores/splitdb"
	"github.com/casebrophy/planner/business/domain/transactionbus"
	"github.com/casebrophy/planner/business/domain/transactionbus/stores/transactiondb"
)

type seedData struct {
	txnID  uuid.UUID
	splits []splitbus.Split
}

func Test_Split(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "Test_Split")

	// Create a transaction to attach splits to.
	ctx := context.Background()
	txnStore := transactiondb.NewStore(test.DB.Log, test.DB.DB)
	txnBus := transactionbus.NewBusiness(test.DB.Log, txnStore, nil)
	txn, err := txnBus.Create(ctx, transactionbus.NewTransaction{
		Source:      "test",
		Date:        time.Now().UTC().Truncate(24 * time.Hour),
		Description: "Dinner test",
		Amount:      10000,
	})
	if err != nil {
		t.Fatalf("creating transaction: %s", err)
	}

	// Create a split directly via store for query test.
	splitStore := splitdb.NewStore(test.DB.Log, test.DB.DB)
	splitBus := splitbus.NewBusiness(test.DB.Log, splitStore)
	venmo := "@alice"
	s, err := splitBus.Create(ctx, splitbus.NewSplit{
		TransactionID: txn.ID,
		PartyName:     "Alice",
		Amount:        3300,
		VenmoHandle:   &venmo,
	})
	if err != nil {
		t.Fatalf("seeding split: %s", err)
	}

	sd := seedData{txnID: txn.ID, splits: []splitbus.Split{s}}

	test.Run(t, queryByTransaction200(sd), "query-by-transaction-200")
	test.Run(t, queryByTransaction401(sd), "query-by-transaction-401")
	test.Run(t, create201(txn.ID), "create-201")
	test.Run(t, create401(txn.ID), "create-401")
}

func queryByTransaction200(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "ok",
			URL:        "/api/v1/transactions/" + sd.txnID.String() + "/splits?page=1&rows=10",
			Token:      apitest.TestAPIKey,
			Method:     http.MethodGet,
			StatusCode: http.StatusOK,
			GotResp:    &query.Result[splitapp.Split]{},
			ExpResp: &query.Result[splitapp.Split]{
				Total:       1,
				Page:        1,
				RowsPerPage: 10,
				Items:       toAppSplits(sd.splits),
			},
			CmpFunc: func(got any, exp any) string {
				gotResp := got.(*query.Result[splitapp.Split])
				expResp := exp.(*query.Result[splitapp.Split])
				clearTimes := func(items []splitapp.Split) {
					for i := range items {
						items[i].CreatedAt = ""
						items[i].UpdatedAt = ""
					}
				}
				clearTimes(gotResp.Items)
				clearTimes(expResp.Items)
				return cmp.Diff(gotResp, expResp, cmpopts.SortSlices(func(a, b splitapp.Split) bool {
					return a.ID < b.ID
				}))
			},
		},
	}
}

func queryByTransaction401(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "no-key",
			URL:        "/api/v1/transactions/" + sd.txnID.String() + "/splits?page=1&rows=10",
			Token:      "",
			Method:     http.MethodGet,
			StatusCode: http.StatusUnauthorized,
			GotResp:    &errs.Error{},
			ExpResp:    &errs.Error{Code: errs.Unauthenticated, Message: "missing api key"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}

func create201(txnID uuid.UUID) []apitest.Table {
	return []apitest.Table{
		{
			Name:   "ok",
			URL:    "/api/v1/splits",
			Token:  apitest.TestAPIKey,
			Method: http.MethodPost,
			Input: splitapp.NewSplit{
				TransactionID: txnID.String(),
				PartyName:     "Bob",
				Amount:        5000,
			},
			StatusCode: http.StatusOK,
			GotResp:    &splitapp.Split{},
			ExpResp: &splitapp.Split{
				TransactionID: txnID.String(),
				PartyName:     "Bob",
				Amount:        5000,
				Settled:       false,
			},
			CmpFunc: func(got any, exp any) string {
				gotResp := got.(*splitapp.Split)
				expResp := exp.(*splitapp.Split)
				gotResp.ID = ""
				gotResp.CreatedAt = ""
				gotResp.UpdatedAt = ""
				expResp.ID = ""
				return cmp.Diff(gotResp, expResp)
			},
		},
	}
}

func create401(txnID uuid.UUID) []apitest.Table {
	return []apitest.Table{
		{
			Name:   "no-key",
			URL:    "/api/v1/splits",
			Token:  "",
			Method: http.MethodPost,
			Input: splitapp.NewSplit{
				TransactionID: txnID.String(),
				PartyName:     "Bob",
				Amount:        5000,
			},
			StatusCode: http.StatusUnauthorized,
			GotResp:    &errs.Error{},
			ExpResp:    &errs.Error{Code: errs.Unauthenticated, Message: "missing api key"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}

func toAppSplits(ss []splitbus.Split) []splitapp.Split {
	items := make([]splitapp.Split, len(ss))
	for i, s := range ss {
		items[i] = splitapp.Split{
			ID:            s.ID.String(),
			TransactionID: s.TransactionID.String(),
			PartyName:     s.PartyName,
			Amount:        s.Amount,
			VenmoHandle:   s.VenmoHandle,
			Settled:       s.Settled,
		}
	}
	return items
}
