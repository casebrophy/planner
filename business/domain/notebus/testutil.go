package notebus

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/google/uuid"
)

// TestGenerateNewNotes generates n unique NewNote values for testing.
// contextID is used as the target (at least one target is required by DB constraint).
func TestGenerateNewNotes(n int, contextID uuid.UUID) []NewNote {
	newNotes := make([]NewNote, n)
	idx := rand.Intn(10000)
	for i := range newNotes {
		idx++
		newNotes[i] = NewNote{
			ContextID: &contextID,
			Content:   fmt.Sprintf("Note content %d", idx),
			Source:    "manual",
		}
	}
	return newNotes
}

// TestSeedNotes creates n notes in the database and returns them.
// contextID is used as the target for all seeded notes.
func TestSeedNotes(ctx context.Context, n int, api *Business, contextID uuid.UUID) ([]Note, error) {
	newNotes := TestGenerateNewNotes(n, contextID)
	notes := make([]Note, len(newNotes))
	for i, nn := range newNotes {
		n, err := api.Create(ctx, nn)
		if err != nil {
			return nil, fmt.Errorf("seeding note: idx: %d : %w", i, err)
		}
		notes[i] = n
	}
	return notes, nil
}
