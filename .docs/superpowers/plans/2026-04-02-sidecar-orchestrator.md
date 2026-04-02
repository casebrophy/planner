# Sidecar Orchestrator Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the fire-and-forget `claude -p` inference endpoint with a persistent session orchestrator that spawns subagents, tracks token usage, and exposes observability endpoints.

**Architecture:** A `SessionManager` in the sidecar manages a single Claude session (Opus) that persists across requests via `--resume`. Each inference request is dispatched to a subagent at the requested model level. Read-only MCP access lets agents query the planner API. Observability endpoints expose token growth, duration trends, and tool call patterns.

**Tech Stack:** Go stdlib (`net/http`, `os/exec`, `sync`, `encoding/json`), Claude Code CLI (`claude -p --resume`), existing planner MCP server.

**Spec:** `.docs/superpowers/specs/2026-04-02-sidecar-orchestrator-design.md`

---

### Task 1: SessionManager Core (`zarf/sidecar/session.go`)

The foundation. All types, session lifecycle, and metrics tracking.

**Files:**
- Create: `zarf/sidecar/session.go`

- [ ] **Step 1: Create the session manager file with all types**

```go
package main

import (
	"sync"
	"time"
)

// SessionManager manages a persistent Claude orchestrator session.
// It serializes inference requests, tracks token usage, and handles
// session rotation when context is exhausted.
type SessionManager struct {
	mu           sync.Mutex
	sessionID    string
	createdAt    time.Time
	systemPrompt string
	contextMax   int
	timeout      time.Duration
	mcpURL       string
	requests     []RequestMetric
	sessions     []SessionSummary
	toolCalls    []ToolCallMetric
}

// RequestMetric records token usage and timing for a single inference request.
type RequestMetric struct {
	ID           string    `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	DurationMs   int       `json:"duration_ms"`
	AgentModel   string    `json:"agent_model"`
	PromptPrefix string    `json:"prompt_prefix"`
	Success      bool      `json:"success"`
	Error        string    `json:"error,omitempty"`
}

// SessionSummary records stats for a completed (rotated) session.
type SessionSummary struct {
	SessionID       string    `json:"session_id"`
	CreatedAt       time.Time `json:"created_at"`
	EndedAt         time.Time `json:"ended_at"`
	TotalRequests   int       `json:"total_requests"`
	EndReason       string    `json:"end_reason"`
	PeakInputTokens int      `json:"peak_input_tokens"`
}

// ToolCallMetric records a single MCP tool call made by a spawned agent.
type ToolCallMetric struct {
	Timestamp  time.Time `json:"timestamp"`
	SessionID  string    `json:"session_id"`
	RequestID  string    `json:"request_id"`
	ToolName   string    `json:"tool_name"`
	DurationMs int       `json:"duration_ms"`
}

// NewSessionManager creates a SessionManager with the given configuration.
func NewSessionManager(systemPrompt string, contextMax int, timeout time.Duration, mcpURL string) *SessionManager {
	return &SessionManager{
		systemPrompt: systemPrompt,
		contextMax:   contextMax,
		timeout:      timeout,
		mcpURL:       mcpURL,
	}
}
```

- [ ] **Step 2: Add the rotate method**

Append to `zarf/sidecar/session.go`:

```go
// rotate archives the current session and clears state for a new one.
// Must be called with mu held.
func (sm *SessionManager) rotate(reason string) SessionSummary {
	var summary SessionSummary
	if sm.sessionID != "" {
		peak := 0
		for _, r := range sm.requests {
			if r.InputTokens > peak {
				peak = r.InputTokens
			}
		}
		summary = SessionSummary{
			SessionID:       sm.sessionID,
			CreatedAt:       sm.createdAt,
			EndedAt:         time.Now(),
			TotalRequests:   len(sm.requests),
			EndReason:       reason,
			PeakInputTokens: peak,
		}
		sm.sessions = append(sm.sessions, summary)
	}

	sm.sessionID = ""
	sm.createdAt = time.Time{}
	sm.requests = nil
	sm.toolCalls = nil

	return summary
}
```

- [ ] **Step 3: Add the status and history methods**

Append to `zarf/sidecar/session.go`:

```go
// StatusResponse is the JSON body for GET /inference/status.
type StatusResponse struct {
	SessionID          string    `json:"session_id"`
	CreatedAt          time.Time `json:"created_at,omitempty"`
	AgeSeconds         float64   `json:"age_seconds"`
	TotalRequests      int       `json:"total_requests"`
	LatestInputTokens  int       `json:"latest_input_tokens"`
	ContextMax         int       `json:"context_max"`
	ContextUsagePct    float64   `json:"context_usage_pct"`
	TokenGrowth        []int     `json:"token_growth"`
	AvgDurationMs      int       `json:"avg_duration_ms"`
	DurationTrend      []int     `json:"duration_trend"`
	RequestsSinceRotation int   `json:"requests_since_rotation"`
}

