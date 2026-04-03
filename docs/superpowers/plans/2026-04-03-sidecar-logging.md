# Sidecar Logging System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add persistent request logging and structured operational logging to the sidecar, with HTTP query endpoints for frontend consumption.

**Architecture:** JSON-lines file for persistent request logs (append-only, pruned on startup). Structured JSON logger to stderr for operational events (captured by journalctl). Two new HTTP endpoints to query logs and aggregated stats.

**Tech Stack:** Go stdlib only (no external deps). JSON-lines file format. systemd/journalctl for operational log capture.

---

## File Map

- **Create:** `zarf/sidecar/logstore.go` — append-only JSON-lines writer, reader with filtering, startup pruning
- **Create:** `zarf/sidecar/logger.go` — structured JSON stderr logger replacing fmt/log calls
- **Modify:** `zarf/sidecar/main.go` — wire LogStore + Logger, parse new env vars/flags
- **Modify:** `zarf/sidecar/handlers.go` — log events, write to LogStore after inference, add query endpoints
- **Modify:** `zarf/sidecar/session.go` — accept logger for rotation events

---

### Task 1: Structured Logger (`logger.go`)

**Files:**
- Create: `zarf/sidecar/logger.go`

- [ ] **Step 1: Create the structured logger**

Write `zarf/sidecar/logger.go`:

```go
package main

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// Logger writes structured JSON log lines to stderr.
type Logger struct {
	mu  sync.Mutex
	enc *json.Encoder
}

// NewLogger creates a Logger that writes to stderr.
func NewLogger() *Logger {
	return &Logger{enc: json.NewEncoder(os.Stderr)}
}

// LogEntry is a single structured log line.
type LogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Msg     string `json:"msg"`
	Fields  map[string]any `json:"fields,omitempty"`
}

// Info logs an informational message with optional key-value fields.
func (l *Logger) Info(msg string, fields map[string]any) {
	l.log("info", msg, fields)
}

// Error logs an error message with optional key-value fields.
func (l *Logger) Error(msg string, fields map[string]any) {
	l.log("error", msg, fields)
}

// Warn logs a warning message with optional key-value fields.
func (l *Logger) Warn(msg string, fields map[string]any) {
	l.log("warn", msg, fields)
}

func (l *Logger) log(level, msg string, fields map[string]any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.enc.Encode(LogEntry{
		Time:   time.Now().UTC().Format(time.RFC3339),
		Level:  level,
		Msg:    msg,
		Fields: fields,
	})
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd zarf/sidecar && go build ./...`
Expected: clean build

- [ ] **Step 3: Commit**

```bash
git add zarf/sidecar/logger.go
git commit -m "feat(sidecar): add structured JSON logger"
```

---

### Task 2: Log Store (`logstore.go`)

**Files:**
- Create: `zarf/sidecar/logstore.go`

- [ ] **Step 1: Create the LogStore with write, read, prune, and stats**

Write `zarf/sidecar/logstore.go`:

