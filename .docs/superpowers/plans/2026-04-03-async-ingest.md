# Async Ingest Pipeline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the synchronous ingestion pipeline (which times out) with a background worker that processes raw inputs asynchronously, with exponential-backoff retry and a frontend queue UI for monitoring and manual reprocess.

**Architecture:** HTTP handlers enqueue raw inputs (store to DB, return immediately) and a ticker-based background goroutine polls for pending items and runs the full pipeline. Failed items are rescheduled with backoff; after exhausting retries they reach terminal `failed` state. Frontend polls `/api/v1/raw-inputs` for a management view.

**Tech Stack:** Go (business layer worker, no external queue), PostgreSQL (retry scheduling via `next_retry_at`), Vue 3 + Pinia (frontend queue view), Vitest (frontend tests), `business/sdk/dbtest` (backend store tests).

---

## File Map

| Action | File | Why |
|--------|------|-----|
| MODIFY | `business/sdk/migrate/sql/migrate.sql` | Add retry_count, next_retry_at, max_retries to raw_inputs |
| MODIFY | `business/domain/rawinputbus/model.go` | Add retry fields to RawInput, UpdateRawInput |
| MODIFY | `business/domain/rawinputbus/rawinputbus.go` | Extend Storer, add MarkForRetry / QueryRetryable / ResetForReprocess |
| MODIFY | `business/domain/rawinputbus/stores/rawinputdb/model.go` | DB struct + converters for new fields |
| MODIFY | `business/domain/rawinputbus/stores/rawinputdb/rawinputdb.go` | SQL for new fields, QueryRetryable, ResetForReprocess |
| MODIFY | `business/domain/ingestbus/ingestbus.go` | Add EnqueueEmail, EnqueueText, ProcessRawInputByID |
| CREATE | `business/sdk/worker/ingestworker.go` | Ticker-based background worker |
| MODIFY | `app/domain/voiceingestapp/voiceingestapp.go` | Call EnqueueText, return immediately |
| MODIFY | `app/domain/voiceingestapp/model.go` | Simplify response to rawInputId |
| MODIFY | `app/domain/rawinputapp/rawinputapp.go` | reprocess → ResetForReprocess + return 200 immediately |
| MODIFY | `api/services/planner/main.go` | Move igBus creation out of SMTP block, start worker goroutine |
| MODIFY | `api/services/frontend/web/src/types/rawinput.ts` | Add retryCount, nextRetryAt, maxRetries |
| MODIFY | `api/services/frontend/web/src/services/rawinputService.ts` | Add list(filter), reprocess(), countNonProcessed() |
| CREATE | `api/services/frontend/web/src/stores/rawinputStore.ts` | Pinia store for queue UI |
| CREATE | `api/services/frontend/web/src/views/RawInputQueueView.vue` | Table + status filter + reprocess buttons |
| MODIFY | `api/services/frontend/web/src/router/index.ts` | Register /ingest-queue route |

---

## Task 1: DB Migration — Add Retry Fields

**Files:**
- Modify: `business/sdk/migrate/sql/migrate.sql`

- [ ] **Step 1: Add migration 1.19 at the end of migrate.sql**

```sql
-- Version 1.19
ALTER TABLE raw_inputs
    ADD COLUMN retry_count   INT         NOT NULL DEFAULT 0,
    ADD COLUMN next_retry_at TIMESTAMPTZ,
    ADD COLUMN max_retries   INT         NOT NULL DEFAULT 5;

CREATE INDEX idx_raw_inputs_retryable ON raw_inputs(created_at)
    WHERE status = 'pending';
```

- [ ] **Step 2: Run migration against dev DB**

```bash
make migrate
```

Expected: migration succeeds with no errors.

- [ ] **Step 3: Verify columns exist**

```bash
PGPASSWORD=planner psql -h localhost -p 5433 -U planner -d planner -c "\d raw_inputs"
```

Expected: `retry_count`, `next_retry_at`, `max_retries` appear in the column list.

- [ ] **Step 4: Commit**

```bash
git add business/sdk/migrate/sql/migrate.sql
git commit -m "feat: add retry fields to raw_inputs (migration 1.19)"
```

---

## Task 2: rawinputbus Model + Business Layer

**Files:**
- Modify: `business/domain/rawinputbus/model.go`
- Modify: `business/domain/rawinputbus/rawinputbus.go`

- [ ] **Step 1: Write the failing test**

Create `business/domain/rawinputbus/rawinputbus_test.go`:

```go
package rawinputbus_test

import (
	"testing"
	"time"

	"github.com/casebrophy/planner/business/domain/rawinputbus"
)

func TestComputeBackoff(t *testing.T) {
	tests := []struct {
		retryCount int
		wantMin    time.Duration
		wantMax    time.Duration
	}{
		{1, 1*time.Minute + 50*time.Second, 2*time.Minute + 10*time.Second}, // ~2 min
		{2, 3*time.Minute + 50*time.Second, 4*time.Minute + 10*time.Second}, // ~4 min
		{5, 29*time.Minute, 30*time.Minute + 10*time.Second},                // capped at 30
		{10, 29*time.Minute, 30*time.Minute + 10*time.Second},               // still capped
	}
	for _, tt := range tests {
		got := rawinputbus.ComputeBackoff(tt.retryCount)
		if got < tt.wantMin || got > tt.wantMax {
			t.Errorf("ComputeBackoff(%d) = %v; want between %v and %v", tt.retryCount, got, tt.wantMin, tt.wantMax)
		}
	}
}
```

- [ ] **Step 2: Run to confirm it fails**

```bash
go test ./business/domain/rawinputbus/... -run TestComputeBackoff -count=1
```

Expected: `undefined: rawinputbus.ComputeBackoff`

- [ ] **Step 3: Update model.go — add retry fields**

In `business/domain/rawinputbus/model.go`, add to `RawInput` and `UpdateRawInput`:

```go
type RawInput struct {
	ID          uuid.UUID
	SourceType  rawinputsource.Source
	Status      rawinputstatus.Status
	RawContent  string
	ProcessedAt *time.Time
	Error       *string
	RetryCount  int
	NextRetryAt *time.Time
	MaxRetries  int
	CreatedAt   time.Time
}

// (NewRawInput stays unchanged)

type UpdateRawInput struct {
	Status      *rawinputstatus.Status
	ProcessedAt *time.Time
	Error       *string
	RetryCount  *int
	NextRetryAt *time.Time
}
```

