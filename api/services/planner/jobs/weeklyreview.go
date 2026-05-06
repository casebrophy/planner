package jobs

import (
	"context"
	"time"

	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/foundation/logger"
)

// TaskQuerier is the subset of taskbus.Business the jobs in this package
// need for fetching tasks.
type TaskQuerier interface {
	Query(ctx context.Context, filter taskbus.QueryFilter, orderBy order.By, pg page.Page) ([]taskbus.Task, error)
}

// WeeklyReviewJob is disabled and kept for backwards compatibility only.
// The observation/debrief system has been removed.
type WeeklyReviewJob struct {
	Log      *logger.Logger
	TaskBus  TaskQuerier
	Now      func() time.Time
	Interval time.Duration
}

// Name satisfies Job.
func (WeeklyReviewJob) Name() string { return "weekly-review" }

// Run blocks until ctx is cancelled.
// DISABLED: Weekly review job is disabled (observation/debrief system removed)
func (j WeeklyReviewJob) Run(ctx context.Context) {
	// No-op: wait for context cancellation
	<-ctx.Done()
}
