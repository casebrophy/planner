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
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"time"

	"github.com/casebrophy/planner/foundation/logger"
)

// Client executes inference via the Claude Code CLI or a sidecar HTTP endpoint.
type Client struct {
	cliPath    string
	models     []string
	timeout    time.Duration
	log        *logger.Logger
	sidecarURL string
	sidecarKey string
	httpClient *http.Client
}

// NewClient creates a Client with the given CLI path and model escalation chain.
// Models are tried in order; the first model that produces acceptable results wins.
func NewClient(log *logger.Logger, cliPath string, models []string) *Client {
	if cliPath == "" {
		cliPath = "claude"
	}
	if len(models) == 0 {
		models = []string{"haiku", "sonnet", "opus"}
	}

	return &Client{
		cliPath: cliPath,
		models:  models,
		timeout: 120 * time.Second,
		log:     log,
	}
}

// SetSidecar configures the client to route inference through the sidecar
// HTTP endpoint instead of the local CLI. When set, run() POSTs to
// {sidecarURL}/inference with the given API key.
func (c *Client) SetSidecar(url, apiKey string) {
	c.sidecarURL = url
	c.sidecarKey = apiKey
	if url != "" {
		c.httpClient = &http.Client{Timeout: c.timeout}
	}
}

// RunJSON sends a prompt to the CLI and unmarshals the JSON response into dest.
// It tries models in escalation order. After each successful parse, it calls
// shouldEscalate (if non-nil) to check result quality. If shouldEscalate returns
// true, the next model in the chain is tried. The last model's result is always
// accepted.
func (c *Client) RunJSON(ctx context.Context, prompt string, schema string, dest any, shouldEscalate func() bool) error {
	var lastErr error

	for i, model := range c.models {
		lastModel := i == len(c.models)-1

		raw, err := c.run(ctx, prompt, schema, model)
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

		c.log.Info(ctx, "claudecli", "msg", "inference complete", "model", model, "escalated", i > 0)
		return nil
	}

	return fmt.Errorf("all models exhausted: %w", lastErr)
}

// run executes a single inference call and returns the raw result bytes.
// When sidecarURL is set, it POSTs to the sidecar; otherwise it shells out
// to the claude CLI.
func (c *Client) run(ctx context.Context, prompt string, schema string, model string) ([]byte, error) {
	if c.sidecarURL != "" {
		return c.runHTTP(ctx, prompt, schema, model)
	}
	return c.runCLI(ctx, prompt, schema, model)
}

// runHTTP sends the inference request to the sidecar's /inference endpoint.
func (c *Client) runHTTP(ctx context.Context, prompt string, schema string, model string) ([]byte, error) {
	reqBody := struct {
		Prompt string `json:"prompt"`
		Schema string `json:"schema,omitempty"`
		Model  string `json:"model"`
	}{
		Prompt: prompt,
		Schema: schema,
		Model:  model,
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

		// Check for nested CLI envelope: {"type":"result","result":"<actual content>",...}
		var nested struct {
			Result string `json:"result"`
			Type   string `json:"type"`
		}
		if err := json.Unmarshal([]byte(result), &nested); err == nil && nested.Type == "result" && nested.Result != "" {
			c.log.Info(ctx, "claudecli", "msg", "sidecar unwrapped nested CLI envelope", "model", model, "result", nested.Result)
			return []byte(nested.Result), nil
		}

		c.log.Info(ctx, "claudecli", "msg", "sidecar extracted result", "model", model, "result", result)
		return []byte(result), nil
	}

	c.log.Info(ctx, "claudecli", "msg", "sidecar no envelope, using raw body", "model", model)
	return respBody, nil
}

// runCLI executes a single claude -p call and returns the raw stdout bytes.
func (c *Client) runCLI(ctx context.Context, prompt string, schema string, model string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	args := []string{
		"-p", prompt,
		"--output-format", "json",
		"--model", model,
		"--bare",
	}

	if schema != "" {
		args = append(args, "--json-schema", schema)
	}

	cmd := exec.CommandContext(ctx, c.cliPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("claude cli (%s): %w, stderr: %s", model, err, stderr.String())
	}

	// The --output-format json wraps the response in a JSON envelope.
	// Extract the actual result text from it.
	var envelope struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err == nil && envelope.Result != "" {
		return []byte(envelope.Result), nil
	}

	// If no envelope, return raw stdout.
	return stdout.Bytes(), nil
}