- [ ] **Step 4: Update rawinputbus.go — Storer, Update, and new methods**

Replace `rawinputbus.go` content with the following (keep all existing methods, add new ones):

```go
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
	failCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	s := rawinputstatus.Failed
	return b.Update(failCtx, ri, UpdateRawInput{Status: &s, Error: &errMsg})
}

// MarkForRetry schedules the next retry attempt for a raw_input that failed processing.
// The ri parameter should be the item as read before the current (failed) attempt.
// Sets status back to pending with exponential backoff on next_retry_at.
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

// ComputeBackoff returns the duration to wait before the nth retry.
// Exponential: 2^n minutes, capped at 30 minutes.
// Exported so it can be tested directly.
func ComputeBackoff(retryCount int) time.Duration {
	d := time.Duration(math.Pow(2, float64(retryCount))) * time.Minute
	const maxBackoff = 30 * time.Minute
	if d > maxBackoff {
		return maxBackoff
	}
	return d
}

// QueryRetryable returns raw_inputs ready for processing: status=pending
// and (next_retry_at IS NULL OR next_retry_at <= NOW()).
func (b *Business) QueryRetryable(ctx context.Context, limit int) ([]RawInput, error) {
	items, err := b.storer.QueryRetryable(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("query retryable: %w", err)
	}
	return items, nil
}

// ResetForReprocess resets a raw_input to its initial pending state so the
// worker will reprocess it. Clears retry_count, next_retry_at, and error.
func (b *Business) ResetForReprocess(ctx context.Context, id uuid.UUID) (RawInput, error) {
	ri, err := b.storer.ResetForReprocess(ctx, id)
	if err != nil {
		return RawInput{}, fmt.Errorf("reset for reprocess: %w", err)
	}
	return ri, nil
}

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
```

- [ ] **Step 5: Run test to confirm it passes**

```bash
go test ./business/domain/rawinputbus/... -run TestComputeBackoff -count=1
```

Expected: PASS

- [ ] **Step 6: Verify it compiles**

```bash
go build ./business/domain/rawinputbus/...
```

Expected: no errors (the storer interface will have new methods but rawinputdb won't compile until Task 3 — that's fine if building just this package).

- [ ] **Step 7: Commit**

```bash
git add business/domain/rawinputbus/model.go business/domain/rawinputbus/rawinputbus.go business/domain/rawinputbus/rawinputbus_test.go
git commit -m "feat: add retry scheduling to rawinputbus (MarkForRetry, QueryRetryable, ResetForReprocess)"
```

---

## Task 3: rawinputdb Store — New Fields + SQL

**Files:**
- Modify: `business/domain/rawinputbus/stores/rawinputdb/model.go`
- Modify: `business/domain/rawinputbus/stores/rawinputdb/rawinputdb.go`

- [ ] **Step 1: Update model.go — add DB struct fields and converters**

Replace `business/domain/rawinputbus/stores/rawinputdb/model.go`:

```go
package rawinputdb

import (
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/types/rawinputsource"
	"github.com/casebrophy/planner/business/types/rawinputstatus"
)

type rawInputDB struct {
	ID          uuid.UUID  `db:"raw_input_id"`
	SourceType  string     `db:"source_type"`
	Status      string     `db:"status"`
	RawContent  string     `db:"raw_content"`
	ProcessedAt *time.Time `db:"processed_at"`
	Error       *string    `db:"error"`
	RetryCount  int        `db:"retry_count"`
	NextRetryAt *time.Time `db:"next_retry_at"`
	MaxRetries  int        `db:"max_retries"`
	CreatedAt   time.Time  `db:"created_at"`
}

func toDBRawInput(ri rawinputbus.RawInput) rawInputDB {
	return rawInputDB{
		ID:          ri.ID,
		SourceType:  ri.SourceType.String(),
		Status:      ri.Status.String(),
		RawContent:  ri.RawContent,
		ProcessedAt: ri.ProcessedAt,
		Error:       ri.Error,
		RetryCount:  ri.RetryCount,
		NextRetryAt: ri.NextRetryAt,
		MaxRetries:  ri.MaxRetries,
		CreatedAt:   ri.CreatedAt,
	}
}

func toBusRawInput(ri rawInputDB) rawinputbus.RawInput {
	return rawinputbus.RawInput{
		ID:          ri.ID,
		SourceType:  rawinputsource.MustParse(ri.SourceType),
		Status:      rawinputstatus.MustParse(ri.Status),
		RawContent:  ri.RawContent,
		ProcessedAt: ri.ProcessedAt,
		Error:       ri.Error,
		RetryCount:  ri.RetryCount,
		NextRetryAt: ri.NextRetryAt,
		MaxRetries:  ri.MaxRetries,
		CreatedAt:   ri.CreatedAt,
	}
}

func toBusRawInputs(ris []rawInputDB) []rawinputbus.RawInput {
	items := make([]rawinputbus.RawInput, len(ris))
	for i, ri := range ris {
		items[i] = toBusRawInput(ri)
	}
	return items
}
```

- [ ] **Step 2: Update rawinputdb.go — fix SQL + add QueryRetryable + ResetForReprocess**

Replace the SQL in the existing methods and add two new ones.

