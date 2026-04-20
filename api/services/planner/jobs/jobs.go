// Package jobs defines the background jobs registered and run by the
// planner service. Each Job owns its own scheduling loop and honors
// ctx.Done() for graceful shutdown.
package jobs

import (
	"context"
	"sync"

	"github.com/casebrophy/planner/foundation/logger"
)

// Job is the contract every background job implements.
type Job interface {
	Name() string
	Run(ctx context.Context)
}

// RunAll starts every job in its own goroutine and blocks until each
// job's Run returns (typically when ctx is cancelled). Each job is
// responsible for honoring ctx.Done() inside its own loop.
func RunAll(ctx context.Context, log *logger.Logger, jobs ...Job) {
	var wg sync.WaitGroup
	for _, j := range jobs {
		wg.Add(1)
		go func(j Job) {
			defer wg.Done()
			log.Info(ctx, "jobs", "msg", "starting", "job", j.Name())
			j.Run(ctx)
		}(j)
	}
	wg.Wait()
}
