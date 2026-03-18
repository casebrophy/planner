package rawinputbus

import (
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/types/rawinputsource"
	"github.com/casebrophy/planner/business/types/rawinputstatus"
)

type RawInput struct {
	ID          uuid.UUID
	SourceType  rawinputsource.Source
	Status      rawinputstatus.Status
	RawContent  string
	ProcessedAt *time.Time
	Error       *string
	CreatedAt   time.Time
}

type NewRawInput struct {
	SourceType rawinputsource.Source
	RawContent string
}

type UpdateRawInput struct {
	Status      *rawinputstatus.Status
	ProcessedAt *time.Time
	Error       *string
}
