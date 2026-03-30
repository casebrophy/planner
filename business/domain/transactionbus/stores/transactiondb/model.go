package transactiondb

import (
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/transactionbus"
)

type transactionDB struct {
	ID          uuid.UUID  `db:"transaction_id"`
	RawInputID  *uuid.UUID `db:"raw_input_id"`
	Source      string     `db:"source"`
	Date        time.Time  `db:"date"`
	Description string     `db:"description"`
	CleanName   *string    `db:"clean_name"`
	Amount      int        `db:"amount"`
	Category    *string    `db:"category"`
	ContextID   *uuid.UUID `db:"context_id"`
	Notes       *string    `db:"notes"`
	Reviewed    bool       `db:"reviewed"`
	CreatedAt   time.Time  `db:"created_at"`
}

func toDBTransaction(t transactionbus.Transaction) transactionDB {
	return transactionDB{
		ID:          t.ID,
		RawInputID:  t.RawInputID,
		Source:      t.Source,
		Date:        t.Date,
		Description: t.Description,
		CleanName:   t.CleanName,
		Amount:      t.Amount,
		Category:    t.Category,
		ContextID:   t.ContextID,
		Notes:       t.Notes,
		Reviewed:    t.Reviewed,
		CreatedAt:   t.CreatedAt,
	}
}

func toBusTransaction(t transactionDB) transactionbus.Transaction {
	return transactionbus.Transaction{
		ID:          t.ID,
		RawInputID:  t.RawInputID,
		Source:      t.Source,
		Date:        t.Date,
		Description: t.Description,
		CleanName:   t.CleanName,
		Amount:      t.Amount,
		Category:    t.Category,
		ContextID:   t.ContextID,
		Notes:       t.Notes,
		Reviewed:    t.Reviewed,
		CreatedAt:   t.CreatedAt,
	}
}

func toBusTransactions(ts []transactionDB) []transactionbus.Transaction {
	items := make([]transactionbus.Transaction, len(ts))
	for i, t := range ts {
		items[i] = toBusTransaction(t)
	}
	return items
}
