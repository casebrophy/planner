package embeddingbus

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/foundation/embed"
	"github.com/casebrophy/planner/foundation/logger"
)

// Storer defines the persistence interface for embeddings.
type Storer interface {
	Create(ctx context.Context, emb NewEmbedding) (Embedding, error)
	SearchByVector(ctx context.Context, vec []float32, sourceTypes []string, limit int) ([]SearchResult, error)
	DeleteBySource(ctx context.Context, sourceType string, sourceID uuid.UUID) error
}

// Business manages embedding operations.
type Business struct {
	log      *logger.Logger
	storer   Storer
	embedder embed.Embedder
}

// NewBusiness creates a new Business.
func NewBusiness(log *logger.Logger, storer Storer, embedder embed.Embedder) *Business {
	return &Business{
		log:      log,
		storer:   storer,
		embedder: embedder,
	}
}

// EmbedAndStore generates an embedding for the content and stores it.
func (b *Business) EmbedAndStore(ctx context.Context, sourceType string, sourceID uuid.UUID, content string) error {
	if b.embedder == nil {
		return nil
	}

	vecs, err := b.embedder.Embed(ctx, []string{content})
	if err != nil {
		return fmt.Errorf("embed: %w", err)
	}
	if len(vecs) == 0 {
		return fmt.Errorf("embedder returned no vectors")
	}

	_, err = b.storer.Create(ctx, NewEmbedding{
		SourceType: sourceType,
		SourceID:   sourceID,
		Content:    content,
		Vector:     vecs[0],
	})
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}

	return nil
}

// Search embeds the query text and returns semantically similar results.
func (b *Business) Search(ctx context.Context, query string, sourceTypes []string, limit int) ([]SearchResult, error) {
	if b.embedder == nil {
		return nil, fmt.Errorf("no embedder configured")
	}

	vecs, err := b.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	if len(vecs) == 0 {
		return nil, fmt.Errorf("embedder returned no vectors")
	}

	results, err := b.storer.SearchByVector(ctx, vecs[0], sourceTypes, limit)
	if err != nil {
		return nil, fmt.Errorf("searchbyvector: %w", err)
	}

	return results, nil
}