```go
package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RequestLog is a single persisted inference request record.
type RequestLog struct {
	ID           string    `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	DurationMs   int       `json:"duration_ms"`
	AgentModel   string    `json:"agent_model"`
	PromptPrefix string    `json:"prompt_prefix"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	SessionID    string    `json:"session_id"`
	Success      bool      `json:"success"`
	Error        string    `json:"error,omitempty"`
	Prompt       string    `json:"prompt,omitempty"`
	Result       string    `json:"result,omitempty"`
}

// LogStore manages an append-only JSON-lines file of request logs.
type LogStore struct {
	mu            sync.Mutex
	path          string
	retentionDays int
	verbose       bool
	logger        *Logger
}

// NewLogStore creates a LogStore, ensuring the parent directory exists,
// and prunes entries older than retentionDays on startup.
func NewLogStore(path string, retentionDays int, verbose bool, logger *Logger) (*LogStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	ls := &LogStore{
		path:          path,
		retentionDays: retentionDays,
		verbose:       verbose,
		logger:        logger,
	}

	pruned, err := ls.prune()
	if err != nil {
		logger.Warn("log prune failed", map[string]any{"error": err.Error()})
	} else if pruned > 0 {
		logger.Info("pruned old log entries", map[string]any{"pruned": pruned, "retention_days": retentionDays})
	}

	return ls, nil
}

// Append writes a RequestLog entry to the file.
func (ls *LogStore) Append(entry RequestLog) error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	// Strip verbose fields if not enabled.
	if !ls.verbose {
		entry.Prompt = ""
		entry.Result = ""
	}

	f, err := os.OpenFile(ls.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(entry)
}

// LogFilter defines query parameters for reading logs.
type LogFilter struct {
	Since   time.Time
	Until   time.Time
	Limit   int
	Success *bool
}

// Query reads log entries matching the filter, returning newest first.
func (ls *LogStore) Query(filter LogFilter) ([]RequestLog, error) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	entries, err := ls.readAll()
	if err != nil {
		return nil, err
	}

	var filtered []RequestLog
	for _, e := range entries {
		if !filter.Since.IsZero() && e.Timestamp.Before(filter.Since) {
			continue
		}
		if !filter.Until.IsZero() && e.Timestamp.After(filter.Until) {
			continue
		}
		if filter.Success != nil && e.Success != *filter.Success {
			continue
		}
		filtered = append(filtered, e)
	}

	// Reverse for newest-first.
	for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
		filtered[i], filtered[j] = filtered[j], filtered[i]
	}

	if filter.Limit > 0 && len(filtered) > filter.Limit {
		filtered = filtered[:filter.Limit]
	}

	if filtered == nil {
		filtered = []RequestLog{}
	}
	return filtered, nil
}

// LogStats holds aggregated statistics over a time window.
type LogStats struct {
	TotalRequests int            `json:"total_requests"`
	Successes     int            `json:"successes"`
	Failures      int            `json:"failures"`
	SuccessRate   float64        `json:"success_rate"`
	AvgDurationMs int            `json:"avg_duration_ms"`
	TotalInputTok int            `json:"total_input_tokens"`
	TotalOutputTok int           `json:"total_output_tokens"`
	ByModel       map[string]ModelStats `json:"by_model"`
}

// ModelStats holds per-model aggregated statistics.
type ModelStats struct {
	Requests     int     `json:"requests"`
	AvgDurationMs int    `json:"avg_duration_ms"`
	SuccessRate  float64 `json:"success_rate"`
}

// Stats returns aggregated statistics for entries after since.
func (ls *LogStore) Stats(since time.Time) (LogStats, error) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	entries, err := ls.readAll()
	if err != nil {
		return LogStats{}, err
	}

	stats := LogStats{ByModel: make(map[string]ModelStats)}
	type modelAcc struct {
		requests  int
		duration  int
		successes int
	}
	models := make(map[string]*modelAcc)

	var totalDuration int
	for _, e := range entries {
		if !since.IsZero() && e.Timestamp.Before(since) {
			continue
		}
		stats.TotalRequests++
		totalDuration += e.DurationMs
		stats.TotalInputTok += e.InputTokens
		stats.TotalOutputTok += e.OutputTokens

		if e.Success {
			stats.Successes++
		} else {
			stats.Failures++
		}

		acc, ok := models[e.AgentModel]
		if !ok {
			acc = &modelAcc{}
			models[e.AgentModel] = acc
		}
		acc.requests++
		acc.duration += e.DurationMs
		if e.Success {
			acc.successes++
		}
	}

	if stats.TotalRequests > 0 {
		stats.SuccessRate = float64(stats.Successes) / float64(stats.TotalRequests)
		stats.AvgDurationMs = totalDuration / stats.TotalRequests
	}

	for model, acc := range models {
		ms := ModelStats{Requests: acc.requests}
		if acc.requests > 0 {
			ms.AvgDurationMs = acc.duration / acc.requests
			ms.SuccessRate = float64(acc.successes) / float64(acc.requests)
		}
		stats.ByModel[model] = ms
	}

	return stats, nil
}

// prune removes entries older than retentionDays by rewriting the file.
// Returns the number of pruned entries.
func (ls *LogStore) prune() (int, error) {
	entries, err := ls.readAll()
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	cutoff := time.Now().AddDate(0, 0, -ls.retentionDays)
	var kept []RequestLog
	pruned := 0
	for _, e := range entries {
		if e.Timestamp.Before(cutoff) {
			pruned++
			continue
		}
		kept = append(kept, e)
	}

	if pruned == 0 {
		return 0, nil
	}

	f, err := os.Create(ls.path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, e := range kept {
		if err := enc.Encode(e); err != nil {
			return 0, err
		}
	}

	return pruned, nil
}

// readAll reads all entries from the log file.
func (ls *LogStore) readAll() ([]RequestLog, error) {
	f, err := os.Open(ls.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []RequestLog
	scanner := bufio.NewScanner(f)
	// Allow up to 1MB per line for verbose entries.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry RequestLog
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue // skip malformed lines
		}
		entries = append(entries, entry)
	}
	return entries, scanner.Err()
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd zarf/sidecar && go build ./...`
Expected: clean build

