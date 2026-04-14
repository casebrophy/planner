package clarificationbus

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/clarificationstatus"
)

// TestGenerateNewClarifications generates n unique NewClarificationItem values for testing.
func TestGenerateNewClarifications(n int) []NewClarificationItem {
	items := make([]NewClarificationItem, n)
	idx := rand.Intn(10000)
	for i := range items {
		idx++
		items[i] = NewClarificationItem{
			Kind:               clarificationkind.StaleTask,
			SubjectType:        "task",
			SubjectID:          uuid.New(),
			SubjectDescription: fmt.Sprintf("Test task %d", idx),
			Question:           fmt.Sprintf("Is item %d still relevant?", idx),
			AnswerOptions:      json.RawMessage(`["yes","no"]`),
			PriorityScore:      0.5,
		}
	}
	return items
}

// TestSeedClarifications creates n clarification items in the database and returns them
// re-queried from DB so AnswerOptions reflects JSONB normalization.
func TestSeedClarifications(ctx context.Context, n int, api *Business) ([]ClarificationItem, error) {
	newItems := TestGenerateNewClarifications(n)
	for i, ni := range newItems {
		if _, err := api.Create(ctx, ni); err != nil {
			return nil, fmt.Errorf("seeding clarification: idx: %d : %w", i, err)
		}
	}

	pendingStatus := clarificationstatus.Pending
	items, err := api.Query(ctx, QueryFilter{Status: &pendingStatus}, DefaultOrderBy, page.New(1, 100))
	if err != nil {
		return nil, fmt.Errorf("querying seeded clarifications: %w", err)
	}
	return items, nil
}
