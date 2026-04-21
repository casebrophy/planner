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

// EventPrepOptions is the typed answer options for event_prep clarifications.
// Lists the event and the tasks detected as likely prerequisites for it.
type EventPrepOptions struct {
	EventID        string   `json:"event_id"`
	EventTitle     string   `json:"event_title"`
	EventStartsAt  string   `json:"event_starts_at"`
	PrepTaskIDs    []string `json:"prep_task_ids"`
	PrepTaskTitles []string `json:"prep_task_titles"`
}

// AmbiguousEntityMatchOptions is the typed answer options for ambiguous_entity_match clarifications.
type AmbiguousEntityMatchOptions struct {
	CandidateID    string   `json:"candidate_id"`
	CandidateType  string   `json:"candidate_type"` // "event", "task", "note"
	CandidateTitle string   `json:"candidate_title"`
	Similarity     float64  `json:"similarity"`
	Choices        []string `json:"choices"` // ["use_existing", "create_new"]
}

// KnowledgeGapOptions is the typed answer options for knowledge_gap clarifications.
// Describes a detected knowledge gap and what information would be useful to capture.
type KnowledgeGapOptions struct {
	GapCategory              string   `json:"gap_category"`               // missing_contact, missing_location, missing_detail, etc.
	RelatedEntityType        string   `json:"related_entity_type"`       // Type of related entity found via search
	RelatedEntityID          string   `json:"related_entity_id"`         // ID of related entity
	SuggestedQuestion        string   `json:"suggested_question"`        // The question prompting the user for info
	ExistingKnowledgeSummary string   `json:"existing_knowledge_summary"` // Summary of what we already know
	Confidence               float64  `json:"confidence"`                // AI confidence in this gap (0-1)
	Options                  []string `json:"options,omitempty"`         // Optional answer choices
}
