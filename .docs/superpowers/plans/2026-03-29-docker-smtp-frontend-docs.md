# Docker SMTP + Frontend Server + Doc Fixes

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expose SMTP ingestion in Docker, add a Go static file server for the frontend, and fix doc drift in roadmap/arch files.

**Architecture:** The backend container gains SMTP port exposure (25→2525). A new `api/services/frontend/main.go` binary serves pre-built Vue assets with SPA history-mode fallback — no reverse proxy, no nginx. Docs are updated to match reality (reprocess wiring done, SMTP is embedded in backend, frontend is a static server).

**Tech Stack:** Go (net/http, `fs.FS`), Docker multi-stage build (Node 22 + Go 1.24), docker-compose

---

### Task 1: Frontend static file server

**Files:**
- Create: `api/services/frontend/main.go`

This is a ~60-line Go binary. It serves files from a directory, with SPA fallback (any path not matching a real file serves `index.html`). Configuration via `PLANNER_FRONTEND_*` env vars using the same `ardanlabs/conf` library.

- [ ] **Step 1: Create the frontend server binary**

```go
// api/services/frontend/main.go
package main

import (
	"context"
	"fmt"
	"net/http"
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
			Dir string `conf:"default:/service/web"`
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

	srv := http.Server{
		Addr:         cfg.Web.Host,
		Handler:      spaHandler(cfg.Frontend.Dir),
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
func spaHandler(dir string) http.Handler {
	fs := http.Dir(dir)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to open the requested file.
		path := filepath.Clean(r.URL.Path)
		f, err := fs.Open(path)
		if err != nil {
			// File doesn't exist — serve index.html for SPA routing.
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		f.Close()

		// File exists — serve it normally.
		http.FileServer(fs).ServeHTTP(w, r)
	})
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./api/services/frontend/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add api/services/frontend/main.go
git commit -m "feat: add frontend static file server binary"
```

---

### Task 2: Update Dockerfile for frontend build

**Files:**
- Modify: `zarf/docker/Dockerfile.planner`

Add a Node build stage that builds the Vue frontend, then copy the dist output and the new frontend binary into the runtime image.

- [ ] **Step 1: Update Dockerfile**

Replace the entire file with:

```dockerfile
# Build stage — Go
FROM golang:1.24 AS build-go
ARG BUILD_REF=develop

WORKDIR /service

COPY go.* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags "-X main.build=${BUILD_REF}" -o planner ./api/services/planner/
RUN CGO_ENABLED=0 go build -ldflags "-X main.build=${BUILD_REF}" -o admin ./api/tooling/admin/
RUN CGO_ENABLED=0 go build -ldflags "-X main.build=${BUILD_REF}" -o frontend ./api/services/frontend/

# Build stage — Frontend
FROM node:22-alpine AS build-web

WORKDIR /web

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ .
RUN npm run build

# Runtime stage
FROM alpine:3.21
RUN addgroup -g 1000 planner && adduser -u 1000 -G planner -D planner

COPY --from=build-go --chown=planner:planner /service/planner /service/planner
COPY --from=build-go --chown=planner:planner /service/admin /service/admin
COPY --from=build-go --chown=planner:planner /service/frontend /service/frontend
COPY --from=build-web --chown=planner:planner /web/dist /service/web

WORKDIR /service
USER planner

EXPOSE 8080 2525 3000
CMD ["/service/planner"]
```

- [ ] **Step 2: Commit**

```bash
git add zarf/docker/Dockerfile.planner
git commit -m "build: add frontend build stage and frontend binary to Dockerfile"
```

---

### Task 3: Update docker-compose for SMTP + frontend

**Files:**
- Modify: `zarf/compose/docker-compose.yml`

Add SMTP port mapping (25→2525), SMTP/Claude API env vars to backend, and a new frontend service.

- [ ] **Step 1: Update docker-compose.yml**

Replace the entire file with:

