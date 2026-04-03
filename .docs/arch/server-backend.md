# Server Sidecar Backend System

> The sidecar is a standalone Go HTTP server that runs on the VPS host alongside the Dockerized backend. It has two jobs: (1) **inference** — it holds a persistent Claude Code CLI session and dispatches AI inference requests from backend pipelines; (2) **monitoring** — it exposes container status, logs, Claude process info, and systemd timer health. The backend communicates with it over HTTP via the `claudecli` package and `serverapp` proxy.

## Architecture

```
AI Pipeline (business layer)
  └─ claudecli.Client.RunJSON()
       ├─ [sidecar configured] → POST http://host.docker.internal:9090/inference
       │     Sidecar orchestrator (Opus, persistent session)
       │       └─ Spawns subagent (haiku/sonnet/opus per request)
       │             └─ Calls planner MCP tools (read-only)
       │     Response: {"result": "<content>", "model": "haiku"}
       └─ [no sidecar]       → exec claude -p (local CLI)

Frontend → Backend (serverapp proxy) → Sidecar monitoring endpoints
           Docker container              VPS host (127.0.0.1:9090)
```

The backend discovers the sidecar via `PLANNER_SIDECAR_URL` env var (empty = feature disabled, all inference falls back to local CLI). The sidecar authenticates with the same `X-API-Key` as the main API.

---

## Inference Pipeline

### claudecli Package (`foundation/claudecli/`)

The `claudecli.Client` is the single entry point for all AI inference in the backend. It is created once in `main.go` and injected into every business component that needs it.

```go
// foundation/claudecli/claudecli.go
type Client struct {
    cliPath    string
    models     []string       // escalation chain, e.g. ["haiku", "sonnet", "opus"]
    timeout    time.Duration
    sidecarURL string         // empty = use local CLI
    sidecarKey string
    httpClient *http.Client
}

// main.go wiring:
cli := claudecli.NewClient(log, cfg.Claude.CLIPath, strings.Split(cfg.Claude.Models, ","))
if cfg.Sidecar.URL != "" {
    cli.SetSidecar(cfg.Sidecar.URL, cfg.Auth.APIKey)
}
```

**Model escalation**: `RunJSON()` tries models in order (`haiku → sonnet → opus`). After each successful parse it calls `shouldEscalate()` (caller-supplied). If the result is low quality (e.g. `confidence < 0.4`), it retries with the next model. Last model is always accepted.

```go
func (c *Client) RunJSON(ctx context.Context, prompt, schema string, dest any, shouldEscalate func() bool) error
```

### Routing: sidecar vs. local CLI

`run()` dispatches based on whether `sidecarURL` is set:

```go
func (c *Client) run(ctx, prompt, schema, model) ([]byte, error) {
    if c.sidecarURL != "" {
        return c.runHTTP(ctx, prompt, schema, model)  // POST /inference
    }
    return c.runCLI(ctx, prompt, schema, model)        // exec claude -p
}
```

### ⚠ Double-Envelope Unwrapping

This is a critical gotcha. The sidecar uses `--output-format json`, which makes the Claude CLI wrap its output in a JSON event stream `[{type:"system",...}, {type:"result", result:"<actual>"}]`. The sidecar's `/inference` handler extracts `result` from this and returns `{"result": "<content>", "model": "haiku"}`. But when the orchestrator spawns a subagent, the subagent's output is *itself* a CLI envelope — so `result` may contain a nested `{"type":"result","result":"<actual content>"}`.

`runHTTP()` handles this with two-level unwrapping:

```go
// Level 1: sidecar envelope {"result": "...", "model": "..."}
var envelope struct{ Result string `json:"result"` }
json.Unmarshal(respBody, &envelope)

// Level 2: nested CLI envelope {"type":"result","result":"<actual>"}
var nested struct{ Result, Type string }
json.Unmarshal([]byte(envelope.Result), &nested)
if nested.Type == "result" && nested.Result != "" {
    return []byte(nested.Result), nil  // actual content
}
```

Missing level 2 causes `json.Unmarshal` to silently produce zero-value structs, which triggers the `shouldEscalate` check, and the system burns through all models for nothing.

---

## Sidecar Internals (`zarf/sidecar/`)

The sidecar is a **standalone Go module** (`github.com/casebrophy/planner/sidecar`) — it does not import any planner packages.

### Inference Endpoint: `POST /inference`

```go
// zarf/sidecar/handlers.go
type InferenceRequest struct {
    Prompt string `json:"prompt"`
    Schema string `json:"schema,omitempty"`
    Model  string `json:"model"`  // "haiku" | "sonnet" | "opus"
}

type InferenceResponse struct {
    Result string `json:"result"`
    Model  string `json:"model"`
}
```

