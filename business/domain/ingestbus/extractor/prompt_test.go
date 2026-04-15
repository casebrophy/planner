package extractor

import (
	"strings"
	"testing"
	"time"
)

func TestBuildTextExtractionPrompt_WithCorrection(t *testing.T) {
	correction := "treat leather on shoes"
	prompt := BuildTextExtractionPrompt("treat leather on blundstones", correction, []byte("[]"), time.Now(), "", nil)
	if !strings.Contains(prompt, correction) {
		t.Errorf("expected prompt to contain correction %q, got: %s", correction, prompt)
	}
	if !strings.Contains(prompt, "IMPORTANT") {
		t.Errorf("expected prompt to contain IMPORTANT preamble")
	}
}

func TestBuildTextExtractionPrompt_WithoutCorrection(t *testing.T) {
	prompt := BuildTextExtractionPrompt("buy groceries", "", []byte("[]"), time.Now(), "", nil)
	if strings.Contains(prompt, "IMPORTANT") {
		t.Errorf("expected no IMPORTANT preamble when correction is empty")
	}
}

func TestBuildEmailExtractionPrompt_WithCorrection(t *testing.T) {
	correction := "this is about the project deadline"
	prompt := BuildEmailExtractionPrompt("from@example.com", "Subject", "Body", correction, []byte("[]"), nil)
	if !strings.Contains(prompt, correction) {
		t.Errorf("expected prompt to contain correction")
	}
	if !strings.Contains(prompt, "IMPORTANT") {
		t.Errorf("expected IMPORTANT preamble")
	}
}

func TestBuildEmailExtractionPrompt_WithoutCorrection(t *testing.T) {
	prompt := BuildEmailExtractionPrompt("from@example.com", "Subject", "Body", "", []byte("[]"), nil)
	if strings.Contains(prompt, "IMPORTANT") {
		t.Errorf("expected no IMPORTANT preamble when correction is empty")
	}
}

func TestBuildTextExtractionPrompt_WithCorrection_TaskHint(t *testing.T) {
	correction := "finish the report by EOD"
	prompt := BuildTextExtractionPrompt("finish the report tomorrow", correction, []byte("[]"), time.Now(), "task", nil)
	if !strings.Contains(prompt, correction) {
		t.Errorf("expected prompt to contain correction for task hint")
	}
	if !strings.Contains(prompt, "IMPORTANT") {
		t.Errorf("expected IMPORTANT preamble for task hint")
	}
	if !strings.Contains(prompt, "This clause has been classified as a task") {
		t.Errorf("expected task-specific prompt text")
	}
}

func TestBuildTextExtractionPrompt_WithCorrection_EventHint(t *testing.T) {
	correction := "meeting is at 3pm, not 2pm"
	prompt := BuildTextExtractionPrompt("dentist at 2pm Thursday", correction, []byte("[]"), time.Now(), "event", nil)
	if !strings.Contains(prompt, correction) {
		t.Errorf("expected prompt to contain correction for event hint")
	}
	if !strings.Contains(prompt, "IMPORTANT") {
		t.Errorf("expected IMPORTANT preamble for event hint")
	}
	if !strings.Contains(prompt, "This clause has been classified as an event") {
		t.Errorf("expected event-specific prompt text")
	}
}

func TestBuildCandidateBlock_Empty(t *testing.T) {
	if got := BuildCandidateBlock(nil); got != "" {
		t.Errorf("expected empty string for nil candidates, got: %q", got)
	}
	if got := BuildCandidateBlock([]EntityMatch{}); got != "" {
		t.Errorf("expected empty string for empty candidates, got: %q", got)
	}
}

func TestBuildCandidateBlock_WithCandidates(t *testing.T) {
	candidates := []EntityMatch{
		{ID: "abc-123", SourceType: "event", Title: "Ethan's Wedding", Content: "Wedding in May", Similarity: 0.85},
		{ID: "def-456", SourceType: "task", Title: "Buy wedding gift", Content: "Get a gift", Similarity: 0.72},
	}
	got := BuildCandidateBlock(candidates)

	checks := []struct {
		desc string
		want string
	}{
		{"header", "Existing Entities"},
		{"event type upper", "[EVENT]"},
		{"task type upper", "[TASK]"},
		{"event ID", "abc-123"},
		{"event title", "Ethan's Wedding"},
		{"event similarity", "0.85"},
		{"task ID", "def-456"},
		{"task title", "Buy wedding gift"},
	}
	for _, c := range checks {
		if !strings.Contains(got, c.want) {
			t.Errorf("BuildCandidateBlock: expected %s (%q) in output; got:\n%s", c.desc, c.want, got)
		}
	}
}

func TestBuildTextExtractionPrompt_WithCandidates(t *testing.T) {
	candidates := []EntityMatch{
		{ID: "abc-123", SourceType: "event", Title: "Team Meeting", Content: "Weekly standup", Similarity: 0.78},
	}
	prompt := BuildTextExtractionPrompt("move the meeting to 3pm", "", []byte("[]"), time.Now(), "", candidates)
	if !strings.Contains(prompt, "Existing Entities") {
		t.Error("expected candidate block header in text prompt")
	}
	if !strings.Contains(prompt, "abc-123") {
		t.Error("expected candidate ID in text prompt")
	}
	if !strings.Contains(prompt, "Team Meeting") {
		t.Error("expected candidate title in text prompt")
	}
}

func TestBuildEmailExtractionPrompt_WithCandidates(t *testing.T) {
	candidates := []EntityMatch{
		{ID: "xyz-789", SourceType: "task", Title: "Fix login bug", Content: "Login issue", Similarity: 0.81},
	}
	prompt := BuildEmailExtractionPrompt("sender@example.com", "Re: Login bug", "Still broken", "", []byte("[]"), candidates)
	if !strings.Contains(prompt, "Existing Entities") {
		t.Error("expected candidate block header in email prompt")
	}
	if !strings.Contains(prompt, "xyz-789") {
		t.Error("expected candidate ID in email prompt")
	}
}

func TestBuildTextExtractionPrompt_NoCandidates(t *testing.T) {
	prompt := BuildTextExtractionPrompt("buy groceries", "", []byte("[]"), time.Now(), "", nil)
	if strings.Contains(prompt, "Existing Entities") {
		t.Error("expected no candidate block when candidates is nil")
	}
}
