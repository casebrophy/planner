package cleanup

import (
	"testing"
)

func TestStripFillers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes um",
			input:    "I want um some coffee",
			expected: "I want some coffee",
		},
		{
			name:     "removes uh",
			input:    "uh this is great",
			expected: "this is great",
		},
		{
			name:     "removes basically",
			input:    "I basically like that basically idea",
			expected: "I like that idea",
		},
		{
			name:     "removes you know",
			input:    "you know it's great you know",
			expected: "it's great",
		},
		{
			name:     "removes basically",
			input:    "basically it's basically done",
			expected: "it's done",
		},
		{
			name:     "removes uh-huh",
			input:    "uh-huh I like it uh-huh",
			expected: "I like it",
		},
		{
			name:     "case insensitive um",
			input:    "Um I want Um some coffee UM",
			expected: "I want some coffee",
		},
		{
			name:     "case insensitive basically",
			input:    "Basically this is Basically great BASICALLY",
			expected: "this is great",
		},
		{
			name:     "multiple fillers combined",
			input:    "um you know basically I want um coffee",
			expected: "I want coffee",
		},
		{
			name:     "collapses multiple spaces",
			input:    "I   want    some   coffee",
			expected: "I want some coffee",
		},
		{
			name:     "trims leading whitespace",
			input:    "   I want coffee",
			expected: "I want coffee",
		},
		{
			name:     "trims trailing whitespace",
			input:    "I want coffee   ",
			expected: "I want coffee",
		},
		{
			name:     "preserves contractions",
			input:    "I'd um like some coffee",
			expected: "I'd like some coffee",
		},
		{
			name:     "does not match partial words",
			input:    "stand and hand um the book",
			expected: "stand and hand the book",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "only fillers",
			input:    "um uh basically you know uh-huh",
			expected: "",
		},
		{
			name:     "no fillers",
			input:    "I want coffee",
			expected: "I want coffee",
		},
		{
			name:     "removes hmm",
			input:    "hmm that's interesting hmm",
			expected: "that's interesting",
		},
		{
			name:     "removes ah",
			input:    "ah I see ah",
			expected: "I see",
		},
		{
			name:     "removes honestly",
			input:    "honestly it's honestly great",
			expected: "it's great",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripFillers(tt.input)
			if got != tt.expected {
				t.Errorf("StripFillers(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSplitClauses(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "splits on and",
			input:    "buy milk and call john",
			expected: []string{"buy milk", "call john"},
		},
		{
			name:     "splits on also",
			input:    "buy milk also call john",
			expected: []string{"buy milk", "call john"},
		},
		{
			name:     "splits on oh and",
			input:    "buy milk oh and call john",
			expected: []string{"buy milk", "call john"},
		},
		{
			name:     "splits on period",
			input:    "buy milk. call john",
			expected: []string{"buy milk", "call john"},
		},
		{
			name:     "splits on question mark",
			input:    "what's up? call john",
			expected: []string{"what's up", "call john"},
		},
		{
			name:     "splits on exclamation",
			input:    "buy milk! call john",
			expected: []string{"buy milk", "call john"},
		},
		{
			name:     "multiple splits",
			input:    "buy milk and call john. then rest! oh and finish up",
			expected: []string{"buy milk", "call john", "then rest", "finish up"},
		},
		{
			name:     "case insensitive and",
			input:    "buy milk AND call john",
			expected: []string{"buy milk", "call john"},
		},
		{
			name:     "case insensitive also",
			input:    "buy milk ALSO call john",
			expected: []string{"buy milk", "call john"},
		},
		{
			name:     "trims whitespace from clauses",
			input:    "  buy milk  and  call john  ",
			expected: []string{"buy milk", "call john"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single clause no splits",
			input:    "buy milk",
			expected: []string{"buy milk"},
		},
		{
			name:     "consecutive conjunctions",
			input:    "milk and also coffee",
			expected: []string{"milk", "also coffee"},
		},
		{
			name:     "filters empty clauses",
			input:    "buy milk and . call john",
			expected: []string{"buy milk", "call john"},
		},
		{
			name:     "preserves and in context",
			input:    "buy milk and call john and finish",
			expected: []string{"buy milk", "call john", "finish"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitClauses(tt.input)
			if !slicesEqual(got, tt.expected) {
				t.Errorf("SplitClauses(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

// slicesEqual is a helper to compare string slices for testing
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
