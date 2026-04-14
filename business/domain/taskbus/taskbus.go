package taskbus

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/debriefstatus"
	"github.com/casebrophy/planner/business/types/recurrence"
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
	DismissTasksByContext(ctx context.Context, contextID uuid.UUID) (int, error)
	DeleteByRawInputUnconfirmed(ctx context.Context, rawInputID uuid.UUID) error
}

type Business struct {
	log       *logger.Logger
	storer    Storer
	depStorer DependencyStorer
}

func NewBusiness(log *logger.Logger, storer Storer, depStorer DependencyStorer) *Business {
	return &Business{
		log:       log,
		storer:    storer,
		depStorer: depStorer,
	}
}

func (b *Business) Create(ctx context.Context, nt NewTask) (Task, error) {
	now := time.Now()

	task := Task{
		ID:             uuid.New(),
		ContextID:      nt.ContextID,
		Title:          nt.Title,
		Description:    nt.Description,
		Status:         nt.Status,
		Priority:       nt.Priority,
		Energy:         nt.Energy,
		DurationMin:    nt.DurationMin,
		DueDate:        nt.DueDate,
		RecurrenceRule: nt.RecurrenceRule,
		TrackOutcome:   nt.TrackOutcome,
		Unconfirmed:    nt.Unconfirmed,
		DebriefStatus:  debriefstatus.Pending,
		CreatedAt:      now,
		UpdatedAt:      now,
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
	if ut.BlockedReason != nil {
		task.BlockedReason = *ut.BlockedReason
	}
	if ut.RecurrenceRule != nil {
		task.RecurrenceRule = ut.RecurrenceRule
	}
	if ut.TrackOutcome != nil {
		task.TrackOutcome = *ut.TrackOutcome
	}
	if ut.Unconfirmed != nil {
		task.Unconfirmed = *ut.Unconfirmed
	}

	task.UpdatedAt = time.Now()

	if err := b.storer.Update(ctx, task); err != nil {
		return Task{}, fmt.Errorf("update: %w", err)
	}

	if task.Status == taskstatus.Done {
		if err := b.UnblockDependents(ctx, task.ID); err != nil {
			b.log.Error(ctx, "unblock dependents", "error", err)
		}

		if task.RecurrenceRule != nil {
			if _, err := b.CreateNextRecurrence(ctx, task); err != nil {
				b.log.Error(ctx, "create next recurrence", "error", err)
			}
		}
	}

	return task, nil
}

// CreateNextRecurrence creates the next recurring task instance from a completed
// recurring task. It copies the task's core fields, computes the next due date
// from the recurrence rule, and links back to the original via RecurrenceParentID.
func (b *Business) CreateNextRecurrence(ctx context.Context, task Task) (Task, error) {
	if task.RecurrenceRule == nil {
		return Task{}, fmt.Errorf("task has no recurrence rule")
	}

	rule, err := recurrence.Parse(*task.RecurrenceRule)
	if err != nil {
		return Task{}, fmt.Errorf("parse recurrence rule: %w", err)
	}

	// Determine the base date for computing the next occurrence.
	var from time.Time
	switch {
	case task.DueDate != nil:
		from = *task.DueDate
	case task.CompletedAt != nil:
		from = *task.CompletedAt
	default:
		from = time.Now()
	}

	nextDue := rule.NextOccurrence(from)

	now := time.Now()
	next := Task{
		ID:             uuid.New(),
		ContextID:      task.ContextID,
		Title:          task.Title,
		Description:    task.Description,
		Status:         taskstatus.Open,
		Priority:       task.Priority,
		Energy:         task.Energy,
		DurationMin:    task.DurationMin,
		DueDate:        &nextDue,
		RecurrenceRule: task.RecurrenceRule,
		RecurrenceParentID: &task.ID,
		DebriefStatus:  debriefstatus.Pending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := b.storer.Create(ctx, next); err != nil {
		return Task{}, fmt.Errorf("create next recurrence: %w", err)
	}

	return next, nil
}

func (b *Business) Delete(ctx context.Context, task Task) error {
	if err := b.storer.Delete(ctx, task); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

func (b *Business) DeleteByRawInputUnconfirmed(ctx context.Context, rawInputID uuid.UUID) error {
	if err := b.storer.DeleteByRawInputUnconfirmed(ctx, rawInputID); err != nil {
		return fmt.Errorf("deletebyrawinputunconfirmed: %w", err)
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

// DismissTasksByContext sets all open/blocked tasks for a context to dismissed.
func (b *Business) DismissTasksByContext(ctx context.Context, contextID uuid.UUID) (int, error) {
	return b.storer.DismissTasksByContext(ctx, contextID)
}
