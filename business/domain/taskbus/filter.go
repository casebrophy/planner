package taskbus

import (
	"time"

	"github.com/google/uuid"
	"github.com/casebrophy/planner/business/types/taskpriority"
	"github.com/casebrophy/planner/business/types/taskstatus"
)

type QueryFilter struct {
	ID           *uuid.UUID
	Status       *taskstatus.Status
	Priority     *taskpriority.Priority
	ContextID    *uuid.UUID
	StartDueDate *time.Time
	EndDueDate   *time.Time
}
