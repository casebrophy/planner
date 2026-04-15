package dailyplanapp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/dailyplanbus"
	"github.com/casebrophy/planner/business/domain/dailyplanbus/generator"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/taskstatus"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	log              *logger.Logger
	dailyPlanBus     *dailyplanbus.Business
	taskBus          *taskbus.Business
	eventBus         *eventbus.Business
	contextBus       *contextbus.Business
	clarificationBus *clarificationbus.Business
	generator        *generator.Generator
	userTZ           *time.Location
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
	pg := page.New(1, 100)
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

	// Resolve user timezone for event filtering and formatting.
	tz := a.userTZ
	if tz == nil {
		tz = time.UTC
	}

	// Fetch today's events
	todayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, tz)
	todayEnd := todayStart.AddDate(0, 0, 1)
	eventFilter := eventbus.QueryFilter{
		DateFrom: &todayStart,
		DateTo:   &todayEnd,
	}
	pg2, err := page.Parse("1", "100")
	if err != nil {
		return errs.Newf(errs.InvalidArgument, "parse page: %s", err)
	}
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
			StartsAt: e.StartsAt.In(tz).Format(time.RFC3339),
			EndsAt:   e.EndsAt.In(tz).Format(time.RFC3339),
			AllDay:   e.AllDay,
		}
		if e.Location != nil {
			eventRefs[i].Location = e.Location
		}
	}

	// Check yesterday's plan for incomplete items to carry over
	var carryover []generator.CarryoverItem
	yesterday := date.AddDate(0, 0, -1)
	_, yesterdayItems, yesterdayErr := a.dailyPlanBus.GetByDate(ctx, yesterday)
	if yesterdayErr == nil {
		// Build task ID → title map from already-loaded tasks
		taskTitles := make(map[string]string, len(allTasks))
		for _, t := range allTasks {
			taskTitles[t.ID.String()] = t.Title
		}

		for _, item := range yesterdayItems {
			if item.Status == "proposed" || item.Status == "accepted" {
				taskID := item.TaskID.String()
				title := taskTitles[taskID]
				if title == "" {
					title = "unknown task"
				}
				carryover = append(carryover, generator.CarryoverItem{
					TaskID: taskID,
					Title:  title,
					Reason: "planned yesterday but not completed",
				})
			}
		}
	}

	// Capture values for the goroutine
	capturedTaskRefs := taskRefs
	capturedEventRefs := eventRefs
	capturedCarryover := carryover
	capturedDate := date
	capturedTZName := tz.String()

	// Spawn goroutine for LLM generation and DB writes — return immediately
	go func() {
		bgCtx := context.Background()

		// Check if plan already exists for this date
		existingPlan, _, planErr := a.dailyPlanBus.GetByDate(bgCtx, capturedDate)
		if planErr != nil && !errors.Is(planErr, sqldb.ErrDBNotFound) {
			a.log.Error(bgCtx, "dailyplan.generate", "msg", "get existing plan failed", "error", planErr)
			return
		}

		planOutput, implications, modelUsed, err := a.generator.Generate(bgCtx, capturedTaskRefs, capturedEventRefs, capturedCarryover, capturedTZName)
		if err != nil {
			a.log.Error(bgCtx, "dailyplan.generate", "msg", "generator failed", "error", err)
			return
		}

		var newPlan dailyplanbus.DailyPlan
		if errors.Is(planErr, sqldb.ErrDBNotFound) {
			// Create new plan at generation 1
			newPlan, err = a.dailyPlanBus.Create(bgCtx, dailyplanbus.NewDailyPlan{
				PlanDate:   capturedDate,
				Generation: 1,
				ModelUsed:  modelUsed,
			})
			if err != nil {
				a.log.Error(bgCtx, "dailyplan.generate", "msg", "create plan failed", "error", err)
				return
			}
		} else {
			newPlanObj := dailyplanbus.NewDailyPlan{
				PlanDate:   capturedDate,
				Generation: existingPlan.Generation + 1,
				ModelUsed:  modelUsed,
			}
			newPlan, err = a.dailyPlanBus.Create(bgCtx, newPlanObj)
			if err != nil {
				a.log.Error(bgCtx, "dailyplan.generate", "msg", "create regenerated plan failed", "error", err)
				return
			}

			if err := a.dailyPlanBus.DeleteItemsByPlan(bgCtx, existingPlan.ID); err != nil {
				a.log.Error(bgCtx, "dailyplan.generate", "msg", "delete old items failed", "error", err)
				return
			}
		}

		// Add items from plan output
		plannedTaskIDs := make(map[string]bool)
		itemPosition := 0
		for _, group := range planOutput.Groups {
			for itemIdx, item := range group.Items {
				taskID, err := uuid.Parse(item.TaskID)
				if err != nil {
					continue
				}

				a.dailyPlanBus.AddItem(bgCtx, dailyplanbus.NewDailyPlanItem{ //nolint:errcheck
					PlanID:           newPlan.ID,
					TaskID:           taskID,
					Position:         itemPosition,
					GroupName:        group.Name,
					GroupPosition:    itemIdx,
					AIDurationMin:    &item.AIDurationMin,
					AIPriorityReason: &item.PriorityReason,
				})
				plannedTaskIDs[item.TaskID] = true
				itemPosition++
			}
		}

		// Create EventPrep clarifications for events whose prep tasks are not in the plan.
		a.createEventPrepClarifications(bgCtx, implications, plannedTaskIDs)
	}()

	return GenerateAccepted{Status: "generating"}
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

