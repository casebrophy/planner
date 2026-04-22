package classify

import (
	"regexp"
	"strings"
)

// ItemType represents the classification of an item.
type ItemType string

const (
	TaskType  ItemType = "task"
	EventType ItemType = "event"
	NoteType  ItemType = "note"
)

// Classification holds the result of classifying a clause.
type Classification struct {
	Type       ItemType
	Confidence float64 // 0.0 - 1.0
}

// obligationVerbs are action words that indicate a task.
var obligationVerbs = []string{
	"need to", "have to", "should", "must", "want to", "remember to",
	"don't forget", "make sure", "pick up", "call", "email", "text",
	"schedule", "book", "buy", "send", "finish", "complete",
	"review", "check", "update", "fix", "create", "write", "read", "plan",
	"setup", "set up", "install", "configure", "prepare", "arrange",
	"contact", "reach out", "follow up", "verify", "confirm", "sign",
	"approve", "reject", "delete", "remove", "add", "edit", "submit",
	"pay", "charge", "invoice", "bill", "order", "reserve", "book",
	"publish", "post", "share", "announce", "notify", "tell", "inform",
	"get rid of", "clean out", "clean up", "throw away", "toss",
}

// timePatterns match temporal anchors (dates, times, day names).
var timePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(monday|tuesday|wednesday|thursday|friday|saturday|sunday)\b`),
	regexp.MustCompile(`(?i)\b(jan|january|feb|february|mar|march|apr|april|may|jun|june|jul|july|aug|august|sep|september|oct|october|nov|november|dec|december)\b`),
	regexp.MustCompile(`(?i)\b(tomorrow|tonight|yesterday|today)\b`),
	regexp.MustCompile(`(?i)\b(next|last|this)\s+(monday|tuesday|wednesday|thursday|friday|saturday|sunday|week|month|year)\b`),
	regexp.MustCompile(`(?i)\bat\s+\d{1,2}(am|pm|:\d{2})\b`),
	regexp.MustCompile(`(?i)\b\d{1,2}(am|pm)\b`),
	regexp.MustCompile(`(?i)\b\d{1,2}:\d{2}\b`),
	regexp.MustCompile(`(?i)\b\d{4}-\d{2}-\d{2}\b`), // ISO 8601 date
	regexp.MustCompile(`(?i)\b\d{1,2}/\d{1,2}(/\d{2,4})?\b`), // MM/DD or MM/DD/YYYY
	regexp.MustCompile(`(?i)\b(in|on)\s+\d+\s+(day|days|week|weeks|month|months|year|years|hour|hours|minute|minutes)\b`),
	regexp.MustCompile(`(?i)\b(morning|afternoon|evening|night)\b`),
}

// phonePattern matches common phone number formats.
var phonePattern = regexp.MustCompile(`(?:\d{3}[-.\s]?)?\d{3}[-.\s]?\d{4}`)

// emailPattern matches email addresses.
var emailPattern = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)

// addressPattern matches street addresses.
var addressPattern = regexp.MustCompile(`(?i)\b\d+\s+[A-Za-z]+(\s+[A-Za-z]+)*\s+(st|street|ave|avenue|blvd|boulevard|ln|lane|rd|road|dr|drive|ct|court|circle|pl|place|pkwy|parkway)\b`)

// Classify returns the inferred type and confidence for a single clause.
func Classify(clause string) Classification {
	clause = strings.TrimSpace(clause)

	if clause == "" {
		return Classification{Type: NoteType, Confidence: 0.0}
	}

	lowerClause := strings.ToLower(clause)

	// Check for obligation verbs (task indicator).
	hasObligation := false
	for _, verb := range obligationVerbs {
		if strings.Contains(lowerClause, strings.ToLower(verb)) {
			hasObligation = true
			break
		}
	}

	// Check for temporal anchors (event indicator).
	hasTemporal := false
	for _, pattern := range timePatterns {
		if pattern.MatchString(clause) {
			hasTemporal = true
			break
		}
	}

	// Check for reference patterns (note indicator).
	hasPhone := phonePattern.MatchString(clause)
	hasEmail := emailPattern.MatchString(clause)
	hasAddress := addressPattern.MatchString(clause)
	hasFactPattern := (strings.Contains(lowerClause, " is ") && !strings.Contains(lowerClause, " is at ")) || strings.Contains(lowerClause, "the best ")

	// Classification logic with confidence scoring.
	switch {
	// Pure reference → high confidence note.
	case hasPhone || hasEmail || hasAddress:
		return Classification{Type: NoteType, Confidence: 0.95}

	// Fact patterns without obligation → note.
	case hasFactPattern && !hasObligation && !hasTemporal:
		return Classification{Type: NoteType, Confidence: 0.85}

	// Obligation verb without temporal → high confidence task.
	case hasObligation && !hasTemporal:
		return Classification{Type: TaskType, Confidence: 0.9}

	// Temporal anchor without obligation → high confidence event.
	case hasTemporal && !hasObligation:
		return Classification{Type: EventType, Confidence: 0.9}

	// Both obligation and temporal → ambiguous, optimistic to task.
	case hasObligation && hasTemporal:
		return Classification{Type: TaskType, Confidence: 0.6}

	// Fact pattern with obligation → ambiguous, optimistic to task.
	case hasFactPattern && hasObligation:
		return Classification{Type: TaskType, Confidence: 0.65}

	// Fact pattern with temporal → ambiguous, optimistic to event.
	case hasFactPattern && hasTemporal:
		return Classification{Type: EventType, Confidence: 0.65}

	// Default: low confidence note.
	default:
		return Classification{Type: NoteType, Confidence: 0.5}
	}
}
