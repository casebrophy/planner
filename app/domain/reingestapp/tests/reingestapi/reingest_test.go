package reingestapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/casebrophy/planner/app/domain/reingestapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/rawinputbus/stores/rawinputdb"
	"github.com/casebrophy/planner/business/types/rawinputstatus"
)

func Test_Reingest(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "Test_Reingest")

	sd, err := insertSeedData(test.DB)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	riStore := rawinputdb.NewStore(test.DB.Log, test.DB.DB)
	riBus := rawinputbus.NewBusiness(test.DB.Log, riStore)

	post := func(path, apiKey string) (int, reingestapp.ReingestResponse) {
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader([]byte{}))
		req.Header.Set("X-API-Key", apiKey)
		rec := httptest.NewRecorder()
		test.ServeHTTP(rec, req)
		var resp reingestapp.ReingestResponse
		if rec.Code == http.StatusOK {
			_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		}
		return rec.Code, resp
	}

	t.Run("linked-task-skip-classify-true", func(t *testing.T) {
		code, resp := post("/api/v1/tasks/"+sd.linkedTask.ID.String()+"/reingest", apitest.TestAPIKey)
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}
		if !resp.SkipClassify {
			t.Error("expected skipClassify=true for linked task")
		}
		if !resp.Enqueued {
			t.Error("expected enqueued=true")
		}
		if resp.RawInputID != sd.linkedTask.RawInputID.String() {
			t.Errorf("expected rawInputId=%s, got %s", sd.linkedTask.RawInputID.String(), resp.RawInputID)
		}
		ri, err := riBus.QueryByID(context.Background(), *sd.linkedTask.RawInputID)
		if err != nil {
			t.Fatalf("query ri: %s", err)
		}
		if ri.Status != rawinputstatus.Pending {
			t.Errorf("expected status=pending, got %s", ri.Status)
		}
		if !ri.SkipClassify {
			t.Error("expected skip_classify=true after linked reingest")
		}
		if !ri.ReingestMode {
			t.Error("expected reingest_mode=true after linked reingest")
		}
	})

	t.Run("unlinked-task-skip-classify-false", func(t *testing.T) {
		code, resp := post("/api/v1/tasks/"+sd.unlinkedTask.ID.String()+"/reingest", apitest.TestAPIKey)
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}
		if resp.SkipClassify {
			t.Error("expected skipClassify=false for unlinked task")
		}
		if !resp.Enqueued {
			t.Error("expected enqueued=true")
		}
		ri, err := riBus.QueryByID(context.Background(), *sd.unlinkedTask.RawInputID)
		if err != nil {
			t.Fatalf("query ri: %s", err)
		}
		if ri.Status != rawinputstatus.Pending {
			t.Errorf("expected status=pending, got %s", ri.Status)
		}
		if ri.SkipClassify {
			t.Error("expected skip_classify=false for unlinked reingest")
		}
		if !ri.ReingestMode {
			t.Error("expected reingest_mode=true to preserve confirmed state")
		}
	})

	t.Run("no-ri-task-400", func(t *testing.T) {
		code, _ := post("/api/v1/tasks/"+sd.noRITask.ID.String()+"/reingest", apitest.TestAPIKey)
		if code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", code)
		}
	})

	t.Run("auth-failure-401", func(t *testing.T) {
		code, _ := post("/api/v1/tasks/"+sd.linkedTask.ID.String()+"/reingest", "")
		if code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", code)
		}
	})

	t.Run("linked-note-skip-classify-true", func(t *testing.T) {
		code, resp := post("/api/v1/notes/"+sd.linkedNote.ID.String()+"/reingest", apitest.TestAPIKey)
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}
		if !resp.SkipClassify {
			t.Error("expected skipClassify=true for linked note")
		}
		if !resp.Enqueued {
			t.Error("expected enqueued=true")
		}
		ri, err := riBus.QueryByID(context.Background(), *sd.linkedNote.RawInputID)
		if err != nil {
			t.Fatalf("query ri: %s", err)
		}
		if ri.Status != rawinputstatus.Pending {
			t.Errorf("expected status=pending, got %s", ri.Status)
		}
	})

	t.Run("linked-event-skip-classify-true", func(t *testing.T) {
		code, resp := post("/api/v1/events/"+sd.linkedEvent.ID.String()+"/reingest", apitest.TestAPIKey)
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}
		if !resp.SkipClassify {
			t.Error("expected skipClassify=true for linked event")
		}
		if !resp.Enqueued {
			t.Error("expected enqueued=true")
		}
		ri, err := riBus.QueryByID(context.Background(), *sd.linkedEvent.RawInputID)
		if err != nil {
			t.Fatalf("query ri: %s", err)
		}
		if ri.Status != rawinputstatus.Pending {
			t.Errorf("expected status=pending, got %s", ri.Status)
		}
	})

	t.Run("unlinked-event-skip-classify-false", func(t *testing.T) {
		code, resp := post("/api/v1/events/"+sd.unlinkedEvent.ID.String()+"/reingest", apitest.TestAPIKey)
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}
		if resp.SkipClassify {
			t.Error("expected skipClassify=false for unlinked event")
		}
		if !resp.Enqueued {
			t.Error("expected enqueued=true")
		}
		ri, err := riBus.QueryByID(context.Background(), *sd.unlinkedEvent.RawInputID)
		if err != nil {
			t.Fatalf("query ri: %s", err)
		}
		if ri.Status != rawinputstatus.Pending {
			t.Errorf("expected status=pending, got %s", ri.Status)
		}
		if ri.SkipClassify {
			t.Error("expected skip_classify=false for unlinked event reingest")
		}
		if !ri.ReingestMode {
			t.Error("expected reingest_mode=true to preserve confirmed state")
		}
	})
}

