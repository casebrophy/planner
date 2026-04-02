package serverapp

import (
	"context"
	"io"
	"net/http"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	sidecarURL string
	apiKey     string
}

// rawJSON passes through pre-encoded JSON from the sidecar.
type rawJSON struct {
	data   []byte
	status int
}

func (r rawJSON) Encode() ([]byte, string, error) {
	return r.data, "application/json", nil
}

func (r rawJSON) HTTPStatus() int {
	return r.status
}

func (a *app) proxyContainers(ctx context.Context, r *http.Request) web.Encoder {
	return a.forward(ctx, "/containers", "")
}

func (a *app) proxyTimers(ctx context.Context, r *http.Request) web.Encoder {
	return a.forward(ctx, "/timers", "")
}

func (a *app) proxyClaude(ctx context.Context, r *http.Request) web.Encoder {
	return a.forward(ctx, "/claude", "")
}

func (a *app) proxyLogs(ctx context.Context, r *http.Request) web.Encoder {
	service := r.PathValue("service")
	qs := r.URL.RawQuery
	return a.forward(ctx, "/logs/"+service, qs)
}

func (a *app) proxyInferenceStatus(ctx context.Context, r *http.Request) web.Encoder {
	return a.forward(ctx, "/inference/status", "")
}

func (a *app) proxyInferenceHistory(ctx context.Context, r *http.Request) web.Encoder {
	return a.forward(ctx, "/inference/history", "")
}

func (a *app) proxyInferenceTools(ctx context.Context, r *http.Request) web.Encoder {
	return a.forward(ctx, "/inference/tools", "")
}

func (a *app) forward(ctx context.Context, path string, qs string) web.Encoder {
	if a.sidecarURL == "" {
		return errs.Newf(errs.Internal, "sidecar not configured")
	}

	url := a.sidecarURL + path
	if qs != "" {
		url += "?" + qs
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return errs.New(errs.Internal, err)
	}
	req.Header.Set("X-API-Key", a.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errs.Newf(errs.Internal, "sidecar unreachable: %s", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errs.New(errs.Internal, err)
	}

	return rawJSON{data: body, status: resp.StatusCode}
}
