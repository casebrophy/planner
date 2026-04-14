package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseClaudeOutput_ToolUseEvents(t *testing.T) {
	events := []map[string]any{
		{"type": "system", "session_id": "sess-abc"},
		{"type": "tool_use", "name": "Bash", "id": "tool-1"},
		{"type": "tool_use", "name": "Read", "id": "tool-2"},
		{"type": "tool_use", "name": "Bash", "id": "tool-3"},
		{"type": "result", "result": "done", "input_tokens": float64(100), "output_tokens": float64(20)},
	}
	data, _ := json.Marshal(events)

	sessionID, result, inputTokens, outputTokens, toolNames := parseClaudeOutput(data)

	if sessionID != "sess-abc" {
		t.Errorf("sessionID = %q, want %q", sessionID, "sess-abc")
	}
	if result != "done" {
		t.Errorf("result = %q, want %q", result, "done")
	}
	if inputTokens != 100 {
		t.Errorf("inputTokens = %d, want 100", inputTokens)
	}
	if outputTokens != 20 {
		t.Errorf("outputTokens = %d, want 20", outputTokens)
	}
	if len(toolNames) != 3 {
		t.Fatalf("len(toolNames) = %d, want 3", len(toolNames))
	}
	want := []string{"Bash", "Read", "Bash"}
	for i, name := range toolNames {
		if name != want[i] {
			t.Errorf("toolNames[%d] = %q, want %q", i, name, want[i])
		}
	}
}

func TestParseClaudeOutput_NoToolUseEvents(t *testing.T) {
	events := []map[string]any{
		{"type": "system", "session_id": "sess-xyz"},
		{"type": "result", "result": "ok", "input_tokens": float64(50), "output_tokens": float64(10)},
	}
	data, _ := json.Marshal(events)

	_, _, _, _, toolNames := parseClaudeOutput(data)
	if len(toolNames) != 0 {
		t.Errorf("expected no tool names, got %v", toolNames)
	}
}

func TestParseClaudeOutput_ToolUseNameEmpty(t *testing.T) {
	events := []map[string]any{
		{"type": "tool_use", "name": ""},
		{"type": "tool_use"},
		{"type": "result", "result": "x"},
	}
	data, _ := json.Marshal(events)

	_, _, _, _, toolNames := parseClaudeOutput(data)
	if len(toolNames) != 0 {
		t.Errorf("expected no tool names for empty/missing names, got %v", toolNames)
	}
}

func TestSessionManagerTools_CommonSequences(t *testing.T) {
	logger := &Logger{}
	sm := NewSessionManager("sys", 10000, 30*time.Second, "http://localhost", logger)
	sm.mu.Lock()
	sm.sessionID = "sess-1"
	sm.createdAt = time.Now()
	sm.requests = []RequestMetric{{ID: "req-1"}, {ID: "req-2"}}
	sm.toolCalls = []ToolCallMetric{
		{RequestID: "req-1", ToolName: "Bash"},
		{RequestID: "req-1", ToolName: "Read"},
		{RequestID: "req-1", ToolName: "Bash"},
		{RequestID: "req-2", ToolName: "Bash"},
		{RequestID: "req-2", ToolName: "Read"},
	}
	resp := sm.Tools()
	sm.mu.Unlock()

	if resp.ToolFrequency["Bash"] != 3 {
		t.Errorf("Bash frequency = %d, want 3", resp.ToolFrequency["Bash"])
	}
	if resp.ToolFrequency["Read"] != 2 {
		t.Errorf("Read frequency = %d, want 2", resp.ToolFrequency["Read"])
	}
	// req-1: Bash→Read, Read→Bash; req-2: Bash→Read
	if resp.CommonSequences["Bash→Read"] != 2 {
		t.Errorf("Bash→Read = %d, want 2", resp.CommonSequences["Bash→Read"])
	}
	if resp.CommonSequences["Read→Bash"] != 1 {
		t.Errorf("Read→Bash = %d, want 1", resp.CommonSequences["Read→Bash"])
	}
}

func TestSessionManagerTools_EmptyToolCalls(t *testing.T) {
	logger := &Logger{}
	sm := NewSessionManager("sys", 10000, 30*time.Second, "http://localhost", logger)
	sm.mu.Lock()
	resp := sm.Tools()
	sm.mu.Unlock()

	if len(resp.ToolFrequency) != 0 {
		t.Errorf("expected empty ToolFrequency, got %v", resp.ToolFrequency)
	}
	if len(resp.CommonSequences) != 0 {
		t.Errorf("expected empty CommonSequences, got %v", resp.CommonSequences)
	}
}
