package noteapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/notebus/stores/notedb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	noteStore := notedb.NewStore(cfg.Log, cfg.DB)
	noteBus := notebus.NewBusiness(cfg.Log, noteStore)

	hdl := &app{noteBus: noteBus}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/notes", hdl.queryAll, authen)
	a.Handle(http.MethodGet, "/api/v1/notes/{note_id}", hdl.queryByID, authen)
	a.Handle(http.MethodPost, "/api/v1/notes", hdl.create, authen)
	a.Handle(http.MethodPut, "/api/v1/notes/{note_id}", hdl.update, authen)
	a.Handle(http.MethodDelete, "/api/v1/notes/{note_id}", hdl.delete, authen)
}
