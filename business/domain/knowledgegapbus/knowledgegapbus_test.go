package knowledgegapbus

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/embeddingbus"
	"github.com/casebrophy/planner/business/types/gapcategory"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/google/uuid"
)

type mockEmbeddingBus struct {
	results []embeddingbus.SearchResult
	err     error
}

func (m *mockEmbeddingBus) Search(ctx context.Context, query string, sourceTypes []string, limit int) ([]embeddingbus.SearchResult, error) {
	return m.results, m.err
}

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

type mockGapAnalyzer struct {
	analysis          GapAnalysis
	err               error
	receivedSummaries []RelatedEntitySummary
	captureReceived   bool
}

func (m *mockGapAnalyzer) AnalyzeGaps(ctx context.Context, entityContent string, relatedSummaries []RelatedEntitySummary) (GapAnalysis, error) {
	if m.captureReceived {
		m.receivedSummaries = relatedSummaries
	}
	return m.analysis, m.err
}

func newTestBusiness(t *testing.T, embed *mockEmbeddingBus, clar *mockClarificationBus, analyzer *mockGapAnalyzer) *Business {
	t.Helper()
	buf := &bytes.Buffer{}
	log := logger.New(buf, slog.LevelDebug, "test")
	return New(log, clar, embed, analyzer, Config{})
}

func TestDetect_NoRelatedEntities(t *testing.T) {
	mockEmbed := &mockEmbeddingBus{results: []embeddingbus.SearchResult{}}
	mockClar := &mockClarificationBus{}
	mockAnalyzer := &mockGapAnalyzer{}

	b := newTestBusiness(t, mockEmbed, mockClar, mockAnalyzer)
	result, err := b.Detect(context.Background(), "task", uuid.New(), "test content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.CardsCreated != 0 || result.Skipped != 0 {
		t.Errorf("expected empty result, got CardsCreated=%d, Skipped=%d", result.CardsCreated, result.Skipped)
	}
}

func TestDetect_CreatesCard(t *testing.T) {
	entityID := uuid.New()
	relatedID := uuid.New()

	mockEmbed := &mockEmbeddingBus{
		results: []embeddingbus.SearchResult{
			{Embedding: embeddingbus.Embedding{SourceType: "context", SourceID: relatedID}, Similarity: 0.8},
		},
	}
	mockClar := &mockClarificationBus{}
	mockAnalyzer := &mockGapAnalyzer{
		analysis: GapAnalysis{Gaps: []GapCandidate{
			{Category: gapcategory.MissingContact, Question: "What is the contact?", Reasoning: "No contact info", Confidence: 0.8, RelatedIDs: []string{relatedID.String()}},
		}},
	}

	b := newTestBusiness(t, mockEmbed, mockClar, mockAnalyzer)
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
	if mockClar.created[0].Question != "What is the contact?" {
		t.Errorf("unexpected Question: %s", mockClar.created[0].Question)
	}
	if mockClar.created[0].GapCategory != "missing_contact" {
		t.Errorf("expected GapCategory=missing_contact, got %s", mockClar.created[0].GapCategory)
	}
}

func TestDetect_LowConfidenceSkipped(t *testing.T) {
	relatedID := uuid.New()
	mockEmbed := &mockEmbeddingBus{
		results: []embeddingbus.SearchResult{
			{Embedding: embeddingbus.Embedding{SourceType: "context", SourceID: relatedID}, Similarity: 0.8},
		},
	}
	mockClar := &mockClarificationBus{}
	mockAnalyzer := &mockGapAnalyzer{
		analysis: GapAnalysis{Gaps: []GapCandidate{
			{Category: gapcategory.MissingContact, Question: "contact?", Reasoning: "no info", Confidence: 0.4},
		}},
	}

	b := newTestBusiness(t, mockEmbed, mockClar, mockAnalyzer)
	result, err := b.Detect(context.Background(), "task", uuid.New(), "test content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.CardsCreated != 0 || result.Skipped != 1 {
		t.Errorf("expected Skipped=1, got CardsCreated=%d Skipped=%d", result.CardsCreated, result.Skipped)
	}
}

