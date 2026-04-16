package embeddingdb

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/casebrophy/planner/business/domain/embeddingbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
)

func TestExistsBySource(t *testing.T) {
	t.Run("returns true for existing embedding", func(t *testing.T) {
		dbt := dbtest.New(t, "exists_true")
		store := NewStore(dbt.Log, dbt.DB)
		emb := embeddingbus.NewEmbedding{
			SourceType: "task",
			SourceID:   uuid.New(),
			Content:    "test content",
			Vector:     make([]float32, 1024),
		}
		_, err := store.Create(context.Background(), emb)
		require.NoError(t, err)

		exists, err := store.ExistsBySource(context.Background(), "task", emb.SourceID)
		require.NoError(t, err)
		require.True(t, exists)
	})

	t.Run("returns false for non-existing embedding", func(t *testing.T) {
		dbt := dbtest.New(t, "exists_false")
		store := NewStore(dbt.Log, dbt.DB)
		exists, err := store.ExistsBySource(context.Background(), "task", uuid.New())
		require.NoError(t, err)
		require.False(t, exists)
	})
}