**Create** (add retry/max columns to INSERT):
```go
func (s *Store) Create(ctx context.Context, ri rawinputbus.RawInput) error {
	const q = `
	INSERT INTO raw_inputs
		(raw_input_id, source_type, status, raw_content, processed_at, error, retry_count, next_retry_at, max_retries, created_at)
	VALUES
		(:raw_input_id, :source_type, :status, :raw_content, :processed_at, :error, :retry_count, :next_retry_at, :max_retries, :created_at)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBRawInput(ri)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}
	return nil
}
```

**Update** (add retry_count and next_retry_at to SET):
```go
func (s *Store) Update(ctx context.Context, ri rawinputbus.RawInput) error {
	const q = `
	UPDATE raw_inputs SET
		status       = :status,
		processed_at = :processed_at,
		error        = :error,
		retry_count  = :retry_count,
		next_retry_at = :next_retry_at
	WHERE
		raw_input_id = :raw_input_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBRawInput(ri)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}
	return nil
}
```

**Query and QueryByID** — update SELECT to include new columns:
```go
// In Query(), change the SELECT line to:
buf.WriteString(`SELECT raw_input_id, source_type, status, raw_content, processed_at, error, retry_count, next_retry_at, max_retries, created_at FROM raw_inputs WHERE 1=1`)

// In QueryByID(), change the SELECT:
const q = `SELECT raw_input_id, source_type, status, raw_content, processed_at, error, retry_count, next_retry_at, max_retries, created_at FROM raw_inputs WHERE raw_input_id = :raw_input_id`
```

**Add QueryRetryable** (append to rawinputdb.go):
```go
func (s *Store) QueryRetryable(ctx context.Context, limit int) ([]rawinputbus.RawInput, error) {
	data := map[string]any{
		"limit": limit,
	}

	const q = `
	SELECT raw_input_id, source_type, status, raw_content, processed_at, error, retry_count, next_retry_at, max_retries, created_at
	FROM raw_inputs
	WHERE status = 'pending'
	  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
	ORDER BY created_at ASC
	FETCH NEXT :limit ROWS ONLY`

	dbItems, err := sqldb.NamedQuerySlice[rawInputDB](ctx, s.log, s.db, q, data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}
	return toBusRawInputs(dbItems), nil
}
```

**Add ResetForReprocess** (append to rawinputdb.go):
```go
func (s *Store) ResetForReprocess(ctx context.Context, id uuid.UUID) (rawinputbus.RawInput, error) {
	data := struct {
		ID uuid.UUID `db:"raw_input_id"`
	}{ID: id}

	const q = `
	UPDATE raw_inputs
	SET status = 'pending', retry_count = 0, next_retry_at = NULL, error = NULL
	WHERE raw_input_id = :raw_input_id
	RETURNING raw_input_id, source_type, status, raw_content, processed_at, error, retry_count, next_retry_at, max_retries, created_at`

	var ri rawInputDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &ri); err != nil {
		return rawinputbus.RawInput{}, fmt.Errorf("namedquerystruct: %w", err)
	}
	return toBusRawInput(ri), nil
}
```

- [ ] **Step 3: Verify it compiles**

```bash
go build ./business/domain/rawinputbus/...
```

Expected: no errors.

- [ ] **Step 4: Run existing tests**

```bash
go test ./business/domain/rawinputbus/... -count=1
```

Expected: PASS (TestComputeBackoff still passes).

- [ ] **Step 5: Commit**

```bash
git add business/domain/rawinputbus/stores/rawinputdb/model.go business/domain/rawinputbus/stores/rawinputdb/rawinputdb.go
git commit -m "feat: update rawinputdb store for retry fields (QueryRetryable, ResetForReprocess)"
```

---

## Task 4: ingestbus — Enqueue Methods + ProcessRawInputByID

**Files:**
- Modify: `business/domain/ingestbus/ingestbus.go`

The goal: add three new methods.
- `EnqueueEmail(ctx, rawContent) (uuid.UUID, error)` — stores raw_input, returns ID immediately
- `EnqueueText(ctx, rawContent) (uuid.UUID, error)` — stores raw_input, returns ID immediately
- `ProcessRawInputByID(ctx, id) error` — runs full pipeline; used by worker

`ProcessEmail` and `ProcessText` are kept untouched (the SMTP server still calls `ProcessEmail`).

- [ ] **Step 1: Add the three methods to ingestbus.go**

Append to `business/domain/ingestbus/ingestbus.go` (after the existing `ProcessText` function):

```go
// EnqueueEmail creates a raw_input record for an email and returns immediately.
// The background worker will process it asynchronously.
func (b *Business) EnqueueEmail(ctx context.Context, rawContent string) (uuid.UUID, error) {
	ri, err := b.rawInputBus.Create(ctx, rawinputbus.NewRawInput{
		SourceType: rawinputsource.Email,
		RawContent: rawContent,
	})
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("enqueue email: %w", err)
	}
	return ri.ID, nil
}

// EnqueueText creates a raw_input record for a voice/text capture and returns immediately.
// The background worker will process it asynchronously.
func (b *Business) EnqueueText(ctx context.Context, rawContent string) (uuid.UUID, error) {
	ri, err := b.rawInputBus.Create(ctx, rawinputbus.NewRawInput{
		SourceType: rawinputsource.Voice,
		RawContent: rawContent,
	})
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("enqueue text: %w", err)
	}
	return ri.ID, nil
}

// ProcessRawInputByID runs the full ingestion pipeline for an existing raw_input.
// It is the entry point for the background worker. On failure it returns the
// error WITHOUT calling MarkFailed — the caller (worker) is responsible for
// deciding whether to retry or mark terminal.
func (b *Business) ProcessRawInputByID(ctx context.Context, id uuid.UUID) error {
	ri, err := b.rawInputBus.QueryByID(ctx, id)
	if err != nil {
		return fmt.Errorf("query raw input: %w", err)
	}

	ri, err = b.rawInputBus.MarkProcessing(ctx, ri)
	if err != nil {
		return fmt.Errorf("mark processing: %w", err)
	}

	switch ri.SourceType {
	case rawinputsource.Email:
		return b.processRawInput(ctx, ri, ri.RawContent)
	case rawinputsource.Voice:
		_, err := b.processTextInput(ctx, ri, ri.RawContent)
		return err
	default:
		return fmt.Errorf("unknown source type: %s", ri.SourceType)
	}
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./business/domain/ingestbus/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add business/domain/ingestbus/ingestbus.go
git commit -m "feat: add EnqueueEmail, EnqueueText, ProcessRawInputByID to ingestbus"
```

---

## Task 5: Background Ingest Worker

**Files:**
- Create: `business/sdk/worker/ingestworker.go`

The worker polls for retryable items every 30 seconds and processes them in goroutines. On failure, it either schedules a retry (MarkForRetry) or marks terminal (MarkFailed) based on retry_count vs max_retries.

- [ ] **Step 1: Write the failing tests**

Create `business/sdk/worker/ingestworker_test.go`:

```go
package worker_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/sdk/worker"
	"github.com/casebrophy/planner/business/types/rawinputsource"
	"github.com/casebrophy/planner/business/types/rawinputstatus"
	"github.com/casebrophy/planner/foundation/logger"
	"os"
)

// mockRawInputBus satisfies worker.RawInputQueuer.
type mockRawInputBus struct {
	retryableItems []rawinputbus.RawInput
	markForRetryID uuid.UUID
	markFailedID   uuid.UUID
}

func (m *mockRawInputBus) QueryRetryable(_ context.Context, _ int) ([]rawinputbus.RawInput, error) {
	return m.retryableItems, nil
}

func (m *mockRawInputBus) MarkForRetry(_ context.Context, ri rawinputbus.RawInput, _ string) (rawinputbus.RawInput, error) {
	m.markForRetryID = ri.ID
	return ri, nil
}

func (m *mockRawInputBus) MarkFailed(_ context.Context, ri rawinputbus.RawInput, _ string) (rawinputbus.RawInput, error) {
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

	// Sleep briefly to let goroutines complete
	time.Sleep(50 * time.Millisecond)

	// On success: no retry, no failed
	if riBus.markForRetryID != uuid.Nil {
		t.Errorf("expected no MarkForRetry call on success")
	}
	if riBus.markFailedID != uuid.Nil {
		t.Errorf("expected no MarkFailed call on success")
	}
}

func TestWorker_SchedulesRetryOnFailure(t *testing.T) {
	log := logger.New(os.Stdout, logger.LevelInfo, "test")
	item := newTestItem(1, 5) // retry_count=1, not yet at max
	riBus := &mockRawInputBus{retryableItems: []rawinputbus.RawInput{item}}
	igBus := &mockIngestBus{processErr: errors.New("claude timeout")}

	w := worker.NewIngestWorker(log, riBus, igBus)
	w.ProcessBatch(context.Background())

	time.Sleep(50 * time.Millisecond)

	if riBus.markForRetryID != item.ID {
		t.Errorf("expected MarkForRetry(%s), got %s", item.ID, riBus.markForRetryID)
	}
	if riBus.markFailedID != uuid.Nil {
		t.Errorf("expected no MarkFailed when retries remain")
	}
}

func TestWorker_MarksTerminalFailWhenRetriesExhausted(t *testing.T) {
	log := logger.New(os.Stdout, logger.LevelInfo, "test")
	item := newTestItem(4, 5) // retry_count=4, next attempt is the 5th = max
	riBus := &mockRawInputBus{retryableItems: []rawinputbus.RawInput{item}}
	igBus := &mockIngestBus{processErr: errors.New("persistent failure")}

	w := worker.NewIngestWorker(log, riBus, igBus)
	w.ProcessBatch(context.Background())

	time.Sleep(50 * time.Millisecond)

	if riBus.markFailedID != item.ID {
		t.Errorf("expected MarkFailed(%s), got %s", item.ID, riBus.markFailedID)
	}
	if riBus.markForRetryID != uuid.Nil {
		t.Errorf("expected no MarkForRetry when max retries exhausted")
	}
}
```

- [ ] **Step 2: Run to confirm they fail**

```bash
go test ./business/sdk/worker/... -count=1 2>&1 | head -5
```

Expected: package not found or compilation errors.

- [ ] **Step 3: Create `business/sdk/worker/ingestworker.go`**

```go
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
	log      *logger.Logger
	riBus    RawInputQueuer
	igBus    RawInputProcessor
	interval time.Duration
	batchSize int
}

// NewIngestWorker creates a new worker. Interval defaults to 30s, batch size to 20.
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
```

- [ ] **Step 4: Run tests**

```bash
go test ./business/sdk/worker/... -count=1 -v
```

Expected: all three TestWorker_* tests PASS.

- [ ] **Step 5: Commit**

```bash
git add business/sdk/worker/ingestworker.go business/sdk/worker/ingestworker_test.go
git commit -m "feat: add background ingest worker with exponential-backoff retry"
```

---

## Task 6: HTTP Handlers — Async Return

**Files:**
- Modify: `app/domain/voiceingestapp/voiceingestapp.go`
- Modify: `app/domain/voiceingestapp/model.go`
- Modify: `app/domain/rawinputapp/rawinputapp.go`

### voiceingestapp

- [ ] **Step 1: Update voiceingestapp/model.go — simplify response**

Replace the `ingestResponse` struct (keep `ingestRequest` unchanged):

```go
type ingestResponse struct {
	RawInputID string `json:"rawInputId"`
}

func (r ingestResponse) Encode() ([]byte, string, error) {
	data, err := json.Marshal(r)
	return data, "application/json", err
}
```

Remove any unused imports (`encoding/json` stays, remove any others that were only for the old response).

- [ ] **Step 2: Update voiceingestapp/voiceingestapp.go — call EnqueueText**

Replace the `ingest` handler body:

```go
func (a *app) ingest(ctx context.Context, r *http.Request) web.Encoder {
	var req ingestRequest
	if err := web.Decode(r, &req); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if req.Text == "" {
		return errs.Newf(errs.InvalidArgument, "text is required")
	}

	id, err := a.ingestBus.EnqueueText(ctx, req.Text)
	if err != nil {
		return errs.Newf(errs.Internal, "enqueue text: %s", err)
	}

	return ingestResponse{RawInputID: id.String()}
}
```

### rawinputapp

- [ ] **Step 3: Update rawinputapp/rawinputapp.go — async reprocess**

Replace the `reprocess` handler body:

```go
func (a *app) reprocess(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "raw_input_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	ri, err := a.rawInputBus.ResetForReprocess(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.Newf(errs.Internal, "reset for reprocess: %s", err)
	}

	return toAppRawInput(ri)
}
```

Remove the `a.ingestBus` field from the `app` struct since the reprocess handler no longer needs it:

```go
type app struct {
	rawInputBus *rawinputbus.Business
}
```

Update `route.go` to remove the ingestbus instantiation code and simplify the handler construction:

```go
func (Routes) Add(a *web.App, cfg mux.Config) {
	riStore := rawinputdb.NewStore(cfg.Log, cfg.DB)
	riBus := rawinputbus.NewBusiness(cfg.Log, riStore)

	hdl := &app{rawInputBus: riBus}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/raw-inputs", hdl.queryAll, authen)
	a.Handle(http.MethodGet, "/api/v1/raw-inputs/{raw_input_id}", hdl.queryByID, authen)
	a.Handle(http.MethodPost, "/api/v1/raw-inputs/{raw_input_id}/reprocess", hdl.reprocess, authen)
}
```

Remove all the ingestbus + dependency imports from route.go (emailbus, emaildb, contextbus, contextdb, ingestbus, extractor, notebus, notedb, tagbus, tagdb, taskbus, taskdb).

- [ ] **Step 4: Build to verify**

```bash
go build ./app/domain/voiceingestapp/... ./app/domain/rawinputapp/...
```

Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add app/domain/voiceingestapp/voiceingestapp.go app/domain/voiceingestapp/model.go \
        app/domain/rawinputapp/rawinputapp.go app/domain/rawinputapp/route.go
git commit -m "feat: make voice ingest and reprocess handlers async (return immediately)"
```

---

## Task 7: Wire Worker in main.go

**Files:**
- Modify: `api/services/planner/main.go`

The worker needs a full `ingestbus.Business`. Currently this is only created inside the SMTP conditional block. We need to create it unconditionally so the worker can use it.

- [ ] **Step 1: Move ingestbus creation out of the SMTP conditional**

In `main.go`, find the `// SMTP Server (Email Ingestion)` section. The SMTP block creates `emStore`, `emBus`, `noteStore`, `noteBus`, `tagStore`, `tgBus`, `ext`, and `igBus`. Move these above the SMTP block so they're always created.

After the existing bus declarations (around line 167 where `riBus` is created), add:

```go
// Ingest bus dependencies (used by both SMTP server and background worker)
emStore := emaildb.NewStore(log, db)
emBus := emailbus.NewBusiness(log, emStore)

noteStore := notedb.NewStore(log, db)
noteBus := notebus.NewBusiness(log, noteStore)

tagStore := tagdb.New(log, db)
tgBus := tagbus.NewBusiness(log, tagStore)

ext := extractor.NewClaudeCodeExtractor(cli)
igBus := ingestbus.NewBusiness(log, riBus, emBus, taskBus, ctxBus, clarBus, evtBus, ext, noteBus, tgBus)
```

Then in the SMTP conditional, remove those same lines (they're now declared above) and just use `igBus` directly:

```go
if cfg.SMTP.Enabled {
	log.Info(ctx, "startup", "status", "initializing smtp server")
	smtpSrv = smtpbus.NewServer(log, igBus, smtpbus.Config{
		Addr:   cfg.SMTP.Addr,
		Domain: cfg.SMTP.Domain,
	})
}
```

Add the required imports: `emaildb`, `emailbus`, `notedb`, `notebus`, `tagdb`, `tagbus`, `extractor`, `ingestbus` — most are already imported in the SMTP block, they just need to stay at the top-level imports.

- [ ] **Step 2: Add the worker to the Background Jobs section**

In the `// Background Jobs` section (after the existing goroutines), add:

```go
// Ingest worker: processes pending raw_inputs and retries failed ones
go func() {
	ingestWorker := worker.NewIngestWorker(log, riBus, igBus)
	ingestWorker.Run(jobCtx)
}()
```

Add import for `"github.com/casebrophy/planner/business/sdk/worker"`.

- [ ] **Step 3: Build the full service**

```bash
go build ./api/services/planner/...
```

Expected: no errors.

- [ ] **Step 4: Run all tests**

```bash
make test
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add api/services/planner/main.go
git commit -m "feat: wire ingest worker in main.go, make igBus unconditionally available"
```

---

## Task 8: Backend Integration Tests

**Files:**
- Create: `business/domain/rawinputbus/stores/rawinputdb/rawinputdb_test.go`

These tests verify `QueryRetryable` and `ResetForReprocess` against a real Postgres instance using `business/sdk/dbtest`.

- [ ] **Step 1: Look at an existing store test for the dbtest pattern**

```bash
ls business/domain/taskbus/stores/taskdb/
```

Find the test file and read it to understand how `dbtest.NewDatabase` and test setup works. Copy the imports and setup pattern.

- [ ] **Step 2: Write the store tests**

Create `business/domain/rawinputbus/stores/rawinputdb/rawinputdb_test.go`:

```go
package rawinputdb_test

import (
	"context"
	"testing"
	"time"

	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/rawinputbus/stores/rawinputdb"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/types/rawinputsource"
	"github.com/casebrophy/planner/business/types/rawinputstatus"
)

func TestQueryRetryable(t *testing.T) {
	db := dbtest.NewDatabase(t, "TestQueryRetryable")
	defer db.Teardown()

	store := rawinputdb.NewStore(db.Log, db.DB)
	ctx := context.Background()

	// Create a pending item (no next_retry_at) — should be retryable
	ri1 := rawinputbus.RawInput{
		ID:         dbtest.NewUUID(),
		SourceType: rawinputsource.Voice,
		Status:     rawinputstatus.Pending,
		RawContent: "test item 1",
		MaxRetries: 5,
		CreatedAt:  time.Now(),
	}
	if err := store.Create(ctx, ri1); err != nil {
		t.Fatalf("create ri1: %v", err)
	}

	// Create a pending item with future next_retry_at — should NOT be retryable yet
	future := time.Now().Add(10 * time.Minute)
	ri2 := rawinputbus.RawInput{
		ID:          dbtest.NewUUID(),
		SourceType:  rawinputsource.Voice,
		Status:      rawinputstatus.Pending,
		RawContent:  "test item 2",
		RetryCount:  1,
		NextRetryAt: &future,
		MaxRetries:  5,
		CreatedAt:   time.Now(),
	}
	if err := store.Create(ctx, ri2); err != nil {
		t.Fatalf("create ri2: %v", err)
	}

	items, err := store.QueryRetryable(ctx, 10)
	if err != nil {
		t.Fatalf("QueryRetryable: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("want 1 retryable item, got %d", len(items))
	}
	if items[0].ID != ri1.ID {
		t.Errorf("want ri1.ID=%s, got %s", ri1.ID, items[0].ID)
	}
}

func TestResetForReprocess(t *testing.T) {
	db := dbtest.NewDatabase(t, "TestResetForReprocess")
	defer db.Teardown()

	store := rawinputdb.NewStore(db.Log, db.DB)
	ctx := context.Background()

	// Create a failed item with retry state
	future := time.Now().Add(5 * time.Minute)
	errMsg := "pipeline failed"
	ri := rawinputbus.RawInput{
		ID:          dbtest.NewUUID(),
		SourceType:  rawinputsource.Voice,
		Status:      rawinputstatus.Failed,
		RawContent:  "voice capture",
		RetryCount:  3,
		NextRetryAt: &future,
		Error:       &errMsg,
		MaxRetries:  5,
		CreatedAt:   time.Now(),
	}
	if err := store.Create(ctx, ri); err != nil {
		t.Fatalf("create: %v", err)
	}

	reset, err := store.ResetForReprocess(ctx, ri.ID)
	if err != nil {
		t.Fatalf("ResetForReprocess: %v", err)
	}

	if reset.Status != rawinputstatus.Pending {
		t.Errorf("want status=pending, got %s", reset.Status)
	}
	if reset.RetryCount != 0 {
		t.Errorf("want retry_count=0, got %d", reset.RetryCount)
	}
	if reset.NextRetryAt != nil {
		t.Errorf("want next_retry_at=nil, got %v", reset.NextRetryAt)
	}
	if reset.Error != nil {
		t.Errorf("want error=nil, got %v", reset.Error)
	}
}
```

- [ ] **Step 3: Check what `dbtest.NewUUID` looks like (adjust if name differs)**

```bash
grep -r "func New" business/sdk/dbtest/ --include="*.go" | head -10
```

If there's no `NewUUID` helper, use `uuid.New()` directly from `"github.com/google/uuid"`.

- [ ] **Step 4: Run the tests**

```bash
go test ./business/domain/rawinputbus/stores/rawinputdb/... -count=1 -v
```

Expected: TestQueryRetryable and TestResetForReprocess PASS.

- [ ] **Step 5: Commit**

```bash
git add business/domain/rawinputbus/stores/rawinputdb/rawinputdb_test.go
git commit -m "test: add store integration tests for QueryRetryable and ResetForReprocess"
```

---

## Task 9: Frontend — Types + Service

**Files:**
- Modify: `api/services/frontend/web/src/types/rawinput.ts`
- Modify: `api/services/frontend/web/src/services/rawinputService.ts`

- [ ] **Step 1: Update types/rawinput.ts**

Replace the file:

```typescript
export interface RawInput {
  id: string
  sourceType: string
  status: string
  rawContent: string
  processedAt?: string
  error?: string
  retryCount: number
  nextRetryAt?: string
  maxRetries: number
  createdAt: string
}
```

- [ ] **Step 2: Update rawinputService.ts**

Replace the file with a custom service (keeping the CRUD base but adding `list`, `reprocess`, `countNonProcessed`):

```typescript
import { request } from './client'
import { createCRUDService } from './createCRUDService'
import type { RawInput } from '@/types/rawinput'
import type { QueryResult } from '@/types/query'

interface RawInputFilter {
  status?: string
  sourceType?: string
}

interface NewRawInput {
  sourceType: string
  rawContent: string
}

type UpdateRawInput = Record<string, never>

// Base CRUD (list, getById, create, update, delete)
const crud = createCRUDService<RawInput, NewRawInput, UpdateRawInput, RawInputFilter>({
  basePath: '/api/v1/raw-inputs',
  mapFilter: (f) => ({
    status: f.status,
    source_type: f.sourceType,
  }),
})

export const rawinputService = {
  ...crud,

  async list(params: {
    page?: number
    rows?: number
    status?: string
    sourceType?: string
    orderBy?: string
  } = {}): Promise<QueryResult<RawInput>> {
    const queryParams: Record<string, string> = {}
    if (params.page) queryParams.page = String(params.page)
    if (params.rows) queryParams.rows_per_page = String(params.rows)
    if (params.status) queryParams.status = params.status
    if (params.sourceType) queryParams.source_type = params.sourceType
    if (params.orderBy) queryParams.orderBy = params.orderBy

    const res = await request<{ items: RawInput[]; total: number; page: number; rowsPerPage: number }>(
      '/api/v1/raw-inputs',
      { params: queryParams },
    )
    return { items: res.items, total: res.total, page: res.page, rowsPerPage: res.rowsPerPage }
  },

  async reprocess(id: string): Promise<RawInput> {
    return request<RawInput>(`/api/v1/raw-inputs/${id}/reprocess`, { method: 'POST' })
  },

  async countNonProcessed(): Promise<number> {
    const res = await rawinputService.list({ status: 'pending', rows: 1 })
    // We just need a rough count for a badge — the total from the paginated response works
    const pending = res.total
    const failedRes = await rawinputService.list({ status: 'failed', rows: 1 })
    return pending + failedRes.total
  },
}
```

- [ ] **Step 3: Build to verify**

```bash
cd api/services/frontend && npm run build 2>&1 | tail -5
```

Expected: no TypeScript errors.

- [ ] **Step 4: Commit**

```bash
git add api/services/frontend/web/src/types/rawinput.ts api/services/frontend/web/src/services/rawinputService.ts
git commit -m "feat: extend rawinput types and service for retry fields and reprocess"
```

---

## Task 10: Frontend — Pinia Store

**Files:**
- Create: `api/services/frontend/web/src/stores/rawinputStore.ts`

- [ ] **Step 1: Write the failing test**

Create `api/services/frontend/web/src/__tests__/stores/rawinputStore.test.ts`:

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useRawInputStore } from '@/stores/rawinputStore'
import { rawinputService } from '@/services/rawinputService'

vi.mock('@/services/rawinputService', () => ({
  rawinputService: {
    list: vi.fn(),
    reprocess: vi.fn(),
  },
}))

const makeItem = (overrides = {}) => ({
  id: 'id-1',
  sourceType: 'voice',
  status: 'failed',
  rawContent: 'test',
  retryCount: 3,
  maxRetries: 5,
  createdAt: '2026-04-03T00:00:00Z',
  ...overrides,
})

describe('useRawInputStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('fetchList populates items', async () => {
    vi.mocked(rawinputService.list).mockResolvedValue({
      items: [makeItem()],
      total: 1,
      page: 1,
      rowsPerPage: 25,
    })

    const store = useRawInputStore()
    await store.fetchList()

    expect(store.items).toHaveLength(1)
    expect(store.total).toBe(1)
  })

  it('reprocess calls service and refreshes list', async () => {
    const updated = makeItem({ status: 'pending', retryCount: 0 })
    vi.mocked(rawinputService.reprocess).mockResolvedValue(updated)
    vi.mocked(rawinputService.list).mockResolvedValue({
      items: [updated],
      total: 1,
      page: 1,
      rowsPerPage: 25,
    })

    const store = useRawInputStore()
    await store.reprocess('id-1')

    expect(rawinputService.reprocess).toHaveBeenCalledWith('id-1')
    expect(rawinputService.list).toHaveBeenCalled()
  })

  it('setStatusFilter resets page and re-fetches', async () => {
    vi.mocked(rawinputService.list).mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      rowsPerPage: 25,
    })

    const store = useRawInputStore()
    store.page = 3
    await store.setStatusFilter('failed')

    expect(store.statusFilter).toBe('failed')
    expect(store.page).toBe(1)
    expect(rawinputService.list).toHaveBeenCalledWith(
      expect.objectContaining({ status: 'failed', page: 1 }),
    )
  })
})
```

- [ ] **Step 2: Run to confirm it fails**

```bash
cd api/services/frontend && npx vitest run src/__tests__/stores/rawinputStore.test.ts 2>&1 | tail -5
```

Expected: error — `@/stores/rawinputStore` not found.

- [ ] **Step 3: Create the store**

Create `api/services/frontend/web/src/stores/rawinputStore.ts`:

```typescript
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useToastStore } from './toastStore'
import { rawinputService } from '@/services/rawinputService'
import type { RawInput } from '@/types/rawinput'

