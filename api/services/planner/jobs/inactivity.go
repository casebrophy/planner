package jobs

import (
	"context"
	"time"

	"github.com/casebrophy/planner/foundation/logger"
)

// InactivityChecker is the subset of inactivitybus.Business the job needs.
type InactivityChecker interface {
	CheckAll(ctx context.Context) error
}

// InactivityJob periodically sweeps for inactive contexts/tasks.
// Interval defaults to 15 minutes when zero.
type InactivityJob struct {
	Log      *logger.Logger
	Checker  InactivityChecker
	Interval time.Duration
}

// Name satisfies Job.
func (InactivityJob) Name() string { return "inactivity" }

// Run blocks until ctx is cancelled, ticking the inactivity sweep on
// each interval.
func (j InactivityJob) Run(ctx context.Context) {
	interval := j.Interval
	if interval == 0 {
		interval = 15 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := j.Checker.CheckAll(ctx); err != nil {
				j.Log.Error(ctx, "inactivity", "msg", "inactivity check failed", "error", err)
			}
		}
	}
}
