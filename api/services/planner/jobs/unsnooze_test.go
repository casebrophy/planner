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

// mockUnsnoozer satisfies jobs.Unsnoozer. It returns an error on the
// first call and (3, nil) thereafter.
type mockUnsnoozer struct {
	mu    sync.Mutex
	calls int32
	errOn int32
}

func (m *mockUnsnoozer) UnsnoozeExpired(_ context.Context) (int, error) {
	n := atomic.AddInt32(&m.calls, 1)
	if m.errOn != 0 && n == m.errOn {
		return 0, errors.New("db down")
	}
	return 3, nil
}

func (m *mockUnsnoozer) Calls() int32 {
	return atomic.LoadInt32(&m.calls)
}

func TestUnsnoozeJob_TicksAndSurvivesErrors(t *testing.T) {
	log := logger.New(os.Stdout, logger.LevelInfo, "test")
	bus := &mockUnsnoozer{errOn: 1}

	job := jobs.UnsnoozeJob{
		Log:      log,
		Bus:      bus,
		Interval: 5 * time.Millisecond,
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
		t.Fatalf("expected at least 2 UnsnoozeExpired calls (error + recovery), got %d", got)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("Run did not return promptly after ctx cancel")
	}
}

func TestUnsnoozeJob_ReturnsOnContextDone(t *testing.T) {
	log := logger.New(os.Stdout, logger.LevelInfo, "test")
	bus := &mockUnsnoozer{}

	job := jobs.UnsnoozeJob{
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
