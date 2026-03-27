package tagbus

import (
	"context"
	"fmt"
	"math/rand"
)

// TestGenerateNewTags generates n unique NewTag values for testing.
func TestGenerateNewTags(n int) []NewTag {
	newTags := make([]NewTag, n)
	idx := rand.Intn(10000)
	for i := range newTags {
		idx++
		newTags[i] = NewTag{
			Name: fmt.Sprintf("tag%d", idx),
		}
	}
	return newTags
}

// TestSeedTags creates n tags in the database and returns them.
func TestSeedTags(ctx context.Context, n int, api *Business) ([]Tag, error) {
	newTags := TestGenerateNewTags(n)
	tags := make([]Tag, len(newTags))
	for i, nt := range newTags {
		t, err := api.Create(ctx, nt)
		if err != nil {
			return nil, fmt.Errorf("seeding tag: idx: %d : %w", i, err)
		}
		tags[i] = t
	}
	return tags, nil
}
