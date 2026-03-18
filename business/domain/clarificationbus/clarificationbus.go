package clarificationbus

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/clarificationstatus"
	"github.com/casebrophy/planner/foundation/logger"
)

type Storer interface {
	Create(ctx context.Context, item ClarificationItem) error
	Update(ctx context.Context, item ClarificationItem) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]ClarificationItem, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (ClarificationItem, error)
	UnsnoozeExpired(ctx context.Context, now time.Time) (int, error)
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

func (b *Business) Create(ctx context.Context, nc NewClarificationItem) (ClarificationItem, error) {
	now := time.Now()

	// Compute priority score: age_hours * 0.4 + kind_weight * 0.6
	kindWeight := clarificationkind.KindWeights[nc.Kind]
	score := float32(0.0)*0.4 + kindWeight*0.6

	status := clarificationstatus.Pending
	if nc.SnoozedUntil != nil {
		status = clarificationstatus.Snoozed
	}

	item := ClarificationItem{
		ID:            uuid.New(),
		Kind:          nc.Kind,
		Status:        status,
		SubjectType:   nc.SubjectType,
		SubjectID:     nc.SubjectID,
		Question:      nc.Question,
		ClaudeGuess:   nc.ClaudeGuess,
		Reasoning:     nc.Reasoning,
		AnswerOptions: nc.AnswerOptions,
		PriorityScore: score,
		SnoozedUntil:  nc.SnoozedUntil,
		CreatedAt:     now,
	}

	if err := b.storer.Create(ctx, item); err != nil {
		return ClarificationItem{}, fmt.Errorf("create: %w", err)
	}

	return item, nil
}

func (b *Business) Resolve(ctx context.Context, item ClarificationItem, rc ResolveClarificationItem) (ClarificationItem, error) {
	now := time.Now()

	item.Status = clarificationstatus.Resolved
	item.Answer = &rc.Answer
	item.ResolvedAt = &now

	if err := b.storer.Update(ctx, item); err != nil {
		return ClarificationItem{}, fmt.Errorf("resolve: %w", err)
	}

	return item, nil
}

func (b *Business) Snooze(ctx context.Context, item ClarificationItem, until time.Time) (ClarificationItem, error) {
	item.Status = clarificationstatus.Snoozed
	item.SnoozedUntil = &until

	if err := b.storer.Update(ctx, item); err != nil {
		return ClarificationItem{}, fmt.Errorf("snooze: %w", err)
	}

	return item, nil
}

func (b *Business) Dismiss(ctx context.Context, item ClarificationItem) (ClarificationItem, error) {
	now := time.Now()

	item.Status = clarificationstatus.Dismissed
	item.ResolvedAt = &now

	if err := b.storer.Update(ctx, item); err != nil {
		return ClarificationItem{}, fmt.Errorf("dismiss: %w", err)
	}

	return item, nil
}

func (b *Business) Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]ClarificationItem, error) {
	items, err := b.storer.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return items, nil
}

func (b *Business) QueryByID(ctx context.Context, id uuid.UUID) (ClarificationItem, error) {
	item, err := b.storer.QueryByID(ctx, id)
	if err != nil {
		return ClarificationItem{}, fmt.Errorf("query by id[%s]: %w", id, err)
	}
	return item, nil
}

func (b *Business) Count(ctx context.Context, filter QueryFilter) (int, error) {
	n, err := b.storer.Count(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	return n, nil
}

func (b *Business) UnsnoozeExpired(ctx context.Context) (int, error) {
	n, err := b.storer.UnsnoozeExpired(ctx, time.Now())
	if err != nil {
		return 0, fmt.Errorf("unsnooze expired: %w", err)
	}
	return n, nil
}

// RecalculatePriority recalculates the priority score for a clarification item.
// Score formula: age_hours * 0.4 + kind_weight * 0.6
func (b *Business) RecalculatePriority(item ClarificationItem) float32 {
	ageHours := float32(time.Since(item.CreatedAt).Hours())
	kindWeight := clarificationkind.KindWeights[item.Kind]
	return ageHours*0.4 + kindWeight*0.6
}
