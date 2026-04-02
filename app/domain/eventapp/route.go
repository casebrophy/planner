package eventapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/eventbus/stores/eventdb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	eventStore := eventdb.NewStore(cfg.Log, cfg.DB)
	eventBus := eventbus.NewBusiness(cfg.Log, eventStore)

	hdl := &app{eventBus: eventBus}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/events", hdl.queryAll, authen)
	a.Handle(http.MethodGet, "/api/v1/events/{event_id}", hdl.queryByID, authen)
	a.Handle(http.MethodPost, "/api/v1/events", hdl.create, authen)
	a.Handle(http.MethodPut, "/api/v1/events/{event_id}", hdl.update, authen)
	a.Handle(http.MethodDelete, "/api/v1/events/{event_id}", hdl.delete, authen)
}
