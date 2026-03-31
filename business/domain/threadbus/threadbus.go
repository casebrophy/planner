package threadbus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/threadentrykind"
	"github.com/casebrophy/planner/foundation/logger"
)

type Storer interface {
	Create(ctx context.Context, entry ThreadEntry) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]ThreadEntry, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (ThreadEntry, error)
}

type Business struct {
	log       *logger.Logger
	storer    Storer
	extractor Extractor // nil = no extraction
}

func NewBusiness(log *logger.Logger, storer Storer) *Business {
	return &Business{
		log:    log,
		storer: storer,
	}
}

// WithExtractor sets an optional AI extractor for thread entry classification.
func (b *Business) WithExtractor(ext Extractor) {
	b.extractor = ext
}

func (b *Business) AddEntry(ctx context.Context, ne NewThreadEntry) (ThreadEntry, error) {
	now := time.Now()

	kind := ne.Kind
	sentiment := ne.Sentiment
	requiresAction := ne.RequiresAction
	metadata := ne.Metadata

	// Run AI extraction if requested and extractor is available
	if ne.Extract && b.extractor != nil {
		result, err := b.extractor.ExtractThreadEntry(ctx, ne.Content, ne.SubjectType)
		if err != nil {
			b.log.Error(ctx, "threadbus", "msg", "extraction failed, using defaults", "error", err)
		} else if result.Confidence >= 0.6 {
			if parsed, err := threadentrykind.Parse(result.Kind); err == nil {
				kind = parsed
			}
			if result.Sentiment != nil {
				sentiment = result.Sentiment
			}
			requiresAction = result.RequiresAction

			metaJSON, _ := json.Marshal(result)
			raw := json.RawMessage(metaJSON)
			metadata = &raw
		}
	}

	entry := ThreadEntry{
		ID:             uuid.New(),
		SubjectType:    ne.SubjectType,
		SubjectID:      ne.SubjectID,
		Kind:           kind,
		Content:        ne.Content,
		Metadata:       metadata,
		Source:         ne.Source,
		SourceID:       ne.SourceID,
		Sentiment:      sentiment,
		RequiresAction: requiresAction,
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
