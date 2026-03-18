package rawinputapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/rawinputbus/stores/rawinputdb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	riStore := rawinputdb.NewStore(cfg.Log, cfg.DB)
	riBus := rawinputbus.NewBusiness(cfg.Log, riStore)

	hdl := &app{rawInputBus: riBus}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/raw-inputs", hdl.queryAll, authen)
	a.Handle(http.MethodGet, "/api/v1/raw-inputs/{raw_input_id}", hdl.queryByID, authen)
	a.Handle(http.MethodPost, "/api/v1/raw-inputs/{raw_input_id}/reprocess", hdl.reprocess, authen)
}
