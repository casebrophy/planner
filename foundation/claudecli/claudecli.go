// Package claudecli wraps the Claude Code CLI (claude -p) for structured
// JSON inference. It supports model escalation — trying cheaper models first
// and bumping up when results are low quality.
package claudecli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/casebrophy/planner/foundation/logger"
)

// Client executes inference via the Claude Code CLI.
type Client struct {
	cliPath string
	models  []string
	timeout time.Duration
	log     *logger.Logger
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
				c.log.Error(ctx, "claudecli", "msg", "escalating due to parse failure", "from", model)
				continue
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

// run executes a single claude -p call and returns the raw stdout bytes.
func (c *Client) run(ctx context.Context, prompt string, schema string, model string) ([]byte, error) {
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
