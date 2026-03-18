package observationbus

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/observationkind"
	"github.com/casebrophy/planner/foundation/logger"
)

type Storer interface {
	Create(ctx context.Context, obs Observation) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Observation, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
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

func (b *Business) Record(ctx context.Context, no NewObservation) (Observation, error) {
	now := time.Now()

	obs := Observation{
		ID:          uuid.New(),
		SubjectType: no.SubjectType,
		SubjectID:   no.SubjectID,
		Kind:        no.Kind,
		Data:        no.Data,
		Source:      no.Source,
		Confidence:  no.Confidence,
		Weight:      no.Weight,
		CreatedAt:   now,
	}

	if err := b.storer.Create(ctx, obs); err != nil {
		return Observation{}, fmt.Errorf("create: %w", err)
	}

	return obs, nil
}

func (b *Business) QueryBySubject(ctx context.Context, subjectType string, subjectID uuid.UUID, pg page.Page) ([]Observation, error) {
	filter := QueryFilter{
		SubjectType: &subjectType,
		SubjectID:   &subjectID,
	}

	obs, err := b.storer.Query(ctx, filter, DefaultOrderBy, pg)
	if err != nil {
		return nil, fmt.Errorf("query by subject: %w", err)
	}
	return obs, nil
}

func (b *Business) QueryByKind(ctx context.Context, kind observationkind.Kind, pg page.Page) ([]Observation, error) {
	filter := QueryFilter{
		Kind: &kind,
	}

	obs, err := b.storer.Query(ctx, filter, DefaultOrderBy, pg)
	if err != nil {
		return nil, fmt.Errorf("query by kind: %w", err)
	}
	return obs, nil
}

func (b *Business) Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Observation, error) {
	obs, err := b.storer.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return obs, nil
}

func (b *Business) Count(ctx context.Context, filter QueryFilter) (int, error) {
	n, err := b.storer.Count(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	return n, nil
}
