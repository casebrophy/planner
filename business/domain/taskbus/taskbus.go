package taskbus

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/debriefstatus"
	"github.com/casebrophy/planner/business/types/taskstatus"
	"github.com/casebrophy/planner/foundation/logger"
)

type Storer interface {
	Create(ctx context.Context, task Task) error
	Update(ctx context.Context, task Task) error
	Delete(ctx context.Context, task Task) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Task, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (Task, error)
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

func (b *Business) Create(ctx context.Context, nt NewTask) (Task, error) {
	now := time.Now()

	task := Task{
		ID:            uuid.New(),
		ContextID:     nt.ContextID,
		Title:         nt.Title,
		Description:   nt.Description,
		Status:        nt.Status,
		Priority:      nt.Priority,
		Energy:        nt.Energy,
		DurationMin:   nt.DurationMin,
		DueDate:       nt.DueDate,
		DebriefStatus: debriefstatus.Pending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := b.storer.Create(ctx, task); err != nil {
		return Task{}, fmt.Errorf("create: %w", err)
	}

	return task, nil
}

func (b *Business) Update(ctx context.Context, task Task, ut UpdateTask) (Task, error) {
	if ut.Title != nil {
		task.Title = *ut.Title
	}
	if ut.Description != nil {
		task.Description = *ut.Description
	}
	if ut.ContextID != nil {
		task.ContextID = ut.ContextID
	}
	if ut.Status != nil {
		task.Status = *ut.Status
		if *ut.Status == taskstatus.Done && task.CompletedAt == nil {
			now := time.Now()
			task.CompletedAt = &now
		}
	}
	if ut.Priority != nil {
		task.Priority = *ut.Priority
	}
	if ut.Energy != nil {
		task.Energy = *ut.Energy
	}
	if ut.DurationMin != nil {
		task.DurationMin = ut.DurationMin
	}
	if ut.DueDate != nil {
		task.DueDate = ut.DueDate
	}
	if ut.ScheduledAt != nil {
		task.ScheduledAt = ut.ScheduledAt
	}
	if ut.ExpectedUpdateDays != nil {
		task.ExpectedUpdateDays = ut.ExpectedUpdateDays
	}
	if ut.DebriefStatus != nil {
		task.DebriefStatus = *ut.DebriefStatus
	}

	task.UpdatedAt = time.Now()

	if err := b.storer.Update(ctx, task); err != nil {
		return Task{}, fmt.Errorf("update: %w", err)
	}

	return task, nil
}

func (b *Business) Delete(ctx context.Context, task Task) error {
	if err := b.storer.Delete(ctx, task); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

func (b *Business) Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Task, error) {
	tasks, err := b.storer.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return tasks, nil
}

func (b *Business) Count(ctx context.Context, filter QueryFilter) (int, error) {
	n, err := b.storer.Count(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	return n, nil
}

func (b *Business) QueryByID(ctx context.Context, id uuid.UUID) (Task, error) {
	task, err := b.storer.QueryByID(ctx, id)
	if err != nil {
		return Task{}, fmt.Errorf("query by id[%s]: %w", id, err)
	}
	return task, nil
}
