package activitylogbus

import (
	"time"

	"github.com/google/uuid"
)

type QueryFilter struct {
	SubjectType *string
	SubjectID   *uuid.UUID
	StartDate   *time.Time
	EndDate     *time.Time
}

// QueryBySubjectsFilter is used for bulk queries across multiple subjects.
type QueryBySubjectsFilter struct {
	SubjectType string
	SubjectIDs  []uuid.UUID
	From        time.Time
	To          time.Time
}
