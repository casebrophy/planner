package extractor

import (
	"context"
	"encoding/json"
	"time"

	"github.com/casebrophy/planner/foundation/claudecli"
)

// emailExtractionSchema is the JSON schema for structured output validation.
const emailExtractionSchema = `{
  "type": "object",
  "properties": {
    "summary": {"type": "string"},
    "sender_name": {"type": "string"},
    "sender_domain": {"type": "string"},
    "action_items": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "title": {"type": "string"},
          "description": {"type": "string"},
          "priority": {"type": "string", "enum": ["low", "medium", "high", "urgent"]},
          "interpretations": {"type": "array", "items": {"type": "string"}}
        },
        "required": ["title", "description", "priority"]
      }
    },
    "deadlines": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "description": {"type": "string"},
          "date": {"type": "string"},
          "is_ambiguous": {"type": "boolean"}
        },
        "required": ["description", "date"]
      }
    },
    "suggested_context_keywords": {"type": "array", "items": {"type": "string"}},
    "sentiment": {"type": "string", "enum": ["positive", "neutral", "negative", "mixed"]},
    "suggested_context_id": {"type": ["string", "null"]},
    "context_confidence": {"type": "number"},
    "suggest_new_context": {"type": "boolean"},
    "suggested_context_title": {"type": "string"}
  },
  "required": ["summary", "sender_name", "sender_domain", "action_items", "deadlines", "suggested_context_keywords", "sentiment"]
}`

// ClaudeCodeExtractor implements Extractor using the Claude Code CLI.
type ClaudeCodeExtractor struct {
	client *claudecli.Client
}

// NewClaudeCodeExtractor creates an extractor backed by the Claude CLI.
func NewClaudeCodeExtractor(client *claudecli.Client) *ClaudeCodeExtractor {
	return &ClaudeCodeExtractor{client: client}
}

// ExtractEmail uses the Claude CLI to extract structured data from an email.
func (e *ClaudeCodeExtractor) ExtractEmail(ctx context.Context, subject, bodyText, fromAddress string, activeContexts []ContextRef) (EmailExtraction, error) {
	contextsJSON, _ := json.Marshal(activeContexts)
	prompt := BuildEmailExtractionPrompt(fromAddress, subject, bodyText, contextsJSON)

	var extraction EmailExtraction
	shouldEscalate := func() bool {
		return len(extraction.ActionItems) == 0 && extraction.ContextConfidence < 0.3
	}

	if err := e.client.RunJSON(ctx, prompt, emailExtractionSchema, &extraction, shouldEscalate); err != nil {
		return EmailExtraction{}, err
	}

	return extraction, nil
}

// textExtractionSchema is the JSON schema for text/voice extraction structured output validation.
const textExtractionSchema = `{
  "type": "object",
  "properties": {
    "summary": {"type": "string"},
    "action_items": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "title": {"type": "string"},
          "description": {"type": "string"},
          "priority": {"type": "string", "enum": ["low", "medium", "high", "urgent"]},
          "interpretations": {"type": "array", "items": {"type": "string"}}
        },
        "required": ["title", "description", "priority"]
      }
    },
    "deadlines": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "description": {"type": "string"},
          "date": {"type": "string"},
          "is_ambiguous": {"type": "boolean"}
        },
        "required": ["description", "date"]
      }
    },
    "events": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "title": {"type": "string"},
          "description": {"type": "string"},
          "location": {"type": "string"},
          "starts_at": {"type": "string"},
          "ends_at": {"type": "string"},
          "all_day": {"type": "boolean"},
          "is_ambiguous": {"type": "boolean"}
        },
        "required": ["title", "starts_at"]
      }
    },
    "suggested_context_keywords": {"type": "array", "items": {"type": "string"}},
    "suggested_context_id": {"type": ["string", "null"]},
    "context_confidence": {"type": "number"},
    "suggest_new_context": {"type": "boolean"},
    "suggested_context_title": {"type": "string"}
  },
  "required": ["summary", "action_items", "deadlines", "events", "suggested_context_keywords"]
}`

// ExtractText uses the Claude CLI to extract structured data from text/voice input.
func (e *ClaudeCodeExtractor) ExtractText(ctx context.Context, text string, activeContexts []ContextRef) (TextExtraction, error) {
	contextsJSON, _ := json.Marshal(activeContexts)
	prompt := BuildTextExtractionPrompt(text, contextsJSON, time.Now())

	var extraction TextExtraction
	shouldEscalate := func() bool {
		return len(extraction.ActionItems) == 0
	}

	if err := e.client.RunJSON(ctx, prompt, textExtractionSchema, &extraction, shouldEscalate); err != nil {
		return TextExtraction{}, err
	}

	return extraction, nil
}
