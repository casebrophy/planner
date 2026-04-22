package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/casebrophy/planner/business/domain/ingestbus/classify"
	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
)

// FixtureContext is a context in a fixture (includes kind, unlike extractor.ContextRef).
type FixtureContext struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Kind  string `json:"kind"` // "project", "area", "list"
}

// FixtureExpected is the set of optional assertions for a fixture.
type FixtureExpected struct {
	PrimaryType        string   `json:"primary_type,omitempty"`           // "task", "event", "note", "mixed"
	TitleContains      []string `json:"title_contains,omitempty"`         // any action item or event title must contain ALL these strings (case-insensitive)
	ContextID          string   `json:"context_id,omitempty"`             // exact fixture context ID, or "new" (asserts SuggestNewContext=true)
	ContextKind        string   `json:"context_kind,omitempty"`           // for known context_id: look it up in active_contexts and check kind matches
	MinActionItems     int      `json:"min_action_items,omitempty"`       // 0 = no minimum
	MaxActionItems     int      `json:"max_action_items,omitempty"`       // 0 = no maximum
	ForbidNotes        bool     `json:"forbid_notes,omitempty"`
	ForbidEvents       bool     `json:"forbid_events,omitempty"`
	MinContextConf     float64  `json:"min_context_confidence,omitempty"` // 0 = no minimum
	MaxContextConf     float64  `json:"max_context_confidence,omitempty"` // 0 = no maximum
}

// Fixture is a single eval test case.
type Fixture struct {
	Name           string           `json:"name"`
	Comment        string           `json:"comment,omitempty"` // one-line reason this case is included
	Source         string           `json:"source"`            // "voice", "text", "email"
	InputText      string           `json:"input_text"`
	ActiveContexts []FixtureContext `json:"active_contexts"`
	Now            string           `json:"now,omitempty"`      // RFC3339 — stored for reference, not passed to extractor
	Timezone       string           `json:"timezone,omitempty"` // IANA tz — same
	Expected       FixtureExpected  `json:"expected"`
}

// FixtureResult is the outcome of running a single fixture.
type FixtureResult struct {
	Fixture    Fixture
	Extraction extractor.TextExtraction
	Pass       bool
	Failures   []string
	LatencyMS  int64
}

// LoadFixtures reads all *.json files in dir, parses them as Fixture.
func LoadFixtures(dir string) ([]Fixture, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %q: %w", dir, err)
	}

	var fixtures []Fixture
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read file %q: %w", path, err)
		}
		var f Fixture
		if err := json.Unmarshal(data, &f); err != nil {
			return nil, fmt.Errorf("parse fixture %q: %w", path, err)
		}
		fixtures = append(fixtures, f)
	}
	return fixtures, nil
}

// Run runs each fixture through the extractor.
// It calls classify.Classify on the input text to get a typeHint (same as the real pipeline),
// then calls extractor.ExtractText with empty userCorrection, empty candidates, empty contextAnnotations.
// Returns one FixtureResult per fixture.
func Run(ctx context.Context, ext extractor.Extractor, fixtures []Fixture) []FixtureResult {
	results := make([]FixtureResult, 0, len(fixtures))

	for _, f := range fixtures {
		cl := classify.Classify(f.InputText)
		typeHint := string(cl.Type)

		refs := make([]extractor.ContextRef, 0, len(f.ActiveContexts))
		for _, ac := range f.ActiveContexts {
			refs = append(refs, extractor.ContextRef{
				ID:    ac.ID,
				Title: ac.Title,
			})
		}

		start := time.Now().UnixMilli()
		ex, err := ext.ExtractText(ctx, f.InputText, "", refs, typeHint, []extractor.EntityMatch{}, []string{})
		latency := time.Now().UnixMilli() - start

		var failures []string
		if err != nil {
			failures = append(failures, fmt.Sprintf("extractor error: %v", err))
		} else {
			failures = assert(f, ex)
		}

		results = append(results, FixtureResult{
			Fixture:    f,
			Extraction: ex,
			Pass:       len(failures) == 0,
			Failures:   failures,
			LatencyMS:  latency,
		})
	}

	return results
}

