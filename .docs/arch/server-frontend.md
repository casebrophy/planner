# Server Monitoring Frontend System

> Server monitoring provides real-time visibility into the backend infrastructure: container status (Docker), systemd timers, Claude Code CLI processes, backend logs, sidecar inference logs, and Ollama model availability. The frontend queries monitoring endpoints on the sidecar/backend and displays unified status in the Settings view. State is held in the `useServerMonitor` composable and refreshed on a 30-second polling interval. There is no store or dedicated routes — monitoring is informational-only and surfaces in the Settings view's tabbed interface.

## Core Types

All types are defined within `useServerMonitor.ts` (L10-78):

```typescript
// Container monitoring
export interface ContainerInfo {
  name: string
  state: string
  status: string
  health: string
  image: string
  ports: string
}

// Systemd timer monitoring
export interface TimerInfo {
  name: string
  active: boolean
  lastRun: string
  nextRun: string
  result: string
}

// Claude Code process monitoring
export interface ClaudeInstance {
  pid: string
  command: string
  cpu: string
  memory: string
  elapsed: string
}

// Current inference session metrics
export interface InferenceStatus {
  session_id: string
  created_at: string
  age_seconds: number
  total_requests: number
  latest_input_tokens: number
  context_max: number
  context_usage_pct: number
  token_growth: number[]        // historical token count trend
  avg_duration_ms: number
  duration_trend: number[]      // historical duration trend
  requests_since_rotation: number
}

// Completed inference session summary
export interface SessionSummary {
  session_id: string
  created_at: string
  ended_at: string
  total_requests: number
  end_reason: string
  peak_input_tokens: number
}

// Collection of past sessions
export interface InferenceHistory {
  sessions: SessionSummary[]
}

// Tool usage metrics for current session
export interface InferenceTools {
  tool_frequency: Record<string, number>  // e.g., {"bd.todo.create": 5, "Read": 12}
  avg_calls_per_request: number
}

// Individual sidecar log entry
export interface SidecarLogEntry {
  id: string
  timestamp: string
  duration_ms: number
  agent_model: string           // "haiku", "sonnet", "opus"
  prompt_prefix: string         // first ~100 chars of prompt
  input_tokens: number
  output_tokens: number
  session_id: string
  success: boolean
  error?: string                // present if success === false
}

// Ollama ML service status (re-used from ollama.ts)
export interface OllamaStatus {
  reachable: boolean
  extractModel: ModelInfo
  embedModel: ModelInfo
  allModels: string[]
}

export interface ModelInfo {
  name: string
  available: boolean
}
```

## File Map

### Types
- `composables/useServerMonitor.ts` L10-78 -- **Type definitions** (ContainerInfo, TimerInfo, ClaudeInstance, InferenceStatus, SessionSummary, InferenceHistory, InferenceTools, SidecarLogEntry)
- `types/ollama.ts` L1-15 -- **OllamaStatus**, **ModelInfo**, **PullResult** interfaces
- `types/index.ts` L17 -- barrel re-export of ollama types

### Services
- `services/ollamaService.ts` (14 lines) -- **ollamaService** object with two methods:
  - `getStatus(): Promise<OllamaStatus>` -- GET `/api/v1/ollama/status`
  - `pullModel(model: string): Promise<PullResult>` -- POST `/api/v1/ollama/pull/{model}` (URI-encodes model name)
  - Uses `request<T>` from `services/client`

