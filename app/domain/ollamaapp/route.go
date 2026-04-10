package ollamaapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	hdl := &app{
		ollamaURL:    cfg.OllamaURL,
		extractModel: cfg.OllamaModel,
		embedModel:   cfg.OllamaEmbedModel,
		enabled:      cfg.OllamaEnabled,
	}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/ollama/status", hdl.status, authen)
	a.Handle(http.MethodPost, "/api/v1/ollama/pull/{model}", hdl.pull, authen)
}
