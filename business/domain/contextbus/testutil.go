package contextbus

import (
	"context"
	"fmt"
	"math/rand"
)

// TestGenerateNewContexts generates n unique NewContext values for testing.
func TestGenerateNewContexts(n int) []NewContext {
	newContexts := make([]NewContext, n)
	idx := rand.Intn(10000)
	for i := range newContexts {
		idx++
		newContexts[i] = NewContext{
			Title:       fmt.Sprintf("Context%d", idx),
			Description: fmt.Sprintf("Description for context %d", idx),
		}
	}
	return newContexts
}

// TestSeedContexts creates n contexts in the database and returns them.
func TestSeedContexts(ctx context.Context, n int, api *Business) ([]Context, error) {
	newContexts := TestGenerateNewContexts(n)
	contexts := make([]Context, len(newContexts))
	for i, nc := range newContexts {
		c, err := api.Create(ctx, nc)
		if err != nil {
			return nil, fmt.Errorf("seeding context: idx: %d : %w", i, err)
		}
		contexts[i] = c
	}
	return contexts, nil
}
