package transactionapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/transactionbus"
	"github.com/casebrophy/planner/business/domain/transactionbus/stores/transactiondb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	txnStore := transactiondb.NewStore(cfg.Log, cfg.DB)
	txnBus := transactionbus.NewBusiness(cfg.Log, txnStore)

	if cfg.Extractor != nil && cfg.OllamaEnabled {
		txnBus.WithEnricher(transactionbus.NewExtractorEnricher(cfg.Extractor))
	}

	hdl := &app{transactionBus: txnBus}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/transactions", hdl.queryAll, authen)
	a.Handle(http.MethodGet, "/api/v1/transactions/{transaction_id}", hdl.queryByID, authen)
	a.Handle(http.MethodPut, "/api/v1/transactions/{transaction_id}", hdl.update, authen)
	a.Handle(http.MethodDelete, "/api/v1/transactions/{transaction_id}", hdl.delete, authen)
	a.Handle(http.MethodPost, "/api/v1/transactions/import", hdl.importCSV, authen)
	a.Handle(http.MethodGet, "/api/v1/transactions/enrichment-status", hdl.enrichmentStatus, authen)
}
