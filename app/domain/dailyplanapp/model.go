package dailyplanapp

import (
	"encoding/json"
	"time"

	"github.com/casebrophy/planner/business/domain/dailyplanbus"
)

type DailyPlan struct {
	ID         string         `json:"id"`
	PlanDate   string         `json:"planDate"`
	Generation int            `json:"generation"`
	ModelUsed  string         `json:"modelUsed"`
	CreatedAt  string         `json:"createdAt"`
	Items      []DailyPlanItem `json:"items"`
}

func (dp DailyPlan) Encode() ([]byte, string, error) {
	data, err := json.Marshal(dp)
	return data, "application/json", err
}

type DailyPlanItem struct {
	ID               string  `json:"id"`
	PlanID           string  `json:"planId"`
	TaskID           string  `json:"taskId"`
	Position         int     `json:"position"`
	GroupName        string  `json:"groupName"`
	GroupPosition    int     `json:"groupPosition"`
	AIDurationMin    *int    `json:"aiDurationMin,omitempty"`
	AIPriorityReason *string `json:"aiPriorityReason,omitempty"`
	UserPosition     *int    `json:"userPosition,omitempty"`
	UserDurationMin  *int    `json:"userDurationMin,omitempty"`
	Status           string  `json:"status"`
	DismissReason    *string `json:"dismissReason,omitempty"`
	DismissNote      *string `json:"dismissNote,omitempty"`
	CompletedAt      *string `json:"completedAt,omitempty"`
	CreatedAt        string  `json:"createdAt"`
}

func (dpi DailyPlanItem) Encode() ([]byte, string, error) {
	data, err := json.Marshal(dpi)
	return data, "application/json", err
}

// GenerateAccepted is returned immediately when plan generation is enqueued.
type GenerateAccepted struct {
	Status string `json:"status"`
}

func (g GenerateAccepted) Encode() ([]byte, string, error) {
	data, err := json.Marshal(g)
	return data, "application/json", err
}

type DismissRequest struct {
	Reason string  `json:"reason"` // not_today, blocked, too_long, not_important, other
	Note   *string `json:"note"`
}

type UpdateItemRequest struct {
	UserPosition    *int `json:"userPosition"`
	UserDurationMin *int `json:"userDurationMin"`
}

func toAppPlan(plan dailyplanbus.DailyPlan, items []dailyplanbus.DailyPlanItem) DailyPlan {
	appItems := make([]DailyPlanItem, len(items))
	for i, item := range items {
		appItems[i] = toAppItem(item)
	}

	return DailyPlan{
		ID:         plan.ID.String(),
		PlanDate:   plan.PlanDate.Format("2006-01-02"),
		Generation: plan.Generation,
		ModelUsed:  plan.ModelUsed,
		CreatedAt:  plan.CreatedAt.Format(time.RFC3339),
		Items:      appItems,
	}
}

func toAppItem(item dailyplanbus.DailyPlanItem) DailyPlanItem {
	appItem := DailyPlanItem{
		ID:            item.ID.String(),
		PlanID:        item.PlanID.String(),
		TaskID:        item.TaskID.String(),
		Position:      item.Position,
		GroupName:     item.GroupName,
		GroupPosition: item.GroupPosition,
		Status:        item.Status,
		CreatedAt:     item.CreatedAt.Format(time.RFC3339),
	}

	if item.AIDurationMin != nil {
		appItem.AIDurationMin = item.AIDurationMin
	}
	if item.AIPriorityReason != nil {
		appItem.AIPriorityReason = item.AIPriorityReason
	}
	if item.UserPosition != nil {
		appItem.UserPosition = item.UserPosition
	}
	if item.UserDurationMin != nil {
		appItem.UserDurationMin = item.UserDurationMin
	}
	if item.DismissReason != nil {
		appItem.DismissReason = item.DismissReason
	}
	if item.DismissNote != nil {
		appItem.DismissNote = item.DismissNote
	}
	if item.CompletedAt != nil {
		appItem.CompletedAt = new(string)
		*appItem.CompletedAt = item.CompletedAt.Format(time.RFC3339)
	}

	return appItem
}

func toAppItems(items []dailyplanbus.DailyPlanItem) []DailyPlanItem {
	appItems := make([]DailyPlanItem, len(items))
	for i, item := range items {
		appItems[i] = toAppItem(item)
	}
	return appItems
}
