package jobs

import (
	"context"
	"time"

	"github.com/casebrophy/planner/foundation/logger"
)

// StuckRecoverer is the subset of rawinputbus.Business the job needs.
type StuckRecoverer interface {
	RecoverStuck(ctx context.Context, threshold time.Duration) (int, error)
}

// RawInputRecoveryJob periodically sweeps raw_input rows stuck in the
// "processing" state longer than Threshold and returns them to the
// retry queue. Interval defaults to 10 minutes, Threshold to 15 minutes
// when zero.
type RawInputRecoveryJob struct {
	Log       *logger.Logger
	Bus       StuckRecoverer
	Interval  time.Duration
	Threshold time.Duration
}

// Name satisfies Job.
func (RawInputRecoveryJob) Name() string { return "rawinput-recovery" }

// Run blocks until ctx is cancelled.
func (j RawInputRecoveryJob) Run(ctx context.Context) {
	interval := j.Interval
	if interval == 0 {
		interval = 10 * time.Minute
	}
	threshold := j.Threshold
	if threshold == 0 {
		threshold = 15 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := j.Bus.RecoverStuck(ctx, threshold)
			if err != nil {
				j.Log.Error(ctx, "rawinput-recovery", "msg", "recovery sweep failed", "error", err)
			} else if n > 0 {
				j.Log.Info(ctx, "rawinput-recovery", "msg", "recovered stuck raw_inputs", "count", n)
			}
		}
	}
}
