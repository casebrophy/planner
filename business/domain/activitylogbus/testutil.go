package activitylogbus

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/google/uuid"
)

// TestGenerateNewLogs generates n unique NewLog values for testing.
func TestGenerateNewLogs(n int, subjectType string, subjectID uuid.UUID) []NewLog {
	logs := make([]NewLog, n)
	idx := rand.Intn(10000)
	for i := range logs {
		idx++
		val := fmt.Sprintf("activity-%d", idx)
		logs[i] = NewLog{
			SubjectType: subjectType,
			SubjectID:   subjectID,
			Value:       &val,
		}
	}
	return logs
}

// TestSeedLogs creates n activity logs in the database and returns them.
func TestSeedLogs(ctx context.Context, n int, subjectType string, subjectID uuid.UUID, api *Business) ([]Log, error) {
	newLogs := TestGenerateNewLogs(n, subjectType, subjectID)
	logs := make([]Log, len(newLogs))
	for i, nl := range newLogs {
		l, err := api.Create(ctx, nl)
		if err != nil {
			return nil, fmt.Errorf("seeding activity log: idx: %d : %w", i, err)
		}
		logs[i] = l
	}
	return logs, nil
}
