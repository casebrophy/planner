package threadbus

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/types/threadentrykind"
	"github.com/casebrophy/planner/business/types/threadsource"
)

// TestGenerateNewThreadEntries generates n unique NewThreadEntry values for testing.
func TestGenerateNewThreadEntries(subjectType string, subjectID uuid.UUID, n int) []NewThreadEntry {
	entries := make([]NewThreadEntry, n)
	idx := rand.Intn(10000)
	for i := range entries {
		idx++
		entries[i] = NewThreadEntry{
			SubjectType: subjectType,
			SubjectID:   subjectID,
			Kind:        threadentrykind.Update,
			Content:     fmt.Sprintf("thread entry content %d", idx),
			Source:      threadsource.User,
		}
	}
	return entries
}

// TestSeedThreadEntries creates n thread entries in the database and returns them.
func TestSeedThreadEntries(ctx context.Context, subjectType string, subjectID uuid.UUID, n int, api *Business) ([]ThreadEntry, error) {
	news := TestGenerateNewThreadEntries(subjectType, subjectID, n)
	entries := make([]ThreadEntry, len(news))
	for i, ne := range news {
		e, err := api.AddEntry(ctx, ne)
		if err != nil {
			return nil, fmt.Errorf("seeding thread entry: idx: %d : %w", i, err)
		}
		entries[i] = e
	}
	return entries, nil
}
