package classificationcorrectionbus

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/casebrophy/planner/business/sdk/page"
)

// TestGenerateNewCorrections generates n unique NewCorrection values for testing.
func TestGenerateNewCorrections(n int) []NewCorrection {
	corrs := make([]NewCorrection, n)
	idx := rand.Intn(10000)
	for i := range corrs {
		idx++
		corrs[i] = NewCorrection{
			ClauseText:    fmt.Sprintf("clause_%d", idx),
			PredictedType: "task",
			Confidence:    0.85,
			ActualType:    "context",
			Source:        "clarification_answered",
		}
	}
	return corrs
}

// TestSeedCorrections creates n corrections in the database and returns them as
// stored (re-queried from DB).
func TestSeedCorrections(ctx context.Context, n int, api *Business) ([]Correction, error) {
	news := TestGenerateNewCorrections(n)
	for i, nc := range news {
		if _, err := api.Record(ctx, nc); err != nil {
			return nil, fmt.Errorf("seeding correction: idx: %d : %w", i, err)
		}
	}

	// Re-read from DB.
	filter := QueryFilter{}
	corrs, err := api.Query(ctx, filter, DefaultOrderBy, page.New(1, 100))
	if err != nil {
		return nil, fmt.Errorf("querying seeded corrections: %w", err)
	}
	return corrs, nil
}
