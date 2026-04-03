package worker

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/foundation/logger"
)

// RawInputQueuer is the subset of rawinputbus.Business the worker needs.
type RawInputQueuer interface {
	QueryRetryable(ctx context.Context, limit int) ([]rawinputbus.RawInput, error)
	MarkForRetry(ctx context.Context, ri rawinputbus.RawInput, errMsg string) (rawinputbus.RawInput, error)
	MarkFailed(ctx context.Context, ri rawinputbus.RawInput, errMsg string) (rawinputbus.RawInput, error)
}

// RawInputProcessor is the subset of ingestbus.Business the worker needs.
type RawInputProcessor interface {
	ProcessRawInputByID(ctx context.Context, id uuid.UUID) error
}

// IngestWorker polls for pending raw_inputs and runs the ingestion pipeline.
type IngestWorker struct {
	log       *logger.Logger
	riBus     RawInputQueuer
	igBus     RawInputProcessor
	interval  time.Duration
	batchSize int
}

// NewIngestWorker creates a new worker. Interval: 30s, batch size: 20.
func NewIngestWorker(log *logger.Logger, riBus RawInputQueuer, igBus RawInputProcessor) *IngestWorker {
	return &IngestWorker{
		log:       log,
		riBus:     riBus,
		igBus:     igBus,
		interval:  30 * time.Second,
		batchSize: 20,
	}
}

// Run starts the worker loop. Blocks until ctx is cancelled.
func (w *IngestWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Process immediately on start, then on each tick.
	w.ProcessBatch(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.ProcessBatch(ctx)
		}
	}
}

// ProcessBatch queries retryable items and dispatches each in a goroutine.
// Exported so tests can call it directly without waiting for ticker.
func (w *IngestWorker) ProcessBatch(ctx context.Context) {
	items, err := w.riBus.QueryRetryable(ctx, w.batchSize)
	if err != nil {
		w.log.Error(ctx, "ingestworker", "msg", "query retryable failed", "error", err)
		return
	}
	if len(items) == 0 {
		return
	}
	w.log.Info(ctx, "ingestworker", "msg", "processing batch", "count", len(items))

	for _, ri := range items {
		ri := ri // capture loop variable
		go func() {
			itemCtx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			err := w.igBus.ProcessRawInputByID(itemCtx, ri.ID)
			if err == nil {
				return // success; MarkProcessed was called inside ProcessRawInputByID
			}

			w.log.Error(itemCtx, "ingestworker", "msg", "pipeline failed",
				"id", ri.ID, "retry_count", ri.RetryCount, "error", err)

			if ri.RetryCount+1 >= ri.MaxRetries {
				if _, fErr := w.riBus.MarkFailed(itemCtx, ri, err.Error()); fErr != nil {
					w.log.Error(itemCtx, "ingestworker", "msg", "MarkFailed failed", "error", fErr)
				}
			} else {
				if _, fErr := w.riBus.MarkForRetry(itemCtx, ri, err.Error()); fErr != nil {
					w.log.Error(itemCtx, "ingestworker", "msg", "MarkForRetry failed", "error", fErr)
				}
			}
		}()
	}
}
