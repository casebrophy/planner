package observationapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/observationbus"
	"github.com/casebrophy/planner/business/domain/observationbus/stores/observationdb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	store := observationdb.NewStore(cfg.Log, cfg.DB)
	bus := observationbus.NewBusiness(cfg.Log, store)

	hdl := &app{observationBus: bus}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodPost, "/api/v1/observations", hdl.record, authen)
	a.Handle(http.MethodGet, "/api/v1/observations/{subject_type}/{subject_id}", hdl.queryBySubject, authen)
}
