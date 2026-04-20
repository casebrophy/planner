package jobs

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/dailyplanbus"
	"github.com/casebrophy/planner/business/domain/dailyplanbus/generator"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/foundation/logger"
)

// ContextQuerier is the subset of contextbus.Business the daily-plan job
// needs for fetching active contexts.
type ContextQuerier interface {
	Query(ctx context.Context, filter contextbus.QueryFilter, orderBy order.By, pg page.Page) ([]contextbus.Context, error)
}

// EventQuerier is the subset of eventbus.Business the daily-plan job
// needs for fetching today's events.
type EventQuerier interface {
	Query(ctx context.Context, filter eventbus.QueryFilter, orderBy order.By, pg page.Page) ([]eventbus.Event, error)
}

// PlanCreator is the subset of dailyplanbus.Business the daily-plan job
// needs for persisting a generated plan and its items.
type PlanCreator interface {
	Create(ctx context.Context, ne dailyplanbus.NewDailyPlan) (dailyplanbus.DailyPlan, error)
	AddItem(ctx context.Context, ni dailyplanbus.NewDailyPlanItem) (dailyplanbus.DailyPlanItem, error)
}

// PlanGenerator is the subset of *generator.Generator the daily-plan job
// needs. Signature mirrors generator.Generator.Generate exactly.
type PlanGenerator interface {
	Generate(ctx context.Context, tasks []generator.TaskRef, events []generator.EventRef, carryover []generator.CarryoverItem, tzName string) (generator.PlanOutput, []generator.ImplicationResult, string, error)
}

// DailyPlanJob runs once per day at the configured fire time (HH:MM) in
// the user's timezone, generating a morning plan over the set of open /
// blocked tasks. Interval defaults to 1 minute when zero.
type DailyPlanJob struct {
	Log       *logger.Logger
	Generator PlanGenerator
	TaskBus   TaskQuerier
	CtxBus    ContextQuerier
	EvtBus    EventQuerier
	DpBus     PlanCreator
	UserTZ    *time.Location
	FireAt    string
	Now       func() time.Time
	Interval  time.Duration
}

// Name satisfies Job.
func (DailyPlanJob) Name() string { return "daily-plan" }

// now returns the current time, using j.Now if set so tests can drive
// a deterministic clock.
func (j DailyPlanJob) now() time.Time {
	if j.Now != nil {
		return j.Now()
	}
	return time.Now()
}

// Run blocks until ctx is cancelled.
func (j DailyPlanJob) Run(ctx context.Context) {
	interval := j.Interval
	if interval == 0 {
		interval = 1 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastGenDate string

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := j.now()
			todayStr := now.Format("2006-01-02")
			timeStr := now.Format("15:04")

			if timeStr != j.FireAt || lastGenDate == todayStr {
				continue
			}
			lastGenDate = todayStr
			j.Log.Info(ctx, "daily-plan", "msg", "generating morning plan", "date", todayStr)

			// Fetch open tasks
			allTasks, err := j.TaskBus.Query(ctx, taskbus.QueryFilter{}, taskbus.DefaultOrderBy, page.New(1, 200))
			if err != nil {
				j.Log.Error(ctx, "daily-plan", "msg", "failed to fetch tasks", "error", err)
				continue
			}

			// Build context title lookup
			activeStatus := contextbus.Active
			contexts, _ := j.CtxBus.Query(ctx, contextbus.QueryFilter{Status: &activeStatus}, contextbus.DefaultOrderBy, page.New(1, 100))
			ctxNames := make(map[uuid.UUID]string, len(contexts))
			for _, c := range contexts {
				ctxNames[c.ID] = c.Title
			}

			var taskRefs []generator.TaskRef
			for _, t := range allTasks {
				if t.Status.String() != "open" && t.Status.String() != "blocked" {
					continue
				}
				ref := generator.TaskRef{
					ID:       t.ID.String(),
					Title:    t.Title,
					Priority: t.Priority.String(),
					Energy:   t.Energy.String(),
					Status:   t.Status.String(),
				}
				if t.DurationMin != nil {
					ref.DurationMin = t.DurationMin
				}
				if t.DueDate != nil {
					s := t.DueDate.Format(time.RFC3339)
					ref.DueDate = &s
				}
				if t.ContextID != nil {
					if name, ok := ctxNames[*t.ContextID]; ok {
						ref.Context = &name
					}
				}
				taskRefs = append(taskRefs, ref)
			}

			// Fetch today's events
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, j.UserTZ)
			tomorrow := today.Add(24 * time.Hour)
			events, err := j.EvtBus.Query(ctx, eventbus.QueryFilter{DateFrom: &today, DateTo: &tomorrow}, eventbus.DefaultOrderBy, page.New(1, 50))
			if err != nil {
				j.Log.Error(ctx, "daily-plan", "msg", "failed to fetch events", "error", err)
				continue
			}

			var eventRefs []generator.EventRef
			for _, e := range events {
				ref := generator.EventRef{
					ID:       e.ID.String(),
					Title:    e.Title,
					StartsAt: e.StartsAt.In(j.UserTZ).Format(time.RFC3339),
					EndsAt:   e.EndsAt.In(j.UserTZ).Format(time.RFC3339),
					AllDay:   e.AllDay,
				}
				if e.Location != nil {
					ref.Location = e.Location
				}
				eventRefs = append(eventRefs, ref)
			}

			// Generate plan
			output, _, modelUsed, err := j.Generator.Generate(ctx, taskRefs, eventRefs, nil, j.UserTZ.String())
			if err != nil {
				j.Log.Error(ctx, "daily-plan", "msg", "plan generation failed", "error", err)
				continue
			}

			// Store plan
			plan, err := j.DpBus.Create(ctx, dailyplanbus.NewDailyPlan{
				PlanDate:   today,
				Generation: 1,
				ModelUsed:  modelUsed,
			})
			if err != nil {
				j.Log.Error(ctx, "daily-plan", "msg", "failed to create plan", "error", err)
				continue
			}

			itemCount := 0
			for gi, group := range output.Groups {
				for ii, item := range group.Items {
					taskID, err := uuid.Parse(item.TaskID)
					if err != nil {
						continue
					}
					reason := item.PriorityReason
					dur := item.AIDurationMin
					if _, err := j.DpBus.AddItem(ctx, dailyplanbus.NewDailyPlanItem{
						PlanID:           plan.ID,
						TaskID:           taskID,
						Position:         ii,
						GroupName:        group.Name,
						GroupPosition:    gi,
						AIDurationMin:    &dur,
						AIPriorityReason: &reason,
					}); err != nil {
						j.Log.Error(ctx, "daily-plan", "msg", "failed to add item", "error", err)
						continue
					}
					itemCount++
				}
			}

			j.Log.Info(ctx, "daily-plan", "msg", "morning plan generated", "items", itemCount, "groups", len(output.Groups))
		}
	}
}
