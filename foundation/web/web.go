package web

import (
	"net/http"

	"github.com/casebrophy/planner/foundation/logger"
)

type App struct {
	mux     *http.ServeMux
	log     *logger.Logger
	mid     []MidFunc
	origins []string
}

func NewApp(log *logger.Logger, mid ...MidFunc) *App {
	return &App{
		mux: http.NewServeMux(),
		log: log,
		mid: mid,
	}
}

func (a *App) EnableCORS(origins []string) {
	a.origins = origins
}

func (a *App) Handle(method string, path string, handler HandlerFunc, mw ...MidFunc) {
	handler = wrapMiddleware(mw, handler)
	handler = wrapMiddleware(a.mid, handler)

	pattern := method + " " + path

	a.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		resp := handler(ctx, r)
		if err := Respond(ctx, w, resp); err != nil {
			a.log.Error(ctx, "respond error", "error", err)
		}
	})
}

func (a *App) HandleNoMiddleware(method string, path string, handler HandlerFunc) {
	pattern := method + " " + path

	a.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		resp := handler(ctx, r)
		if err := Respond(ctx, w, resp); err != nil {
			a.log.Error(ctx, "respond error", "error", err)
		}
	})
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions && len(a.origins) > 0 {
		origin := r.Header.Get("Origin")
		if a.isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, Authorization")
			w.Header().Set("Access-Control-Max-Age", "3600")
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	if len(a.origins) > 0 {
		origin := r.Header.Get("Origin")
		if a.isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
	}

	a.mux.ServeHTTP(w, r)
}

func (a *App) isAllowedOrigin(origin string) bool {
	for _, o := range a.origins {
		if o == "*" || o == origin {
			return true
		}
	}
	return false
}
