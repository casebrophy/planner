package mlclient

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/casebrophy/planner/foundation/logger"
)

func newTestLogger() *logger.Logger {
	return logger.New(io.Discard, slog.LevelError, "test")
}

func TestClusterTasks_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/cluster" {
			t.Errorf("expected /cluster, got %s", r.URL.Path)
		}

		var req clusterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if len(req.Tasks) != 1 {
			t.Errorf("expected 1 task, got %d", len(req.Tasks))
		}
		if req.Tasks[0].ID != "task-1" {
			t.Errorf("expected task-1, got %s", req.Tasks[0].ID)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(clusterResponse{
			Results: []ClusterResult{
				{TaskID: "task-1", ClusterID: "cluster-1"},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, newTestLogger())
	tasks := []TaskInput{
		{
			ID:          "task-1",
			Title:       "Test Task",
			Description: "A test task",
			Status:      "active",
			Priority:    "high",
		},
	}

	results, err := client.ClusterTasks(context.Background(), tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if results[0].TaskID != "task-1" {
		t.Errorf("expected task-1, got %s", results[0].TaskID)
	}
	if results[0].ClusterID != "cluster-1" {
		t.Errorf("expected cluster-1, got %s", results[0].ClusterID)
	}
}

func TestClusterTasks_Stub(t *testing.T) {
	client := NewClient("http://unavailable:9999", newTestLogger())
	tasks := []TaskInput{
		{ID: "task-1", Title: "Test", Description: "Test", Status: "active", Priority: "high"},
	}

	_, err := client.ClusterTasks(context.Background(), tasks)
	if err != ErrServiceUnavailable {
		t.Errorf("expected ErrServiceUnavailable, got %v", err)
	}
}

func TestFindSimilarSituations_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/similar" {
			t.Errorf("expected /similar, got %s", r.URL.Path)
		}

		var req similarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.ContextID != "ctx-1" {
			t.Errorf("expected ctx-1, got %s", req.ContextID)
		}
		if len(req.Embeddings) != 1 {
			t.Errorf("expected 1 embedding, got %d", len(req.Embeddings))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(similarResponse{
			Matches: []SimilarMatch{
				{ContextID: "ctx-2", Similarity: 0.95, Description: "Similar situation"},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, newTestLogger())
	embeddings := [][]float32{{0.1, 0.2, 0.3}}

	matches, err := client.FindSimilarSituations(context.Background(), "ctx-1", embeddings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matches) != 1 {
		t.Errorf("expected 1 match, got %d", len(matches))
	}
	if matches[0].ContextID != "ctx-2" {
		t.Errorf("expected ctx-2, got %s", matches[0].ContextID)
	}
	if matches[0].Similarity != 0.95 {
		t.Errorf("expected 0.95, got %f", matches[0].Similarity)
	}
	if matches[0].Description != "Similar situation" {
		t.Errorf("expected 'Similar situation', got %s", matches[0].Description)
	}
}

func TestFindSimilarSituations_Stub(t *testing.T) {
	client := NewClient("http://unavailable:9999", newTestLogger())
	embeddings := [][]float32{{0.1, 0.2, 0.3}}

	_, err := client.FindSimilarSituations(context.Background(), "ctx-1", embeddings)
	if err != ErrServiceUnavailable {
		t.Errorf("expected ErrServiceUnavailable, got %v", err)
	}
}

func TestHealth_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/health" {
			t.Errorf("expected /health, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, newTestLogger())
	err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHealth_Stub(t *testing.T) {
	client := NewClient("http://unavailable:9999", newTestLogger())

	err := client.Health(context.Background())
	if err != ErrServiceUnavailable {
		t.Errorf("expected ErrServiceUnavailable, got %v", err)
	}
}

func TestClusterTasks_RequestBodyValidation(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capturedBody = body

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(clusterResponse{Results: []ClusterResult{}})
	}))
	defer server.Close()

	client := NewClient(server.URL, newTestLogger())
	tasks := []TaskInput{
		{
			ID:          "test-id",
			Title:       "Test Title",
			Description: "Test Desc",
			Status:      "pending",
			Priority:    "medium",
		},
	}

	client.ClusterTasks(context.Background(), tasks)

	var req clusterRequest
	if err := json.Unmarshal(capturedBody, &req); err != nil {
		t.Fatalf("failed to unmarshal captured body: %v", err)
	}

	if req.Tasks[0].ID != "test-id" {
		t.Errorf("expected test-id, got %s", req.Tasks[0].ID)
	}
	if req.Tasks[0].Title != "Test Title" {
		t.Errorf("expected 'Test Title', got %s", req.Tasks[0].Title)
	}
}

func TestClusterTasks_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer server.Close()

	client := NewClient(server.URL, newTestLogger())
	tasks := []TaskInput{{ID: "1", Title: "t", Description: "d", Status: "s", Priority: "p"}}

	_, err := client.ClusterTasks(context.Background(), tasks)
	if err != ErrServiceUnavailable {
		t.Errorf("expected ErrServiceUnavailable, got %v", err)
	}
}

func TestFindSimilarSituations_RequestBodyValidation(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capturedBody = body

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(similarResponse{Matches: []SimilarMatch{}})
	}))
	defer server.Close()

	client := NewClient(server.URL, newTestLogger())
	embeddings := [][]float32{{0.5, 0.6, 0.7}}

	client.FindSimilarSituations(context.Background(), "context-123", embeddings)

	var req similarRequest
	if err := json.Unmarshal(capturedBody, &req); err != nil {
		t.Fatalf("failed to unmarshal captured body: %v", err)
	}

	if req.ContextID != "context-123" {
		t.Errorf("expected context-123, got %s", req.ContextID)
	}
	if len(req.Embeddings) != 1 || len(req.Embeddings[0]) != 3 {
		t.Errorf("expected 1 embedding with 3 values, got %d embeddings", len(req.Embeddings))
	}
}
