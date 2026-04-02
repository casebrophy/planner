package dailyplanapp

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/dailyplanbus"
	"github.com/casebrophy/planner/business/domain/dailyplanbus/generator"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/business/types/taskstatus"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	dailyPlanBus *dailyplanbus.Business
	taskBus      *taskbus.Business
	eventBus     *eventbus.Business
	contextBus   *contextbus.Business
	generator    *generator.Generator
}

func (a *app) getPlan(ctx context.Context, r *http.Request) web.Encoder {
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	plan, items, err := a.dailyPlanBus.GetByDate(ctx, date)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			// Return empty plan instead of 404
			return DailyPlan{
				PlanDate:   dateStr,
				Generation: 0,
				Items:      []DailyPlanItem{},
			}
		}
		return errs.Newf(errs.Internal, "query plan: %s", err)
	}

	return toAppPlan(plan, items)
}

func (a *app) generate(ctx context.Context, r *http.Request) web.Encoder {
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	// Fetch open tasks (query all, filter in Go for todo and in_progress)
	pg, _ := page.Parse("1", "1000")
	allTasks, err := a.taskBus.Query(ctx, taskbus.QueryFilter{}, taskbus.DefaultOrderBy, pg)
	if err != nil {
		return errs.Newf(errs.Internal, "query tasks: %s", err)
	}

	// Filter for open and blocked tasks
	var tasks []taskbus.Task
	for _, t := range allTasks {
		if t.Status == taskstatus.Open || t.Status == taskstatus.Blocked {
			tasks = append(tasks, t)
		}
	}

	// Convert tasks to generator references
	taskRefs := make([]generator.TaskRef, len(tasks))
	for i, t := range tasks {
		taskRefs[i] = generator.TaskRef{
			ID:          t.ID.String(),
			Title:       t.Title,
			Priority:    t.Priority.String(),
			Energy:      t.Energy.String(),
			DurationMin: t.DurationMin,
			Status:      t.Status.String(),
		}
		if t.DueDate != nil {
			dueDateStr := t.DueDate.Format("2006-01-02")
			taskRefs[i].DueDate = &dueDateStr
		}
		if t.ContextID != nil {
			// Get context name
			ctx_, err := a.contextBus.QueryByID(ctx, *t.ContextID)
			if err == nil {
				taskRefs[i].Context = &ctx_.Title
			}
		}
	}

	// Fetch today's events
	todayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)
	todayEnd := todayStart.AddDate(0, 0, 1)
	eventFilter := eventbus.QueryFilter{
		DateFrom: &todayStart,
		DateTo:   &todayEnd,
	}
	pg2, _ := page.Parse("1", "100")
	events, err := a.eventBus.Query(ctx, eventFilter, eventbus.DefaultOrderBy, pg2)
	if err != nil {
		return errs.Newf(errs.Internal, "query events: %s", err)
	}

	// Convert events to generator references
	eventRefs := make([]generator.EventRef, len(events))
	for i, e := range events {
		eventRefs[i] = generator.EventRef{
			ID:       e.ID.String(),
			Title:    e.Title,
			StartsAt: e.StartsAt.Format(time.RFC3339),
			EndsAt:   e.EndsAt.Format(time.RFC3339),
			AllDay:   e.AllDay,
		}
		if e.Location != nil {
			eventRefs[i].Location = e.Location
		}
	}

	// TODO: Check for yesterday's plan and carryover items
	// For now, pass empty carryover
	var carryover []generator.CarryoverItem

	// Generate the plan
	planOutput, err := a.generator.Generate(ctx, taskRefs, eventRefs, carryover)
	if err != nil {
		return errs.Newf(errs.Internal, "generate plan: %s", err)
	}

	// Check if plan already exists for this date
	existingPlan, _, err := a.dailyPlanBus.GetByDate(ctx, date)
	if err != nil && !errors.Is(err, sqldb.ErrDBNotFound) {
		return errs.Newf(errs.Internal, "query existing plan: %s", err)
	}

	var newPlan dailyplanbus.DailyPlan
	if errors.Is(err, sqldb.ErrDBNotFound) {
		// Create new plan at generation 1
		newPlan, err = a.dailyPlanBus.Create(ctx, dailyplanbus.NewDailyPlan{
			PlanDate:   date,
			Generation: 1,
			ModelUsed:  "haiku", // TODO: extract from generator
		})
		if err != nil {
			return errs.Newf(errs.Internal, "create plan: %s", err)
		}
	} else {
		// Increment generation and delete old items
		newPlanObj := dailyplanbus.NewDailyPlan{
			PlanDate:   date,
			Generation: existingPlan.Generation + 1,
			ModelUsed:  "haiku",
		}
		newPlan, err = a.dailyPlanBus.Create(ctx, newPlanObj)
		if err != nil {
			return errs.Newf(errs.Internal, "create plan: %s", err)
		}

		if err := a.dailyPlanBus.DeleteItemsByPlan(ctx, existingPlan.ID); err != nil {
			return errs.Newf(errs.Internal, "delete old items: %s", err)
		}
	}

	// Add items from plan output
	var allItems []dailyplanbus.DailyPlanItem
	itemPosition := 0
	for _, group := range planOutput.Groups {
		for itemIdx, item := range group.Items {
			taskID, err := uuid.Parse(item.TaskID)
			if err != nil {
				return errs.Newf(errs.InvalidArgument, "invalid task_id: %s", err)
			}

			newItem, err := a.dailyPlanBus.AddItem(ctx, dailyplanbus.NewDailyPlanItem{
				PlanID:           newPlan.ID,
				TaskID:           taskID,
				Position:         itemPosition,
				GroupName:        group.Name,
				GroupPosition:    itemIdx,
				AIDurationMin:    &item.AIDurationMin,
				AIPriorityReason: &item.PriorityReason,
			})
			if err != nil {
				return errs.Newf(errs.Internal, "add item: %s", err)
			}

			allItems = append(allItems, newItem)
			itemPosition++
		}
	}

	return toAppPlan(newPlan, allItems)
}

func (a *app) updateItem(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "item_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	item, err := a.dailyPlanBus.QueryItemByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query item: %s", err)
	}

	var input UpdateItemRequest
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	update := dailyplanbus.UpdatePlanItem{
		UserPosition:    input.UserPosition,
		UserDurationMin: input.UserDurationMin,
	}

	updated, err := a.dailyPlanBus.UpdateItem(ctx, item, update)
	if err != nil {
		return errs.Newf(errs.Internal, "update item: %s", err)
	}

	return toAppItem(updated)
}

func (a *app) completeItem(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "item_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	item, err := a.dailyPlanBus.QueryItemByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query item: %s", err)
	}

	now := time.Now()
	status := "completed"
	update := dailyplanbus.UpdatePlanItem{
		Status:      &status,
		CompletedAt: &now,
	}

	updated, err := a.dailyPlanBus.UpdateItem(ctx, item, update)
	if err != nil {
		return errs.Newf(errs.Internal, "update item: %s", err)
	}

	return toAppItem(updated)
}

func (a *app) dismissItem(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "item_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	item, err := a.dailyPlanBus.QueryItemByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "query item: %s", err)
	}

	var input DismissRequest
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	status := "dismissed"
	update := dailyplanbus.UpdatePlanItem{
		Status:        &status,
		DismissReason: &input.Reason,
		DismissNote:   input.Note,
	}

	updated, err := a.dailyPlanBus.UpdateItem(ctx, item, update)
	if err != nil {
		return errs.Newf(errs.Internal, "update item: %s", err)
	}

	return toAppItem(updated)
}
