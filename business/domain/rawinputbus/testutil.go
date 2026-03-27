package rawinputbus

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/casebrophy/planner/business/types/rawinputsource"
)

// TestGenerateNewRawInputs generates n unique NewRawInput values for testing.
func TestGenerateNewRawInputs(n int) []NewRawInput {
	inputs := make([]NewRawInput, n)
	idx := rand.Intn(10000)
	for i := range inputs {
		idx++
		inputs[i] = NewRawInput{
			SourceType: rawinputsource.Email,
			RawContent: fmt.Sprintf("raw content %d", idx),
		}
	}
	return inputs
}

// TestSeedRawInputs creates n raw inputs in the database and returns them.
func TestSeedRawInputs(ctx context.Context, n int, api *Business) ([]RawInput, error) {
	news := TestGenerateNewRawInputs(n)
	ris := make([]RawInput, len(news))
	for i, nr := range news {
		ri, err := api.Create(ctx, nr)
		if err != nil {
			return nil, fmt.Errorf("seeding raw input: idx: %d : %w", i, err)
		}
		ris[i] = ri
	}
	return ris, nil
}
