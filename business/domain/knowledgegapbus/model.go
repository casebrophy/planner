package knowledgegapbus

import (
	"context"

	"github.com/casebrophy/planner/business/types/gapcategory"
)

// Config holds tunable parameters for gap detection.
type Config struct {
	ConfidenceThreshold float64 // minimum confidence to create a card (default 0.6)
	EmbeddingLimit      int     // max search results from embedding bus (default 10)
	SimilarityThreshold float64 // minimum similarity to consider a related entity (default 0.5)
}

func defaultConfig(cfg Config) Config {
	if cfg.ConfidenceThreshold == 0 {
		cfg.ConfidenceThreshold = 0.6
	}
	if cfg.EmbeddingLimit == 0 {
		cfg.EmbeddingLimit = 10
	}
	if cfg.SimilarityThreshold == 0 {
		cfg.SimilarityThreshold = 0.5
	}
	return cfg
}

// DetectOptions controls per-call behavior of Detect.
type DetectOptions struct {
	DryRun bool // if true, analyze gaps but do not create clarification cards
}

// GapCandidate represents a detected knowledge gap.
type GapCandidate struct {
	Category   gapcategory.Category
	Question   string
	Reasoning  string
	Confidence float64
	RelatedIDs []string
}

// GapDetectionResult is the outcome of a Detect() call.
type GapDetectionResult struct {
	CardsCreated int
	Skipped      int
}

// GapAnalysis is the structured response from the AI extractor.
type GapAnalysis struct {
	Gaps []GapCandidate
}

// GapAnalyzer is the interface knowledgegapbus depends on for AI gap analysis.
type GapAnalyzer interface {
	AnalyzeGaps(ctx context.Context, entityContent string, relatedSummaries []RelatedEntitySummary) (GapAnalysis, error)
}

// RelatedEntitySummary is a lightweight description of a related entity found via semantic search.
type RelatedEntitySummary struct {
	SourceType string
	SourceID   string
	Similarity float64
	Content    string
}