// createEventPrepClarifications surfaces EventPrep clarifications for events whose
// implied prep tasks are not present in the current plan. One clarification is created
// per event (grouped across all unscheduled prep tasks for that event).
func (a *app) createEventPrepClarifications(ctx context.Context, implications []generator.ImplicationResult, plannedTaskIDs map[string]bool) {
	if a.clarificationBus == nil {
		return
	}

	// Group unscheduled prep tasks by event.
	type eventGroup struct {
		eventID       string
		eventTitle    string
		eventStartsAt time.Time
		taskIDs       []string
		taskTitles    []string
	}
	order := []string{}
	byEvent := map[string]*eventGroup{}

	for _, imp := range implications {
		if plannedTaskIDs[imp.TaskID] {
			// Task already in the plan — Claude has it covered.
			continue
		}
		if _, ok := byEvent[imp.EventID]; !ok {
			order = append(order, imp.EventID)
			byEvent[imp.EventID] = &eventGroup{
				eventID:       imp.EventID,
				eventTitle:    imp.EventTitle,
				eventStartsAt: imp.EventStartsAt,
			}
		}
		g := byEvent[imp.EventID]
		g.taskIDs = append(g.taskIDs, imp.TaskID)
		g.taskTitles = append(g.taskTitles, imp.TaskTitle)
	}

	for _, evID := range order {
		g := byEvent[evID]

		opts := clarificationbus.EventPrepOptions{
			EventID:        g.eventID,
			EventTitle:     g.eventTitle,
			EventStartsAt:  g.eventStartsAt.Format(time.RFC3339),
			PrepTaskIDs:    g.taskIDs,
			PrepTaskTitles: g.taskTitles,
		}
		optsJSON, err := json.Marshal(opts)
		if err != nil {
			a.log.Error(ctx, "dailyplan.eventprep", "msg", "marshal options failed", "error", err)
			continue
		}
		rawOpts := json.RawMessage(optsJSON)

		// Use a synthetic subject UUID derived from the event ID so dedup is stable.
		subjectID, err := uuid.Parse(evID)
		if err != nil {
			continue
		}

		question := "These tasks may need to happen before \"" + g.eventTitle + "\" (" +
			g.eventStartsAt.Format("3:04 PM") + "): " + formatTaskList(g.taskTitles) +
			". Should they be scheduled before this event?"

		if _, err := a.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
			Kind:               clarificationkind.EventPrep,
			SubjectType:        "event",
			SubjectID:          subjectID,
			SubjectDescription: g.eventTitle,
			Question:           question,
			AnswerOptions:      rawOpts,
			PriorityScore:      0.7,
		}); err != nil {
			a.log.Error(ctx, "dailyplan.eventprep", "msg", "create clarification failed", "error", err)
		}
	}
}

// formatTaskList joins task titles into a readable list.
func formatTaskList(titles []string) string {
	switch len(titles) {
	case 0:
		return ""
	case 1:
		return "\"" + titles[0] + "\""
	default:
		result := ""
		for i, t := range titles {
			if i > 0 {
				result += ", "
			}
			result += "\"" + t + "\""
		}
		return result
	}
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
