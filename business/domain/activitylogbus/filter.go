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
