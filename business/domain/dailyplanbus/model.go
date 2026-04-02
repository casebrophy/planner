package dailyplanbus

import (
	"time"

	"github.com/google/uuid"
)

type DailyPlan struct {
	ID         uuid.UUID
	PlanDate   time.Time
	Generation int
	ModelUsed  string
	PromptHash *string
	CreatedAt  time.Time
}

type NewDailyPlan struct {
	PlanDate   time.Time
	Generation int
	ModelUsed  string
	PromptHash *string
}

type DailyPlanItem struct {
	ID               uuid.UUID
	PlanID           uuid.UUID
	TaskID           uuid.UUID
	Position         int
	GroupName        string
	GroupPosition    int
	AIDurationMin    *int
	AIPriorityReason *string
	UserPosition     *int
	UserDurationMin  *int
	Status           string
	DismissReason    *string
	DismissNote      *string
	CompletedAt      *time.Time
	CreatedAt        time.Time
}

type NewDailyPlanItem struct {
	PlanID           uuid.UUID
	TaskID           uuid.UUID
	Position         int
	GroupName        string
	GroupPosition    int
	AIDurationMin    *int
	AIPriorityReason *string
}

type UpdatePlanItem struct {
	UserPosition    *int
	UserDurationMin *int
	Status          *string
	DismissReason   *string
	DismissNote     *string
	CompletedAt     *time.Time
}
