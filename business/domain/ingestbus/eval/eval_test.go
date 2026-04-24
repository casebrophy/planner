package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
	"github.com/casebrophy/planner/foundation/claudecli"
	"github.com/casebrophy/planner/foundation/logger"
)

// =============================================================================
// Mock Extractor

type mockExtractor struct {
	result extractor.TextExtraction
	err    error
}

func (m *mockExtractor) ExtractText(_ context.Context, _, _ string, _ []extractor.ContextRef, _ string, _ float64, _ []extractor.EntityMatch, _ []string) (extractor.TextExtraction, error) {
	return m.result, m.err
}

func (m *mockExtractor) ExtractEmail(_ context.Context, _, _, _, _ string, _ []extractor.ContextRef) (extractor.EmailExtraction, error) {
	return extractor.EmailExtraction{}, nil
}

func (m *mockExtractor) ExtractReceipt(_ context.Context, _ string) (extractor.ReceiptExtraction, error) {
	return extractor.ReceiptExtraction{}, nil
}

func (m *mockExtractor) AnalyzeGaps(_ context.Context, _, _ string, _ []extractor.RelatedEntity) (extractor.GapAnalysis, error) {
	return extractor.GapAnalysis{}, nil
}

// =============================================================================
// Helpers

func strPtr(s string) *string { return &s }

func runSingle(fixture Fixture, result extractor.TextExtraction) FixtureResult {
	ext := &mockExtractor{result: result}
	results := Run(context.Background(), ext, []Fixture{fixture})
	return results[0]
}

// =============================================================================
// TestRunnerAssertions — unit test, no external deps

