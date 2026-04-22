package eval

import (
	"testing"

	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
)

func TestScore_AllPass(t *testing.T) {
	results := []FixtureResult{
		{
			Fixture:  Fixture{Expected: FixtureExpected{PrimaryType: "task", ContextID: "ctx-1", ContextKind: "list"}},
			Pass:     true,
			Failures: nil,
		},
		{
			Fixture:  Fixture{Expected: FixtureExpected{PrimaryType: "event", ContextID: "ctx-2"}},
			Pass:     true,
			Failures: nil,
		},
	}
	m := Score(results)
	if m.TypeAccuracy != 1.0 {
		t.Errorf("TypeAccuracy: expected 1.0, got %.2f", m.TypeAccuracy)
	}
	if m.ContextAccuracy != 1.0 {
		t.Errorf("ContextAccuracy: expected 1.0, got %.2f", m.ContextAccuracy)
	}
	if m.ListAssignmentRate != 1.0 {
		t.Errorf("ListAssignmentRate: expected 1.0, got %.2f", m.ListAssignmentRate)
	}
	if m.EscalationRate != 0.0 {
		t.Errorf("EscalationRate: expected 0.0, got %.2f", m.EscalationRate)
	}
}

func TestScore_AllFail(t *testing.T) {
	results := []FixtureResult{
		{
			Fixture:  Fixture{Expected: FixtureExpected{PrimaryType: "task", ContextID: "ctx-1", ContextKind: "list"}},
			Pass:     false,
			Failures: []string{"primary_type: expected \"task\", got \"note\"", "context_id: expected \"ctx-1\", got \"ctx-2\""},
		},
	}
	m := Score(results)
	if m.TypeAccuracy != 0.0 {
		t.Errorf("TypeAccuracy: expected 0.0, got %.2f", m.TypeAccuracy)
	}
	if m.ContextAccuracy != 0.0 {
		t.Errorf("ContextAccuracy: expected 0.0, got %.2f", m.ContextAccuracy)
	}
	if m.ListAssignmentRate != 0.0 {
		t.Errorf("ListAssignmentRate: expected 0.0, got %.2f", m.ListAssignmentRate)
	}
}

func TestScore_Mixed(t *testing.T) {
	results := []FixtureResult{
		{
			Fixture:  Fixture{Expected: FixtureExpected{PrimaryType: "task"}},
			Pass:     true,
			Failures: nil,
		},
		{
			Fixture:  Fixture{Expected: FixtureExpected{PrimaryType: "event"}},
			Pass:     false,
			Failures: []string{"primary_type: expected \"event\", got \"note\""},
		},
		{
			Fixture:  Fixture{Expected: FixtureExpected{PrimaryType: "note"}},
			Pass:     true,
			Failures: nil,
		},
	}
	m := Score(results)
	// 2 out of 3 pass
	if m.TypeAccuracy < 0.66 || m.TypeAccuracy > 0.67 {
		t.Errorf("TypeAccuracy: expected ~0.667, got %.4f", m.TypeAccuracy)
	}
}

func TestScore_NoPrimaryType_ExcludedFromDenominator(t *testing.T) {
	results := []FixtureResult{
		{
			// No PrimaryType: should not count toward TypeAccuracy denominator
			Fixture:  Fixture{Expected: FixtureExpected{MinActionItems: 1}},
			Pass:     true,
			Failures: nil,
		},
		{
			Fixture:  Fixture{Expected: FixtureExpected{PrimaryType: "task"}},
			Pass:     true,
			Failures: nil,
		},
	}
	m := Score(results)
	// denominator = 1 (only the fixture with PrimaryType)
	if m.TypeAccuracy != 1.0 {
		t.Errorf("TypeAccuracy: expected 1.0 (only 1 fixture has primary_type), got %.2f", m.TypeAccuracy)
	}
}

func TestScore_ListAssignmentRate_OnlyListContextKind(t *testing.T) {
	results := []FixtureResult{
		{
			// project kind: excluded from ListAssignmentRate
			Fixture:  Fixture{Expected: FixtureExpected{ContextKind: "project", ContextID: "ctx-1"}},
			Pass:     true,
			Failures: nil,
		},
		{
			// list kind: included in ListAssignmentRate, passes
			Fixture:  Fixture{Expected: FixtureExpected{ContextKind: "list", ContextID: "ctx-2"}},
			Pass:     true,
			Failures: nil,
		},
		{
			// list kind: included, fails context_id
			Fixture:  Fixture{Expected: FixtureExpected{ContextKind: "list", ContextID: "ctx-3"}},
			Pass:     false,
			Failures: []string{"context_id: expected \"ctx-3\", got \"ctx-4\""},
		},
	}
	m := Score(results)
	// denominator = 2 (list kind fixtures), numerator = 1
	expected := 0.5
	if m.ListAssignmentRate != expected {
		t.Errorf("ListAssignmentRate: expected %.1f, got %.2f", expected, m.ListAssignmentRate)
	}
}

func TestScore_NoFixtures(t *testing.T) {
	m := Score(nil)
	if m.TypeAccuracy != 0.0 {
		t.Errorf("TypeAccuracy: expected 0.0 for empty, got %.2f", m.TypeAccuracy)
	}
	if m.ContextAccuracy != 0.0 {
		t.Errorf("ContextAccuracy: expected 0.0 for empty, got %.2f", m.ContextAccuracy)
	}
}

func TestPrimaryType(t *testing.T) {
	tests := []struct {
		name string
		ex   extractor.TextExtraction
		want string
	}{
		{
			name: "task",
			ex:   extractor.TextExtraction{ActionItems: []extractor.ActionItem{{Title: "do it"}}},
			want: "task",
		},
		{
			name: "event",
			ex:   extractor.TextExtraction{Events: []extractor.ExtractedEvent{{Title: "meeting"}}},
			want: "event",
		},
		{
			name: "note",
			ex:   extractor.TextExtraction{Notes: []extractor.ExtractedNote{{Content: "info"}}},
			want: "note",
		},
		{
			name: "mixed task+event",
			ex: extractor.TextExtraction{
				ActionItems: []extractor.ActionItem{{Title: "do it"}},
				Events:      []extractor.ExtractedEvent{{Title: "meeting"}},
			},
			want: "mixed",
		},
		{
			name: "mixed task+note",
			ex: extractor.TextExtraction{
				ActionItems: []extractor.ActionItem{{Title: "do it"}},
				Notes:       []extractor.ExtractedNote{{Content: "info"}},
			},
			want: "mixed",
		},
		{
			name: "mixed event+note",
			ex: extractor.TextExtraction{
				Events: []extractor.ExtractedEvent{{Title: "meeting"}},
				Notes:  []extractor.ExtractedNote{{Content: "info"}},
			},
			want: "mixed",
		},
		{
			name: "unknown",
			ex:   extractor.TextExtraction{},
			want: "unknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := primaryType(tt.ex)
			if got != tt.want {
				t.Errorf("primaryType() = %q, want %q", got, tt.want)
			}
		})
	}
}
