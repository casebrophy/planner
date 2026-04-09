package extractor

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/casebrophy/planner/foundation/logger"
)

func testLogger() *logger.Logger {
	return logger.New(os.Stdout, logger.LevelInfo, "test")
}

// TestFailover_ClaudeSuccess — claude returns result, ollama mock set to return sentinel error.
// Assert claude result returned without error; sentinel ensures ollama error would surface if called.
func TestFailover_ClaudeSuccess(t *testing.T) {
	want := EmailExtraction{Summary: "hello from claude"}
	sentinelErr := errors.New("ollama: should not be called")

	claude := &MockExtractor{Result: want}
	ollama := &MockExtractor{Err: sentinelErr}

	f := newFailoverExtractorForTest(testLogger(), claude, ollama)

	got, err := f.ExtractEmail(context.Background(), "subj", "body", "from@example.com", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Summary != want.Summary {
		t.Errorf("got summary %q, want %q", got.Summary, want.Summary)
	}
}

// TestFailover_429TriggersOllama_Success — claude returns 429 error, ollama returns result.
func TestFailover_429TriggersOllama_Success(t *testing.T) {
	want := EmailExtraction{Summary: "hello from ollama"}

	claude := &MockExtractor{Err: errors.New("rate limit: 429 too many requests")}
	ollama := &MockExtractor{Result: want}

	f := newFailoverExtractorForTest(testLogger(), claude, ollama)

	got, err := f.ExtractEmail(context.Background(), "subj", "body", "from@example.com", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Summary != want.Summary {
		t.Errorf("got summary %q, want %q", got.Summary, want.Summary)
	}
}

// TestFailover_ContextLimitTriggersOllama — claude returns context limit error, fallback triggered.
func TestFailover_ContextLimitTriggersOllama(t *testing.T) {
	want := EmailExtraction{Summary: "context limit fallback"}

	claude := &MockExtractor{Err: errors.New("context limit exceeded")}
	ollama := &MockExtractor{Result: want}

	f := newFailoverExtractorForTest(testLogger(), claude, ollama)

	got, err := f.ExtractEmail(context.Background(), "subj", "body", "from@example.com", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Summary != want.Summary {
		t.Errorf("got summary %q, want %q", got.Summary, want.Summary)
	}
}

// TestFailover_ConnectionRefusedTriggersOllama — claude returns connection refused, fallback triggered.
func TestFailover_ConnectionRefusedTriggersOllama(t *testing.T) {
	want := EmailExtraction{Summary: "connection refused fallback"}

	claude := &MockExtractor{Err: errors.New("connection refused")}
	ollama := &MockExtractor{Result: want}

	f := newFailoverExtractorForTest(testLogger(), claude, ollama)

	got, err := f.ExtractEmail(context.Background(), "subj", "body", "from@example.com", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Summary != want.Summary {
		t.Errorf("got summary %q, want %q", got.Summary, want.Summary)
	}
}

// TestFailover_400DoesNotTriggerOllama — claude returns 400, ollama set to sentinel error.
// Assert claude error returned, NOT ollama sentinel.
func TestFailover_400DoesNotTriggerOllama(t *testing.T) {
	claudeErr := errors.New("400 bad request: malformed input")
	sentinelErr := errors.New("ollama: should not be called")

	claude := &MockExtractor{Err: claudeErr}
	ollama := &MockExtractor{Err: sentinelErr}

	f := newFailoverExtractorForTest(testLogger(), claude, ollama)

	_, err := f.ExtractEmail(context.Background(), "subj", "body", "from@example.com", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, sentinelErr) {
		t.Error("ollama should not have been called for 400 error")
	}
	if err.Error() != claudeErr.Error() {
		t.Errorf("got error %q, want %q", err.Error(), claudeErr.Error())
	}
}

// TestFailover_OllamaAlsoFails — claude returns 429, ollama also fails.
// Assert ollama error returned.
func TestFailover_OllamaAlsoFails(t *testing.T) {
	ollamaErr := errors.New("ollama: connection refused")

	claude := &MockExtractor{Err: errors.New("429 rate limit")}
	ollama := &MockExtractor{Err: ollamaErr}

	f := newFailoverExtractorForTest(testLogger(), claude, ollama)

	_, err := f.ExtractEmail(context.Background(), "subj", "body", "from@example.com", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ollamaErr) {
		t.Errorf("expected ollama error, got: %v", err)
	}
}

// TestFailover_ExtractText_FallbackWorks — claude returns timeout, ollama returns text result.
func TestFailover_ExtractText_FallbackWorks(t *testing.T) {
	want := TextExtraction{Summary: "text from ollama"}

	claude := &MockExtractor{Err: errors.New("timeout: connection timed out")}
	ollama := &MockExtractor{TextResult: want}

	f := newFailoverExtractorForTest(testLogger(), claude, ollama)

	got, err := f.ExtractText(context.Background(), "some text", nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Summary != want.Summary {
		t.Errorf("got summary %q, want %q", got.Summary, want.Summary)
	}
}
