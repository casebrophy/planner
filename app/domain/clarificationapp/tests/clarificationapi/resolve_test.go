package clarificationapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/domain/clarificationapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/taskstatus"
)

func resolve200(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "basic",
			URL:        fmt.Sprintf("/api/v1/clarifications/%s/resolve", sd.items[0].ID),
			Token:      apitest.TestAPIKey,
			Method:     http.MethodPost,
			StatusCode: http.StatusOK,
			Input:      &clarificationapp.ResolveInput{Answer: json.RawMessage(`"yes"`)},
			GotResp:    &clarificationapp.ClarificationItem{},
			ExpResp:    &clarificationapp.ClarificationItem{Status: "resolved"},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(*clarificationapp.ClarificationItem)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(*clarificationapp.ClarificationItem)
				// Only verify status transition.
				if gotResp.Status != expResp.Status {
					return cmp.Diff(gotResp.Status, expResp.Status)
				}
				return ""
			},
		},
	}
}

func resolve401(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "no-key",
			URL:        fmt.Sprintf("/api/v1/clarifications/%s/resolve", sd.items[0].ID),
			Token:      "",
			Method:     http.MethodPost,
			StatusCode: http.StatusUnauthorized,
			GotResp:    &errs.Error{},
			ExpResp:    &errs.Error{Code: errs.Unauthenticated, Message: "missing api key"},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}

// TestResolve_KnowledgeGap_SelectedOption verifies that resolving a KnowledgeGap clarification
// with a selected_option creates a note with the option value and marks the clarification resolved.
func TestResolve_KnowledgeGap_SelectedOption(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "TestResolve_KnowledgeGap_SelectedOption")
	ctx := context.Background()
	db := test.DB

	// Create a KnowledgeGap clarification against a real task so the
	// dispatched note can satisfy notes.task_id_fkey.
	task, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title: "task for knowledge gap",
	})
	if err != nil {
		t.Fatalf("create task: %s", err)
	}
	selectedOptions := []string{"This week", "Next week", "Later"}
	answerOptions, _ := json.Marshal(selectedOptions)

	clarItem, err := db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:               clarificationkind.KnowledgeGap,
		SubjectType:        "task",
		SubjectID:          task.ID,
		SubjectDescription: "test task for knowledge gap",
		Question:           "When should this be done?",
		AnswerOptions:      answerOptions,
		PriorityScore:      0.7,
	})
	if err != nil {
		t.Fatalf("create clarification: %s", err)
	}

	// Resolve with selected_option
	selectedValue := "This week"
	input := clarificationapp.ResolveInput{
		Answer: json.RawMessage(fmt.Sprintf(`{"selected_option": %q}`, selectedValue)),
	}
	body, _ := json.Marshal(input)

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/clarifications/%s/resolve", clarItem.ID), bytes.NewBuffer(body))
	r.Header.Set("X-API-Key", apitest.TestAPIKey)
	w := httptest.NewRecorder()
	test.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify clarification is resolved
	var resp clarificationapp.ClarificationItem
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %s", err)
	}
	if resp.Status != "resolved" {
		t.Errorf("expected status=resolved, got %s", resp.Status)
	}

	// Verify the clarification answer was captured
	if resp.Answer == nil || len(resp.Answer) == 0 {
		t.Error("expected clarification answer to be captured")
	}

	// Verify a note was created with the selected_option content
	clarificationSource := "clarification"
	notes, err := db.BusDomain.Note.Query(ctx, notebus.QueryFilter{Source: &clarificationSource}, notebus.DefaultOrderBy, page.New(1, 100))
	if err != nil {
		t.Fatalf("query notes: %s", err)
	}

	// Find the note created by the resolution
	found := false
	for _, note := range notes {
		if note.Content == selectedValue {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected a note to be created with content %q and source 'clarification'", selectedValue)
	}
}

// TestResolve_KnowledgeGap_AnswerText_Regression verifies that resolving with answer_text
// (the free-text path) still works for KnowledgeGap and does not regress.
func TestResolve_KnowledgeGap_AnswerText_Regression(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "TestResolve_KnowledgeGap_AnswerText_Regression")
	ctx := context.Background()
	db := test.DB

	// Create a KnowledgeGap clarification
	taskID := uuid.New()
	answerOptions := json.RawMessage(`["Option A", "Option B"]`)

	clarItem, err := db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:               clarificationkind.KnowledgeGap,
		SubjectType:        "task",
		SubjectID:          taskID,
		SubjectDescription: "test task for knowledge gap",
		Question:           "What's your custom answer?",
		AnswerOptions:      answerOptions,
		PriorityScore:      0.7,
	})
	if err != nil {
		t.Fatalf("create clarification: %s", err)
	}

	// Resolve with answer_text (free-form text)
	answerText := "Custom answer provided by user"
	input := clarificationapp.ResolveInput{
		Answer: json.RawMessage(fmt.Sprintf(`{"answer_text": %q}`, answerText)),
	}
	body, _ := json.Marshal(input)

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/clarifications/%s/resolve", clarItem.ID), bytes.NewBuffer(body))
	r.Header.Set("X-API-Key", apitest.TestAPIKey)
	w := httptest.NewRecorder()
	test.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify clarification is resolved
	var resp clarificationapp.ClarificationItem
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %s", err)
	}
	if resp.Status != "resolved" {
		t.Errorf("expected status=resolved, got %s", resp.Status)
	}
}

