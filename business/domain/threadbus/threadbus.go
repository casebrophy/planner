package threadbus

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
	Create(ctx context.Context, entry ThreadEntry) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]ThreadEntry, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (ThreadEntry, error)
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

func (b *Business) AddEntry(ctx context.Context, ne NewThreadEntry) (ThreadEntry, error) {
	now := time.Now()

	entry := ThreadEntry{
		ID:             uuid.New(),
		SubjectType:    ne.SubjectType,
		SubjectID:      ne.SubjectID,
		Kind:           ne.Kind,
		Content:        ne.Content,
		Metadata:       ne.Metadata,
		Source:         ne.Source,
		SourceID:       ne.SourceID,
		Sentiment:      ne.Sentiment,
		RequiresAction: ne.RequiresAction,
		CreatedAt:      now,
	}

	if err := b.storer.Create(ctx, entry); err != nil {
		return ThreadEntry{}, fmt.Errorf("create: %w", err)
	}

	return entry, nil
}

func (b *Business) QueryBySubject(ctx context.Context, subjectType string, subjectID uuid.UUID, pg page.Page) ([]ThreadEntry, error) {
	filter := QueryFilter{
		SubjectType: &subjectType,
		SubjectID:   &subjectID,
	}

	entries, err := b.storer.Query(ctx, filter, DefaultOrderBy, pg)
	if err != nil {
		return nil, fmt.Errorf("query by subject: %w", err)
	}
	return entries, nil
}

func (b *Business) CountBySubject(ctx context.Context, subjectType string, subjectID uuid.UUID) (int, error) {
	filter := QueryFilter{
		SubjectType: &subjectType,
		SubjectID:   &subjectID,
	}

	n, err := b.storer.Count(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("count by subject: %w", err)
	}
	return n, nil
}

func (b *Business) QueryByID(ctx context.Context, id uuid.UUID) (ThreadEntry, error) {
	entry, err := b.storer.QueryByID(ctx, id)
	if err != nil {
		return ThreadEntry{}, fmt.Errorf("query by id[%s]: %w", id, err)
	}
	return entry, nil
}

func (b *Business) Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]ThreadEntry, error) {
	entries, err := b.storer.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return entries, nil
}

func (b *Business) Count(ctx context.Context, filter QueryFilter) (int, error) {
	n, err := b.storer.Count(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	return n, nil
}
