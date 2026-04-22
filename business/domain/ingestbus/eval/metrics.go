package eval

import "strings"

// Metrics holds aggregate accuracy results.
type Metrics struct {
	TypeAccuracy       float64 // fraction of primary_type assertions that passed (out of fixtures with primary_type expected)
	ContextAccuracy    float64 // fraction of context_id assertions that passed (out of fixtures with context_id expected)
	ListAssignmentRate float64 // fraction of fixtures with expected context_kind=="list" that passed context_id assertion
	EscalationRate     float64 // always 0.0 in Track 1 (extractor doesn't expose model used)
	PerFixture         []FixtureResult
}

// Score computes aggregate metrics from a slice of FixtureResults.
func Score(results []FixtureResult) Metrics {
	var (
		typeTotal, typePass       int
		ctxTotal, ctxPass         int
		listTotal, listPass       int
	)

	for _, r := range results {
		exp := r.Fixture.Expected

		// TypeAccuracy: fixtures with PrimaryType set
		if exp.PrimaryType != "" {
			typeTotal++
			if !hasFailurePrefix(r.Failures, "primary_type") {
				typePass++
			}
		}

		// ContextAccuracy: fixtures with ContextID set
		if exp.ContextID != "" {
			ctxTotal++
			if !hasFailurePrefix(r.Failures, "context_id") {
				ctxPass++
			}
		}

		// ListAssignmentRate: fixtures with ContextKind=="list"
		if exp.ContextKind == "list" {
			listTotal++
			if !hasFailurePrefix(r.Failures, "context_id") && !hasFailurePrefix(r.Failures, "context_kind") {
				listPass++
			}
		}
	}

	m := Metrics{
		EscalationRate: 0.0,
		PerFixture:     results,
	}

	if typeTotal > 0 {
		m.TypeAccuracy = float64(typePass) / float64(typeTotal)
	}
	if ctxTotal > 0 {
		m.ContextAccuracy = float64(ctxPass) / float64(ctxTotal)
	}
	if listTotal > 0 {
		m.ListAssignmentRate = float64(listPass) / float64(listTotal)
	}

	return m
}

// hasFailurePrefix returns true if any failure string has the given prefix.
func hasFailurePrefix(failures []string, prefix string) bool {
	for _, f := range failures {
		if strings.HasPrefix(f, prefix) {
			return true
		}
	}
	return false
}