export const useRawInputStore = defineStore('rawinput', () => {
  const items = ref<RawInput[]>([])
  const total = ref(0)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const page = ref(1)
  const rowsPerPage = ref(25)
  const statusFilter = ref<string | undefined>(undefined)

  const toast = useToastStore()

  const totalPages = computed(() => Math.ceil(total.value / rowsPerPage.value))
  const failedCount = computed(() => items.value.filter((i) => i.status === 'failed').length)

  async function fetchList(force = false) {
    if (loading.value && !force) return
    loading.value = true
    error.value = null
    try {
      const result = await rawinputService.list({
        page: page.value,
        rows: rowsPerPage.value,
        status: statusFilter.value,
        orderBy: 'created_at',
      })
      items.value = result.items
      total.value = result.total
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch raw inputs'
      toast.error(error.value)
    } finally {
      loading.value = false
    }
  }

  async function reprocess(id: string) {
    try {
      await rawinputService.reprocess(id)
      toast.success('Requeued for processing')
      await fetchList(true)
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to reprocess'
      toast.error(msg)
      throw e
    }
  }

  async function setStatusFilter(status: string | undefined) {
    statusFilter.value = status
    page.value = 1
    await fetchList(true)
  }

  function setPage(newPage: number) {
    page.value = newPage
    fetchList(true)
  }

  return {
    items,
    total,
    loading,
    error,
    page,
    rowsPerPage,
    statusFilter,
    totalPages,
    failedCount,
    fetchList,
    reprocess,
    setStatusFilter,
    setPage,
  }
})
```

- [ ] **Step 4: Run tests**

```bash
cd api/services/frontend && npx vitest run src/__tests__/stores/rawinputStore.test.ts
```

Expected: all 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add api/services/frontend/web/src/stores/rawinputStore.ts api/services/frontend/web/src/__tests__/stores/rawinputStore.test.ts
git commit -m "feat: add rawinputStore with fetchList, reprocess, statusFilter"
```

