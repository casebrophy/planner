package extractor

import (
	"strings"
	"testing"
	"time"
)

func TestBuildTextExtractionPrompt(t *testing.T) {
	tests := []struct {
		name      string
		typeHint  string
		wantCheck string
	}{
		{
			name:      "empty typeHint uses generic prompt",
			typeHint:  "",
			wantCheck: "voice capture",
		},
		{
			name:      "task typeHint uses task prompt",
			typeHint:  "task",
			wantCheck: "classified as a task",
		},
		{
			name:      "event typeHint uses event prompt",
			typeHint:  "event",
			wantCheck: "classified as an event",
		},
		{
			name:      "note typeHint uses note prompt",
			typeHint:  "note",
			wantCheck: "classified as a note",
		},
		{
			name:      "unknown typeHint falls back to generic",
			typeHint:  "unknown",
			wantCheck: "voice capture",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			prompt := BuildTextExtractionPrompt("test clause", "", []byte("[]"), now, tt.typeHint, nil)

			if !strings.Contains(prompt, tt.wantCheck) {
				t.Errorf("prompt did not contain expected text %q", tt.wantCheck)
			}
		})
	}
}

func TestBuildTextExtractionPrompt_TaskDetails(t *testing.T) {
	now := time.Now()
	prompt := BuildTextExtractionPrompt("call the dentist", "", []byte("[]"), now, "task", nil)

	checks := []string{
		"classified as a task",
		"action-oriented title",
		"Call dentist",
		"Do not reclassify",
	}

	for _, check := range checks {
		if !strings.Contains(prompt, check) {
			t.Errorf("task prompt missing expected text: %q", check)
		}
	}
}

func TestBuildTextExtractionPrompt_EventDetails(t *testing.T) {
	now := time.Now()
	prompt := BuildTextExtractionPrompt("dentist at 2pm Thursday", "", []byte("[]"), now, "event", nil)

	checks := []string{
		"classified as an event",
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
	prompt := BuildTextExtractionPrompt("Mario's is the best pizza", "", []byte("[]"), now, "note", nil)

	checks := []string{
		"classified as a note",
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
	prompt := BuildTextExtractionPrompt("test clause", "", []byte("[]"), now, "", nil)

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
