package contextapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/clarificationbus/stores/clarificationdb"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/contextbus/stores/contextdb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	store := contextdb.NewStore(cfg.Log, cfg.DB)
	bus := contextbus.NewBusiness(cfg.Log, store)

	clarStore := clarificationdb.NewStore(cfg.Log, cfg.DB)
	clarBus := clarificationbus.NewBusiness(cfg.Log, clarStore)

	hdl := &app{contextBus: bus, clarificationBus: clarBus}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/contexts", hdl.queryAll, authen)
	a.Handle(http.MethodGet, "/api/v1/contexts/{context_id}", hdl.queryByID, authen)
	a.Handle(http.MethodPost, "/api/v1/contexts", hdl.create, authen)
	a.Handle(http.MethodPut, "/api/v1/contexts/{context_id}", hdl.update, authen)
	a.Handle(http.MethodDelete, "/api/v1/contexts/{context_id}", hdl.delete, authen)
	a.Handle(http.MethodPost, "/api/v1/contexts/{context_id}/events", hdl.addEvent, authen)
	a.Handle(http.MethodGet, "/api/v1/contexts/{context_id}/events", hdl.queryEvents, authen)
}
