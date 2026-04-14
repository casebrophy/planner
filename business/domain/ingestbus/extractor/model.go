package extractor

import "context"

// Extractor defines the interface for AI extraction.
type Extractor interface {
	ExtractEmail(ctx context.Context, subject, bodyText, fromAddress, userCorrection string, activeContexts []ContextRef) (EmailExtraction, error)
	ExtractText(ctx context.Context, text, userCorrection string, activeContexts []ContextRef, typeHint string) (TextExtraction, error)
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
	Summary                  string               `json:"summary"`
	SenderName               string               `json:"sender_name"`
	SenderDomain             string               `json:"sender_domain"`
	ActionItems              []ActionItem         `json:"action_items"`
	Deadlines                []Deadline           `json:"deadlines"`
	SuggestedContextKeywords []string             `json:"suggested_context_keywords"`
	Sentiment                string               `json:"sentiment"`
	SuggestedContextID       *string              `json:"suggested_context_id,omitempty"`
	ContextConfidence        float64              `json:"context_confidence,omitempty"`
	SuggestNewContext        bool                 `json:"suggest_new_context,omitempty"`
	SuggestedContextTitle    string               `json:"suggested_context_title,omitempty"`
	EntityResolutions        []EntityResolution   `json:"entity_resolutions,omitempty"`
}

// ExtractedEvent represents an event extracted from text input.
type ExtractedEvent struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location,omitempty"`
	StartsAt    string `json:"starts_at"`
	EndsAt      string `json:"ends_at,omitempty"`
	AllDay      bool   `json:"all_day"`
	IsAmbiguous bool   `json:"is_ambiguous"`
}

// ExtractedNote represents a note extracted from text input.
type ExtractedNote struct {
	Content       string   `json:"content"`
	SuggestedTags []string `json:"suggested_tags,omitempty"`
}

// AmbiguousReference represents a vague reference in voice input that the AI couldn't resolve.
type AmbiguousReference struct {
	OriginalText  string `json:"original_text"`
	ReferenceType string `json:"reference_type"` // pronoun, vague_noun, implicit
}

// EntityMatch represents a candidate entity found via semantic search.
type EntityMatch struct {
	ID         string  `json:"id"`
	SourceType string  `json:"source_type"` // "event", "task", "note"
	Title      string  `json:"title"`
	Content    string  `json:"content"`
	Similarity float64 `json:"similarity"`
}

// EntityResolution is Claude's decision about whether the input references an existing entity.
type EntityResolution struct {
	Action      string  `json:"action"`                // "update", "create", "ambiguous"
	MatchedID   string  `json:"matched_id,omitempty"`   // UUID of matched entity
	MatchedType string  `json:"matched_type,omitempty"` // "event", "task", "note"
	Confidence  float64 `json:"confidence"`             // 0-1
	Reasoning   string  `json:"reasoning"`              // Why this decision
}

// TextExtraction holds the AI-extracted data from a voice capture or text input.
type TextExtraction struct {
	Summary                  string                 `json:"summary"`
	ActionItems              []ActionItem           `json:"action_items"`
	Deadlines                []Deadline             `json:"deadlines"`
	Events                   []ExtractedEvent       `json:"events"`
	Notes                    []ExtractedNote        `json:"notes"`
	AmbiguousReferences      []AmbiguousReference   `json:"ambiguous_references,omitempty"`
	SuggestedContextKeywords []string               `json:"suggested_context_keywords"`
	SuggestedContextID       *string                `json:"suggested_context_id,omitempty"`
	ContextConfidence        float64                `json:"context_confidence,omitempty"`
	SuggestNewContext        bool                   `json:"suggest_new_context,omitempty"`
	SuggestedContextTitle    string                 `json:"suggested_context_title,omitempty"`
	EntityResolutions        []EntityResolution     `json:"entity_resolutions,omitempty"`
}
