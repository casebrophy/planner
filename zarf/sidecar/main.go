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

	logger := NewLogger()

	if *apiKey == "" {
		*apiKey = os.Getenv("PLANNER_AUTH_API_KEY")
	}
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

	// Log store configuration from env.
	logPath := os.Getenv("SIDECAR_LOG_PATH")
	if logPath == "" {
		logPath = "/var/log/planner/sidecar-requests.jsonl"
	}

	logRetentionDays := 30
	if v := os.Getenv("SIDECAR_LOG_RETENTION_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			logRetentionDays = n
		}
	}

	logVerbose := false
	if v := os.Getenv("SIDECAR_LOG_VERBOSE"); v == "true" || v == "1" {
		logVerbose = true
	}

	logStore, err := NewLogStore(logPath, logRetentionDays, logVerbose, logger)
	if err != nil {
		logger.Error("failed to create log store", map[string]any{"error": err.Error()})
		os.Exit(1)
	}

	session := NewSessionManager(orchestratorSystemPrompt, contextMax, requestTimeout, mcpURL, logger)

	h := &handlers{
		composeFile: *composeFile,
		session:     session,
		apiKey:      *apiKey,
		logStore:    logStore,
		logger:      logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /containers", h.containers)
	mux.HandleFunc("GET /logs/sidecar/stats", h.sidecarLogStats)
	mux.HandleFunc("GET /logs/sidecar", h.sidecarLogs)
	mux.HandleFunc("GET /logs/{service}", h.logs)
	mux.HandleFunc("GET /claude", h.claude)
	mux.HandleFunc("GET /timers", h.timers)
	mux.HandleFunc("POST /inference", h.inference)
	mux.HandleFunc("GET /inference/status", h.inferenceStatus)
	mux.HandleFunc("GET /inference/history", h.inferenceHistory)
	mux.HandleFunc("GET /inference/tools", h.inferenceTools)
	mux.HandleFunc("POST /inference/rotate", h.inferenceRotate)

	handler := authMiddleware(*apiKey, mux, logger)

	logger.Info("sidecar starting", map[string]any{
		"addr":               *addr,
		"context_max":        contextMax,
		"timeout":            requestTimeout.String(),
		"log_path":           logPath,
		"log_retention_days": logRetentionDays,
		"log_verbose":        logVerbose,
	})

	if err := http.ListenAndServe(*addr, handler); err != nil {
		logger.Error("server failed", map[string]any{"error": err.Error()})
		os.Exit(1)
	}
}

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

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
