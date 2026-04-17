package commands

import (
	"context"
	"testing"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/embeddingbus"
	"github.com/casebrophy/planner/business/domain/knowledgegapbus"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/google/uuid"
)

// mockGapAnalyzer implements knowledgegapbus.GapAnalyzer and tracks call counts.
type mockGapAnalyzer struct {
	callCount int
}

func (m *mockGapAnalyzer) AnalyzeGaps(ctx context.Context, entityContent string, relatedSummaries []knowledgegapbus.RelatedEntitySummary) (knowledgegapbus.GapAnalysis, error) {
	m.callCount++
	// Return empty gaps - we just want to test that it was called
	return knowledgegapbus.GapAnalysis{Gaps: []knowledgegapbus.GapCandidate{}}, nil
}

// stubEmbeddingSearcher returns a single above-threshold related entity so
// knowledgegapbus.Detect proceeds past its "no related entities" early-return
// and invokes the analyzer. Avoids wiring a real embedder in tests.
type stubEmbeddingSearcher struct{}

func (stubEmbeddingSearcher) Search(ctx context.Context, query string, sourceTypes []string, limit int) ([]embeddingbus.SearchResult, error) {
	return []embeddingbus.SearchResult{
		{
			Embedding: embeddingbus.Embedding{
				ID:         uuid.New(),
				SourceType: "task",
				SourceID:   uuid.New(),
				Content:    "stub related entity",
			},
			Similarity: 0.9,
		},
	}, nil
}

func TestGapBackfillCommandDryRun(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "TestGapBackfillCommandDryRun")
	ctx := context.Background()

	// Create a test context (required for notes)
	testCtx, err := db.BusDomain.Context.Create(ctx, contextbus.NewContext{
		Title:       "Test Context",
		Description: "Test context for gap backfill",
	})
	if err != nil {
		t.Fatalf("creating test context: %v", err)
	}

	// Seed 2 tasks using taskbus.TestSeedTasks
	tasks, err := taskbus.TestSeedTasks(ctx, 2, db.BusDomain.Task)
	if err != nil {
		t.Fatalf("seeding tasks: %v", err)
	}

	// Seed 1 note using notebus.TestSeedNotes
	notes, err := notebus.TestSeedNotes(ctx, 1, db.BusDomain.Note, testCtx.ID)
	if err != nil {
		t.Fatalf("seeding notes: %v", err)
	}

	// Verify seeding worked
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(notes))
	}

	// Create mock gap analyzer to track calls
	mockAnalyzer := &mockGapAnalyzer{}

	// Wire up gap bus with stub embedding searcher so Detect() reaches the
	// analyzer without needing a real embedder. Pairs with the mock analyzer.
	gapBus := knowledgegapbus.New(db.Log, db.BusDomain.Clarification, stubEmbeddingSearcher{}, mockAnalyzer, knowledgegapbus.Config{})

	// Run the command with --dry-run and --limit=10
	cmd := &GapBackfillCmd{}
	args := []string{"--dry-run", "--limit=10"}
	err = cmd.Run(ctx, db.Log, db.BusDomain.Task, db.BusDomain.Event, db.BusDomain.Note, db.BusDomain.Context, gapBus, args)
	if err != nil {
		t.Fatalf("running gap-backfill command: %v", err)
	}

	// Assert: Analyzer was called exactly 4 times (2 tasks + 1 note + 1 context).
	// The test context created above for the note is also picked up by the
	// backfill's context sweep. No events were seeded.
	if mockAnalyzer.callCount != 4 {
		t.Errorf("expected analyzer.AnalyzeGaps to be called 4 times, got %d", mockAnalyzer.callCount)
	}

	// Assert: No clarification rows were created (dry-run mode skips card creation)
	clarifications, err := db.BusDomain.Clarification.Query(ctx, clarificationbus.QueryFilter{}, clarificationbus.DefaultOrderBy, page.New(1, 100))
	if err != nil {
		t.Fatalf("querying clarifications: %v", err)
	}

	if len(clarifications) != 0 {
		t.Errorf("expected 0 clarification cards created in dry-run mode, got %d", len(clarifications))
	}
}
