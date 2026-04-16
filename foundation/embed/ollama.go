package embed

import (
	"context"

	"github.com/casebrophy/planner/foundation/ollamaclient"
)

// OllamaEmbedder implements Embedder by delegating to a shared ollamaclient.Client.
type OllamaEmbedder struct {
	client *ollamaclient.Client
	model  string
	dims   int
}

// NewOllamaEmbedder wraps a shared Ollama client as an Embedder.
func NewOllamaEmbedder(client *ollamaclient.Client, model string, dims int) *OllamaEmbedder {
	return &OllamaEmbedder{client: client, model: model, dims: dims}
}

// Embed sends texts to the shared Ollama client and returns the embeddings.
func (e *OllamaEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return e.client.Embed(ctx, e.model, texts)
}

// Dimensions returns the dimensionality of embeddings produced by this embedder.
func (e *OllamaEmbedder) Dimensions() int { return e.dims }
