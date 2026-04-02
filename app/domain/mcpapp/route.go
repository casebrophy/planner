package mcpapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/clarificationbus/stores/clarificationdb"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/contextbus/stores/contextdb"
	"github.com/casebrophy/planner/business/domain/dailyplanbus"
	"github.com/casebrophy/planner/business/domain/dailyplanbus/stores/dailyplandb"
	"github.com/casebrophy/planner/business/domain/debriefbus"
	"github.com/casebrophy/planner/business/domain/emailbus"
	"github.com/casebrophy/planner/business/domain/emailbus/stores/emaildb"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/eventbus/stores/eventdb"
	"github.com/casebrophy/planner/business/domain/timeblockbus"
	"github.com/casebrophy/planner/business/domain/timeblockbus/stores/timeblockdb"
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

	dpStore := dailyplandb.NewStore(cfg.Log, cfg.DB)
	dpBus := dailyplanbus.NewBusiness(cfg.Log, dpStore)

	emStore := emaildb.NewStore(cfg.Log, cfg.DB)
	emBus := emailbus.NewBusiness(cfg.Log, emStore)

	evStore := eventdb.NewStore(cfg.Log, cfg.DB)
	evBus := eventbus.NewBusiness(cfg.Log, evStore)

	tbStore := timeblockdb.NewStore(cfg.Log, cfg.DB)
	tbBus := timeblockbus.NewBusiness(cfg.Log, tbStore)

	clStore := clarificationdb.NewStore(cfg.Log, cfg.DB)
	clBus := clarificationbus.NewBusiness(cfg.Log, clStore)

	thStore := threaddb.NewStore(cfg.Log, cfg.DB)
	thBus := threadbus.NewBusiness(cfg.Log, thStore)

	obStore := observationdb.NewStore(cfg.Log, cfg.DB)
	obBus := observationbus.NewBusiness(cfg.Log, obStore)

	dbBus := debriefbus.NewBusiness(cfg.Log, clBus, thBus)

	hdl := &app{
		taskBus:          taskBus,
		contextBus:       ctxBus,
		emailBus:         emBus,
		eventBus:         evBus,
		timeBlockBus:     tbBus,
		clarificationBus: clBus,
		threadBus:        thBus,
		observationBus:   obBus,
		debriefBus:       dbBus,
		dailyPlanBus:     dpBus,
	}

	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodPost, "/mcp", hdl.handle, authen)
}
