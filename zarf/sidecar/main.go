package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9090", "listen address")
	apiKey := flag.String("api-key", "", "required API key (or PLANNER_AUTH_API_KEY env)")
	composeFile := flag.String("compose-file", "/opt/planner/zarf/compose/docker-compose.yml", "docker-compose.yml path")
	flag.Parse()

	if *apiKey == "" {
		*apiKey = os.Getenv("PLANNER_AUTH_API_KEY")
	}
	if *apiKey == "" {
		log.Fatal("--api-key or PLANNER_AUTH_API_KEY required")
	}

	h := &handlers{composeFile: *composeFile}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /containers", h.containers)
	mux.HandleFunc("GET /logs/{service}", h.logs)
	mux.HandleFunc("GET /claude", h.claude)
	mux.HandleFunc("GET /timers", h.timers)

	handler := authMiddleware(*apiKey, mux)

	fmt.Printf("sidecar listening on %s\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, handler))
}

func authMiddleware(apiKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != apiKey {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
