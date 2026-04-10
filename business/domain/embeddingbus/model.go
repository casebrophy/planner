package embeddingbus

import (
	"time"

	"github.com/google/uuid"
)

// Embedding represents a stored vector embedding.
type Embedding struct {
	ID         uuid.UUID
	SourceType string
	SourceID   uuid.UUID
	Content    string
	Vector     []float32
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// NewEmbedding contains the fields needed to create a new embedding.
type NewEmbedding struct {
	SourceType string
	SourceID   uuid.UUID
	Content    string
	Vector     []float32
}

// SearchResult is an embedding with its similarity score.
type SearchResult struct {
	Embedding
	Similarity float64
}
