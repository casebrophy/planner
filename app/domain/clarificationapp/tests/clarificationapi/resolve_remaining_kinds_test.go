package clarificationapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/domain/clarificationapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/observationbus"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/contextkind"
	"github.com/casebrophy/planner/business/types/observationkind"
	"github.com/casebrophy/planner/business/types/rawinputsource"
)

// TestResolve_ContextAssignment_UpdatesTaskContext verifies that resolving a ContextAssignment
// clarification with {context_id: UUID} updates the subject task's context_id field.
func TestResolve_ContextAssignment_UpdatesTaskContext(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "TestResolve_ContextAssignment_UpdatesTaskContext")
	ctx := context.Background()
	db := test.DB

	// Create a context to assign to
	context1, err := db.BusDomain.Context.Create(ctx, contextbus.NewContext{
		Title: "project alpha",
		Kind:  contextkind.Project,
	})
	if err != nil {
		t.Fatalf("create context: %s", err)
	}

	// Create a task with no context
	task, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title: "unassigned task",
	})
	if err != nil {
		t.Fatalf("create task: %s", err)
	}
	if task.ContextID != nil {
		t.Fatalf("precondition: task.ContextID expected nil, got %v", task.ContextID)
	}

	// Create a ContextAssignment clarification
	clarItem, err := db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:               clarificationkind.ContextAssignment,
		SubjectType:        "task",
		SubjectID:          task.ID,
		SubjectDescription: "unassigned task",
		Question:           "Which context?",
		AnswerOptions:      json.RawMessage(`[{"id":"ctx1","title":"project alpha"}]`),
		PriorityScore:      0.8,
	})
	if err != nil {
		t.Fatalf("create clarification: %s", err)
	}

	// Resolve with context_id
	input := clarificationapp.ResolveInput{
		Answer: json.RawMessage(fmt.Sprintf(`{"context_id": %q}`, context1.ID.String())),
	}
	body, _ := json.Marshal(input)

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/clarifications/%s/resolve", clarItem.ID), bytes.NewBuffer(body))
	r.Header.Set("X-API-Key", apitest.TestAPIKey)
	w := httptest.NewRecorder()
	test.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify task context_id was updated
	updatedTask, err := db.BusDomain.Task.QueryByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("query task: %s", err)
	}
	if updatedTask.ContextID == nil || *updatedTask.ContextID != context1.ID {
		t.Errorf("expected task.ContextID=%s, got %v", context1.ID, updatedTask.ContextID)
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

// TestResolve_NewContext_ConfirmUpdatesContext verifies that resolving a NewContext clarification
// with action="confirm" and title/description updates the context.
func TestResolve_NewContext_ConfirmUpdatesContext(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "TestResolve_NewContext_ConfirmUpdatesContext")
	ctx := context.Background()
	db := test.DB

	// Create a context with placeholder title
	context1, err := db.BusDomain.Context.Create(ctx, contextbus.NewContext{
		Title: "placeholder",
		Kind:  contextkind.Project,
	})
	if err != nil {
		t.Fatalf("create context: %s", err)
	}

	// Create a NewContext clarification
	clarItem, err := db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:               clarificationkind.NewContext,
		SubjectType:        "context",
		SubjectID:          context1.ID,
		SubjectDescription: "new context",
		Question:           "Confirm this context?",
		AnswerOptions:      json.RawMessage(`{"title":"placeholder","description":""}`),
		PriorityScore:      0.7,
	})
	if err != nil {
		t.Fatalf("create clarification: %s", err)
	}

	// Resolve with action=confirm and updated title/description
	newTitle := "Q2 Planning"
	newDesc := "Second quarter planning cycle"
	input := clarificationapp.ResolveInput{
		Answer: json.RawMessage(fmt.Sprintf(`{
			"action": "confirm",
			"title": %q,
			"description": %q
		}`, newTitle, newDesc)),
	}
	body, _ := json.Marshal(input)

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/clarifications/%s/resolve", clarItem.ID), bytes.NewBuffer(body))
	r.Header.Set("X-API-Key", apitest.TestAPIKey)
	w := httptest.NewRecorder()
	test.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify context was updated
	updatedContext, err := db.BusDomain.Context.QueryByID(ctx, context1.ID)
	if err != nil {
		t.Fatalf("query context: %s", err)
	}
	if updatedContext.Title != newTitle {
		t.Errorf("expected context.Title=%q, got %q", newTitle, updatedContext.Title)
	}
	if updatedContext.Description != newDesc {
		t.Errorf("expected context.Description=%q, got %q", newDesc, updatedContext.Description)
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

// TestResolve_NewContext_MergeDeletesContext verifies that resolving with action="merge"
// deletes the subject context.
func TestResolve_NewContext_MergeDeletesContext(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "TestResolve_NewContext_MergeDeletesContext")
	ctx := context.Background()
	db := test.DB

	// Create two contexts
	context1, err := db.BusDomain.Context.Create(ctx, contextbus.NewContext{
		Title: "duplicate context",
		Kind:  contextkind.Project,
	})
	if err != nil {
		t.Fatalf("create context1: %s", err)
	}

	context2, err := db.BusDomain.Context.Create(ctx, contextbus.NewContext{
		Title: "merge target",
		Kind:  contextkind.Project,
	})
	if err != nil {
		t.Fatalf("create context2: %s", err)
	}

	// Create a NewContext clarification
	clarItem, err := db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:               clarificationkind.NewContext,
		SubjectType:        "context",
		SubjectID:          context1.ID,
		SubjectDescription: "duplicate context",
		Question:           "Merge this context?",
		AnswerOptions:      json.RawMessage(`{}`),
		PriorityScore:      0.7,
	})
	if err != nil {
		t.Fatalf("create clarification: %s", err)
	}

	// Resolve with action=merge
	input := clarificationapp.ResolveInput{
		Answer: json.RawMessage(fmt.Sprintf(`{
			"action": "merge",
			"merge_target_id": %q
		}`, context2.ID.String())),
	}
	body, _ := json.Marshal(input)

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/clarifications/%s/resolve", clarItem.ID), bytes.NewBuffer(body))
	r.Header.Set("X-API-Key", apitest.TestAPIKey)
	w := httptest.NewRecorder()
	test.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify context1 was deleted
	_, err = db.BusDomain.Context.QueryByID(ctx, context1.ID)
	if err == nil {
		t.Error("expected context1 to be deleted, but it still exists")
	} else if !isNotFound(err) {
		t.Errorf("unexpected error querying context1: %s", err)
	}

	// Verify context2 still exists
	_, err = db.BusDomain.Context.QueryByID(ctx, context2.ID)
	if err != nil {
		t.Errorf("context2 should still exist, got error: %s", err)
	}
}

