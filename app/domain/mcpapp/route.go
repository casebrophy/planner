package mcpapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/clarificationbus/stores/clarificationdb"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/contextbus/stores/contextdb"
	"github.com/casebrophy/planner/business/domain/emailbus"
	"github.com/casebrophy/planner/business/domain/emailbus/stores/emaildb"
	"github.com/casebrophy/planner/business/domain/observationbus"
	"github.com/casebrophy/planner/business/domain/observationbus/stores/observationdb"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/taskbus/stores/taskdb"
	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/business/domain/threadbus/stores/threaddb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	taskStore := taskdb.NewStore(cfg.Log, cfg.DB)
	taskBus := taskbus.NewBusiness(cfg.Log, taskStore)

	ctxStore := contextdb.NewStore(cfg.Log, cfg.DB)
	ctxBus := contextbus.NewBusiness(cfg.Log, ctxStore)

	emStore := emaildb.NewStore(cfg.Log, cfg.DB)
	emBus := emailbus.NewBusiness(cfg.Log, emStore)

	clStore := clarificationdb.NewStore(cfg.Log, cfg.DB)
	clBus := clarificationbus.NewBusiness(cfg.Log, clStore)

	thStore := threaddb.NewStore(cfg.Log, cfg.DB)
	thBus := threadbus.NewBusiness(cfg.Log, thStore)

	obStore := observationdb.NewStore(cfg.Log, cfg.DB)
	obBus := observationbus.NewBusiness(cfg.Log, obStore)

	hdl := &app{
		taskBus:          taskBus,
		contextBus:       ctxBus,
		emailBus:         emBus,
		clarificationBus: clBus,
		threadBus:        thBus,
		observationBus:   obBus,
	}

	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodPost, "/mcp", hdl.handle, authen)
}