// Status returns the current session health snapshot.
// Must be called with mu held.
func (sm *SessionManager) Status() StatusResponse {
	resp := StatusResponse{
		SessionID: sm.sessionID,
		ContextMax: sm.contextMax,
	}

	if sm.sessionID == "" {
		return resp
	}

	resp.CreatedAt = sm.createdAt
	resp.AgeSeconds = time.Since(sm.createdAt).Seconds()
	resp.TotalRequests = len(sm.requests)
	resp.RequestsSinceRotation = len(sm.requests)

	var totalDuration int
	for _, r := range sm.requests {
		resp.TokenGrowth = append(resp.TokenGrowth, r.InputTokens)
		resp.DurationTrend = append(resp.DurationTrend, r.DurationMs)
		totalDuration += r.DurationMs
	}

	if len(sm.requests) > 0 {
		last := sm.requests[len(sm.requests)-1]
		resp.LatestInputTokens = last.InputTokens
		resp.ContextUsagePct = float64(last.InputTokens) / float64(sm.contextMax) * 100
		resp.AvgDurationMs = totalDuration / len(sm.requests)
	}

	return resp
}

// HistoryResponse is the JSON body for GET /inference/history.
type HistoryResponse struct {
	Sessions []SessionSummary `json:"sessions"`
}

// History returns all past session summaries.
// Must be called with mu held.
func (sm *SessionManager) History() HistoryResponse {
	sessions := sm.sessions
	if sessions == nil {
		sessions = []SessionSummary{}
	}
	return HistoryResponse{Sessions: sessions}
}
```

- [ ] **Step 4: Add the tools analytics method**

Append to `zarf/sidecar/session.go`:

```go
// ToolsResponse is the JSON body for GET /inference/tools.
type ToolsResponse struct {
	ToolFrequency    map[string]int `json:"tool_frequency"`
	AvgCallsPerReq   float64       `json:"avg_calls_per_request"`
}

// Tools returns tool call frequency data for the current session.
// Must be called with mu held.
func (sm *SessionManager) Tools() ToolsResponse {
	freq := make(map[string]int)
	for _, tc := range sm.toolCalls {
		freq[tc.ToolName]++
	}

	var avg float64
	if len(sm.requests) > 0 {
		avg = float64(len(sm.toolCalls)) / float64(len(sm.requests))
	}

	return ToolsResponse{
		ToolFrequency:  freq,
		AvgCallsPerReq: avg,
	}
}
```

- [ ] **Step 5: Verify it compiles**

Run: `cd /Users/casebrophy/personal/planner/zarf/sidecar && go build ./...`
Expected: No errors.

- [ ] **Step 6: Commit**

```bash
cd /Users/casebrophy/personal/planner
git add zarf/sidecar/session.go
git commit -m "feat(sidecar): add SessionManager types and lifecycle methods"
```

---

### Task 2: Session-Managed Inference Handler (`zarf/sidecar/handlers.go`)

Replace the fire-and-forget `inference()` handler with one that uses `--resume` and the orchestrator system prompt. Add observability handlers.

**Files:**
- Modify: `zarf/sidecar/handlers.go` (replace `inference` method, add new handlers)
- Modify: `zarf/sidecar/main.go` (add routes, initialize SessionManager)

- [ ] **Step 1: Add the system prompt constant to handlers.go**

Add after the `handlers` struct definition in `zarf/sidecar/handlers.go`:

```go
const orchestratorSystemPrompt = `You are the planner system's inference orchestrator. You run persistently on the server and handle automated inference requests from the planner's backend pipelines.

Your ONLY job is to dispatch work to subagents and return their raw results.

When you receive a request:
1. Read the "model" field to determine which model the subagent should use
2. Read the "prompt" field for the task to dispatch
3. If a "schema" field is present, include it in the agent's instructions as the required output format
4. Spawn a subagent using the Agent tool at the specified model
5. The subagent has read-only access to the planner's MCP tools (list_tasks, list_events, get_context, etc.) and a specialized get_inference_context tool for pre-assembled context bundles
6. Return ONLY the subagent's output text. No commentary, no wrapping, no additional formatting.

The requests come from automated pipelines:
- Daily plan generation: groups and prioritizes tasks for the day
- Email extraction: extracts action items, deadlines, and context from emails
- Text/voice extraction: extracts tasks and events from free-form input
- Thread classification: classifies thread entries by kind, sentiment, and urgency

Each request includes its own detailed prompt with classification rules and expected output schema. Trust the prompt. Do not second-guess the pipeline's instructions.

Never reason about the task yourself. Never add your own analysis. You are a dispatcher.`
```

- [ ] **Step 2: Update the handlers struct to include SessionManager**

Replace the `handlers` struct in `zarf/sidecar/handlers.go`:

```go
type handlers struct {
	composeFile string
	session     *SessionManager
}
```

- [ ] **Step 3: Rewrite the inference handler**

Replace the entire `inference` method (and its request/response types) in `zarf/sidecar/handlers.go` with:

```go
// =========================================================================
// POST /inference

