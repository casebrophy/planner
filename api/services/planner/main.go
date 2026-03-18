package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ardanlabs/conf"

	"github.com/casebrophy/planner/app/domain/checkapp"
	"github.com/casebrophy/planner/app/domain/contextapp"
	"github.com/casebrophy/planner/app/domain/mcpapp"
	"github.com/casebrophy/planner/app/domain/tagapp"
	"github.com/casebrophy/planner/app/domain/taskapp"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/foundation/sqldb"
)

var build = "develop"

func main() {
	log := logger.New(os.Stdout, logger.LevelInfo, "planner")

	if err := run(log); err != nil {
		log.Error(context.Background(), "startup", "error", err)
		os.Exit(1)
	}
}

func run(log *logger.Logger) error {
	// -------------------------------------------------------------------------
	// Configuration

	cfg := struct {
		Web struct {
			APIHost         string        `conf:"default:0.0.0.0:8080"`
			ReadTimeout     time.Duration `conf:"default:5s"`
			WriteTimeout    time.Duration `conf:"default:10s"`
			IdleTimeout     time.Duration `conf:"default:120s"`
			ShutdownTimeout time.Duration `conf:"default:20s"`
			CORSOrigins     string        `conf:"default:*"`
		}
		DB   sqldb.Config
		Auth struct {
			APIKey string `conf:"mask"`
		}
	}{}

	const prefix = "PLANNER"
	err := conf.Parse(os.Args[1:], prefix, &cfg)
	if err != nil {
		if err == conf.ErrHelpWanted {
			return nil
		}
		return fmt.Errorf("parsing config: %w", err)
	}

	// -------------------------------------------------------------------------
	// App Starting

	log.Info(context.Background(), "starting service", "version", build)

	// -------------------------------------------------------------------------
	// Database

	log.Info(context.Background(), "startup", "status", "initializing database")

	db, err := sqldb.Open(cfg.DB)
	if err != nil {
		return fmt.Errorf("connecting to db: %w", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := sqldb.StatusCheck(ctx, db); err != nil {
		return fmt.Errorf("db status check: %w", err)
	}

	// -------------------------------------------------------------------------
	// Build Handler

	log.Info(ctx, "startup", "status", "initializing api")

	muxCfg := mux.Config{
		Log:    log,
		DB:     db,
		APIKey: cfg.Auth.APIKey,
	}

	handler := mux.WebAPI(muxCfg,
		checkapp.Routes{},
		taskapp.Routes{},
		contextapp.Routes{},
		tagapp.Routes{},
		mcpapp.Routes{},
	)

	// -------------------------------------------------------------------------
	// Start Server

	api := http.Server{
		Addr:         cfg.Web.APIHost,
		Handler:      handler,
		ReadTimeout:  cfg.Web.ReadTimeout,
		WriteTimeout: cfg.Web.WriteTimeout,
		IdleTimeout:  cfg.Web.IdleTimeout,
		ErrorLog:     logger.NewStdLogger(log, logger.LevelError),
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Info(ctx, "startup", "status", "api router started", "host", api.Addr)
		serverErrors <- api.ListenAndServe()
	}()

	// -------------------------------------------------------------------------
	// Shutdown

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		log.Info(ctx, "shutdown", "status", "shutdown started", "signal", sig)
		defer log.Info(ctx, "shutdown", "status", "shutdown complete", "signal", sig)

		ctx, cancel := context.WithTimeout(ctx, cfg.Web.ShutdownTimeout)
		defer cancel()

		if err := api.Shutdown(ctx); err != nil {
			api.Close()
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	return nil
}
