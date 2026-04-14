package extractor

import (
	"strings"
	"testing"
	"time"
)

func TestBuildTextExtractionPrompt_WithCorrection(t *testing.T) {
	correction := "treat leather on shoes"
	prompt := BuildTextExtractionPrompt("treat leather on blundstones", correction, []byte("[]"), time.Now(), "")
	if !strings.Contains(prompt, correction) {
		t.Errorf("expected prompt to contain correction %q, got: %s", correction, prompt)
	}
	if !strings.Contains(prompt, "IMPORTANT") {
		t.Errorf("expected prompt to contain IMPORTANT preamble")
	}
}

func TestBuildTextExtractionPrompt_WithoutCorrection(t *testing.T) {
	prompt := BuildTextExtractionPrompt("buy groceries", "", []byte("[]"), time.Now(), "")
	if strings.Contains(prompt, "IMPORTANT") {
		t.Errorf("expected no IMPORTANT preamble when correction is empty")
	}
}

func TestBuildEmailExtractionPrompt_WithCorrection(t *testing.T) {
	correction := "this is about the project deadline"
	prompt := BuildEmailExtractionPrompt("from@example.com", "Subject", "Body", correction, []byte("[]"))
	if !strings.Contains(prompt, correction) {
		t.Errorf("expected prompt to contain correction")
	}
	if !strings.Contains(prompt, "IMPORTANT") {
		t.Errorf("expected IMPORTANT preamble")
	}
}

func TestBuildEmailExtractionPrompt_WithoutCorrection(t *testing.T) {
	prompt := BuildEmailExtractionPrompt("from@example.com", "Subject", "Body", "", []byte("[]"))
	if strings.Contains(prompt, "IMPORTANT") {
		t.Errorf("expected no IMPORTANT preamble when correction is empty")
	}
}

func TestBuildTextExtractionPrompt_WithCorrection_TaskHint(t *testing.T) {
	correction := "finish the report by EOD"
	prompt := BuildTextExtractionPrompt("finish the report tomorrow", correction, []byte("[]"), time.Now(), "task")
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
	prompt := BuildTextExtractionPrompt("dentist at 2pm Thursday", correction, []byte("[]"), time.Now(), "event")
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
