package dailyplandb

import (
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/dailyplanbus"
)

type dailyPlanDB struct {
	ID         uuid.UUID `db:"plan_id"`
	PlanDate   time.Time `db:"plan_date"`
	Generation int       `db:"generation"`
	ModelUsed  string    `db:"model_used"`
	PromptHash *string   `db:"prompt_hash"`
	CreatedAt  time.Time `db:"created_at"`
}

type dailyPlanItemDB struct {
	ID               uuid.UUID  `db:"item_id"`
	PlanID           uuid.UUID  `db:"plan_id"`
	TaskID           uuid.UUID  `db:"task_id"`
	Position         int        `db:"position"`
	GroupName        string     `db:"group_name"`
	GroupPosition    int        `db:"group_position"`
	AIDurationMin    *int       `db:"ai_duration_min"`
	AIPriorityReason *string    `db:"ai_priority_reason"`
	UserPosition     *int       `db:"user_position"`
	UserDurationMin  *int       `db:"user_duration_min"`
	Status           string     `db:"status"`
	DismissReason    *string    `db:"dismiss_reason"`
	DismissNote      *string    `db:"dismiss_note"`
	CompletedAt      *time.Time `db:"completed_at"`
	CreatedAt        time.Time  `db:"created_at"`
}

func toDBPlan(p dailyplanbus.DailyPlan) dailyPlanDB {
	return dailyPlanDB{
		ID:         p.ID,
		PlanDate:   p.PlanDate,
		Generation: p.Generation,
		ModelUsed:  p.ModelUsed,
		PromptHash: p.PromptHash,
		CreatedAt:  p.CreatedAt,
	}
}

func toBusPlan(p dailyPlanDB) dailyplanbus.DailyPlan {
	return dailyplanbus.DailyPlan{
		ID:         p.ID,
		PlanDate:   p.PlanDate,
		Generation: p.Generation,
		ModelUsed:  p.ModelUsed,
		PromptHash: p.PromptHash,
		CreatedAt:  p.CreatedAt,
	}
}

func toDBItem(i dailyplanbus.DailyPlanItem) dailyPlanItemDB {
	return dailyPlanItemDB{
		ID:               i.ID,
		PlanID:           i.PlanID,
		TaskID:           i.TaskID,
		Position:         i.Position,
		GroupName:        i.GroupName,
		GroupPosition:    i.GroupPosition,
		AIDurationMin:    i.AIDurationMin,
		AIPriorityReason: i.AIPriorityReason,
		UserPosition:     i.UserPosition,
		UserDurationMin:  i.UserDurationMin,
		Status:           i.Status,
		DismissReason:    i.DismissReason,
		DismissNote:      i.DismissNote,
		CompletedAt:      i.CompletedAt,
		CreatedAt:        i.CreatedAt,
	}
}

func toBusItem(i dailyPlanItemDB) dailyplanbus.DailyPlanItem {
	return dailyplanbus.DailyPlanItem{
		ID:               i.ID,
		PlanID:           i.PlanID,
		TaskID:           i.TaskID,
		Position:         i.Position,
		GroupName:        i.GroupName,
		GroupPosition:    i.GroupPosition,
		AIDurationMin:    i.AIDurationMin,
		AIPriorityReason: i.AIPriorityReason,
		UserPosition:     i.UserPosition,
		UserDurationMin:  i.UserDurationMin,
		Status:           i.Status,
		DismissReason:    i.DismissReason,
		DismissNote:      i.DismissNote,
		CompletedAt:      i.CompletedAt,
		CreatedAt:        i.CreatedAt,
	}
}

func toBusItems(items []dailyPlanItemDB) []dailyplanbus.DailyPlanItem {
	result := make([]dailyplanbus.DailyPlanItem, len(items))
	for i, item := range items {
		result[i] = toBusItem(item)
	}
	return result
}
