package eval

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
			if !hasFailure(r.Failures, PrimaryType) {
				typePass++
			}
		}

		// ContextAccuracy: fixtures with ContextID set
		if exp.ContextID != "" {
			ctxTotal++
			if !hasFailure(r.Failures, ContextID) {
				ctxPass++
			}
		}

		// ListAssignmentRate: fixtures with ContextKind=="list"
		if exp.ContextKind == "list" {
			listTotal++
			if !hasFailure(r.Failures, ContextID) && !hasFailure(r.Failures, ContextKind) {
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

// hasFailure returns true if any failure has the given kind.
func hasFailure(failures []Failure, kind FailureKind) bool {
	for _, f := range failures {
		if f.Kind == kind {
			return true
		}
	}
	return false
}
