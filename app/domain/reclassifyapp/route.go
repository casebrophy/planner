package reclassifyapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/classificationcorrectionbus"
	"github.com/casebrophy/planner/business/domain/classificationcorrectionbus/stores/correctiondb"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/notebus/stores/notedb"
	"github.com/casebrophy/planner/business/domain/reclassifybus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/taskbus/stores/taskdb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	taskStore := taskdb.NewStore(cfg.Log, cfg.DB)
	depStore := taskdb.NewDependencyStore(cfg.Log, cfg.DB)
	taskBus := taskbus.NewBusiness(cfg.Log, taskStore, depStore)

	noteStore := notedb.NewStore(cfg.Log, cfg.DB)
	noteBus := notebus.NewBusiness(cfg.Log, noteStore)

	corrStore := correctiondb.NewStore(cfg.Log, cfg.DB)
	corrBus := classificationcorrectionbus.NewBusiness(cfg.Log, corrStore)

	reclassifyBus := reclassifybus.NewBusiness(cfg.Log, taskBus, noteBus, corrBus, cfg.DB)

	hdl := &app{log: cfg.Log, reclassifyBus: reclassifyBus}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodPost, "/api/v1/tasks/{task_id}/convert-to-note", hdl.convertTaskToNote, authen)
	a.Handle(http.MethodPost, "/api/v1/notes/{note_id}/convert-to-task", hdl.convertNoteToTask, authen)
}
