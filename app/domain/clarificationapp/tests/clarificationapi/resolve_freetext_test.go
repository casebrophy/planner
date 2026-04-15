package clarificationapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/domain/clarificationapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/rawinputstatus"
)

// Test_FreeTextResolve_RawInput verifies that resolving a clarification with a free_text answer
// when subject_type=raw_input deletes unconfirmed entities, sets user_correction, and resets the
// raw input to pending for re-ingest. Confirmed entities must survive.
func Test_FreeTextResolve_RawInput(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "Test_FreeTextResolve_RawInput")
	ctx := context.Background()
	db := test.DB

	// --- Seed raw input ---
	ris, err := rawinputbus.TestSeedRawInputs(ctx, 1, db.BusDomain.RawInput)
	if err != nil {
		t.Fatalf("seed raw input: %s", err)
	}
	ri := ris[0]

	// Mark raw input as processed (simulates the normal ingest cycle completing).
	ri, err = db.BusDomain.RawInput.MarkProcessed(ctx, ri)
	if err != nil {
		t.Fatalf("mark processed: %s", err)
	}

	// --- Seed entities linked to the raw input ---

	// Unconfirmed task (should be deleted)
	unconfirmedTask, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title:       "bad transcription task",
		RawInputID:  &ri.ID,
		Unconfirmed: true,
	})
	if err != nil {
		t.Fatalf("create unconfirmed task: %s", err)
	}

	// Confirmed task (should survive)
	confirmedTask, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title:       "confirmed task (survives)",
		RawInputID:  &ri.ID,
		Unconfirmed: false,
	})
	if err != nil {
		t.Fatalf("create confirmed task: %s", err)
	}

	// Unconfirmed note (should be deleted)
	unconfirmedNote, err := db.BusDomain.Note.Create(ctx, notebus.NewNote{
		Content:     "bad note",
		Source:      "voice",
		RawInputID:  &ri.ID,
		Unconfirmed: true,
	})
	if err != nil {
		t.Fatalf("create unconfirmed note: %s", err)
	}

	// Unconfirmed event (should be deleted)
	now := time.Now()
	unconfirmedEvent, err := db.BusDomain.Event.Create(ctx, eventbus.NewEvent{
		Title:       "bad event",
		Description: "event from bad transcription",
		StartsAt:    now.Add(time.Hour),
		EndsAt:      now.Add(2 * time.Hour),
		RawInputID:  &ri.ID,
		Unconfirmed: true,
	})
	if err != nil {
		t.Fatalf("create unconfirmed event: %s", err)
	}

	// --- Seed clarification with subject_type=raw_input ---
	clarItem, err := db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:               clarificationkind.StaleTask,
		SubjectType:        "raw_input",
		SubjectID:          ri.ID,
		SubjectDescription: "test raw input",
		Question:           "Does this look right?",
		AnswerOptions:      json.RawMessage(`["yes","no"]`),
		PriorityScore:      0.5,
	})
	if err != nil {
		t.Fatalf("create clarification: %s", err)
	}

	// --- Resolve with free_text ---
	correction := "treat leather on shoes"
	input := clarificationapp.ResolveInput{
		Answer: json.RawMessage(fmt.Sprintf(`{"free_text": %q}`, correction)),
	}
	body, _ := json.Marshal(input)

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/clarifications/%s/resolve", clarItem.ID), bytes.NewBuffer(body))
	r.Header.Set("X-API-Key", apitest.TestAPIKey)
	w := httptest.NewRecorder()
	test.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify clarification is resolved.
	var resp clarificationapp.ClarificationItem
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %s", err)
	}
	if resp.Status != "resolved" {
		t.Errorf("expected status=resolved, got %s", resp.Status)
	}

	// Verify unconfirmed task deleted.
	_, err = db.BusDomain.Task.QueryByID(ctx, unconfirmedTask.ID)
	if err == nil {
		t.Error("expected unconfirmed task to be deleted, but it still exists")
	} else if !isNotFound(err) {
		t.Errorf("unexpected error querying unconfirmed task: %s", err)
	}

	// Verify confirmed task survives.
	_, err = db.BusDomain.Task.QueryByID(ctx, confirmedTask.ID)
	if err != nil {
		t.Errorf("confirmed task should survive cleanup, got error: %s", err)
	}

	// Verify unconfirmed note deleted.
	_, err = db.BusDomain.Note.QueryByID(ctx, unconfirmedNote.ID)
	if err == nil {
		t.Error("expected unconfirmed note to be deleted, but it still exists")
	} else if !isNotFound(err) {
		t.Errorf("unexpected error querying unconfirmed note: %s", err)
	}

	// Verify unconfirmed event deleted.
	_, err = db.BusDomain.Event.QueryByID(ctx, unconfirmedEvent.ID)
	if err == nil {
		t.Error("expected unconfirmed event to be deleted, but it still exists")
	} else if !isNotFound(err) {
		t.Errorf("unexpected error querying unconfirmed event: %s", err)
	}

	// Verify raw input user_correction set and status reset to pending.
	updatedRI, err := db.BusDomain.RawInput.QueryByID(ctx, ri.ID)
	if err != nil {
		t.Fatalf("query raw input: %s", err)
	}
	if updatedRI.UserCorrection == nil || *updatedRI.UserCorrection != correction {
		t.Errorf("expected user_correction=%q, got %v", correction, updatedRI.UserCorrection)
	}
	if updatedRI.Status != rawinputstatus.Pending {
		t.Errorf("expected raw input status=pending, got %s", updatedRI.Status)
	}
}

// Test_FreeTextResolve_NonRawInput verifies that resolving with free_text on a non-raw_input
// subject succeeds without crashing and marks the clarification resolved.
func Test_FreeTextResolve_NonRawInput(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "Test_FreeTextResolve_NonRawInput")
	ctx := context.Background()
	db := test.DB

	// Clarification with subject_type=task (not raw_input)
	taskID := uuid.New()
	clarItem, err := db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:               clarificationkind.StaleTask,
		SubjectType:        "task",
		SubjectID:          taskID,
		SubjectDescription: "test task",
		Question:           "Still relevant?",
		AnswerOptions:      json.RawMessage(`["yes","no"]`),
		PriorityScore:      0.5,
	})
	if err != nil {
		t.Fatalf("create clarification: %s", err)
	}

	input := clarificationapp.ResolveInput{
		Answer: json.RawMessage(`{"free_text": "some override text"}`),
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
}

func isNotFound(err error) bool {
	return errors.Is(err, sqldb.ErrDBNotFound)
}
