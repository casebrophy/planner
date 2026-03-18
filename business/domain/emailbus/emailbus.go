package emailbus

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
	Create(ctx context.Context, e Email) error
	Update(ctx context.Context, e Email) error
	Delete(ctx context.Context, e Email) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Email, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (Email, error)
	QueryByMessageID(ctx context.Context, messageID string) (Email, error)
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

func (b *Business) Create(ctx context.Context, ne NewEmail) (Email, error) {
	now := time.Now()

	e := Email{
		ID:          uuid.New(),
		RawInputID:  ne.RawInputID,
		MessageID:   ne.MessageID,
		FromAddress: ne.FromAddress,
		FromName:    ne.FromName,
		ToAddress:   ne.ToAddress,
		Subject:     ne.Subject,
		BodyText:    ne.BodyText,
		BodyHTML:    ne.BodyHTML,
		ReceivedAt:  ne.ReceivedAt,
		ContextID:   ne.ContextID,
		CreatedAt:   now,
	}

	if err := b.storer.Create(ctx, e); err != nil {
		return Email{}, fmt.Errorf("create: %w", err)
	}

	return e, nil
}

func (b *Business) Update(ctx context.Context, e Email, ue UpdateEmail) (Email, error) {
	if ue.ContextID != nil {
		e.ContextID = ue.ContextID
	}

	if err := b.storer.Update(ctx, e); err != nil {
		return Email{}, fmt.Errorf("update: %w", err)
	}

	return e, nil
}

func (b *Business) Delete(ctx context.Context, e Email) error {
	if err := b.storer.Delete(ctx, e); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

func (b *Business) Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Email, error) {
	emails, err := b.storer.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return emails, nil
}

func (b *Business) Count(ctx context.Context, filter QueryFilter) (int, error) {
	n, err := b.storer.Count(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	return n, nil
}

func (b *Business) QueryByID(ctx context.Context, id uuid.UUID) (Email, error) {
	e, err := b.storer.QueryByID(ctx, id)
	if err != nil {
		return Email{}, fmt.Errorf("query by id[%s]: %w", id, err)
	}
	return e, nil
}

func (b *Business) QueryByMessageID(ctx context.Context, messageID string) (Email, error) {
	e, err := b.storer.QueryByMessageID(ctx, messageID)
	if err != nil {
		return Email{}, fmt.Errorf("query by message id[%s]: %w", messageID, err)
	}
	return e, nil
}