// assert checks each non-zero/non-false field in expected against the extraction result.
// Returns a list of failure strings (empty = pass).
func assert(fixture Fixture, ex extractor.TextExtraction) []string {
	exp := fixture.Expected
	var failures []string

	// PrimaryType assertion
	if exp.PrimaryType != "" {
		got := primaryType(ex)
		if got != exp.PrimaryType {
			failures = append(failures, fmt.Sprintf("primary_type: expected %q, got %q", exp.PrimaryType, got))
		}
	}

	// TitleContains assertion
	if len(exp.TitleContains) > 0 {
		found := false
		// Check ActionItems
		for _, item := range ex.ActionItems {
			if titleMatchesAll(item.Title, exp.TitleContains) {
				found = true
				break
			}
		}
		// Check Events
		if !found {
			for _, ev := range ex.Events {
				if titleMatchesAll(ev.Title, exp.TitleContains) {
					found = true
					break
				}
			}
		}
		if !found {
			failures = append(failures, fmt.Sprintf("title_contains: no title matched all of %v", exp.TitleContains))
		}
	}

	// ContextID assertion
	if exp.ContextID == "new" {
		if !ex.SuggestNewContext {
			failures = append(failures, "context_id(new): expected SuggestNewContext=true, got false")
		}
	} else if exp.ContextID != "" {
		if ex.SuggestedContextID == nil || *ex.SuggestedContextID != exp.ContextID {
			got := "<nil>"
			if ex.SuggestedContextID != nil {
				got = *ex.SuggestedContextID
			}
			failures = append(failures, fmt.Sprintf("context_id: expected %q, got %q", exp.ContextID, got))
		}
	}

	// ContextKind assertion (only when ContextID is set and not "new")
	if exp.ContextKind != "" && exp.ContextID != "" && exp.ContextID != "new" {
		// Look up in fixture.ActiveContexts
		foundCtx := false
		for _, ac := range fixture.ActiveContexts {
			if ac.ID == exp.ContextID {
				foundCtx = true
				if ac.Kind != exp.ContextKind {
					failures = append(failures, fmt.Sprintf("context_kind: context %q has kind %q, expected %q", exp.ContextID, ac.Kind, exp.ContextKind))
				}
				break
			}
		}
		if !foundCtx {
			failures = append(failures, fmt.Sprintf("context_kind: context_id %q not found in active_contexts", exp.ContextID))
		}
	}

	// MinActionItems assertion
	if exp.MinActionItems > 0 {
		if len(ex.ActionItems) < exp.MinActionItems {
			failures = append(failures, fmt.Sprintf("min_action_items: expected >= %d, got %d", exp.MinActionItems, len(ex.ActionItems)))
		}
	}

	// MaxActionItems assertion
	if exp.MaxActionItems > 0 {
		if len(ex.ActionItems) > exp.MaxActionItems {
			failures = append(failures, fmt.Sprintf("max_action_items: expected <= %d, got %d", exp.MaxActionItems, len(ex.ActionItems)))
		}
	}

	// ForbidNotes assertion
	if exp.ForbidNotes {
		if len(ex.Notes) > 0 {
			failures = append(failures, fmt.Sprintf("forbid_notes: expected 0 notes, got %d", len(ex.Notes)))
		}
	}

	// ForbidEvents assertion
	if exp.ForbidEvents {
		if len(ex.Events) > 0 {
			failures = append(failures, fmt.Sprintf("forbid_events: expected 0 events, got %d", len(ex.Events)))
		}
	}

	// MinContextConf assertion
	if exp.MinContextConf > 0 {
		if ex.ContextConfidence < exp.MinContextConf {
			failures = append(failures, fmt.Sprintf("min_context_confidence: expected >= %.2f, got %.2f", exp.MinContextConf, ex.ContextConfidence))
		}
	}

	// MaxContextConf assertion
	if exp.MaxContextConf > 0 {
		if ex.ContextConfidence > exp.MaxContextConf {
			failures = append(failures, fmt.Sprintf("max_context_confidence: expected <= %.2f, got %.2f", exp.MaxContextConf, ex.ContextConfidence))
		}
	}

	return failures
}

// primaryType returns "task", "event", "note", "mixed", or "unknown" from a TextExtraction.
func primaryType(ex extractor.TextExtraction) string {
	hasActions := len(ex.ActionItems) > 0
	hasEvents := len(ex.Events) > 0
	hasNotes := len(ex.Notes) > 0

	switch {
	case hasActions && (hasEvents || hasNotes):
		return "mixed"
	case hasEvents && hasNotes:
		return "mixed"
	case hasActions:
		return "task"
	case hasEvents:
		return "event"
	case hasNotes:
		return "note"
	default:
		return "unknown"
	}
}

// titleMatchesAll returns true if title contains ALL strings in required (case-insensitive).
func titleMatchesAll(title string, required []string) bool {
	lower := strings.ToLower(title)
	for _, s := range required {
		if !strings.Contains(lower, strings.ToLower(s)) {
			return false
		}
	}
	return true
}
