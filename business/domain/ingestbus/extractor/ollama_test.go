package extractor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// makeOllamaServer creates a test HTTP server that wraps the given value as an
// Ollama generate response. The value is first marshaled to JSON (the inner
// extraction payload), then that JSON string is embedded in the outer Ollama
// envelope {"response": "<json-string>", "done": true}.
func makeOllamaServer(t *testing.T, inner any) *httptest.Server {
	t.Helper()

	innerBytes, err := json.Marshal(inner)
	if err != nil {
		t.Fatalf("makeOllamaServer: marshal inner: %v", err)
	}

	envelope := ollamaGenerateResponse{
		Response: string(innerBytes),
		Done:     true,
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(envelope); err != nil {
			t.Errorf("makeOllamaServer: encode response: %v", err)
		}
	}))
}

func TestOllamaExtractEmail_Success(t *testing.T) {
	want := EmailExtraction{
		Summary:      "Test email summary",
		SenderName:   "Alice",
		SenderDomain: "example.com",
		Sentiment:    "positive",
	}

	srv := makeOllamaServer(t, want)
	defer srv.Close()

	ex := NewOllamaExtractor(srv.URL, "llama3")
	got, err := ex.ExtractEmail(context.Background(), "Hello", "body text", "alice@example.com", nil)
	if err != nil {
		t.Fatalf("ExtractEmail: unexpected error: %v", err)
	}

	if got.Summary != want.Summary {
		t.Errorf("Summary: got %q, want %q", got.Summary, want.Summary)
	}
	if got.ContextConfidence != 0.85 {
		t.Errorf("ContextConfidence: got %v, want 0.85", got.ContextConfidence)
	}
}

func TestOllamaExtractText_Success(t *testing.T) {
	want := TextExtraction{
		Summary: "Test text summary",
	}

	srv := makeOllamaServer(t, want)
	defer srv.Close()

	ex := NewOllamaExtractor(srv.URL, "llama3")
	got, err := ex.ExtractText(context.Background(), "some voice capture text", nil, "")
	if err != nil {
		t.Fatalf("ExtractText: unexpected error: %v", err)
	}

	if got.Summary != want.Summary {
		t.Errorf("Summary: got %q, want %q", got.Summary, want.Summary)
	}
	if got.ContextConfidence != 0.85 {
		t.Errorf("ContextConfidence: got %v, want 0.85", got.ContextConfidence)
	}
}

func TestOllamaExtractEmail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	ex := NewOllamaExtractor(srv.URL, "llama3")
	_, err := ex.ExtractEmail(context.Background(), "Hello", "body text", "alice@example.com", nil)
	if err == nil {
		t.Fatal("ExtractEmail: expected error for HTTP 500, got nil")
	}
}

func TestOllamaExtractEmail_MalformedJSON(t *testing.T) {
	// Outer envelope is valid, but inner response string is not valid JSON.
	envelope := ollamaGenerateResponse{
		Response: "not json at all",
		Done:     true,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(envelope); err != nil {
			t.Errorf("encode: %v", err)
		}
	}))
	defer srv.Close()

	ex := NewOllamaExtractor(srv.URL, "llama3")
	_, err := ex.ExtractEmail(context.Background(), "Hello", "body text", "alice@example.com", nil)
	if err == nil {
		t.Fatal("ExtractEmail: expected error for malformed inner JSON, got nil")
	}
}