---

## Task 11: Frontend — Queue View + Router

**Files:**
- Create: `api/services/frontend/web/src/views/RawInputQueueView.vue`
- Modify: `api/services/frontend/web/src/router/index.ts`

- [ ] **Step 1: Create RawInputQueueView.vue**

Create `api/services/frontend/web/src/views/RawInputQueueView.vue`:

```vue
<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { useRawInputStore } from '@/stores/rawinputStore'

const store = useRawInputStore()

let pollInterval: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  store.fetchList()
  // Poll every 15 seconds while view is active
  pollInterval = setInterval(() => store.fetchList(), 15_000)
})

onUnmounted(() => {
  if (pollInterval) clearInterval(pollInterval)
})

const statusOptions = [
  { label: 'All', value: undefined },
  { label: 'Pending', value: 'pending' },
  { label: 'Processing', value: 'processing' },
  { label: 'Processed', value: 'processed' },
  { label: 'Failed', value: 'failed' },
]

function statusBadgeClass(status: string): string {
  const classes: Record<string, string> = {
    pending: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
    processing: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
    processed: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
    failed: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
  }
  return classes[status] ?? 'bg-gray-100 text-gray-800'
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString()
}

function isRetryScheduled(item: { status: string; nextRetryAt?: string }): boolean {
  return item.status === 'pending' && !!item.nextRetryAt && new Date(item.nextRetryAt) > new Date()
}
</script>

<template>
  <div class="p-6 space-y-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-xl font-semibold text-gray-900 dark:text-white">Ingest Queue</h1>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-0.5">
          {{ store.total }} total · {{ store.failedCount }} failed
        </p>
      </div>
      <button
        class="text-sm px-3 py-1.5 rounded-md bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-200"
        @click="store.fetchList(true)"
      >
        Refresh
      </button>
    </div>

    <!-- Status Filter -->
    <div class="flex gap-1.5 flex-wrap">
      <button
        v-for="opt in statusOptions"
        :key="String(opt.value)"
        :class="[
          'px-3 py-1 text-sm rounded-full transition-colors',
          store.statusFilter === opt.value
            ? 'bg-indigo-600 text-white'
            : 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-200 hover:bg-gray-200 dark:hover:bg-gray-600',
        ]"
        @click="store.setStatusFilter(opt.value)"
      >
        {{ opt.label }}
      </button>
    </div>

    <!-- Loading -->
    <div v-if="store.loading && store.items.length === 0" class="text-center py-12 text-gray-400">
      Loading…
    </div>

    <!-- Empty State -->
    <div
      v-else-if="!store.loading && store.items.length === 0"
      class="text-center py-12 text-gray-400 dark:text-gray-500"
    >
      No raw inputs found.
    </div>

    <!-- Table -->
    <div v-else class="overflow-x-auto rounded-lg border border-gray-200 dark:border-gray-700">
      <table class="w-full text-sm">
        <thead>
          <tr class="bg-gray-50 dark:bg-gray-800 text-left text-gray-600 dark:text-gray-300">
            <th class="px-4 py-3 font-medium">Source</th>
            <th class="px-4 py-3 font-medium">Status</th>
            <th class="px-4 py-3 font-medium">Retries</th>
            <th class="px-4 py-3 font-medium">Created</th>
            <th class="px-4 py-3 font-medium">Error</th>
            <th class="px-4 py-3 font-medium"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 dark:divide-gray-700">
          <tr
            v-for="item in store.items"
            :key="item.id"
            class="bg-white dark:bg-gray-900 hover:bg-gray-50 dark:hover:bg-gray-800"
          >
            <td class="px-4 py-3 font-mono text-xs text-gray-500">{{ item.sourceType }}</td>
            <td class="px-4 py-3">
              <span
                :class="[
                  'inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium',
                  statusBadgeClass(item.status),
                ]"
              >
                {{ item.status }}
              </span>
              <span
                v-if="isRetryScheduled(item)"
                class="ml-1 text-xs text-gray-400"
              >
                (retry at {{ formatDate(item.nextRetryAt!) }})
              </span>
            </td>
            <td class="px-4 py-3 text-gray-500">
              {{ item.retryCount }} / {{ item.maxRetries }}
            </td>
            <td class="px-4 py-3 text-gray-500">{{ formatDate(item.createdAt) }}</td>
            <td class="px-4 py-3 max-w-xs truncate text-red-500 text-xs">
              {{ item.error ?? '—' }}
            </td>
            <td class="px-4 py-3">
              <button
                v-if="item.status === 'failed' || item.status === 'pending'"
                class="text-xs px-2.5 py-1 rounded bg-indigo-600 text-white hover:bg-indigo-700 disabled:opacity-50"
                :disabled="store.loading"
                @click="store.reprocess(item.id)"
              >
                Reprocess
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Pagination -->
    <div
      v-if="store.totalPages > 1"
      class="flex items-center justify-between text-sm text-gray-500"
    >
      <span>Page {{ store.page }} of {{ store.totalPages }}</span>
      <div class="flex gap-2">
        <button
          :disabled="store.page <= 1"
          class="px-3 py-1 rounded bg-gray-100 dark:bg-gray-700 disabled:opacity-40"
          @click="store.setPage(store.page - 1)"
        >
          Prev
        </button>
        <button
          :disabled="store.page >= store.totalPages"
          class="px-3 py-1 rounded bg-gray-100 dark:bg-gray-700 disabled:opacity-40"
          @click="store.setPage(store.page + 1)"
        >
          Next
        </button>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 2: Register the route in router/index.ts**

Add to the import block:
```typescript
const RawInputQueueView = () => import('@/views/RawInputQueueView.vue')
```

Add to the routes array (after the `/transactions` route):
```typescript
{ path: '/ingest-queue', name: 'ingest-queue', component: RawInputQueueView },
```

- [ ] **Step 3: Build to verify**

```bash
cd api/services/frontend && npm run build 2>&1 | tail -5
```

Expected: build succeeds with no TypeScript errors.

- [ ] **Step 4: Commit**

```bash
git add api/services/frontend/web/src/views/RawInputQueueView.vue \
        api/services/frontend/web/src/router/index.ts
