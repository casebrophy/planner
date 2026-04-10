package embed

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaEmbedder_SingleText(t *testing.T) {
	embeddings := [][]float32{
		{0.1, 0.2, 0.3, 0.4, 0.5},
	}
	dims := 5

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := ollamaEmbedResponse{Embeddings: embeddings}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer srv.Close()

	embedder := NewOllamaEmbedder(srv.URL, "nomic-embed-text", dims)
	got, err := embedder.Embed(context.Background(), []string{"hello world"})
	if err != nil {
		t.Fatalf("Embed: unexpected error: %v", err)
	}

	if len(got) != 1 {
		t.Errorf("Embed: got %d vectors, want 1", len(got))
	}
	if embedder.Dimensions() != dims {
		t.Errorf("Dimensions: got %d, want %d", embedder.Dimensions(), dims)
	}
	for i, v := range got[0] {
		if v != embeddings[0][i] {
			t.Errorf("Embed[0][%d]: got %v, want %v", i, v, embeddings[0][i])
		}
	}
}

func TestOllamaEmbedder_MultipleTexts(t *testing.T) {
	embeddings := [][]float32{
		{0.1, 0.2, 0.3},
		{0.4, 0.5, 0.6},
		{0.7, 0.8, 0.9},
	}
	dims := 3

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := ollamaEmbedResponse{Embeddings: embeddings}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer srv.Close()

	embedder := NewOllamaEmbedder(srv.URL, "nomic-embed-text", dims)
	got, err := embedder.Embed(context.Background(), []string{"text1", "text2", "text3"})
	if err != nil {
		t.Fatalf("Embed: unexpected error: %v", err)
	}

	if len(got) != 3 {
		t.Errorf("Embed: got %d vectors, want 3", len(got))
	}
	if embedder.Dimensions() != dims {
		t.Errorf("Dimensions: got %d, want %d", embedder.Dimensions(), dims)
	}
}

func TestOllamaEmbedder_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	embedder := NewOllamaEmbedder(srv.URL, "nomic-embed-text", 5)
	_, err := embedder.Embed(context.Background(), []string{"hello world"})
	if err == nil {
		t.Fatal("Embed: expected error for HTTP 500, got nil")
	}
}
