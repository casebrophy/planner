# Server Monitoring Dashboard — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a "Server" section to the Settings page showing container statuses, application logs, Claude instance status, and deploy/backup timer health — powered by a lightweight Go sidecar on the VPS host.

**Architecture:** A Go HTTP server (`zarf/sidecar/`) runs on the VPS host (not in Docker) and exposes JSON endpoints by shelling out to `docker`, `journalctl`, and `ps`. The planner backend proxies `/api/v1/server/*` requests to the sidecar via `PLANNER_SIDECAR_URL`. The frontend adds a tabbed "Server" section to SettingsView that polls these endpoints.

**Tech Stack:** Go 1.24 (sidecar binary + backend proxy), Vue 3 + Tailwind (frontend), systemd (sidecar service)

---

## File Map

### Sidecar (new standalone Go binary — `zarf/sidecar/`)

| File | Responsibility |
|------|---------------|
| `zarf/sidecar/main.go` | Entry point: flags, HTTP server on `127.0.0.1:9090`, auth middleware, JSON helpers |
| `zarf/sidecar/handlers.go` | 4 handler functions: `containers`, `logs`, `claude`, `timers` |
| `zarf/sidecar/go.mod` | Standalone module (no shared deps with main app) |
| `zarf/sidecar/planner-sidecar.service` | systemd unit file for deployment |

### Backend proxy (new domain — `app/domain/serverapp/`)

| File | Responsibility |
|------|---------------|
| `app/domain/serverapp/serverapp.go` | Proxy handler: forwards requests to sidecar, returns raw JSON |
| `app/domain/serverapp/route.go` | `Routes.Add()` — registers 4 GET endpoints under `/api/v1/server/` |
| `app/sdk/mux/mux.go` | Add `SidecarURL string` to `Config` struct |
| `api/services/planner/main.go` | Add `Sidecar` config section, pass URL to mux config, add `serverapp.Routes{}` |

### Frontend (modify existing settings page)

| File | Responsibility |
|------|---------------|
| `api/services/frontend/web/src/composables/useServerMonitor.ts` | Composable: fetch containers/logs/claude/timers, auto-poll |
| `api/services/frontend/web/src/views/SettingsView.vue` | Add tabbed "Server" section below existing settings |

---

## Task 1: Sidecar Binary

**Files:**
- Create: `zarf/sidecar/go.mod`
- Create: `zarf/sidecar/main.go`
- Create: `zarf/sidecar/handlers.go`

- [ ] **Step 1: Initialize Go module**

```bash
mkdir -p zarf/sidecar
cd zarf/sidecar
go mod init github.com/casebrophy/planner/sidecar
```

- [ ] **Step 2: Write `main.go`**

```go
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
```

- [ ] **Step 3: Write `handlers.go`**

