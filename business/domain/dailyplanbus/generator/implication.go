package generator

import (
	"strings"
	"time"
	"unicode"
)

// ImplicationResult represents a detected prerequisite relationship between a task and an event.
type ImplicationResult struct {
	EventID       string
	EventTitle    string
	EventStartsAt time.Time
	TaskID        string
	TaskTitle     string
	Confidence    float64
}

// prepKeywords are prefixes that suggest a task is prep work for another commitment.
var prepKeywords = []string{
	"prep", "prepare", "preparing", "preparation",
	"review", "reviewing",
	"draft", "drafting",
	"write", "writing",
	"slides", "deck", "presentation",
	"agenda", "notes",
	"brief", "briefing",
	"research",
	"plan", "planning",
	"outline",
}

// stopWords are common English words excluded from keyword matching.
var stopWords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
	"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
	"with": true, "by": true, "from": true, "up": true, "about": true,
	"into": true, "through": true, "during": true, "this": true, "that": true,
}

// ReasonImplications identifies tasks that are likely prerequisites for upcoming events.
// It uses keyword matching: prep-related keywords in the task title and/or shared
// meaningful keywords between the task title and event title.
// Only events that start at least 30 minutes in the future (from now) are considered.
func ReasonImplications(tasks []TaskRef, events []EventRef, now time.Time) []ImplicationResult {
	var results []ImplicationResult

	for _, ev := range events {
		if ev.AllDay {
			continue
		}

		startsAt, err := time.Parse(time.RFC3339, ev.StartsAt)
		if err != nil {
			continue
		}

		// Only reason about events with enough lead time to still prep.
		if !startsAt.After(now.Add(30 * time.Minute)) {
			continue
		}

		eventKeywords := extractKeywords(ev.Title)
		if len(eventKeywords) == 0 {
			continue
		}

		for _, task := range tasks {
			confidence := computeImplicationScore(task.Title, eventKeywords)
			if confidence >= 0.3 {
				results = append(results, ImplicationResult{
					EventID:       ev.ID,
					EventTitle:    ev.Title,
					EventStartsAt: startsAt,
					TaskID:        task.ID,
					TaskTitle:     task.Title,
					Confidence:    confidence,
				})
			}
		}
	}

	return results
}

// computeImplicationScore returns a [0,1] score estimating whether taskTitle is a prerequisite
// for an event described by eventKeywords.
//
// Keyword overlap with the event is required — a prep keyword alone is not sufficient to
// establish relevance, which would produce false positives against unrelated events.
func computeImplicationScore(taskTitle string, eventKeywords map[string]bool) float64 {
	taskWords := tokenize(taskTitle)

	// Overlap is required; without it the task is probably unrelated to this event.
	overlapCount := 0
	if n := len(eventKeywords); n > 0 {
		for _, w := range taskWords {
			if len(w) > 3 && eventKeywords[w] {
				overlapCount++
			}
		}
		if overlapCount == 0 {
			return 0
		}
	}

	var prepScore float64
	for _, w := range taskWords {
		for _, kw := range prepKeywords {
			if strings.HasPrefix(w, kw) {
				prepScore = 0.5
				break
			}
		}
		if prepScore > 0 {
			break
		}
	}

	n := len(eventKeywords)
	overlapScore := float64(overlapCount) / float64(n) * 0.8

	score := prepScore + overlapScore
	if score > 1.0 {
		score = 1.0
	}
	return score
}

// extractKeywords tokenizes a title, removing stop words and short words.
func extractKeywords(title string) map[string]bool {
	kw := make(map[string]bool)
	for _, w := range tokenize(title) {
		if len(w) > 3 && !stopWords[w] {
			kw[w] = true
		}
	}
	return kw
}

// tokenize lowercases a string and splits on whitespace, stripping non-letter runes.
func tokenize(s string) []string {
	raw := strings.Fields(strings.ToLower(s))
	out := make([]string, 0, len(raw))
	for _, w := range raw {
		w = strings.TrimFunc(w, func(r rune) bool { return !unicode.IsLetter(r) })
		if w != "" {
			out = append(out, w)
		}
	}
	return out
}