func TestRunnerAssertions(t *testing.T) {
	t.Run("primary_type task pass", func(t *testing.T) {
		f := Fixture{
			Name:      "pt-task-pass",
			InputText: "call mom",
			Expected:  FixtureExpected{PrimaryType: "task"},
		}
		ex := extractor.TextExtraction{
			ActionItems: []extractor.ActionItem{{Title: "call mom"}},
		}
		r := runSingle(f, ex)
		if !r.Pass {
			t.Errorf("expected pass, got failures: %v", r.Failures)
		}
	})

	t.Run("primary_type task fail", func(t *testing.T) {
		f := Fixture{
			Name:      "pt-task-fail",
			InputText: "call mom",
			Expected:  FixtureExpected{PrimaryType: "task"},
		}
		ex := extractor.TextExtraction{
			Notes: []extractor.ExtractedNote{{Content: "info"}},
		}
		r := runSingle(f, ex)
		if r.Pass {
			t.Error("expected fail")
		}
		if !hasFailurePrefix(r.Failures, "primary_type") {
			t.Errorf("expected primary_type failure, got: %v", r.Failures)
		}
	})

	t.Run("primary_type note pass", func(t *testing.T) {
		f := Fixture{
			Name:      "pt-note-pass",
			InputText: "the code is 1234",
			Expected:  FixtureExpected{PrimaryType: "note"},
		}
		ex := extractor.TextExtraction{
			Notes: []extractor.ExtractedNote{{Content: "the code is 1234"}},
		}
		r := runSingle(f, ex)
		if !r.Pass {
			t.Errorf("expected pass, got: %v", r.Failures)
		}
	})

	t.Run("primary_type note fail", func(t *testing.T) {
		f := Fixture{
			Name:      "pt-note-fail",
			InputText: "the code is 1234",
			Expected:  FixtureExpected{PrimaryType: "note"},
		}
		ex := extractor.TextExtraction{
			ActionItems: []extractor.ActionItem{{Title: "check code"}},
		}
		r := runSingle(f, ex)
		if r.Pass {
			t.Error("expected fail")
		}
	})

	t.Run("title_contains pass", func(t *testing.T) {
		f := Fixture{
			Name:      "tc-pass",
			InputText: "buy dish soap",
			Expected:  FixtureExpected{TitleContains: []string{"dish soap"}},
		}
		ex := extractor.TextExtraction{
			ActionItems: []extractor.ActionItem{{Title: "Buy dish soap at the store"}},
		}
		r := runSingle(f, ex)
		if !r.Pass {
			t.Errorf("expected pass, got: %v", r.Failures)
		}
	})

	t.Run("title_contains fail", func(t *testing.T) {
		f := Fixture{
			Name:      "tc-fail",
			InputText: "buy dish soap",
			Expected:  FixtureExpected{TitleContains: []string{"dish soap"}},
		}
		ex := extractor.TextExtraction{
			ActionItems: []extractor.ActionItem{{Title: "buy groceries"}},
		}
		r := runSingle(f, ex)
		if r.Pass {
			t.Error("expected fail")
		}
		if !hasFailurePrefix(r.Failures, "title_contains") {
			t.Errorf("expected title_contains failure, got: %v", r.Failures)
		}
	})

	t.Run("title_contains multi-term pass", func(t *testing.T) {
		f := Fixture{
			Name:      "tc-multi-pass",
			InputText: "book oil change",
			Expected:  FixtureExpected{TitleContains: []string{"oil", "change"}},
		}
		ex := extractor.TextExtraction{
			ActionItems: []extractor.ActionItem{{Title: "Book oil change appointment"}},
		}
		r := runSingle(f, ex)
		if !r.Pass {
			t.Errorf("expected pass, got: %v", r.Failures)
		}
	})

	t.Run("title_contains in event pass", func(t *testing.T) {
		f := Fixture{
			Name:      "tc-event-pass",
			InputText: "dentist at 2pm",
			Expected:  FixtureExpected{TitleContains: []string{"dentist"}},
		}
		ex := extractor.TextExtraction{
			Events: []extractor.ExtractedEvent{{Title: "Dentist appointment", StartsAt: "2pm"}},
		}
		r := runSingle(f, ex)
		if !r.Pass {
			t.Errorf("expected pass, got: %v", r.Failures)
		}
	})

	t.Run("context_id exact pass", func(t *testing.T) {
		f := Fixture{
			Name:           "ctx-exact-pass",
			InputText:      "buy milk",
			ActiveContexts: []FixtureContext{{ID: "ctx-shop", Title: "Shopping", Kind: "list"}},
			Expected:       FixtureExpected{ContextID: "ctx-shop"},
		}
		ex := extractor.TextExtraction{
			ActionItems:       []extractor.ActionItem{{Title: "buy milk"}},
			SuggestedContextID: strPtr("ctx-shop"),
		}
		r := runSingle(f, ex)
		if !r.Pass {
			t.Errorf("expected pass, got: %v", r.Failures)
		}
	})

	t.Run("context_id exact fail", func(t *testing.T) {
		f := Fixture{
			Name:           "ctx-exact-fail",
			InputText:      "buy milk",
			ActiveContexts: []FixtureContext{{ID: "ctx-shop", Title: "Shopping", Kind: "list"}},
			Expected:       FixtureExpected{ContextID: "ctx-shop"},
		}
		ex := extractor.TextExtraction{
			ActionItems:       []extractor.ActionItem{{Title: "buy milk"}},
			SuggestedContextID: strPtr("ctx-other"),
		}
		r := runSingle(f, ex)
		if r.Pass {
			t.Error("expected fail")
		}
		if !hasFailurePrefix(r.Failures, "context_id") {
			t.Errorf("expected context_id failure, got: %v", r.Failures)
		}
	})

	t.Run("context_id nil fail", func(t *testing.T) {
		f := Fixture{
			Name:           "ctx-nil-fail",
			InputText:      "buy milk",
			ActiveContexts: []FixtureContext{{ID: "ctx-shop", Title: "Shopping", Kind: "list"}},
			Expected:       FixtureExpected{ContextID: "ctx-shop"},
		}
		ex := extractor.TextExtraction{
			ActionItems: []extractor.ActionItem{{Title: "buy milk"}},
		}
		r := runSingle(f, ex)
		if r.Pass {
			t.Error("expected fail")
		}
	})

	t.Run("context_id new pass", func(t *testing.T) {
		f := Fixture{
			Name:      "ctx-new-pass",
			InputText: "things to pack: passport, chargers",
			Expected:  FixtureExpected{ContextID: "new"},
		}
		ex := extractor.TextExtraction{
			ActionItems:      []extractor.ActionItem{{Title: "pack passport"}, {Title: "pack chargers"}},
			SuggestNewContext: true,
		}
		r := runSingle(f, ex)
		if !r.Pass {
			t.Errorf("expected pass, got: %v", r.Failures)
		}
	})

	t.Run("context_id new fail", func(t *testing.T) {
		f := Fixture{
			Name:      "ctx-new-fail",
			InputText: "things to pack: passport",
			Expected:  FixtureExpected{ContextID: "new"},
		}
		ex := extractor.TextExtraction{
			ActionItems:      []extractor.ActionItem{{Title: "pack passport"}},
			SuggestNewContext: false,
		}
		r := runSingle(f, ex)
		if r.Pass {
			t.Error("expected fail")
		}
		if !hasFailurePrefix(r.Failures, "context_id(new)") {
			t.Errorf("expected context_id(new) failure, got: %v", r.Failures)
		}
	})

	t.Run("context_kind pass", func(t *testing.T) {
		f := Fixture{
			Name:           "ck-pass",
			InputText:      "buy milk",
			ActiveContexts: []FixtureContext{{ID: "ctx-shop", Title: "Shopping", Kind: "list"}},
			Expected:       FixtureExpected{ContextID: "ctx-shop", ContextKind: "list"},
		}
		ex := extractor.TextExtraction{
			ActionItems:       []extractor.ActionItem{{Title: "buy milk"}},
			SuggestedContextID: strPtr("ctx-shop"),
		}
		r := runSingle(f, ex)
		if !r.Pass {
			t.Errorf("expected pass, got: %v", r.Failures)
		}
	})

	t.Run("context_kind fail", func(t *testing.T) {
		f := Fixture{
			Name:           "ck-fail",
			InputText:      "buy milk",
			ActiveContexts: []FixtureContext{{ID: "ctx-shop", Title: "Shopping", Kind: "area"}}, // kind is "area", not "list"
			Expected:       FixtureExpected{ContextID: "ctx-shop", ContextKind: "list"},
		}
		ex := extractor.TextExtraction{
			ActionItems:       []extractor.ActionItem{{Title: "buy milk"}},
			SuggestedContextID: strPtr("ctx-shop"),
		}
		r := runSingle(f, ex)
		if r.Pass {
			t.Error("expected fail")
		}
		if !hasFailurePrefix(r.Failures, "context_kind") {
			t.Errorf("expected context_kind failure, got: %v", r.Failures)
		}
	})

	t.Run("min_action_items pass", func(t *testing.T) {
		f := Fixture{
			Name:      "min-ai-pass",
			InputText: "buy milk, call mom, email bob",
			Expected:  FixtureExpected{MinActionItems: 3},
		}
		ex := extractor.TextExtraction{
			ActionItems: []extractor.ActionItem{
				{Title: "buy milk"},
				{Title: "call mom"},
				{Title: "email bob"},
			},
		}
		r := runSingle(f, ex)
		if !r.Pass {
			t.Errorf("expected pass, got: %v", r.Failures)
		}
	})

	t.Run("min_action_items fail", func(t *testing.T) {
		f := Fixture{
			Name:      "min-ai-fail",
			InputText: "buy milk, call mom, email bob",
			Expected:  FixtureExpected{MinActionItems: 3},
		}
		ex := extractor.TextExtraction{
			ActionItems: []extractor.ActionItem{{Title: "buy milk"}},
		}
		r := runSingle(f, ex)
		if r.Pass {
			t.Error("expected fail")
		}
		if !hasFailurePrefix(r.Failures, "min_action_items") {
			t.Errorf("expected min_action_items failure, got: %v", r.Failures)
		}
	})

	t.Run("max_action_items pass", func(t *testing.T) {
		f := Fixture{
			Name:      "max-ai-pass",
			InputText: "call mom",
			Expected:  FixtureExpected{MaxActionItems: 1},
		}
		ex := extractor.TextExtraction{
			ActionItems: []extractor.ActionItem{{Title: "call mom"}},
		}
		r := runSingle(f, ex)
		if !r.Pass {
			t.Errorf("expected pass, got: %v", r.Failures)
		}
	})

	t.Run("max_action_items fail", func(t *testing.T) {
		f := Fixture{
			Name:      "max-ai-fail",
			InputText: "call mom",
			Expected:  FixtureExpected{MaxActionItems: 1},
		}
		ex := extractor.TextExtraction{
			ActionItems: []extractor.ActionItem{
				{Title: "call mom"},
				{Title: "email mom"},
			},
		}
		r := runSingle(f, ex)
		if r.Pass {
			t.Error("expected fail")
		}
		if !hasFailurePrefix(r.Failures, "max_action_items") {
			t.Errorf("expected max_action_items failure, got: %v", r.Failures)
		}
	})

	t.Run("forbid_notes pass", func(t *testing.T) {
		f := Fixture{
			Name:      "fn-pass",
			InputText: "call mom",
			Expected:  FixtureExpected{ForbidNotes: true},
		}
		ex := extractor.TextExtraction{
			ActionItems: []extractor.ActionItem{{Title: "call mom"}},
		}
		r := runSingle(f, ex)
		if !r.Pass {
			t.Errorf("expected pass, got: %v", r.Failures)
		}
	})

	t.Run("forbid_notes fail", func(t *testing.T) {
		f := Fixture{
			Name:      "fn-fail",
			InputText: "call mom",
			Expected:  FixtureExpected{ForbidNotes: true},
		}
		ex := extractor.TextExtraction{
			Notes: []extractor.ExtractedNote{{Content: "some note"}},
		}
		r := runSingle(f, ex)
		if r.Pass {
			t.Error("expected fail")
		}
		if !hasFailurePrefix(r.Failures, "forbid_notes") {
			t.Errorf("expected forbid_notes failure, got: %v", r.Failures)
		}
	})

	t.Run("forbid_events pass", func(t *testing.T) {
		f := Fixture{
			Name:      "fe-pass",
			InputText: "call mom",
			Expected:  FixtureExpected{ForbidEvents: true},
		}
		ex := extractor.TextExtraction{
			ActionItems: []extractor.ActionItem{{Title: "call mom"}},
		}
		r := runSingle(f, ex)
		if !r.Pass {
			t.Errorf("expected pass, got: %v", r.Failures)
		}
	})

	t.Run("forbid_events fail", func(t *testing.T) {
		f := Fixture{
			Name:      "fe-fail",
			InputText: "meeting at 3pm",
			Expected:  FixtureExpected{ForbidEvents: true},
		}
		ex := extractor.TextExtraction{
			Events: []extractor.ExtractedEvent{{Title: "meeting", StartsAt: "3pm"}},
		}
		r := runSingle(f, ex)
		if r.Pass {
			t.Error("expected fail")
		}
		if !hasFailurePrefix(r.Failures, "forbid_events") {
			t.Errorf("expected forbid_events failure, got: %v", r.Failures)
		}
	})
}