```go
package main

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
)

type handlers struct {
	composeFile string
}

// =========================================================================
// GET /containers

type ContainerInfo struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Status string `json:"status"`
	Health string `json:"health"`
	Image  string `json:"image"`
	Ports  string `json:"ports"`
}

func (h *handlers) containers(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command(
		"docker", "compose", "-f", h.composeFile,
		"ps", "--format", "json", "-a",
	).Output()
	if err != nil {
		writeError(w, 500, "docker compose ps failed: "+err.Error())
		return
	}

	var containers []ContainerInfo
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		var raw struct {
			Name   string `json:"Name"`
			State  string `json:"State"`
			Status string `json:"Status"`
			Health string `json:"Health"`
			Image  string `json:"Image"`
			Ports  string `json:"Ports"`
		}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		containers = append(containers, ContainerInfo{
			Name:   raw.Name,
			State:  raw.State,
			Status: raw.Status,
			Health: raw.Health,
			Image:  raw.Image,
			Ports:  raw.Ports,
		})
	}
	if containers == nil {
		containers = []ContainerInfo{}
	}
	writeJSON(w, containers)
}

// =========================================================================
// GET /timers

type TimerInfo struct {
	Name    string `json:"name"`
	Active  bool   `json:"active"`
	LastRun string `json:"lastRun"`
	NextRun string `json:"nextRun"`
	Result  string `json:"result"`
}

func (h *handlers) timers(w http.ResponseWriter, r *http.Request) {
	units := []string{"planner-deploy", "planner-backup"}
	var timers []TimerInfo

	for _, unit := range units {
		t := TimerInfo{Name: unit}

		activeOut, _ := exec.Command("systemctl", "is-active", unit+".timer").Output()
		t.Active = strings.TrimSpace(string(activeOut)) == "active"

		lastOut, _ := exec.Command("systemctl", "show", unit+".timer", "--property=LastTriggerUSec", "--value").Output()
		t.LastRun = strings.TrimSpace(string(lastOut))

		nextOut, _ := exec.Command("systemctl", "show", unit+".timer", "--property=NextElapseUSecRealtime", "--value").Output()
		t.NextRun = strings.TrimSpace(string(nextOut))

		resultOut, _ := exec.Command("systemctl", "show", unit+".service", "--property=Result", "--value").Output()
		t.Result = strings.TrimSpace(string(resultOut))

		timers = append(timers, t)
	}

	writeJSON(w, timers)
}

// =========================================================================
// GET /logs/{service}?lines=100

func (h *handlers) logs(w http.ResponseWriter, r *http.Request) {
	service := r.PathValue("service")

	allowed := map[string]bool{
		"backend": true, "frontend": true, "db": true,
		"planner-deploy": true, "planner-backup": true,
	}
	if !allowed[service] {
		writeError(w, 400, "unknown service: "+service)
		return
	}

	lines := 100
	if n := r.URL.Query().Get("lines"); n != "" {
		if parsed, err := strconv.Atoi(n); err == nil && parsed > 0 && parsed <= 500 {
			lines = parsed
		}
	}

	var out []byte
	var err error

	switch service {
	case "planner-deploy", "planner-backup":
		out, err = exec.Command(
			"journalctl", "-u", service+".service",
			"--no-pager", "-n", strconv.Itoa(lines), "--output=short-iso",
		).Output()
	default:
		out, err = exec.Command(
			"docker", "compose", "-f", h.composeFile,
			"logs", "--tail", strconv.Itoa(lines), "--no-color", service,
		).Output()
	}
	if err != nil {
		writeError(w, 500, "failed to get logs: "+err.Error())
		return
	}

	writeJSON(w, map[string]string{"logs": string(out)})
}

// =========================================================================
// GET /claude

type ClaudeInstance struct {
	PID     string `json:"pid"`
	Command string `json:"command"`
	CPU     string `json:"cpu"`
	Memory  string `json:"memory"`
	Elapsed string `json:"elapsed"`
}

func (h *handlers) claude(w http.ResponseWriter, r *http.Request) {
	out, _ := exec.Command("ps", "aux").Output()

	var instances []ClaudeInstance
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "claude") || strings.Contains(line, "grep") || strings.Contains(line, "sidecar") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}
		instances = append(instances, ClaudeInstance{
			PID:     fields[1],
			CPU:     fields[2],
			Memory:  fields[3],
			Elapsed: fields[9],
			Command: strings.Join(fields[10:], " "),
		})
	}
	if instances == nil {
		instances = []ClaudeInstance{}
	}
	writeJSON(w, instances)
}
```

- [ ] **Step 4: Build and verify**

```bash
cd zarf/sidecar && go build -o sidecar .
```

Expected: binary built, no errors.

- [ ] **Step 5: Create systemd unit**

Create `zarf/sidecar/planner-sidecar.service`:

```ini
[Unit]
Description=Planner sidecar monitoring agent
After=docker.service

[Service]
Type=simple
User=deployer
ExecStart=/opt/planner/zarf/sidecar/sidecar --compose-file=/opt/planner/zarf/compose/docker-compose.yml
Restart=on-failure
RestartSec=5
EnvironmentFile=/opt/planner/.env

[Install]
WantedBy=multi-user.target
```

- [ ] **Step 6: Commit**

```bash
git add zarf/sidecar/
git commit -m "feat: add sidecar monitoring agent binary"
```

---

## Task 2: Backend Proxy

