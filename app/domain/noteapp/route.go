package noteapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/clarificationbus/stores/clarificationdb"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/contextbus/stores/contextdb"
	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/notebus/stores/notedb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	noteStore := notedb.NewStore(cfg.Log, cfg.DB)
	noteBus := notebus.NewBusiness(cfg.Log, noteStore)

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

	hdl := &app{
		noteBus:          noteBus,
		contextBus:       ctxBus,
		clarificationBus: clBus,
		extractor:        ext,
		embeddingBus:     cfg.EmbeddingBus,
	}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/notes", hdl.queryAll, authen)
	a.Handle(http.MethodGet, "/api/v1/notes/{note_id}", hdl.queryByID, authen)
	a.Handle(http.MethodPost, "/api/v1/notes", hdl.create, authen)
	a.Handle(http.MethodPut, "/api/v1/notes/{note_id}", hdl.update, authen)
	a.Handle(http.MethodDelete, "/api/v1/notes/{note_id}", hdl.delete, authen)
}
