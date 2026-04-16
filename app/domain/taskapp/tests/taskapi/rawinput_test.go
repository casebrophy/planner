package taskapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/domain/taskapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/rawinputbus/stores/rawinputdb"
	"github.com/casebrophy/planner/business/types/rawinputsource"
)

func TestManualTaskCreate_ProducesRawInput(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "TestManualTaskCreate_ProducesRawInput")

	input := taskapp.NewTask{
		Title:    "RI Test Task",
		Priority: "medium",
		Energy:   "low",
	}
	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apitest.TestAPIKey)
	rec := httptest.NewRecorder()
	test.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var got taskapp.Task
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode task response: %s", err)
	}

	if got.RawInputID == nil {
		t.Fatal("expected task.RawInputID to be set")
	}
	riID, err := uuid.Parse(*got.RawInputID)
	if err != nil {
		t.Fatalf("parse raw_input_id: %s", err)
	}
	taskID, err := uuid.Parse(got.ID)
	if err != nil {
		t.Fatalf("parse task id: %s", err)
	}

	riStore := rawinputdb.NewStore(test.DB.Log, test.DB.DB)
	riBus := rawinputbus.NewBusiness(test.DB.Log, riStore)

	ri, err := riBus.QueryByID(context.Background(), riID)
	if err != nil {
		t.Fatalf("query raw_input: %s", err)
	}

	if ri.SourceType != rawinputsource.Manual {
		t.Errorf("expected source_type=manual, got %s", ri.SourceType)
	}
	if !ri.SkipClassify {
		t.Error("expected skip_classify=true")
	}
	if ri.SourceEntityID == nil || *ri.SourceEntityID != taskID {
		t.Errorf("expected source_entity_id=%s, got %v", taskID, ri.SourceEntityID)
	}
	if ri.SourceEntityKind != "task" {
		t.Errorf("expected source_entity_kind=task, got %s", ri.SourceEntityKind)
	}
}
