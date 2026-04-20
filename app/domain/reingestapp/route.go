package reingestapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/clarificationbus/stores/clarificationdb"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/eventbus/stores/eventdb"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/notebus/stores/notedb"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/rawinputbus/stores/rawinputdb"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/taskbus/stores/taskdb"
	"github.com/casebrophy/planner/foundation/web"
)

// Routes implements mux.Adder.
type Routes struct{}

// Add registers the reingest endpoints.
func (Routes) Add(a *web.App, cfg mux.Config) {
	taskStore := taskdb.NewStore(cfg.Log, cfg.DB)
	depStore := taskdb.NewDependencyStore(cfg.Log, cfg.DB)
	taskBus := taskbus.NewBusiness(cfg.Log, taskStore, depStore)

	noteStore := notedb.NewStore(cfg.Log, cfg.DB)
	noteBus := notebus.NewBusiness(cfg.Log, noteStore)

	evtStore := eventdb.NewStore(cfg.Log, cfg.DB)
	evtBus := eventbus.NewBusiness(cfg.Log, evtStore)

	riStore := rawinputdb.NewStore(cfg.Log, cfg.DB)
	riBus := rawinputbus.NewBusiness(cfg.Log, riStore)

	clarStore := clarificationdb.NewStore(cfg.Log, cfg.DB)
	clarBus := clarificationbus.NewBusiness(cfg.Log, clarStore)

	hdl := &app{
		log:      cfg.Log,
		taskBus:  taskBus,
		noteBus:  noteBus,
		eventBus: evtBus,
		riBus:    riBus,
		clarBus:  clarBus,
	}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodPost, "/api/v1/tasks/{task_id}/reingest", hdl.reingestTask, authen)
	a.Handle(http.MethodPost, "/api/v1/notes/{note_id}/reingest", hdl.reingestNote, authen)
	a.Handle(http.MethodPost, "/api/v1/events/{event_id}/reingest", hdl.reingestEvent, authen)
	a.Handle(http.MethodPost, "/api/v1/reingest/bulk", hdl.reingestBulk, authen)
}
