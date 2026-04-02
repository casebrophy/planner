package observationbus

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/observationkind"
)

// TestGenerateNewObservations generates n unique NewObservation values for testing.
func TestGenerateNewObservations(subjectType string, subjectID uuid.UUID, n int) []NewObservation {
	obs := make([]NewObservation, n)
	idx := rand.Intn(10000)
	for i := range obs {
		idx++
		obs[i] = NewObservation{
			SubjectType: subjectType,
			SubjectID:   subjectID,
			Kind:        observationkind.Lesson,
			Data:        json.RawMessage(`{"key":"val"}`),
			Source:      "user",
			Confidence:  0.9,
			Weight:      1.0,
		}
	}
	return obs
}

// TestSeedObservations creates n observations in the database and returns them as
// stored (re-queried from DB so Data reflects JSONB normalization).
func TestSeedObservations(ctx context.Context, subjectType string, subjectID uuid.UUID, n int, api *Business) ([]Observation, error) {
	news := TestGenerateNewObservations(subjectType, subjectID, n)
	for i, no := range news {
		if _, err := api.Record(ctx, no); err != nil {
			return nil, fmt.Errorf("seeding observation: idx: %d : %w", i, err)
		}
	}

	// Re-read from DB so returned Data matches what the DB actually stores (JSONB-normalized).
	obs, err := api.QueryBySubject(ctx, subjectType, subjectID, page.New(1, 100))
	if err != nil {
		return nil, fmt.Errorf("querying seeded observations: %w", err)
	}
	return obs, nil
}
