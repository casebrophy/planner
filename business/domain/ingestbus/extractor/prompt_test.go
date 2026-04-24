package extractor

import (
	"strings"
	"testing"
	"time"
)

func TestBuildGapAnalysisPrompt_IncludesScopeConstraints(t *testing.T) {
	prompt := BuildGapAnalysisPrompt("task", "Walk Daily", []RelatedEntity{
		{ID: "abc", SourceType: "task", Title: "Walk Daily", Content: "Walk Daily"},
	})
	if !strings.Contains(prompt, "Do NOT ask about duplicates") {
		t.Errorf("expected prompt to contain 'Do NOT ask about duplicates'")
	}
	if !strings.Contains(prompt, "Every gap MUST be about information missing from the NEW entity") {
		t.Errorf("expected prompt to contain 'Every gap MUST be about information missing from the NEW entity'")
	}
}

func TestBuildTextExtractionPrompt_WithCorrection(t *testing.T) {
	correction := "treat leather on shoes"
	prompt := BuildTextExtractionPrompt("treat leather on blundstones", correction, []byte("[]"), time.Now(), "", 0, nil, nil)
	if !strings.Contains(prompt, correction) {
		t.Errorf("expected prompt to contain correction %q, got: %s", correction, prompt)
	}
	if !strings.Contains(prompt, "IMPORTANT") {
		t.Errorf("expected prompt to contain IMPORTANT preamble")
	}
}

func TestBuildTextExtractionPrompt_WithoutCorrection(t *testing.T) {
	prompt := BuildTextExtractionPrompt("buy groceries", "", []byte("[]"), time.Now(), "", 0, nil, nil)
	if strings.Contains(prompt, "IMPORTANT") {
		t.Errorf("expected no IMPORTANT preamble when correction is empty")
	}
}

func TestBuildEmailExtractionPrompt_WithCorrection(t *testing.T) {
	correction := "this is about the project deadline"
	prompt := BuildEmailExtractionPrompt("from@example.com", "Subject", "Body", correction, []byte("[]"), nil, nil)
	if !strings.Contains(prompt, correction) {
		t.Errorf("expected prompt to contain correction")
	}
	if !strings.Contains(prompt, "IMPORTANT") {
		t.Errorf("expected IMPORTANT preamble")
	}
}

func TestBuildEmailExtractionPrompt_WithoutCorrection(t *testing.T) {
	prompt := BuildEmailExtractionPrompt("from@example.com", "Subject", "Body", "", []byte("[]"), nil, nil)
	if strings.Contains(prompt, "IMPORTANT") {
		t.Errorf("expected no IMPORTANT preamble when correction is empty")
	}
}

func TestBuildTextExtractionPrompt_WithCorrection_TaskHint(t *testing.T) {
	correction := "finish the report by EOD"
	prompt := BuildTextExtractionPrompt("finish the report tomorrow", correction, []byte("[]"), time.Now(), "task", 0.9, nil, nil)
	if !strings.Contains(prompt, correction) {
		t.Errorf("expected prompt to contain correction for task hint")
	}
	if !strings.Contains(prompt, "IMPORTANT") {
		t.Errorf("expected IMPORTANT preamble for task hint")
	}
	if !strings.Contains(prompt, "heuristic classifier suggested this clause is likely a task") {
		t.Errorf("expected task-specific prompt text")
	}
}

