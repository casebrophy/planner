package knowledgegapbus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	Upsert(ctx context.Context, nc clarificationbus.NewClarificationItem) (clarificationbus.ClarificationItem, error)
}

// Business encapsulates knowledge gap detection logic.
type Business struct {
	log              *logger.Logger
	clarificationBus ClarificationCreator
	embeddingBus     EmbeddingSearcher
	analyzer         GapAnalyzer
	cfg              Config
}

// New creates a new knowledge gap business instance.
func New(log *logger.Logger, clarificationBus ClarificationCreator, embeddingBus EmbeddingSearcher, analyzer GapAnalyzer, cfg Config) *Business {
	return &Business{
		log:              log,
		clarificationBus: clarificationBus,
		embeddingBus:     embeddingBus,
		analyzer:         analyzer,
		cfg:              defaultConfig(cfg),
	}
}

// Detect analyzes entity content for knowledge gaps and creates clarification cards.
func (b *Business) Detect(ctx context.Context, entityType string, entityID uuid.UUID, content string) (GapDetectionResult, error) {
	return b.DetectWithOptions(ctx, entityType, entityID, content, DetectOptions{})
}

// DetectWithOptions is the full implementation; Detect is a convenience wrapper.
func (b *Business) DetectWithOptions(ctx context.Context, entityType string, entityID uuid.UUID, content string, opts DetectOptions) (GapDetectionResult, error) {
	searchResults, err := b.embeddingBus.Search(ctx, content, nil, b.cfg.EmbeddingLimit)
	if err != nil {
		return GapDetectionResult{}, err
	}

	var filtered []embeddingbus.SearchResult
	for _, result := range searchResults {
		if result.Similarity > b.cfg.SimilarityThreshold {
			filtered = append(filtered, result)
		}
	}

	filtered = dedupeByContent(filtered)

	if len(filtered) == 0 {
		return GapDetectionResult{}, nil
	}

	// Build a map from source ID → search result for per-candidate entity lookup.
	resultByID := make(map[string]embeddingbus.SearchResult, len(filtered))
	for _, r := range filtered {
		resultByID[r.SourceID.String()] = r
	}

	var summaries []RelatedEntitySummary
	for _, result := range filtered {
		summaries = append(summaries, RelatedEntitySummary{
			SourceType: result.SourceType,
			SourceID:   result.SourceID.String(),
			Similarity: result.Similarity,
			Content:    result.Content,
		})
	}

	analysis, err := b.analyzer.AnalyzeGaps(ctx, content, summaries)
	if err != nil {
		return GapDetectionResult{}, err
	}

	var cardsCreated, skipped int

	for _, gap := range analysis.Gaps {
		if gap.Confidence <= b.cfg.ConfidenceThreshold {
			skipped++
			continue
		}

		// Pick the best related entity by matching gap.RelatedIDs against search results.
		relatedEntity := filtered[0]
		for _, rid := range gap.RelatedIDs {
			if r, ok := resultByID[rid]; ok {
				relatedEntity = r
				break
			}
		}

		options := clarificationbus.KnowledgeGapOptions{
			GapCategory:              gap.Category.String(),
			RelatedEntityType:        relatedEntity.SourceType,
			RelatedEntityID:          relatedEntity.SourceID.String(),
			SuggestedQuestion:        gap.Question,
			ExistingKnowledgeSummary: fmt.Sprintf("Found %d related entities", len(summaries)),
		}

		optionsJSON, err := json.Marshal(options)
		if err != nil {
			b.log.Error(ctx, "knowledgegapbus: marshal options", "err", err)
			continue
		}

		if opts.DryRun {
			cardsCreated++
			continue
		}

		nc := clarificationbus.NewClarificationItem{
			Kind:               clarificationkind.KnowledgeGap,
			SubjectType:        entityType,
			SubjectID:          entityID,
			SubjectDescription: buildSubjectDescription(entityType, content),
			GapCategory:        gap.Category.String(),
			Question:           gap.Question,
			ClaudeGuess:        nil,
			Reasoning:          &gap.Reasoning,
			AnswerOptions:      optionsJSON,
			PriorityScore:      float32(gap.Confidence),
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

// buildSubjectDescription produces a human-readable label for the clarification
// card by taking the first non-empty line of the entity content (typically the
// title) and prefixing with the entity type. Truncates overly long titles.
func buildSubjectDescription(entityType, content string) string {
	const maxLen = 120
	title := strings.TrimSpace(content)
	if idx := strings.IndexByte(title, '\n'); idx >= 0 {
		title = strings.TrimSpace(title[:idx])
	}
	if len([]rune(title)) > maxLen {
		r := []rune(title)
		title = string(r[:maxLen]) + "..."
	}
	if title == "" {
		return entityType
	}
	if entityType == "" {
		return title
	}
	return fmt.Sprintf("%s: %s", entityType, title)
}

// dedupeByContent collapses near-identical related entities (e.g. recurring
// task siblings) so the analyzer isn't biased by N copies of the same content.
// Keys by a normalized content prefix; keeps the highest-similarity result per key.
func dedupeByContent(results []embeddingbus.SearchResult) []embeddingbus.SearchResult {
	const prefixLen = 200
	type bucket struct {
		idx int
		sim float64
	}
	buckets := make(map[string]bucket)
	order := make([]string, 0, len(results))
	for i, r := range results {
		key := normalizeContentKey(r.Content, prefixLen)
		if b, ok := buckets[key]; ok {
			if r.Similarity > b.sim {
				buckets[key] = bucket{idx: i, sim: r.Similarity}
			}
			continue
		}
		buckets[key] = bucket{idx: i, sim: r.Similarity}
		order = append(order, key)
	}
	out := make([]embeddingbus.SearchResult, 0, len(buckets))
	for _, key := range order {
		out = append(out, results[buckets[key].idx])
	}
	return out
}

// normalizeContentKey produces a stable key for near-duplicate detection:
// trimmed, lowercased, collapsed whitespace, truncated to prefixLen runes.
func normalizeContentKey(s string, prefixLen int) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Join(strings.Fields(s), " ")
	if len([]rune(s)) > prefixLen {
		r := []rune(s)
		s = string(r[:prefixLen])
	}
	return s
}