**Files:**
- Modify: `app/sdk/mux/mux.go` (add `SidecarURL` to Config)
- Create: `app/domain/serverapp/serverapp.go`
- Create: `app/domain/serverapp/route.go`
- Modify: `api/services/planner/main.go` (config + wiring)

- [ ] **Step 1: Add SidecarURL to mux.Config**

In `app/sdk/mux/mux.go`, add `SidecarURL` to the `Config` struct:

```go
type Config struct {
	Log         *logger.Logger
	DB          *sqlx.DB
	APIKey      string
	ClaudeCLI   *claudecli.Client
	CORSOrigins []string
	SidecarURL  string
}
```

- [ ] **Step 2: Write `serverapp.go`**

The proxy handler forwards requests to the sidecar and returns the raw JSON response. It implements `web.Encoder` via a `rawJSON` type.

```go
// app/domain/serverapp/serverapp.go
package serverapp

import (
	"context"
	"io"
	"net/http"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	sidecarURL string
	apiKey     string
}

// rawJSON passes through pre-encoded JSON from the sidecar.
type rawJSON struct {
	data   []byte
	status int
}

func (r rawJSON) Encode() ([]byte, string, error) {
	return r.data, "application/json", nil
}

func (r rawJSON) HTTPStatus() int {
	return r.status
}

func (a *app) proxyContainers(ctx context.Context, r *http.Request) web.Encoder {
	return a.forward(ctx, "/containers", "")
}

func (a *app) proxyTimers(ctx context.Context, r *http.Request) web.Encoder {
	return a.forward(ctx, "/timers", "")
}

func (a *app) proxyClaude(ctx context.Context, r *http.Request) web.Encoder {
	return a.forward(ctx, "/claude", "")
}

func (a *app) proxyLogs(ctx context.Context, r *http.Request) web.Encoder {
	service := r.PathValue("service")
	qs := r.URL.RawQuery
	return a.forward(ctx, "/logs/"+service, qs)
}

func (a *app) forward(ctx context.Context, path string, qs string) web.Encoder {
	if a.sidecarURL == "" {
		return errs.Newf(errs.Internal, "sidecar not configured")
	}

	url := a.sidecarURL + path
	if qs != "" {
		url += "?" + qs
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return errs.New(errs.Internal, err)
	}
	req.Header.Set("X-API-Key", a.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errs.Newf(errs.Internal, "sidecar unreachable: %s", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errs.New(errs.Internal, err)
	}

	return rawJSON{data: body, status: resp.StatusCode}
}
```

- [ ] **Step 3: Write `route.go`**

Follow the existing `Routes.Add(a *web.App, cfg mux.Config)` pattern from `taskapp/route.go`:

```go
// app/domain/serverapp/route.go
package serverapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	hdl := &app{
		sidecarURL: cfg.SidecarURL,
		apiKey:     cfg.APIKey,
	}

	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/server/containers", hdl.proxyContainers, authen)
	a.Handle(http.MethodGet, "/api/v1/server/timers", hdl.proxyTimers, authen)
	a.Handle(http.MethodGet, "/api/v1/server/claude", hdl.proxyClaude, authen)
	a.Handle(http.MethodGet, "/api/v1/server/logs/{service}", hdl.proxyLogs, authen)
}
```

- [ ] **Step 4: Wire into main.go**

In `api/services/planner/main.go`:

a) Add import:
```go
"github.com/casebrophy/planner/app/domain/serverapp"
```

b) Add config section after `DailyPlan`:
```go
Sidecar struct {
    URL string `conf:"default:"`
}
```

c) Set `SidecarURL` on `muxCfg`:
```go
muxCfg := mux.Config{
    Log:         log,
    DB:          db,
    APIKey:      cfg.Auth.APIKey,
    ClaudeCLI:   cli,
    CORSOrigins: strings.Split(cfg.Web.CORSOrigins, ","),
    SidecarURL:  cfg.Sidecar.URL,
}
```

d) Add `serverapp.Routes{}` to the route list:
```go
handler := mux.WebAPI(muxCfg,
    // ... existing routes ...
    serverapp.Routes{},
)
```

