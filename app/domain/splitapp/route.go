package splitapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/splitbus"
	"github.com/casebrophy/planner/business/domain/splitbus/stores/splitdb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	splitStore := splitdb.NewStore(cfg.Log, cfg.DB)
	splitBus := splitbus.NewBusiness(cfg.Log, splitStore)
	hdl := &app{splitBus: splitBus}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodPost, "/api/v1/splits", hdl.create, authen)
	a.Handle(http.MethodGet, "/api/v1/transactions/{transaction_id}/splits", hdl.queryByTransaction, authen)
	a.Handle(http.MethodPut, "/api/v1/splits/{split_id}", hdl.update, authen)
	a.Handle(http.MethodDelete, "/api/v1/splits/{split_id}", hdl.delete, authen)
}
