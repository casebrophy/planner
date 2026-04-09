package activitylogbus

import (
	"time"

	"github.com/google/uuid"
)

type Log struct {
	ID          uuid.UUID
	SubjectType string
	SubjectID   uuid.UUID
	Value       *string
	LoggedAt    time.Time
}

type NewLog struct {
	SubjectType string
	SubjectID   uuid.UUID
	Value       *string
}

type StreakInfo struct {
	Current    int
	Longest    int
	TotalCount int
	LastLogged *time.Time
}

// QueryBySubjectsFilter defines the filter for bulk subject queries.
type QueryBySubjectsFilter struct {
	SubjectType string
	SubjectIDs  []uuid.UUID
	From        time.Time
	To          time.Time
}