- [ ] **Step 5: Verify build**

```bash
go build ./...
```

Expected: clean build, no errors.

- [ ] **Step 6: Commit**

```bash
git add app/domain/serverapp/ app/sdk/mux/mux.go api/services/planner/main.go
git commit -m "feat: add server monitoring proxy endpoints"
```

---

## Task 3: Frontend — Server Monitor Composable

**Files:**
- Create: `api/services/frontend/web/src/composables/useServerMonitor.ts`

- [ ] **Step 1: Write the composable**

Uses the existing `client.ts` `get()` function for API calls:

```ts
// api/services/frontend/web/src/composables/useServerMonitor.ts
import { ref, onMounted, onUnmounted } from 'vue'
import { get } from '@/services/client'

export interface ContainerInfo {
  name: string
  state: string
  status: string
  health: string
  image: string
  ports: string
}

export interface TimerInfo {
  name: string
  active: boolean
  lastRun: string
  nextRun: string
  result: string
}

export interface ClaudeInstance {
  pid: string
  command: string
  cpu: string
  memory: string
  elapsed: string
}

export function useServerMonitor() {
  const containers = ref<ContainerInfo[]>([])
  const timers = ref<TimerInfo[]>([])
  const claudeInstances = ref<ClaudeInstance[]>([])
  const logs = ref<string>('')
  const logService = ref<string>('backend')
  const loading = ref(false)
  const error = ref<string>('')
  const available = ref(true)

  let pollInterval: ReturnType<typeof setInterval> | null = null

  async function fetchContainers() {
    try {
      containers.value = await get<ContainerInfo[]>('/api/v1/server/containers')
    } catch (e: any) {
      if (e?.status === 500 && e?.message?.includes('sidecar')) {
        available.value = false
      }
      error.value = e?.message || 'Failed to fetch containers'
    }
  }

  async function fetchTimers() {
    try {
      timers.value = await get<TimerInfo[]>('/api/v1/server/timers')
    } catch (e: any) {
      error.value = e?.message || 'Failed to fetch timers'
    }
  }

  async function fetchClaude() {
    try {
      claudeInstances.value = await get<ClaudeInstance[]>('/api/v1/server/claude')
    } catch (e: any) {
      error.value = e?.message || 'Failed to fetch Claude instances'
    }
  }

  async function fetchLogs(service?: string) {
    if (service) logService.value = service
    try {
      const resp = await get<{ logs: string }>(`/api/v1/server/logs/${logService.value}?lines=100`)
      logs.value = resp.logs
    } catch (e: any) {
      error.value = e?.message || 'Failed to fetch logs'
    }
  }

  async function refresh() {
    loading.value = true
    error.value = ''
    await Promise.all([fetchContainers(), fetchTimers(), fetchClaude()])
    loading.value = false
  }

  onMounted(() => {
    refresh()
    fetchLogs()
    pollInterval = setInterval(refresh, 30000)
  })

  onUnmounted(() => {
    if (pollInterval) clearInterval(pollInterval)
  })

  return {
    containers,
    timers,
    claudeInstances,
    logs,
    logService,
    loading,
    error,
    available,
    refresh,
    fetchLogs,
  }
}
```

- [ ] **Step 2: Verify the `get` function import path**

Check `api/services/frontend/web/src/services/client.ts` for the exact export name and path. The composable imports `get` from `@/services/client`.

- [ ] **Step 3: Commit**

```bash
git add api/services/frontend/web/src/composables/useServerMonitor.ts
git commit -m "feat: add useServerMonitor composable"
```

---

## Task 4: Frontend — Settings Page Server Section

**Files:**
- Modify: `api/services/frontend/web/src/views/SettingsView.vue`

- [ ] **Step 1: Add the Server section to SettingsView**

Add the following after the existing "About" section and before "Actions". The section includes 4 sub-panels: Containers, Logs, Claude, and Timers.

Replace the full contents of `SettingsView.vue`:

