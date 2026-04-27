package extractor

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBuildTextExtractionPrompt(t *testing.T) {
	tests := []struct {
		name       string
		typeHint   string
		confidence float64
		wantCheck  string
	}{
		{
			name:       "empty typeHint uses generic prompt",
			typeHint:   "",
			confidence: 0,
			wantCheck:  "voice capture",
		},
		{
			name:       "task typeHint uses task prompt",
			typeHint:   "task",
			confidence: 0.9,
			wantCheck:  "heuristic classifier suggested this clause is likely a task",
		},
		{
			name:       "event typeHint uses event prompt",
			typeHint:   "event",
			confidence: 0.9,
			wantCheck:  "heuristic classifier suggested this clause is likely an event",
		},
		{
			name:       "note typeHint uses note prompt",
			typeHint:   "note",
			confidence: 0.9,
			wantCheck:  "heuristic classifier suggested this clause is likely a note",
		},
		{
			name:       "unknown typeHint falls back to generic",
			typeHint:   "unknown",
			confidence: 0,
			wantCheck:  "voice capture",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			prompt := BuildTextExtractionPrompt("test clause", "", []byte("[]"), now, tt.typeHint, tt.confidence, nil, nil)

			if !strings.Contains(prompt, tt.wantCheck) {
				t.Errorf("prompt did not contain expected text %q", tt.wantCheck)
			}
		})
	}
}

func TestBuildTextExtractionPrompt_TaskDetails(t *testing.T) {
	now := time.Now()
	prompt := BuildTextExtractionPrompt("call the dentist", "", []byte("[]"), now, "task", 0.9, nil, nil)

	checks := []string{
		"heuristic classifier suggested this clause is likely a task",
		"action-oriented title",
		"Call dentist",
		"reclassified_as",
	}

	for _, check := range checks {
		if !strings.Contains(prompt, check) {
			t.Errorf("task prompt missing expected text: %q", check)
		}
	}
}

func TestBuildTextExtractionPrompt_EventDetails(t *testing.T) {
	now := time.Now()
	prompt := BuildTextExtractionPrompt("dentist at 2pm Thursday", "", []byte("[]"), now, "event", 0.9, nil, nil)

	checks := []string{
		"heuristic classifier suggested this clause is likely an event",
		"fixed commitment",
		"starts_at is required",
		"all_day=true",
	}

	for _, check := range checks {
		if !strings.Contains(prompt, check) {
			t.Errorf("event prompt missing expected text: %q", check)
		}
	}
}

func TestBuildTextExtractionPrompt_NoteDetails(t *testing.T) {
	now := time.Now()
	prompt := BuildTextExtractionPrompt("Mario's is the best pizza", "", []byte("[]"), now, "note", 0.9, nil, nil)

	checks := []string{
		"heuristic classifier suggested this clause is likely a note",
		"reference information",
		"Preserve the user's own words",
		"Suggest 1-3 tags",
	}

	for _, check := range checks {
		if !strings.Contains(prompt, check) {
			t.Errorf("note prompt missing expected text: %q", check)
		}
	}
}

func TestBuildTextExtractionPrompt_GenericFallback(t *testing.T) {
	now := time.Now()
	prompt := BuildTextExtractionPrompt("test clause", "", []byte("[]"), now, "", 0, nil, nil)

	checks := []string{
		"voice capture",
		"Distinguish between tasks",
		"events (fixed commitments",
		"notes (information/knowledge",
	}

	for _, check := range checks {
		if !strings.Contains(prompt, check) {
			t.Errorf("generic prompt missing expected text: %q", check)
		}
	}
}

func TestGapAnalysisCategoriesRoundTrip(t *testing.T) {
	categories := []string{
		"missing_contact",
		"missing_location",
		"missing_detail",
		"missing_dependency",
		"missing_context",
		"missing_deadline",
		"missing_stakeholder",
		"missing_outcome",
	}

	for _, cat := range categories {
		t.Run(cat, func(t *testing.T) {
			gap := GapCandidate{
				Category:   cat,
				Question:   "Test question?",
				Reasoning:  "Test reasoning",
				Confidence: 0.8,
				RelatedIDs: []string{"id1", "id2"},
			}

			data, err := json.Marshal(gap)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			var unmarshaled GapCandidate
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}

			if unmarshaled.Category != cat {
				t.Errorf("category mismatch: got %q, want %q", unmarshaled.Category, cat)
			}
		})
	}
}

func TestGapAnalysisOptionsRoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		options   []string
		optConf   float64
		expectOpt []string
		expectConf float64
	}{
		{
			name:       "enumerable with options",
			options:    []string{"option1", "option2", "option3"},
			optConf:    0.9,
			expectOpt: []string{"option1", "option2", "option3"},
			expectConf: 0.9,
		},
		{
			name:       "open-ended with empty options",
			options:    []string{},
			optConf:    0,
			expectOpt: []string{},
			expectConf: 0,
		},
		{
			name:       "nil options becomes empty array",
			options:    nil,
			optConf:    0,
			expectOpt: []string{},
			expectConf: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gap := GapCandidate{
				Category:          "missing_location",
				Question:          "Where is the meeting?",
				Reasoning:         "Location not specified",
				Confidence:        0.8,
				RelatedIDs:        []string{"id1"},
				Options:           tt.options,
				OptionsConfidence: tt.optConf,
			}

			analysis := GapAnalysis{Gaps: []GapCandidate{gap}}
			data, err := json.Marshal(analysis)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			var unmarshaled GapAnalysis
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}

			if len(unmarshaled.Gaps) != 1 {
				t.Errorf("expected 1 gap, got %d", len(unmarshaled.Gaps))
			}

			got := unmarshaled.Gaps[0]
			if len(got.Options) != len(tt.expectOpt) {
				t.Errorf("options length mismatch: got %d, want %d", len(got.Options), len(tt.expectOpt))
			}

			for i, opt := range got.Options {
				if i >= len(tt.expectOpt) {
					break
				}
				if opt != tt.expectOpt[i] {
					t.Errorf("option[%d] mismatch: got %q, want %q", i, opt, tt.expectOpt[i])
				}
			}

			if got.OptionsConfidence != tt.expectConf {
				t.Errorf("options_confidence mismatch: got %v, want %v", got.OptionsConfidence, tt.expectConf)
			}
		})
	}
}
