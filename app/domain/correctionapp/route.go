package correctionapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/classificationcorrectionbus"
	"github.com/casebrophy/planner/business/domain/classificationcorrectionbus/stores/correctiondb"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/eventbus/stores/eventdb"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/notebus/stores/notedb"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/taskbus/stores/taskdb"
	"github.com/casebrophy/planner/foundation/web"
)

// Routes is the Routes struct for correctionapp.
type Routes struct{}

// Add registers the correction routes.
func (Routes) Add(a *web.App, cfg mux.Config) {
	taskStore := taskdb.NewStore(cfg.Log, cfg.DB)
	taskDepStore := taskdb.NewDependencyStore(cfg.Log, cfg.DB)
	tBus := taskbus.NewBusiness(cfg.Log, taskStore, taskDepStore)

	nStore := notedb.NewStore(cfg.Log, cfg.DB)
	nBus := notebus.NewBusiness(cfg.Log, nStore)

	evStore := eventdb.NewStore(cfg.Log, cfg.DB)
	evBus := eventbus.NewBusiness(cfg.Log, evStore)

	corrStore := correctiondb.NewStore(cfg.Log, cfg.DB)
	corrBus := classificationcorrectionbus.NewBusiness(cfg.Log, corrStore)

	hdl := &app{taskBus: tBus, noteBus: nBus, eventBus: evBus, correctionBus: corrBus}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodPost, "/api/v1/corrections", hdl.correct, authen)
}