```yaml
services:
  db:
    image: postgres:17
    environment:
      POSTGRES_USER: planner
      POSTGRES_PASSWORD: planner
      POSTGRES_DB: planner
    ports:
      - "127.0.0.1:5433:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U planner"]
      interval: 5s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  backend:
    build:
      context: ../..
      dockerfile: zarf/docker/Dockerfile.planner
    ports:
      - "127.0.0.1:8080:8080"
      - "0.0.0.0:25:2525"
    depends_on:
      db:
        condition: service_healthy
    environment:
      PLANNER_DB_HOST: db
      PLANNER_DB_PORT: 5432
      PLANNER_DB_USER: planner
      PLANNER_DB_PASSWORD: planner
      PLANNER_DB_NAME: planner
      PLANNER_DB_DISABLE_TLS: "true"
      PLANNER_AUTH_API_KEY: "${PLANNER_API_KEY:-devkey123}"
      PLANNER_SMTP_ENABLED: "${PLANNER_SMTP_ENABLED:-false}"
      PLANNER_SMTP_ADDR: ":2525"
      PLANNER_SMTP_DOMAIN: "${PLANNER_SMTP_DOMAIN:-localhost}"
      PLANNER_ANTHROPIC_API_KEY: "${PLANNER_ANTHROPIC_API_KEY:-}"
      PLANNER_ANTHROPIC_MODEL: "${PLANNER_ANTHROPIC_MODEL}"
    restart: unless-stopped

  frontend:
    build:
      context: ../..
      dockerfile: zarf/docker/Dockerfile.planner
    command: ["/service/frontend"]
    ports:
      - "127.0.0.1:3000:3000"
    restart: unless-stopped

volumes:
  pgdata:
```

- [ ] **Step 2: Commit**

```bash
git add zarf/compose/docker-compose.yml
git commit -m "build: expose SMTP port, add Claude API config, add frontend service"
```

---

### Task 4: Update Makefile with new targets

**Files:**
- Modify: `Makefile`

Add targets for SMTP logs, frontend logs, and building/running frontend locally.

- [ ] **Step 1: Add new targets to Makefile**

After the existing `logs:` target, add:

```makefile
logs-all:
	$(COMPOSE) logs -f backend frontend

frontend-dev:
	cd web && npm run build && cd .. && \
	PLANNER_FRONTEND_DIR=web/dist go run api/services/frontend/main.go
```

Update the `help:` target to include:

```makefile
	@echo "  make logs-all       - Tail backend + frontend logs"
	@echo "  make frontend-dev   - Build and serve frontend locally"
```

- [ ] **Step 2: Commit**

```bash
git add Makefile
git commit -m "build: add frontend-dev and logs-all Makefile targets"
```

---

### Task 5: Fix doc drift — 07-roadmap.md

**Files:**
- Modify: `.docs/07-roadmap.md`

Mark the reprocess wiring as done.

- [ ] **Step 1: Update Phase 3 deliverables**

In the Phase 3 section, change:

```
- Wire `ingestbus.Reprocess()` into `rawinputapp` reprocess endpoint — currently the endpoint only resets status to `processing` but does not re-run the pipeline
```

to:

```
- ~~Wire `ingestbus.Reprocess()` into `rawinputapp` reprocess endpoint~~ done
```

- [ ] **Step 2: Commit**

```bash
git add .docs/07-roadmap.md
git commit -m "docs: mark reprocess wiring as done in roadmap"
```

---

### Task 6: Fix doc drift — ingest-backend.md

**Files:**
- Modify: `.docs/arch/ingest-backend.md`

The Routes section says the reprocess endpoint does NOT call `ingestbus.Reprocess()`. It does now.

- [ ] **Step 1: Update Routes section**

Replace the existing Routes section with:

```markdown
## Routes

`ingestbus` has no HTTP routes. It is invoked exclusively via `smtpbus` (SMTP DATA command) or via `rawinputapp`'s reprocess endpoint.

The related REST endpoint that triggers reprocessing is in `rawinputapp`:

| Method | Path | Handler | Notes |
|--------|------|---------|-------|
| POST | `/api/v1/raw-inputs/{raw_input_id}/reprocess` | `rawinputapp.reprocess` | Verifies record exists, then calls `ingestbus.Reprocess()` which marks processing, re-runs the full pipeline, and marks processed/failed |
```

