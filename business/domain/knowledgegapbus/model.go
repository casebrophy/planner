package knowledgegapbus

import "context"

// Gap category constants.
const (
	CategoryMissingContact    = "missing_contact"
	CategoryMissingLocation   = "missing_location"
	CategoryMissingContext    = "missing_context"
	CategoryMissingDependency = "missing_dependency"
	CategoryMissingDetail     = "missing_detail"
)

// GapCandidate represents a detected knowledge gap.
type GapCandidate struct {
	Category   string
	Question   string   // e.g. "What is Dr. Smith's phone number?"
	Reasoning  string   // e.g. "You have an appointment but no contact info stored"
	Confidence float64  // 0-1
	RelatedIDs []string // IDs of related entities that informed this gap
}

// GapDetectionResult is the outcome of a Detect() call.
type GapDetectionResult struct {
	CardsCreated int
	Skipped      int // duplicates or low-confidence gaps skipped
}

// GapAnalysis is the structured response from the AI extractor.
type GapAnalysis struct {
	Gaps []GapCandidate
}

// GapAnalyzer is the interface knowledgegapbus depends on for AI gap analysis.
// It is implemented by the extractor package (Task 3).
type GapAnalyzer interface {
	AnalyzeGaps(ctx context.Context, entityContent string, relatedSummaries []RelatedEntitySummary) (GapAnalysis, error)
}

// RelatedEntitySummary is a lightweight description of a related entity found via semantic search.
type RelatedEntitySummary struct {
	SourceType string
	SourceID   string
	Similarity float64
	// Content summary is passed as part of entity description to the AI prompt.
}
