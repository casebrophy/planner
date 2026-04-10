package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ollamaEmbedRequest is the request body for the Ollama /api/embed endpoint.
type ollamaEmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// ollamaEmbedResponse is the response body from the Ollama /api/embed endpoint.
type ollamaEmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// OllamaEmbedder implements Embedder using a local Ollama instance.
type OllamaEmbedder struct {
	url    string
	model  string
	client *http.Client
	dims   int
}

// NewOllamaEmbedder creates an embedder backed by a local Ollama server.
func NewOllamaEmbedder(url, model string, dims int) *OllamaEmbedder {
	return &OllamaEmbedder{
		url:   url,
		model: model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		dims: dims,
	}
}

// Embed sends texts to the Ollama embed API and returns the embeddings.
func (e *OllamaEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody := ollamaEmbedRequest{
		Model: e.model,
		Input: texts,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.url+"/api/embed", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("ollama: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("ollama: unexpected status %d", resp.StatusCode)
	}

	var ollamaResp ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("ollama: decode response: %w", err)
	}

	return ollamaResp.Embeddings, nil
}

// Dimensions returns the dimensionality of embeddings produced by this embedder.
func (e *OllamaEmbedder) Dimensions() int {
	return e.dims
}