```vue
<script setup lang="ts">
import { useSettings } from '@/composables/useSettings'
import { useServerMonitor } from '@/composables/useServerMonitor'
import PageHeader from '@/components/layout/PageHeader.vue'
import { computed, ref } from 'vue'

const { apiBaseUrl, pollingIntervalMs, rowsPerPage, sidebarCollapsed, saved, save, reset } =
  useSettings()

const {
  containers,
  timers,
  claudeInstances,
  logs,
  logService,
  loading,
  error: serverError,
  available,
  refresh,
  fetchLogs,
} = useServerMonitor()

const pollingSeconds = computed({
  get: () => pollingIntervalMs.value / 1000,
  set: (v: number) => {
    pollingIntervalMs.value = v * 1000
  },
})

const serverTab = ref<'containers' | 'logs' | 'claude' | 'timers'>('containers')

const logServices = ['backend', 'frontend', 'db', 'planner-deploy', 'planner-backup']
</script>

<template>
  <div>
    <PageHeader
      title="Settings"
      subtitle="Configure your preferences"
    />

    <div class="p-6 space-y-8 max-w-2xl">
      <!-- API Configuration -->
      <div>
        <h2 class="text-base font-semibold text-gray-100 mb-4 border-b border-gray-800 pb-2">
          API Configuration
        </h2>
        <div>
          <label
            for="api-base-url"
            class="text-sm font-medium text-gray-300 mb-1.5 block"
          >
            API Base URL
          </label>
          <input
            id="api-base-url"
            v-model="apiBaseUrl"
            type="text"
            class="bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100 focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500 w-full outline-none"
            placeholder="http://localhost:8080"
          >
          <p class="text-xs text-gray-500 mt-1">
            The base URL for the Planner API
          </p>
        </div>
      </div>

      <!-- Display Preferences -->
      <div>
        <h2 class="text-base font-semibold text-gray-100 mb-4 border-b border-gray-800 pb-2">
          Display Preferences
        </h2>
        <div class="space-y-4">
          <div>
            <label
              for="polling-interval"
              class="text-sm font-medium text-gray-300 mb-1.5 block"
            >
              Polling Interval (seconds)
            </label>
            <input
              id="polling-interval"
              v-model.number="pollingSeconds"
              type="number"
              min="5"
              class="bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100 focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500 w-full outline-none"
            >
            <p class="text-xs text-gray-500 mt-1">
              How often to refresh data from the API
            </p>
          </div>

          <div>
            <label
              for="rows-per-page"
              class="text-sm font-medium text-gray-300 mb-1.5 block"
            >
              Rows Per Page
            </label>
            <input
              id="rows-per-page"
              v-model.number="rowsPerPage"
              type="number"
              min="5"
              max="100"
              class="bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100 focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500 w-full outline-none"
            >
            <p class="text-xs text-gray-500 mt-1">
              Number of items displayed per page in lists
            </p>
          </div>

          <div class="flex items-center justify-between">
            <div>
              <label
                for="sidebar-collapsed"
                class="text-sm font-medium text-gray-300 block"
              >
                Sidebar Collapsed by Default
              </label>
              <p class="text-xs text-gray-500 mt-0.5">
                Start with the sidebar minimized
              </p>
            </div>
            <label class="relative inline-flex items-center cursor-pointer">
              <input
                id="sidebar-collapsed"
                v-model="sidebarCollapsed"
                type="checkbox"
                class="sr-only peer"
              >
              <div class="w-9 h-5 bg-gray-700 rounded-full peer peer-checked:bg-blue-600 peer-focus:ring-2 peer-focus:ring-blue-500/50 after:content-[''] after:absolute after:top-0.5 after:start-[2px] after:bg-gray-300 after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:after:translate-x-full" />
            </label>
          </div>
        </div>
      </div>

      <!-- Server Monitoring -->
      <div v-if="available">
        <div class="flex items-center justify-between mb-4 border-b border-gray-800 pb-2">
          <h2 class="text-base font-semibold text-gray-100">
            Server
          </h2>
          <button
            class="text-xs text-gray-400 hover:text-gray-200 transition-colors"
            :class="{ 'animate-spin': loading }"
            @click="refresh"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
          </button>
        </div>

        <p
          v-if="serverError"
          class="text-sm text-red-400 mb-3"
        >
          {{ serverError }}
        </p>

        <!-- Tabs -->
        <div class="flex gap-1 mb-4 bg-gray-900 rounded-lg p-1">
          <button
            v-for="tab in (['containers', 'logs', 'claude', 'timers'] as const)"
            :key="tab"
            class="flex-1 px-3 py-1.5 text-xs font-medium rounded-md transition-colors capitalize"
            :class="serverTab === tab
              ? 'bg-gray-800 text-gray-100'
              : 'text-gray-400 hover:text-gray-200'"
            @click="serverTab = tab"
          >
            {{ tab }}
          </button>
        </div>

        <!-- Containers Tab -->
        <div v-if="serverTab === 'containers'" class="space-y-2">
          <div
            v-for="c in containers"
            :key="c.name"
            class="flex items-center justify-between bg-gray-900 rounded-lg px-4 py-3 border border-gray-800"
          >
            <div>
              <p class="text-sm font-medium text-gray-100">{{ c.name }}</p>
              <p class="text-xs text-gray-500">{{ c.image }}</p>
            </div>
            <div class="text-right">
              <span
                class="inline-block px-2 py-0.5 rounded-full text-xs font-medium"
                :class="c.state === 'running'
                  ? 'bg-green-500/20 text-green-400'
                  : 'bg-red-500/20 text-red-400'"
              >
                {{ c.state }}
              </span>
              <p class="text-xs text-gray-500 mt-1">{{ c.status }}</p>
            </div>
          </div>
          <p v-if="containers.length === 0" class="text-sm text-gray-500">No containers found</p>
        </div>

        <!-- Logs Tab -->
        <div v-if="serverTab === 'logs'" class="space-y-3">
          <div class="flex gap-1 flex-wrap">
            <button
              v-for="svc in logServices"
              :key="svc"
              class="px-2.5 py-1 text-xs font-medium rounded-md transition-colors"
              :class="logService === svc
                ? 'bg-blue-600 text-white'
                : 'bg-gray-800 text-gray-400 hover:text-gray-200'"
              @click="fetchLogs(svc)"
            >
              {{ svc }}
            </button>
          </div>
          <pre class="bg-gray-900 border border-gray-800 rounded-lg p-4 text-xs text-gray-300 overflow-x-auto max-h-96 overflow-y-auto font-mono whitespace-pre-wrap">{{ logs || 'No logs available' }}</pre>
        </div>

        <!-- Claude Tab -->
        <div v-if="serverTab === 'claude'" class="space-y-2">
          <div
            v-for="inst in claudeInstances"
            :key="inst.pid"
            class="bg-gray-900 rounded-lg px-4 py-3 border border-gray-800"
          >
            <div class="flex items-center justify-between mb-1">
              <span class="text-sm font-medium text-gray-100">PID {{ inst.pid }}</span>
              <div class="flex gap-3 text-xs text-gray-400">
                <span>CPU {{ inst.cpu }}%</span>
                <span>MEM {{ inst.memory }}%</span>
                <span>{{ inst.elapsed }}</span>
              </div>
            </div>
            <p class="text-xs text-gray-500 font-mono truncate">{{ inst.command }}</p>
          </div>
          <p v-if="claudeInstances.length === 0" class="text-sm text-gray-500">No Claude instances running</p>
        </div>

        <!-- Timers Tab -->
        <div v-if="serverTab === 'timers'" class="space-y-2">
          <div
            v-for="t in timers"
            :key="t.name"
            class="flex items-center justify-between bg-gray-900 rounded-lg px-4 py-3 border border-gray-800"
          >
            <div>
              <p class="text-sm font-medium text-gray-100">{{ t.name }}</p>
              <p class="text-xs text-gray-500">Last: {{ t.lastRun || 'never' }}</p>
            </div>
            <div class="text-right">
              <span
                class="inline-block px-2 py-0.5 rounded-full text-xs font-medium"
                :class="t.active
                  ? 'bg-green-500/20 text-green-400'
                  : 'bg-red-500/20 text-red-400'"
              >
                {{ t.active ? 'active' : 'inactive' }}
              </span>
              <p class="text-xs text-gray-500 mt-1">
                {{ t.result === 'success' ? 'Last run OK' : t.result }}
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- About -->
      <div>
        <h2 class="text-base font-semibold text-gray-100 mb-4 border-b border-gray-800 pb-2">
          About
        </h2>
        <div class="space-y-1 text-sm text-gray-400">
          <p>
            <span class="text-gray-300 font-medium">Planner</span> &mdash; Phase 4
          </p>
          <p>Built with Vue 3, Pinia, and Tailwind CSS</p>
        </div>
      </div>

      <!-- Actions -->
      <div class="flex items-center gap-3 pt-2">
        <button
          class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors"
          @click="save"
        >
          Save
        </button>
        <button
          class="bg-gray-800 hover:bg-gray-700 text-gray-300 px-4 py-2 rounded-lg text-sm font-medium border border-gray-700 transition-colors"
          @click="reset"
        >
          Reset to Defaults
        </button>
        <span
          v-if="saved"
          class="text-sm text-green-400 font-medium"
        >
          Saved!
        </span>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 2: Verify build**

```bash
cd api/services/frontend/web && npm run build
```

Expected: clean build, no type errors.

- [ ] **Step 3: Commit**

```bash
git add api/services/frontend/web/src/
git commit -m "feat: add server monitoring section to settings page"
```

---

## Task 5: Docker Compose + Deployment Config

**Files:**
- Modify: `zarf/compose/docker-compose.yml` (add PLANNER_SIDECAR_URL env var)
- Modify: `.docs/06-infrastructure.md` (document sidecar)
- Modify: `zarf/deploy/VPS-SETUP.md` (add sidecar setup steps)

- [ ] **Step 1: Add sidecar env var to compose backend service**

In `zarf/compose/docker-compose.yml`, add to the backend service environment:

```yaml
PLANNER_SIDECAR_URL: "http://host.docker.internal:9090"
```

Also add `extra_hosts` to the backend service so `host.docker.internal` resolves on Linux:

```yaml
extra_hosts:
  - "host.docker.internal:host-gateway"
