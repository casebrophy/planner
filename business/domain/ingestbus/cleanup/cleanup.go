package cleanup

import (
	"regexp"
	"strings"
)

// ClauseRole enum for clause classification
type ClauseRole string

const (
	RoleMain                 ClauseRole = "main"
	RoleSubordinate          ClauseRole = "subordinate"
	RoleExpandedFromCommaList ClauseRole = "expanded_from_comma_list"
)

// Clause represents a split clause with metadata about its role and position.
type Clause struct {
	Text       string
	Role       ClauseRole
	SiblingIdx int // 0-indexed position within siblings of the same role
}

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

// actionVerbs is the set of imperative verbs common in personal task management.
// Used by the verb heuristic to decide whether " and " joins two tasks or two
// objects within a single task.
var actionVerbs = map[string]bool{
	"add": true, "apply": true, "arrange": true, "ask": true, "assign": true,
	"attend": true, "book": true, "bring": true, "build": true, "buy": true,
	"call": true, "cancel": true, "check": true, "clean": true, "close": true,
	"confirm": true, "contact": true, "cook": true, "coordinate": true,
	"copy": true, "create": true, "debug": true, "delete": true, "deliver": true,
	"deploy": true, "deposit": true, "do": true, "download": true, "draft": true,
	"drive": true, "drop": true, "edit": true, "email": true, "file": true,
	"fill": true, "find": true, "finish": true, "fix": true, "follow": true,
	"get": true, "give": true, "go": true, "grab": true, "help": true,
	"install": true, "investigate": true, "launch": true, "look": true,
	"mail": true, "make": true, "meet": true, "message": true, "move": true,
	"notify": true, "open": true, "order": true, "organize": true, "pack": true,
	"pay": true, "pick": true, "plan": true, "post": true, "practice": true,
	"prepare": true, "print": true, "pull": true, "push": true, "put": true,
	"read": true, "register": true, "remind": true, "remove": true,
	"renew": true, "reply": true, "request": true, "reschedule": true,
	"respond": true, "return": true, "review": true, "run": true,
	"scan": true, "schedule": true, "send": true, "set": true, "setup": true,
	"share": true, "ship": true, "sign": true, "sort": true, "start": true,
	"stop": true, "submit": true, "take": true, "talk": true, "tell": true,
	"test": true, "text": true, "throw": true, "transfer": true, "try": true,
	"turn": true, "update": true, "upload": true, "visit": true, "walk": true,
	"wash": true, "watch": true, "work": true, "write": true,
}

// hasActionVerb reports whether text contains at least one word from actionVerbs.
func hasActionVerb(text string) bool {
	for _, w := range strings.Fields(strings.ToLower(text)) {
		if actionVerbs[w] {
			return true
		}
	}
	return false
}

// splitOnAnd tentatively splits text on " and " then merges any verb-less
// segment back into its predecessor. This preserves compound objects
// ("milk and eggs") while splitting independent tasks ("buy milk and call john").
func splitOnAnd(text string) []string {
	andPattern := regexp.MustCompile(`(?i) and `)
	parts := andPattern.Split(text, -1)

	if len(parts) <= 1 {
		return []string{strings.TrimSpace(text)}
	}

	var merged []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if len(merged) == 0 || hasActionVerb(p) {
			merged = append(merged, p)
		} else {
			// No verb — merge back with the preceding clause.
			merged[len(merged)-1] = merged[len(merged)-1] + " and " + p
		}
	}

	return merged
}

// SplitClauses splits text into discrete clauses on sentence-ending punctuation,
// discourse markers (" oh and ", " also "), and — when a verb heuristic confirms
// independent tasks — on " and ".
// It returns non-empty clauses in the order they appear, trimmed of whitespace.
func SplitClauses(text string) []string {
	if text == "" {
		return []string{}
	}

	// Phase 1: Replace strong discourse markers with a delimiter.
	// These almost always signal a new thought.
	result := text
	for _, marker := range []string{" oh and ", " also "} {
		pattern := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(marker))
		result = pattern.ReplaceAllString(result, "|")
	}

	// Phase 2: Split on sentence-ending punctuation.
	result = strings.NewReplacer(
		".", "|",
		"?", "|",
		"!", "|",
	).Replace(result)

	// Phase 3: Collect segments, then apply verb-aware "and" splitting per segment.
	parts := strings.Split(result, "|")

	var clauses []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		clauses = append(clauses, splitOnAnd(part)...)
	}

	return clauses
}

