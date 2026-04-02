package serverapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	hdl := &app{
		sidecarURL: cfg.SidecarURL,
		apiKey:     cfg.APIKey,
	}

	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/server/containers", hdl.proxyContainers, authen)
	a.Handle(http.MethodGet, "/api/v1/server/timers", hdl.proxyTimers, authen)
	a.Handle(http.MethodGet, "/api/v1/server/claude", hdl.proxyClaude, authen)
	a.Handle(http.MethodGet, "/api/v1/server/logs/{service}", hdl.proxyLogs, authen)
	a.Handle(http.MethodGet, "/api/v1/server/inference/status", hdl.proxyInferenceStatus, authen)
	a.Handle(http.MethodGet, "/api/v1/server/inference/history", hdl.proxyInferenceHistory, authen)
	a.Handle(http.MethodGet, "/api/v1/server/inference/tools", hdl.proxyInferenceTools, authen)
}
