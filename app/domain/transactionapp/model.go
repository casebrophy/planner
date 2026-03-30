package transactionapp

import (
	"encoding/json"
	"time"

	"github.com/casebrophy/planner/business/domain/transactionbus"
)

type Transaction struct {
	ID          string  `json:"id"`
	RawInputID  *string `json:"rawInputId,omitempty"`
	Source      string  `json:"source"`
	Date        string  `json:"date"`
	Description string  `json:"description"`
	CleanName   *string `json:"cleanName,omitempty"`
	Amount      int     `json:"amount"`
	Category    *string `json:"category,omitempty"`
	ContextID   *string `json:"contextId,omitempty"`
	Notes       *string `json:"notes,omitempty"`
	Reviewed    bool    `json:"reviewed"`
	CreatedAt   string  `json:"createdAt"`
}

func (t Transaction) Encode() ([]byte, string, error) {
	data, err := json.Marshal(t)
	return data, "application/json", err
}

func toAppTransaction(t transactionbus.Transaction) Transaction {
	at := Transaction{
		ID:          t.ID.String(),
		Source:      t.Source,
		Date:        t.Date.Format(time.RFC3339),
		Description: t.Description,
		CleanName:   t.CleanName,
		Amount:      t.Amount,
		Category:    t.Category,
		Notes:       t.Notes,
		Reviewed:    t.Reviewed,
		CreatedAt:   t.CreatedAt.Format(time.RFC3339),
	}

	if t.RawInputID != nil {
		s := t.RawInputID.String()
		at.RawInputID = &s
	}

	if t.ContextID != nil {
		s := t.ContextID.String()
		at.ContextID = &s
	}

	return at
}

func toAppTransactions(ts []transactionbus.Transaction) []Transaction {
	items := make([]Transaction, len(ts))
	for i, t := range ts {
		items[i] = toAppTransaction(t)
	}
	return items
}

type UpdateTransaction struct {
	CleanName *string `json:"cleanName"`
	Category  *string `json:"category"`
	ContextID *string `json:"contextId"`
	Notes     *string `json:"notes"`
	Reviewed  *bool   `json:"reviewed"`
}

type ImportResult struct {
	Total    int `json:"total"`
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
}

func (ir ImportResult) Encode() ([]byte, string, error) {
	data, err := json.Marshal(ir)
	return data, "application/json", err
}