type InferenceRequest struct {
	Prompt string `json:"prompt"`
	Schema string `json:"schema,omitempty"`
	Model  string `json:"model"`
}

type InferenceResponse struct {
	Result string `json:"result"`
	Model  string `json:"model"`
}

func (h *handlers) inference(w http.ResponseWriter, r *http.Request) {
	var req InferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid json: "+err.Error())
		return
	}
	if req.Prompt == "" {
		writeError(w, 400, "prompt is required")
		return
	}
	if req.Model == "" {
		req.Model = "haiku"
	}

	h.session.mu.Lock()
	defer h.session.mu.Unlock()

	// Build the dispatch message for the orchestrator.
	dispatchMsg := buildDispatchMessage(req)

	// Build CLI args.
	args := []string{
		"-p", dispatchMsg,
		"--output-format", "json",
		"--model", "opus",
	}

	if h.session.sessionID == "" {
		// First request: include system prompt.
		args = append(args, "--system-prompt", h.session.systemPrompt)
	} else {
		// Resume existing session.
		args = append(args, "--resume", h.session.sessionID)
	}

	// Execute with timeout.
	ctx, cancel := context.WithTimeout(r.Context(), h.session.timeout)
	defer cancel()

	start := time.Now()
	cmd := exec.CommandContext(ctx, "claude", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	durationMs := int(time.Since(start).Milliseconds())

	// Truncate prompt for metrics.
	prefix := req.Prompt
	if len(prefix) > 100 {
		prefix = prefix[:100]
	}

	metric := RequestMetric{
		ID:           fmt.Sprintf("req-%d", time.Now().UnixNano()),
		Timestamp:    start,
		DurationMs:   durationMs,
		AgentModel:   req.Model,
		PromptPrefix: prefix,
	}

	if err != nil {
		metric.Success = false
		exitErr, ok := err.(*exec.ExitError)
		errMsg := err.Error()
		if ok {
			errMsg += "; stderr: " + string(exitErr.Stderr)
		}
		if len(stderr.Bytes()) > 0 {
			errMsg += "; stderr: " + stderr.String()
		}
		metric.Error = errMsg
		h.session.requests = append(h.session.requests, metric)

		// Rotate on error.
		h.session.rotate("error")
		writeError(w, 502, "claude cli failed: "+errMsg)
		return
	}

	// Parse the JSON output to extract session ID, result, and token counts.
	sessionID, result, inputTokens, outputTokens := parseClaudeOutput(stdout.Bytes())

	if h.session.sessionID == "" && sessionID != "" {
		h.session.sessionID = sessionID
		h.session.createdAt = start
	}

	metric.Success = true
	metric.InputTokens = inputTokens
	metric.OutputTokens = outputTokens
	h.session.requests = append(h.session.requests, metric)

	// Check for context threshold rotation.
	if inputTokens >= h.session.contextMax {
		h.session.rotate("context_full")
	}

	writeJSON(w, InferenceResponse{Result: result, Model: req.Model})
}

