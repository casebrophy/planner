// Package claudecli wraps the Claude Code CLI (claude -p) for structured
// JSON inference. It supports model escalation — trying cheaper models first
// and bumping up when results are low quality.
//
// When a sidecar URL is configured, inference is routed over HTTP to the
// sidecar's /inference endpoint instead of shelling out to the CLI.
package claudecli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/casebrophy/planner/foundation/logger"
)

// Client executes inference via a sidecar HTTP endpoint.
type Client struct {
	models     []string
	timeout    time.Duration
	log        *logger.Logger
	sidecarURL string
	sidecarKey string
	httpClient *http.Client
	lastModel  string
}

// LastModel returns the model that was used in the most recent successful RunJSON call.
func (c *Client) LastModel() string {
	return c.lastModel
}

// NewClient creates a Client that routes inference through the sidecar HTTP endpoint.
// Models are tried in order; the first model that produces acceptable results wins.
func NewClient(log *logger.Logger, models []string, sidecarURL, sidecarKey string) *Client {
	if len(models) == 0 {
		models = []string{"haiku", "sonnet", "opus"}
	}

	timeout := 120 * time.Second
	return &Client{
		models:     models,
		timeout:    timeout,
		log:        log,
		sidecarURL: sidecarURL,
		sidecarKey: sidecarKey,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// RunOptions configures optional behaviour for a RunJSON call.
type RunOptions struct {
	// Direct bypasses the sidecar orchestrator session and runs claude -p
	// immediately with the embedded prompt. Use when the prompt already contains
	// all needed data and MCP tool access is not required (e.g. plan generation).
	Direct bool
}

// RunJSON sends a prompt to the CLI and unmarshals the JSON response into dest.
// It tries models in escalation order. After each successful parse, it calls
// shouldEscalate (if non-nil) to check result quality. If shouldEscalate returns
// true, the next model in the chain is tried. The last model's result is always
// accepted.
//
// Pass a RunOptions value to enable optional behaviour such as direct mode.
// Existing callers that omit opts retain their current behaviour unchanged.
func (c *Client) RunJSON(ctx context.Context, prompt string, schema string, dest any, shouldEscalate func() bool, opts ...RunOptions) error {
	var lastErr error

	var opt RunOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	for i, model := range c.models {
		lastModel := i == len(c.models)-1

		raw, err := c.run(ctx, prompt, schema, model, opt.Direct)
		if err != nil {
			c.log.Info(ctx, "claudecli", "msg", "cli call failed", "model", model, "error", err)
			lastErr = err
			if !lastModel {
				c.log.Error(ctx, "claudecli", "msg", "failed due to error", "from", model)
				return fmt.Errorf("failed to run claude cli with model %s: %w", model, err)
			}
			return fmt.Errorf("all models failed, last error (%s): %w", model, err)
		}

		if err := json.Unmarshal(raw, dest); err != nil {
			c.log.Info(ctx, "claudecli", "msg", "json parse failed", "model", model, "error", err)
			lastErr = fmt.Errorf("parse json from %s: %w", model, err)
			if !lastModel {
				c.log.Error(ctx, "claudecli", "msg", "failed due to parse failure", "from", model)
				return fmt.Errorf("failed to parse json from with model: %s: %w", model, err)
			}
			return lastErr
		}

		// Check quality if caller provided a threshold.
		if shouldEscalate != nil && !lastModel && shouldEscalate() {
			c.log.Info(ctx, "claudecli", "msg", "escalating due to low quality", "from", model, "to", c.models[i+1])
			continue
		}

		c.lastModel = model
		c.log.Info(ctx, "claudecli", "msg", "inference complete", "model", model, "escalated", i > 0)
		return nil
	}

	return fmt.Errorf("all models exhausted: %w", lastErr)
}

// run executes a single inference call and returns the raw result bytes.
func (c *Client) run(ctx context.Context, prompt string, schema string, model string, direct bool) ([]byte, error) {
	if c.sidecarURL != "" {
		return c.runHTTP(ctx, prompt, schema, model, direct)
	}

	return nil, errors.New("sidecar not configured")
}

// runHTTP sends the inference request to the sidecar's /inference endpoint.
func (c *Client) runHTTP(ctx context.Context, prompt string, schema string, model string, direct bool) ([]byte, error) {
	reqBody := struct {
		Prompt string `json:"prompt"`
		Schema string `json:"schema,omitempty"`
		Model  string `json:"model"`
		Direct bool   `json:"direct,omitempty"`
	}{
		Prompt: prompt,
		Schema: schema,
		Model:  model,
		Direct: direct,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal inference request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.sidecarURL+"/inference", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create inference request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.sidecarKey != "" {
		req.Header.Set("X-API-Key", c.sidecarKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sidecar inference (%s): %w", model, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read sidecar response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sidecar returned %d: %s", resp.StatusCode, string(respBody))
	}

	c.log.Info(ctx, "claudecli", "msg", "sidecar raw response", "model", model, "body", string(respBody))

	// The sidecar returns {"result": "...", "model": "..."}.
	// The result may itself be a Claude CLI JSON envelope containing a nested
	// "result" field with the actual content. Unwrap up to two levels.
	var envelope struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(respBody, &envelope); err == nil && envelope.Result != "" {
		result := envelope.Result

		// Check for nested CLI envelope: {"type":"result","result":"...","structured_output":{...},...}
		// When --json-schema is used, the actual JSON lives in structured_output, not result.
		var nested struct {
			Result           string          `json:"result"`
			Type             string          `json:"type"`
			StructuredOutput json.RawMessage `json:"structured_output"`
		}
		if err := json.Unmarshal([]byte(result), &nested); err == nil && nested.Type == "result" {
			if len(nested.StructuredOutput) > 0 {
				c.log.Info(ctx, "claudecli", "msg", "sidecar unwrapped structured_output from CLI envelope", "model", model)
				return []byte(nested.StructuredOutput), nil
			}
			if nested.Result != "" {
				c.log.Info(ctx, "claudecli", "msg", "sidecar unwrapped nested CLI envelope", "model", model, "result", nested.Result)
				return []byte(nested.Result), nil
			}
		}

		c.log.Info(ctx, "claudecli", "msg", "sidecar extracted result", "model", model, "result", result)
		return []byte(result), nil
	}

	c.log.Info(ctx, "claudecli", "msg", "sidecar no envelope, using raw body", "model", model)
	return respBody, nil
}
