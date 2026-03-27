package taskbus

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/casebrophy/planner/business/types/taskenergy"
	"github.com/casebrophy/planner/business/types/taskpriority"
	"github.com/casebrophy/planner/business/types/taskstatus"
)

// TestGenerateNewTasks generates n unique NewTask values for testing.
func TestGenerateNewTasks(n int) []NewTask {
	newTasks := make([]NewTask, n)
	idx := rand.Intn(10000)
	for i := range newTasks {
		idx++
		newTasks[i] = NewTask{
			Title:       fmt.Sprintf("Task%d", idx),
			Description: fmt.Sprintf("Description for task %d", idx),
			Status:      taskstatus.Todo,
			Priority:    taskpriority.Medium,
			Energy:      taskenergy.Medium,
		}
	}
	return newTasks
}

// TestSeedTasks creates n tasks in the database and returns them.
func TestSeedTasks(ctx context.Context, n int, api *Business) ([]Task, error) {
	newTasks := TestGenerateNewTasks(n)
	tasks := make([]Task, len(newTasks))
	for i, nt := range newTasks {
		t, err := api.Create(ctx, nt)
		if err != nil {
			return nil, fmt.Errorf("seeding task: idx: %d : %w", i, err)
		}
		tasks[i] = t
	}
	return tasks, nil
}
