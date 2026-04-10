package transactionbus

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
	ID          uuid.UUID
	RawInputID  *uuid.UUID
	Source      string
	Date        time.Time
	Description string
	CleanName   *string
	Amount      int
	Category    *string
	ContextID   *uuid.UUID
	Notes       *string
	Reviewed    bool
	CreatedAt   time.Time
}

type NewTransaction struct {
	RawInputID  *uuid.UUID
	Source      string
	Date        time.Time
	Description string
	CleanName   *string
	Amount      int
	Category    *string
	ContextID   *uuid.UUID
	Notes       *string
}

type UpdateTransaction struct {
	CleanName *string
	Category  *string
	ContextID *uuid.UUID
	Notes     *string
	Reviewed  *bool
}

// Enricher enriches a transaction with AI-extracted metadata.
type Enricher interface {
	EnrichTransaction(ctx context.Context, txn Transaction) (TransactionEnrichment, error)
}

// TransactionEnrichment holds the AI-generated metadata for a transaction.
type TransactionEnrichment struct {
	CleanName          string
	Category           string
	SuggestedContextID *uuid.UUID
	ContextConfidence  float64
}
