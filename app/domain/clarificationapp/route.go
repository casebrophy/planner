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
	"github.com/casebrophy/planner/business/domain/entitylinkbus"
	"github.com/casebrophy/planner/business/domain/entitylinkbus/stores/entitylinkdb"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/eventbus/stores/eventdb"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/notebus/stores/notedb"
	"github.com/casebrophy/planner/business/domain/observationbus"
	"github.com/casebrophy/planner/business/domain/observationbus/stores/observationdb"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/rawinputbus/stores/rawinputdb"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/taskbus/stores/taskdb"
	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/business/domain/threadbus/stores/threaddb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	clarStore := clarificationdb.NewStore(cfg.Log, cfg.DB)
	clarBus := clarificationbus.NewBusiness(cfg.Log, clarStore)

	tStore := taskdb.NewStore(cfg.Log, cfg.DB)
	tDepStore := taskdb.NewDependencyStore(cfg.Log, cfg.DB)
	tBus := taskbus.NewBusiness(cfg.Log, tStore, tDepStore)

	cStore := contextdb.NewStore(cfg.Log, cfg.DB)
	cBus := contextbus.NewBusiness(cfg.Log, cStore)

	nStore := notedb.NewStore(cfg.Log, cfg.DB)
	nBus := notebus.NewBusiness(cfg.Log, nStore)

	evStore := eventdb.NewStore(cfg.Log, cfg.DB)
	evBus := eventbus.NewBusiness(cfg.Log, evStore)

	emStore := emaildb.NewStore(cfg.Log, cfg.DB)
	emBus := emailbus.NewBusiness(cfg.Log, emStore)

	obsStore := observationdb.NewStore(cfg.Log, cfg.DB)
	obsBus := observationbus.NewBusiness(cfg.Log, obsStore)

	riStore := rawinputdb.NewStore(cfg.Log, cfg.DB)
	riBus := rawinputbus.NewBusiness(cfg.Log, riStore)

	thStore := threaddb.NewStore(cfg.Log, cfg.DB)
	thBus := threadbus.NewBusiness(cfg.Log, thStore)

	elStore := entitylinkdb.NewStore(cfg.Log, cfg.DB)
	elBus := entitylinkbus.NewBusiness(cfg.Log, elStore)

	hdl := &app{
		clarificationBus: clarBus,
		taskBus:          tBus,
		noteBus:          nBus,
		eventBus:         evBus,
		contextBus:       cBus,
		emailBus:         emBus,
		observationBus:   obsBus,
		rawinputBus:      riBus,
		threadBus:        thBus,
		entityLinkBus:    elBus,
	}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/clarifications", hdl.queryQueue, authen)
	a.Handle(http.MethodGet, "/api/v1/clarifications/count", hdl.countPending, authen)
	a.Handle(http.MethodGet, "/api/v1/clarifications/{id}", hdl.queryByID, authen)
	a.Handle(http.MethodPost, "/api/v1/clarifications/{id}/resolve", hdl.resolve, authen)
	a.Handle(http.MethodPost, "/api/v1/clarifications/{id}/snooze", hdl.snooze, authen)
	a.Handle(http.MethodPost, "/api/v1/clarifications/{id}/dismiss", hdl.dismiss, authen)
}
