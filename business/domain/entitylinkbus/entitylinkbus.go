package entitylinkbus

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/foundation/logger"
)

// Storer defines the persistence interface for entity links.
type Storer interface {
	Create(ctx context.Context, link EntityLink) error
	Delete(ctx context.Context, id uuid.UUID) error
	QueryByID(ctx context.Context, id uuid.UUID) (EntityLink, error)
	QueryBySource(ctx context.Context, sourceType string, sourceID uuid.UUID) ([]EntityLink, error)
	QueryByTarget(ctx context.Context, targetType string, targetID uuid.UUID) ([]EntityLink, error)
}

// Business manages entity link operations.
type Business struct {
	log    *logger.Logger
	storer Storer
}

// NewBusiness constructs an entitylinkbus.Business.
func NewBusiness(log *logger.Logger, storer Storer) *Business {
	return &Business{log: log, storer: storer}
}

// Create persists a new entity link.
func (b *Business) Create(ctx context.Context, nl NewEntityLink) (EntityLink, error) {
	if nl.Kind == "" {
		nl.Kind = "manual"
	}
	if nl.Confidence == 0 && nl.Kind == "manual" {
		nl.Confidence = 1.0
	}

	link := EntityLink{
		ID:         uuid.New(),
		SourceType: nl.SourceType,
		SourceID:   nl.SourceID,
		TargetType: nl.TargetType,
		TargetID:   nl.TargetID,
		Confidence: nl.Confidence,
		Kind:       nl.Kind,
		CreatedAt:  time.Now(),
	}

	if err := b.storer.Create(ctx, link); err != nil {
		return EntityLink{}, fmt.Errorf("create: %w", err)
	}

	return link, nil
}

// QueryByID retrieves an entity link by its ID.
func (b *Business) QueryByID(ctx context.Context, id uuid.UUID) (EntityLink, error) {
	link, err := b.storer.QueryByID(ctx, id)
	if err != nil {
		return EntityLink{}, fmt.Errorf("query by id: %w", err)
	}
	return link, nil
}

// Delete removes an entity link by ID.
func (b *Business) Delete(ctx context.Context, id uuid.UUID) error {
	if err := b.storer.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

// QueryByEntity returns all links where the entity appears as source or target.
func (b *Business) QueryByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]EntityLink, error) {
	bySource, err := b.storer.QueryBySource(ctx, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("query by source: %w", err)
	}

	byTarget, err := b.storer.QueryByTarget(ctx, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("query by target: %w", err)
	}

	return append(bySource, byTarget...), nil
}
