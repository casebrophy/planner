package taskbus

import (
	"time"

	"github.com/google/uuid"
	"github.com/casebrophy/planner/business/types/debriefstatus"
	"github.com/casebrophy/planner/business/types/taskenergy"
	"github.com/casebrophy/planner/business/types/taskpriority"
	"github.com/casebrophy/planner/business/types/taskstatus"
)

type Task struct {
	ID                 uuid.UUID
	ContextID          *uuid.UUID
	Title              string
	Description        string
	Status             taskstatus.Status
	Priority           taskpriority.Priority
	Energy             taskenergy.Energy
	DurationMin        *int
	DueDate            *time.Time
	ScheduledAt        *time.Time
	ExpectedUpdateDays *float64
	LastThreadAt       *time.Time
	DebriefStatus      debriefstatus.Status
	CreatedAt          time.Time
	UpdatedAt          time.Time
	CompletedAt        *time.Time
}

type NewTask struct {
	Title       string
	Description string
	ContextID   *uuid.UUID
	Status      taskstatus.Status
	Priority    taskpriority.Priority
	Energy      taskenergy.Energy
	DurationMin *int
	DueDate     *time.Time
}

type UpdateTask struct {
	Title              *string
	Description        *string
	ContextID          *uuid.UUID
	Status             *taskstatus.Status
	Priority           *taskpriority.Priority
	Energy             *taskenergy.Energy
	DurationMin        *int
	DueDate            *time.Time
	ScheduledAt        *time.Time
	ExpectedUpdateDays *float64
	DebriefStatus      *debriefstatus.Status
}
