package clarificationbus

// ContextRef is a lightweight context pointer used in clarification options.
// Defined here so clarificationbus has no dependency on ingestbus/extractor.
type ContextRef struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// ContextAssignmentOptions is the typed answer options for context_assignment clarifications.
type ContextAssignmentOptions struct {
	SuggestedContext  string       `json:"suggested_context"`
	Confidence        float64      `json:"confidence"`
	AvailableContexts []ContextRef `json:"available_contexts"`
}

// NewContextOptions is the typed answer options for new_context clarifications.
type NewContextOptions struct {
	ContextID string `json:"context_id"`
	Title     string `json:"title"`
}

// AmbiguousActionOptions is the typed answer options for ambiguous_action clarifications.
type AmbiguousActionOptions struct {
	Interpretations []string `json:"interpretations"`
}

// AmbiguousDeadlineOptions is the typed answer options for ambiguous_deadline clarifications.
type AmbiguousDeadlineOptions struct {
	Description string `json:"description"`
	RawDate     string `json:"raw_date"`
}

// EntityLinkOptions is the typed answer options for entity_link clarifications.
// Describes a suggested link between two entities.
type EntityLinkOptions struct {
	SourceType string  `json:"source_type"`
	SourceID   string  `json:"source_id"`
	TargetType string  `json:"target_type"`
	TargetID   string  `json:"target_id"`
	Confidence float64 `json:"confidence"`
}

// TypeAssignmentOptions is the typed answer options for type_assignment clarifications.
// It presents the user with the clause text and the available types to choose from.
type TypeAssignmentOptions struct {
	ClauseText    string   `json:"clause_text"`
	PredictedType string   `json:"predicted_type"`
	Confidence    float64  `json:"confidence"`
	Options       []string `json:"options"` // e.g. ["task", "note", "event"]
}

// VoiceReferenceOptions is the typed answer options for voice_reference clarifications.
// Presents the ambiguous text so the user can provide the correct reference.
type VoiceReferenceOptions struct {
	OriginalText  string `json:"original_text"`
	ReferenceType string `json:"reference_type"`
	ClauseText    string `json:"clause_text"`
}