// TestResolve_InactivityPrompt_CompleteTask verifies that resolving an InactivityPrompt
// clarification with action="completed" completes the subject task and adds a thread entry.
func TestResolve_InactivityPrompt_CompleteTask(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "TestResolve_InactivityPrompt_CompleteTask")
	ctx := context.Background()
	db := test.DB

	// Create a task
	task, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title: "inactive task",
	})
	if err != nil {
		t.Fatalf("create task: %s", err)
	}

	// Create an InactivityPrompt clarification
	clarItem, err := db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:               clarificationkind.InactivityPrompt,
		SubjectType:        "task",
		SubjectID:          task.ID,
		SubjectDescription: "inactive task",
		Question:           "Still working on this?",
		AnswerOptions:      json.RawMessage(`["completed","postpone"]`),
		PriorityScore:      0.6,
	})
	if err != nil {
		t.Fatalf("create clarification: %s", err)
	}

	// Resolve with action=completed
	input := clarificationapp.ResolveInput{
		Answer: json.RawMessage(`{"action": "completed"}`),
	}
	body, _ := json.Marshal(input)

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/clarifications/%s/resolve", clarItem.ID), bytes.NewBuffer(body))
	r.Header.Set("X-API-Key", apitest.TestAPIKey)
	w := httptest.NewRecorder()
	test.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify task was marked done
	updatedTask, err := db.BusDomain.Task.QueryByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("query task: %s", err)
	}
	if updatedTask.Status.String() != "Done" {
		t.Errorf("expected task.Status=Done, got %s", updatedTask.Status)
	}

	// Verify a thread entry was created
	entries, err := db.BusDomain.Thread.Query(ctx, threadbus.QueryFilter{
		SubjectType: ptr("task"),
		SubjectID:   &task.ID,
	}, threadbus.DefaultOrderBy, page.New(1, 10))
	if err != nil {
		t.Fatalf("query thread entries: %s", err)
	}
	if len(entries) == 0 {
		t.Error("expected at least one thread entry")
	}
}