// buildDispatchMessage wraps the inference request into a structured message
// for the orchestrator to parse and dispatch.
func buildDispatchMessage(req InferenceRequest) string {
	msg := fmt.Sprintf(`{"model":"%s","prompt":%s`, req.Model, mustMarshalString(req.Prompt))
	if req.Schema != "" {
		msg += fmt.Sprintf(`,"schema":%s`, mustMarshalString(req.Schema))
	}
	msg += "}"
	return msg
}

func mustMarshalString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// parseClaudeOutput extracts session_id, result text, and token counts from
// the claude --output-format json output.
func parseClaudeOutput(data []byte) (sessionID, result string, inputTokens, outputTokens int) {
	// The output is a JSON array of event objects.
	var events []map[string]any
	if err := json.Unmarshal(data, &events); err != nil {
		// Fallback: return raw output as result.
		return "", string(data), 0, 0
	}

	for _, evt := range events {
		typ, _ := evt["type"].(string)

		// Capture session_id from the init event.
		if typ == "system" {
			if sid, ok := evt["session_id"].(string); ok && sid != "" {
				sessionID = sid
			}
		}

		// Capture result from the result event.
		if typ == "result" {
			if r, ok := evt["result"].(string); ok {
				result = r
			}
			if v, ok := evt["input_tokens"].(float64); ok {
				inputTokens = int(v)
			}
			if v, ok := evt["output_tokens"].(float64); ok {
				outputTokens = int(v)
			}
		}
	}

	// If no result found, use raw output.
	if result == "" {
		result = string(data)
	}

	return sessionID, result, inputTokens, outputTokens
}
```

- [ ] **Step 4: Add the observability handlers**

Append to `zarf/sidecar/handlers.go`:

```go
// =========================================================================
// GET /inference/status

func (h *handlers) inferenceStatus(w http.ResponseWriter, r *http.Request) {
	h.session.mu.Lock()
	status := h.session.Status()
	h.session.mu.Unlock()

	writeJSON(w, status)
}

// =========================================================================
// GET /inference/history

func (h *handlers) inferenceHistory(w http.ResponseWriter, r *http.Request) {
	h.session.mu.Lock()
	history := h.session.History()
	h.session.mu.Unlock()

	writeJSON(w, history)
}

// =========================================================================
// GET /inference/tools

func (h *handlers) inferenceTools(w http.ResponseWriter, r *http.Request) {
	h.session.mu.Lock()
	tools := h.session.Tools()
	h.session.mu.Unlock()

	writeJSON(w, tools)
}

// =========================================================================
// POST /inference/rotate

func (h *handlers) inferenceRotate(w http.ResponseWriter, r *http.Request) {
	h.session.mu.Lock()
	summary := h.session.rotate("manual")
	h.session.mu.Unlock()

	writeJSON(w, map[string]any{
		"old_session_id":    summary.SessionID,
		"requests_served":   summary.TotalRequests,
		"peak_input_tokens": summary.PeakInputTokens,
		"rotated":           true,
	})
}
```

- [ ] **Step 5: Update main.go — add imports, SessionManager init, and new routes**

Replace the contents of `zarf/sidecar/main.go` with:

```go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9090", "listen address")
	apiKey := flag.String("api-key", "", "required API key (or PLANNER_AUTH_API_KEY env)")
	composeFile := flag.String("compose-file", "/opt/planner/zarf/compose/docker-compose.yml", "docker-compose.yml path")
	flag.Parse()

	if *apiKey == "" {
		*apiKey = os.Getenv("PLANNER_AUTH_API_KEY")
	}
	if *apiKey == "" {
		log.Fatal("--api-key or PLANNER_AUTH_API_KEY required")
	}

	// Session manager configuration from env.
	contextMax := 150000
	if v := os.Getenv("SIDECAR_CONTEXT_MAX"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			contextMax = n
		}
	}

	requestTimeout := 180 * time.Second
	if v := os.Getenv("SIDECAR_REQUEST_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			requestTimeout = d
		}
	}

	mcpURL := os.Getenv("PLANNER_MCP_URL")
	if mcpURL == "" {
		mcpURL = "http://localhost:8080/mcp"
	}

	session := NewSessionManager(orchestratorSystemPrompt, contextMax, requestTimeout, mcpURL)

	h := &handlers{composeFile: *composeFile, session: session}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /containers", h.containers)
	mux.HandleFunc("GET /logs/{service}", h.logs)
	mux.HandleFunc("GET /claude", h.claude)
	mux.HandleFunc("GET /timers", h.timers)
	mux.HandleFunc("POST /inference", h.inference)
	mux.HandleFunc("GET /inference/status", h.inferenceStatus)
	mux.HandleFunc("GET /inference/history", h.inferenceHistory)
	mux.HandleFunc("GET /inference/tools", h.inferenceTools)
	mux.HandleFunc("POST /inference/rotate", h.inferenceRotate)

	handler := authMiddleware(*apiKey, mux)

	fmt.Printf("sidecar listening on %s (context_max=%d, timeout=%s)\n", *addr, contextMax, requestTimeout)
	log.Fatal(http.ListenAndServe(*addr, handler))
}

