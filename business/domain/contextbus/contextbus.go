package contextbus

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/debriefstatus"
	"github.com/casebrophy/planner/foundation/logger"
)

type Storer interface {
	Create(ctx context.Context, c Context) error
	Update(ctx context.Context, c Context) error
	Delete(ctx context.Context, c Context) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Context, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (Context, error)
	CreateEvent(ctx context.Context, e Event) error
	QueryEvents(ctx context.Context, contextID uuid.UUID, pg page.Page) ([]Event, error)
	CountEvents(ctx context.Context, contextID uuid.UUID) (int, error)
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

func (b *Business) Create(ctx context.Context, nc NewContext) (Context, error) {
	now := time.Now()

	c := Context{
		ID:            uuid.New(),
		Title:         nc.Title,
		Description:   nc.Description,
		Status:        Active,
		DebriefStatus: debriefstatus.Pending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := b.storer.Create(ctx, c); err != nil {
		return Context{}, fmt.Errorf("create: %w", err)
	}

	return c, nil
}

func (b *Business) Update(ctx context.Context, c Context, uc UpdateContext) (Context, error) {
	if uc.Title != nil {
		c.Title = *uc.Title
	}
	if uc.Description != nil {
		c.Description = *uc.Description
	}
	if uc.Status != nil {
		c.Status = *uc.Status
	}
	if uc.Summary != nil {
		c.Summary = *uc.Summary
	}
	if uc.DebriefStatus != nil {
		c.DebriefStatus = *uc.DebriefStatus
	}
	if uc.Outcome != nil {
		c.Outcome = uc.Outcome
	}

	c.UpdatedAt = time.Now()

	if err := b.storer.Update(ctx, c); err != nil {
		return Context{}, fmt.Errorf("update: %w", err)
	}

	return c, nil
}

func (b *Business) Delete(ctx context.Context, c Context) error {
	if err := b.storer.Delete(ctx, c); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

func (b *Business) Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Context, error) {
	cs, err := b.storer.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return cs, nil
}

func (b *Business) Count(ctx context.Context, filter QueryFilter) (int, error) {
	n, err := b.storer.Count(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	return n, nil
}

func (b *Business) QueryByID(ctx context.Context, id uuid.UUID) (Context, error) {
	c, err := b.storer.QueryByID(ctx, id)
	if err != nil {
		return Context{}, fmt.Errorf("query by id[%s]: %w", id, err)
	}
	return c, nil
}

func (b *Business) AddEvent(ctx context.Context, ne NewEvent) (Event, error) {
	now := time.Now()

	e := Event{
		ID:        uuid.New(),
		ContextID: ne.ContextID,
		Kind:      ne.Kind,
		Content:   ne.Content,
		Metadata:  ne.Metadata,
		SourceID:  ne.SourceID,
		CreatedAt: now,
	}

	if err := b.storer.CreateEvent(ctx, e); err != nil {
		return Event{}, fmt.Errorf("create event: %w", err)
	}

	// Update last_event on the context
	c, err := b.storer.QueryByID(ctx, ne.ContextID)
	if err != nil {
		return Event{}, fmt.Errorf("query context for event update: %w", err)
	}

	c.LastEvent = &now
	c.UpdatedAt = now
	if err := b.storer.Update(ctx, c); err != nil {
		return Event{}, fmt.Errorf("update context last_event: %w", err)
	}

	return e, nil
}

func (b *Business) QueryEvents(ctx context.Context, contextID uuid.UUID, pg page.Page) ([]Event, error) {
	events, err := b.storer.QueryEvents(ctx, contextID, pg)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	return events, nil
}

func (b *Business) CountEvents(ctx context.Context, contextID uuid.UUID) (int, error) {
	n, err := b.storer.CountEvents(ctx, contextID)
	if err != nil {
		return 0, fmt.Errorf("count events: %w", err)
	}
	return n, nil
}