// TestResolve_InactivityPrompt_WithNote verifies that resolving with a custom note
// creates a thread entry with that note.
func TestResolve_InactivityPrompt_WithNote(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "TestResolve_InactivityPrompt_WithNote")
	ctx := context.Background()
	db := test.DB

	// Create a task
	task, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title: "inactive task",
	})
	if err != nil {
		t.Fatalf("create task: %s", err)
	}

	// Create an InactivityPrompt clarification
	clarItem, err := db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:               clarificationkind.InactivityPrompt,
		SubjectType:        "task",
		SubjectID:          task.ID,
		SubjectDescription: "inactive task",
		Question:           "Still working on this?",
		AnswerOptions:      json.RawMessage(`["completed","postpone"]`),
		PriorityScore:      0.6,
	})
	if err != nil {
		t.Fatalf("create clarification: %s", err)
	}

	// Resolve with a custom note
	noteText := "Moved to backlog pending Q3 planning"
	input := clarificationapp.ResolveInput{
		Answer: json.RawMessage(fmt.Sprintf(`{"action": "postpone", "note": %q}`, noteText)),
	}
	body, _ := json.Marshal(input)

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/clarifications/%s/resolve", clarItem.ID), bytes.NewBuffer(body))
	r.Header.Set("X-API-Key", apitest.TestAPIKey)
	w := httptest.NewRecorder()
	test.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
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

// TestResolve_ContextDebrief_RecordsObservation verifies that resolving a ContextDebrief
// clarification records an Observation with Kind=Debrief.
func TestResolve_ContextDebrief_RecordsObservation(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "TestResolve_ContextDebrief_RecordsObservation")
	ctx := context.Background()
	db := test.DB

	// Create a context
	context1, err := db.BusDomain.Context.Create(ctx, contextbus.NewContext{
		Title: "debrief context",
		Kind:  contextkind.Project,
	})
	if err != nil {
		t.Fatalf("create context: %s", err)
	}

	// Create a ContextDebrief clarification
	question := "What did you learn?"
	clarItem, err := db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:               clarificationkind.ContextDebrief,
		SubjectType:        "context",
		SubjectID:          context1.ID,
		SubjectDescription: "debrief context",
		Question:           question,
		AnswerOptions:      json.RawMessage(`{}`),
		PriorityScore:      0.8,
	})
	if err != nil {
		t.Fatalf("create clarification: %s", err)
	}

	// Resolve with a response
	responseText := "Key takeaway: better upfront planning saves time"
	input := clarificationapp.ResolveInput{
		Answer: json.RawMessage(fmt.Sprintf(`{"response": %q}`, responseText)),
	}
	body, _ := json.Marshal(input)

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/clarifications/%s/resolve", clarItem.ID), bytes.NewBuffer(body))
	r.Header.Set("X-API-Key", apitest.TestAPIKey)
	w := httptest.NewRecorder()
	test.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify an observation was recorded
	kind := observationkind.Debrief
	obs, err := db.BusDomain.Observation.Query(ctx, observationbus.QueryFilter{
		SubjectType: ptr("context"),
		SubjectID:   &context1.ID,
		Kind:        &kind,
	}, observationbus.DefaultOrderBy, page.New(1, 10))
	if err != nil {
		t.Fatalf("query observations: %s", err)
	}

	if len(obs) == 0 {
		t.Fatal("expected at least one debrief observation")
	}

	// Verify the observation data contains response and question
	var data map[string]string
	if err := json.Unmarshal(obs[0].Data, &data); err != nil {
		t.Fatalf("unmarshal observation data: %s", err)
	}
	if data["response"] != responseText {
		t.Errorf("expected response=%q, got %q", responseText, data["response"])
	}
	if data["question"] != question {
		t.Errorf("expected question=%q, got %q", question, data["question"])
	}
}

