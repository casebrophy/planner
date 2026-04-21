package jobs

import (
	"context"
	"time"

	"github.com/casebrophy/planner/foundation/logger"
)

// Unsnoozer is the subset of clarificationbus.Business the job needs.
type Unsnoozer interface {
	UnsnoozeExpired(ctx context.Context) (int, error)
}

// UnsnoozeJob periodically unsnoozes clarifications whose snooze has
// expired. Interval defaults to 5 minutes when zero.
type UnsnoozeJob struct {
	Log      *logger.Logger
	Bus      Unsnoozer
	Interval time.Duration
}

// Name satisfies Job.
func (UnsnoozeJob) Name() string { return "unsnooze" }

// Run blocks until ctx is cancelled, unsnoozing expired clarifications
// on each interval.
func (j UnsnoozeJob) Run(ctx context.Context) {
	interval := j.Interval
	if interval == 0 {
		interval = 5 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := j.Bus.UnsnoozeExpired(ctx)
			if err != nil {
				j.Log.Error(ctx, "unsnooze", "msg", "unsnooze expired failed", "error", err)
			} else if n > 0 {
				j.Log.Info(ctx, "unsnooze", "msg", "unsnoozed expired items", "count", n)
			}
		}
	}
}