func TestBuildTextExtractionPrompt_WithCorrection_EventHint(t *testing.T) {
	correction := "meeting is at 3pm, not 2pm"
	prompt := BuildTextExtractionPrompt("dentist at 2pm Thursday", correction, []byte("[]"), time.Now(), "event", 0.9, nil, nil)
	if !strings.Contains(prompt, correction) {
		t.Errorf("expected prompt to contain correction for event hint")
	}
	if !strings.Contains(prompt, "IMPORTANT") {
		t.Errorf("expected IMPORTANT preamble for event hint")
	}
	if !strings.Contains(prompt, "heuristic classifier suggested this clause is likely an event") {
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
	prompt := BuildTextExtractionPrompt("move the meeting to 3pm", "", []byte("[]"), time.Now(), "", 0, candidates, nil)
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
	prompt := BuildEmailExtractionPrompt("sender@example.com", "Re: Login bug", "Still broken", "", []byte("[]"), candidates, nil)
	if !strings.Contains(prompt, "Existing Entities") {
		t.Error("expected candidate block header in email prompt")
	}
	if !strings.Contains(prompt, "xyz-789") {
		t.Error("expected candidate ID in email prompt")
	}
}

func TestBuildTextExtractionPrompt_NoCandidates(t *testing.T) {
	prompt := BuildTextExtractionPrompt("buy groceries", "", []byte("[]"), time.Now(), "", 0, nil, nil)
	if strings.Contains(prompt, "Existing Entities") {
		t.Error("expected no candidate block when candidates is nil")
	}
}

func TestBuildContextAnnotationsBlock_Empty(t *testing.T) {
	if got := BuildContextAnnotationsBlock(nil); got != "" {
		t.Errorf("expected empty string for nil annotations, got: %q", got)
	}
	if got := BuildContextAnnotationsBlock([]string{}); got != "" {
		t.Errorf("expected empty string for empty annotations, got: %q", got)
	}
}

func TestBuildContextAnnotationsBlock_WithAnnotations(t *testing.T) {
	annotations := []string{
		"User is planning a vacation to Europe",
		"Project deadline is next Friday",
	}
	got := BuildContextAnnotationsBlock(annotations)

	checks := []struct {
		desc string
		want string
	}{
		{"header", "Context Annotations"},
		{"annotation 1", "User is planning a vacation to Europe"},
		{"annotation 2", "Project deadline is next Friday"},
	}
	for _, c := range checks {
		if !strings.Contains(got, c.want) {
			t.Errorf("BuildContextAnnotationsBlock: expected %s (%q) in output; got:\n%s", c.desc, c.want, got)
		}
	}
}

func TestBuildTextExtractionPrompt_ListExpansionRulePresent(t *testing.T) {
	// Generic prompt
	prompt := BuildTextExtractionPrompt("item 1, item 2, item 3", "", []byte("[]"), time.Now(), "", 0, nil, nil)
	if !strings.Contains(prompt, "If the input contains a list with 3+ items") {
		t.Error("expected list-expansion rule in generic text extraction prompt")
	}

	// Task prompt
	prompt = BuildTextExtractionPrompt("buy milk, eggs, bread, and butter", "", []byte("[]"), time.Now(), "task", 0.9, nil, nil)
	if !strings.Contains(prompt, "If the input contains a list with 3+ items") {
		t.Error("expected list-expansion rule in task extraction prompt")
	}

	// Event prompt
	prompt = BuildTextExtractionPrompt("meeting, lunch, presentation", "", []byte("[]"), time.Now(), "event", 0.9, nil, nil)
	if !strings.Contains(prompt, "If the input contains a list with 3+ items") {
		t.Error("expected list-expansion rule in event extraction prompt")
	}

	// Note prompt
	prompt = BuildTextExtractionPrompt("tip 1, tip 2, tip 3", "", []byte("[]"), time.Now(), "note", 0.9, nil, nil)
	if !strings.Contains(prompt, "If the input contains a list with 3+ items") {
		t.Error("expected list-expansion rule in note extraction prompt")
	}
}

func TestBuildGapAnalysisPrompt_RejectsMeta_ConsolidationQuestions(t *testing.T) {
	prompt := BuildGapAnalysisPrompt("task", "Walk Daily", []RelatedEntity{
		{ID: "abc", SourceType: "task", Title: "Walk Daily", Content: "Walk Daily recurring every morning"},
		{ID: "def", SourceType: "task", Title: "Walk Daily", Content: "Walk Daily recurring every morning"},
	})
	if !strings.Contains(prompt, "Should we consolidate these near-identical tasks?") {
		t.Errorf("expected explicit rejection of consolidation meta-question")
	}
	if !strings.Contains(prompt, "Why are there multiple copies of") {
		t.Errorf("expected explicit rejection of duplicate count meta-question")
	}
	if !strings.Contains(prompt, "Explicitly forbidden:") {
		t.Errorf("expected 'Explicitly forbidden:' section with examples")
	}
}

func TestBuildGapAnalysisPrompt_RejectsMeta_DuplicateCleanupQuestions(t *testing.T) {
	prompt := BuildGapAnalysisPrompt("task", "Review documents", []RelatedEntity{
		{ID: "g1", SourceType: "task", Title: "Review documents", Content: "Review documents from Jane"},
	})
	if !strings.Contains(prompt, "This looks like a duplicate") {
		t.Errorf("expected explicit rejection of duplicate cleanup meta-question")
	}
	if !strings.Contains(prompt, "should we clean it up?") {
		t.Errorf("expected cleanup suggestion to be forbidden in prompt")
	}
}

func TestBuildTextExtractionPrompt_TaskPromptContainsConfidence(t *testing.T) {
	prompt := BuildTextExtractionPrompt("do the dishes", "", []byte("[]"), time.Now(), "task", 0.8, nil, nil)
	if !strings.Contains(prompt, "80%") {
		t.Error("expected task prompt to include heuristic confidence percentage")
	}
	if !strings.Contains(prompt, "reclassified_as") {
		t.Error("expected task prompt to include reclassified_as field")
	}
}

func TestBuildTextExtractionPrompt_NotePromptContainsConfidence(t *testing.T) {
	prompt := BuildTextExtractionPrompt("my PT's phone is 555-1234", "", []byte("[]"), time.Now(), "note", 0.6, nil, nil)
	if !strings.Contains(prompt, "60%") {
		t.Error("expected note prompt to include heuristic confidence percentage")
	}
	if !strings.Contains(prompt, "reclassified_as") {
		t.Error("expected note prompt to include reclassified_as field")
	}
}

func TestBuildTextExtractionPrompt_EventPromptContainsReclassifiedAs(t *testing.T) {
	prompt := BuildTextExtractionPrompt("team standup tomorrow at 10", "", []byte("[]"), time.Now(), "event", 0.9, nil, nil)
	if !strings.Contains(prompt, "reclassified_as") {
		t.Error("expected event prompt to include reclassified_as field")
	}
	if !strings.Contains(prompt, "90%") {
		t.Error("expected event prompt to include heuristic confidence percentage")
	}
}

func TestBuildTextExtractionPrompt_LowConfidenceUsesGeneric(t *testing.T) {
	// confidence < 0.5 should fall back to generic prompt regardless of typeHint
	prompt := BuildTextExtractionPrompt("get rid of cooking oil", "", []byte("[]"), time.Now(), "note", 0.3, nil, nil)
	// Generic prompt has "voice capture" text; type-specific prompts do not
	if !strings.Contains(prompt, "voice capture") {
		t.Error("expected low-confidence hint to fall back to generic prompt")
	}
}

func TestBuildTextExtractionPrompt_GenericPromptHasFewShotExamples(t *testing.T) {
	prompt := BuildTextExtractionPrompt("some input", "", []byte("[]"), time.Now(), "", 0, nil, nil)
	if !strings.Contains(prompt, "Examples") {
		t.Error("expected generic prompt to include few-shot examples block")
	}
	if !strings.Contains(prompt, "dentist at 2pm") {
		t.Error("expected generic prompt examples to include event example")
	}
}

func TestBuildTextExtractionPrompt_ContainsContextKindGuidance(t *testing.T) {
	now := time.Now()
	hints := []struct {
		typeHint   string
		confidence float64
	}{
		{"", 0},
		{"task", 0.9},
		{"event", 0.9},
		{"note", 0.9},
	}
	for _, h := range hints {
		prompt := BuildTextExtractionPrompt("test input", "", []byte(`[{"id":"ctx-1","title":"Shopping","kind":"list"}]`), now, h.typeHint, h.confidence, nil, nil)
		if !strings.Contains(prompt, "suggested_new_context_kind") {
			t.Errorf("typeHint=%q: prompt missing suggested_new_context_kind field", h.typeHint)
		}
		if !strings.Contains(prompt, "Project") || !strings.Contains(prompt, "Area") || !strings.Contains(prompt, "List") {
			t.Errorf("typeHint=%q: prompt missing context kind guidance (Project/Area/List)", h.typeHint)
		}
	}
}

func TestBuildGapAnalysisPrompt_RejectsMeta_HygieneObservations(t *testing.T) {
	prompt := BuildGapAnalysisPrompt("task", "Daily standup", []RelatedEntity{
		{ID: "h1", SourceType: "task", Title: "Daily standup", Content: "Daily team standup meeting"},
	})
	if !strings.Contains(prompt, "We should consider a recurrence pattern") {
		t.Errorf("expected explicit rejection of process improvement meta-question")
	}
	if !strings.Contains(prompt, "recurrence pattern") {
		t.Errorf("expected recurrence pattern suggestion to be forbidden")
	}
	if !strings.Contains(prompt, "Do NOT emit gaps that are observations about the system") {
		t.Errorf("expected explicit ban on hygiene observations")
	}
}

func TestBuildTransactionExtractionPrompt_WithContextAnnotations(t *testing.T) {
	annotations := []string{
		"User is tracking travel expenses for Q2 trip",
		"Recent context shift to expense management",
	}
	prompt := buildTransactionExtractionPrompt("AMZN MKTP US*ABC123", "", []byte("[]"), annotations)
	if !strings.Contains(prompt, "Context Annotations") {
		t.Error("expected context annotations header in transaction prompt")
	}
	if !strings.Contains(prompt, "User is tracking travel expenses for Q2 trip") {
		t.Error("expected first annotation in transaction prompt")
	}
	if !strings.Contains(prompt, "Recent context shift to expense management") {
		t.Error("expected second annotation in transaction prompt")
	}
}

func TestBuildTransactionExtractionPrompt_WithoutContextAnnotations(t *testing.T) {
	prompt := buildTransactionExtractionPrompt("AMZN MKTP US*ABC123", "", []byte("[]"), nil)
	if strings.Contains(prompt, "Context Annotations") {
		t.Error("expected no context annotations header when annotations is empty")
	}
}

func TestBuildTransactionExtractionPrompt_WithEmptyContextAnnotations(t *testing.T) {
	prompt := buildTransactionExtractionPrompt("AMZN MKTP US*ABC123", "", []byte("[]"), []string{})
	if strings.Contains(prompt, "Context Annotations") {
		t.Error("expected no context annotations header when annotations slice is empty")
	}
}
