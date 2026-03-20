package clarificationapp

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
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/rawinputbus/stores/rawinputdb"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/taskbus/stores/taskdb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	clarStore := clarificationdb.NewStore(cfg.Log, cfg.DB)
	clarBus := clarificationbus.NewBusiness(cfg.Log, clarStore)

	tStore := taskdb.NewStore(cfg.Log, cfg.DB)
	tBus := taskbus.NewBusiness(cfg.Log, tStore)

	cStore := contextdb.NewStore(cfg.Log, cfg.DB)
	cBus := contextbus.NewBusiness(cfg.Log, cStore)

	emStore := emaildb.NewStore(cfg.Log, cfg.DB)
	emBus := emailbus.NewBusiness(cfg.Log, emStore)

	obsStore := observationdb.NewStore(cfg.Log, cfg.DB)
	obsBus := observationbus.NewBusiness(cfg.Log, obsStore)

	riStore := rawinputdb.NewStore(cfg.Log, cfg.DB)
	riBus := rawinputbus.NewBusiness(cfg.Log, riStore)

	hdl := &app{
		clarificationBus: clarBus,
		taskBus:          tBus,
		contextBus:       cBus,
		emailBus:         emBus,
		observationBus:   obsBus,
		rawinputBus:      riBus,
	}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/clarifications", hdl.queryQueue, authen)
	a.Handle(http.MethodGet, "/api/v1/clarifications/count", hdl.countPending, authen)
	a.Handle(http.MethodGet, "/api/v1/clarifications/{id}", hdl.queryByID, authen)
	a.Handle(http.MethodPost, "/api/v1/clarifications/{id}/resolve", hdl.resolve, authen)
	a.Handle(http.MethodPost, "/api/v1/clarifications/{id}/snooze", hdl.snooze, authen)
	a.Handle(http.MethodPost, "/api/v1/clarifications/{id}/dismiss", hdl.dismiss, authen)
}
