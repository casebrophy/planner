# Ollama Fallback Extractor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a local Ollama model as a fallback extraction path so classification continues working when Claude hits rate limits (429), context limits, or API downtime.

**Architecture:** A new `OllamaExtractor` posts to a local Ollama `/api/generate` endpoint and unmarshals the response into the same `EmailExtraction`/`TextExtraction` types. A `FailoverExtractor` wraps both extractors and routes to Ollama only on specific error conditions (429, context limit, connection failure). The `Extractor` interface is unchanged; only the wiring in `classifyapp/route.go` and `main.go` (SMTP path) changes.

**Tech Stack:** Go stdlib `net/http`, `net/http/httptest` (tests), `github.com/casebrophy/planner/foundation/logger`

---

## File Map

| Action | File | Responsibility |
|--------|------|----------------|
| CREATE | `business/domain/ingestbus/extractor/ollama.go` | HTTP client to local Ollama `/api/generate` |
| CREATE | `business/domain/ingestbus/extractor/ollama_test.go` | Unit tests for OllamaExtractor (httptest server) |
| CREATE | `business/domain/ingestbus/extractor/failover.go` | FailoverExtractor + `isFallbackError` logic |
| CREATE | `business/domain/ingestbus/extractor/failover_test.go` | Unit tests for FailoverExtractor (MockExtractor injection) |
| MODIFY | `app/sdk/mux/mux.go` | Add `OllamaURL`, `OllamaModel`, `OllamaEnabled` to `Config` |
| MODIFY | `api/services/planner/main.go` | Add `Ollama` conf struct, compute `ollamaEnabled`, update muxCfg, update SMTP extractor |
| MODIFY | `app/domain/classifyapp/route.go` | Replace bare `ClaudeCodeExtractor` with failover-aware construction |

> **Out of scope (same pattern applies):** `rawinputapp/route.go`, `voiceingestapp/route.go`, and `mcpapp/route.go` also instantiate `ClaudeCodeExtractor`. They can be updated identically in a follow-up once this is validated.

---

### Task 1: OllamaExtractor

**Files:**
- Create: `business/domain/ingestbus/extractor/ollama.go`
- Create: `business/domain/ingestbus/extractor/ollama_test.go`

- [ ] **Step 1: Write failing tests**

Create `business/domain/ingestbus/extractor/ollama_test.go`:

```go
package extractor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ollamaEnvelope matches the JSON shape returned by Ollama /api/generate (non-streaming).
type testOllamaEnvelope struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func makeOllamaServer(t *testing.T, inner any) *httptest.Server {
	t.Helper()
	payload, err := json.Marshal(inner)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := json.Marshal(testOllamaEnvelope{Response: string(payload), Done: true})
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(envelope)
	}))
}

func TestOllamaExtractEmail_Success(t *testing.T) {
	want := EmailExtraction{
		Summary:                  "Test summary",
		SenderName:               "Alice",
		SenderDomain:             "example.com",
		ActionItems:              []ActionItem{{Title: "Do it", Description: "Do the thing", Priority: "medium"}},
		Deadlines:                []Deadline{},
		SuggestedContextKeywords: []string{"work"},
		Sentiment:                "neutral",
	}

	srv := makeOllamaServer(t, want)
	defer srv.Close()

	ext := NewOllamaExtractor(srv.URL, "llama3")
	got, err := ext.ExtractEmail(context.Background(), "Subject", "Body", "alice@example.com", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Summary != want.Summary {
		t.Errorf("Summary: got %q want %q", got.Summary, want.Summary)
	}
	if got.ContextConfidence != 0.85 {
		t.Errorf("ContextConfidence: got %f want 0.85", got.ContextConfidence)
	}
}

func TestOllamaExtractText_Success(t *testing.T) {
	want := TextExtraction{
		Summary:                  "Voice note summary",
		ActionItems:              []ActionItem{{Title: "Buy milk", Description: "Get milk from store", Priority: "low"}},
		Deadlines:                []Deadline{},
		Events:                   []ExtractedEvent{},
		SuggestedContextKeywords: []string{"errands"},
	}

	srv := makeOllamaServer(t, want)
	defer srv.Close()

	ext := NewOllamaExtractor(srv.URL, "llama3")
	got, err := ext.ExtractText(context.Background(), "Buy milk from the store", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Summary != want.Summary {
		t.Errorf("Summary: got %q want %q", got.Summary, want.Summary)
	}
	if got.ContextConfidence != 0.85 {
		t.Errorf("ContextConfidence: got %f want 0.85", got.ContextConfidence)
	}
}

func TestOllamaExtractEmail_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	ext := NewOllamaExtractor(srv.URL, "llama3")
	_, err := ext.ExtractEmail(context.Background(), "Subject", "Body", "alice@example.com", nil)
	if err == nil {
		t.Fatal("expected error for HTTP 500, got nil")
	}
}

func TestOllamaExtractEmail_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Valid outer envelope but inner response is not valid JSON for EmailExtraction.
		envelope := testOllamaEnvelope{Response: "not json at all", Done: true}
		json.NewEncoder(w).Encode(envelope)
	}))
	defer srv.Close()

	ext := NewOllamaExtractor(srv.URL, "llama3")
	_, err := ext.ExtractEmail(context.Background(), "Subject", "Body", "alice@example.com", nil)
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/casebrophy/personal/planner
go test ./business/domain/ingestbus/extractor/... -run TestOllama -count=1 -v
```