func authMiddleware(apiKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != apiKey {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
```

- [ ] **Step 6: Add missing imports to handlers.go**

Ensure the import block at the top of `zarf/sidecar/handlers.go` includes:

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)
```

- [ ] **Step 7: Verify it compiles**

Run: `cd /Users/casebrophy/personal/planner/zarf/sidecar && go build ./...`
Expected: No errors.

- [ ] **Step 8: Commit**

```bash
cd /Users/casebrophy/personal/planner
git add zarf/sidecar/handlers.go zarf/sidecar/main.go
git commit -m "feat(sidecar): session-managed inference with observability endpoints"
```

---

### Task 3: `get_inference_context` MCP Tool (`app/domain/mcpapp/`)

Add the composite MCP tool that returns pre-assembled context bundles for each pipeline use case.

**Files:**
- Modify: `app/domain/mcpapp/tools.go` (add tool definition)
- Modify: `app/domain/mcpapp/mcpapp.go` (add case in `callTool` switch, implement handler)

- [ ] **Step 1: Add tool definition to tools.go**

Add to the `tools` slice in `app/domain/mcpapp/tools.go`, after the last entry (`confirm_time_block`):

```go
	{
		Name:        "get_inference_context",
		Description: "Get pre-assembled context for a specific inference use case. Returns all relevant data in a single call instead of requiring multiple tool calls.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"use_case": map[string]any{
					"type":        "string",
					"enum":        []string{"daily_plan", "email_extraction", "text_extraction", "thread_classification"},
					"description": "The inference pipeline requesting context",
				},
				"date": map[string]any{
					"type":        "string",
					"description": "ISO 8601 date for date-scoped use cases (daily_plan, text_extraction)",
				},
				"subject_id": map[string]any{
					"type":        "string",
					"description": "UUID of the subject (for thread_classification)",
				},
				"subject_type": map[string]any{
					"type":        "string",
					"enum":        []string{"task", "context"},
					"description": "Type of subject (for thread_classification)",
				},
			},
			"required": []string{"use_case"},
		},
	},
```

- [ ] **Step 2: Add case to callTool switch in mcpapp.go**

In `app/domain/mcpapp/mcpapp.go`, add before the `default:` case in the `callTool` method:

```go
	case "get_inference_context":
		return a.toolGetInferenceContext(ctx, params.Arguments)
```

- [ ] **Step 3: Implement the handler in mcpapp.go**

Add the following method to `app/domain/mcpapp/mcpapp.go`:

```go
func (a *app) toolGetInferenceContext(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		UseCase     string `json:"use_case"`
		Date        string `json:"date"`
		SubjectID   string `json:"subject_id"`
		SubjectType string `json:"subject_type"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, fmt.Errorf("invalid arguments: %w", err)
	}

	switch input.UseCase {
	case "daily_plan":
		return a.inferenceContextDailyPlan(ctx, input.Date)
	case "email_extraction":
		return a.inferenceContextEmailExtraction(ctx)
	case "text_extraction":
		return a.inferenceContextTextExtraction(ctx, input.Date)
	case "thread_classification":
		return a.inferenceContextThreadClassification(ctx, input.SubjectID, input.SubjectType)
	default:
		return toolResult{}, fmt.Errorf("unknown use_case: %s", input.UseCase)
	}
}

