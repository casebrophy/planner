package jobs_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/casebrophy/planner/api/services/planner/jobs"
)

// countingJob satisfies jobs.Job. It increments runs when Run is invoked
// and blocks on ctx.Done() before returning.
type countingJob struct {
	name string
	runs *int32
	// returnAfter, if set, causes Run to return after this delay without
	// waiting for ctx.Done(). Used to prove RunAll still waits on the
	// slower jobs.
	returnAfter time.Duration
}

func (j countingJob) Name() string { return j.name }

func (j countingJob) Run(ctx context.Context) {
	atomic.AddInt32(j.runs, 1)
	if j.returnAfter > 0 {
		select {
		case <-ctx.Done():
		case <-time.After(j.returnAfter):
		}
		return
	}
	<-ctx.Done()
}

func TestRunAll_InvokesEveryJob(t *testing.T) {
	var runs int32

	j1 := countingJob{name: "a", runs: &runs}
	j2 := countingJob{name: "b", runs: &runs}
	j3 := countingJob{name: "c", runs: &runs}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		jobs.RunAll(ctx, newTestLogger(), j1, j2, j3)
		close(done)
	}()

	// Give the goroutines a moment to start and increment runs.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&runs) >= 3 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}

	if got := atomic.LoadInt32(&runs); got != 3 {
		cancel()
		<-done
		t.Fatalf("expected 3 job Run invocations, got %d", got)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("RunAll did not return after ctx cancel")
	}
}

func TestRunAll_ReturnsOnContextCancel(t *testing.T) {
	var runs int32
	j := countingJob{name: "solo", runs: &runs}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		jobs.RunAll(ctx, newTestLogger(), j)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("RunAll did not return after ctx cancel")
	}
}

func TestRunAll_WaitsForRemainingJobs(t *testing.T) {
	var runs int32

	// One job returns almost immediately...
	fast := countingJob{name: "fast", runs: &runs, returnAfter: 5 * time.Millisecond}
	// ...the other blocks on ctx.Done().
	slow := countingJob{name: "slow", runs: &runs}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		jobs.RunAll(ctx, newTestLogger(), fast, slow)
		close(done)
	}()

	// Wait long enough for `fast` to have returned but NOT cancel yet.
	time.Sleep(40 * time.Millisecond)

	select {
	case <-done:
		t.Fatalf("RunAll returned before ctx cancel even though one job is still running")
	default:
	}

	cancel()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("RunAll did not return after ctx cancel")
	}
}
