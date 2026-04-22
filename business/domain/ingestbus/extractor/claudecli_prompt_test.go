package extractor

import (
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