git commit -m "feat: add RawInputQueueView for ingest monitoring and manual reprocess"
```

---

## Self-Review Checklist

**Spec coverage:**
- [x] Ingestion returns immediately (Tasks 4, 6 — EnqueueText/EnqueueEmail + handler update)
- [x] Background worker processes pending items (Task 5 — IngestWorker)
- [x] Failed items are retried with backoff (Task 2 — MarkForRetry with ComputeBackoff, Task 5 — worker logic)
- [x] Items reaching max retries are marked terminal failed (Task 5 — worker calls MarkFailed)
- [x] Frontend queue view to see status (Task 11 — RawInputQueueView)
- [x] Manual reprocess from frontend (Task 6 — reprocess handler; Task 10 — store.reprocess; Task 11 — Reprocess button)
- [x] Both email and voice pipelines made async (Task 4 — ProcessRawInputByID handles both source types; Task 6 — voiceingestapp updated; SMTP still uses ProcessEmail synchronously)

**Placeholder scan:** No TBD/TODO/placeholder text in code steps — all code is complete.

**Type consistency check:**
- `ComputeBackoff` exported in rawinputbus.go, tested directly in rawinputbus_test.go ✓
- `ProcessRawInputByID` defined in ingestbus.go, used by worker's `RawInputProcessor` interface ✓
- `RawInputQueuer` interface in worker matches `rawinputbus.Business` methods ✓
- `RawInputProcessor` interface in worker matches `ingestbus.Business.ProcessRawInputByID` ✓
- Frontend `RawInput.retryCount` matches Go JSON `retryCount` in app model (Task 9 adds to types, Task 3 model.go step reminds to update app/domain/rawinputapp/model.go)

**Gap found:** The app-layer model in `app/domain/rawinputapp/model.go` must also expose the new fields. Add this step:

### Addendum: Update app/domain/rawinputapp/model.go

After Task 3, run this additional fix:

- [ ] **Add retry fields to app DTO**

In `app/domain/rawinputapp/model.go`, update `RawInput` and `toAppRawInput`:

```go
type RawInput struct {
	ID          string  `json:"id"`
	SourceType  string  `json:"sourceType"`
	Status      string  `json:"status"`
	RawContent  string  `json:"rawContent"`
	ProcessedAt *string `json:"processedAt,omitempty"`
	Error       *string `json:"error,omitempty"`
	RetryCount  int     `json:"retryCount"`
	NextRetryAt *string `json:"nextRetryAt,omitempty"`
	MaxRetries  int     `json:"maxRetries"`
	CreatedAt   string  `json:"createdAt"`
}

