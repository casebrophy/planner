package eventbus

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
	Create(ctx context.Context, event Event) error
	Update(ctx context.Context, event Event) error
	Delete(ctx context.Context, event Event) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Event, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (Event, error)
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

func (b *Business) Create(ctx context.Context, ne NewEvent) (Event, error) {
	now := time.Now()

	event := Event{
		ID:          uuid.New(),
		ContextID:   ne.ContextID,
		Title:       ne.Title,
		Description: ne.Description,
		Location:    ne.Location,
		StartsAt:    ne.StartsAt,
		EndsAt:      ne.EndsAt,
		AllDay:      ne.AllDay,
		RawInputID:  ne.RawInputID,
		Unconfirmed: ne.Unconfirmed,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := b.storer.Create(ctx, event); err != nil {
		return Event{}, fmt.Errorf("create: %w", err)
	}

	return event, nil
}

func (b *Business) Update(ctx context.Context, event Event, ue UpdateEvent) (Event, error) {
	if ue.ContextID != nil {
		event.ContextID = ue.ContextID
	}
	if ue.Title != nil {
		event.Title = *ue.Title
	}
	if ue.Description != nil {
		event.Description = *ue.Description
	}
	if ue.Location != nil {
		event.Location = ue.Location
	}
	if ue.StartsAt != nil {
		event.StartsAt = *ue.StartsAt
	}
	if ue.EndsAt != nil {
		event.EndsAt = *ue.EndsAt
	}
	if ue.AllDay != nil {
		event.AllDay = *ue.AllDay
	}
	if ue.Unconfirmed != nil {
		event.Unconfirmed = *ue.Unconfirmed
	}

	event.UpdatedAt = time.Now()

	if err := b.storer.Update(ctx, event); err != nil {
		return Event{}, fmt.Errorf("update: %w", err)
	}

	return event, nil
}

func (b *Business) Delete(ctx context.Context, event Event) error {
	if err := b.storer.Delete(ctx, event); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

func (b *Business) Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Event, error) {
	events, err := b.storer.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return events, nil
}

func (b *Business) Count(ctx context.Context, filter QueryFilter) (int, error) {
	n, err := b.storer.Count(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	return n, nil
}

func (b *Business) QueryByID(ctx context.Context, id uuid.UUID) (Event, error) {
	event, err := b.storer.QueryByID(ctx, id)
	if err != nil {
		return Event{}, fmt.Errorf("query by id[%s]: %w", id, err)
	}
	return event, nil
}