// TestResolve_StaleTask_StatusDone verifies that resolving a StaleTask clarification
// with status=done updates the task status to Done.
func TestResolve_StaleTask_StatusDone(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "TestResolve_StaleTask_StatusDone")
	ctx := context.Background()
	db := test.DB

	// Create a task
	task, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title: "stale task",
	})
	if err != nil {
		t.Fatalf("create task: %s", err)
	}

	// Create a StaleTask clarification
	clarItem, err := db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:               clarificationkind.StaleTask,
		SubjectType:        "task",
		SubjectID:          task.ID,
		SubjectDescription: "test task",
		Question:           "Is this task still active?",
		AnswerOptions:      json.RawMessage(`["done","open"]`),
		PriorityScore:      0.8,
	})
	if err != nil {
		t.Fatalf("create clarification: %s", err)
	}

	// Resolve with status=done
	input := clarificationapp.ResolveInput{
		Answer: json.RawMessage(`{"status": "done"}`),
	}
	body, _ := json.Marshal(input)

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/clarifications/%s/resolve", clarItem.ID), bytes.NewBuffer(body))
	r.Header.Set("X-API-Key", apitest.TestAPIKey)
	w := httptest.NewRecorder()
	test.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify clarification is resolved
	var resp clarificationapp.ClarificationItem
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %s", err)
	}
	if resp.Status != "resolved" {
		t.Errorf("expected status=resolved, got %s", resp.Status)
	}

	// Verify task status was updated to Done
	updatedTask, err := db.BusDomain.Task.QueryByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("query task: %s", err)
	}
	if updatedTask.Status != taskstatus.Done {
		t.Errorf("expected task.Status=Done, got %s", updatedTask.Status)
	}
}

// TestResolve_StaleTask_NoteAddsThreadEntry verifies that resolving a StaleTask clarification
// with a note creates a thread entry on the task.
func TestResolve_StaleTask_NoteAddsThreadEntry(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "TestResolve_StaleTask_NoteAddsThreadEntry")
	ctx := context.Background()
	db := test.DB

	// Create a task
	task, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title: "stale task",
	})
	if err != nil {
		t.Fatalf("create task: %s", err)
	}

	// Create a StaleTask clarification
	clarItem, err := db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:               clarificationkind.StaleTask,
		SubjectType:        "task",
		SubjectID:          task.ID,
		SubjectDescription: "test task",
		Question:           "Is this task still active?",
		AnswerOptions:      json.RawMessage(`["done","open"]`),
		PriorityScore:      0.8,
	})
	if err != nil {
		t.Fatalf("create clarification: %s", err)
	}

	// Resolve with status=open + note
	noteText := "still blocked on review"
	input := clarificationapp.ResolveInput{
		Answer: json.RawMessage(fmt.Sprintf(`{"status": "open", "note": %q}`, noteText)),
	}
	body, _ := json.Marshal(input)

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/clarifications/%s/resolve", clarItem.ID), bytes.NewBuffer(body))
	r.Header.Set("X-API-Key", apitest.TestAPIKey)
	w := httptest.NewRecorder()
	test.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify clarification is resolved
	var resp clarificationapp.ClarificationItem
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %s", err)
	}
	if resp.Status != "resolved" {
		t.Errorf("expected status=resolved, got %s", resp.Status)
	}

	// Verify task status is open
	updatedTask, err := db.BusDomain.Task.QueryByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("query task: %s", err)
	}
	if updatedTask.Status != taskstatus.Open {
		t.Errorf("expected task.Status=Open, got %s", updatedTask.Status)
	}

	// Verify a thread entry was created with the note
	entries, err := db.BusDomain.Thread.Query(ctx, threadbus.QueryFilter{
		SubjectType: ptr("task"),
		SubjectID:   &task.ID,
	}, threadbus.DefaultOrderBy, page.New(1, 10))
	if err != nil {
		t.Fatalf("query thread entries: %s", err)
	}

	found := false
	for _, entry := range entries {
		if entry.Content == noteText {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a thread entry with content %q", noteText)
	}
}

func ptr[T any](v T) *T {
	return &v
}