### Composables
- `composables/useServerMonitor.ts` (227 lines) -- **useServerMonitor** -- unified server monitoring orchestrator. Manages:
  - **Container state** (L81, L100-107):
    - `containers: Ref<ContainerInfo[]>`
    - `fetchContainers()` -- GET `/api/v1/server/containers`
  - **Timer state** (L82, L109-115):
    - `timers: Ref<TimerInfo[]>`
    - `fetchTimers()` -- GET `/api/v1/server/timers`
  - **Claude process state** (L83, L117-123):
    - `claudeInstances: Ref<ClaudeInstance[]>`
    - `fetchClaude()` -- GET `/api/v1/server/claude`
  - **Logs** (L84, L125-138):
    - `logs: Ref<string>` -- backend/sidecar raw logs
    - `logService: Ref<string>` -- which service to display ("backend", "sidecar", or other)
    - `fetchLogs(service?: string)` -- GET `/api/v1/server/logs/{service}?lines=100` or `/api/v1/server/logs/sidecar?limit=50` for structured sidecar logs
  - **Sidecar logs** (L85):
    - `sidecarLogs: Ref<SidecarLogEntry[]>` -- parsed sidecar inference logs
  - **Inference metrics** (L87-89, L140-163):
    - `inferenceStatus: Ref<InferenceStatus | null>` -- current session metrics
    - `inferenceHistory: Ref<SessionSummary[]>` -- past sessions
    - `inferenceTools: Ref<InferenceTools | null>` -- tool frequency in current session
    - `fetchInferenceStatus()` -- GET `/api/v1/server/inference/status`
    - `fetchInferenceHistory()` -- GET `/api/v1/server/inference/history` (extracts `.sessions` array)
    - `fetchInferenceTools()` -- GET `/api/v1/server/inference/tools`
    - `refreshInference()` -- Promise.all() of all three fetch methods
  - **Ollama state** (L90, L165-183):
    - `ollamaStatus: Ref<OllamaStatus | null>`
    - `pullingModel: Ref<string | null>` -- model name currently being pulled
    - `fetchOllamaStatus()` -- calls `ollamaService.getStatus()`
    - `pullOllamaModel(model: string)` -- calls `ollamaService.pullModel()` then re-fetches status
  - **Global state** (L92-94):
    - `loading: Ref<boolean>` -- set during `refresh()`
    - `error: Ref<string>` -- error message from any fetch
    - `available: Ref<boolean>` -- set to false if sidecar is unreachable
  - **Refresh logic** (L189-200):
    - `refresh()` -- Promise.all() of fetchContainers, fetchTimers, fetchClaude, refreshInference, fetchOllamaStatus; sets loading state
    - Polling: `setInterval(refresh, 30000)` on mount (L199); cleared on unmount (L202-203)
  - **Returns** (L206-226): all state refs and methods

### Views
- `views/SettingsView.vue` L1-370 -- **Settings view** with unified server monitoring panel:
  - L2-3: imports useSettings and useServerMonitor
  - L10-28: destructures all server monitoring state and methods
  - L37: `serverTab` ref with tabs: 'containers', 'logs', 'claude', 'timers', 'inference', 'ollama'
  - L73-370: tabbed interface for each monitoring domain:
    - **Containers tab** (L73-100): displays `containers` array, each with name, state, health, image, ports
    - **Timers tab** (L102-137): displays `timers` array with last/next run timestamps
    - **Claude tab** (L139-165): displays `claudeInstances` array with CPU/memory/elapsed
    - **Logs tab** (L167-230): shows either raw logs (backend/other) or sidecar logs (structured). Service selector. For sidecar logs, displays SidecarLogEntry table with duration, model, tokens, success/error
    - **Inference tab** (L232-380): three sub-sections:
      - Current session: InferenceStatus metrics (context usage %, token growth chart, request rate, duration trend)
      - Session history: SessionSummary table (created, ended, request count, end reason)
      - Tool frequency: InferenceTools table showing tool names and call counts
    - **Ollama tab** (L348-370): ollama connection status, required models (extract/embed), pull buttons, all available models list
  - L24: `available` prop — hides entire server panel with "Sidecar unavailable" message if false

## API Endpoints Consumed

| Method | Endpoint | Used By | Purpose |
|--------|----------|---------|---------|
| GET | `/api/v1/server/containers` | `fetchContainers()` | Fetch Docker container info (state, health, image, ports) |
| GET | `/api/v1/server/timers` | `fetchTimers()` | Fetch systemd timer status (active, last/next run) |
| GET | `/api/v1/server/claude` | `fetchClaude()` | Fetch Claude Code process list (PID, CPU, memory, elapsed) |
| GET | `/api/v1/server/logs/{service}?lines=N` | `fetchLogs()` | Fetch raw logs from backend or other service |
| GET | `/api/v1/server/logs/sidecar?limit=N` | `fetchLogs()` | Fetch structured sidecar inference logs |
| GET | `/api/v1/server/inference/status` | `fetchInferenceStatus()` | Fetch current session metrics (tokens, context %, trends) |
| GET | `/api/v1/server/inference/history` | `fetchInferenceHistory()` | Fetch past session summaries |
| GET | `/api/v1/server/inference/tools` | `fetchInferenceTools()` | Fetch tool frequency for current session |
| GET | `/api/v1/ollama/status` | `ollamaService.getStatus()` | Fetch Ollama service status and model availability |
| POST | `/api/v1/ollama/pull/{model}` | `ollamaService.pullModel()` | Trigger model download on Ollama server |

## Component Hierarchy & Data Flow