// TestResolve_ContextDebrief_UpdatesContextStatus verifies that when all debrief
// clarifications for a context are resolved, context.DebriefStatus is set to Done.
func TestResolve_ContextDebrief_UpdatesContextStatus(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "TestResolve_ContextDebrief_UpdatesContextStatus")
	ctx := context.Background()
	db := test.DB

	// Create a context
	context1, err := db.BusDomain.Context.Create(ctx, contextbus.NewContext{
		Title: "debrief context",
		Kind:  contextkind.Project,
	})
	if err != nil {
		t.Fatalf("create context: %s", err)
	}

	// Create a single ContextDebrief clarification (only one pending)
	clarItem, err := db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:               clarificationkind.ContextDebrief,
		SubjectType:        "context",
		SubjectID:          context1.ID,
		SubjectDescription: "debrief context",
		Question:           "What did you learn?",
		AnswerOptions:      json.RawMessage(`{}`),
		PriorityScore:      0.8,
	})
	if err != nil {
		t.Fatalf("create clarification: %s", err)
	}

	// Resolve the clarification
	input := clarificationapp.ResolveInput{
		Answer: json.RawMessage(`{"response": "Good progress overall"}`),
	}
	body, _ := json.Marshal(input)

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/clarifications/%s/resolve", clarItem.ID), bytes.NewBuffer(body))
	r.Header.Set("X-API-Key", apitest.TestAPIKey)
	w := httptest.NewRecorder()
	test.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify context.DebriefStatus was updated to Done
	updatedContext, err := db.BusDomain.Context.QueryByID(ctx, context1.ID)
	if err != nil {
		t.Fatalf("query context: %s", err)
	}
	if updatedContext.DebriefStatus.String() != "Done" {
		t.Errorf("expected DebriefStatus=Done, got %s", updatedContext.DebriefStatus.String())
	}
}

// TestResolve_EntityLink_CreatesLink verifies that resolving an EntityLink
// clarification with confirmed=true creates an EntityLink between source and target.
func TestResolve_EntityLink_CreatesLink(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "TestResolve_EntityLink_CreatesLink")
	ctx := context.Background()
	db := test.DB

	// Create two tasks to link
	task1, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title: "source task",
	})
	if err != nil {
		t.Fatalf("create task1: %s", err)
	}

	task2, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title: "target task",
	})
	if err != nil {
		t.Fatalf("create task2: %s", err)
	}

	// Create EntityLink options
	opts := clarificationbus.EntityLinkOptions{
		SourceType: "task",
		SourceID:   task1.ID.String(),
		TargetType: "task",
		TargetID:   task2.ID.String(),
		Confidence: 0.9,
	}
	optsJSON, _ := json.Marshal(opts)

	// Create an EntityLink clarification
	clarItem, err := db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:               clarificationkind.EntityLink,
		SubjectType:        "task",
		SubjectID:          task1.ID,
		SubjectDescription: "entity link suggestion",
		Question:           "Link these tasks?",
		AnswerOptions:      optsJSON,
		PriorityScore:      0.7,
	})
	if err != nil {
		t.Fatalf("create clarification: %s", err)
	}

	// Resolve with confirmed=true
	input := clarificationapp.ResolveInput{
		Answer: json.RawMessage(`{"confirmed": true}`),
	}
	body, _ := json.Marshal(input)

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/clarifications/%s/resolve", clarItem.ID), bytes.NewBuffer(body))
	r.Header.Set("X-API-Key", apitest.TestAPIKey)
	w := httptest.NewRecorder()
	test.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify EntityLink was created
	links, err := db.BusDomain.EntityLink.QueryByEntity(ctx, "task", task1.ID)
	if err != nil {
		t.Fatalf("query entity links: %s", err)
	}

	found := false
	for _, link := range links {
		if link.SourceType == "task" && link.SourceID == task1.ID &&
			link.TargetType == "task" && link.TargetID == task2.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected an entity link to be created")
	}
}