- [ ] **Step 3: Commit**

```bash
git add zarf/sidecar/logstore.go
git commit -m "feat(sidecar): add persistent JSON-lines log store"
```

---

### Task 3: Wire Logger + LogStore into main.go

**Files:**
- Modify: `zarf/sidecar/main.go`

- [ ] **Step 1: Add env var parsing and initialization**

In `main.go`, replace the imports and the `main` function body. The new `main()`:

```go
package main

import (
	"encoding/json"
	"flag"
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

	logger := NewLogger()

	if *apiKey == "" {
		logger.Error("--api-key or PLANNER_AUTH_API_KEY required", nil)
		os.Exit(1)
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

	// Log store configuration.
	logPath := os.Getenv("SIDECAR_LOG_PATH")
	if logPath == "" {
		logPath = "/var/log/planner/sidecar-requests.jsonl"
	}

	retentionDays := 30
	if v := os.Getenv("SIDECAR_LOG_RETENTION_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			retentionDays = n
		}
	}

	verbose := false
	if v := os.Getenv("SIDECAR_LOG_VERBOSE"); v == "true" || v == "1" {
		verbose = true
	}

	logStore, err := NewLogStore(logPath, retentionDays, verbose, logger)
	if err != nil {
		logger.Error("failed to initialize log store", map[string]any{"error": err.Error()})
		os.Exit(1)
	}

	session := NewSessionManager(orchestratorSystemPrompt, contextMax, requestTimeout, mcpURL, logger)

	h := &handlers{composeFile: *composeFile, session: session, apiKey: *apiKey, logStore: logStore, logger: logger}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /containers", h.containers)
	mux.HandleFunc("GET /logs/{service}", h.logs)
	mux.HandleFunc("GET /logs/sidecar", h.sidecarLogs)
	mux.HandleFunc("GET /logs/sidecar/stats", h.sidecarLogStats)
	mux.HandleFunc("GET /claude", h.claude)
	mux.HandleFunc("GET /timers", h.timers)
	mux.HandleFunc("POST /inference", h.inference)
	mux.HandleFunc("GET /inference/status", h.inferenceStatus)
	mux.HandleFunc("GET /inference/history", h.inferenceHistory)
	mux.HandleFunc("GET /inference/tools", h.inferenceTools)
	mux.HandleFunc("POST /inference/rotate", h.inferenceRotate)

	handler := authMiddleware(*apiKey, mux)

	logger.Info("sidecar starting", map[string]any{
		"addr":           *addr,
		"context_max":    contextMax,
		"timeout":        requestTimeout.String(),
		"log_path":       logPath,
		"retention_days": retentionDays,
		"verbose":        verbose,
	})
	if err := http.ListenAndServe(*addr, handler); err != nil {
		logger.Error("server failed", map[string]any{"error": err.Error()})
		os.Exit(1)
	}
}
```

Keep `authMiddleware`, `writeJSON`, and `writeError` unchanged.

- [ ] **Step 2: Verify it compiles (will fail — handlers/session not updated yet)**

