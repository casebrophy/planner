package timeblockbus

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/foundation/logger"
)

type Storer interface {
	Create(ctx context.Context, block TimeBlock) error
	Update(ctx context.Context, block TimeBlock) error
	Delete(ctx context.Context, block TimeBlock) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]TimeBlock, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (TimeBlock, error)
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

func (b *Business) Create(ctx context.Context, ntb NewTimeBlock) (TimeBlock, error) {
	now := time.Now()

	block := TimeBlock{
		ID:        uuid.New(),
		TaskID:    ntb.TaskID,
		StartsAt:  ntb.StartsAt,
		EndsAt:    ntb.EndsAt,
		Confirmed: false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := b.storer.Create(ctx, block); err != nil {
		return TimeBlock{}, fmt.Errorf("create: %w", err)
	}

	return block, nil
}

func (b *Business) Update(ctx context.Context, block TimeBlock, utb UpdateTimeBlock) (TimeBlock, error) {
	if utb.StartsAt != nil {
		block.StartsAt = *utb.StartsAt
	}
	if utb.EndsAt != nil {
		block.EndsAt = *utb.EndsAt
	}
	if utb.Confirmed != nil {
		block.Confirmed = *utb.Confirmed
	}

	block.UpdatedAt = time.Now()

	if err := b.storer.Update(ctx, block); err != nil {
		return TimeBlock{}, fmt.Errorf("update: %w", err)
	}

	return block, nil
}

func (b *Business) Delete(ctx context.Context, block TimeBlock) error {
	if err := b.storer.Delete(ctx, block); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

func (b *Business) Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]TimeBlock, error) {
	blocks, err := b.storer.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return blocks, nil
}

func (b *Business) Count(ctx context.Context, filter QueryFilter) (int, error) {
	n, err := b.storer.Count(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	return n, nil
}

func (b *Business) QueryByID(ctx context.Context, id uuid.UUID) (TimeBlock, error) {
	block, err := b.storer.QueryByID(ctx, id)
	if err != nil {
		return TimeBlock{}, fmt.Errorf("query by id[%s]: %w", id, err)
	}
	return block, nil
}
