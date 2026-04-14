package generator

import (
	"fmt"
	"testing"
	"time"
)

func TestReasonImplications_PrepKeywordMatch(t *testing.T) {
	now := time.Date(2026, 4, 14, 9, 0, 0, 0, time.UTC)
	eventStart := now.Add(3 * time.Hour).Format(time.RFC3339) // 12:00 PM

	tasks := []TaskRef{
		{ID: "task-1", Title: "Prepare slides for board review", Status: "open"},
		{ID: "task-2", Title: "Buy groceries", Status: "open"},
	}
	events := []EventRef{
		{ID: "event-1", Title: "Board meeting", StartsAt: eventStart, EndsAt: eventStart, AllDay: false},
	}

	results := ReasonImplications(tasks, events, now)

	// "Prepare slides for board review" should match "Board meeting" via prep keyword + overlap
	found := false
	for _, r := range results {
		if r.TaskID == "task-1" && r.EventID == "event-1" {
			found = true
			if r.Confidence < 0.3 {
				t.Errorf("expected confidence >= 0.3, got %f", r.Confidence)
			}
		}
	}
	if !found {
		t.Error("expected task-1 to be implied as prep for event-1")
	}

	// "Buy groceries" should NOT match "Board meeting"
	for _, r := range results {
		if r.TaskID == "task-2" && r.EventID == "event-1" {
			t.Error("did not expect task-2 to match event-1")
		}
	}
}

func TestReasonImplications_KeywordOverlapOnly(t *testing.T) {
	now := time.Date(2026, 4, 14, 9, 0, 0, 0, time.UTC)
	eventStart := now.Add(2 * time.Hour).Format(time.RFC3339)

	tasks := []TaskRef{
		{ID: "task-1", Title: "Review proposal document", Status: "open"},
	}
	events := []EventRef{
		{ID: "event-1", Title: "Proposal review call", StartsAt: eventStart, EndsAt: eventStart, AllDay: false},
	}

	results := ReasonImplications(tasks, events, now)

	// "Review proposal document" shares "review" + "proposal" with "Proposal review call"
	found := false
	for _, r := range results {
		if r.TaskID == "task-1" && r.EventID == "event-1" {
			found = true
		}
	}
	if !found {
		t.Error("expected task-1 to be implied as prep for event-1 via keyword overlap")
	}
}

func TestReasonImplications_SkipsAllDayEvents(t *testing.T) {
	now := time.Date(2026, 4, 14, 9, 0, 0, 0, time.UTC)
	eventStart := now.Add(2 * time.Hour).Format(time.RFC3339)

	tasks := []TaskRef{
		{ID: "task-1", Title: "Prepare conference notes", Status: "open"},
	}
	events := []EventRef{
		{ID: "event-1", Title: "Conference", StartsAt: eventStart, EndsAt: eventStart, AllDay: true},
	}

	results := ReasonImplications(tasks, events, now)
	if len(results) != 0 {
		t.Errorf("expected no implications for all-day events, got %d", len(results))
	}
}

func TestReasonImplications_SkipsPastEvents(t *testing.T) {
	now := time.Date(2026, 4, 14, 14, 0, 0, 0, time.UTC)
	// Event started 1 hour ago
	eventStart := now.Add(-1 * time.Hour).Format(time.RFC3339)

	tasks := []TaskRef{
		{ID: "task-1", Title: "Prepare meeting notes", Status: "open"},
	}
	events := []EventRef{
		{ID: "event-1", Title: "Meeting", StartsAt: eventStart, EndsAt: eventStart, AllDay: false},
	}

	results := ReasonImplications(tasks, events, now)
	if len(results) != 0 {
		t.Errorf("expected no implications for past events, got %d", len(results))
	}
}

func TestReasonImplications_SkipsEventWithin30Min(t *testing.T) {
	now := time.Date(2026, 4, 14, 13, 45, 0, 0, time.UTC)
	// Event starts in 15 minutes — not enough lead time
	eventStart := now.Add(15 * time.Minute).Format(time.RFC3339)

	tasks := []TaskRef{
		{ID: "task-1", Title: "Draft agenda", Status: "open"},
	}
	events := []EventRef{
		{ID: "event-1", Title: "Team standup", StartsAt: eventStart, EndsAt: eventStart, AllDay: false},
	}

	results := ReasonImplications(tasks, events, now)
	if len(results) != 0 {
		t.Errorf("expected no implications for events within 30 min, got %d", len(results))
	}
}

