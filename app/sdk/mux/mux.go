package mux

import (
	"net/http"

	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/business/domain/embeddingbus"
	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
	"github.com/casebrophy/planner/foundation/claudecli"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/foundation/web"
)

type RouteAdder interface {
	Add(app *web.App, cfg Config)
}

type Config struct {
	Log           *logger.Logger
	DB            *sqlx.DB
	APIKey        string
	ClaudeCLI     *claudecli.Client
	CORSOrigins   []string
	SidecarURL    string
	OllamaURL        string
	OllamaModel      string
	OllamaEmbedModel string
	OllamaEnabled    bool
	Extractor     extractor.Extractor // nil when AI extraction is disabled
	EmbeddingBus  *embeddingbus.Business
}

func WebAPI(cfg Config, routeAdders ...RouteAdder) http.Handler {
	app := web.NewApp(
		cfg.Log,
		mid.Logger(cfg.Log),
		mid.Errors(cfg.Log),
		mid.Panics(cfg.Log),
	)

	if len(cfg.CORSOrigins) > 0 {
		app.EnableCORS(cfg.CORSOrigins)
	}

	for _, ra := range routeAdders {
		ra.Add(app, cfg)
	}

	return app
}
