package splitbus

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/foundation/logger"
)

// Storer defines the interface for the split store.
type Storer interface {
	Create(ctx context.Context, s Split) error
	Update(ctx context.Context, s Split) error
	Delete(ctx context.Context, s Split) error
	DeleteByTransaction(ctx context.Context, transactionID uuid.UUID) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Split, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (Split, error)
}

// Business manages splits.
type Business struct {
	log    *logger.Logger
	storer Storer
}

// NewBusiness constructs a Business.
func NewBusiness(log *logger.Logger, storer Storer) *Business {
	return &Business{
		log:    log,
		storer: storer,
	}
}

// Create adds a new split.
func (b *Business) Create(ctx context.Context, ns NewSplit) (Split, error) {
	now := time.Now()

	s := Split{
		ID:            uuid.New(),
		TransactionID: ns.TransactionID,
		PartyName:     ns.PartyName,
		Amount:        ns.Amount,
		VenmoHandle:   ns.VenmoHandle,
		Settled:       false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := b.storer.Create(ctx, s); err != nil {
		return Split{}, fmt.Errorf("create: %w", err)
	}

	return s, nil
}

// Update modifies an existing split.
func (b *Business) Update(ctx context.Context, s Split, us UpdateSplit) (Split, error) {
	if us.PartyName != nil {
		s.PartyName = *us.PartyName
	}
	if us.Amount != nil {
		s.Amount = *us.Amount
	}
	if us.VenmoHandle != nil {
		s.VenmoHandle = us.VenmoHandle
	}
	if us.Settled != nil {
		s.Settled = *us.Settled
	}
	s.UpdatedAt = time.Now()

	if err := b.storer.Update(ctx, s); err != nil {
		return Split{}, fmt.Errorf("update: %w", err)
	}

	return s, nil
}

// Delete removes a split.
func (b *Business) Delete(ctx context.Context, s Split) error {
	if err := b.storer.Delete(ctx, s); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

// DeleteByTransaction removes all splits for a transaction.
func (b *Business) DeleteByTransaction(ctx context.Context, transactionID uuid.UUID) error {
	if err := b.storer.DeleteByTransaction(ctx, transactionID); err != nil {
		return fmt.Errorf("delete by transaction: %w", err)
	}
	return nil
}

// Query returns a filtered, ordered, paginated list of splits.
func (b *Business) Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Split, error) {
	splits, err := b.storer.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return splits, nil
}

// Count returns the number of splits matching the filter.
func (b *Business) Count(ctx context.Context, filter QueryFilter) (int, error) {
	n, err := b.storer.Count(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	return n, nil
}

// QueryByID returns the split with the given ID.
func (b *Business) QueryByID(ctx context.Context, id uuid.UUID) (Split, error) {
	s, err := b.storer.QueryByID(ctx, id)
	if err != nil {
		return Split{}, fmt.Errorf("query by id[%s]: %w", id, err)
	}
	return s, nil
}