Run: `cd zarf/sidecar && go build ./... 2>&1 | head -20`
Expected: compile errors for missing fields on `handlers` and `NewSessionManager` — that's correct, we fix those in the next tasks.

- [ ] **Step 3: Commit**

```bash
git add zarf/sidecar/main.go
git commit -m "feat(sidecar): wire logger and logstore into main"
```

---

### Task 4: Update `session.go` to accept Logger

**Files:**
- Modify: `zarf/sidecar/session.go`

- [ ] **Step 1: Add logger field and update constructor and rotate**

Update `SessionManager` struct to include `logger *Logger`. Update `NewSessionManager` to accept it. Update `rotate` to log the event:

In `session.go`, change `SessionManager` struct:

```go
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
	logger       *Logger
}
```

Change `NewSessionManager`:

```go
func NewSessionManager(systemPrompt string, contextMax int, timeout time.Duration, mcpURL string, logger *Logger) *SessionManager {
	return &SessionManager{
		systemPrompt: systemPrompt,
		contextMax:   contextMax,
		timeout:      timeout,
		mcpURL:       mcpURL,
		logger:       logger,
	}
}
```

Add a log call at the end of `rotate`, just before `return summary`:

```go
	sm.logger.Info("session rotated", map[string]any{
		"session_id":     summary.SessionID,
		"reason":         reason,
		"total_requests": summary.TotalRequests,
		"peak_tokens":    summary.PeakInputTokens,
	})
```

- [ ] **Step 2: Verify it compiles**

Run: `cd zarf/sidecar && go build ./...`
Expected: compile errors only from handlers.go (missing logStore/logger fields) — session.go should be clean.

- [ ] **Step 3: Commit**

```bash
git add zarf/sidecar/session.go
git commit -m "feat(sidecar): add logger to session manager"
```

---

### Task 5: Update `handlers.go` — add fields, logging, log endpoints

**Files:**
- Modify: `zarf/sidecar/handlers.go`

- [ ] **Step 1: Add logStore and logger fields to handlers struct**

Update the `handlers` struct:

```go
type handlers struct {
	composeFile string
	session     *SessionManager
	apiKey      string
	logStore    *LogStore
	logger      *Logger
}
```

- [ ] **Step 2: Add log persistence to the inference handler**

In the `inference` method, after `h.session.requests = append(h.session.requests, metric)` on the error path (around line 281), add:

```go
		h.logStore.Append(RequestLog{
			ID:           metric.ID,
			Timestamp:    metric.Timestamp,
			DurationMs:   metric.DurationMs,
			AgentModel:   metric.AgentModel,
			PromptPrefix: metric.PromptPrefix,
			InputTokens:  metric.InputTokens,
			OutputTokens: metric.OutputTokens,
			SessionID:    h.session.sessionID,
			Success:      false,
			Error:        metric.Error,
			Prompt:       req.Prompt,
		})
		h.logger.Error("inference failed", map[string]any{
			"request_id": metric.ID,
			"model":      req.Model,
			"duration_ms": metric.DurationMs,
			"error":      metric.Error,
		})
```

On the success path, after `h.session.requests = append(h.session.requests, metric)` (around line 304), add:

```go
		h.logStore.Append(RequestLog{
			ID:           metric.ID,
			Timestamp:    metric.Timestamp,
			DurationMs:   metric.DurationMs,
			AgentModel:   metric.AgentModel,
			PromptPrefix: metric.PromptPrefix,
			InputTokens:  metric.InputTokens,
			OutputTokens: metric.OutputTokens,
			SessionID:    h.session.sessionID,
			Success:      true,
			Prompt:       req.Prompt,
			Result:       result,
		})
		h.logger.Info("inference complete", map[string]any{
			"request_id":   metric.ID,
			"model":        req.Model,
			"duration_ms":  metric.DurationMs,
			"input_tokens": metric.InputTokens,
			"output_tokens": metric.OutputTokens,
		})
```

- [ ] **Step 3: Add operational logging to authMiddleware**

In `main.go`, update `authMiddleware` to accept a logger and log auth failures:

