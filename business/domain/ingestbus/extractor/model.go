package extractor

import "context"

// Extractor defines the interface for AI extraction.
type Extractor interface {
	ExtractEmail(ctx context.Context, subject, bodyText, fromAddress, userCorrection string, activeContexts []ContextRef) (EmailExtraction, error)
	ExtractText(ctx context.Context, text, userCorrection string, activeContexts []ContextRef, typeHint string, typeHintConfidence float64, candidates []EntityMatch, contextAnnotations []string) (TextExtraction, error)
	ExtractReceipt(ctx context.Context, ocrText string) (ReceiptExtraction, error)
	AnalyzeGaps(ctx context.Context, entityType, entityContent string, relatedEntities []RelatedEntity) (GapAnalysis, error)
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
	ReclassifiedAs           string                 `json:"reclassified_as,omitempty"` // "task", "event", or "note" when overriding the heuristic hint; empty otherwise
}

// ReceiptExtraction holds structured data extracted from OCR'd receipt text.
type ReceiptExtraction struct {
	Merchant string            `json:"merchant"`
	Date     string            `json:"date"`     // YYYY-MM-DD
	Total    int               `json:"total"`    // cents
	Tax      int               `json:"tax"`      // cents
	Subtotal int               `json:"subtotal"` // cents
	Items    []ReceiptLineItem `json:"items"`
	Notes    string            `json:"notes,omitempty"`
}

// ReceiptLineItem is a single line item on a receipt.
type ReceiptLineItem struct {
	Description string `json:"description"`
	Amount      int    `json:"amount"`   // cents
	Quantity    int    `json:"quantity"`
}

// RelatedEntity is a lightweight summary of an entity related to a new entity, used for gap analysis.
type RelatedEntity struct {
	ID         string `json:"id"`
	SourceType string `json:"source_type"` // "task", "event", "note"
	Title      string `json:"title"`
	Content    string `json:"content"`
}

// GapCandidate is a single gap identified by the AI.
type GapCandidate struct {
	Category           string   `json:"category"`            // missing_contact, missing_location, missing_detail, missing_dependency, missing_context
	Question           string   `json:"question"`            // e.g. "What is Dr. Smith's phone number?"
	Reasoning          string   `json:"reasoning"`           // e.g. "You have an appointment but no contact info stored"
	Confidence         float64  `json:"confidence"`          // 0-1
	RelatedIDs         []string `json:"related_ids"`         // IDs of related entities that informed this gap
	Options            []string `json:"options,omitempty"`   // Optional answer choices for the user
	OptionsConfidence  float64  `json:"options_confidence,omitempty"` // Confidence in the options (0-1), 0 if no options
}

// GapAnalysis holds the AI-identified gaps for a new entity.
type GapAnalysis struct {
	Gaps []GapCandidate `json:"gaps"`
}
