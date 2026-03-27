package clarificationbus

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/types/clarificationkind"
)

// TestGenerateNewClarifications generates n unique NewClarificationItem values for testing.
func TestGenerateNewClarifications(n int) []NewClarificationItem {
	items := make([]NewClarificationItem, n)
	idx := rand.Intn(10000)
	for i := range items {
		idx++
		items[i] = NewClarificationItem{
			Kind:          clarificationkind.StaleTask,
			SubjectType:   "task",
			SubjectID:     uuid.New(),
			Question:      fmt.Sprintf("Is item %d still relevant?", idx),
			AnswerOptions: json.RawMessage(`["yes","no"]`),
			PriorityScore: 0.5,
		}
	}
	return items
}

// TestSeedClarifications creates n clarification items in the database and returns them.
func TestSeedClarifications(ctx context.Context, n int, api *Business) ([]ClarificationItem, error) {
	newItems := TestGenerateNewClarifications(n)
	items := make([]ClarificationItem, len(newItems))
	for i, ni := range newItems {
		item, err := api.Create(ctx, ni)
		if err != nil {
			return nil, fmt.Errorf("seeding clarification: idx: %d : %w", i, err)
		}
		items[i] = item
	}
	return items, nil
}
