package ollamaapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	ollamaURL    string
	extractModel string
	embedModel   string
	enabled      bool
}

// ModelInfo describes one expected model and whether it's available.
type ModelInfo struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
}

// OllamaStatus is the response from the status endpoint.
type OllamaStatus struct {
	Reachable    bool      `json:"reachable"`
	ExtractModel ModelInfo `json:"extractModel"`
	EmbedModel   ModelInfo `json:"embedModel"`
	AllModels    []string  `json:"allModels"`
}

func (s OllamaStatus) Encode() ([]byte, string, error) {
	data, err := json.Marshal(s)
	return data, "application/json", err
}

// ollamaTagsResponse matches the Ollama /api/tags response shape.
type ollamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

var probeClient = &http.Client{Timeout: 3 * time.Second}

func (a *app) status(ctx context.Context, r *http.Request) web.Encoder {
	resp := OllamaStatus{
		ExtractModel: ModelInfo{Name: a.extractModel},
		EmbedModel:   ModelInfo{Name: a.embedModel},
	}

	if !a.enabled || a.ollamaURL == "" {
		return resp
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.ollamaURL+"/api/tags", nil)
	if err != nil {
		return resp
	}

	httpResp, err := probeClient.Do(req)
	if err != nil {
		return resp
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return resp
	}

	resp.Reachable = true

	var tags ollamaTagsResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&tags); err != nil {
		return resp
	}

	available := make(map[string]bool, len(tags.Models))
	for _, m := range tags.Models {
		resp.AllModels = append(resp.AllModels, m.Name)
		available[m.Name] = true
		available[stripTag(m.Name)] = true
	}

	resp.ExtractModel.Available = available[a.extractModel] || available[stripTag(a.extractModel)]
	resp.EmbedModel.Available = available[a.embedModel] || available[stripTag(a.embedModel)]

	return resp
}

// stripTag removes the :tag suffix from a model name.
func stripTag(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == ':' {
			return name[:i]
		}
	}
	return name
}

// pullResult is the response from a pull request.
type pullResult struct {
	Status string `json:"status"`
}

func (p pullResult) Encode() ([]byte, string, error) {
	data, err := json.Marshal(p)
	return data, "application/json", err
}

type pullRequest struct {
	Name   string `json:"name"`
	Stream bool   `json:"stream"`
}

func (a *app) pull(ctx context.Context, r *http.Request) web.Encoder {
	if !a.enabled || a.ollamaURL == "" {
		return errs.Newf(errs.InvalidArgument, "ollama is not enabled")
	}

	model := r.PathValue("model")
	if model != a.extractModel && model != a.embedModel {
		return errs.Newf(errs.InvalidArgument, "unknown model: %s", model)
	}

	body, err := json.Marshal(pullRequest{Name: model, Stream: false})
	if err != nil {
		return errs.Newf(errs.Internal, "marshal pull request: %s", err)
	}

	pullClient := &http.Client{Timeout: 10 * time.Minute}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.ollamaURL+"/api/pull", bytes.NewReader(body))
	if err != nil {
		return errs.Newf(errs.Internal, "create pull request: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")

	httpResp, err := pullClient.Do(req)
	if err != nil {
		return errs.Newf(errs.Internal, "pull model: %s", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return errs.Newf(errs.Internal, "ollama pull returned status %d", httpResp.StatusCode)
	}

	var result pullResult
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		return errs.Newf(errs.Internal, "decode pull response: %s", err)
	}

	if result.Status == "" {
		result.Status = fmt.Sprintf("pulled %s", model)
	}

	return result
}
