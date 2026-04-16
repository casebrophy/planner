package eventapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/clarificationbus/stores/clarificationdb"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/contextbus/stores/contextdb"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/eventbus/stores/eventdb"
	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/rawinputbus/stores/rawinputdb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	eventStore := eventdb.NewStore(cfg.Log, cfg.DB)
	eventBus := eventbus.NewBusiness(cfg.Log, eventStore)

	ctxStore := contextdb.NewStore(cfg.Log, cfg.DB)
	ctxBus := contextbus.NewBusiness(cfg.Log, ctxStore)

	clStore := clarificationdb.NewStore(cfg.Log, cfg.DB)
	clBus := clarificationbus.NewBusiness(cfg.Log, clStore)

	var ext extractor.Extractor
	if cfg.ClaudeCLI != nil {
		claudeExt := extractor.NewClaudeCodeExtractor(cfg.ClaudeCLI)
		ext = claudeExt
		if cfg.OllamaEnabled && cfg.OllamaURL != "" {
			ollamaExt := extractor.NewOllamaExtractor(cfg.OllamaURL, cfg.OllamaModel)
			ext = extractor.NewFailoverExtractor(cfg.Log, claudeExt, ollamaExt)
		}
	}

	riStore := rawinputdb.NewStore(cfg.Log, cfg.DB)
	riBus := rawinputbus.NewBusiness(cfg.Log, riStore)

	hdl := &app{
		log:              cfg.Log,
		eventBus:         eventBus,
		contextBus:       ctxBus,
		clarificationBus: clBus,
		embeddingBus:     cfg.EmbeddingBus,
		rawinputBus:      riBus,
		extractor:        ext,
	}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/events", hdl.queryAll, authen)
	a.Handle(http.MethodGet, "/api/v1/events/{event_id}", hdl.queryByID, authen)
	a.Handle(http.MethodPost, "/api/v1/events", hdl.create, authen)
	a.Handle(http.MethodPut, "/api/v1/events/{event_id}", hdl.update, authen)
	a.Handle(http.MethodDelete, "/api/v1/events/{event_id}", hdl.delete, authen)
}
