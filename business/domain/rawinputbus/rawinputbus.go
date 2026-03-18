package rawinputbus

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/rawinputstatus"
	"github.com/casebrophy/planner/foundation/logger"
)

type Storer interface {
	Create(ctx context.Context, ri RawInput) error
	Update(ctx context.Context, ri RawInput) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]RawInput, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (RawInput, error)
}

type Business struct {
	log    *logger.Logger
	storer Storer
}

func NewBusiness(log *logger.Logger, storer Storer) *Business {
	return &Business{
		log:    log,
		storer: storer,
	}
}

func (b *Business) Create(ctx context.Context, nri NewRawInput) (RawInput, error) {
	now := time.Now()

	ri := RawInput{
		ID:         uuid.New(),
		SourceType: nri.SourceType,
		Status:     rawinputstatus.Pending,
		RawContent: nri.RawContent,
		CreatedAt:  now,
	}

	if err := b.storer.Create(ctx, ri); err != nil {
		return RawInput{}, fmt.Errorf("create: %w", err)
	}

	return ri, nil
}

func (b *Business) Update(ctx context.Context, ri RawInput, uri UpdateRawInput) (RawInput, error) {
	if uri.Status != nil {
		ri.Status = *uri.Status
	}
	if uri.ProcessedAt != nil {
		ri.ProcessedAt = uri.ProcessedAt
	}
	if uri.Error != nil {
		ri.Error = uri.Error
	}

	if err := b.storer.Update(ctx, ri); err != nil {
		return RawInput{}, fmt.Errorf("update: %w", err)
	}

	return ri, nil
}

func (b *Business) MarkProcessing(ctx context.Context, ri RawInput) (RawInput, error) {
	s := rawinputstatus.Processing
	return b.Update(ctx, ri, UpdateRawInput{Status: &s})
}

func (b *Business) MarkProcessed(ctx context.Context, ri RawInput) (RawInput, error) {
	s := rawinputstatus.Processed
	now := time.Now()
	return b.Update(ctx, ri, UpdateRawInput{Status: &s, ProcessedAt: &now})
}

func (b *Business) MarkFailed(ctx context.Context, ri RawInput, errMsg string) (RawInput, error) {
	s := rawinputstatus.Failed
	return b.Update(ctx, ri, UpdateRawInput{Status: &s, Error: &errMsg})
}

func (b *Business) Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]RawInput, error) {
	ris, err := b.storer.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return ris, nil
}

func (b *Business) Count(ctx context.Context, filter QueryFilter) (int, error) {
	n, err := b.storer.Count(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	return n, nil
}

func (b *Business) QueryByID(ctx context.Context, id uuid.UUID) (RawInput, error) {
	ri, err := b.storer.QueryByID(ctx, id)
	if err != nil {
		return RawInput{}, fmt.Errorf("query by id[%s]: %w", id, err)
	}
	return ri, nil
}
