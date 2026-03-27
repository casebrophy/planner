package observationbus

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/google/uuid"

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
			Data:        json.RawMessage(`{"key": "val"}`),
			Source:      fmt.Sprintf("test-source-%d", idx),
			Confidence:  0.9,
			Weight:      1.0,
		}
	}
	return obs
}

// TestSeedObservations creates n observations in the database and returns them.
func TestSeedObservations(ctx context.Context, subjectType string, subjectID uuid.UUID, n int, api *Business) ([]Observation, error) {
	news := TestGenerateNewObservations(subjectType, subjectID, n)
	obs := make([]Observation, len(news))
	for i, no := range news {
		o, err := api.Record(ctx, no)
		if err != nil {
			return nil, fmt.Errorf("seeding observation: idx: %d : %w", i, err)
		}
		obs[i] = o
	}
	return obs, nil
}
