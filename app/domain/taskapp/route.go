package taskapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/activitylogbus"
	"github.com/casebrophy/planner/business/domain/activitylogbus/stores/activitylogdb"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/clarificationbus/stores/clarificationdb"
	"github.com/casebrophy/planner/business/domain/debriefbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/taskbus/stores/taskdb"
	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/business/domain/threadbus/stores/threaddb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	taskStore := taskdb.NewStore(cfg.Log, cfg.DB)
	depStore := taskdb.NewDependencyStore(cfg.Log, cfg.DB)
	taskBus := taskbus.NewBusiness(cfg.Log, taskStore, depStore)

	threadStore := threaddb.NewStore(cfg.Log, cfg.DB)
	threadBus := threadbus.NewBusiness(cfg.Log, threadStore)

	clarStore := clarificationdb.NewStore(cfg.Log, cfg.DB)
	clarBus := clarificationbus.NewBusiness(cfg.Log, clarStore)
	debriefBus := debriefbus.NewBusiness(cfg.Log, clarBus, threadBus)

	alStore := activitylogdb.NewStore(cfg.Log, cfg.DB)
	alBus := activitylogbus.NewBusiness(cfg.Log, alStore)

	hdl := &app{taskBus: taskBus, threadBus: threadBus, debriefBus: debriefBus, embeddingBus: cfg.EmbeddingBus, gapBus: cfg.KnowledgeGapBus}
	authen := mid.Auth(cfg.APIKey)
	logActivity := mid.ActivityLog(cfg.Log, alBus)

	a.Handle(http.MethodGet, "/api/v1/tasks", hdl.queryAll, authen)
	a.Handle(http.MethodGet, "/api/v1/tasks/{task_id}", hdl.queryByID, authen)
	a.Handle(http.MethodPost, "/api/v1/tasks", hdl.create, authen)
	a.Handle(http.MethodPut, "/api/v1/tasks/{task_id}", hdl.update, authen, logActivity)
	a.Handle(http.MethodDelete, "/api/v1/tasks/{task_id}", hdl.delete, authen)
	a.Handle(http.MethodDelete, "/api/v1/tasks/batch", hdl.deleteBatch, authen)

	a.Handle(http.MethodPost, "/api/v1/tasks/{task_id}/dependencies/{depends_on_id}", hdl.addDependency, authen)
	a.Handle(http.MethodDelete, "/api/v1/tasks/{task_id}/dependencies/{depends_on_id}", hdl.removeDependency, authen)
	a.Handle(http.MethodGet, "/api/v1/tasks/{task_id}/dependencies", hdl.queryDependencies, authen)
	a.Handle(http.MethodGet, "/api/v1/tasks/{task_id}/dependents", hdl.queryDependents, authen)
}
