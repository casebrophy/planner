package extractor

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// ContextRef is a lightweight reference to an active context for the AI prompt.
type ContextRef struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// ActionItem represents a task extracted from an email.
type ActionItem struct {
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Priority        string   `json:"priority"`
	Interpretations []string `json:"interpretations,omitempty"`
}

// Deadline represents a deadline mentioned in an email.
type Deadline struct {
	Description string `json:"description"`
	Date        string `json:"date"`
	IsAmbiguous bool   `json:"is_ambiguous,omitempty"`
}

// EmailExtraction holds the AI-extracted data from an email.
type EmailExtraction struct {
	Summary                  string       `json:"summary"`
	SenderName               string       `json:"sender_name"`
	SenderDomain             string       `json:"sender_domain"`
	ActionItems              []ActionItem `json:"action_items"`
	Deadlines                []Deadline   `json:"deadlines"`
	SuggestedContextKeywords []string     `json:"suggested_context_keywords"`
	Sentiment                string       `json:"sentiment"`
	SuggestedContextID       *string      `json:"suggested_context_id,omitempty"`
	ContextConfidence        float64      `json:"context_confidence,omitempty"`
	SuggestNewContext        bool         `json:"suggest_new_context,omitempty"`
	SuggestedContextTitle    string       `json:"suggested_context_title,omitempty"`
}

// Extractor defines the interface for email AI extraction.
type Extractor interface {
	ExtractEmail(ctx context.Context, subject, bodyText, fromAddress string, activeContexts []ContextRef) (EmailExtraction, error)
}

// AnthropicExtractor implements Extractor using the Anthropic API.
type AnthropicExtractor struct {
	client *anthropic.Client
	model  string
}

// NewAnthropicExtractor creates an AnthropicExtractor with the given API key and model.
func NewAnthropicExtractor(apiKey, model string) *AnthropicExtractor {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	return &AnthropicExtractor{
		client: &client,
		model:  model,
	}
}

// ExtractEmail uses Anthropic to extract structured data from an email.
func (e *AnthropicExtractor) ExtractEmail(ctx context.Context, subject, bodyText, fromAddress string, activeContexts []ContextRef) (EmailExtraction, error) {
	contextsJSON, _ := json.Marshal(activeContexts)

	prompt := fmt.Sprintf(`Analyze this email and extract structured data. Return ONLY valid JSON with no other text.

Email:
From: %s
Subject: %s
Body:
%s

Active contexts (match this email to one if relevant):
%s

Return JSON with this exact schema:
{
  "summary": "1-2 sentence summary",
  "sender_name": "sender's name or empty string",
  "sender_domain": "sender's domain from email address",
  "action_items": [{"title": "short title", "description": "detail", "priority": "low|medium|high|urgent", "interpretations": ["interpretation1", "interpretation2"]}],
  "deadlines": [{"description": "what is due", "date": "YYYY-MM-DD or natural language if ambiguous", "is_ambiguous": false}],
  "suggested_context_keywords": ["keyword1", "keyword2"],
  "sentiment": "positive|neutral|negative|mixed",
  "suggested_context_id": "UUID of best matching context or null",
  "context_confidence": 0.0,
  "suggest_new_context": false,
  "suggested_context_title": "title for new context if suggest_new_context is true"
}

Rules:
- Set context_confidence to a value between 0.0 and 1.0 reflecting how well this email matches the suggested context
- If no existing context matches well, set suggest_new_context to true and provide a suggested_context_title
- Set is_ambiguous on deadlines when the date is relative or vague (e.g. "end of month", "soon", "next week")
- Include interpretations array on action_items only when the item is genuinely ambiguous (could be a pleasantry vs. real task)`, fromAddress, subject, bodyText, string(contextsJSON))

	resp, err := e.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     e.model,
		MaxTokens: 1024,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return EmailExtraction{}, fmt.Errorf("anthropic api: %w", err)
	}

	if len(resp.Content) == 0 {
		return EmailExtraction{}, fmt.Errorf("empty response from anthropic")
	}

	var text string
	for _, block := range resp.Content {
		if block.Type == "text" {
			text = block.Text
			break
		}
	}

	if text == "" {
		return EmailExtraction{}, fmt.Errorf("no text content in anthropic response")
	}

	var extraction EmailExtraction
	if err := json.Unmarshal([]byte(text), &extraction); err != nil {
		return EmailExtraction{}, fmt.Errorf("parse extraction: %w", err)
	}

	return extraction, nil
}
