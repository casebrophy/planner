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
