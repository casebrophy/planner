package scheduleapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/eventbus/stores/eventdb"
	"github.com/casebrophy/planner/business/domain/timeblockbus"
	"github.com/casebrophy/planner/business/domain/timeblockbus/stores/timeblockdb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	evStore := eventdb.NewStore(cfg.Log, cfg.DB)
	evBus := eventbus.NewBusiness(cfg.Log, evStore)

	tbStore := timeblockdb.NewStore(cfg.Log, cfg.DB)
	tbBus := timeblockbus.NewBusiness(cfg.Log, tbStore)

	hdl := &app{
		eventBus:     evBus,
		timeBlockBus: tbBus,
	}

	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/schedule", hdl.querySchedule, authen)
}