The handler:
1. Locks the session mutex (requests are serialized)
2. Wraps the request in a dispatch message (JSON with `model`, `prompt`, `schema`)
3. Builds MCP config pointing to `http://localhost:8080/mcp` with the API key
4. Calls `claude` CLI: `-p <dispatch_msg> --output-format json --model opus --mcp-config <json>`
5. On first request: includes `--system-prompt <orchestratorSystemPrompt>`; on subsequent: `--resume <sessionID>`
6. Parses the JSON event stream to extract `session_id`, `result`, `input_tokens`, `output_tokens`
7. Rotates session if `input_tokens >= contextMax` (default 150k)

### Orchestrator Role

The sidecar holds a **persistent Opus session** that acts as a pure dispatcher:

```
System prompt: "You are the planner system's inference orchestrator. Your ONLY job is to
dispatch work to subagents and return their raw results. Never reason about the task yourself.
You are a dispatcher."
```

When a request arrives, the orchestrator reads `model`/`prompt`/`schema` from the dispatch message, spawns a subagent at the requested model via the Agent tool, and returns the subagent's raw output text.

The subagent has read-only access to planner MCP tools: `list_tasks`, `list_events`, `get_context`, `get_inference_context`, etc.

### Session Management (`zarf/sidecar/session.go`)

```go
type SessionManager struct {
    sessionID  string           // Claude Code session ID (--resume)
    createdAt  time.Time
    contextMax int              // rotate threshold (default 150k input tokens)
    timeout    time.Duration    // per-request timeout (default 180s)
    requests   []RequestMetric  // in-memory, cleared on rotate
    sessions   []SessionSummary // history of completed sessions
}
```

**Rotation triggers**: context full (`input_tokens >= contextMax`), CLI error, timeout, or manual `POST /inference/rotate`. Rotation clears `sessionID` — next request starts a fresh session with the system prompt.

### Log Store (`zarf/sidecar/logstore.go`)

All inference requests are persisted to JSONL at `SIDECAR_LOG_PATH` (default `/var/log/planner/sidecar-requests.jsonl`). Each entry includes `id`, `timestamp`, `duration_ms`, `agent_model`, `prompt_prefix` (first 100 chars), `input_tokens`, `output_tokens`, `session_id`, `success`, `error`, `prompt`, `result`.

Retention is 30 days by default (`SIDECAR_LOG_RETENTION_DAYS`).

---

## Monitoring Endpoints

The sidecar also exposes host-level observability for the frontend settings panel:

| Method | Sidecar Path | What it does |
|--------|-------------|--------------|
| GET | `/containers` | `docker compose ps --format json` |
| GET | `/timers` | `systemctl show planner-*.timer` |
| GET | `/claude` | `ps aux` filtered for `claude` processes |
| GET | `/logs/{service}?lines=N` | `docker compose logs` or `journalctl` |
| GET | `/logs/sidecar` | Query JSONL log store |
| GET | `/logs/sidecar/stats` | Aggregate stats from log store |
| GET | `/inference/status` | Session health: age, context%, token growth trend |
| GET | `/inference/history` | Past session summaries |
| GET | `/inference/tools` | Tool call frequency in current session |
| POST | `/inference/rotate` | Manually rotate the Claude session |

Allowed log services: `backend`, `frontend`, `db`, `planner-deploy`, `planner-backup`

---

## File Map

### claudecli package (used by ALL AI inference in backend)
- `foundation/claudecli/claudecli.go` — `Client`, `NewClient()`, `SetSidecar()`, `RunJSON()`, `runHTTP()`, `runCLI()`, double-envelope logic

### Backend proxy (app layer only)
- `app/domain/serverapp/serverapp.go` — `forward()` proxy, `proxyContainers()`, `proxyTimers()`, `proxyClaude()`, `proxyLogs()`, `proxyInferenceStatus()`, `proxyInferenceHistory()`, `proxyInferenceTools()`
- `app/domain/serverapp/route.go` — routes `/api/v1/server/**` → sidecar via proxy

### Sidecar binary (standalone, VPS host)
- `zarf/sidecar/main.go` — HTTP server, auth middleware, env config
- `zarf/sidecar/handlers.go` — `inference()`, `containers()`, `timers()`, `logs()`, `claude()`, monitoring handlers
- `zarf/sidecar/session.go` — `SessionManager`, `rotate()`, `Status()`, `History()`, `Tools()`
- `zarf/sidecar/logstore.go` — JSONL persistence, `Append()`, `Query()`, `Stats()`
- `zarf/sidecar/logger.go` — structured logger
- `zarf/sidecar/go.mod` — standalone module
- `zarf/sidecar/planner-sidecar.service` — systemd unit (reads `PLANNER_AUTH_API_KEY` from `.env`)

