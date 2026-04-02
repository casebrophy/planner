package dailyplanbus

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/foundation/logger"
)

type Storer interface {
	CreatePlan(ctx context.Context, plan DailyPlan) error
	CreateItem(ctx context.Context, item DailyPlanItem) error
	UpdateItem(ctx context.Context, item DailyPlanItem) error
	QueryPlanByDate(ctx context.Context, date time.Time) (DailyPlan, error)
	QueryItemsByPlan(ctx context.Context, planID uuid.UUID) ([]DailyPlanItem, error)
	QueryItemByID(ctx context.Context, itemID uuid.UUID) (DailyPlanItem, error)
	DeleteItemsByPlan(ctx context.Context, planID uuid.UUID) error
}

type Business struct {
	log    *logger.Logger
	storer Storer
}

func NewBusiness(log *logger.Logger, storer Storer) *Business {
	return &Business{
		log:    log,
		storer: storer,
	}
}

func (b *Business) Create(ctx context.Context, ne NewDailyPlan) (DailyPlan, error) {
	now := time.Now()

	plan := DailyPlan{
		ID:         uuid.New(),
		PlanDate:   ne.PlanDate,
		Generation: ne.Generation,
		ModelUsed:  ne.ModelUsed,
		PromptHash: ne.PromptHash,
		CreatedAt:  now,
	}

	if err := b.storer.CreatePlan(ctx, plan); err != nil {
		return DailyPlan{}, fmt.Errorf("create plan: %w", err)
	}

	return plan, nil
}

func (b *Business) AddItem(ctx context.Context, ni NewDailyPlanItem) (DailyPlanItem, error) {
	now := time.Now()

	item := DailyPlanItem{
		ID:               uuid.New(),
		PlanID:           ni.PlanID,
		TaskID:           ni.TaskID,
		Position:         ni.Position,
		GroupName:        ni.GroupName,
		GroupPosition:    ni.GroupPosition,
		AIDurationMin:    ni.AIDurationMin,
		AIPriorityReason: ni.AIPriorityReason,
		Status:           "proposed",
		CreatedAt:        now,
	}

	if err := b.storer.CreateItem(ctx, item); err != nil {
		return DailyPlanItem{}, fmt.Errorf("create item: %w", err)
	}

	return item, nil
}

func (b *Business) GetByDate(ctx context.Context, date time.Time) (DailyPlan, []DailyPlanItem, error) {
	plan, err := b.storer.QueryPlanByDate(ctx, date)
	if err != nil {
		return DailyPlan{}, nil, fmt.Errorf("query plan by date: %w", err)
	}

	items, err := b.storer.QueryItemsByPlan(ctx, plan.ID)
	if err != nil {
		return DailyPlan{}, nil, fmt.Errorf("query items by plan: %w", err)
	}

	return plan, items, nil
}

func (b *Business) UpdateItem(ctx context.Context, item DailyPlanItem, update UpdatePlanItem) (DailyPlanItem, error) {
	if update.UserPosition != nil {
		item.UserPosition = update.UserPosition
	}
	if update.UserDurationMin != nil {
		item.UserDurationMin = update.UserDurationMin
	}
	if update.Status != nil {
		item.Status = *update.Status
	}
	if update.DismissReason != nil {
		item.DismissReason = update.DismissReason
	}
	if update.DismissNote != nil {
		item.DismissNote = update.DismissNote
	}
	if update.CompletedAt != nil {
		item.CompletedAt = update.CompletedAt
	}

	if err := b.storer.UpdateItem(ctx, item); err != nil {
		return DailyPlanItem{}, fmt.Errorf("update item: %w", err)
	}

	return item, nil
}

func (b *Business) QueryItemByID(ctx context.Context, itemID uuid.UUID) (DailyPlanItem, error) {
	item, err := b.storer.QueryItemByID(ctx, itemID)
	if err != nil {
		return DailyPlanItem{}, fmt.Errorf("query item by id[%s]: %w", itemID, err)
	}
	return item, nil
}

func (b *Business) DeleteItemsByPlan(ctx context.Context, planID uuid.UUID) error {
	if err := b.storer.DeleteItemsByPlan(ctx, planID); err != nil {
		return fmt.Errorf("delete items by plan: %w", err)
	}
	return nil
}
