package notebus

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
	Create(ctx context.Context, note Note) error
	Update(ctx context.Context, note Note) error
	Delete(ctx context.Context, note Note) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Note, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (Note, error)
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

func (b *Business) Create(ctx context.Context, nn NewNote) (Note, error) {
	now := time.Now()

	source := nn.Source
	if source == "" {
		source = "manual"
	}

	note := Note{
		ID:         uuid.New(),
		ContextID:  nn.ContextID,
		TaskID:     nn.TaskID,
		Content:    nn.Content,
		Source:      source,
		RawInputID:  nn.RawInputID,
		Unconfirmed: nn.Unconfirmed,
		CreatedAt:   now,
		UpdatedAt:  now,
	}

	if err := b.storer.Create(ctx, note); err != nil {
		return Note{}, fmt.Errorf("create: %w", err)
	}

	return note, nil
}

func (b *Business) Update(ctx context.Context, note Note, un UpdateNote) (Note, error) {
	if un.ContextID != nil {
		note.ContextID = un.ContextID
	}
	if un.TaskID != nil {
		note.TaskID = un.TaskID
	}
	if un.Content != nil {
		note.Content = *un.Content
	}
	if un.Source != nil {
		note.Source = *un.Source
	}
	if un.Unconfirmed != nil {
		note.Unconfirmed = *un.Unconfirmed
	}

	note.UpdatedAt = time.Now()

	if err := b.storer.Update(ctx, note); err != nil {
		return Note{}, fmt.Errorf("update: %w", err)
	}

	return note, nil
}

func (b *Business) Delete(ctx context.Context, note Note) error {
	if err := b.storer.Delete(ctx, note); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

func (b *Business) Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Note, error) {
	notes, err := b.storer.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return notes, nil
}

func (b *Business) Count(ctx context.Context, filter QueryFilter) (int, error) {
	n, err := b.storer.Count(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	return n, nil
}

func (b *Business) QueryByID(ctx context.Context, id uuid.UUID) (Note, error) {
	note, err := b.storer.QueryByID(ctx, id)
	if err != nil {
		return Note{}, fmt.Errorf("query by id[%s]: %w", id, err)
	}
	return note, nil
}
