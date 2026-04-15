package splitapp

import (
	"encoding/json"
	"time"

	"github.com/casebrophy/planner/business/domain/splitbus"
)

type Split struct {
	ID            string  `json:"id"`
	TransactionID string  `json:"transactionId"`
	PartyName     string  `json:"partyName"`
	Amount        int     `json:"amount"`
	VenmoHandle   *string `json:"venmoHandle,omitempty"`
	Settled       bool    `json:"settled"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
}

func (s Split) Encode() ([]byte, string, error) {
	data, err := json.Marshal(s)
	return data, "application/json", err
}

func toAppSplit(s splitbus.Split) Split {
	return Split{
		ID:            s.ID.String(),
		TransactionID: s.TransactionID.String(),
		PartyName:     s.PartyName,
		Amount:        s.Amount,
		VenmoHandle:   s.VenmoHandle,
		Settled:       s.Settled,
		CreatedAt:     s.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     s.UpdatedAt.Format(time.RFC3339),
	}
}

func toAppSplits(ss []splitbus.Split) []Split {
	items := make([]Split, len(ss))
	for i, s := range ss {
		items[i] = toAppSplit(s)
	}
	return items
}

type NewSplit struct {
	TransactionID string  `json:"transactionId"`
	PartyName     string  `json:"partyName"`
	Amount        int     `json:"amount"`
	VenmoHandle   *string `json:"venmoHandle,omitempty"`
}

type UpdateSplit struct {
	PartyName   *string `json:"partyName,omitempty"`
	Amount      *int    `json:"amount,omitempty"`
	VenmoHandle *string `json:"venmoHandle,omitempty"`
	Settled     *bool   `json:"settled,omitempty"`
}