func (a *app) inferenceContextDailyPlan(ctx context.Context, dateStr string) (toolResult, error) {
	// Open tasks.
	taskFilter := taskbus.QueryFilter{}
	openStatus := taskstatus.MustParse("todo")
	taskFilter.Status = &openStatus
	tasks, err := a.taskBus.Query(ctx, taskFilter, taskbus.DefaultOrderBy, page.MustParse("1", "100"))
	if err != nil {
		return toolResult{}, fmt.Errorf("query tasks: %w", err)
	}

	// Also get in_progress tasks.
	ipStatus := taskstatus.MustParse("in_progress")
	taskFilter.Status = &ipStatus
	ipTasks, err := a.taskBus.Query(ctx, taskFilter, taskbus.DefaultOrderBy, page.MustParse("1", "100"))
	if err != nil {
		return toolResult{}, fmt.Errorf("query in_progress tasks: %w", err)
	}
	tasks = append(tasks, ipTasks...)

	// Today's events.
	now := time.Now()
	if dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			now = t
		}
	}
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayEnd := dayStart.Add(24 * time.Hour)
	eventFilter := eventbus.QueryFilter{
		DateFrom: &dayStart,
		DateTo:   &dayEnd,
	}
	events, err := a.eventBus.Query(ctx, eventFilter, eventbus.DefaultOrderBy, page.MustParse("1", "50"))
	if err != nil {
		return toolResult{}, fmt.Errorf("query events: %w", err)
	}

	// Active contexts for enrichment.
	ctxFilter := contextbus.QueryFilter{}
	contexts, err := a.contextBus.Query(ctx, ctxFilter, contextbus.DefaultOrderBy, page.MustParse("1", "50"))
	if err != nil {
		return toolResult{}, fmt.Errorf("query contexts: %w", err)
	}

	result := map[string]any{
		"tasks":    tasks,
		"events":   events,
		"contexts": contexts,
	}

	return textResult(result)
}

func (a *app) inferenceContextEmailExtraction(ctx context.Context) (toolResult, error) {
	ctxFilter := contextbus.QueryFilter{}
	contexts, err := a.contextBus.Query(ctx, ctxFilter, contextbus.DefaultOrderBy, page.MustParse("1", "50"))
	if err != nil {
		return toolResult{}, fmt.Errorf("query contexts: %w", err)
	}

	result := map[string]any{
		"active_contexts": contexts,
	}

	return textResult(result)
}

func (a *app) inferenceContextTextExtraction(ctx context.Context, dateStr string) (toolResult, error) {
	// Active contexts.
	ctxFilter := contextbus.QueryFilter{}
	contexts, err := a.contextBus.Query(ctx, ctxFilter, contextbus.DefaultOrderBy, page.MustParse("1", "50"))
	if err != nil {
		return toolResult{}, fmt.Errorf("query contexts: %w", err)
	}

	// Today's events for temporal grounding.
	now := time.Now()
	if dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			now = t
		}
	}
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayEnd := dayStart.Add(24 * time.Hour)
	eventFilter := eventbus.QueryFilter{
		DateFrom: &dayStart,
		DateTo:   &dayEnd,
	}
	events, err := a.eventBus.Query(ctx, eventFilter, eventbus.DefaultOrderBy, page.MustParse("1", "50"))
	if err != nil {
		return toolResult{}, fmt.Errorf("query events: %w", err)
	}

	result := map[string]any{
		"active_contexts": contexts,
		"todays_events":   events,
	}

	return textResult(result)
}

