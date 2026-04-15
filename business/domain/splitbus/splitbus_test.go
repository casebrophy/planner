package splitbus_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/splitbus"
	"github.com/casebrophy/planner/business/domain/splitbus/stores/splitdb"
	"github.com/casebrophy/planner/business/domain/transactionbus"
	"github.com/casebrophy/planner/business/domain/transactionbus/stores/transactiondb"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
)

func Test_Split(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Split")
	ctx := context.Background()

	// Wire up transaction bus to create prerequisite transactions.
	txnStore := transactiondb.NewStore(db.Log, db.DB)
	txnBus := transactionbus.NewBusiness(db.Log, txnStore, nil)

	// Create a transaction to attach splits to.
	txn, err := txnBus.Create(ctx, transactionbus.NewTransaction{
		Source:      "test",
		Date:        time.Now().UTC().Truncate(24 * time.Hour),
		Description: "Dinner at restaurant",
		Amount:      10000, // $100.00
	})
	if err != nil {
		t.Fatalf("creating transaction: %s", err)
	}

	// Wire up split bus.
	splitStore := splitdb.NewStore(db.Log, db.DB)
	splitBus := splitbus.NewBusiness(db.Log, splitStore)

	cmpOpts := cmp.Options{
		cmpopts.EquateApproxTime(time.Second),
	}

	t.Run("create", func(t *testing.T) {
		venmo := "@alice"
		s, err := splitBus.Create(ctx, splitbus.NewSplit{
			TransactionID: txn.ID,
			PartyName:     "Alice",
			Amount:        3300,
			VenmoHandle:   &venmo,
		})
		if err != nil {
			t.Fatalf("create split: %s", err)
		}
		if s.ID == uuid.Nil {
			t.Error("expected non-nil ID")
		}
		if s.TransactionID != txn.ID {
			t.Errorf("expected transaction ID %s, got %s", txn.ID, s.TransactionID)
		}
		if s.PartyName != "Alice" {
			t.Errorf("expected party name Alice, got %s", s.PartyName)
		}
		if s.Amount != 3300 {
			t.Errorf("expected amount 3300, got %d", s.Amount)
		}
		if s.Settled {
			t.Error("expected settled=false on create")
		}
	})

	t.Run("query_by_transaction", func(t *testing.T) {
		// Create a second split.
		_, err := splitBus.Create(ctx, splitbus.NewSplit{
			TransactionID: txn.ID,
			PartyName:     "Bob",
			Amount:        2500,
		})
		if err != nil {
			t.Fatalf("create second split: %s", err)
		}

		filter := splitbus.QueryFilter{TransactionID: &txn.ID}
		splits, err := splitBus.Query(ctx, filter, splitbus.DefaultOrderBy, page.New(1, 10))
		if err != nil {
			t.Fatalf("query splits: %s", err)
		}
		if len(splits) < 2 {
			t.Errorf("expected at least 2 splits, got %d", len(splits))
		}
	})

	t.Run("query_by_id", func(t *testing.T) {
		created, err := splitBus.Create(ctx, splitbus.NewSplit{
			TransactionID: txn.ID,
			PartyName:     "Charlie",
			Amount:        1500,
		})
		if err != nil {
			t.Fatalf("create split: %s", err)
		}

		got, err := splitBus.QueryByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("query by id: %s", err)
		}

		if diff := cmp.Diff(created, got, cmpOpts...); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("update", func(t *testing.T) {
		created, err := splitBus.Create(ctx, splitbus.NewSplit{
			TransactionID: txn.ID,
			PartyName:     "Dave",
			Amount:        2000,
		})
		if err != nil {
			t.Fatalf("create split: %s", err)
		}

		settled := true
		newName := "David"
		updated, err := splitBus.Update(ctx, created, splitbus.UpdateSplit{
			PartyName: &newName,
			Settled:   &settled,
		})
		if err != nil {
			t.Fatalf("update split: %s", err)
		}
		if updated.PartyName != "David" {
			t.Errorf("expected PartyName=David, got %s", updated.PartyName)
		}
		if !updated.Settled {
			t.Error("expected Settled=true after update")
		}
	})

	t.Run("delete", func(t *testing.T) {
		created, err := splitBus.Create(ctx, splitbus.NewSplit{
			TransactionID: txn.ID,
			PartyName:     "Eve",
			Amount:        1000,
		})
		if err != nil {
			t.Fatalf("create split: %s", err)
		}

		if err := splitBus.Delete(ctx, created); err != nil {
			t.Fatalf("delete split: %s", err)
		}

		_, err = splitBus.QueryByID(ctx, created.ID)
		if err == nil {
			t.Error("expected error querying deleted split, got nil")
		}
	})

	t.Run("count", func(t *testing.T) {
		// Create a separate transaction to avoid interference.
		txn2, err := txnBus.Create(ctx, transactionbus.NewTransaction{
			Source:      "test",
			Date:        time.Now().UTC().Truncate(24 * time.Hour),
			Description: "Lunch split count test",
			Amount:      5000,
		})
		if err != nil {
			t.Fatalf("creating transaction2: %s", err)
		}

		for i := 0; i < 3; i++ {
			_, err := splitBus.Create(ctx, splitbus.NewSplit{
				TransactionID: txn2.ID,
				PartyName:     "Person",
				Amount:        1000,
			})
			if err != nil {
				t.Fatalf("create split %d: %s", i, err)
			}
		}

		n, err := splitBus.Count(ctx, splitbus.QueryFilter{TransactionID: &txn2.ID})
		if err != nil {
			t.Fatalf("count: %s", err)
		}
		if n != 3 {
			t.Errorf("expected count=3, got %d", n)
		}
	})

	t.Run("delete_by_transaction", func(t *testing.T) {
		txn3, err := txnBus.Create(ctx, transactionbus.NewTransaction{
			Source:      "test",
			Date:        time.Now().UTC().Truncate(24 * time.Hour),
			Description: "Delete by txn test",
			Amount:      6000,
		})
		if err != nil {
			t.Fatalf("creating transaction3: %s", err)
		}

		for i := 0; i < 2; i++ {
			_, err := splitBus.Create(ctx, splitbus.NewSplit{
				TransactionID: txn3.ID,
				PartyName:     "Person",
				Amount:        1500,
			})
			if err != nil {
				t.Fatalf("create split %d: %s", i, err)
			}
		}

		if err := splitBus.DeleteByTransaction(ctx, txn3.ID); err != nil {
			t.Fatalf("delete by transaction: %s", err)
		}

		n, err := splitBus.Count(ctx, splitbus.QueryFilter{TransactionID: &txn3.ID})
		if err != nil {
			t.Fatalf("count after delete: %s", err)
		}
		if n != 0 {
			t.Errorf("expected count=0 after DeleteByTransaction, got %d", n)
		}
	})

	t.Run("order_by", func(t *testing.T) {
		ob := order.NewBy(splitbus.OrderByCreatedAt, order.ASC)
		filter := splitbus.QueryFilter{TransactionID: &txn.ID}
		splits, err := splitBus.Query(ctx, filter, ob, page.New(1, 100))
		if err != nil {
			t.Fatalf("query with order: %s", err)
		}
		for i := 1; i < len(splits); i++ {
			if splits[i].CreatedAt.Before(splits[i-1].CreatedAt) {
				t.Errorf("splits not in ascending order at index %d", i)
			}
		}
	})
}
