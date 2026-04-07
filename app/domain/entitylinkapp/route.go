package entitylinkapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/entitylinkbus"
	"github.com/casebrophy/planner/business/domain/entitylinkbus/stores/entitylinkdb"
	"github.com/casebrophy/planner/foundation/web"
)

// Routes registers entitylinkapp routes.
type Routes struct{}

// Add wires up the entity link endpoints.
func (Routes) Add(a *web.App, cfg mux.Config) {
	store := entitylinkdb.NewStore(cfg.Log, cfg.DB)
	bus := entitylinkbus.NewBusiness(cfg.Log, store)

	hdl := &app{entityLinkBus: bus}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/entity-links", hdl.queryByEntity, authen)
	a.Handle(http.MethodPost, "/api/v1/entity-links", hdl.create, authen)
	a.Handle(http.MethodDelete, "/api/v1/entity-links/{link_id}", hdl.delete, authen)
}