func toAppRawInput(ri rawinputbus.RawInput) RawInput {
	a := RawInput{
		ID:         ri.ID.String(),
		SourceType: ri.SourceType.String(),
		Status:     ri.Status.String(),
		RawContent: ri.RawContent,
		Error:      ri.Error,
		RetryCount: ri.RetryCount,
		MaxRetries: ri.MaxRetries,
		CreatedAt:  ri.CreatedAt.Format(time.RFC3339),
	}
	if ri.ProcessedAt != nil {
		s := ri.ProcessedAt.Format(time.RFC3339)
		a.ProcessedAt = &s
	}
	if ri.NextRetryAt != nil {
		s := ri.NextRetryAt.Format(time.RFC3339)
		a.NextRetryAt = &s
	}
	return a
}
```

Commit alongside Task 3 or as a follow-up after Task 3.

---

## Execution Order

Tasks must be done in this order (each depends on the previous):

1. Migration (changes DB schema)
2. rawinputbus model + Storer (compiles against new schema)
3. rawinputdb store + app model DTO (implements Storer)
4. ingestbus enqueue methods (uses rawinputbus)
5. Worker (uses rawinputbus + ingestbus interfaces)
6. HTTP handlers (calls EnqueueText, ResetForReprocess)
7. main.go wiring (uses worker + igBus)
8. Backend tests (validates store behavior)
9. Frontend types + service (API surface is stable by now)
10. Frontend store
11. Frontend view + router
