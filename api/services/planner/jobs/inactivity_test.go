package jobs_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/casebrophy/planner/api/services/planner/jobs"
	"github.com/casebrophy/planner/foundation/logger"
)

// mockInactivityChecker satisfies jobs.InactivityChecker. It returns an
// error on the first call and nil thereafter so tests can verify the
// loop survives errors.
type mockInactivityChecker struct {
	mu    sync.Mutex
	calls int32
	errOn int32 // 1-indexed tick number on which to return an error; 0 = never
}

func (m *mockInactivityChecker) CheckAll(_ context.Context) error {
	n := atomic.AddInt32(&m.calls, 1)
	if m.errOn != 0 && n == m.errOn {
		return errors.New("boom")
	}
	return nil
}

func (m *mockInactivityChecker) Calls() int32 {
	return atomic.LoadInt32(&m.calls)
}

func TestInactivityJob_TicksAndSurvivesErrors(t *testing.T) {
	log := logger.New(os.Stdout, logger.LevelInfo, "test")
	checker := &mockInactivityChecker{errOn: 1}

	job := jobs.InactivityJob{
		Log:      log,
		Checker:  checker,
		Interval: 5 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		job.Run(ctx)
		close(done)
	}()

	// Wait for at least two ticks: first returns error, second returns nil.
	// Poll with a bounded deadline so the test can't hang.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if checker.Calls() >= 2 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}

	if got := checker.Calls(); got < 2 {
		t.Fatalf("expected at least 2 CheckAll calls (error + recovery), got %d", got)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("Run did not return promptly after ctx cancel")
	}
}

func TestInactivityJob_ReturnsOnContextDone(t *testing.T) {
	log := logger.New(os.Stdout, logger.LevelInfo, "test")
	checker := &mockInactivityChecker{}

	job := jobs.InactivityJob{
		Log:      log,
		Checker:  checker,
		Interval: 10 * time.Second, // long enough that we exit via ctx, not tick
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		job.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("Run did not return after ctx timed out")
	}
}
