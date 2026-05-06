package knowledgegapbus

import (
	"context"
	"fmt"
	"strings"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/embeddingbus"
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
	// Knowledge gap detection is disabled. Keep the domain dormant for future re-enablement.
	return GapDetectionResult{}, nil
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

// buildExistingKnowledgeSummary formats related entities into a human-readable summary
// for the ExistingKnowledgeSummary field. Returns a formatted string with entity types,
// content snippets (truncated to ~60 chars), and similarity percentages.
func buildExistingKnowledgeSummary(summaries []RelatedEntitySummary) string {
	if len(summaries) == 0 {
		return ""
	}

	const contentSnippetLen = 60
	var buf strings.Builder

	fmt.Fprintf(&buf, "Found %d related items:\n", len(summaries))
	for _, s := range summaries {
		// Truncate content to ~60 chars, preserving word boundaries where possible.
		content := strings.TrimSpace(s.Content)
		if len([]rune(content)) > contentSnippetLen {
			r := []rune(content)
			content = string(r[:contentSnippetLen]) + "..."
		}

		// Format: "• type (XX% match): "snippet...""
		similarityPct := int(s.Similarity * 100)
		fmt.Fprintf(&buf, "• %s (%d%% match): %q\n", s.SourceType, similarityPct, content)
	}

	return strings.TrimSuffix(buf.String(), "\n")
}
