package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/casebrophy/planner/business/domain/debriefbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/taskstatus"
	"github.com/casebrophy/planner/foundation/logger"
)

// TaskQuerier is the subset of taskbus.Business the jobs in this package
// need for fetching tasks.
type TaskQuerier interface {
	Query(ctx context.Context, filter taskbus.QueryFilter, orderBy order.By, pg page.Page) ([]taskbus.Task, error)
}

// WeeklyReviewer is the subset of debriefbus.Business the weekly review
// job needs.
type WeeklyReviewer interface {
	GenerateWeeklyReview(ctx context.Context, weekID string, tasks []debriefbus.CompletedTask) error
}

// WeeklyReviewJob runs once per week on Sunday at or after 18:00 local
// time and generates an impact review card for the tasks completed in
// the past seven days. Interval defaults to 1 minute when zero.
type WeeklyReviewJob struct {
	Log        *logger.Logger
	TaskBus    TaskQuerier
	DebriefBus WeeklyReviewer
	Now        func() time.Time
	Interval   time.Duration
}

// Name satisfies Job.
func (WeeklyReviewJob) Name() string { return "weekly-review" }

// now returns the current time, using j.Now if set so tests can drive
// a deterministic clock.
func (j WeeklyReviewJob) now() time.Time {
	if j.Now != nil {
		return j.Now()
	}
	return time.Now()
}

// Run blocks until ctx is cancelled.
func (j WeeklyReviewJob) Run(ctx context.Context) {
	interval := j.Interval
	if interval == 0 {
		interval = 1 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastGenWeek string

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := j.now()
			year, week := now.ISOWeek()
			weekID := fmt.Sprintf("%d-W%02d", year, week)

			// Fire on Sunday at or after 18:00 (lastGenWeek guard prevents re-runs)
			if now.Weekday() != time.Sunday || now.Hour() < 18 || lastGenWeek == weekID {
				continue
			}
			lastGenWeek = weekID
			j.Log.Info(ctx, "weekly-review", "msg", "generating weekly impact review", "week", weekID)

			// Query tasks completed in the past 7 days
			done := taskstatus.Done
			completedTasks, err := j.TaskBus.Query(ctx, taskbus.QueryFilter{Status: &done}, taskbus.DefaultOrderBy, page.New(1, 200))
			if err != nil {
				j.Log.Error(ctx, "weekly-review", "msg", "failed to fetch completed tasks", "error", err)
				continue
			}

			// Filter to tasks completed in the last 7 days
			cutoff := now.AddDate(0, 0, -7)
			var recentTasks []debriefbus.CompletedTask
			for _, t := range completedTasks {
				if t.CompletedAt != nil && t.CompletedAt.After(cutoff) {
					ct := debriefbus.CompletedTask{
						ID:    t.ID,
						Title: t.Title,
					}
					recentTasks = append(recentTasks, ct)
				}
			}

			if err := j.DebriefBus.GenerateWeeklyReview(ctx, weekID, recentTasks); err != nil {
				j.Log.Error(ctx, "weekly-review", "msg", "failed to generate weekly review", "error", err)
			}
		}
	}
}