// TestResolve_AmbiguousDeadline_UpdatesTaskDueDate verifies that resolving an
// AmbiguousDeadline clarification applies the parsed due_date to the subject task.
func TestResolve_AmbiguousDeadline_UpdatesTaskDueDate(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "TestResolve_AmbiguousDeadline_UpdatesTaskDueDate")
	ctx := context.Background()
	db := test.DB

	// Create a task with no due date.
	task, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title: "task with ambiguous deadline",
	})
	if err != nil {
		t.Fatalf("create task: %s", err)
	}
	if task.DueDate != nil {
		t.Fatalf("precondition: task.DueDate expected nil, got %v", task.DueDate)
	}

	// Create an AmbiguousDeadline clarification for the task.
	clarItem, err := db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:               clarificationkind.AmbiguousDeadline,
		SubjectType:        "task",
		SubjectID:          task.ID,
		SubjectDescription: "task with ambiguous deadline",
		Question:           "When is this due?",
		AnswerOptions:      json.RawMessage(`{"description":"soon","raw_date":"soon"}`),
		PriorityScore:      0.8,
	})
	if err != nil {
		t.Fatalf("create clarification: %s", err)
	}

	// Resolve with a due_date.
	input := clarificationapp.ResolveInput{
		Answer: json.RawMessage(`{"due_date": "2026-05-15"}`),
	}
	body, _ := json.Marshal(input)

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/clarifications/%s/resolve", clarItem.ID), bytes.NewBuffer(body))
	r.Header.Set("X-API-Key", apitest.TestAPIKey)
	w := httptest.NewRecorder()
	test.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the task's due date was updated.
	updated, err := db.BusDomain.Task.QueryByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("query task: %s", err)
	}
	if updated.DueDate == nil {
		t.Fatalf("expected task.DueDate to be set, got nil")
	}
	due := updated.DueDate.UTC()
	if due.Year() != 2026 || due.Month() != 5 || due.Day() != 15 {
		t.Errorf("expected due date 2026-05-15, got %s", due.Format("2006-01-02"))
	}

	// Verify clarification is resolved.
	var resp clarificationapp.ClarificationItem
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %s", err)
	}
	if resp.Status != "resolved" {
		t.Errorf("expected status=resolved, got %s", resp.Status)
	}
}

// TestResolve_AmbiguousAction_CreatesTask verifies that resolving an
// AmbiguousAction clarification with {selected: idx} creates a new task with
// Title=interpretations[idx].
func TestResolve_AmbiguousAction_CreatesTask(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "TestResolve_AmbiguousAction_CreatesTask")
	ctx := context.Background()
	db := test.DB

	// Build AnswerOptions with two interpretations.
	opts := clarificationbus.AmbiguousActionOptions{
		Interpretations: []string{
			"Ship the redesign by Friday",
			"Review the redesign doc by Friday",
		},
	}
	optsRaw, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("marshal options: %s", err)
	}

	// Create an AmbiguousAction clarification. The dispatcher does not query
	// the subject for this kind, so a real raw_input row is not needed.
	clarItem, err := db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:               clarificationkind.AmbiguousAction,
		SubjectType:        "raw_input",
		SubjectID:          uuid.New(),
		SubjectDescription: "ship or review the redesign",
		Question:           "Which interpretation is correct?",
		AnswerOptions:      optsRaw,
		PriorityScore:      0.8,
	})
	if err != nil {
		t.Fatalf("create clarification: %s", err)
	}

	// Baseline task count.
	baseline, err := db.BusDomain.Task.Count(ctx, taskbus.QueryFilter{})
	if err != nil {
		t.Fatalf("count tasks baseline: %s", err)
	}

	// Resolve with selected=1.
	input := clarificationapp.ResolveInput{
		Answer: json.RawMessage(`{"selected": 1}`),
	}
	body, _ := json.Marshal(input)

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/clarifications/%s/resolve", clarItem.ID), bytes.NewBuffer(body))
	r.Header.Set("X-API-Key", apitest.TestAPIKey)
	w := httptest.NewRecorder()
	test.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify task count increased by 1.
	after, err := db.BusDomain.Task.Count(ctx, taskbus.QueryFilter{})
	if err != nil {
		t.Fatalf("count tasks after: %s", err)
	}
	if after != baseline+1 {
		t.Fatalf("expected task count to grow by 1 (baseline=%d, after=%d)", baseline, after)
	}

	// Verify a task exists with the expected title and Open status.
	tasks, err := db.BusDomain.Task.Query(ctx, taskbus.QueryFilter{}, order.NewBy(taskbus.OrderByCreatedAt, order.DESC), page.New(1, 200))
	if err != nil {
		t.Fatalf("query tasks: %s", err)
	}
	wantTitle := "Review the redesign doc by Friday"
	var found *taskbus.Task
	for i := range tasks {
		if tasks[i].Title == wantTitle {
			found = &tasks[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected a task with title %q, none found", wantTitle)
	}
	if found.Status != taskstatus.Open {
		t.Errorf("expected created task.Status=Open, got %s", found.Status)
	}

	// Verify clarification is resolved.
	var resp clarificationapp.ClarificationItem
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %s", err)
	}
	if resp.Status != "resolved" {
		t.Errorf("expected status=resolved, got %s", resp.Status)
	}
}