func Test_BulkReingest(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "Test_BulkReingest")

	sd, err := insertSeedData(test.DB)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	riStore := rawinputdb.NewStore(test.DB.Log, test.DB.DB)
	riBus := rawinputbus.NewBusiness(test.DB.Log, riStore)

	postBulk := func(body interface{}, apiKey string) (int, reingestapp.BulkReingestResponse) {
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/reingest/bulk", bytes.NewReader(bodyBytes))
		req.Header.Set("X-API-Key", apiKey)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		test.ServeHTTP(rec, req)
		var resp reingestapp.BulkReingestResponse
		if rec.Code == http.StatusOK {
			_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		}
		return rec.Code, resp
	}

	t.Run("bulk-reingest-tasks-all", func(t *testing.T) {
		code, resp := postBulk(map[string]interface{}{"entityType": "task"}, apitest.TestAPIKey)
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}
		// linkedTask, unlinkedTask, noRITask (has no RI, so skipped)
		if resp.Queued != 2 {
			t.Errorf("expected queued=2, got %d", resp.Queued)
		}
	})

	t.Run("bulk-reingest-tasks-by-context", func(t *testing.T) {
		code, resp := postBulk(map[string]interface{}{
			"entityType": "task",
			"contextId":  sd.ctx.ID.String(),
		}, apitest.TestAPIKey)
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}
		// Only linkedTask has the context
		if resp.Queued != 1 {
			t.Errorf("expected queued=1, got %d", resp.Queued)
		}
	})

	t.Run("bulk-reingest-notes-all", func(t *testing.T) {
		code, resp := postBulk(map[string]interface{}{"entityType": "note"}, apitest.TestAPIKey)
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}
		// linkedNote (context_id is required for notes, so no unlinked notes)
		if resp.Queued != 1 {
			t.Errorf("expected queued=1, got %d", resp.Queued)
		}
	})

	t.Run("bulk-reingest-notes-by-context", func(t *testing.T) {
		code, resp := postBulk(map[string]interface{}{
			"entityType": "note",
			"contextId":  sd.ctx.ID.String(),
		}, apitest.TestAPIKey)
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}
		if resp.Queued != 1 {
			t.Errorf("expected queued=1, got %d", resp.Queued)
		}
	})

	t.Run("bulk-reingest-events-all", func(t *testing.T) {
		code, resp := postBulk(map[string]interface{}{"entityType": "event"}, apitest.TestAPIKey)
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}
		// linkedEvent, unlinkedEvent
		if resp.Queued != 2 {
			t.Errorf("expected queued=2, got %d", resp.Queued)
		}
	})

	t.Run("bulk-reingest-events-by-context", func(t *testing.T) {
		code, resp := postBulk(map[string]interface{}{
			"entityType": "event",
			"contextId":  sd.ctx.ID.String(),
		}, apitest.TestAPIKey)
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}
		// Only linkedEvent has the context
		if resp.Queued != 1 {
			t.Errorf("expected queued=1, got %d", resp.Queued)
		}
	})

	t.Run("bulk-reingest-invalid-entity-type", func(t *testing.T) {
		code, _ := postBulk(map[string]interface{}{"entityType": "invalid"}, apitest.TestAPIKey)
		if code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", code)
		}
	})

	t.Run("bulk-reingest-missing-entity-type", func(t *testing.T) {
		code, _ := postBulk(map[string]interface{}{}, apitest.TestAPIKey)
		if code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", code)
		}
	})

	t.Run("bulk-reingest-invalid-context-id-format", func(t *testing.T) {
		code, _ := postBulk(map[string]interface{}{
			"entityType": "task",
			"contextId":  "invalid-uuid",
		}, apitest.TestAPIKey)
		if code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", code)
		}
	})

	t.Run("bulk-reingest-auth-failure-401", func(t *testing.T) {
		code, _ := postBulk(map[string]interface{}{"entityType": "task"}, "")
		if code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", code)
		}
	})

	// Verify raw_input state after bulk reingest
	t.Run("verify-raw-input-state-after-bulk-reingest", func(t *testing.T) {
		// Reingest all tasks
		code, resp := postBulk(map[string]interface{}{"entityType": "task"}, apitest.TestAPIKey)
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}
		if resp.Queued != 2 {
			t.Errorf("expected queued=2, got %d", resp.Queued)
		}

		// Check linkedTask raw_input state
		ri, err := riBus.QueryByID(context.Background(), *sd.linkedTask.RawInputID)
		if err != nil {
			t.Fatalf("query linkedTask raw_input: %s", err)
		}
		if ri.Status != rawinputstatus.Pending {
			t.Errorf("expected linkedTask status=pending, got %s", ri.Status)
		}
		if !ri.SkipClassify {
			t.Error("expected linkedTask skip_classify=true")
		}
		if !ri.ReingestMode {
			t.Error("expected linkedTask reingest_mode=true")
		}

		// Check unlinkedTask raw_input state
		ri, err = riBus.QueryByID(context.Background(), *sd.unlinkedTask.RawInputID)
		if err != nil {
			t.Fatalf("query unlinkedTask raw_input: %s", err)
		}
		if ri.Status != rawinputstatus.Pending {
			t.Errorf("expected unlinkedTask status=pending, got %s", ri.Status)
		}
		if ri.SkipClassify {
			t.Error("expected unlinkedTask skip_classify=false")
		}
		if !ri.ReingestMode {
			t.Error("expected unlinkedTask reingest_mode=true")
		}
	})
}
