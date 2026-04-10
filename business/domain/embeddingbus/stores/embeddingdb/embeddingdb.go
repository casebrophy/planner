package embeddingdb

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pgvector/pgvector-go"

	"github.com/casebrophy/planner/business/domain/embeddingbus"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/foundation/logger"
)

// Store implements embeddingbus.Storer using pgvector.
type Store struct {
	log *logger.Logger
	db  *sqlx.DB
}

// NewStore creates a new Store.
func NewStore(log *logger.Logger, db *sqlx.DB) *Store {
	return &Store{log: log, db: db}
}

// Create inserts a new embedding row.
func (s *Store) Create(ctx context.Context, emb embeddingbus.NewEmbedding) (embeddingbus.Embedding, error) {
	dbEmb := toDBEmbedding(emb)

	const q = `
	INSERT INTO embeddings
		(embedding_id, source_type, source_id, content, embedding, created_at, updated_at)
	VALUES
		(:embedding_id, :source_type, :source_id, :content, :embedding, :created_at, :updated_at)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, dbEmb); err != nil {
		return embeddingbus.Embedding{}, fmt.Errorf("namedexeccontext: %w", err)
	}

	return toBusEmbedding(dbEmb), nil
}

// SearchByVector finds the nearest embeddings by cosine similarity.
func (s *Store) SearchByVector(ctx context.Context, vec []float32, sourceTypes []string, limit int) ([]embeddingbus.SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	pgVec := pgvector.NewVector(vec)

	var rows []searchResultDB
	var err error

	if len(sourceTypes) > 0 {
		q := `
		SELECT embedding_id, source_type, source_id, content, embedding, created_at, updated_at,
		       1 - (embedding <=> $1) AS similarity
		FROM embeddings
		WHERE source_type = ANY($2)
		ORDER BY embedding <=> $1
		LIMIT $3`
		err = s.db.SelectContext(ctx, &rows, q, pgVec, sourceTypes, limit)
	} else {
		q := `
		SELECT embedding_id, source_type, source_id, content, embedding, created_at, updated_at,
		       1 - (embedding <=> $1) AS similarity
		FROM embeddings
		ORDER BY embedding <=> $1
		LIMIT $2`
		err = s.db.SelectContext(ctx, &rows, q, pgVec, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("selectcontext: %w", err)
	}

	results := make([]embeddingbus.SearchResult, len(rows))
	for i, r := range rows {
		results[i] = toBusSearchResult(r)
	}

	return results, nil
}

// DeleteBySource removes all embeddings for a given source entity.
func (s *Store) DeleteBySource(ctx context.Context, sourceType string, sourceID uuid.UUID) error {
	data := struct {
		SourceType string    `db:"source_type"`
		SourceID   uuid.UUID `db:"source_id"`
	}{
		SourceType: sourceType,
		SourceID:   sourceID,
	}

	const q = `DELETE FROM embeddings WHERE source_type = :source_type AND source_id = :source_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}
