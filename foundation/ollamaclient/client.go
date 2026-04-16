// Package ollamaclient provides a shared, FIFO-serialized HTTP client for all
// Ollama traffic. All requests flow through a single worker goroutine so that
// a local Ollama server sees at most one request at a time. Once a job is
// enqueued, it runs to completion regardless of the caller's context; the
// caller's ctx only governs the enqueue wait.
package ollamaclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultTimeout   = 180 * time.Second
	defaultQueueSize = 64
)

// Config configures a Client.
type Config struct {
	BaseURL   string        // e.g. http://ollama:11434 (required)
	Timeout   time.Duration // per-request; default 180s
	QueueSize int           // FIFO buffer; default 64
}

// Client serializes all Ollama HTTP requests through a single worker goroutine.
// Once a job is enqueued, it runs to completion regardless of the caller's
// context; caller ctx only governs the enqueue-wait.
type Client struct {
	baseURL string
	hc      *http.Client
	timeout time.Duration
	queue   chan *job
	wg      sync.WaitGroup
	closed  atomic.Bool
}

type job struct {
	buildReq func(ctx context.Context) (*http.Request, error)
	result   chan jobResult
}

type jobResult struct {
	body []byte
	err  error
}

// ErrClosed is returned by Do when the client has been closed.
var ErrClosed = errors.New("ollamaclient: client closed")

// New constructs a Client and starts its worker goroutine.
func New(cfg Config) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = defaultQueueSize
	}
	c := &Client{
		baseURL: cfg.BaseURL,
		// No http.Client.Timeout: we use per-request ctx deadlines instead so
		// the worker retains full control over request lifetime.
		hc:      &http.Client{},
		timeout: cfg.Timeout,
		queue:   make(chan *job, cfg.QueueSize),
	}
	c.wg.Add(1)
	go c.run()
	return c
}

func (c *Client) run() {
	defer c.wg.Done()
	for j := range c.queue {
		c.process(j)
	}
}

func (c *Client) process(j *job) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	req, err := j.buildReq(ctx)
	if err != nil {
		j.result <- jobResult{err: err}
		return
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		j.result <- jobResult{err: err}
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		j.result <- jobResult{err: err}
		return
	}
	if resp.StatusCode != http.StatusOK {
		j.result <- jobResult{body: body, err: fmt.Errorf("ollama: unexpected status %d: %s", resp.StatusCode, string(body))}
		return
	}
	j.result <- jobResult{body: body}
}

// Do enqueues a request-builder and waits for the result. Caller ctx governs
// only the enqueue wait; once enqueued, the job runs to completion.
func (c *Client) Do(ctx context.Context, buildReq func(context.Context) (*http.Request, error)) ([]byte, error) {
	if c.closed.Load() {
		return nil, ErrClosed
	}
	j := &job{
		buildReq: buildReq,
		result:   make(chan jobResult, 1),
	}
	select {
	case c.queue <- j:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	res := <-j.result
	return res.body, res.err
}

// Close stops accepting new jobs and waits for the worker to drain the queue.
func (c *Client) Close() {
	if !c.closed.CompareAndSwap(false, true) {
		return
	}
	close(c.queue)
	c.wg.Wait()
}

// --- Typed helpers ---

type embedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// Embed calls POST /api/embed.
func (c *Client) Embed(ctx context.Context, model string, texts []string) ([][]float32, error) {
	body, err := c.Do(ctx, func(ctx context.Context) (*http.Request, error) {
		data, err := json.Marshal(embedRequest{Model: model, Input: texts})
		if err != nil {
			return nil, fmt.Errorf("ollama: marshal embed request: %w", err)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/embed", bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("ollama: create embed request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	})
	if err != nil {
		return nil, fmt.Errorf("ollama: do request: %w", err)
	}
	var out embedResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("ollama: decode embed response: %w", err)
	}
	return out.Embeddings, nil
}

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format"`
}

type generateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// Generate calls POST /api/generate with JSON-mode output. Returns the raw
// `response` string (typically a JSON document the caller unmarshals).
func (c *Client) Generate(ctx context.Context, model, prompt string) (string, error) {
	body, err := c.Do(ctx, func(ctx context.Context) (*http.Request, error) {
		data, err := json.Marshal(generateRequest{Model: model, Prompt: prompt, Stream: false, Format: "json"})
		if err != nil {
			return nil, fmt.Errorf("ollama: marshal generate request: %w", err)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/generate", bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("ollama: create generate request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	})
	if err != nil {
		return "", fmt.Errorf("ollama: do request: %w", err)
	}
	var out generateResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("ollama: decode generate response: %w", err)
	}
	return out.Response, nil
}
