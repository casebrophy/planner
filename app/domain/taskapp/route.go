package taskapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
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

	hdl := &app{taskBus: taskBus, threadBus: threadBus}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/tasks", hdl.queryAll, authen)
	a.Handle(http.MethodGet, "/api/v1/tasks/{task_id}", hdl.queryByID, authen)
	a.Handle(http.MethodPost, "/api/v1/tasks", hdl.create, authen)
	a.Handle(http.MethodPut, "/api/v1/tasks/{task_id}", hdl.update, authen)
	a.Handle(http.MethodDelete, "/api/v1/tasks/{task_id}", hdl.delete, authen)

	a.Handle(http.MethodPost, "/api/v1/tasks/{task_id}/dependencies/{depends_on_id}", hdl.addDependency, authen)
	a.Handle(http.MethodDelete, "/api/v1/tasks/{task_id}/dependencies/{depends_on_id}", hdl.removeDependency, authen)
	a.Handle(http.MethodGet, "/api/v1/tasks/{task_id}/dependencies", hdl.queryDependencies, authen)
	a.Handle(http.MethodGet, "/api/v1/tasks/{task_id}/dependents", hdl.queryDependents, authen)
}
