package activitylogbus

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
	Create(ctx context.Context, log Log) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Log, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryStreaks(ctx context.Context, subjectType string, subjectID uuid.UUID) (StreakInfo, error)
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

func (b *Business) Create(ctx context.Context, nl NewLog) (Log, error) {
	log := Log{
		ID:          uuid.New(),
		SubjectType: nl.SubjectType,
		SubjectID:   nl.SubjectID,
		Value:       nl.Value,
		LoggedAt:    time.Now(),
	}

	if err := b.storer.Create(ctx, log); err != nil {
		return Log{}, fmt.Errorf("create: %w", err)
	}

	return log, nil
}

func (b *Business) Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Log, error) {
	logs, err := b.storer.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return logs, nil
}

func (b *Business) Count(ctx context.Context, filter QueryFilter) (int, error) {
	n, err := b.storer.Count(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	return n, nil
}

func (b *Business) QueryStreaks(ctx context.Context, subjectType string, subjectID uuid.UUID) (StreakInfo, error) {
	info, err := b.storer.QueryStreaks(ctx, subjectType, subjectID)
	if err != nil {
		return StreakInfo{}, fmt.Errorf("query streaks: %w", err)
	}
	return info, nil
}