// expandCommaList distributes a leading action verb over comma-separated items.
// Example: "buy milk, bread, and eggs" → ["buy milk", "buy bread", "buy eggs"]
// If no action verb is found, returns the original text as a single-item slice.
func expandCommaList(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{}
	}

	// Check if text contains commas; if not, no expansion needed
	if !strings.Contains(text, ",") {
		return []string{text}
	}

	// Extract the first word to check if it's an action verb
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return []string{text}
	}

	firstWordLower := strings.ToLower(fields[0])
	if !actionVerbs[firstWordLower] {
		return []string{text}
	}

	// Find where the first word ends in the original text
	// to preserve casing
	spaceIdx := strings.Index(text, " ")
	if spaceIdx == -1 {
		return []string{text}
	}

	verb := text[:spaceIdx]
	afterVerb := strings.TrimSpace(text[spaceIdx:])
	if afterVerb == "" {
		return []string{text}
	}

	// Split on commas and clean up "and" prefix if present
	items := strings.Split(afterVerb, ",")
	var expanded []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		// Remove leading "and " if present
		if strings.HasPrefix(strings.ToLower(item), "and ") {
			item = strings.TrimSpace(item[4:])
		}
		if item != "" {
			expanded = append(expanded, verb+" "+item)
		}
	}

	if len(expanded) == 0 {
		return []string{text}
	}
	return expanded
}

// DetectSubordinateClause reports whether text appears to be a subordinate clause.
// A subordinate clause has:
//   - A subordinate marker: when, while, after, before, once, whenever, if
//   - Followed by a subject pronoun opener: I, we
//   - AND the subordinate segment contains at least one action verb
func DetectSubordinateClause(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}

	subordinateMarkers := []string{
		"whenever", "when", "while", "after", "before", "once", "if",
	}

	lowerText := strings.ToLower(text)
	var foundMarker bool
	var markerEndIdx int

	// Find the first subordinate marker
	for _, marker := range subordinateMarkers {
		pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(marker) + `\b`)
		matches := pattern.FindStringIndex(lowerText)
		if matches != nil {
			foundMarker = true
			markerEndIdx = matches[1] // End of marker
			break
		}
	}

	if !foundMarker {
		return false
	}

	// Extract text after the marker
	afterMarker := strings.TrimSpace(lowerText[markerEndIdx:])
	if afterMarker == "" {
		return false
	}

	// Check for subject pronoun opener (I, we)
	openerPattern := regexp.MustCompile(`^(i|we)\b`)
	if !openerPattern.MatchString(afterMarker) {
		return false
	}

	// Check for action verb in the subordinate segment
	return hasActionVerb(text[markerEndIdx:])
}

// SplitClausesWithRoles splits text into Clause objects with role and sibling
// index metadata. It builds on SplitClauses logic, detecting subordinate clauses
// and expanding comma lists before role assignment.
func SplitClausesWithRoles(text string) []Clause {
	if text == "" {
		return []Clause{}
	}

	// Process with standard SplitClauses
	clauses := SplitClauses(text)
	if len(clauses) == 0 {
		return []Clause{}
	}

	var result []Clause
	expandedIdx := 0
	mainIdx := 0

	for _, clause := range clauses {
		// Check if this clause is subordinate
		if DetectSubordinateClause(clause) {
			result = append(result, Clause{
				Text:       clause,
				Role:       RoleSubordinate,
				SiblingIdx: mainIdx,
			})
			mainIdx++
		} else {
			// Try to expand comma list
			expanded := expandCommaList(clause)
			if len(expanded) > 1 {
				// Multiple items from expansion
				for _, item := range expanded {
					result = append(result, Clause{
						Text:       item,
						Role:       RoleExpandedFromCommaList,
						SiblingIdx: expandedIdx,
					})
					expandedIdx++
				}
			} else {
				// Single clause, mark as main
				result = append(result, Clause{
					Text:       clause,
					Role:       RoleMain,
					SiblingIdx: mainIdx,
				})
				mainIdx++
			}
		}
	}

	return result
}
