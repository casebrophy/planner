package reclassifyapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/domain/noteapp"
	"github.com/casebrophy/planner/app/domain/taskapp"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/types/taskpriority"
	"github.com/casebrophy/planner/business/types/taskstatus"
)

const testAPIKey = "test-api-key"

func TestConvertTaskToNote(t *testing.T) {
	db := dbtest.New(t, "TestConvertTaskToNote")
	ctx := context.Background()

	// Create a test task
	task, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title:       "Test Task",
		Description: "Test Description",
		Status:      taskstatus.Open,
		Priority:    taskpriority.Medium,
	})
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	// Setup handler
	cfg := mux.Config{
		Log:    db.Log,
		DB:     db.DB,
		APIKey: testAPIKey,
	}

	reclassifyBus := db.BusDomain.Reclassify
	hdl := &app{log: cfg.Log, reclassifyBus: reclassifyBus}

	// Test happy path
	url := fmt.Sprintf("/api/v1/tasks/%s/convert-to-note", task.ID.String())
	r := httptest.NewRequest(http.MethodPost, url, nil)
	r.Header.Set("X-API-Key", testAPIKey)
	r.SetPathValue("task_id", task.ID.String())

	result := hdl.convertTaskToNote(ctx, r)
	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	data, _, err := result.Encode()
	if err != nil {
		t.Fatalf("Failed to encode result: %v", err)
	}

	var responseNote noteapp.Note
	if err := json.Unmarshal(data, &responseNote); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if responseNote.Content != "Test Task\n\nTest Description" {
		t.Errorf("Expected content 'Test Task\\n\\nTest Description', got '%s'", responseNote.Content)
	}
	if responseNote.Source != "reclassified_from_task" {
		t.Errorf("Expected source 'reclassified_from_task', got '%s'", responseNote.Source)
	}
}

func TestConvertNoteToTask(t *testing.T) {
	db := dbtest.New(t, "TestConvertNoteToTask")
	ctx := context.Background()

	// Create a test note
	note, err := db.BusDomain.Note.Create(ctx, notebus.NewNote{
		Content: "First line\nSecond line",
		Source:  "manual",
	})
	if err != nil {
		t.Fatalf("Failed to create test note: %v", err)
	}

	// Setup handler
	cfg := mux.Config{
		Log:    db.Log,
		DB:     db.DB,
		APIKey: testAPIKey,
	}

	reclassifyBus := db.BusDomain.Reclassify
	hdl := &app{log: cfg.Log, reclassifyBus: reclassifyBus}

	// Test happy path
	url := fmt.Sprintf("/api/v1/notes/%s/convert-to-task", note.ID.String())
	r := httptest.NewRequest(http.MethodPost, url, nil)
	r.Header.Set("X-API-Key", testAPIKey)
	r.SetPathValue("note_id", note.ID.String())

	result := hdl.convertNoteToTask(ctx, r)
	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	data, _, err := result.Encode()
	if err != nil {
		t.Fatalf("Failed to encode result: %v", err)
	}

	var responseTask taskapp.Task
	if err := json.Unmarshal(data, &responseTask); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if responseTask.Title != "First line" {
		t.Errorf("Expected title 'First line', got '%s'", responseTask.Title)
	}
	if responseTask.Description != "Second line" {
		t.Errorf("Expected description 'Second line', got '%s'", responseTask.Description)
	}
	if responseTask.Status != "open" {
		t.Errorf("Expected status 'open', got '%s'", responseTask.Status)
	}
	if !responseTask.Unconfirmed {
		t.Error("Expected unconfirmed=true")
	}
}

func TestConvertTaskToNoteWithRecurrence(t *testing.T) {
	db := dbtest.New(t, "TestConvertTaskToNoteWithRecurrence")
	ctx := context.Background()

	// Create a recurring task (should fail conversion)
	recurrenceRule := "FREQ=DAILY"
	task, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title:          "Recurring Task",
		Status:         taskstatus.Open,
		Priority:       taskpriority.Medium,
		RecurrenceRule: &recurrenceRule,
	})
	if err != nil {
		t.Fatalf("Failed to create recurring task: %v", err)
	}

	// Setup handler
	cfg := mux.Config{
		Log:    db.Log,
		DB:     db.DB,
		APIKey: testAPIKey,
	}

	reclassifyBus := db.BusDomain.Reclassify
	hdl := &app{log: cfg.Log, reclassifyBus: reclassifyBus}

	// Test should return error
	url := fmt.Sprintf("/api/v1/tasks/%s/convert-to-note", task.ID.String())
	r := httptest.NewRequest(http.MethodPost, url, nil)
	r.Header.Set("X-API-Key", testAPIKey)
	r.SetPathValue("task_id", task.ID.String())

	result := hdl.convertTaskToNote(ctx, r)
	if result == nil {
		t.Fatal("Expected error result, got nil")
	}

	// Verify it's an error response
	data, _, _ := result.Encode()
	var errMap map[string]interface{}
	if err := json.Unmarshal(data, &errMap); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	// Should have an error response (errs.Error serializes with "message" and "code")
	if errMap["message"] == nil && errMap["code"] == nil {
		t.Error("Expected error in response for recurring task conversion")
	}
}

func TestConvertTaskNotFound(t *testing.T) {
	db := dbtest.New(t, "TestConvertTaskNotFound")
	ctx := context.Background()

	// Setup handler with non-existent task ID
	cfg := mux.Config{
		Log:    db.Log,
		DB:     db.DB,
		APIKey: testAPIKey,
	}

	reclassifyBus := db.BusDomain.Reclassify
	hdl := &app{log: cfg.Log, reclassifyBus: reclassifyBus}

	nonExistentID := uuid.New()
	url := fmt.Sprintf("/api/v1/tasks/%s/convert-to-note", nonExistentID.String())
	r := httptest.NewRequest(http.MethodPost, url, nil)
	r.Header.Set("X-API-Key", testAPIKey)
	r.SetPathValue("task_id", nonExistentID.String())

	result := hdl.convertTaskToNote(ctx, r)
	if result == nil {
		t.Fatal("Expected error result, got nil")
	}

	// Should return a NotFound error
	data, _, _ := result.Encode()
	var errMap map[string]interface{}
	if err := json.Unmarshal(data, &errMap); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	// Check if error is present (errs.Error serializes with "message" and "code")
	if errMap["message"] == nil && errMap["code"] == nil {
		t.Error("Expected NotFound error for non-existent task")
	}
}
