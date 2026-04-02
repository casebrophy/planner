package timeblockapp

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/timeblockbus"
)

type TimeBlock struct {
	ID        string `json:"id"`
	TaskID    string `json:"taskId"`
	StartsAt  string `json:"startsAt"`
	EndsAt    string `json:"endsAt"`
	Confirmed bool   `json:"confirmed"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func (tb TimeBlock) Encode() ([]byte, string, error) {
	data, err := json.Marshal(tb)
	return data, "application/json", err
}

type NewTimeBlock struct {
	TaskID   string `json:"taskId"`
	StartsAt string `json:"startsAt"`
	EndsAt   string `json:"endsAt"`
}

type UpdateTimeBlock struct {
	StartsAt  *string `json:"startsAt"`
	EndsAt    *string `json:"endsAt"`
	Confirmed *bool   `json:"confirmed"`
}

func toAppTimeBlock(tb timeblockbus.TimeBlock) TimeBlock {
	return TimeBlock{
		ID:        tb.ID.String(),
		TaskID:    tb.TaskID.String(),
		StartsAt:  tb.StartsAt.Format(time.RFC3339),
		EndsAt:    tb.EndsAt.Format(time.RFC3339),
		Confirmed: tb.Confirmed,
		CreatedAt: tb.CreatedAt.Format(time.RFC3339),
		UpdatedAt: tb.UpdatedAt.Format(time.RFC3339),
	}
}

func toAppTimeBlocks(tbs []timeblockbus.TimeBlock) []TimeBlock {
	blocks := make([]TimeBlock, len(tbs))
	for i, tb := range tbs {
		blocks[i] = toAppTimeBlock(tb)
	}
	return blocks
}

func toBusNewTimeBlock(ntb NewTimeBlock) (timeblockbus.NewTimeBlock, error) {
	taskID, err := uuid.Parse(ntb.TaskID)
	if err != nil {
		return timeblockbus.NewTimeBlock{}, fmt.Errorf("taskId: %w", err)
	}

	startsAt, err := time.Parse(time.RFC3339, ntb.StartsAt)
	if err != nil {
		return timeblockbus.NewTimeBlock{}, fmt.Errorf("startsAt: %w", err)
	}

	endsAt, err := time.Parse(time.RFC3339, ntb.EndsAt)
	if err != nil {
		return timeblockbus.NewTimeBlock{}, fmt.Errorf("endsAt: %w", err)
	}

	return timeblockbus.NewTimeBlock{
		TaskID:   taskID,
		StartsAt: startsAt,
		EndsAt:   endsAt,
	}, nil
}

func toBusUpdateTimeBlock(utb UpdateTimeBlock) (timeblockbus.UpdateTimeBlock, error) {
	var butb timeblockbus.UpdateTimeBlock

	if utb.StartsAt != nil {
		t, err := time.Parse(time.RFC3339, *utb.StartsAt)
		if err != nil {
			return timeblockbus.UpdateTimeBlock{}, fmt.Errorf("startsAt: %w", err)
		}
		butb.StartsAt = &t
	}

	if utb.EndsAt != nil {
		t, err := time.Parse(time.RFC3339, *utb.EndsAt)
		if err != nil {
			return timeblockbus.UpdateTimeBlock{}, fmt.Errorf("endsAt: %w", err)
		}
		butb.EndsAt = &t
	}

	butb.Confirmed = utb.Confirmed

	return butb, nil
}