// TestResolve_TaskDebrief_RecordsObservation verifies that resolving a TaskDebrief
// clarification records an Observation with Kind=Debrief.
func TestResolve_TaskDebrief_RecordsObservation(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "TestResolve_TaskDebrief_RecordsObservation")
	ctx := context.Background()
	db := test.DB

	// Create a task
	task, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title: "debrief task",
	})
	if err != nil {
		t.Fatalf("create task: %s", err)
	}

	// Create a TaskDebrief clarification
	question := "How important was this?"
	clarItem, err := db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:               clarificationkind.TaskDebrief,
		SubjectType:        "task",
		SubjectID:          task.ID,
		SubjectDescription: "debrief task",
		Question:           question,
		AnswerOptions:      json.RawMessage(`["high","medium","low"]`),
		PriorityScore:      0.7,
	})
	if err != nil {
		t.Fatalf("create clarification: %s", err)
	}

	// Resolve with importance level
	input := clarificationapp.ResolveInput{
		Answer: json.RawMessage(`{"value": "high"}`),
	}
	body, _ := json.Marshal(input)

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/clarifications/%s/resolve", clarItem.ID), bytes.NewBuffer(body))
	r.Header.Set("X-API-Key", apitest.TestAPIKey)
	w := httptest.NewRecorder()
	test.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify an observation was recorded
	kind := observationkind.Debrief
	obs, err := db.BusDomain.Observation.Query(ctx, observationbus.QueryFilter{
		SubjectType: ptr("task"),
		SubjectID:   &task.ID,
		Kind:        &kind,
	}, observationbus.DefaultOrderBy, page.New(1, 10))
	if err != nil {
		t.Fatalf("query observations: %s", err)
	}

	if len(obs) == 0 {
		t.Fatal("expected at least one debrief observation")
	}

	// Verify the observation data
	var data map[string]string
	if err := json.Unmarshal(obs[0].Data, &data); err != nil {
		t.Fatalf("unmarshal observation data: %s", err)
	}
	if data["importance"] != "high" {
		t.Errorf("expected importance=high, got %q", data["importance"])
	}
	if data["question"] != question {
		t.Errorf("expected question=%q, got %q", question, data["question"])
	}
}

// TestResolve_TaskDebrief_SkipNoObservation verifies that resolving with value="skip"
// does not create an observation.
func TestResolve_TaskDebrief_SkipNoObservation(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "TestResolve_TaskDebrief_SkipNoObservation")
	ctx := context.Background()
	db := test.DB

	// Create a task
	task, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title: "debrief task",
	})
	if err != nil {
		t.Fatalf("create task: %s", err)
	}

	// Create a TaskDebrief clarification
	clarItem, err := db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:               clarificationkind.TaskDebrief,
		SubjectType:        "task",
		SubjectID:          task.ID,
		SubjectDescription: "debrief task",
		Question:           "How important was this?",
		AnswerOptions:      json.RawMessage(`["high","medium","low","skip"]`),
		PriorityScore:      0.7,
	})
	if err != nil {
		t.Fatalf("create clarification: %s", err)
	}

	// Baseline observation count
	kind := observationkind.Debrief
	baselineObs, err := db.BusDomain.Observation.Query(ctx, observationbus.QueryFilter{
		SubjectType: ptr("task"),
		SubjectID:   &task.ID,
		Kind:        &kind,
	}, observationbus.DefaultOrderBy, page.New(1, 10))
	if err != nil {
		t.Fatalf("query observations baseline: %s", err)
	}
	baselineCount := len(baselineObs)

	// Resolve with value=skip
	input := clarificationapp.ResolveInput{
		Answer: json.RawMessage(`{"value": "skip"}`),
	}
	body, _ := json.Marshal(input)

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/clarifications/%s/resolve", clarItem.ID), bytes.NewBuffer(body))
	r.Header.Set("X-API-Key", apitest.TestAPIKey)
	w := httptest.NewRecorder()
	test.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify no new observation was created
	afterObs, err := db.BusDomain.Observation.Query(ctx, observationbus.QueryFilter{
		SubjectType: ptr("task"),
		SubjectID:   &task.ID,
		Kind:        &kind,
	}, observationbus.DefaultOrderBy, page.New(1, 10))
	if err != nil {
		t.Fatalf("query observations after: %s", err)
	}

	if len(afterObs) != baselineCount {
		t.Errorf("expected observation count to stay %d, got %d", baselineCount, len(afterObs))
	}
}