Expected: compile error — `NewOllamaExtractor` undefined.

- [ ] **Step 3: Implement OllamaExtractor**

Create `business/domain/ingestbus/extractor/ollama.go`:

```go
package extractor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ollamaGenerateRequest is the request body for Ollama /api/generate.
type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format"`
}

// ollamaGenerateResponse is the non-streaming response envelope from Ollama /api/generate.
// The model output is in the Response field as a JSON string.
type ollamaGenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// OllamaExtractor implements Extractor using a local Ollama instance.
type OllamaExtractor struct {
	url    string
	model  string
	client *http.Client
}

// NewOllamaExtractor creates an extractor that calls the Ollama /api/generate endpoint.
// url is the base URL of the Ollama instance (e.g. "http://localhost:11434").
// model is the Ollama model name (e.g. "llama3").
func NewOllamaExtractor(url, model string) *OllamaExtractor {
	return &OllamaExtractor{
		url:   url,
		model: model,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// generate calls Ollama and returns the raw model output string.
func (e *OllamaExtractor) generate(ctx context.Context, prompt string) (string, error) {
	reqBody := ollamaGenerateRequest{
		Model:  e.model,
		Prompt: prompt,
		Stream: false,
		Format: "json",
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("ollama: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.url+"/api/generate", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("ollama: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama: unexpected status %d", resp.StatusCode)
	}

	var ollamaResp ollamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("ollama: decode response envelope: %w", err)
	}

	return ollamaResp.Response, nil
}

// ExtractEmail calls Ollama to extract structured data from an email.
// ContextConfidence is always set to 0.85 for successful Ollama extractions.
func (e *OllamaExtractor) ExtractEmail(ctx context.Context, subject, bodyText, fromAddress string, activeContexts []ContextRef) (EmailExtraction, error) {
	contextsJSON, _ := json.Marshal(activeContexts)
	prompt := BuildEmailExtractionPrompt(fromAddress, subject, bodyText, contextsJSON)

	raw, err := e.generate(ctx, prompt)
	if err != nil {
		return EmailExtraction{}, err
	}

	var extraction EmailExtraction
	if err := json.Unmarshal([]byte(raw), &extraction); err != nil {
		return EmailExtraction{}, fmt.Errorf("ollama: unmarshal email extraction: %w", err)
	}

	extraction.ContextConfidence = 0.85
	return extraction, nil
}

// ExtractText calls Ollama to extract structured data from text/voice input.
// ContextConfidence is always set to 0.85 for successful Ollama extractions.
func (e *OllamaExtractor) ExtractText(ctx context.Context, text string, activeContexts []ContextRef) (TextExtraction, error) {
	contextsJSON, _ := json.Marshal(activeContexts)
	prompt := BuildTextExtractionPrompt(text, contextsJSON, time.Now())

	raw, err := e.generate(ctx, prompt)
	if err != nil {
		return TextExtraction{}, err
	}

	var extraction TextExtraction
	if err := json.Unmarshal([]byte(raw), &extraction); err != nil {
		return TextExtraction{}, fmt.Errorf("ollama: unmarshal text extraction: %w", err)
	}

	extraction.ContextConfidence = 0.85
	return extraction, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./business/domain/ingestbus/extractor/... -run TestOllama -count=1 -v
```

Expected: all 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add business/domain/ingestbus/extractor/ollama.go \
        business/domain/ingestbus/extractor/ollama_test.go
git commit -m "feat: add OllamaExtractor with httptest coverage"
```

---

### Task 2: FailoverExtractor

**Files:**
- Create: `business/domain/ingestbus/extractor/failover.go`
- Create: `business/domain/ingestbus/extractor/failover_test.go`

- [ ] **Step 1: Write failing tests**

Create `business/domain/ingestbus/extractor/failover_test.go`:

```go
package extractor

import (
	"context"
	"errors"
	"testing"

	"github.com/casebrophy/planner/foundation/logger"
	"os"
)

func testLogger() *logger.Logger {
	return logger.New(os.Stdout, logger.LevelInfo, "test")
}

// sentinel errors used to detect which extractor was called.
var (
	errOllamaSentinel = errors.New("ollama was called unexpectedly")
)

func TestFailover_ClaudeSuccess(t *testing.T) {
	wantEmail := EmailExtraction{Summary: "from claude"}
	claude := &MockExtractor{Result: wantEmail}
	ollama := &MockExtractor{Err: errOllamaSentinel}

	f := NewFailoverExtractor(testLogger(), claude, ollama)
	got, err := f.ExtractEmail(context.Background(), "s", "b", "f@x.com", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Summary != wantEmail.Summary {
		t.Errorf("Summary: got %q want %q", got.Summary, wantEmail.Summary)
	}
}

func TestFailover_429TriggersOllama_Success(t *testing.T) {
	wantEmail := EmailExtraction{Summary: "from ollama"}
	claude := &MockExtractor{Err: errors.New("rate limit: 429 too many requests")}
	ollama := &MockExtractor{Result: wantEmail}

	f := NewFailoverExtractor(testLogger(), claude, ollama)
	got, err := f.ExtractEmail(context.Background(), "s", "b", "f@x.com", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Summary != wantEmail.Summary {
		t.Errorf("Summary: got %q want %q", got.Summary, wantEmail.Summary)
	}
}

func TestFailover_ContextLimitTriggersOllama(t *testing.T) {
	claude := &MockExtractor{Err: errors.New("context limit exceeded")}
	ollama := &MockExtractor{Result: EmailExtraction{Summary: "from ollama"}}

	f := NewFailoverExtractor(testLogger(), claude, ollama)
	_, err := f.ExtractEmail(context.Background(), "s", "b", "f@x.com", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFailover_ConnectionRefusedTriggersOllama(t *testing.T) {
	claude := &MockExtractor{Err: errors.New("connection refused")}
	ollama := &MockExtractor{Result: EmailExtraction{Summary: "from ollama"}}

	f := NewFailoverExtractor(testLogger(), claude, ollama)
	_, err := f.ExtractEmail(context.Background(), "s", "b", "f@x.com", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFailover_400DoesNotTriggerOllama(t *testing.T) {
	claudeErr := errors.New("400 bad request: malformed input")
	claude := &MockExtractor{Err: claudeErr}
	ollama := &MockExtractor{Err: errOllamaSentinel}

	f := NewFailoverExtractor(testLogger(), claude, ollama)
	_, err := f.ExtractEmail(context.Background(), "s", "b", "f@x.com", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Must return the Claude error, not the Ollama sentinel.
	if errors.Is(err, errOllamaSentinel) {
		t.Error("Ollama was called for a 400 error — it should not be")
	}
	if !errors.Is(err, claudeErr) {
		t.Errorf("expected Claude error %v, got %v", claudeErr, err)
	}
}

func TestFailover_OllamaAlsoFails(t *testing.T) {
	ollamaErr := errors.New("ollama: request failed: connection refused")
	claude := &MockExtractor{Err: errors.New("429 rate limit")}
	ollama := &MockExtractor{Err: ollamaErr}

	f := NewFailoverExtractor(testLogger(), claude, ollama)
	_, err := f.ExtractEmail(context.Background(), "s", "b", "f@x.com", nil)
	if err == nil {
		t.Fatal("expected error when both fail, got nil")
	}
	if !errors.Is(err, ollamaErr) {
		t.Errorf("expected Ollama error %v, got %v", ollamaErr, err)
	}
}

func TestFailover_ExtractText_FallbackWorks(t *testing.T) {
	wantText := TextExtraction{Summary: "from ollama text"}
	claude := &MockExtractor{Err: errors.New("timeout: connection timed out")}
	ollama := &MockExtractor{TextResult: wantText}

	f := NewFailoverExtractor(testLogger(), claude, ollama)
	got, err := f.ExtractText(context.Background(), "some text", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Summary != wantText.Summary {
		t.Errorf("Summary: got %q want %q", got.Summary, wantText.Summary)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./business/domain/ingestbus/extractor/... -run TestFailover -count=1 -v
```

Expected: compile error — `NewFailoverExtractor` undefined.

- [ ] **Step 3: Implement FailoverExtractor**

Create `business/domain/ingestbus/extractor/failover.go`:

```go
package extractor

import (
	"context"
	"strings"

	"github.com/casebrophy/planner/foundation/logger"
)

// FailoverExtractor tries the primary extractor and falls back to the fallback
// extractor only when specific error conditions indicate Claude is unavailable
// (rate limit, context limit, API downtime). Other errors pass through unchanged.
type FailoverExtractor struct {
	log      *logger.Logger
	primary  Extractor
	fallback Extractor
}

// NewFailoverExtractor creates a FailoverExtractor. The primary is always tried
// first; fallback is used only on triggering errors. Accepts concrete types to
// prevent accidental nesting of FailoverExtractors.
func NewFailoverExtractor(log *logger.Logger, primary *ClaudeCodeExtractor, fallback *OllamaExtractor) *FailoverExtractor {
	return &FailoverExtractor{
		log:      log,
		primary:  primary,
		fallback: fallback,
	}
}

// isFallbackError reports whether err should trigger the Ollama fallback.
// Only rate limits, context limits, and connection failures qualify.
// Bad requests (400), auth failures (401), and server errors (500) do not.
func isFallbackError(err error) bool {
	msg := err.Error()
	if strings.Contains(msg, "429") {
		return true
	}
	if strings.Contains(msg, "context") && strings.Contains(msg, "limit") {
		return true
	}
	if strings.Contains(msg, "connection") || strings.Contains(msg, "timeout") || strings.Contains(msg, "refused") {
		return true
	}
	return false
}

// ExtractEmail tries Claude first; falls back to Ollama on qualifying errors.
func (f *FailoverExtractor) ExtractEmail(ctx context.Context, subject, bodyText, fromAddress string, activeContexts []ContextRef) (EmailExtraction, error) {
	result, err := f.primary.ExtractEmail(ctx, subject, bodyText, fromAddress, activeContexts)
	if err == nil {
		return result, nil
	}

	if !isFallbackError(err) {
		return EmailExtraction{}, err
	}

	f.log.Info(ctx, "extractor", "status", "claude failed, falling back to ollama", "error", err.Error())

	result, err = f.fallback.ExtractEmail(ctx, subject, bodyText, fromAddress, activeContexts)
	if err != nil {
		f.log.Error(ctx, "extractor", "status", "ollama fallback failed", "error", err.Error())
		return EmailExtraction{}, err
	}

	f.log.Info(ctx, "extractor", "status", "ollama fallback succeeded")
	return result, nil
}

// ExtractText tries Claude first; falls back to Ollama on qualifying errors.
func (f *FailoverExtractor) ExtractText(ctx context.Context, text string, activeContexts []ContextRef) (TextExtraction, error) {
	result, err := f.primary.ExtractText(ctx, text, activeContexts)
	if err == nil {
		return result, nil
	}

	if !isFallbackError(err) {
		return TextExtraction{}, err
	}

	f.log.Info(ctx, "extractor", "status", "claude failed, falling back to ollama", "error", err.Error())

	result, err = f.fallback.ExtractText(ctx, text, activeContexts)
	if err != nil {
		f.log.Error(ctx, "extractor", "status", "ollama fallback failed", "error", err.Error())
		return TextExtraction{}, err
	}

	f.log.Info(ctx, "extractor", "status", "ollama fallback succeeded")
	return result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./business/domain/ingestbus/extractor/... -run TestFailover -count=1 -v
```

Expected: all 7 tests PASS.

- [ ] **Step 5: Run all extractor tests together**

```bash
go test ./business/domain/ingestbus/extractor/... -count=1 -v
```

Expected: all tests PASS (Ollama + Failover suites).

- [ ] **Step 6: Commit**

```bash
git add business/domain/ingestbus/extractor/failover.go \
        business/domain/ingestbus/extractor/failover_test.go
git commit -m "feat: add FailoverExtractor with Ollama fallback on rate limit/context/connection errors"
```

---

### Task 3: Config + Wiring

**Files:**
- Modify: `app/sdk/mux/mux.go` (lines 18–25)
- Modify: `api/services/planner/main.go` (lines 82–111, 180–187, 215–234)
- Modify: `app/domain/classifyapp/route.go` (line 33)

- [ ] **Step 1: Add Ollama fields to mux.Config**

In `app/sdk/mux/mux.go`, replace the `Config` struct:

```go
type Config struct {
	Log          *logger.Logger
	DB           *sqlx.DB
	APIKey       string
	ClaudeCLI    *claudecli.Client
	CORSOrigins  []string
	SidecarURL   string
	OllamaURL    string
	OllamaModel  string
	OllamaEnabled bool
}
```

- [ ] **Step 2: Add Ollama conf struct to main.go**

In `api/services/planner/main.go`, inside the anonymous `cfg` struct (after the `Sidecar` block at line 108–110), add:

```go
		Ollama struct {
			URL     string
			Model   string `conf:"default:llama3"`
			Enabled bool   `conf:"default:true"`
		}
```

This produces env vars `PLANNER_OLLAMA_URL`, `PLANNER_OLLAMA_MODEL`, `PLANNER_OLLAMA_ENABLED`.

`Enabled` defaults to `true` so that setting `PLANNER_OLLAMA_URL` is sufficient to enable Ollama. Set `PLANNER_OLLAMA_ENABLED=false` to disable even when URL is set.

- [ ] **Step 3: Compute ollamaEnabled and populate muxCfg**

In `main.go`, after `conf.Parse` (around line 120) and before the `muxCfg` block (around line 180), add:

```go
	ollamaEnabled := cfg.Ollama.URL != "" && cfg.Ollama.Enabled
```

Then update the `muxCfg` literal (lines 180–187) to include Ollama fields:

```go
	muxCfg := mux.Config{
		Log:           log,
		DB:            db,
		APIKey:        cfg.Auth.APIKey,
		ClaudeCLI:     cli,
		CORSOrigins:   strings.Split(cfg.Web.CORSOrigins, ","),
		SidecarURL:    cfg.Sidecar.URL,
		OllamaURL:     cfg.Ollama.URL,
		OllamaModel:   cfg.Ollama.Model,
		OllamaEnabled: ollamaEnabled,
	}
```

- [ ] **Step 4: Update SMTP extractor instantiation in main.go**

In `main.go`, replace lines 227–228 inside `if cfg.SMTP.Enabled`:

```go
		// Before:
		ext := extractor.NewClaudeCodeExtractor(cli)

		// After:
		claudeExt := extractor.NewClaudeCodeExtractor(cli)
		var ext extractor.Extractor = claudeExt
		if ollamaEnabled {
			ollamaExt := extractor.NewOllamaExtractor(cfg.Ollama.URL, cfg.Ollama.Model)
			ext = extractor.NewFailoverExtractor(log, claudeExt, ollamaExt)
		}
```

- [ ] **Step 5: Update classifyapp/route.go**

In `app/domain/classifyapp/route.go`, replace line 33:

```go
	// Before:
	ext := extractor.NewClaudeCodeExtractor(cfg.ClaudeCLI)

	// After:
	claudeExt := extractor.NewClaudeCodeExtractor(cfg.ClaudeCLI)
	var ext extractor.Extractor = claudeExt
	if cfg.OllamaEnabled && cfg.OllamaURL != "" {
		ollamaExt := extractor.NewOllamaExtractor(cfg.OllamaURL, cfg.OllamaModel)
		ext = extractor.NewFailoverExtractor(cfg.Log, claudeExt, ollamaExt)
	}
```

Add the `extractor` import if not already present (it already is at line 12).

- [ ] **Step 6: Verify the project compiles**

```bash
go build ./...
```

Expected: no errors. If you see "OllamaEnabled undefined" on Config, check Step 1 carefully. If you see "cannot use" type errors, check that `ext` is typed as `extractor.Extractor` (interface), not as a concrete pointer.

- [ ] **Step 7: Run all tests**

```bash
go test ./... -count=1
```

Expected: all tests pass. The extractor unit tests run in-process with no external dependencies.

- [ ] **Step 8: Commit**

```bash
git add app/sdk/mux/mux.go \
        api/services/planner/main.go \
        app/domain/classifyapp/route.go
git commit -m "feat: wire Ollama fallback into classifyapp and SMTP ingestion paths"
```

---

## Manual Verification

Once deployed or running locally with Ollama:

```bash
# Start Ollama locally with a compatible model
ollama pull llama3
ollama serve

# Run the planner with Ollama enabled
PLANNER_OLLAMA_URL=http://localhost:11434 \
PLANNER_OLLAMA_MODEL=llama3 \
make dev
```

To verify fallback activates, you can temporarily return a 429 error from the Claude CLI path in a test build, or watch logs for `"claude failed, falling back to ollama"` when Claude is rate-limited in production.

## Rollout

1. Deploy without `PLANNER_OLLAMA_URL` set — Ollama is disabled, no behavior change.
2. Once locally verified, set `PLANNER_OLLAMA_URL=http://localhost:11434` in `.env`.
3. Monitor logs for `"ollama fallback"` events to track activation frequency.
