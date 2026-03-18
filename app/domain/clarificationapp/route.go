package clarificationapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/clarificationbus/stores/clarificationdb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	store := clarificationdb.NewStore(cfg.Log, cfg.DB)
	bus := clarificationbus.NewBusiness(cfg.Log, store)

	hdl := &app{clarificationBus: bus}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/clarifications", hdl.queryQueue, authen)
	a.Handle(http.MethodGet, "/api/v1/clarifications/count", hdl.countPending, authen)
	a.Handle(http.MethodGet, "/api/v1/clarifications/{id}", hdl.queryByID, authen)
	a.Handle(http.MethodPost, "/api/v1/clarifications/{id}/resolve", hdl.resolve, authen)
	a.Handle(http.MethodPost, "/api/v1/clarifications/{id}/snooze", hdl.snooze, authen)
	a.Handle(http.MethodPost, "/api/v1/clarifications/{id}/dismiss", hdl.dismiss, authen)
}
