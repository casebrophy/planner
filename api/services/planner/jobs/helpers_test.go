package jobs_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/casebrophy/planner/foundation/logger"
)

func newTestLogger() *logger.Logger {
	return logger.New(os.Stdout, logger.LevelError, "test")
}

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
