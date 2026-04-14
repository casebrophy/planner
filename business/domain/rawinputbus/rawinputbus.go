package rawinputbus

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/rawinputstatus"
	"github.com/casebrophy/planner/foundation/logger"
)

type Storer interface {
	Create(ctx context.Context, ri RawInput) error
	Update(ctx context.Context, ri RawInput) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]RawInput, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (RawInput, error)
	QueryRetryable(ctx context.Context, limit int) ([]RawInput, error)
	ResetForReprocess(ctx context.Context, id uuid.UUID) (RawInput, error)
}

type Business struct {
	log    *logger.Logger
	storer Storer
}

func NewBusiness(log *logger.Logger, storer Storer) *Business {
	return &Business{log: log, storer: storer}
}

func (b *Business) Create(ctx context.Context, nri NewRawInput) (RawInput, error) {
	now := time.Now()
	ri := RawInput{
		ID:         uuid.New(),
		SourceType: nri.SourceType,
		Status:     rawinputstatus.Pending,
		RawContent: nri.RawContent,
		MaxRetries: 5,
		CreatedAt:  now,
	}
	if err := b.storer.Create(ctx, ri); err != nil {
		return RawInput{}, fmt.Errorf("create: %w", err)
	}
	return ri, nil
}

func (b *Business) Update(ctx context.Context, ri RawInput, uri UpdateRawInput) (RawInput, error) {
	if uri.Status != nil {
		ri.Status = *uri.Status
	}
	if uri.ProcessedAt != nil {
		ri.ProcessedAt = uri.ProcessedAt
	}
	if uri.Error != nil {
		ri.Error = uri.Error
	}
	if uri.RetryCount != nil {
		ri.RetryCount = *uri.RetryCount
	}
	if uri.NextRetryAt != nil {
		ri.NextRetryAt = uri.NextRetryAt
	}
	if uri.Result != nil {
		ri.Result = *uri.Result
	}
	if err := b.storer.Update(ctx, ri); err != nil {
		return RawInput{}, fmt.Errorf("update: %w", err)
	}
	return ri, nil
}

func (b *Business) MarkProcessing(ctx context.Context, ri RawInput) (RawInput, error) {
	s := rawinputstatus.Processing
	return b.Update(ctx, ri, UpdateRawInput{Status: &s})
}

func (b *Business) MarkProcessed(ctx context.Context, ri RawInput) (RawInput, error) {
	s := rawinputstatus.Processed
	now := time.Now()
	return b.Update(ctx, ri, UpdateRawInput{Status: &s, ProcessedAt: &now})
}

func (b *Business) MarkFailed(ctx context.Context, ri RawInput, errMsg string) (RawInput, error) {
	// Use a detached context so we can still write to DB even if the
	// parent context was cancelled (e.g., request timeout).
	failCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	s := rawinputstatus.Failed
	return b.Update(failCtx, ri, UpdateRawInput{Status: &s, Error: &errMsg})
}

// MarkForRetry schedules the next retry attempt for a raw_input that failed processing.
// ri should be the item as read before the current (failed) attempt.
// Increments retry_count and sets next_retry_at with exponential backoff.
// Sets status back to pending so the worker will pick it up again.
func (b *Business) MarkForRetry(ctx context.Context, ri RawInput, errMsg string) (RawInput, error) {
	newCount := ri.RetryCount + 1
	backoff := ComputeBackoff(newCount)
	nextRetry := time.Now().Add(backoff)
	s := rawinputstatus.Pending
	b.log.Info(ctx, "rawinputbus", "msg", "scheduling retry",
		"id", ri.ID, "retry_count", newCount, "next_retry_at", nextRetry)
	return b.Update(ctx, ri, UpdateRawInput{
		Status:      &s,
		RetryCount:  &newCount,
		NextRetryAt: &nextRetry,
		Error:       &errMsg,
	})
}

// ComputeBackoff returns the wait duration before the nth retry.
// Formula: 2^n minutes, capped at 30 minutes.
// Exported for testing.
func ComputeBackoff(retryCount int) time.Duration {
	d := time.Duration(math.Pow(2, float64(retryCount))) * time.Minute
	const maxBackoff = 30 * time.Minute
	if d > maxBackoff {
		return maxBackoff
	}
	return d
}

// QueryRetryable returns raw_inputs ready to process: status=pending
// where next_retry_at IS NULL or has passed.
func (b *Business) QueryRetryable(ctx context.Context, limit int) ([]RawInput, error) {
	items, err := b.storer.QueryRetryable(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("query retryable: %w", err)
	}
	return items, nil
}

// ResetForReprocess resets a raw_input to initial pending state for manual reprocessing.
// Clears retry_count, next_retry_at, and error.
// Only allows reprocessing if status is failed or pending; returns error otherwise.
func (b *Business) ResetForReprocess(ctx context.Context, id uuid.UUID) (RawInput, error) {
	// Fetch the raw input to check its status
	ri, err := b.storer.QueryByID(ctx, id)
	if err != nil {
		return RawInput{}, fmt.Errorf("reset for reprocess: %w", err)
	}

	// Guard: only allow reprocessing if status is failed, pending, or processed
	if ri.Status != rawinputstatus.Failed && ri.Status != rawinputstatus.Pending && ri.Status != rawinputstatus.Processed {
		return RawInput{}, fmt.Errorf("reset for reprocess: cannot reprocess item with status %s; only failed, pending, or processed items can be reprocessed", ri.Status)
	}

	// Proceed with reset
	result, err := b.storer.ResetForReprocess(ctx, id)
	if err != nil {
		return RawInput{}, fmt.Errorf("reset for reprocess: %w", err)
	}
	return result, nil
}

// RecoverStuck finds raw_inputs stuck in "processing" and marks them failed.
func (b *Business) RecoverStuck(ctx context.Context, threshold time.Duration) (int, error) {
	processingStatus := rawinputstatus.Processing
	items, err := b.storer.Query(ctx, QueryFilter{Status: &processingStatus}, DefaultOrderBy, page.New(1, 100))
	if err != nil {
		return 0, fmt.Errorf("query stuck: %w", err)
	}
	cutoff := time.Now().Add(-threshold)
	var recovered int
	for _, ri := range items {
		if ri.CreatedAt.Before(cutoff) {
			errMsg := fmt.Sprintf("recovered: stuck in processing for longer than %s", threshold)
			if _, err := b.MarkFailed(ctx, ri, errMsg); err != nil {
				b.log.Error(ctx, "rawinputbus", "msg", "failed to recover stuck raw_input", "id", ri.ID, "error", err)
				continue
			}
			recovered++
		}
	}
	return recovered, nil
}

func (b *Business) Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]RawInput, error) {
	ris, err := b.storer.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return ris, nil
}

func (b *Business) Count(ctx context.Context, filter QueryFilter) (int, error) {
	n, err := b.storer.Count(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	return n, nil
}

func (b *Business) QueryByID(ctx context.Context, id uuid.UUID) (RawInput, error) {
	ri, err := b.storer.QueryByID(ctx, id)
	if err != nil {
		return RawInput{}, fmt.Errorf("query by id[%s]: %w", id, err)
	}
	return ri, nil
}