func TestDetect_MultiGap(t *testing.T) {
	entityID := uuid.New()
	relatedID := uuid.New()

	mockEmbed := &mockEmbeddingBus{
		results: []embeddingbus.SearchResult{
			{Embedding: embeddingbus.Embedding{SourceType: "task", SourceID: relatedID}, Similarity: 0.9},
		},
	}
	mockClar := &mockClarificationBus{}
	mockAnalyzer := &mockGapAnalyzer{
		analysis: GapAnalysis{Gaps: []GapCandidate{
			{Category: gapcategory.MissingContact, Question: "contact?", Reasoning: "r1", Confidence: 0.8},
			{Category: gapcategory.MissingLocation, Question: "location?", Reasoning: "r2", Confidence: 0.75},
		}},
	}

	b := newTestBusiness(t, mockEmbed, mockClar, mockAnalyzer)
	result, err := b.Detect(context.Background(), "task", entityID, "test content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.CardsCreated != 2 {
		t.Errorf("expected CardsCreated=2, got %d", result.CardsCreated)
	}
	if len(mockClar.created) != 2 {
		t.Fatalf("expected 2 created cards, got %d", len(mockClar.created))
	}
	if mockClar.created[0].GapCategory != "missing_contact" {
		t.Errorf("expected first card GapCategory=missing_contact, got %s", mockClar.created[0].GapCategory)
	}
	if mockClar.created[1].GapCategory != "missing_location" {
		t.Errorf("expected second card GapCategory=missing_location, got %s", mockClar.created[1].GapCategory)
	}
}

func TestDetect_ThresholdBoundary(t *testing.T) {
	// Gap at exactly the threshold should be skipped (gap.Confidence <= threshold).
	relatedID := uuid.New()
	mockEmbed := &mockEmbeddingBus{
		results: []embeddingbus.SearchResult{
			{Embedding: embeddingbus.Embedding{SourceType: "task", SourceID: relatedID}, Similarity: 0.8},
		},
	}
	mockClar := &mockClarificationBus{}
	mockAnalyzer := &mockGapAnalyzer{
		analysis: GapAnalysis{Gaps: []GapCandidate{
			{Category: gapcategory.MissingDetail, Question: "q?", Reasoning: "r", Confidence: 0.6}, // exactly at threshold → skipped
		}},
	}

	buf := &bytes.Buffer{}
	log := logger.New(buf, slog.LevelDebug, "test")
	b := New(log, mockClar, mockEmbed, mockAnalyzer, Config{ConfidenceThreshold: 0.6})
	result, err := b.Detect(context.Background(), "task", uuid.New(), "content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.CardsCreated != 0 || result.Skipped != 1 {
		t.Errorf("expected Skipped=1 at exact threshold, got CardsCreated=%d Skipped=%d", result.CardsCreated, result.Skipped)
	}
}

