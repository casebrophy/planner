package extractor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ollamaGenerateRequest is the request body for the Ollama /api/generate endpoint.
type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format"`
}

// ollamaGenerateResponse is the response body from the Ollama /api/generate endpoint.
type ollamaGenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// OllamaExtractor implements Extractor using a local Ollama instance.
type OllamaExtractor struct {
	url    string
	model  string
	client *http.Client
}

// NewOllamaExtractor creates an extractor backed by a local Ollama server.
func NewOllamaExtractor(url, model string) *OllamaExtractor {
	return &OllamaExtractor{
		url:   url,
		model: model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// generate sends a prompt to the Ollama generate API and returns the response string.
func (e *OllamaExtractor) generate(ctx context.Context, prompt string) (string, error) {
	reqBody := ollamaGenerateRequest{
		Model:  e.model,
		Prompt: prompt,
		Stream: false,
		Format: "json",
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("ollama: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.url+"/api/generate", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("ollama: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return "", fmt.Errorf("ollama: unexpected status %d", resp.StatusCode)
	}

	var ollamaResp ollamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("ollama: decode response: %w", err)
	}

	return ollamaResp.Response, nil
}

// ExtractEmail uses Ollama to extract structured data from an email.
func (e *OllamaExtractor) ExtractEmail(ctx context.Context, subject, bodyText, fromAddress string, activeContexts []ContextRef) (EmailExtraction, error) {
	contextsJSON, _ := json.Marshal(activeContexts)
	prompt := BuildEmailExtractionPrompt(fromAddress, subject, bodyText, contextsJSON)

	raw, err := e.generate(ctx, prompt)
	if err != nil {
		return EmailExtraction{}, err
	}

	var extraction EmailExtraction
	if err := json.Unmarshal([]byte(raw), &extraction); err != nil {
		return EmailExtraction{}, fmt.Errorf("ollama: unmarshal email extraction: %w", err)
	}

	// ContextConfidence is fixed at 0.85 for Ollama extractions: local models do not
	// reliably self-report confidence, so we assign a consistent moderate value.
	extraction.ContextConfidence = 0.85
	return extraction, nil
}

// ExtractText uses Ollama to extract structured data from text/voice input.
// When typeHint is "transaction", uses a transaction-specific prompt for merchant
// name cleanup and category suggestion.
func (e *OllamaExtractor) ExtractText(ctx context.Context, text string, activeContexts []ContextRef, typeHint string) (TextExtraction, error) {
	contextsJSON, _ := json.Marshal(activeContexts)
	prompt := BuildTextExtractionPrompt(text, contextsJSON, time.Now(), typeHint)

	raw, err := e.generate(ctx, prompt)
	if err != nil {
		return TextExtraction{}, err
	}

	var extraction TextExtraction
	if err := json.Unmarshal([]byte(raw), &extraction); err != nil {
		return TextExtraction{}, fmt.Errorf("ollama: unmarshal text extraction: %w", err)
	}

	// ContextConfidence is fixed at 0.85 for Ollama extractions: local models do not
	// reliably self-report confidence, so we assign a consistent moderate value.
	extraction.ContextConfidence = 0.85
	return extraction, nil
}