// TestResolve_WeeklyReview_RecordsObservations verifies that resolving a WeeklyReview
// clarification records an Observation for each selected task.
func TestResolve_WeeklyReview_RecordsObservations(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "TestResolve_WeeklyReview_RecordsObservations")
	ctx := context.Background()
	db := test.DB

	// Create three tasks
	task1, _ := db.BusDomain.Task.Create(ctx, taskbus.NewTask{Title: "task1"})
	task2, _ := db.BusDomain.Task.Create(ctx, taskbus.NewTask{Title: "task2"})
	task3, _ := db.BusDomain.Task.Create(ctx, taskbus.NewTask{Title: "task3"})

	// Create a WeeklyReview clarification
	clarItem, err := db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:               clarificationkind.WeeklyReview,
		SubjectType:        "context",
		SubjectID:          uuid.New(),
		SubjectDescription: "weekly review",
		Question:           "Which tasks were high impact?",
		AnswerOptions:      json.RawMessage(`{}`),
		PriorityScore:      0.8,
	})
	if err != nil {
		t.Fatalf("create clarification: %s", err)
	}

	// Resolve with selected task IDs (select task1 and task3)
	input := clarificationapp.ResolveInput{
		Answer: json.RawMessage(fmt.Sprintf(`{
			"selected_task_ids": [%q, %q]
		}`, task1.ID.String(), task3.ID.String())),
	}
	body, _ := json.Marshal(input)

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/clarifications/%s/resolve", clarItem.ID), bytes.NewBuffer(body))
	r.Header.Set("X-API-Key", apitest.TestAPIKey)
	w := httptest.NewRecorder()
	test.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify observations were recorded for selected tasks
	kind := observationkind.Debrief
	for _, taskID := range []*uuid.UUID{&task1.ID, &task3.ID} {
		obs, err := db.BusDomain.Observation.Query(ctx, observationbus.QueryFilter{
			SubjectType: ptr("task"),
			SubjectID:   taskID,
			Kind:        &kind,
		}, observationbus.DefaultOrderBy, page.New(1, 10))
		if err != nil {
			t.Fatalf("query observations for task: %s", err)
		}

		if len(obs) == 0 {
			t.Errorf("expected observation for task %s", taskID)
		} else {
			var data map[string]string
			if err := json.Unmarshal(obs[0].Data, &data); err == nil {
				if data["source"] != "weekly_review" {
					t.Errorf("expected source=weekly_review, got %q", data["source"])
				}
				if data["importance"] != "high" {
					t.Errorf("expected importance=high, got %q", data["importance"])
				}
			}
		}
	}

	// Verify no observation was recorded for task2 (not selected)
	obs2, err := db.BusDomain.Observation.Query(ctx, observationbus.QueryFilter{
		SubjectType: ptr("task"),
		SubjectID:   &task2.ID,
		Kind:        &kind,
	}, observationbus.DefaultOrderBy, page.New(1, 10))
	if err != nil {
		t.Fatalf("query observations for task2: %s", err)
	}
	if len(obs2) > 0 {
		t.Error("expected no observation for task2, but found one")
	}
}

