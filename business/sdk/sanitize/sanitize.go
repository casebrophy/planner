// Package sanitize provides rule-based PII redaction for text content
// before it is sent to external APIs.
package sanitize

import (
	"regexp"
)

// Finding describes a type of PII found and how many times it was redacted.
type Finding struct {
	Kind  string // SSN, PHONE, CREDIT_CARD, ROUTING_NUMBER, BANK_ACCOUNT
	Count int
}

// Result is the output of a sanitize pass.
type Result struct {
	Text     string    // redacted text, safe to send to external API
	Findings []Finding // summary of what was redacted, in deterministic order
}

// Compiled regexps — evaluated once at package init.
var (
	reSSN = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)

	// No leading \b: the opening ( and country-code + are non-word chars that
	// \b would prevent from being consumed, leaving them dangling in the output.
	rePhone = regexp.MustCompile(`(?:\+?1[-.\s]?)?\(?\d{3}\)?[-.\s]\d{3}[-.\s]\d{4}\b`)

	// Require separators between groups OR exactly 16 bare digits.
	// This prevents matching arbitrary 13-digit strings (invoice numbers, etc.).
	reCreditCard = regexp.MustCompile(`\b\d{4}[-\s]\d{4}[-\s]\d{4}[-\s]\d{1,4}\b|\b\d{16}\b`)

	// ABA routing: keyword-preceded 9 digits.
	reRouting = regexp.MustCompile(`(?i)(?:routing(?:\s+number)?|aba)[:\s#]*\d{9}\b`)

	// Bank account: keyword-preceded 6–17 digits.
	reBankAccount = regexp.MustCompile(`(?i)(?:account(?:\s+(?:number|no|#))?|acct\.?)[:\s#]*\d{6,17}\b`)
)

type rule struct {
	re          *regexp.Regexp
	token       string
	kind        string
	replaceFunc func(match string, token string) string
}

var rules = []rule{
	{reSSN, "[REDACTED-SSN]", "SSN", replaceAll},
	{rePhone, "[REDACTED-PHONE]", "PHONE", replaceAll},
	{reCreditCard, "[REDACTED-CARD]", "CREDIT_CARD", replaceAll},
	{reRouting, "", "ROUTING_NUMBER", replaceDigits},
	{reBankAccount, "", "BANK_ACCOUNT", replaceDigits},
}

// replaceAll substitutes the entire match with the token.
func replaceAll(match, token string) string { return token }

// replaceDigits substitutes only the digit sequence within a keyword-prefixed
// match, preserving the label (e.g. "Routing Number: [REDACTED-ROUTING]").
func replaceDigits(match, _ string) string {
	reDigits := regexp.MustCompile(`\d{6,17}|\d{9}`)
	return reDigits.ReplaceAllStringFunc(match, func(d string) string {
		if len(d) == 9 {
			return "[REDACTED-ROUTING]"
		}
		return "[REDACTED-ACCOUNT]"
	})
}

// Sanitize applies all PII rules to text and returns the redacted result.
// The original text is never modified. Findings are returned in deterministic
// order (SSN, PHONE, CREDIT_CARD, ROUTING_NUMBER, BANK_ACCOUNT).
func Sanitize(text string) Result {
	if text == "" {
		return Result{Text: text}
	}

	out := text
	var findings []Finding

	for _, r := range rules {
		matches := r.re.FindAllString(out, -1)
		if len(matches) == 0 {
			continue
		}

		token := r.token
		out = r.re.ReplaceAllStringFunc(out, func(match string) string {
			return r.replaceFunc(match, token)
		})

		// De-duplicate: if the same kind was already added (shouldn't happen
		// with non-overlapping rules, but be safe), increment its count.
		found := false
		for i := range findings {
			if findings[i].Kind == r.kind {
				findings[i].Count += len(matches)
				found = true
				break
			}
		}
		if !found {
			findings = append(findings, Finding{Kind: r.kind, Count: len(matches)})
		}
	}

	return Result{Text: out, Findings: findings}
}
