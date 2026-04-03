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
	Time   string         `json:"time"`
	Level  string         `json:"level"`
	Msg    string         `json:"msg"`
	Fields map[string]any `json:"fields,omitempty"`
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