// TestResolve_AmbiguousEntityMatch_UseExistingDeletesDuplicate verifies that resolving
// with choice="use_existing" deletes the unconfirmed duplicate entity.
func TestResolve_AmbiguousEntityMatch_UseExistingDeletesDuplicate(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "TestResolve_AmbiguousEntityMatch_UseExistingDeletesDuplicate")
	ctx := context.Background()
	db := test.DB

	// Create a raw_input
	ri, err := db.BusDomain.RawInput.Create(ctx, rawinputbus.NewRawInput{
		SourceType: rawinputsource.Voice,
		RawContent: "schedule meeting with john",
	})
	if err != nil {
		t.Fatalf("create raw_input: %s", err)
	}

	// Create an unconfirmed task from the raw_input
	unconfirmedTask, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title:       "schedule meeting",
		RawInputID:  &ri.ID,
		Unconfirmed: true,
	})
	if err != nil {
		t.Fatalf("create unconfirmed task: %s", err)
	}

	// Create AmbiguousEntityMatch options
	opts := clarificationbus.AmbiguousEntityMatchOptions{
		CandidateID:    unconfirmedTask.ID.String(),
		CandidateType:  "task",
		CandidateTitle: "schedule meeting",
		Similarity:     0.8,
		Choices:        []string{"use_existing", "create_new"},
	}
	optsJSON, _ := json.Marshal(opts)

	// Create an AmbiguousEntityMatch clarification
	clarItem, err := db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:               clarificationkind.AmbiguousEntityMatch,
		SubjectType:        "raw_input",
		SubjectID:          ri.ID,
		SubjectDescription: "ambiguous entity match",
		Question:           "Use existing or create new?",
		AnswerOptions:      optsJSON,
		PriorityScore:      0.7,
	})
	if err != nil {
		t.Fatalf("create clarification: %s", err)
	}

	// Resolve with choice=use_existing
	input := clarificationapp.ResolveInput{
		Answer: json.RawMessage(`{"choice": "use_existing"}`),
	}
	body, _ := json.Marshal(input)

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/clarifications/%s/resolve", clarItem.ID), bytes.NewBuffer(body))
	r.Header.Set("X-API-Key", apitest.TestAPIKey)
	w := httptest.NewRecorder()
	test.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify unconfirmed task was deleted
	_, err = db.BusDomain.Task.QueryByID(ctx, unconfirmedTask.ID)
	if err == nil {
		t.Error("expected unconfirmed task to be deleted")
	} else if !isNotFound(err) {
		t.Errorf("unexpected error querying task: %s", err)
	}
}

// TestResolve_AmbiguousEntityMatch_CreateNewKeepsTask verifies that resolving
// with choice="create_new" leaves the unconfirmed task intact.
func TestResolve_AmbiguousEntityMatch_CreateNewKeepsTask(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "TestResolve_AmbiguousEntityMatch_CreateNewKeepsTask")
	ctx := context.Background()
	db := test.DB

	// Create a raw_input
	ri, err := db.BusDomain.RawInput.Create(ctx, rawinputbus.NewRawInput{
		SourceType: rawinputsource.Voice,
		RawContent: "schedule meeting with john",
	})
	if err != nil {
		t.Fatalf("create raw_input: %s", err)
	}

	// Create an unconfirmed task from the raw_input
	unconfirmedTask, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title:       "schedule meeting",
		RawInputID:  &ri.ID,
		Unconfirmed: true,
	})
	if err != nil {
		t.Fatalf("create unconfirmed task: %s", err)
	}

	// Create AmbiguousEntityMatch options
	opts := clarificationbus.AmbiguousEntityMatchOptions{
		CandidateID:    unconfirmedTask.ID.String(),
		CandidateType:  "task",
		CandidateTitle: "schedule meeting",
		Similarity:     0.8,
		Choices:        []string{"use_existing", "create_new"},
	}
	optsJSON, _ := json.Marshal(opts)

	// Create an AmbiguousEntityMatch clarification
	clarItem, err := db.BusDomain.Clarification.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:               clarificationkind.AmbiguousEntityMatch,
		SubjectType:        "raw_input",
		SubjectID:          ri.ID,
		SubjectDescription: "ambiguous entity match",
		Question:           "Use existing or create new?",
		AnswerOptions:      optsJSON,
		PriorityScore:      0.7,
	})
	if err != nil {
		t.Fatalf("create clarification: %s", err)
	}

	// Resolve with choice=create_new
	input := clarificationapp.ResolveInput{
		Answer: json.RawMessage(`{"choice": "create_new"}`),
	}
	body, _ := json.Marshal(input)

	r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/clarifications/%s/resolve", clarItem.ID), bytes.NewBuffer(body))
	r.Header.Set("X-API-Key", apitest.TestAPIKey)
	w := httptest.NewRecorder()
	test.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify unconfirmed task still exists
	_, err = db.BusDomain.Task.QueryByID(ctx, unconfirmedTask.ID)
	if err != nil {
		t.Errorf("expected unconfirmed task to survive, got error: %s", err)
	}
}

