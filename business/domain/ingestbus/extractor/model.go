package extractor

import "context"

// Extractor defines the interface for email AI extraction.
type Extractor interface {
	ExtractEmail(ctx context.Context, subject, bodyText, fromAddress string, activeContexts []ContextRef) (EmailExtraction, error)
}

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
