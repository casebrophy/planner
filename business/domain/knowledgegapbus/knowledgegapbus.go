package knowledgegapbus

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/embeddingbus"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/google/uuid"
)

// EmbeddingSearcher is the interface for semantic search.
type EmbeddingSearcher interface {
	Search(ctx context.Context, query string, sourceTypes []string, limit int) ([]embeddingbus.SearchResult, error)
}

// ClarificationCreator is the interface for creating clarification cards.
type ClarificationCreator interface {
	Count(ctx context.Context, filter clarificationbus.QueryFilter) (int, error)
	Create(ctx context.Context, nc clarificationbus.NewClarificationItem) (clarificationbus.ClarificationItem, error)
}

// Business encapsulates knowledge gap detection logic.
type Business struct {
	log              *logger.Logger
	clarificationBus ClarificationCreator
	embeddingBus     EmbeddingSearcher
	analyzer         GapAnalyzer
}

// New creates a new knowledge gap business instance.
func New(log *logger.Logger, clarificationBus ClarificationCreator, embeddingBus EmbeddingSearcher, analyzer GapAnalyzer) *Business {
	return &Business{
		log:              log,
		clarificationBus: clarificationBus,
		embeddingBus:     embeddingBus,
		analyzer:         analyzer,
	}
}

// Detect analyzes entity content for knowledge gaps and creates clarification cards.
func (b *Business) Detect(ctx context.Context, entityType string, entityID uuid.UUID, content string) (GapDetectionResult, error) {
	// Search for related entities.
	searchResults, err := b.embeddingBus.Search(ctx, content, nil, 10)
	if err != nil {
		return GapDetectionResult{}, err
	}

	// Filter results by similarity threshold.
	var filtered []embeddingbus.SearchResult
	for _, result := range searchResults {
		if result.Similarity > 0.5 {
			filtered = append(filtered, result)
		}
	}

	// Early return if no related entities found.
	if len(filtered) == 0 {
		return GapDetectionResult{}, nil
	}

	// Build RelatedEntitySummary list.
	var summaries []RelatedEntitySummary
	for _, result := range filtered {
		summaries = append(summaries, RelatedEntitySummary{
			SourceType: result.SourceType,
			SourceID:   result.SourceID.String(),
			Similarity: result.Similarity,
		})
	}

	// Analyze gaps using the AI analyzer.
	analysis, err := b.analyzer.AnalyzeGaps(ctx, content, summaries)
	if err != nil {
		return GapDetectionResult{}, err
	}

	var cardsCreated, skipped int

	// Process each gap candidate.
	for _, gap := range analysis.Gaps {
		// Skip low-confidence gaps.
		if gap.Confidence <= 0.6 {
			skipped++
			continue
		}

		// Build KnowledgeGapOptions.
		options := clarificationbus.KnowledgeGapOptions{
			GapCategory:              gap.Category,
			RelatedEntityType:        filtered[0].SourceType,
			RelatedEntityID:          filtered[0].SourceID.String(),
			SuggestedQuestion:        gap.Question,
			ExistingKnowledgeSummary: fmt.Sprintf("Found %d related entities", len(summaries)),
		}

		optionsJSON, err := json.Marshal(options)
		if err != nil {
			b.log.Error(ctx, "knowledgegapbus: marshal options", "err", err)
			continue
		}

		// Create clarification card.
		nc := clarificationbus.NewClarificationItem{
			Kind:            clarificationkind.KnowledgeGap,
			SubjectType:     entityType,
			SubjectID:       entityID,
			SubjectDescription: "",
			Question:        gap.Question,
			ClaudeGuess:     nil,
			Reasoning:       &gap.Reasoning,
			AnswerOptions:   optionsJSON,
			PriorityScore:   float32(gap.Confidence),
		}

		_, err = b.clarificationBus.Upsert(ctx, nc)
		if err != nil {
			b.log.Error(ctx, "knowledgegapbus: create clarification", "err", err)
			continue
		}

		cardsCreated++
	}

	return GapDetectionResult{
		CardsCreated: cardsCreated,
		Skipped:      skipped,
	}, nil
}
