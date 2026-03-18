package tagapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/tagbus"
	"github.com/casebrophy/planner/business/domain/tagbus/stores/tagdb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	store := tagdb.New(cfg.Log, cfg.DB)
	bus := tagbus.NewBusiness(cfg.Log, store)

	hdl := &app{tagBus: bus}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/tags", hdl.queryAll, authen)
	a.Handle(http.MethodPost, "/api/v1/tags", hdl.create, authen)
	a.Handle(http.MethodDelete, "/api/v1/tags/{tag_id}", hdl.delete, authen)
	a.Handle(http.MethodPost, "/api/v1/tasks/{task_id}/tags/{tag_id}", hdl.addToTask, authen)
	a.Handle(http.MethodDelete, "/api/v1/tasks/{task_id}/tags/{tag_id}", hdl.removeFromTask, authen)
	a.Handle(http.MethodPost, "/api/v1/contexts/{context_id}/tags/{tag_id}", hdl.addToContext, authen)
	a.Handle(http.MethodDelete, "/api/v1/contexts/{context_id}/tags/{tag_id}", hdl.removeFromContext, authen)
	a.Handle(http.MethodGet, "/api/v1/tasks/{task_id}/tags", hdl.queryByTask, authen)
	a.Handle(http.MethodGet, "/api/v1/contexts/{context_id}/tags", hdl.queryByContext, authen)
}