```
SettingsView.vue
  └── useServerMonitor() (single instance, shared polling)
        ├── fetchContainers() → containers array
        ├── fetchTimers() → timers array
        ├── fetchClaude() → claudeInstances array
        ├── fetchLogs(service) → logs string or sidecarLogs array
        ├── fetchInferenceStatus() → inferenceStatus ref
        ├── fetchInferenceHistory() → inferenceHistory array
        ├── fetchInferenceTools() → inferenceTools ref
        ├── fetchOllamaStatus() → ollamaStatus ref (via ollamaService)
        ├── pullOllamaModel() → calls ollamaService.pullModel() → re-fetches status
        └── refresh() -- called every 30s, also on manual tab change/refresh button
```

No dedicated components or stores. All state and methods are co-located in the composable and consumed directly by SettingsView.

## Impact Callouts

### ContainerInfo (composables/useServerMonitor.ts L12-18)
Changing this interface affects:
- `useServerMonitor()` L81 -- `containers` ref type
- `views/SettingsView.vue` L73-100 -- template reads `.name`, `.state`, `.health`, `.image`, `.ports` for table rows

### TimerInfo (composables/useServerMonitor.ts L20-25)
Changing this interface affects:
- `useServerMonitor()` L82 -- `timers` ref type
- `views/SettingsView.vue` L102-137 -- template reads `.name`, `.active`, `.lastRun`, `.nextRun`, `.result`

### ClaudeInstance (composables/useServerMonitor.ts L27-33)
Changing this interface affects:
- `useServerMonitor()` L83 -- `claudeInstances` ref type
- `views/SettingsView.vue` L139-165 -- template reads `.pid`, `.command`, `.cpu`, `.memory`, `.elapsed`

### InferenceStatus (composables/useServerMonitor.ts L35-47)
Changing this interface affects:
- `useServerMonitor()` L87 -- `inferenceStatus` ref type
- `views/SettingsView.vue` L290-330 -- template reads `.context_usage_pct`, `.token_growth`, `.avg_duration_ms`, `.duration_trend`, `.requests_since_rotation`, `.age_seconds`
- Chart rendering (token growth and duration trend arrays)

### SessionSummary (composables/useServerMonitor.ts L49-56)
Changing this interface affects:
- `useServerMonitor()` L88 -- elements in `inferenceHistory` array
- `views/SettingsView.vue` L332-350 -- template reads `.created_at`, `.ended_at`, `.total_requests`, `.end_reason`, `.peak_input_tokens` for table rows

### InferenceTools (composables/useServerMonitor.ts L58-65)
Changing this interface affects:
- `useServerMonitor()` L89 -- `inferenceTools` ref type
- `views/SettingsView.vue` L352-370 -- template iterates `.tool_frequency` entries and reads `.avg_calls_per_request`

### SidecarLogEntry (composables/useServerMonitor.ts L67-78)
Changing this interface affects:
- `useServerMonitor()` L85 -- elements in `sidecarLogs` array
- `views/SettingsView.vue` L186-210 -- template reads `.timestamp`, `.duration_ms`, `.agent_model`, `.prompt_prefix`, `.input_tokens`, `.output_tokens`, `.success`, `.error` for table rows

### OllamaStatus & ModelInfo (types/ollama.ts)
Changing these interfaces affects:
- `services/ollamaService.ts` -- `getStatus()` response type
- `composables/useServerMonitor.ts` L90 -- `ollamaStatus` ref type
- `views/SettingsView.vue` L248-370 -- template reads `.reachable`, `.extractModel`, `.embedModel`, `.allModels` for badge/button/list rendering
- See `ollama-frontend.md` for full impact detail

### ollamaService (services/ollamaService.ts)
Adding or changing methods affects:
- `composables/useServerMonitor.ts` -- sole consumer; calls `getStatus()` and `pullModel()` (L167, L176)

## Cross-Domain Dependencies

- **views/SettingsView.vue** -- shares tab navigation, polling interval settings, and error display with general settings (API base URL, polling interval, rows per page). The server monitoring panel is one section of a larger settings view.
- **services/client** -- all `useServerMonitor` fetch methods and `ollamaService` use the shared `request<T>` function for HTTP calls (auth headers via X-API-Key, base URL, error handling).
- **backend serverapp** -- all `/api/v1/server/*` and `/api/v1/ollama/*` endpoints are proxied from the backend to the sidecar (no database involvement). Frontend calls backend, backend proxies to sidecar at `127.0.0.1:9090`.
- **Ollama service (external)** -- the sidecar exposes Ollama status and pull endpoints. If Ollama is unreachable, the status display reflects that. Model pull is fire-and-refresh (result not inspected).
- **Inference pipeline (backend)** -- the sidecar's inference metrics reflect activity from the backend's AI orchestration. Polling shows session age, token growth, and session rotation events.
- **Enrichment pipeline (backend)** -- Ollama models power document extraction and semantic embedding. The status display here is the diagnostic tool for enrichment failures.
