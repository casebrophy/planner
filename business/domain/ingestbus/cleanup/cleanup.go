package cleanup

import (
	"regexp"
	"strings"
)

// StripFillers removes transcription noise (um, uh, like, you know, etc.) from text.
// It preserves the capitalization of non-filler words and collapses multiple spaces.
func StripFillers(text string) string {
	if text == "" {
		return ""
	}

	// Define filler words/phrases prioritizing longer phrases first to avoid partial matches
	// Order matters: longer patterns first to avoid breaking up words like "uh-huh"
	fillers := []string{
		"you know", "I mean", "kind of", "sort of",
		"uh-huh", "oh well",
		"um", "uh", "hmm", "ah", "er",
		"basically", "literally", "honestly",
	}

	result := text
	for _, filler := range fillers {
		// Create a case-insensitive pattern with word boundaries
		// Use \b for word boundaries to avoid partial matches
		pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(filler) + `\b`)
		result = pattern.ReplaceAllString(result, "")
	}

	// Collapse multiple spaces into single space
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")

	// Trim leading and trailing whitespace
	result = strings.TrimSpace(result)

	return result
}

// SplitClauses splits text into discrete clauses on conjunctions and sentence-ending punctuation.
// It returns non-empty clauses in the order they appear, trimmed of whitespace.
func SplitClauses(text string) []string {
	if text == "" {
		return []string{}
	}

	// Split on conjunctions with word boundaries: " and ", " also ", " oh and "
	// Also split on sentence-ending punctuation: . ? !
	// Use regex to split on these delimiters but preserve content

	// Replace conjunctions with a delimiter
	result := text
	conjunctions := []string{" oh and ", " and ", " also "}
	for _, conj := range conjunctions {
		pattern := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(conj))
		result = pattern.ReplaceAllString(result, "|")
	}

	// Split on sentence-ending punctuation
	// Replace . ? ! with pipe delimiter, preserving them for context
	result = strings.NewReplacer(
		".", "|",
		"?", "|",
		"!", "|",
	).Replace(result)

	// Split on the delimiter
	parts := strings.Split(result, "|")

	// Process and filter clauses
	var clauses []string
	for _, part := range parts {
		clause := strings.TrimSpace(part)
		if clause != "" {
			clauses = append(clauses, clause)
		}
	}

	return clauses
}