### Config
- `app/sdk/mux/mux.go` — `SidecarURL string` on `Config`
- `api/services/planner/main.go` — `cfg.Sidecar.URL` → `cli.SetSidecar()` + `muxCfg.SidecarURL`
- `zarf/compose/docker-compose.yml` — sets `PLANNER_SIDECAR_URL=http://host.docker.internal:9090` + `extra_hosts` for Linux

---

## Impact Callouts

### ⚠ Auth alignment (critical)
The sidecar systemd unit reads `PLANNER_AUTH_API_KEY` from `/opt/planner/.env` directly. The backend container gets `PLANNER_API_KEY` from SOPS-decrypted secrets, mapped to `PLANNER_AUTH_API_KEY` in `docker-compose.yml`. If these diverge, backend→sidecar requests get 401. Always ensure both use the same value.

### ⚠ Double-envelope (critical)
Any change to the sidecar's `--output-format json` usage or how the orchestrator returns results must account for the two-level unwrap in `runHTTP()`. Breaking this silently produces zero-value structs.

### ⚠ Inference serialization
The sidecar's `inference()` handler holds a global mutex for the entire Claude CLI call. All inference requests from all backend pipelines are serialized. This is intentional (single persistent session) but means long-running classifications block other pipelines. Timeouts are 180s per request.

### ⚠ Sidecar is a separate build
`zarf/sidecar/` has its own `go.mod` — it is NOT built by `go build ./...` from the root. To build it: `cd zarf/sidecar && go build -o sidecar .`. The `make deploy` script handles this.

### ⚠ MCP URL for subagents
The sidecar's MCP config points to `PLANNER_MCP_URL` (default `http://localhost:8080/mcp`). This is `localhost` from the sidecar's perspective (VPS host), not from Docker. If the backend port changes, update this env var.

---

## Routes

### Backend proxy routes
| Method | Path | Handler | Auth |
|--------|------|---------|------|
| GET | /api/v1/server/containers | proxyContainers | API key |
| GET | /api/v1/server/timers | proxyTimers | API key |
| GET | /api/v1/server/claude | proxyClaude | API key |
| GET | /api/v1/server/logs/{service} | proxyLogs | API key |
| GET | /api/v1/server/inference/status | proxyInferenceStatus | API key |
| GET | /api/v1/server/inference/history | proxyInferenceHistory | API key |
| GET | /api/v1/server/inference/tools | proxyInferenceTools | API key |

### Sidecar direct routes (127.0.0.1:9090)
| Method | Path | Auth |
|--------|------|------|
| POST | /inference | X-API-Key |
| GET | /inference/status | X-API-Key |
| GET | /inference/history | X-API-Key |
| GET | /inference/tools | X-API-Key |
| POST | /inference/rotate | X-API-Key |
| GET | /containers | X-API-Key |
| GET | /timers | X-API-Key |
| GET | /claude | X-API-Key |
| GET | /logs/{service}?lines=N | X-API-Key |
| GET | /logs/sidecar | X-API-Key |
| GET | /logs/sidecar/stats | X-API-Key |

---

## Env Vars

| Var | Default | Used by |
|-----|---------|---------|
| `PLANNER_SIDECAR_URL` | `""` (disabled) | Backend: routes inference to sidecar |
| `PLANNER_AUTH_API_KEY` | — | Sidecar: auth key (from `.env` on VPS host) |
| `PLANNER_MCP_URL` | `http://localhost:8080/mcp` | Sidecar: MCP endpoint for subagents |
| `SIDECAR_CONTEXT_MAX` | `150000` | Sidecar: rotate session at this input token count |
| `SIDECAR_REQUEST_TIMEOUT` | `180s` | Sidecar: per-request Claude CLI timeout |
| `SIDECAR_LOG_PATH` | `/var/log/planner/sidecar-requests.jsonl` | Sidecar: inference log file |
| `SIDECAR_LOG_RETENTION_DAYS` | `30` | Sidecar: JSONL log retention |
| `SIDECAR_LOG_VERBOSE` | `false` | Sidecar: include full prompt+result in logs |

---

## Cross-Domain Dependencies

- `claudecli.Client` is used by: `threadbus` (ClaudeCodeExtractor), `ingestbus`, `emailbus`, `dailyplanbus`, `classifyapp`, and any domain that does AI inference
- Sidecar depends on host tools: `docker`, `systemctl`, `journalctl`, `ps`, `claude` (CLI)
- Frontend `useServerMonitor.ts` + `SettingsView.vue` consume the monitoring endpoints
- No planner database dependencies