- [ ] **Step 2: Update Cross-Domain Dependencies**

In the Cross-Domain Dependencies table, change the `rawinputapp` row from:

```
| `rawinputapp` | Partial caller; marks raw_input as processing but does not invoke `ingestbus.Reprocess()` |
```

to:

```
| `rawinputapp` | Full caller; `reprocess` handler verifies record exists then calls `ingestbus.Reprocess()` for full pipeline re-run |
```

- [ ] **Step 3: Commit**

```bash
git add .docs/arch/ingest-backend.md
git commit -m "docs: fix ingest-backend arch — reprocess is fully wired"
```

---

### Task 7: Fix doc drift — rawinput-backend.md

**Files:**
- Modify: `.docs/arch/rawinput-backend.md`

The file map and impact callouts reference the old behavior.

- [ ] **Step 1: Update rawinputapp.go file map entry**

In the App Layer (HTTP Handlers) section, replace the `reprocess()` description:

```
  - **reprocess()** — POST /api/v1/raw-inputs/{raw_input_id}/reprocess, fetches record by UUID then calls `rawInputBus.MarkProcessing` to reset status to `processing`; returns updated record
```

with:

```
  - **reprocess()** — POST /api/v1/raw-inputs/{raw_input_id}/reprocess, verifies record exists by UUID then calls `ingestBus.Reprocess()` which runs the full ingestion pipeline; returns updated record
```

- [ ] **Step 2: Update the `app` struct description**

The rawinputapp.go file now holds `ingestBus *ingestbus.Business` in addition to `rawInputBus`. The arch file should reflect this if it describes the struct. Also update the route.go section to note that it wires the full ingestbus dependency chain.

In the route.go file map entry, replace:

```
- **`app/domain/rawinputapp/route.go`** — **Routes.Add()** — registers three endpoints with Auth middleware, instantiates `rawinputdb.Store` and `rawinputbus.Business`
```

with:

```
- **`app/domain/rawinputapp/route.go`** — **Routes.Add()** — registers three endpoints with Auth middleware; instantiates full ingestbus dependency chain (rawinputdb → rawinputbus → emaildb → emailbus → taskdb → taskbus → contextdb → contextbus → clarificationdb → clarificationbus → extractor → ingestbus) for the reprocess endpoint
```

- [ ] **Step 3: Update reprocess endpoint behavior callout**

Replace the entire "reprocess endpoint behavior" impact callout:

```
### reprocess endpoint behavior
`reprocess` does not reset a record to `pending`; it sets status to `processing`. Any pipeline consumer watching for `pending` records will not pick up a reprocessed record unless the ingest pipeline also polls `processing`. If the intent changes to reset to `pending`, update `app/domain/rawinputapp/rawinputapp.go` to call `MarkPending` (which would need to be added to `rawinputbus.go`).
```

with:

```
### reprocess endpoint behavior
`reprocess` calls `ingestbus.Reprocess()` which marks the record as `processing`, then re-runs the full 10-step ingestion pipeline. On success the record is marked `processed`; on failure it is marked `failed` with an error message. The handler verifies the record exists before invoking the pipeline.
```

- [ ] **Step 4: Update Routes table notes**

Replace:

```
| POST | /api/v1/raw-inputs/{raw_input_id}/reprocess | reprocess | Sets status=processing on the given record; 404 if not found |
```

with:

```
| POST | /api/v1/raw-inputs/{raw_input_id}/reprocess | reprocess | Runs the full ingestion pipeline via ingestbus.Reprocess(); 404 if not found |
```

- [ ] **Step 5: Commit**

```bash
git add .docs/arch/rawinput-backend.md
git commit -m "docs: fix rawinput-backend arch — reprocess calls full pipeline"
```

