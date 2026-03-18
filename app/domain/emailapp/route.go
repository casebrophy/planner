package emailapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/emailbus"
	"github.com/casebrophy/planner/business/domain/emailbus/stores/emaildb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	emailStore := emaildb.NewStore(cfg.Log, cfg.DB)
	emailBus := emailbus.NewBusiness(cfg.Log, emailStore)

	hdl := &app{emailBus: emailBus}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/emails", hdl.queryAll, authen)
	a.Handle(http.MethodGet, "/api/v1/emails/{email_id}", hdl.queryByID, authen)
}
