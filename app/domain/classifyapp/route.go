package classifyapp

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
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/notebus/stores/notedb"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/taskbus/stores/taskdb"
	"github.com/casebrophy/planner/foundation/web"
)

// Routes registers classifyapp routes.
type Routes struct{}

// Add wires up the classify endpoint.
func (Routes) Add(a *web.App, cfg mux.Config) {
	taskStore := taskdb.NewStore(cfg.Log, cfg.DB)
	depStore := taskdb.NewDependencyStore(cfg.Log, cfg.DB)
	taskBus := taskbus.NewBusiness(cfg.Log, taskStore, depStore)

	noteStore := notedb.NewStore(cfg.Log, cfg.DB)
	noteBus := notebus.NewBusiness(cfg.Log, noteStore)

	evtStore := eventdb.NewStore(cfg.Log, cfg.DB)
	evtBus := eventbus.NewBusiness(cfg.Log, evtStore)

	ctxStore := contextdb.NewStore(cfg.Log, cfg.DB)
	ctxBus := contextbus.NewBusiness(cfg.Log, ctxStore)

	clStore := clarificationdb.NewStore(cfg.Log, cfg.DB)
	clBus := clarificationbus.NewBusiness(cfg.Log, clStore)

	claudeExt := extractor.NewClaudeCodeExtractor(cfg.ClaudeCLI)
	var ext extractor.Extractor = claudeExt
	if cfg.OllamaEnabled && cfg.OllamaURL != "" {
		ollamaExt := extractor.NewOllamaExtractor(cfg.OllamaURL, cfg.OllamaModel)
		ext = extractor.NewFailoverExtractor(cfg.Log, claudeExt, ollamaExt)
	}

	hdl := &app{
		taskBus:          taskBus,
		noteBus:          noteBus,
		eventBus:         evtBus,
		contextBus:       ctxBus,
		clarificationBus: clBus,
		extractor:        ext,
	}

	authen := mid.Auth(cfg.APIKey)

	// Backward-compatible route (entity_type defaults to "task")
	a.Handle(http.MethodPost, "/api/v1/tasks/classify", hdl.classify, authen)
	// New unified route
	a.Handle(http.MethodPost, "/api/v1/classify", hdl.classify, authen)
}
