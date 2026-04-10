package mlclient

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

// ErrServiceUnavailable is returned by stub methods until the Python ML service is deployed.
var ErrServiceUnavailable = errors.New("ml service unavailable")

// Client is an HTTP wrapper for the Python ML service.
type Client struct {
	url    string
	client *http.Client
	log    *logger.Logger
}

// NewClient creates a new ML service client with a 30-second timeout.
func NewClient(url string, log *logger.Logger) *Client {
	return &Client{
		url: url,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		log: log,
	}
}

// ClusterTasks sends tasks to the /cluster endpoint for clustering.
// This is a stub that returns ErrServiceUnavailable until the Python service exists.
func (c *Client) ClusterTasks(ctx context.Context, tasks []TaskInput) ([]ClusterResult, error) {
	reqBody := clusterRequest{
		Tasks: tasks,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("mlclient: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/cluster", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("mlclient: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, ErrServiceUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, ErrServiceUnavailable
	}

	var clusterResp clusterResponse
	if err := json.NewDecoder(resp.Body).Decode(&clusterResp); err != nil {
		return nil, fmt.Errorf("mlclient: decode response: %w", err)
	}

	return clusterResp.Results, nil
}

// FindSimilarSituations sends a context query to the /similar endpoint.
// This is a stub that returns ErrServiceUnavailable until the Python service exists.
func (c *Client) FindSimilarSituations(ctx context.Context, contextID string, embeddings [][]float32) ([]SimilarMatch, error) {
	reqBody := similarRequest{
		ContextID:  contextID,
		Embeddings: embeddings,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("mlclient: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/similar", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("mlclient: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, ErrServiceUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, ErrServiceUnavailable
	}

	var similarResp similarResponse
	if err := json.NewDecoder(resp.Body).Decode(&similarResp); err != nil {
		return nil, fmt.Errorf("mlclient: decode response: %w", err)
	}

	return similarResp.Matches, nil
}

// Health checks if the ML service is available.
// This is a stub that returns ErrServiceUnavailable until the Python service exists.
func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"/health", nil)
	if err != nil {
		return fmt.Errorf("mlclient: create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return ErrServiceUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return ErrServiceUnavailable
	}

	return nil
}

// clusterRequest is the request body for the /cluster endpoint.
type clusterRequest struct {
	Tasks []TaskInput `json:"tasks"`
}

// clusterResponse is the response body from the /cluster endpoint.
type clusterResponse struct {
	Results []ClusterResult `json:"results"`
}

// similarRequest is the request body for the /similar endpoint.
type similarRequest struct {
	ContextID  string      `json:"context_id"`
	Embeddings [][]float32 `json:"embeddings"`
}

// similarResponse is the response body from the /similar endpoint.
type similarResponse struct {
	Matches []SimilarMatch `json:"matches"`
}
