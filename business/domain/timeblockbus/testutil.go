package timeblockbus

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TestGenerateNewTimeBlocks generates n unique NewTimeBlock values for testing.
// Requires a valid taskID since TimeBlock has a FK to tasks.
func TestGenerateNewTimeBlocks(n int, taskID uuid.UUID) []NewTimeBlock {
	blocks := make([]NewTimeBlock, n)
	baseTime := time.Now().Add(time.Hour)
	for i := range blocks {
		blocks[i] = NewTimeBlock{
			TaskID:   taskID,
			StartsAt: baseTime.Add(time.Duration(i*2) * time.Hour),
			EndsAt:   baseTime.Add(time.Duration(i*2+1) * time.Hour),
		}
	}
	return blocks
}

// TestSeedTimeBlocks creates n time blocks in the database and returns them.
func TestSeedTimeBlocks(ctx context.Context, n int, taskID uuid.UUID, api *Business) ([]TimeBlock, error) {
	newBlocks := TestGenerateNewTimeBlocks(n, taskID)
	blocks := make([]TimeBlock, len(newBlocks))
	for i, nb := range newBlocks {
		b, err := api.Create(ctx, nb)
		if err != nil {
			return nil, fmt.Errorf("seeding time block: idx: %d : %w", i, err)
		}
		blocks[i] = b
	}
	return blocks, nil
}
