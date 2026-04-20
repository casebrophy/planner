package jobs_test

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/api/services/planner/jobs"
	"github.com/casebrophy/planner/business/domain/debriefbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/foundation/logger"
)

// mockWeeklyTaskBus satisfies jobs.TaskQuerier.
type mockWeeklyTaskBus struct {
	mu     sync.Mutex
	calls  int32
	tasks  []taskbus.Task
	err    error
}

func (m *mockWeeklyTaskBus) Query(_ context.Context, _ taskbus.QueryFilter, _ order.By, _ page.Page) ([]taskbus.Task, error) {
	atomic.AddInt32(&m.calls, 1)
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.tasks, m.err
}

func (m *mockWeeklyTaskBus) Calls() int32 { return atomic.LoadInt32(&m.calls) }

// mockWeeklyReviewer satisfies jobs.WeeklyReviewer.
type mockWeeklyReviewer struct {
	mu      sync.Mutex
	calls   int32
	lastID  string
	lastLen int
}

func (m *mockWeeklyReviewer) GenerateWeeklyReview(_ context.Context, weekID string, tasks []debriefbus.CompletedTask) error {
	atomic.AddInt32(&m.calls, 1)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastID = weekID
	m.lastLen = len(tasks)
	return nil
}

func (m *mockWeeklyReviewer) Calls() int32 { return atomic.LoadInt32(&m.calls) }

func newTestLogger() *logger.Logger {
	return logger.New(os.Stdout, logger.LevelError, "test")
}

// runJobFor starts job.Run in a goroutine, waits for waitFor, cancels ctx
// and blocks until Run returns (or the cancel deadline fires).
func runJobFor(t *testing.T, run func(context.Context), waitFor time.Duration) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		run(ctx)
		close(done)
	}()

	time.Sleep(waitFor)
	cancel()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("Run did not return promptly after ctx cancel")
	}
}

func TestWeeklyReviewJob_NonSundayIsNoop(t *testing.T) {
	tb := &mockWeeklyTaskBus{}
	rv := &mockWeeklyReviewer{}

	// Monday 18:30 — not Sunday, should be a no-op
	fixed := time.Date(2026, 4, 20, 18, 30, 0, 0, time.UTC)

	job := jobs.WeeklyReviewJob{
		Log:        newTestLogger(),
		TaskBus:    tb,
		DebriefBus: rv,
		Now:        func() time.Time { return fixed },
		Interval:   5 * time.Millisecond,
	}

	runJobFor(t, job.Run, 40*time.Millisecond)

	if tb.Calls() != 0 {
		t.Fatalf("expected 0 TaskBus.Query calls on non-Sunday, got %d", tb.Calls())
	}
	if rv.Calls() != 0 {
		t.Fatalf("expected 0 GenerateWeeklyReview calls on non-Sunday, got %d", rv.Calls())
	}
}

func TestWeeklyReviewJob_SundayFiresOnce(t *testing.T) {
	completedAt := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	tb := &mockWeeklyTaskBus{
		tasks: []taskbus.Task{
			{
				ID:          uuid.New(),
				Title:       "shipped thing",
				CompletedAt: &completedAt,
			},
		},
	}
	rv := &mockWeeklyReviewer{}

	// Sunday 2026-04-19 at 18:00 UTC
	fixed := time.Date(2026, 4, 19, 18, 0, 0, 0, time.UTC)

	job := jobs.WeeklyReviewJob{
		Log:        newTestLogger(),
		TaskBus:    tb,
		DebriefBus: rv,
		Now:        func() time.Time { return fixed },
		Interval:   5 * time.Millisecond,
	}

	runJobFor(t, job.Run, 60*time.Millisecond)

	if tb.Calls() < 1 {
		t.Fatalf("expected TaskBus.Query to fire at least once, got %d", tb.Calls())
	}
	if rv.Calls() != 1 {
		t.Fatalf("expected GenerateWeeklyReview to fire exactly once (lastGenWeek guard), got %d", rv.Calls())
	}
	if rv.lastLen != 1 {
		t.Fatalf("expected 1 recent task passed through, got %d", rv.lastLen)
	}
	if rv.lastID == "" {
		t.Fatalf("expected non-empty weekID")
	}
}

func TestWeeklyReviewJob_ContextCancelExits(t *testing.T) {
	tb := &mockWeeklyTaskBus{}
	rv := &mockWeeklyReviewer{}

	job := jobs.WeeklyReviewJob{
		Log:        newTestLogger(),
		TaskBus:    tb,
		DebriefBus: rv,
		Interval:   10 * time.Second,
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
		t.Fatalf("Run did not return after ctx timeout")
	}
}