func (a *app) inferenceContextThreadClassification(ctx context.Context, subjectID, subjectType string) (toolResult, error) {
	if subjectID == "" || subjectType == "" {
		return toolResult{}, fmt.Errorf("subject_id and subject_type are required for thread_classification")
	}

	sid, err := uuid.Parse(subjectID)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid subject_id: %w", err)
	}

	// Thread history (last 10 entries).
	entries, err := a.threadBus.QueryBySubject(ctx, subjectType, sid, page.MustParse("1", "10"))
	if err != nil {
		return toolResult{}, fmt.Errorf("query thread: %w", err)
	}

	// Subject details.
	var subject any
	switch subjectType {
	case "task":
		task, err := a.taskBus.QueryByID(ctx, sid)
		if err != nil {
			return toolResult{}, fmt.Errorf("query task: %w", err)
		}
		subject = task
	case "context":
		ctxObj, err := a.contextBus.QueryByID(ctx, sid)
		if err != nil {
			return toolResult{}, fmt.Errorf("query context: %w", err)
		}
		subject = ctxObj
	default:
		return toolResult{}, fmt.Errorf("invalid subject_type: %s", subjectType)
	}

	result := map[string]any{
		"thread_entries": entries,
		"subject":        subject,
	}

	return textResult(result)
}
```

- [ ] **Step 4: Verify it compiles**

Run: `cd /Users/casebrophy/personal/planner && make lint`
Expected: No errors (go vet passes).

- [ ] **Step 5: Commit**

```bash
cd /Users/casebrophy/personal/planner
git add app/domain/mcpapp/tools.go app/domain/mcpapp/mcpapp.go
git commit -m "feat(mcp): add get_inference_context composite tool"
```

---

### Task 4: Update AI Model Layer Documentation

Update the planning doc to reflect the new architecture.

**Files:**
- Modify: `.docs/08-ai-model-layer.md`

- [ ] **Step 1: Update the doc**

Replace the contents of `.docs/08-ai-model-layer.md` with updated documentation that includes the orchestrator architecture. The key changes:

1. Add a new "Orchestrator Architecture" section after "Claude CLI client" explaining the sidecar session manager, `--resume` flow, and rotation strategy.
2. Add a "Sidecar Observability" section listing the new endpoints (`/inference/status`, `/inference/history`, `/inference/tools`, `/inference/rotate`).
3. Add documentation for the `get_inference_context` MCP tool under the Extractors section.
4. Update the Configuration reference to include `SIDECAR_CONTEXT_MAX`, `SIDECAR_REQUEST_TIMEOUT`, `PLANNER_MCP_URL`.

Keep all existing content about extractors, embedder, RAG, and vector storage unchanged.

- [ ] **Step 2: Commit**

```bash
cd /Users/casebrophy/personal/planner
git add .docs/08-ai-model-layer.md
git commit -m "docs: update AI model layer with orchestrator architecture"
```

---

### Task 5: Build and Smoke Test

Verify everything compiles and the sidecar binary works.

- [ ] **Step 1: Build the sidecar**

Run: `cd /Users/casebrophy/personal/planner/zarf/sidecar && go build -o sidecar .`
Expected: Binary `sidecar` produced, no errors.

- [ ] **Step 2: Build the main planner**

Run: `cd /Users/casebrophy/personal/planner && make lint`
Expected: `go vet` passes.

- [ ] **Step 3: Run planner tests**

Run: `cd /Users/casebrophy/personal/planner && make test`
Expected: All tests pass.

- [ ] **Step 4: Verify sidecar starts**

Run: `cd /Users/casebrophy/personal/planner/zarf/sidecar && PLANNER_AUTH_API_KEY=testkey ./sidecar --addr=127.0.0.1:0 &`
Expected: Prints `sidecar listening on 127.0.0.1:0 (context_max=150000, timeout=3m0s)` and exits cleanly on SIGINT.

- [ ] **Step 5: Test observability endpoints (no session yet)**

```bash
# Status should return empty session.
curl -s -H "X-API-Key: testkey" http://127.0.0.1:<port>/inference/status | jq .

# History should return empty array.
curl -s -H "X-API-Key: testkey" http://127.0.0.1:<port>/inference/history | jq .

# Tools should return empty frequency map.
curl -s -H "X-API-Key: testkey" http://127.0.0.1:<port>/inference/tools | jq .
```

Expected: All return valid JSON with empty/zero values.

- [ ] **Step 6: Kill the test sidecar and commit any fixes**

```bash
kill %1
```

If any fixes were needed, commit them:

```bash
git add -A && git commit -m "fix(sidecar): address build/smoke-test issues"
```