---

### Task 8: Fix doc drift — 06-infrastructure.md

**Files:**
- Modify: `.docs/06-infrastructure.md`

SMTP is embedded in the backend binary, not a separate container. Frontend is served by a Go static file server. Update the Docker services table and env vars.

- [ ] **Step 1: Update Docker services table**

Replace:

```markdown
| Service | Internal port | External bind | Volume |
|---------|--------------|---------------|--------|
| `backend` | 8080 | `127.0.0.1:8080` | `taskdata:/data`, `./data/imports:/data/imports` |
| `smtp` | 25, 587 | `0.0.0.0:25`, `0.0.0.0:587` | — |
| `frontend` | 80 | `127.0.0.1:3000` | — |
| `ml` | 8090 | `127.0.0.1:8090` | — |
```

with:

```markdown
| Service | Internal port | External bind | Volume | Notes |
|---------|--------------|---------------|--------|-------|
| `db` | 5432 | `127.0.0.1:5433` | `pgdata` | PostgreSQL 17 |
| `backend` | 8080, 2525 | `127.0.0.1:8080`, `0.0.0.0:25:2525` | — | REST API + embedded SMTP (when enabled) |
| `frontend` | 3000 | `127.0.0.1:3000` | — | Go static file server serving pre-built Vue app |
| `ml` | 8090 | `127.0.0.1:8090` | — | Future — Phase 8 |

SMTP is embedded in the backend binary via `smtpbus`, not a separate container. It listens on `:2525` internally and is mapped to host port 25 for external MTA delivery. Enabled via `PLANNER_SMTP_ENABLED=true`.
```

- [ ] **Step 2: Update Environment variables table**

Replace the existing table with:

```markdown
| Variable | Required | Default | Notes |
|----------|----------|---------|-------|
| `PLANNER_AUTH_API_KEY` | yes | — | `openssl rand -hex 32` |
| `PLANNER_DB_HOST` | no | `localhost` | `db` in Docker |
| `PLANNER_DB_PORT` | no | `5432` | — |
| `PLANNER_DB_USER` | no | `planner` | — |
| `PLANNER_DB_PASSWORD` | yes | — | — |
| `PLANNER_DB_NAME` | no | `planner` | — |
| `PLANNER_DB_DISABLE_TLS` | no | `true` | — |
| `PLANNER_SMTP_ENABLED` | no | `false` | Set `true` to start SMTP listener |
| `PLANNER_SMTP_ADDR` | no | `:2525` | Internal listen address |
| `PLANNER_SMTP_DOMAIN` | no | `localhost` | Domain for RCPT TO validation |
| `PLANNER_ANTHROPIC_API_KEY` | no | — | Required when SMTP is enabled |
| `PLANNER_ANTHROPIC_MODEL` | no | (environment-dependent) | — |
| `PLANNER_FRONTEND_DIR` | no | `/service/web` | Path to pre-built frontend assets |
| `PLANNER_WEB_CORS_ORIGINS` | no | `*` | CORS allowed origins |
```

- [ ] **Step 3: Commit**

```bash
git add .docs/06-infrastructure.md
git commit -m "docs: fix infrastructure — SMTP is embedded, add frontend server, update env vars"
```

---

### Task 9: Update CLAUDE.md planner app context

**Files:**
- Modify: `CLAUDE.md`

Update the "Built" list and phase status.

- [ ] **Step 1: Update the Built line**

In the "Planner App Context" section, add to the **Built** list after the existing items:

- `frontend static file server (api/services/frontend — Go SPA server)`
- Update the note about SMTP: change "disabled by default" to include "Docker-exposed on port 25→2525 when enabled"

- [ ] **Step 2: Update the docker-compose commands section**

Add to the Commands section:

```bash
# Frontend (local dev — build + serve)
make frontend-dev

# Tail all service logs
make logs-all
```

- [ ] **Step 3: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: update CLAUDE.md with frontend server and SMTP docker exposure"
```
