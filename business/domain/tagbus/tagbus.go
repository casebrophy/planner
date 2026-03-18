package tagbus

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/foundation/logger"
)

type Storer interface {
	Create(ctx context.Context, tag Tag) error
	Delete(ctx context.Context, tag Tag) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Tag, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	AddToTask(ctx context.Context, taskID, tagID uuid.UUID) error
	RemoveFromTask(ctx context.Context, taskID, tagID uuid.UUID) error
	AddToContext(ctx context.Context, contextID, tagID uuid.UUID) error
	RemoveFromContext(ctx context.Context, contextID, tagID uuid.UUID) error
	QueryByTask(ctx context.Context, taskID uuid.UUID) ([]Tag, error)
	QueryByContext(ctx context.Context, contextID uuid.UUID) ([]Tag, error)
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

func (b *Business) Create(ctx context.Context, nt NewTag) (Tag, error) {
	tag := Tag{
		ID:   uuid.New(),
		Name: nt.Name,
	}

	if err := b.storer.Create(ctx, tag); err != nil {
		return Tag{}, fmt.Errorf("create: %w", err)
	}

	return tag, nil
}

func (b *Business) Delete(ctx context.Context, tag Tag) error {
	if err := b.storer.Delete(ctx, tag); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

func (b *Business) Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Tag, error) {
	tags, err := b.storer.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return tags, nil
}

func (b *Business) Count(ctx context.Context, filter QueryFilter) (int, error) {
	n, err := b.storer.Count(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	return n, nil
}

func (b *Business) AddToTask(ctx context.Context, taskID, tagID uuid.UUID) error {
	if err := b.storer.AddToTask(ctx, taskID, tagID); err != nil {
		return fmt.Errorf("add to task: %w", err)
	}
	return nil
}

func (b *Business) RemoveFromTask(ctx context.Context, taskID, tagID uuid.UUID) error {
	if err := b.storer.RemoveFromTask(ctx, taskID, tagID); err != nil {
		return fmt.Errorf("remove from task: %w", err)
	}
	return nil
}

func (b *Business) AddToContext(ctx context.Context, contextID, tagID uuid.UUID) error {
	if err := b.storer.AddToContext(ctx, contextID, tagID); err != nil {
		return fmt.Errorf("add to context: %w", err)
	}
	return nil
}

func (b *Business) RemoveFromContext(ctx context.Context, contextID, tagID uuid.UUID) error {
	if err := b.storer.RemoveFromContext(ctx, contextID, tagID); err != nil {
		return fmt.Errorf("remove from context: %w", err)
	}
	return nil
}

func (b *Business) QueryByTask(ctx context.Context, taskID uuid.UUID) ([]Tag, error) {
	tags, err := b.storer.QueryByTask(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("query by task: %w", err)
	}
	return tags, nil
}

func (b *Business) QueryByContext(ctx context.Context, contextID uuid.UUID) ([]Tag, error) {
	tags, err := b.storer.QueryByContext(ctx, contextID)
	if err != nil {
		return nil, fmt.Errorf("query by context: %w", err)
	}
	return tags, nil
}
