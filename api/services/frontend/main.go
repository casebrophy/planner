package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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
		Auth struct {
			APIKey   string `conf:"mask"`
			Password string `conf:"mask"`
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
	if cfg.Auth.APIKey != "" {
		orig := proxy.Director
		proxy.Director = func(r *http.Request) {
			orig(r)
			r.Header.Set("X-API-Key", cfg.Auth.APIKey)
		}
	}
	mux := http.NewServeMux()
	mux.Handle("/api/", proxy)
	mux.Handle("/", spaHandler(cfg.Frontend.Dir))

	var handler http.Handler = mux
	if cfg.Auth.Password != "" {
		log.Info(context.Background(), "auth", "status", "cookie auth enabled")
		handler = authMiddleware(mux, cfg.Auth.Password)
	}

	srv := http.Server{
		Addr:         cfg.Web.Host,
		Handler:      handler,
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

// ----------------------------------------------------------------------------
// Cookie auth middleware
// ----------------------------------------------------------------------------

const (
	cookieName  = "planner_session"
	cookieMaxAge = 30 * 24 * 60 * 60 // 30 days in seconds
)

// deriveKey creates an HMAC key from the password so we don't store the
// password directly in the cookie. Deterministic — same password always
// yields the same key, so server restarts don't invalidate sessions.
func deriveKey(password string) []byte {
	h := sha256.Sum256([]byte("planner-session-key:" + password))
	return h[:]
}

// signToken returns hex(token:signature).
func signToken(token []byte, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(token)
	sig := mac.Sum(nil)
	return hex.EncodeToString(token) + ":" + hex.EncodeToString(sig)
}

// validCookie checks if the cookie value has a valid HMAC signature.
func validCookie(value string, key []byte) bool {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return false
	}
	token, err := hex.DecodeString(parts[0])
	if err != nil {
		return false
	}
	sig, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, key)
	mac.Write(token)
	return hmac.Equal(mac.Sum(nil), sig)
}

func authMiddleware(next http.Handler, password string) http.Handler {
	key := deriveKey(password)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow service worker and manifest through (they need no auth
		// and blocking them breaks PWA install/update).
		if r.URL.Path == "/sw.js" || r.URL.Path == "/manifest.json" {
			next.ServeHTTP(w, r)
			return
		}

		// Check existing session cookie.
		if c, err := r.Cookie(cookieName); err == nil && validCookie(c.Value, key) {
			next.ServeHTTP(w, r)
			return
		}

		// Handle login POST.
		if r.Method == http.MethodPost && r.URL.Path == "/login" {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			if r.FormValue("password") != password {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprint(w, loginPage("Incorrect password."))
				return
			}

			// Generate random token and sign it.
			token := make([]byte, 32)
			if _, err := rand.Read(token); err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			http.SetCookie(w, &http.Cookie{
				Name:     cookieName,
				Value:    signToken(token, key),
				Path:     "/",
				MaxAge:   cookieMaxAge,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
			})
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// Not authenticated — show login form.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, loginPage(""))
	})
}

func loginPage(errMsg string) string {
	errorHTML := ""
	if errMsg != "" {
		errorHTML = `<p style="color:#e74c3c;margin-bottom:1rem">` + errMsg + `</p>`
	}
	return `<!DOCTYPE html>
<html lang="en"><head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Planner — Login</title>
<style>
  *{box-sizing:border-box;margin:0;padding:0}
  body{font-family:system-ui,sans-serif;background:#0f172a;color:#e2e8f0;display:flex;align-items:center;justify-content:center;min-height:100vh}
  .card{background:#1e293b;border-radius:12px;padding:2.5rem;width:100%;max-width:360px;box-shadow:0 4px 24px rgba(0,0,0,.4)}
  h1{font-size:1.25rem;margin-bottom:1.5rem;text-align:center;font-weight:500}
  input[type=password]{width:100%;padding:.75rem 1rem;border:1px solid #334155;border-radius:8px;background:#0f172a;color:#e2e8f0;font-size:1rem;outline:none;transition:border-color .2s}
  input[type=password]:focus{border-color:#3b82f6}
  button{width:100%;padding:.75rem;margin-top:1rem;border:none;border-radius:8px;background:#3b82f6;color:#fff;font-size:1rem;cursor:pointer;font-weight:500;transition:background .2s}
  button:hover{background:#2563eb}
</style>
</head><body>
<div class="card">
  <h1>Planner</h1>
  ` + errorHTML + `
  <form method="POST" action="/login">
    <input type="password" name="password" placeholder="Password" autofocus required>
    <button type="submit">Sign in</button>
  </form>
</div>
</body></html>`
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
