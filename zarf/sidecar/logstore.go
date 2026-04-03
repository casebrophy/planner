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
	TotalRequests  int                   `json:"total_requests"`
	Successes      int                   `json:"successes"`
	Failures       int                   `json:"failures"`
	SuccessRate    float64               `json:"success_rate"`
	AvgDurationMs  int                   `json:"avg_duration_ms"`
	TotalInputTok  int                   `json:"total_input_tokens"`
	TotalOutputTok int                   `json:"total_output_tokens"`
	ByModel        map[string]ModelStats `json:"by_model"`
}

// ModelStats holds per-model aggregated statistics.
type ModelStats struct {
	Requests      int     `json:"requests"`
	AvgDurationMs int     `json:"avg_duration_ms"`
	SuccessRate   float64 `json:"success_rate"`
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
