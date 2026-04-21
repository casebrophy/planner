package clarificationapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/casebrophy/planner/app/domain/clarificationapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/taskenergy"
	"github.com/casebrophy/planner/business/types/taskpriority"
	"github.com/casebrophy/planner/business/types/taskstatus"
)

// TestResolve_TypeAssignment_RecordsCorrectionAndClearsUnconfirmed verifies that resolving
// a TypeAssignment clarification with {actual_type: "task"} records a classification
// correction and clears the subject task's Unconfirmed flag.
func TestResolve_TypeAssignment_RecordsCorrectionAndClearsUnconfirmed(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "TestResolve_TypeAssignment_RecordsCorrectionAndClearsUnconfirmed")
	ctx := context.Background()
	db := test.DB

	// Create an unconfirmed task as the clarification subject.
	task, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title:       "maybe a task",
		Status:      taskstatus.Open,
		Priority:    taskpriority.Medium,
		Energy:      taskenergy.Medium,
		Unconfirmed: true,
	})
	if err != nil {
		t.Fatalf("create task: %s", err)
	}

	// Build TypeAssignmentOptions JSON.
	opts := clarificationbus.TypeAssignmentOptions{
		ClauseText:    "maybe a task",
		PredictedType: "note",
		Confidence:    0.42,
		Options:       []string{"task", "note", "event"},
	}
	optsBytes, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("marshal options: %s", err)
	}

	// Create the TypeAssignment clarification.
	clarItem, err := db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:               clarificationkind.TypeAssignment,
		SubjectType:        "task",
		SubjectID:          task.ID,
		SubjectDescription: "maybe a task",
		Question:           "What is this?",
		AnswerOptions:      optsBytes,
		PriorityScore:      0.5,
	})
	if err != nil {
		t.Fatalf("create clarification: %s", err)
	}

	// Resolve with {actual_type: "task"}.
	input := clarificationapp.ResolveInput{
		Answer: json.RawMessage(`{"actual_type": "task"}`),
	}
	body, _ := json.Marshal(input)

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/clarifications/%s/resolve", clarItem.ID), bytes.NewBuffer(body))
	r.Header.Set("X-API-Key", apitest.TestAPIKey)
	w := httptest.NewRecorder()
	test.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp clarificationapp.ClarificationItem
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %s", err)
	}
	if resp.Status != "resolved" {
		t.Errorf("expected status=resolved, got %s", resp.Status)
	}

	// Assert a classification_corrections row was written.
	corrs, err := db.BusDomain.ClassificationCorrection.QueryBySource(ctx, "clarification_answered", page.New(1, 100))
	if err != nil {
		t.Fatalf("query corrections: %s", err)
	}

	found := false
	for _, c := range corrs {
		if c.ClauseText == opts.ClauseText {
			found = true
			if c.PredictedType != opts.PredictedType {
				t.Errorf("correction PredictedType: got %q, want %q", c.PredictedType, opts.PredictedType)
			}
			if c.Confidence != opts.Confidence {
				t.Errorf("correction Confidence: got %v, want %v", c.Confidence, opts.Confidence)
			}
			if c.ActualType != "task" {
				t.Errorf("correction ActualType: got %q, want %q", c.ActualType, "task")
			}
			if c.Source != "clarification_answered" {
				t.Errorf("correction Source: got %q, want %q", c.Source, "clarification_answered")
			}
			break
		}
	}

	if !found {
		t.Fatalf("expected a classification correction row with ClauseText %q, not found", opts.ClauseText)
	}

	// Assert the task's Unconfirmed flag is now false.
	refreshed, err := db.BusDomain.Task.QueryByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("requery task: %s", err)
	}
	if refreshed.Unconfirmed {
		t.Errorf("expected task.Unconfirmed=false, got true")
	}
}
