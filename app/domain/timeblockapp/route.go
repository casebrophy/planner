package timeblockapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/timeblockbus"
	"github.com/casebrophy/planner/business/domain/timeblockbus/stores/timeblockdb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	tbStore := timeblockdb.NewStore(cfg.Log, cfg.DB)
	tbBus := timeblockbus.NewBusiness(cfg.Log, tbStore)

	hdl := &app{timeBlockBus: tbBus}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/time-blocks", hdl.queryAll, authen)
	a.Handle(http.MethodGet, "/api/v1/time-blocks/{block_id}", hdl.queryByID, authen)
	a.Handle(http.MethodPost, "/api/v1/time-blocks", hdl.create, authen)
	a.Handle(http.MethodPut, "/api/v1/time-blocks/{block_id}", hdl.update, authen)
	a.Handle(http.MethodDelete, "/api/v1/time-blocks/{block_id}", hdl.delete, authen)
}
