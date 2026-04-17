package knowledgegapbus

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/embeddingbus"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/google/uuid"
)

// mockEmbeddingBus is a test mock for EmbeddingSearcher.
type mockEmbeddingBus struct {
	results []embeddingbus.SearchResult
	err     error
}

func (m *mockEmbeddingBus) Search(ctx context.Context, query string, sourceTypes []string, limit int) ([]embeddingbus.SearchResult, error) {
	return m.results, m.err
}

// mockClarificationBus is a test mock for ClarificationCreator.
type mockClarificationBus struct {
	countResult int
	countErr    error
	created     []clarificationbus.NewClarificationItem
	createErr   error
}

func (m *mockClarificationBus) Count(ctx context.Context, filter clarificationbus.QueryFilter) (int, error) {
	return m.countResult, m.countErr
}

func (m *mockClarificationBus) Create(ctx context.Context, nc clarificationbus.NewClarificationItem) (clarificationbus.ClarificationItem, error) {
	m.created = append(m.created, nc)
	return clarificationbus.ClarificationItem{}, m.createErr
}

func (m *mockClarificationBus) Upsert(ctx context.Context, nc clarificationbus.NewClarificationItem) (clarificationbus.ClarificationItem, error) {
	m.created = append(m.created, nc)
	return clarificationbus.ClarificationItem{}, m.createErr
}

// mockGapAnalyzer is a test mock for GapAnalyzer.
type mockGapAnalyzer struct {
	analysis              GapAnalysis
	err                   error
	receivedSummaries     []RelatedEntitySummary
	captureReceived       bool
}

func (m *mockGapAnalyzer) AnalyzeGaps(ctx context.Context, entityContent string, relatedSummaries []RelatedEntitySummary) (GapAnalysis, error) {
	if m.captureReceived {
		m.receivedSummaries = relatedSummaries
	}
	return m.analysis, m.err
}

func TestDetect_NoRelatedEntities(t *testing.T) {
	buf := &bytes.Buffer{}
	log := logger.New(buf, slog.LevelDebug, "test")
	mockEmbed := &mockEmbeddingBus{results: []embeddingbus.SearchResult{}}
	mockClar := &mockClarificationBus{}
	mockAnalyzer := &mockGapAnalyzer{}

	b := New(log, mockClar, mockEmbed, mockAnalyzer)

	result, err := b.Detect(context.Background(), "task", uuid.New(), "test content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.CardsCreated != 0 || result.Skipped != 0 {
		t.Errorf("expected empty result, got CardsCreated=%d, Skipped=%d", result.CardsCreated, result.Skipped)
	}
}

func TestDetect_CreatesCard(t *testing.T) {
	buf := &bytes.Buffer{}
	log := logger.New(buf, slog.LevelDebug, "test")

	entityID := uuid.New()
	relatedID := uuid.New()

	// Set up embedding search to return 1 result above threshold.
	mockEmbed := &mockEmbeddingBus{
		results: []embeddingbus.SearchResult{
			{
				Embedding: embeddingbus.Embedding{
					SourceType: "context",
					SourceID:   relatedID,
				},
				Similarity: 0.8,
			},
		},
	}

	// Set up clarification bus to report no duplicates.
	mockClar := &mockClarificationBus{countResult: 0}

	// Set up analyzer to return 1 high-confidence gap.
	mockAnalyzer := &mockGapAnalyzer{
		analysis: GapAnalysis{
			Gaps: []GapCandidate{
				{
					Category:   CategoryMissingContact,
					Question:   "What is the contact?",
					Reasoning:  "No contact info stored",
					Confidence: 0.8,
					RelatedIDs: []string{relatedID.String()},
				},
			},
		},
	}

	b := New(log, mockClar, mockEmbed, mockAnalyzer)

	result, err := b.Detect(context.Background(), "task", entityID, "test content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.CardsCreated != 1 {
		t.Errorf("expected CardsCreated=1, got %d", result.CardsCreated)
	}
	if result.Skipped != 0 {
		t.Errorf("expected Skipped=0, got %d", result.Skipped)
	}

	if len(mockClar.created) != 1 {
		t.Fatalf("expected 1 created card, got %d", len(mockClar.created))
	}

	created := mockClar.created[0]
	if created.Question != "What is the contact?" {
		t.Errorf("expected Question='What is the contact?', got '%s'", created.Question)
	}
}

func TestDetect_LowConfidenceSkipped(t *testing.T) {
	buf := &bytes.Buffer{}
	log := logger.New(buf, slog.LevelDebug, "test")

	entityID := uuid.New()
	relatedID := uuid.New()

	mockEmbed := &mockEmbeddingBus{
		results: []embeddingbus.SearchResult{
			{
				Embedding: embeddingbus.Embedding{
					SourceType: "context",
					SourceID:   relatedID,
				},
				Similarity: 0.8,
			},
		},
	}

	mockClar := &mockClarificationBus{}

	// Return a gap with low confidence (0.4, below the 0.6 threshold).
	mockAnalyzer := &mockGapAnalyzer{
		analysis: GapAnalysis{
			Gaps: []GapCandidate{
				{
					Category:   CategoryMissingContact,
					Question:   "What is the contact?",
					Reasoning:  "No contact info stored",
					Confidence: 0.4,
					RelatedIDs: []string{relatedID.String()},
				},
			},
		},
	}

	b := New(log, mockClar, mockEmbed, mockAnalyzer)

	result, err := b.Detect(context.Background(), "task", entityID, "test content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.CardsCreated != 0 {
		t.Errorf("expected CardsCreated=0, got %d", result.CardsCreated)
	}
	if result.Skipped != 1 {
		t.Errorf("expected Skipped=1, got %d", result.Skipped)
	}

	if len(mockClar.created) != 0 {
		t.Errorf("expected no cards created, got %d", len(mockClar.created))
	}
}

func TestDetect_ContentPassedToAnalyzer(t *testing.T) {
	buf := &bytes.Buffer{}
	log := logger.New(buf, slog.LevelDebug, "test")

	entityID := uuid.New()
	relatedID := uuid.New()
	relatedContent := "This is related content from the search result"

	// Set up embedding search with Content field populated.
	mockEmbed := &mockEmbeddingBus{
		results: []embeddingbus.SearchResult{
			{
				Embedding: embeddingbus.Embedding{
					SourceType: "task",
					SourceID:   relatedID,
					Content:    relatedContent,
				},
				Similarity: 0.8,
			},
		},
	}

	mockClar := &mockClarificationBus{countResult: 0}

	mockAnalyzer := &mockGapAnalyzer{
		analysis: GapAnalysis{
			Gaps: []GapCandidate{
				{
					Category:   CategoryMissingDetail,
					Question:   "Need more info",
					Confidence: 0.7,
				},
			},
		},
		captureReceived: true,
	}

	b := New(log, mockClar, mockEmbed, mockAnalyzer)
	result, err := b.Detect(context.Background(), "note", entityID, "test content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.CardsCreated != 1 {
		t.Errorf("expected CardsCreated=1, got %d", result.CardsCreated)
	}

	if len(mockAnalyzer.receivedSummaries) != 1 {
		t.Fatalf("expected 1 related summary, got %d", len(mockAnalyzer.receivedSummaries))
	}

	summary := mockAnalyzer.receivedSummaries[0]
	if summary.Content != relatedContent {
		t.Errorf("expected Content='%s', got '%s'", relatedContent, summary.Content)
	}
}
