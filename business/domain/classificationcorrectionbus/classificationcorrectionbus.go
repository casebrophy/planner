package classificationcorrectionbus

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/foundation/logger"
)

type Storer interface {
	Create(ctx context.Context, corr Correction) error
	CreateWithTx(ctx context.Context, tx sqlx.ExtContext, corr Correction) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Correction, error)
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

func (b *Business) Record(ctx context.Context, nc NewCorrection) (Correction, error) {
	now := time.Now()

	corr := Correction{
		ID:            uuid.New(),
		ClauseText:    nc.ClauseText,
		PredictedType: nc.PredictedType,
		Confidence:    nc.Confidence,
		ActualType:    nc.ActualType,
		Source:        nc.Source,
		CreatedAt:     now,
	}

	if err := b.storer.Create(ctx, corr); err != nil {
		return Correction{}, fmt.Errorf("create: %w", err)
	}

	return corr, nil
}

func (b *Business) RecordWithTx(ctx context.Context, tx sqlx.ExtContext, nc NewCorrection) (Correction, error) {
	now := time.Now()

	corr := Correction{
		ID:            uuid.New(),
		ClauseText:    nc.ClauseText,
		PredictedType: nc.PredictedType,
		Confidence:    nc.Confidence,
		ActualType:    nc.ActualType,
		Source:        nc.Source,
		CreatedAt:     now,
	}

	if err := b.storer.CreateWithTx(ctx, tx, corr); err != nil {
		return Correction{}, fmt.Errorf("create with tx: %w", err)
	}

	return corr, nil
}

func (b *Business) Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Correction, error) {
	corrs, err := b.storer.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return corrs, nil
}

func (b *Business) Count(ctx context.Context, filter QueryFilter) (int, error) {
	n, err := b.storer.Count(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	return n, nil
}

func (b *Business) QueryBySource(ctx context.Context, source string, pg page.Page) ([]Correction, error) {
	filter := QueryFilter{
		Source: &source,
	}

	corrs, err := b.storer.Query(ctx, filter, DefaultOrderBy, pg)
	if err != nil {
		return nil, fmt.Errorf("query by source: %w", err)
	}
	return corrs, nil
}
