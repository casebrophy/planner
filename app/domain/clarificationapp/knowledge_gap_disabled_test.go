package clarificationapp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/clarificationstatus"
	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/google/uuid"
)

// TestKnowledgeGapDisabled verifies that:
// 1. Existing knowledge_gap rows remain readable (not deleted)
// 2. Queue API filters them out
// 3. Count pending excludes them
func TestKnowledgeGapDisabled(t *testing.T) {
	test := apitest.New(t, "knowledge_gap_disabled")

	ctx := context.Background()
	subjectID := uuid.New()

	// Insert a manual knowledge_gap clarification directly into the database
	// to simulate existing data. This verifies rows are not deleted.
	existingKnowledgeGapID := uuid.New()
	_, err := test.DB.DB.ExecContext(ctx, `
		INSERT INTO clarifications
		(id, kind, status, subject_type, subject_id, subject_description, gap_category,
		 question, claude_guess, reasoning, answer_options, priority_score, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW())
	`, existingKnowledgeGapID, "knowledge_gap", "pending", "note", subjectID,
		"Test: manual knowledge gap", "missing_context", "What is the intent?",
		nil, nil, nil, 0.95)
	if err != nil {
		t.Fatalf("insert existing knowledge_gap: %v", err)
	}

	// Insert a non-knowledge_gap clarification for comparison
	otherClarificationID := uuid.New()
	_, err = test.DB.DB.ExecContext(ctx, `
		INSERT INTO clarifications
		(id, kind, status, subject_type, subject_id, subject_description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`, otherClarificationID, "context_assignment", "pending", "task", uuid.New(),
		"Test: context assignment")
	if err != nil {
		t.Fatalf("insert other clarification: %v", err)
	}

	// Test 1: Verify existing knowledge_gap row is still readable by ID
	t.Run("existing_knowledge_gap_readable_by_id", func(t *testing.T) {
		req := httptest.NewRequest("GET",
			fmt.Sprintf("/clarifications/%s", existingKnowledgeGapID),
			nil)
		req.Header.Set("X-API-Key", apitest.TestAPIKey)

		w := httptest.NewRecorder()
		test.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var result map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}

		if kind, ok := result["kind"].(string); !ok || kind != "knowledge_gap" {
			t.Errorf("expected kind=knowledge_gap, got %v", result["kind"])
		}
	})

	// Test 2: Verify queryQueue filters out knowledge_gap
	t.Run("queue_excludes_knowledge_gap", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/clarifications", nil)
		req.Header.Set("X-API-Key", apitest.TestAPIKey)

		w := httptest.NewRecorder()
		test.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var result map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}

		items, ok := result["items"].([]interface{})
		if !ok {
			t.Fatalf("expected items array, got %T", result["items"])
		}

		// Verify no knowledge_gap items in queue
		for _, item := range items {
			itemMap := item.(map[string]interface{})
			if kind, ok := itemMap["kind"].(string); ok && kind == "knowledge_gap" {
				t.Errorf("queue should not include knowledge_gap items")
			}
		}

		// Verify the other clarification IS in queue
		foundOther := false
		for _, item := range items {
			itemMap := item.(map[string]interface{})
			if id, ok := itemMap["id"].(string); ok && id == otherClarificationID.String() {
				foundOther = true
				break
			}
		}
		if !foundOther {
			t.Errorf("queue should include context_assignment clarification")
		}
	})

	// Test 3: Verify countPending excludes knowledge_gap
	t.Run("count_pending_excludes_knowledge_gap", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/clarifications/count", nil)
		req.Header.Set("X-API-Key", apitest.TestAPIKey)

		w := httptest.NewRecorder()
		test.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var result map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}

		count, ok := result["count"].(float64)
		if !ok {
			t.Fatalf("expected count number, got %T", result["count"])
		}

		// Should count only the context_assignment (1), not the knowledge_gap
		if count != 1 {
			t.Errorf("expected count=1, got %v", count)
		}
	})

	// Test 4: Verify explicit filter for knowledge_gap still works (direct query)
	t.Run("explicit_kind_filter_still_works", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/clarifications?kind=knowledge_gap", nil)
		req.Header.Set("X-API-Key", apitest.TestAPIKey)

		w := httptest.NewRecorder()
		test.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var result map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}

		items, ok := result["items"].([]interface{})
		if !ok {
			t.Fatalf("expected items array, got %T", result["items"])
		}

		// Verify explicit filter returns knowledge_gap items
		foundKnowledgeGap := false
		for _, item := range items {
			itemMap := item.(map[string]interface{})
			if kind, ok := itemMap["kind"].(string); ok && kind == "knowledge_gap" {
				foundKnowledgeGap = true
				break
			}
		}
		if !foundKnowledgeGap {
			t.Errorf("explicit kind=knowledge_gap filter should return knowledge_gap items")
		}
	})
}

// TestExcludeKindFilter verifies the ExcludeKind filter logic in QueryFilter.
func TestExcludeKindFilter(t *testing.T) {
	test := apitest.New(t, "exclude_kind_filter")

	ctx := context.Background()

	// Create test clarifications of different kinds
	kg := uuid.New()
	ca := uuid.New()
	sa := uuid.New()

	kinds := []struct {
		id   uuid.UUID
		kind string
	}{
		{kg, "knowledge_gap"},
		{ca, "context_assignment"},
		{sa, "stale_task"},
	}

	for _, k := range kinds {
		_, err := test.DB.DB.ExecContext(ctx, `
			INSERT INTO clarifications
			(id, kind, status, subject_type, subject_id, subject_description, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		`, k.id, k.kind, "pending", "note", uuid.New(), "Test")
		if err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	// Query without exclusion
	pg := page.New(1, 100)
	orderBy := order.NewBy("priority_score", order.DESC)
	all, err := test.DB.BusDomain.Clarification.Query(ctx,
		clarificationbus.QueryFilter{},
		orderBy,
		pg)
	if err != nil {
		t.Fatalf("query all: %v", err)
	}

	if len(all) != 3 {
		t.Errorf("expected 3 clarifications, got %d", len(all))
	}

	// Query with ExcludeKind
	pending := clarificationstatus.Pending
	exclude := clarificationkind.KnowledgeGap
	filtered, err := test.DB.BusDomain.Clarification.Query(ctx,
		clarificationbus.QueryFilter{
			Status:      &pending,
			ExcludeKind: &exclude,
		},
		orderBy,
		pg)
	if err != nil {
		t.Fatalf("query filtered: %v", err)
	}

	if len(filtered) != 2 {
		t.Errorf("expected 2 clarifications (excluding knowledge_gap), got %d", len(filtered))
	}

	// Verify no knowledge_gap in filtered results
	for _, item := range filtered {
		if item.Kind == clarificationkind.KnowledgeGap {
			t.Errorf("filtered results should not contain knowledge_gap")
		}
	}
}
