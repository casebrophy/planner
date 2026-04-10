package embeddingdb

import (
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"

	"github.com/casebrophy/planner/business/domain/embeddingbus"
)

type embeddingDB struct {
	ID         uuid.UUID      `db:"embedding_id"`
	SourceType string         `db:"source_type"`
	SourceID   uuid.UUID      `db:"source_id"`
	Content    string         `db:"content"`
	Embedding  pgvector.Vector `db:"embedding"`
	CreatedAt  time.Time      `db:"created_at"`
	UpdatedAt  time.Time      `db:"updated_at"`
}

type searchResultDB struct {
	embeddingDB
	Similarity float64 `db:"similarity"`
}

func toDBEmbedding(e embeddingbus.NewEmbedding) embeddingDB {
	return embeddingDB{
		ID:         uuid.New(),
		SourceType: e.SourceType,
		SourceID:   e.SourceID,
		Content:    e.Content,
		Embedding:  pgvector.NewVector(e.Vector),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func toBusEmbedding(e embeddingDB) embeddingbus.Embedding {
	return embeddingbus.Embedding{
		ID:         e.ID,
		SourceType: e.SourceType,
		SourceID:   e.SourceID,
		Content:    e.Content,
		Vector:     e.Embedding.Slice(),
		CreatedAt:  e.CreatedAt,
		UpdatedAt:  e.UpdatedAt,
	}
}

func toBusSearchResult(r searchResultDB) embeddingbus.SearchResult {
	return embeddingbus.SearchResult{
		Embedding:  toBusEmbedding(r.embeddingDB),
		Similarity: r.Similarity,
	}
}
