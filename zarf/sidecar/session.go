package main

import (
	"sync"
	"time"
)

const maxSessionHistory = 100

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

type SessionSummary struct {
	SessionID       string    `json:"session_id"`
	CreatedAt       time.Time `json:"created_at"`
	EndedAt         time.Time `json:"ended_at"`
	TotalRequests   int       `json:"total_requests"`
	EndReason       string    `json:"end_reason"`
	PeakInputTokens int      `json:"peak_input_tokens"`
}

type ToolCallMetric struct {
	Timestamp  time.Time `json:"timestamp"`
	SessionID  string    `json:"session_id"`
	RequestID  string    `json:"request_id"`
	ToolName   string    `json:"tool_name"`
	DurationMs int       `json:"duration_ms"`
}

// NewSessionManager creates a SessionManager with the given configuration.
func NewSessionManager(systemPrompt string, contextMax int, timeout time.Duration, mcpURL string, logger *Logger) *SessionManager {
	return &SessionManager{
		systemPrompt: systemPrompt,
		contextMax:   contextMax,
		timeout:      timeout,
		mcpURL:       mcpURL,
		logger:       logger,
	}
}

// rotate archives the current session and clears state. Must be called with mu held.
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
		if len(sm.sessions) > maxSessionHistory {
			sm.sessions = sm.sessions[len(sm.sessions)-maxSessionHistory:]
		}
	}
	sm.sessionID = ""
	sm.createdAt = time.Time{}
	sm.requests = nil
	sm.toolCalls = nil
	sm.logger.Info("session rotated", map[string]any{
		"session_id":     summary.SessionID,
		"reason":         reason,
		"total_requests": summary.TotalRequests,
		"peak_tokens":    summary.PeakInputTokens,
	})
	return summary
}

// StatusResponse holds session health information.
type StatusResponse struct {
	SessionID             string    `json:"session_id"`
	CreatedAt             time.Time `json:"created_at,omitempty"`
	AgeSeconds            float64   `json:"age_seconds"`
	TotalRequests         int       `json:"total_requests"`
	LatestInputTokens     int       `json:"latest_input_tokens"`
	ContextMax            int       `json:"context_max"`
	ContextUsagePct       float64   `json:"context_usage_pct"`
	TokenGrowth           []int     `json:"token_growth"`
	AvgDurationMs         int       `json:"avg_duration_ms"`
	DurationTrend         []int     `json:"duration_trend"`
	RequestsSinceRotation int       `json:"requests_since_rotation"`
}

// Status returns a StatusResponse with session health. Must be called with mu held.
func (sm *SessionManager) Status() StatusResponse {
	resp := StatusResponse{
		SessionID:  sm.sessionID,
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

// HistoryResponse holds all past session summaries.
type HistoryResponse struct {
	Sessions []SessionSummary `json:"sessions"`
}

// History returns a HistoryResponse with all past session summaries. Must be called with mu held.
func (sm *SessionManager) History() HistoryResponse {
	sessions := sm.sessions
	if sessions == nil {
		sessions = []SessionSummary{}
	}
	return HistoryResponse{Sessions: sessions}
}

// ToolsResponse holds tool call frequency data.
type ToolsResponse struct {
	ToolFrequency  map[string]int `json:"tool_frequency"`
	AvgCallsPerReq float64        `json:"avg_calls_per_request"`
}

// Tools returns a ToolsResponse with tool call frequency data. Must be called with mu held.
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
