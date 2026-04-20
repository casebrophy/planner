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

func TestHasActionVerb(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"buy milk", true},
		{"call john", true},
		{"eggs", false},
		{"Sarah", false},
		{"bread and butter", false},
		{"schedule a meeting", true},
		{"development", false},
		{"Pick up groceries", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := hasActionVerb(tt.input); got != tt.want {
				t.Errorf("hasActionVerb(%q) = %v, want %v", tt.input, got, tt.want)
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
		// Verb heuristic: both sides have verbs → split
		{
			name:     "splits on and when both sides have verbs",
			input:    "buy milk and call john",
			expected: []string{"buy milk", "call john"},
		},
		// Verb heuristic: second side has no verb → keep together
		{
			name:     "preserves compound objects",
			input:    "buy milk and eggs",
			expected: []string{"buy milk and eggs"},
		},
		{
			name:     "preserves names joined by and",
			input:    "schedule a meeting with John and Sarah",
			expected: []string{"schedule a meeting with John and Sarah"},
		},
		{
			name:     "preserves compound nouns",
			input:    "research and development is important",
			expected: []string{"research and development is important"},
		},
		// Mixed: verb-less segment merges with predecessor
		{
			name:     "merges verb-less segment with predecessor",
			input:    "buy milk and eggs and call john",
			expected: []string{"buy milk and eggs", "call john"},
		},
		// Discourse markers always split
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
		// Punctuation always splits
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
		// Combined
		{
			name:     "multiple splits mixed",
			input:    "buy milk and call john. then rest! oh and finish up",
			expected: []string{"buy milk", "call john", "then rest", "finish up"},
		},
		{
			name:     "case insensitive and with verbs",
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
			name:     "three verbed clauses split",
			input:    "buy milk and call john and finish homework",
			expected: []string{"buy milk", "call john", "finish homework"},
		},
		{
			name:     "no verbs on either side keeps together",
			input:    "bread and butter",
			expected: []string{"bread and butter"},
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

func TestExpandCommaList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "expands comma list with leading verb",
			input:    "buy milk, bread, and eggs",
			expected: []string{"buy milk", "buy bread", "buy eggs"},
		},
		{
			name:     "single item with verb",
			input:    "buy milk",
			expected: []string{"buy milk"},
		},
		{
			name:     "no verb, no expansion",
			input:    "bread and eggs",
			expected: []string{"bread and eggs"},
		},
		{
			name:     "two comma-separated items",
			input:    "call john, and email sarah",
			expected: []string{"call john", "call email sarah"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "only verb",
			input:    "buy",
			expected: []string{"buy"},
		},
		{
			name:     "verb with no commas",
			input:    "schedule a meeting",
			expected: []string{"schedule a meeting"},
		},
		{
			name:     "case insensitive verb",
			input:    "BUY milk, bread, eggs",
			expected: []string{"BUY milk", "BUY bread", "BUY eggs"},
		},
		{
			name:     "three items with and before last",
			input:    "buy milk, bread, and eggs",
			expected: []string{"buy milk", "buy bread", "buy eggs"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandCommaList(tt.input)
			if !slicesEqual(got, tt.expected) {
				t.Errorf("expandCommaList(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDetectSubordinateClause(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "when I + action verb",
			input:    "when I finish tasks",
			expected: true,
		},
		{
			name:     "when we + action verb",
			input:    "when we book a flight",
			expected: true,
		},
		{
			name:     "after I + action verb",
			input:    "after I call john",
			expected: true,
		},
		{
			name:     "if I + action verb",
			input:    "if I find the key",
			expected: true,
		},
		{
			name:     "while we + action verb",
			input:    "while we work on it",
			expected: true,
		},
		{
			name:     "once I + action verb",
			input:    "once I send the email",
			expected: true,
		},
		{
			name:     "whenever I + action verb",
			input:    "whenever I remind john",
			expected: true,
		},
		{
			name:     "when without pronoun",
			input:    "when it's done",
			expected: false,
		},
		{
			name:     "when I without action verb",
			input:    "when I'm tired",
			expected: false,
		},
		{
			name:     "before + action verb no pronoun",
			input:    "before going home",
			expected: false,
		},
		{
			name:     "no subordinate marker",
			input:    "buy milk and bread",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "case insensitive marker",
			input:    "WHEN I finish work",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectSubordinateClause(tt.input)
			if got != tt.expected {
				t.Errorf("DetectSubordinateClause(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSplitClausesWithRoles(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		checkLen bool
		lenWant  int
		checks   []struct {
			idx      int
			text     string
			role     ClauseRole
			sibIdx   int
		}
	}{
		{
			name:     "simple main clause",
			input:    "buy milk",
			checkLen: true,
			lenWant:  1,
			checks: []struct {
				idx      int
				text     string
				role     ClauseRole
				sibIdx   int
			}{
				{idx: 0, text: "buy milk", role: RoleMain, sibIdx: 0},
			},
		},
		{
			name:     "comma list expansion",
			input:    "buy milk, bread, and eggs",
			checkLen: true,
			lenWant:  3,
			checks: []struct {
				idx      int
				text     string
				role     ClauseRole
				sibIdx   int
			}{
				{idx: 0, text: "buy milk", role: RoleExpandedFromCommaList, sibIdx: 0},
				{idx: 1, text: "buy bread", role: RoleExpandedFromCommaList, sibIdx: 1},
				{idx: 2, text: "buy eggs", role: RoleExpandedFromCommaList, sibIdx: 2},
			},
		},
		{
			name:     "multiple main clauses",
			input:    "buy milk. call john",
			checkLen: true,
			lenWant:  2,
			checks: []struct {
				idx      int
				text     string
				role     ClauseRole
				sibIdx   int
			}{
				{idx: 0, text: "buy milk", role: RoleMain, sibIdx: 0},
				{idx: 1, text: "call john", role: RoleMain, sibIdx: 1},
			},
		},
		{
			name:     "empty input",
			input:    "",
			checkLen: true,
			lenWant:  0,
			checks:   []struct {
				idx      int
				text     string
				role     ClauseRole
				sibIdx   int
			}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitClausesWithRoles(tt.input)
			if tt.checkLen && len(got) != tt.lenWant {
				t.Errorf("SplitClausesWithRoles(%q) length = %d, want %d", tt.input, len(got), tt.lenWant)
			}
			for _, check := range tt.checks {
				if check.idx >= len(got) {
					t.Errorf("SplitClausesWithRoles(%q) index %d out of range (len=%d)", tt.input, check.idx, len(got))
					continue
				}
				if got[check.idx].Text != check.text {
					t.Errorf("SplitClausesWithRoles(%q)[%d].Text = %q, want %q", tt.input, check.idx, got[check.idx].Text, check.text)
				}
				if got[check.idx].Role != check.role {
					t.Errorf("SplitClausesWithRoles(%q)[%d].Role = %v, want %v", tt.input, check.idx, got[check.idx].Role, check.role)
				}
				if got[check.idx].SiblingIdx != check.sibIdx {
					t.Errorf("SplitClausesWithRoles(%q)[%d].SiblingIdx = %d, want %d", tt.input, check.idx, got[check.idx].SiblingIdx, check.sibIdx)
				}
			}
		})
	}
}
