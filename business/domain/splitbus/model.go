package splitbus

import (
	"time"

	"github.com/google/uuid"
)

type Split struct {
	ID            uuid.UUID
	TransactionID uuid.UUID
	PartyName     string
	Amount        int // cents
	VenmoHandle   *string
	Settled       bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type NewSplit struct {
	TransactionID uuid.UUID
	PartyName     string
	Amount        int
	VenmoHandle   *string
}

type UpdateSplit struct {
	PartyName   *string
	Amount      *int
	VenmoHandle *string
	Settled     *bool
}
