package dailyplanapp

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/business/domain/dailyplanbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/business/types/taskstatus"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	log          *logger.Logger
	dailyPlanBus *dailyplanbus.Business
	taskBus      *taskbus.Business
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

	// Mark the underlying task as done.
	task, err := a.taskBus.QueryByID(ctx, item.TaskID)
	if err != nil && !errors.Is(err, sqldb.ErrDBNotFound) {
		return errs.Newf(errs.Internal, "query task: %s", err)
	}
	if err == nil && task.Status != taskstatus.Done {
		doneStatus := taskstatus.Done
		if _, err := a.taskBus.Update(ctx, task, taskbus.UpdateTask{Status: &doneStatus}); err != nil {
			return errs.Newf(errs.Internal, "update task status: %s", err)
		}
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
