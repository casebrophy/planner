package rawinputapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/clarificationbus/stores/clarificationdb"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/contextbus/stores/contextdb"
	"github.com/casebrophy/planner/business/domain/emailbus"
	"github.com/casebrophy/planner/business/domain/emailbus/stores/emaildb"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/eventbus/stores/eventdb"
	"github.com/casebrophy/planner/business/domain/ingestbus"
	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/rawinputbus/stores/rawinputdb"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/taskbus/stores/taskdb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	riStore := rawinputdb.NewStore(cfg.Log, cfg.DB)
	riBus := rawinputbus.NewBusiness(cfg.Log, riStore)

	emStore := emaildb.NewStore(cfg.Log, cfg.DB)
	emBus := emailbus.NewBusiness(cfg.Log, emStore)

	tStore := taskdb.NewStore(cfg.Log, cfg.DB)
	tDepStore := taskdb.NewDependencyStore(cfg.Log, cfg.DB)
	tBus := taskbus.NewBusiness(cfg.Log, tStore, tDepStore)

	cStore := contextdb.NewStore(cfg.Log, cfg.DB)
	cBus := contextbus.NewBusiness(cfg.Log, cStore)

	clStore := clarificationdb.NewStore(cfg.Log, cfg.DB)
	clBus := clarificationbus.NewBusiness(cfg.Log, clStore)

	eStore := eventdb.NewStore(cfg.Log, cfg.DB)
	eBus := eventbus.NewBusiness(cfg.Log, eStore)

	ext := extractor.NewClaudeCodeExtractor(cfg.ClaudeCLI)
	igBus := ingestbus.NewBusiness(cfg.Log, riBus, emBus, tBus, cBus, clBus, eBus, ext)

	hdl := &app{rawInputBus: riBus, ingestBus: igBus}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/raw-inputs", hdl.queryAll, authen)
	a.Handle(http.MethodGet, "/api/v1/raw-inputs/{raw_input_id}", hdl.queryByID, authen)
	a.Handle(http.MethodPost, "/api/v1/raw-inputs/{raw_input_id}/reprocess", hdl.reprocess, authen)
}
