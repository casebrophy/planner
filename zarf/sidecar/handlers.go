package main

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

type handlers struct {
	composeFile string
	session     *SessionManager
}

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

// =========================================================================
// GET /containers

type ContainerInfo struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Status string `json:"status"`
	Health string `json:"health"`
	Image  string `json:"image"`
	Ports  string `json:"ports"`
}

func (h *handlers) containers(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command(
		"docker", "compose", "-f", h.composeFile,
		"ps", "--format", "json", "-a",
	).Output()
	if err != nil {
		writeError(w, 500, "docker compose ps failed: "+err.Error())
		return
	}

	var containers []ContainerInfo
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		var raw struct {
			Name   string `json:"Name"`
			State  string `json:"State"`
			Status string `json:"Status"`
			Health string `json:"Health"`
			Image  string `json:"Image"`
			Ports  string `json:"Ports"`
		}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		containers = append(containers, ContainerInfo{
			Name:   raw.Name,
			State:  raw.State,
			Status: raw.Status,
			Health: raw.Health,
			Image:  raw.Image,
			Ports:  raw.Ports,
		})
	}
	if containers == nil {
		containers = []ContainerInfo{}
	}
	writeJSON(w, containers)
}

// =========================================================================
// GET /timers

type TimerInfo struct {
	Name    string `json:"name"`
	Active  bool   `json:"active"`
	LastRun string `json:"lastRun"`
	NextRun string `json:"nextRun"`
	Result  string `json:"result"`
}

func (h *handlers) timers(w http.ResponseWriter, r *http.Request) {
	units := []string{"planner-deploy", "planner-backup"}
	var timers []TimerInfo

	for _, unit := range units {
		t := TimerInfo{Name: unit}

		activeOut, _ := exec.Command("systemctl", "is-active", unit+".timer").Output()
		t.Active = strings.TrimSpace(string(activeOut)) == "active"

		lastOut, _ := exec.Command("systemctl", "show", unit+".timer", "--property=LastTriggerUSec", "--value").Output()
		t.LastRun = strings.TrimSpace(string(lastOut))

		nextOut, _ := exec.Command("systemctl", "show", unit+".timer", "--property=NextElapseUSecRealtime", "--value").Output()
		t.NextRun = strings.TrimSpace(string(nextOut))

		resultOut, _ := exec.Command("systemctl", "show", unit+".service", "--property=Result", "--value").Output()
		t.Result = strings.TrimSpace(string(resultOut))

		timers = append(timers, t)
	}

	writeJSON(w, timers)
}

// =========================================================================
// GET /logs/{service}?lines=100

func (h *handlers) logs(w http.ResponseWriter, r *http.Request) {
	service := r.PathValue("service")

	allowed := map[string]bool{
		"backend": true, "frontend": true, "db": true,
		"planner-deploy": true, "planner-backup": true,
	}
	if !allowed[service] {
		writeError(w, 400, "unknown service: "+service)
		return
	}

	lines := 100
	if n := r.URL.Query().Get("lines"); n != "" {
		if parsed, err := strconv.Atoi(n); err == nil && parsed > 0 && parsed <= 500 {
			lines = parsed
		}
	}

	var out []byte
	var err error

	switch service {
	case "planner-deploy", "planner-backup":
		out, err = exec.Command(
			"journalctl", "-u", service+".service",
			"--no-pager", "-n", strconv.Itoa(lines), "--output=short-iso",
		).Output()
	default:
		out, err = exec.Command(
			"docker", "compose", "-f", h.composeFile,
			"logs", "--tail", strconv.Itoa(lines), "--no-color", service,
		).Output()
	}
	if err != nil {
		writeError(w, 500, "failed to get logs: "+err.Error())
		return
	}

	writeJSON(w, map[string]string{"logs": string(out)})
}

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
		errMsg := err.Error()
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
	var events []map[string]any
	if err := json.Unmarshal(data, &events); err != nil {
		return "", string(data), 0, 0
	}

	for _, evt := range events {
		typ, _ := evt["type"].(string)

		if typ == "system" {
			if sid, ok := evt["session_id"].(string); ok && sid != "" {
				sessionID = sid
			}
		}

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

	if result == "" {
		result = string(data)
	}

	return sessionID, result, inputTokens, outputTokens
}

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

// =========================================================================
// GET /claude

type ClaudeInstance struct {
	PID     string `json:"pid"`
	Command string `json:"command"`
	CPU     string `json:"cpu"`
	Memory  string `json:"memory"`
	Elapsed string `json:"elapsed"`
}

func (h *handlers) claude(w http.ResponseWriter, r *http.Request) {
	out, _ := exec.Command("ps", "aux").Output()

	var instances []ClaudeInstance
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "claude") || strings.Contains(line, "grep") || strings.Contains(line, "sidecar") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}
		instances = append(instances, ClaudeInstance{
			PID:     fields[1],
			CPU:     fields[2],
			Memory:  fields[3],
			Elapsed: fields[9],
			Command: strings.Join(fields[10:], " "),
		})
	}
	if instances == nil {
		instances = []ClaudeInstance{}
	}
	writeJSON(w, instances)
}