func TestDetect_DryRun(t *testing.T) {
	relatedID := uuid.New()
	mockEmbed := &mockEmbeddingBus{
		results: []embeddingbus.SearchResult{
			{Embedding: embeddingbus.Embedding{SourceType: "task", SourceID: relatedID}, Similarity: 0.8},
		},
	}
	mockClar := &mockClarificationBus{}
	mockAnalyzer := &mockGapAnalyzer{
		analysis: GapAnalysis{Gaps: []GapCandidate{
			{Category: gapcategory.MissingContact, Question: "contact?", Reasoning: "r", Confidence: 0.9},
			{Category: gapcategory.MissingLocation, Question: "where?", Reasoning: "r2", Confidence: 0.85},
		}},
	}

	b := newTestBusiness(t, mockEmbed, mockClar, mockAnalyzer)
	result, err := b.DetectWithOptions(context.Background(), "task", uuid.New(), "content", DetectOptions{DryRun: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.CardsCreated != 2 {
		t.Errorf("expected CardsCreated=2 in dry-run, got %d", result.CardsCreated)
	}
	if len(mockClar.created) != 0 {
		t.Errorf("expected no cards written in dry-run, got %d", len(mockClar.created))
	}
}

func TestDetect_PerCandidateRelatedEntity(t *testing.T) {
	entityID := uuid.New()
	relatedID1 := uuid.New()
	relatedID2 := uuid.New()

	mockEmbed := &mockEmbeddingBus{
		results: []embeddingbus.SearchResult{
			{Embedding: embeddingbus.Embedding{SourceType: "task", SourceID: relatedID1, Content: "content1"}, Similarity: 0.9},
			{Embedding: embeddingbus.Embedding{SourceType: "note", SourceID: relatedID2, Content: "content2"}, Similarity: 0.8},
		},
	}
	mockClar := &mockClarificationBus{}
	mockAnalyzer := &mockGapAnalyzer{
		analysis: GapAnalysis{Gaps: []GapCandidate{
			// Gap 1 references relatedID2 → should use note entity
			{Category: gapcategory.MissingLocation, Question: "where?", Reasoning: "r", Confidence: 0.85, RelatedIDs: []string{relatedID2.String()}},
			// Gap 2 references unknown ID → falls back to filtered[0] (relatedID1)
			{Category: gapcategory.MissingContact, Question: "who?", Reasoning: "r2", Confidence: 0.9, RelatedIDs: []string{uuid.New().String()}},
		}},
	}

	b := newTestBusiness(t, mockEmbed, mockClar, mockAnalyzer)
	result, err := b.Detect(context.Background(), "task", entityID, "content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.CardsCreated != 2 {
		t.Fatalf("expected 2 cards, got %d", result.CardsCreated)
	}

	// Unmarshal answer_options for both cards to check RelatedEntityType/ID
	var opts0, opts1 struct {
		RelatedEntityType string `json:"related_entity_type"`
		RelatedEntityID   string `json:"related_entity_id"`
	}
	if err := json.Unmarshal(mockClar.created[0].AnswerOptions, &opts0); err != nil {
		t.Fatalf("unmarshal opts0: %v", err)
	}
	if err := json.Unmarshal(mockClar.created[1].AnswerOptions, &opts1); err != nil {
		t.Fatalf("unmarshal opts1: %v", err)
	}

	// Gap 1 (missing_location) references relatedID2 → RelatedEntityType should be "note"
	if opts0.RelatedEntityType != "note" || opts0.RelatedEntityID != relatedID2.String() {
		t.Errorf("gap0 expected note/%s, got %s/%s", relatedID2, opts0.RelatedEntityType, opts0.RelatedEntityID)
	}
	// Gap 2 falls back to filtered[0] which is relatedID1 → RelatedEntityType should be "task"
	if opts1.RelatedEntityType != "task" || opts1.RelatedEntityID != relatedID1.String() {
		t.Errorf("gap1 expected task/%s (fallback), got %s/%s", relatedID1, opts1.RelatedEntityType, opts1.RelatedEntityID)
	}
}

func TestDetect_DedupesNearIdenticalRelatedEntities(t *testing.T) {
	// 8 near-identical embedding results (recurring task siblings) with minor
	// case/whitespace variations. All have similarity > 0.5 (default threshold).
	variations := []string{
		"Walk Daily",
		"walk daily",
		"  Walk Daily  ",
		"WALK   DAILY",
		"Walk daily",
		"walk  daily",
		"Walk\tDaily",
		"walk daily ",
	}
	sims := []float64{0.72, 0.81, 0.75, 0.90, 0.78, 0.83, 0.77, 0.88}
	results := make([]embeddingbus.SearchResult, len(variations))
	for i, v := range variations {
		results[i] = embeddingbus.SearchResult{
			Embedding: embeddingbus.Embedding{
				SourceType: "task",
				SourceID:   uuid.New(),
				Content:    v,
			},
			Similarity: sims[i],
		}
	}

	mockEmbed := &mockEmbeddingBus{results: results}
	mockClar := &mockClarificationBus{}
	mockAnalyzer := &mockGapAnalyzer{
		analysis:        GapAnalysis{Gaps: []GapCandidate{}},
		captureReceived: true,
	}

	b := newTestBusiness(t, mockEmbed, mockClar, mockAnalyzer)
	_, err := b.Detect(context.Background(), "task", uuid.New(), "Walk Daily")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mockAnalyzer.receivedSummaries) != 1 {
		t.Fatalf("expected 1 deduped summary, got %d", len(mockAnalyzer.receivedSummaries))
	}
	// The retained summary must have the highest similarity (0.90).
	if mockAnalyzer.receivedSummaries[0].Similarity != 0.90 {
		t.Errorf("expected retained similarity=0.90 (max), got %v", mockAnalyzer.receivedSummaries[0].Similarity)
	}
}

func TestDedupeByContent_DistinctContentPreserved(t *testing.T) {
	results := []embeddingbus.SearchResult{
		{Embedding: embeddingbus.Embedding{SourceType: "task", SourceID: uuid.New(), Content: "Walk the dog"}, Similarity: 0.8},
		{Embedding: embeddingbus.Embedding{SourceType: "task", SourceID: uuid.New(), Content: "Call the dentist"}, Similarity: 0.75},
		{Embedding: embeddingbus.Embedding{SourceType: "note", SourceID: uuid.New(), Content: "Buy groceries"}, Similarity: 0.9},
	}

	out := dedupeByContent(results)
	if len(out) != 3 {
		t.Fatalf("expected 3 distinct entries preserved, got %d", len(out))
	}
}

func TestDetect_ContentPassedToAnalyzer(t *testing.T) {
	entityID := uuid.New()
	relatedID := uuid.New()
	relatedContent := "This is related content from the search result"

	mockEmbed := &mockEmbeddingBus{
		results: []embeddingbus.SearchResult{
			{Embedding: embeddingbus.Embedding{SourceType: "task", SourceID: relatedID, Content: relatedContent}, Similarity: 0.8},
		},
	}
	mockClar := &mockClarificationBus{}
	mockAnalyzer := &mockGapAnalyzer{
		analysis: GapAnalysis{Gaps: []GapCandidate{
			{Category: gapcategory.MissingDetail, Question: "Need more info", Confidence: 0.7},
		}},
		captureReceived: true,
	}

	b := newTestBusiness(t, mockEmbed, mockClar, mockAnalyzer)
	result, err := b.Detect(context.Background(), "note", entityID, "test content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.CardsCreated != 1 {
		t.Errorf("expected CardsCreated=1, got %d", result.CardsCreated)
	}
	if len(mockAnalyzer.receivedSummaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(mockAnalyzer.receivedSummaries))
	}
	if mockAnalyzer.receivedSummaries[0].Content != relatedContent {
		t.Errorf("expected Content=%q, got %q", relatedContent, mockAnalyzer.receivedSummaries[0].Content)
	}
}