```

- [ ] **Step 2: Update docs**

Add sidecar section to `06-infrastructure.md` under Monitoring.

Add sidecar setup steps to `VPS-SETUP.md`:
```bash
# Build sidecar
cd /opt/planner/zarf/sidecar && go build -o sidecar .

# Install systemd unit
sudo cp /opt/planner/zarf/sidecar/planner-sidecar.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now planner-sidecar
```

- [ ] **Step 3: Commit**

```bash
git add zarf/compose/docker-compose.yml .docs/06-infrastructure.md zarf/deploy/VPS-SETUP.md
git commit -m "docs: add sidecar deployment config and documentation"
```

---

## Task 6: End-to-End Verification

- [ ] **Step 1: Build everything**

```bash
go build ./...
cd zarf/sidecar && go build -o sidecar .
cd ../../api/services/frontend/web && npm run build
```

All three should pass with no errors.

- [ ] **Step 2: Local smoke test (sidecar)**

Start the sidecar locally (will work for docker containers, won't have systemd timers on macOS):

```bash
cd zarf/sidecar
./sidecar --api-key=devkey123 --compose-file=../../zarf/compose/docker-compose.yml
```

In another terminal:
```bash
curl -H "X-API-Key: devkey123" http://localhost:9090/containers
curl -H "X-API-Key: devkey123" http://localhost:9090/claude
curl -H "X-API-Key: devkey123" http://localhost:9090/logs/backend?lines=20
```

- [ ] **Step 3: Local smoke test (backend proxy)**

With the backend running (`make dev`) and `PLANNER_SIDECAR_URL=http://localhost:9090`:

```bash
curl -H "X-API-Key: devkey123" http://localhost:8080/api/v1/server/containers
```

- [ ] **Step 4: Frontend verification**

Open the Settings page. The "Server" section should appear with 4 tabs. If the sidecar isn't running, the section hides itself gracefully (`available = false`).

- [ ] **Step 5: Final commit + push**

```bash
git push
```