// =============================================================================
// TestClassificationEval — real eval, skips if EVAL_SIDECAR_URL not set

func TestClassificationEval(t *testing.T) {
	sidecarURL := os.Getenv("EVAL_SIDECAR_URL")
	if sidecarURL == "" {
		t.Skip("EVAL_SIDECAR_URL not set")
	}
	sidecarKey := os.Getenv("PLANNER_AUTH_API_KEY")
	log := logger.New(os.Stdout, logger.LevelInfo, "eval-test")
	client := claudecli.NewClient(log, []string{"claude-haiku-4-5-20251001"}, sidecarURL, sidecarKey)
	ext := extractor.NewClaudeCodeExtractor(client)

	fixtures, err := LoadFixtures("testdata/classification")
	if err != nil {
		t.Fatalf("LoadFixtures: %v", err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no fixtures loaded")
	}

	results := Run(context.Background(), ext, fixtures)
	metrics := Score(results)

	t.Logf("TypeAccuracy: %.0f%%  ContextAccuracy: %.0f%%  ListAssignmentRate: %.0f%%",
		metrics.TypeAccuracy*100, metrics.ContextAccuracy*100, metrics.ListAssignmentRate*100)
	for _, r := range results {
		if r.Pass {
			t.Logf("  PASS  %s (%dms)", r.Fixture.Name, r.LatencyMS)
		} else {
			t.Logf("  FAIL  %s (%dms): %v", r.Fixture.Name, r.LatencyMS, r.Failures)
		}
	}

	if err := writeJSONL(results); err != nil {
		t.Logf("warn: failed to write JSONL: %v", err)
	}

	// Check accuracy floors from env vars (default 0.7 each)
	minTypeAccuracy := 0.7
	if v := os.Getenv("EVAL_MIN_TYPE_ACCURACY"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			minTypeAccuracy = f
		} else {
			t.Logf("warn: failed to parse EVAL_MIN_TYPE_ACCURACY=%q: %v", v, err)
		}
	}

	minContextAccuracy := 0.7
	if v := os.Getenv("EVAL_MIN_CONTEXT_ACCURACY"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			minContextAccuracy = f
		} else {
			t.Logf("warn: failed to parse EVAL_MIN_CONTEXT_ACCURACY=%q: %v", v, err)
		}
	}

	if metrics.TypeAccuracy < minTypeAccuracy {
		t.Errorf("TypeAccuracy %.2f below floor %.2f", metrics.TypeAccuracy, minTypeAccuracy)
	}
	if metrics.ContextAccuracy < minContextAccuracy {
		t.Errorf("ContextAccuracy %.2f below floor %.2f", metrics.ContextAccuracy, minContextAccuracy)
	}
}

// writeJSONL writes results to .tmp/eval-<timestamp>.jsonl.
func writeJSONL(results []FixtureResult) error {
	if err := os.MkdirAll(".tmp", 0755); err != nil {
		return fmt.Errorf("mkdir .tmp: %w", err)
	}
	filename := fmt.Sprintf(".tmp/eval-%d.jsonl", time.Now().UnixMilli())
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create %s: %w", filename, err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, r := range results {
		if err := enc.Encode(r); err != nil {
			return fmt.Errorf("encode result: %w", err)
		}
	}
	return nil
}