func TestBuildExistingKnowledgeSummary_EmptySummaries(t *testing.T) {
	result := buildExistingKnowledgeSummary([]RelatedEntitySummary{})
	if result != "" {
		t.Errorf("expected empty string for empty summaries, got %q", result)
	}
}

func TestBuildExistingKnowledgeSummary_SingleSummary(t *testing.T) {
	summaries := []RelatedEntitySummary{
		{SourceType: "task", Content: "Buy groceries", Similarity: 0.85},
	}
	result := buildExistingKnowledgeSummary(summaries)

	if !strings.Contains(result, "Found 1 related items:") {
		t.Errorf("expected header, got %q", result)
	}
	if !strings.Contains(result, "task") {
		t.Errorf("expected 'task', got %q", result)
	}
	if !strings.Contains(result, "85%") {
		t.Errorf("expected 85%% match in output, got %q", result)
	}
	if !strings.Contains(result, "Buy groceries") {
		t.Errorf("expected content snippet, got %q", result)
	}
}

func TestBuildExistingKnowledgeSummary_MultipleSummaries(t *testing.T) {
	summaries := []RelatedEntitySummary{
		{SourceType: "task", Content: "Item 1", Similarity: 0.9},
		{SourceType: "note", Content: "Item 2", Similarity: 0.75},
		{SourceType: "context", Content: "Item 3", Similarity: 0.65},
	}
	result := buildExistingKnowledgeSummary(summaries)

	if !strings.Contains(result, "Found 3 related items:") {
		t.Errorf("expected header with count 3, got %q", result)
	}
	if !strings.Contains(result, "task") || !strings.Contains(result, "note") || !strings.Contains(result, "context") {
		t.Errorf("expected all entity types, got %q", result)
	}
	if !strings.Contains(result, "90%") || !strings.Contains(result, "75%") || !strings.Contains(result, "65%") {
		t.Errorf("expected all similarity percentages, got %q", result)
	}
}

func TestBuildExistingKnowledgeSummary_LongContent(t *testing.T) {
	longContent := "This is a very long piece of content that should be truncated to approximately sixty characters or so"
	summaries := []RelatedEntitySummary{
		{SourceType: "task", Content: longContent, Similarity: 0.8},
	}
	result := buildExistingKnowledgeSummary(summaries)

	if strings.Contains(result, longContent) {
		t.Errorf("expected content to be truncated, got %q", result)
	}
	if !strings.Contains(result, "...") {
		t.Errorf("expected truncation indicator '...', got %q", result)
	}
	// The snippet should be roughly bounded.
	lines := strings.Split(result, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(lines))
	}
	snippetLine := lines[1]
	if len(snippetLine) > 150 {
		t.Errorf("snippet line too long (%d chars), got %q", len(snippetLine), snippetLine)
	}
}

func TestBuildExistingKnowledgeSummary_SpecialCharacters(t *testing.T) {
	summaries := []RelatedEntitySummary{
		{SourceType: "note", Content: "Call \"John\" about \"project\"", Similarity: 0.72},
	}
	result := buildExistingKnowledgeSummary(summaries)

	// The quotes should be escaped/preserved in the output.
	if !strings.Contains(result, "John") {
		t.Errorf("expected content with special characters preserved, got %s", result)
	}
}
