package worker_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/sdk/worker"
	"github.com/casebrophy/planner/business/types/rawinputsource"
	"github.com/casebrophy/planner/business/types/rawinputstatus"
	"github.com/casebrophy/planner/foundation/logger"
)

// mockRawInputBus satisfies worker.RawInputQueuer.
type mockRawInputBus struct {
	mu              sync.Mutex
	retryableItems  []rawinputbus.RawInput
	markForRetryID  uuid.UUID
	markFailedID    uuid.UUID
}

func (m *mockRawInputBus) QueryRetryable(_ context.Context, _ int) ([]rawinputbus.RawInput, error) {
	return m.retryableItems, nil
}

func (m *mockRawInputBus) MarkForRetry(_ context.Context, ri rawinputbus.RawInput, _ string) (rawinputbus.RawInput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.markForRetryID = ri.ID
	return ri, nil
}

func (m *mockRawInputBus) MarkFailed(_ context.Context, ri rawinputbus.RawInput, _ string) (rawinputbus.RawInput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.markFailedID = ri.ID
	return ri, nil
}

// mockIngestBus satisfies worker.RawInputProcessor.
type mockIngestBus struct {
	processErr error
}

func (m *mockIngestBus) ProcessRawInputByID(_ context.Context, _ uuid.UUID) error {
	return m.processErr
}

func newTestItem(retryCount, maxRetries int) rawinputbus.RawInput {
	return rawinputbus.RawInput{
		ID:         uuid.New(),
		SourceType: rawinputsource.Voice,
		Status:     rawinputstatus.Pending,
		RawContent: "test",
		RetryCount: retryCount,
		MaxRetries: maxRetries,
		CreatedAt:  time.Now(),
	}
}

func TestWorker_ProcessesRetryableItems(t *testing.T) {
	log := logger.New(os.Stdout, logger.LevelInfo, "test")
	item := newTestItem(0, 5)
	riBus := &mockRawInputBus{retryableItems: []rawinputbus.RawInput{item}}
	igBus := &mockIngestBus{processErr: nil}

	w := worker.NewIngestWorker(log, riBus, igBus)
	w.ProcessBatch(context.Background())

	time.Sleep(100 * time.Millisecond)

	riBus.mu.Lock()
	defer riBus.mu.Unlock()
	if riBus.markForRetryID != uuid.Nil {
		t.Errorf("expected no MarkForRetry call on success")
	}
	if riBus.markFailedID != uuid.Nil {
		t.Errorf("expected no MarkFailed call on success")
	}
}

func TestWorker_SchedulesRetryOnFailure(t *testing.T) {
	log := logger.New(os.Stdout, logger.LevelInfo, "test")
	item := newTestItem(1, 5)
	riBus := &mockRawInputBus{retryableItems: []rawinputbus.RawInput{item}}
	igBus := &mockIngestBus{processErr: errors.New("claude timeout")}

	w := worker.NewIngestWorker(log, riBus, igBus)
	w.ProcessBatch(context.Background())

	time.Sleep(100 * time.Millisecond)

	riBus.mu.Lock()
	defer riBus.mu.Unlock()
	if riBus.markForRetryID != item.ID {
		t.Errorf("expected MarkForRetry(%s), got %s", item.ID, riBus.markForRetryID)
	}
	if riBus.markFailedID != uuid.Nil {
		t.Errorf("expected no MarkFailed when retries remain")
	}
}

func TestIngestWorker_Name(t *testing.T) {
	w := worker.NewIngestWorker(nil, nil, nil)
	if got := w.Name(); got != "ingest" {
		t.Fatalf("Name() = %q, want %q", got, "ingest")
	}
}

func TestWorker_MarksTerminalFailWhenRetriesExhausted(t *testing.T) {
	log := logger.New(os.Stdout, logger.LevelInfo, "test")
	item := newTestItem(4, 5) // next attempt is 5th = max
	riBus := &mockRawInputBus{retryableItems: []rawinputbus.RawInput{item}}
	igBus := &mockIngestBus{processErr: errors.New("persistent failure")}

	w := worker.NewIngestWorker(log, riBus, igBus)
	w.ProcessBatch(context.Background())

	time.Sleep(100 * time.Millisecond)

	riBus.mu.Lock()
	defer riBus.mu.Unlock()
	if riBus.markFailedID != item.ID {
		t.Errorf("expected MarkFailed(%s), got %s", item.ID, riBus.markFailedID)
	}
	if riBus.markForRetryID != uuid.Nil {
		t.Errorf("expected no MarkForRetry when max retries exhausted")
	}
}
