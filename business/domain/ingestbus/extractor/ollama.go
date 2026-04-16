package extractor

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/casebrophy/planner/foundation/ollamaclient"
)

// OllamaExtractor implements Extractor by delegating to a shared ollamaclient.Client.
type OllamaExtractor struct {
	client *ollamaclient.Client
	model  string
}

// NewOllamaExtractor creates an extractor backed by a shared Ollama client.
func NewOllamaExtractor(client *ollamaclient.Client, model string) *OllamaExtractor {
	return &OllamaExtractor{client: client, model: model}
}

// generate sends a prompt to the Ollama generate API and returns the response string.
func (e *OllamaExtractor) generate(ctx context.Context, prompt string) (string, error) {
	return e.client.Generate(ctx, e.model, prompt)
}

// ExtractEmail uses Ollama to extract structured data from an email.
func (e *OllamaExtractor) ExtractEmail(ctx context.Context, subject, bodyText, fromAddress, userCorrection string, activeContexts []ContextRef) (EmailExtraction, error) {
	contextsJSON, _ := json.Marshal(activeContexts)
	prompt := BuildEmailExtractionPrompt(fromAddress, subject, bodyText, userCorrection, contextsJSON, nil)

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
func (e *OllamaExtractor) ExtractText(ctx context.Context, text, userCorrection string, activeContexts []ContextRef, typeHint string) (TextExtraction, error) {
	contextsJSON, _ := json.Marshal(activeContexts)
	prompt := BuildTextExtractionPrompt(text, userCorrection, contextsJSON, time.Now(), typeHint, nil)

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

// ExtractReceipt uses Ollama to parse OCR text from a receipt into structured data.
func (e *OllamaExtractor) ExtractReceipt(ctx context.Context, ocrText string) (ReceiptExtraction, error) {
	prompt := BuildReceiptExtractionPrompt(ocrText)

	raw, err := e.generate(ctx, prompt)
	if err != nil {
		return ReceiptExtraction{}, err
	}

	var extraction ReceiptExtraction
	if err := json.Unmarshal([]byte(raw), &extraction); err != nil {
		return ReceiptExtraction{}, fmt.Errorf("ollama: unmarshal receipt extraction: %w", err)
	}

	return extraction, nil
}

// AnalyzeGaps uses Ollama to identify knowledge gaps for a new entity.
func (e *OllamaExtractor) AnalyzeGaps(ctx context.Context, entityType, entityContent string, relatedEntities []RelatedEntity) (GapAnalysis, error) {
	prompt := BuildGapAnalysisPrompt(entityType, entityContent, relatedEntities)

	raw, err := e.generate(ctx, prompt)
	if err != nil {
		return GapAnalysis{}, err
	}

	var analysis GapAnalysis
	if err := json.Unmarshal([]byte(raw), &analysis); err != nil {
		return GapAnalysis{}, fmt.Errorf("ollama: unmarshal gap analysis: %w", err)
	}

	return analysis, nil
}
