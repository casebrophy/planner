# Server Monitoring Backend System

> Reverse proxy to a sidecar agent running on the VPS host. Exposes container status, logs, Claude instances, and systemd timer health through the planner API. App-layer only — no business or store layer. Sidecar binary lives in `zarf/sidecar/`.

## Architecture

```
Frontend → Backend (serverapp proxy) → Sidecar (zarf/sidecar/) → docker/systemctl/ps
           Docker container              VPS host (127.0.0.1:9090)
```

The backend discovers the sidecar via `PLANNER_SIDECAR_URL` env var (empty = feature disabled). The sidecar authenticates requests with the same `X-API-Key` as the main API.

## Core Types

```go
// app/domain/serverapp/serverapp.go
type app struct {
    sidecarURL string
    apiKey     string
}

// rawJSON passes through pre-encoded JSON from the sidecar.
type rawJSON struct {
    data   []byte
    status int
}
// Implements web.Encoder and web.HTTPStatuser
```

### Sidecar Response Types (zarf/sidecar/handlers.go)

```go
type ContainerInfo struct {
    Name   string `json:"name"`
    State  string `json:"state"`
    Status string `json:"status"`
    Health string `json:"health"`
    Image  string `json:"image"`
    Ports  string `json:"ports"`
}

type TimerInfo struct {
    Name    string `json:"name"`
    Active  bool   `json:"active"`
    LastRun string `json:"lastRun"`
    NextRun string `json:"nextRun"`
    Result  string `json:"result"`
}

type ClaudeInstance struct {
    PID     string `json:"pid"`
    Command string `json:"command"`
    CPU     string `json:"cpu"`
    Memory  string `json:"memory"`
    Elapsed string `json:"elapsed"`
}
```

## File Map

### Backend Proxy (app layer)
- `app/domain/serverapp/serverapp.go` — **proxyContainers()**, **proxyTimers()**, **proxyClaude()**, **proxyLogs()** — each forwards to sidecar, returns raw JSON
- `app/domain/serverapp/route.go` — Routes.Add() wiring, reads SidecarURL from mux.Config

### Sidecar Binary (standalone)
- `zarf/sidecar/main.go` — HTTP server entry point, auth middleware, flags (--addr, --api-key, --compose-file)
- `zarf/sidecar/handlers.go` — **containers()** docker compose ps, **timers()** systemctl show, **logs()** docker compose logs / journalctl, **claude()** ps aux filtered
- `zarf/sidecar/go.mod` — standalone module `github.com/casebrophy/planner/sidecar`
- `zarf/sidecar/planner-sidecar.service` — systemd unit for deployment

### Config
- `app/sdk/mux/mux.go` — `SidecarURL string` field on Config struct
- `api/services/planner/main.go` — `Sidecar.URL` config section (`PLANNER_SIDECAR_URL`)
- `zarf/compose/docker-compose.yml` — sets `PLANNER_SIDECAR_URL=http://host.docker.internal:9090` + `extra_hosts` for Linux

## Impact Callouts

### ⚠ Sidecar API shape (zarf/sidecar/handlers.go)
Changing sidecar endpoints or response shapes affects:
- `app/domain/serverapp/serverapp.go` — proxy path strings in forward() calls
- Frontend `useServerMonitor.ts` — TypeScript interfaces must match JSON response shapes
- No migration needed — sidecar has no database

### ⚠ mux.Config.SidecarURL (app/sdk/mux/mux.go)
Adding/changing this field affects:
- `api/services/planner/main.go` — must set the value from config
- `app/domain/serverapp/route.go` — reads it from cfg

## Routes

### Backend proxy routes
| Method | Path | Handler | Auth |
|--------|------|---------|------|
| GET | /api/v1/server/containers | proxyContainers | API key |
| GET | /api/v1/server/timers | proxyTimers | API key |
| GET | /api/v1/server/claude | proxyClaude | API key |
| GET | /api/v1/server/logs/{service} | proxyLogs | API key |

### Sidecar routes (127.0.0.1:9090)
| Method | Path | Handler | Auth |
|--------|------|---------|------|
| GET | /containers | containers | X-API-Key |
| GET | /timers | timers | X-API-Key |
| GET | /claude | claude | X-API-Key |
| GET | /logs/{service}?lines=N | logs | X-API-Key |

Allowed log services: `backend`, `frontend`, `db`, `planner-deploy`, `planner-backup`

## Cross-Domain Dependencies

- No dependencies on other planner domains
- Sidecar depends on host-level tools: `docker`, `systemctl`, `journalctl`, `ps`
- Frontend `useServerMonitor.ts` composable + `SettingsView.vue` consume these endpoints
