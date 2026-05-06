package dailyplanapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/dailyplanbus"
	"github.com/casebrophy/planner/business/domain/dailyplanbus/stores/dailyplandb"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/taskbus/stores/taskdb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	dpStore := dailyplandb.NewStore(cfg.Log, cfg.DB)
	dpBus := dailyplanbus.NewBusiness(cfg.Log, dpStore)

	taskStore := taskdb.NewStore(cfg.Log, cfg.DB)
	depStore := taskdb.NewDependencyStore(cfg.Log, cfg.DB)
	taskBus := taskbus.NewBusiness(cfg.Log, taskStore, depStore)

	hdl := &app{
		log:          cfg.Log,
		dailyPlanBus: dpBus,
		taskBus:      taskBus,
	}

	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/daily-plan", hdl.getPlan, authen)
	a.Handle(http.MethodPut, "/api/v1/daily-plan/items/{item_id}", hdl.updateItem, authen)
	a.Handle(http.MethodPost, "/api/v1/daily-plan/items/{item_id}/complete", hdl.completeItem, authen)
	a.Handle(http.MethodPost, "/api/v1/daily-plan/items/{item_id}/dismiss", hdl.dismissItem, authen)
}
