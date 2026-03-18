package threadapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/business/domain/threadbus/stores/threaddb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	store := threaddb.NewStore(cfg.Log, cfg.DB)
	bus := threadbus.NewBusiness(cfg.Log, store)

	hdl := &app{threadBus: bus}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodPost, "/api/v1/threads", hdl.addEntry, authen)
	a.Handle(http.MethodGet, "/api/v1/threads/{subject_type}/{subject_id}", hdl.queryThread, authen)
}
