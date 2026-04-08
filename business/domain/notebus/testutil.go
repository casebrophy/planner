package notebus

import (
	"context"
	"fmt"
	"math/rand"
)

// TestGenerateNewNotes generates n unique NewNote values for testing.
func TestGenerateNewNotes(n int) []NewNote {
	newNotes := make([]NewNote, n)
	idx := rand.Intn(10000)
	for i := range newNotes {
		idx++
		newNotes[i] = NewNote{
			Content: fmt.Sprintf("Note content %d", idx),
			Source:  "manual",
		}
	}
	return newNotes
}

// TestSeedNotes creates n notes in the database and returns them.
func TestSeedNotes(ctx context.Context, n int, api *Business) ([]Note, error) {
	newNotes := TestGenerateNewNotes(n)
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
