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

// mockStuckRecoverer satisfies jobs.StuckRecoverer. Returns error on the
// first call, (2, nil) thereafter. Also records the threshold it was
// invoked with so tests can verify the default.
type mockStuckRecoverer struct {
	mu             sync.Mutex
	calls          int32
	errOn          int32
	lastThreshold  time.Duration
}

func (m *mockStuckRecoverer) RecoverStuck(_ context.Context, threshold time.Duration) (int, error) {
	m.mu.Lock()
	m.lastThreshold = threshold
	m.mu.Unlock()
	n := atomic.AddInt32(&m.calls, 1)
	if m.errOn != 0 && n == m.errOn {
		return 0, errors.New("recover failed")
	}
	return 2, nil
}

func (m *mockStuckRecoverer) Calls() int32 {
	return atomic.LoadInt32(&m.calls)
}

func (m *mockStuckRecoverer) Threshold() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastThreshold
}

func TestRawInputRecoveryJob_TicksAndSurvivesErrors(t *testing.T) {
	log := logger.New(os.Stdout, logger.LevelInfo, "test")
	bus := &mockStuckRecoverer{errOn: 1}

	job := jobs.RawInputRecoveryJob{
		Log:      log,
		Bus:      bus,
		Interval: 5 * time.Millisecond,
		// Threshold left zero so the default (15m) applies.
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		job.Run(ctx)
		close(done)
	}()

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if bus.Calls() >= 2 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}

	if got := bus.Calls(); got < 2 {
		t.Fatalf("expected at least 2 RecoverStuck calls (error + recovery), got %d", got)
	}

	if got := bus.Threshold(); got != 15*time.Minute {
		t.Errorf("expected default threshold 15m, got %s", got)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("Run did not return promptly after ctx cancel")
	}
}

func TestRawInputRecoveryJob_ReturnsOnContextDone(t *testing.T) {
	log := logger.New(os.Stdout, logger.LevelInfo, "test")
	bus := &mockStuckRecoverer{}

	job := jobs.RawInputRecoveryJob{
		Log:      log,
		Bus:      bus,
		Interval: 10 * time.Second,
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
