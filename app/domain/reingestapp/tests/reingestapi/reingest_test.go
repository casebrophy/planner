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
