package rawinputdb_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/rawinputbus/stores/rawinputdb"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/types/rawinputsource"
	"github.com/casebrophy/planner/business/types/rawinputstatus"
)

func TestQueryRetryable(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "TestQueryRetryable")
	store := rawinputdb.NewStore(db.Log, db.DB)
	ctx := context.Background()

	// ri1: status=pending, next_retry_at=nil → should be returned
	ri1 := rawinputbus.RawInput{
		ID:         uuid.New(),
		SourceType: rawinputsource.Voice,
		Status:     rawinputstatus.Pending,
		RawContent: "retryable content",
		MaxRetries: 5,
		CreatedAt:  time.Now(),
	}
	if err := store.Create(ctx, ri1); err != nil {
		t.Fatalf("creating ri1: %v", err)
	}

	// ri2: status=pending, retry_count=1, next_retry_at=10 minutes in the future → should NOT be returned
	futureTime := time.Now().Add(10 * time.Minute)
	ri2 := rawinputbus.RawInput{
		ID:          uuid.New(),
		SourceType:  rawinputsource.Voice,
		Status:      rawinputstatus.Pending,
		RawContent:  "future retry content",
		RetryCount:  1,
		NextRetryAt: &futureTime,
		MaxRetries:  5,
		CreatedAt:   time.Now(),
	}
	if err := store.Create(ctx, ri2); err != nil {
		t.Fatalf("creating ri2: %v", err)
	}

	results, err := store.QueryRetryable(ctx, 10)
	if err != nil {
		t.Fatalf("QueryRetryable: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 retryable item, got %d", len(results))
	}

	if results[0].ID != ri1.ID {
		t.Errorf("expected ri1 (ID=%s) to be returned, got ID=%s", ri1.ID, results[0].ID)
	}
}

func TestResetForReprocess(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "TestResetForReprocess")
	store := rawinputdb.NewStore(db.Log, db.DB)
	ctx := context.Background()

	errMsg := "pipeline failed"
	futureTime := time.Now().Add(5 * time.Minute)

	ri := rawinputbus.RawInput{
		ID:          uuid.New(),
		SourceType:  rawinputsource.Voice,
		Status:      rawinputstatus.Failed,
		RawContent:  "failed content",
		RetryCount:  3,
		NextRetryAt: &futureTime,
		Error:       &errMsg,
		MaxRetries:  5,
		CreatedAt:   time.Now(),
	}
	if err := store.Create(ctx, ri); err != nil {
		t.Fatalf("creating raw input: %v", err)
	}

	updated, err := store.ResetForReprocess(ctx, ri.ID)
	if err != nil {
		t.Fatalf("ResetForReprocess: %v", err)
	}

	if updated.Status != rawinputstatus.Pending {
		t.Errorf("expected status=pending, got %s", updated.Status)
	}
	if updated.RetryCount != 0 {
		t.Errorf("expected retry_count=0, got %d", updated.RetryCount)
	}
	if updated.NextRetryAt != nil {
		t.Errorf("expected next_retry_at=nil, got %v", updated.NextRetryAt)
	}
	if updated.Error != nil {
		t.Errorf("expected error=nil, got %v", *updated.Error)
	}
}

func TestUserCorrectionPersists(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "TestUserCorrectionPersists")
	store := rawinputdb.NewStore(db.Log, db.DB)
	ctx := context.Background()

	// Create raw input with no user correction
	correction := "treat leather on shoes"
	ri := rawinputbus.RawInput{
		ID:         uuid.New(),
		SourceType: rawinputsource.Voice,
		Status:     rawinputstatus.Pending,
		RawContent: "original content with error",
		MaxRetries: 5,
		CreatedAt:  time.Now(),
	}
	if err := store.Create(ctx, ri); err != nil {
		t.Fatalf("creating raw input: %v", err)
	}

	// Update with user correction
	ri.UserCorrection = &correction
	if err := store.Update(ctx, ri); err != nil {
		t.Fatalf("updating raw input: %v", err)
	}

	// Query back and verify correction persists
	retrieved, err := store.QueryByID(ctx, ri.ID)
	if err != nil {
		t.Fatalf("querying raw input: %v", err)
	}

	if retrieved.UserCorrection == nil {
		t.Error("expected UserCorrection to be set, got nil")
	}
	if retrieved.UserCorrection != nil && *retrieved.UserCorrection != correction {
		t.Errorf("expected UserCorrection=%q, got %q", correction, *retrieved.UserCorrection)
	}

	// Test clearing correction
	ri.UserCorrection = nil
	if err := store.Update(ctx, ri); err != nil {
		t.Fatalf("clearing user correction: %v", err)
	}

	retrieved, err = store.QueryByID(ctx, ri.ID)
	if err != nil {
		t.Fatalf("querying raw input after clear: %v", err)
	}

	if retrieved.UserCorrection != nil {
		t.Errorf("expected UserCorrection to be nil after clear, got %q", *retrieved.UserCorrection)
	}
}
