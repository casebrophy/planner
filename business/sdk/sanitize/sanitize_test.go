package sanitize_test

import (
	"testing"

	"github.com/casebrophy/planner/business/sdk/sanitize"
)

func TestSanitize(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantText   string
		wantKinds  []string
		wantCounts map[string]int
	}{
		// ── SSN ──────────────────────────────────────────────────────
		{
			name:      "SSN basic",
			input:     "My SSN is 123-45-6789.",
			wantText:  "My SSN is [REDACTED-SSN].",
			wantKinds: []string{"SSN"},
		},
		{
			name:      "SSN two occurrences",
			input:     "SSN: 123-45-6789 and backup 987-65-4321",
			wantText:  "SSN: [REDACTED-SSN] and backup [REDACTED-SSN]",
			wantKinds: []string{"SSN"},
			wantCounts: map[string]int{"SSN": 2},
		},
		{
			name:      "SSN not matched in long digit string",
			input:     "Ref: 1234567890123",
			wantText:  "Ref: 1234567890123",
			wantKinds: nil,
		},

		// ── Phone ─────────────────────────────────────────────────────
		{
			name:      "phone parens format",
			input:     "Call me at (555) 123-4567.",
			wantText:  "Call me at [REDACTED-PHONE].",
			wantKinds: []string{"PHONE"},
		},
		{
			name:      "phone dot format",
			input:     "555.123.4567",
			wantText:  "[REDACTED-PHONE]",
			wantKinds: []string{"PHONE"},
		},
		{
			name:      "phone with country code",
			input:     "+1-800-555-1234",
			wantText:  "[REDACTED-PHONE]",
			wantKinds: []string{"PHONE"},
		},
		{
			name:      "phone dash format",
			input:     "555-123-4567",
			wantText:  "[REDACTED-PHONE]",
			wantKinds: []string{"PHONE"},
		},

		// ── Credit card ───────────────────────────────────────────────
		{
			name:      "credit card spaced",
			input:     "card 4111 1111 1111 1111 exp",
			wantText:  "card [REDACTED-CARD] exp",
			wantKinds: []string{"CREDIT_CARD"},
		},
		{
			name:      "credit card dashed",
			input:     "4111-1111-1111-1111",
			wantText:  "[REDACTED-CARD]",
			wantKinds: []string{"CREDIT_CARD"},
		},

		// ── Routing number ────────────────────────────────────────────
		{
			name:      "routing with label",
			input:     "Routing Number: 021000021",
			wantText:  "Routing Number: [REDACTED-ROUTING]",
			wantKinds: []string{"ROUTING_NUMBER"},
		},
		{
			name:      "ABA abbreviation",
			input:     "ABA 021000021 is the number",
			wantText:  "ABA [REDACTED-ROUTING] is the number",
			wantKinds: []string{"ROUTING_NUMBER"},
		},
		{
			name:      "bare 9 digits not matched",
			input:     "Order #123456789 confirmed",
			wantText:  "Order #123456789 confirmed",
			wantKinds: nil,
		},

		// ── Bank account ──────────────────────────────────────────────
		{
			name:      "account with label",
			input:     "Account Number: 98765432",
			wantText:  "Account Number: [REDACTED-ACCOUNT]",
			wantKinds: []string{"BANK_ACCOUNT"},
		},
		{
			name:      "acct abbreviation",
			input:     "acct. 123456789012",
			wantText:  "acct. [REDACTED-ACCOUNT]",
			wantKinds: []string{"BANK_ACCOUNT"},
		},

		// ── Mixed / edge cases ────────────────────────────────────────
		{
			name:      "multiple kinds",
			input:     "SSN 123-45-6789 and call (555) 123-4567",
			wantText:  "SSN [REDACTED-SSN] and call [REDACTED-PHONE]",
			wantKinds: []string{"SSN", "PHONE"},
		},
		{
			name:      "email address not touched",
			input:     "from: user@example.com",
			wantText:  "from: user@example.com",
			wantKinds: nil,
		},
		{
			name:      "empty string",
			input:     "",
			wantText:  "",
			wantKinds: nil,
		},
		{
			name:      "no PII",
			input:     "Please review the attached report.",
			wantText:  "Please review the attached report.",
			wantKinds: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitize.Sanitize(tt.input)

			if result.Text != tt.wantText {
				t.Errorf("Text:\n  got  %q\n  want %q", result.Text, tt.wantText)
			}

			if len(result.Findings) != len(tt.wantKinds) {
				t.Errorf("Findings count: got %d, want %d (findings: %v)", len(result.Findings), len(tt.wantKinds), result.Findings)
			} else {
				for i, kind := range tt.wantKinds {
					if result.Findings[i].Kind != kind {
						t.Errorf("Findings[%d].Kind: got %q, want %q", i, result.Findings[i].Kind, kind)
					}
				}
			}

			// Check specific counts when provided.
			for kind, wantCount := range tt.wantCounts {
				for _, f := range result.Findings {
					if f.Kind == kind && f.Count != wantCount {
						t.Errorf("Findings[%q].Count: got %d, want %d", kind, f.Count, wantCount)
					}
				}
			}
		})
	}
}
