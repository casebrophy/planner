# Ollama Frontend System

Ollama integration provides status reporting and model management for local ML services. The frontend queries ollama status (reachability, available models for extraction/embedding), displays model health in the Settings view, and exposes a manual pull endpoint for users to fetch missing models. This is informational and operational, not data-bearing. There is no dedicated store or router entry -- ollama state lives in the `useServerMonitor` composable and surfaces on the Settings view's "ollama" tab.

## Core Types

```typescript
// types/ollama.ts (L1-15)
export interface ModelInfo {
  name: string
  available: boolean
}

export interface OllamaStatus {
  reachable: boolean
  extractModel: ModelInfo
  embedModel: ModelInfo
  allModels: string[]
}

export interface PullResult {
  status: string
}

// Re-exported from types/index.ts (L17)
export type { OllamaStatus, ModelInfo, PullResult } from './ollama'
```

## File Map

### Types
- `types/ollama.ts` (15 lines) -- `ModelInfo`, `OllamaStatus`, `PullResult` interfaces
- `types/index.ts` L17 -- barrel re-export

### Services
- `services/ollamaService.ts` (14 lines) -- **ollamaService** object with two methods:
  - `getStatus(): Promise<OllamaStatus>` -- GET `/api/v1/ollama/status`
  - `pullModel(model: string): Promise<PullResult>` -- POST `/api/v1/ollama/pull/{model}` (URI-encodes model name)
  - Uses `request<T>` from `services/client`

### Composables
- `composables/useServerMonitor.ts` (227 lines) -- **useServerMonitor** -- multi-domain server monitoring composable. Ollama-specific state and methods:
  - `ollamaStatus: Ref<OllamaStatus | null>` (L90) -- cached status, initially null
  - `pullingModel: Ref<string | null>` (L91) -- name of model currently being pulled, null when idle
  - `fetchOllamaStatus()` (L165-170) -- calls `ollamaService.getStatus()`, sets `ollamaStatus.value`
  - `pullOllamaModel(model: string)` (L173-183) -- sets `pullingModel`, calls `ollamaService.pullModel()`, then re-fetches status; clears `pullingModel` in finally block
  - Ollama status is fetched as part of `refresh()` (L192) which runs on mount and every 30s via `setInterval`
  - Both `ollamaStatus` and `pullingModel` are returned from the composable (L219-220)

### Views
- `views/SettingsView.vue` L248-370 -- **Ollama tab** within the server monitoring panel. Rendered when `serverTab === 'ollama'`. Three sections:
  1. **Connection Status** (L254-268) -- green/red badge showing `ollamaStatus.reachable`
  2. **Required Models** (L271-343) -- two rows for extract model (ingest/transactions) and embed model (semantic search). Each shows:
     - Model name (monospace) or em-dash if missing
     - Green "available" badge if `model.available === true`
     - Blue "pull" button if reachable but not available (disabled while any pull is in progress; shows "pulling..." state)
     - Red "unavailable" badge if unreachable
  3. **All Pulled Models** (L347-363) -- lists `ollamaStatus.allModels` as monospace text; hidden when empty
  - Loading state: "Loading Ollama status..." shown when `ollamaStatus` is null (L366-370)

## API Endpoints Consumed

| Method | Endpoint | Used By | Purpose |
|--------|----------|---------|---------|
| GET | `/api/v1/ollama/status` | `ollamaService.getStatus()` | Fetch reachability + model availability |
| POST | `/api/v1/ollama/pull/{model}` | `ollamaService.pullModel()` | Trigger model download on server |

## Component Hierarchy & Data Flow

```
SettingsView.vue
  └── useServerMonitor()
        ├── ollamaService.getStatus() → ollamaStatus ref
        └── ollamaService.pullModel() → pullingModel ref → re-fetch status
```

No dedicated components -- the ollama tab is inlined in `SettingsView.vue`. The `useServerMonitor` composable owns all state and is the sole consumer of `ollamaService`.

## Impact Callouts

### OllamaStatus (types/ollama.ts)
Changing this interface affects:
- `services/ollamaService.ts` -- `getStatus()` response type
- `composables/useServerMonitor.ts` L90 -- `ollamaStatus` ref type
- `views/SettingsView.vue` L261-370 -- template reads `reachable`, `extractModel`, `embedModel`, `allModels`

### ModelInfo (types/ollama.ts)
Changing this interface affects:
- `OllamaStatus` -- `extractModel` and `embedModel` fields
- `views/SettingsView.vue` -- reads `.name` and `.available` for badge/button rendering

### PullResult (types/ollama.ts)
Changing this interface affects:
- `services/ollamaService.ts` -- `pullModel()` return type
- Currently the result is not inspected by the UI (fire-and-refresh pattern)

### ollamaService (services/ollamaService.ts)
Adding or changing methods affects:
- `composables/useServerMonitor.ts` -- sole consumer; calls `getStatus()` and `pullModel()`

## Cross-Domain Dependencies

- **useServerMonitor** -- ollama state is co-located with container, timer, Claude instance, inference, and sidecar log monitoring in a single composable. Changes to the composable's refresh cycle or error handling affect all domains.
- **SettingsView** -- the ollama tab shares tab navigation, polling interval, and error display with all other server monitoring tabs. Adding new ollama UI sections means editing the shared view.
- **services/client** -- `ollamaService` uses the shared `request<T>` function for HTTP calls (auth headers, base URL, error handling).
- **enrichment pipeline (backend)** -- ollama models power the extraction and embedding pipeline. If ollama is unreachable or models are unavailable, enrichment jobs will fail. The status display here is the diagnostic tool for that.
