package checkapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	hdl := &app{db: cfg.DB}

	a.HandleNoMiddleware(http.MethodGet, "/api/v1/readiness", hdl.readiness)
	a.HandleNoMiddleware(http.MethodGet, "/api/v1/liveness", hdl.liveness)
}