```go
func authMiddleware(apiKey string, next http.Handler, logger *Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != apiKey {
			logger.Warn("auth failure", map[string]any{
				"remote": r.RemoteAddr,
				"path":   r.URL.Path,
			})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

Update the call in `main()`:

```go
	handler := authMiddleware(*apiKey, mux, logger)
```

- [ ] **Step 4: Add GET /logs/sidecar endpoint**

Add the `sidecarLogs` handler to `handlers.go`:

```go
// =========================================================================
// GET /logs/sidecar?since=...&until=...&limit=50&success=true|false

func (h *handlers) sidecarLogs(w http.ResponseWriter, r *http.Request) {
	filter := LogFilter{Limit: 50}

	if v := r.URL.Query().Get("since"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.Since = t
		}
	}
	if v := r.URL.Query().Get("until"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.Until = t
		}
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			filter.Limit = n
		}
	}
	if v := r.URL.Query().Get("success"); v != "" {
		b := v == "true"
		filter.Success = &b
	}

	entries, err := h.logStore.Query(filter)
	if err != nil {
		writeError(w, 500, "failed to read logs: "+err.Error())
		return
	}

	writeJSON(w, entries)
}
```

- [ ] **Step 5: Add GET /logs/sidecar/stats endpoint**

Add the `sidecarLogStats` handler to `handlers.go`:

```go
// =========================================================================
// GET /logs/sidecar/stats?since=...

func (h *handlers) sidecarLogStats(w http.ResponseWriter, r *http.Request) {
	var since time.Time
	if v := r.URL.Query().Get("since"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			since = t
		}
	}

	stats, err := h.logStore.Stats(since)
	if err != nil {
		writeError(w, 500, "failed to compute stats: "+err.Error())
		return
	}

	writeJSON(w, stats)
}
```

- [ ] **Step 6: Fix route registration order**

In `main.go`, the route `GET /logs/sidecar` must be registered BEFORE `GET /logs/{service}`, otherwise the mux will match `sidecar` as a `{service}` path value. Reorder:

```go
	mux.HandleFunc("GET /logs/sidecar/stats", h.sidecarLogStats)
	mux.HandleFunc("GET /logs/sidecar", h.sidecarLogs)
	mux.HandleFunc("GET /logs/{service}", h.logs)
```

- [ ] **Step 7: Verify it compiles**

Run: `cd zarf/sidecar && go build ./...`
Expected: clean build

- [ ] **Step 8: Commit**

```bash
git add zarf/sidecar/handlers.go zarf/sidecar/main.go
git commit -m "feat(sidecar): persist inference logs, add query endpoints"
```

---

### Task 6: Update systemd service for log directory

**Files:**
- Modify: `zarf/sidecar/planner-sidecar.service`

- [ ] **Step 1: Add LogsDirectory directive**

Update the `[Service]` section to include:

```ini
[Service]
Type=simple
User=deployer
ExecStart=/opt/planner/zarf/sidecar/sidecar --compose-file=/opt/planner/zarf/compose/docker-compose.yml
Restart=on-failure
RestartSec=5
EnvironmentFile=/opt/planner/.env
LogsDirectory=planner
```

`LogsDirectory=planner` tells systemd to create `/var/log/planner/` owned by the service user (`deployer`) on startup.

- [ ] **Step 2: Commit**

```bash
git add zarf/sidecar/planner-sidecar.service
git commit -m "feat(sidecar): add LogsDirectory for persistent log storage"
```

---

### Task 7: Build and smoke test

- [ ] **Step 1: Build the sidecar binary**

Run: `cd zarf/sidecar && go build -o sidecar .`
Expected: clean build, `sidecar` binary produced

- [ ] **Step 2: Verify binary runs with --help**

Run: `cd zarf/sidecar && ./sidecar --help`
Expected: shows flag usage including `--addr`, `--api-key`, `--compose-file`

- [ ] **Step 3: Commit the final state**

```bash
git add -A zarf/sidecar/
git commit -m "feat(sidecar): complete logging system — persistent logs, structured logging, query endpoints"
```