func TestReasonImplications_MultipleEvents(t *testing.T) {
	now := time.Date(2026, 4, 14, 8, 0, 0, 0, time.UTC)

	// task-1 shares "presentation" and "quarterly" with event-1; "write" is a prep keyword.
	// task-2 shares "quarterly" with event-1; "review" is a prep keyword.
	// task-3 ("book restaurant") has no overlap with event-1 or event-2, so no match.
	tasks := []TaskRef{
		{ID: "task-1", Title: "Write presentation slides for quarterly", Status: "open"},
		{ID: "task-2", Title: "Review quarterly numbers", Status: "open"},
		{ID: "task-3", Title: "Book restaurant", Status: "open"},
	}
	events := []EventRef{
		{
			ID:       "event-1",
			Title:    "Quarterly review presentation",
			StartsAt: now.Add(2 * time.Hour).Format(time.RFC3339),
			EndsAt:   now.Add(3 * time.Hour).Format(time.RFC3339),
			AllDay:   false,
		},
		{
			ID:       "event-2",
			Title:    "Team lunch",
			StartsAt: now.Add(4 * time.Hour).Format(time.RFC3339),
			EndsAt:   now.Add(5 * time.Hour).Format(time.RFC3339),
			AllDay:   false,
		},
	}

	results := ReasonImplications(tasks, events, now)

	// Neither task should match "Team lunch" (no keyword overlap).
	for _, r := range results {
		if r.EventID == "event-2" {
			t.Errorf("did not expect any implications for 'Team lunch', got task %s", r.TaskID)
		}
	}

	// At least one task should match event-1.
	eventOneMatches := 0
	for _, r := range results {
		if r.EventID == "event-1" {
			eventOneMatches++
		}
	}
	if eventOneMatches == 0 {
		t.Error("expected at least one task implied for event-1")
	}

	// task-3 should not match anything.
	for _, r := range results {
		if r.TaskID == "task-3" {
			t.Errorf("did not expect task-3 to match any event, but matched %s", r.EventID)
		}
	}
}

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		title    string
		expected []string
	}{
		{"Board meeting", []string{"board", "meeting"}},
		{"The quarterly review", []string{"quarterly", "review"}},
		{"A", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			kw := extractKeywords(tt.title)
			for _, exp := range tt.expected {
				if !kw[exp] {
					t.Errorf("expected keyword %q in %v", exp, kw)
				}
			}
		})
	}
}

func TestComputeImplicationScore(t *testing.T) {
	tests := []struct {
		taskTitle     string
		eventKeywords map[string]bool
		wantAbove     float64
		wantBelow     float64
		desc          string
	}{
		{
			// "agenda" overlaps with event keyword "agenda"; "prepare" is a prep keyword.
			taskTitle:     "prepare agenda",
			eventKeywords: map[string]bool{"meeting": true, "agenda": true},
			wantAbove:     0.3,
			wantBelow:     1.01,
			desc:          "prep keyword + overlap gives score above threshold",
		},
		{
			taskTitle:     "buy coffee",
			eventKeywords: map[string]bool{"board": true, "review": true},
			wantAbove:     -1,
			wantBelow:     0.3,
			desc:          "unrelated task should score below threshold",
		},
		{
			taskTitle:     "review board proposal",
			eventKeywords: map[string]bool{"board": true, "proposal": true, "review": true},
			wantAbove:     0.5,
			wantBelow:     1.01,
			desc:          "full overlap should score high",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			score := computeImplicationScore(tt.taskTitle, tt.eventKeywords)
			if tt.wantAbove >= 0 && score < tt.wantAbove {
				t.Errorf("score %f < wantAbove %f", score, tt.wantAbove)
			}
			if score >= tt.wantBelow {
				t.Errorf("score %f >= wantBelow %f", score, tt.wantBelow)
			}
		})
	}
}

func TestBuildImplicationSection_Empty(t *testing.T) {
	result := buildImplicationSection(nil)
	if result != "" {
		t.Errorf("expected empty string for nil implications, got %q", result)
	}
}

func TestBuildImplicationSection_GroupsByEvent(t *testing.T) {
	now := time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC)
	eventStart := now.Add(2 * time.Hour)

	implications := []ImplicationResult{
		{EventID: "ev-1", EventTitle: "Board meeting", EventStartsAt: eventStart, TaskID: "t-1", TaskTitle: "Prepare slides"},
		{EventID: "ev-1", EventTitle: "Board meeting", EventStartsAt: eventStart, TaskID: "t-2", TaskTitle: "Review agenda"},
	}

	section := buildImplicationSection(implications)
	if section == "" {
		t.Error("expected non-empty section")
	}
	// Both tasks should appear in the same group
	occurrences := 0
	for _, title := range []string{"Prepare slides", "Review agenda"} {
		if contains(section, title) {
			occurrences++
		}
	}
	if occurrences != 2 {
		t.Errorf("expected both task titles in section, found %d/2", occurrences)
	}
	// Should appear only once grouped under the event
	count := countOccurrences(section, "Board meeting")
	if count != 1 {
		t.Errorf("expected event title once in section, found %d times", count)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		})())
}

func countOccurrences(s, substr string) int {
	count := 0
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			count++
			i += len(substr) - 1
		}
	}
	return count
}

// Ensure ImplicationResult fields are exported and accessible.
func TestImplicationResultFields(_ *testing.T) {
	r := ImplicationResult{
		EventID:       "e1",
		EventTitle:    "Meeting",
		EventStartsAt: time.Now(),
		TaskID:        "t1",
		TaskTitle:     "Prep",
		Confidence:    0.8,
	}
	_ = fmt.Sprintf("%v", r)
}
