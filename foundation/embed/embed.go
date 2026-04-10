package embed

import (
	"context"
)

// Embedder generates vector embeddings for text inputs.
type Embedder interface {
	// Embed returns embeddings for the given texts.
	Embed(ctx context.Context, texts []string) ([][]float32, error)

	// Dimensions returns the dimensionality of embeddings produced by this embedder.
	Dimensions() int
}
