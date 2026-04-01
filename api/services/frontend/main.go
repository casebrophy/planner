package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ardanlabs/conf"

	"github.com/casebrophy/planner/foundation/logger"
)

var build = "develop"

func main() {
	log := logger.New(os.Stdout, logger.LevelInfo, "frontend")

	if err := run(log); err != nil {
		log.Error(context.Background(), "startup", "error", err)
		os.Exit(1)
	}
}

func run(log *logger.Logger) error {
	cfg := struct {
		Web struct {
			Host            string        `conf:"default:0.0.0.0:3000"`
			ReadTimeout     time.Duration `conf:"default:5s"`
			WriteTimeout    time.Duration `conf:"default:10s"`
			IdleTimeout     time.Duration `conf:"default:120s"`
			ShutdownTimeout time.Duration `conf:"default:20s"`
		}
		Frontend struct {
			Dir     string `conf:"default:/service/web"`
			Backend string `conf:"default:http://backend:8080"`
		}
	}{}

	const prefix = "PLANNER"
	if err := conf.Parse(os.Args[1:], prefix, &cfg); err != nil {
		if err == conf.ErrHelpWanted {
			return nil
		}
		return fmt.Errorf("parsing config: %w", err)
	}

	log.Info(context.Background(), "starting frontend server", "version", build, "dir", cfg.Frontend.Dir)

	backendURL, err := url.Parse(cfg.Frontend.Backend)
	if err != nil {
		return fmt.Errorf("parsing backend URL: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(backendURL)
	mux := http.NewServeMux()
	mux.Handle("/api/", proxy)
	mux.Handle("/", spaHandler(cfg.Frontend.Dir))

	srv := http.Server{
		Addr:         cfg.Web.Host,
		Handler:      mux,
		ReadTimeout:  cfg.Web.ReadTimeout,
		WriteTimeout: cfg.Web.WriteTimeout,
		IdleTimeout:  cfg.Web.IdleTimeout,
		ErrorLog:     logger.NewStdLogger(log, logger.LevelError),
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Info(context.Background(), "startup", "status", "frontend server started", "host", srv.Addr)
		serverErrors <- srv.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case sig := <-shutdown:
		log.Info(context.Background(), "shutdown", "signal", sig)
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Web.ShutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			srv.Close()
			return fmt.Errorf("shutdown: %w", err)
		}
	}

	return nil
}

// spaHandler serves static files from dir. If the requested file does not
// exist, it falls back to index.html for SPA client-side routing.
// Service worker and manifest are served with no-cache to ensure updates propagate.
func spaHandler(dir string) http.Handler {
	fs := http.Dir(dir)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Clean(r.URL.Path)

		// Service worker and manifest must not be cached by intermediaries
		// so that updates propagate immediately.
		switch path {
		case "/sw.js", "/manifest.json":
			w.Header().Set("Cache-Control", "no-cache")
		}

		f, err := fs.Open(path)
		if err != nil {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		f.Close()

		http.FileServer(fs).ServeHTTP(w, r)
	})
}
