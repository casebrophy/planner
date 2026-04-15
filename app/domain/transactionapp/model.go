package transactionapp

import (
	"encoding/json"
	"time"

	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
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

type EnrichmentStatus struct {
	Active  int64 `json:"active"`
	Pending int64 `json:"pending"`
	Done    int64 `json:"done"`
	Failed  int64 `json:"failed"`
	Enabled bool  `json:"enabled"`
}

func (es EnrichmentStatus) Encode() ([]byte, string, error) {
	data, err := json.Marshal(es)
	return data, "application/json", err
}

func toAppEnrichmentStatus(s transactionbus.EnrichmentStatus) EnrichmentStatus {
	return EnrichmentStatus{
		Active:  s.Active,
		Pending: s.Pending,
		Done:    s.Done,
		Failed:  s.Failed,
		Enabled: s.Enabled,
	}
}

type ReceiptExtractionRequest struct {
	OCRText string `json:"ocrText"`
}

type AppReceiptExtraction struct {
	Merchant string                `json:"merchant"`
	Date     string                `json:"date"`
	Total    int                   `json:"total"`
	Tax      int                   `json:"tax"`
	Subtotal int                   `json:"subtotal"`
	Items    []AppReceiptLineItem  `json:"items"`
	Notes    string                `json:"notes,omitempty"`
}

type AppReceiptLineItem struct {
	Description string `json:"description"`
	Amount      int    `json:"amount"`
	Quantity    int    `json:"quantity"`
}

func (are AppReceiptExtraction) Encode() ([]byte, string, error) {
	data, err := json.Marshal(are)
	return data, "application/json", err
}

func toAppReceiptExtraction(re extractor.ReceiptExtraction) AppReceiptExtraction {
	items := make([]AppReceiptLineItem, len(re.Items))
	for i, item := range re.Items {
		items[i] = AppReceiptLineItem{
			Description: item.Description,
			Amount:      item.Amount,
			Quantity:    item.Quantity,
		}
	}
	return AppReceiptExtraction{
		Merchant: re.Merchant,
		Date:     re.Date,
		Total:    re.Total,
		Tax:      re.Tax,
		Subtotal: re.Subtotal,
		Items:    items,
		Notes:    re.Notes,
	}
}
