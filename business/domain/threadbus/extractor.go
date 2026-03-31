package threadbus

import "context"

// Extractor classifies raw thread entry text into structured fields.
type Extractor interface {
	ExtractThreadEntry(ctx context.Context, content string, subjectType string) (ExtractionResult, error)
}

// ExtractionResult contains the structured fields extracted from a thread entry.
type ExtractionResult struct {
	Kind              string  `json:"kind"`
	Sentiment         *string `json:"sentiment"`
	BlockingParty     *string `json:"blocking_party"`
	TimelineDeltaDays *int    `json:"timeline_delta_days"`
	RequiresAction    bool    `json:"requires_action"`
	Confidence        float64 `json:"confidence"`
}
