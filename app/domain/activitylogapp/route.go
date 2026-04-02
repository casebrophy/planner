package activitylogapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/activitylogbus"
	"github.com/casebrophy/planner/business/domain/activitylogbus/stores/activitylogdb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	store := activitylogdb.NewStore(cfg.Log, cfg.DB)
	bus := activitylogbus.NewBusiness(cfg.Log, store)

	hdl := &app{logBus: bus}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodPost, "/api/v1/activity-logs", hdl.create, authen)
	a.Handle(http.MethodGet, "/api/v1/activity-logs", hdl.queryAll, authen)
	a.Handle(http.MethodGet, "/api/v1/activity-logs/streaks/{subject_type}/{subject_id}", hdl.streaks, authen)
}
