# Ollama System

> Ollama integration provides status reporting and model management for local ML services. The frontend queries ollama status (reachability, available models for extraction/embedding), displays model health, and exposes a manual pull endpoint for users to fetch new models. This is informational and operational, not data-bearing.

## Core Types

```typescript
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
```

## File Map

### Services
- `services/ollamaService.ts` — **ollamaService** — queries /api/v1/ollama/status (GET) and /api/v1/ollama/pull/{model} (POST)

## Impact Callouts

### ⚠ OllamaStatus (types/ollama.ts)
Changing this interface affects:
- `services/ollamaService.ts` — getStatus() response deserialization
- Any admin/settings view that displays model status (not yet in main nav, but called from enrichment status board)

### ⚠ ModelInfo (types/ollama.ts)
Changing this interface affects:
- `OllamaStatus` — extractModel and embedModel fields reference this type
- Any UI rendering model availability (name display, available boolean for enable/disable icons)

## Cross-Domain Dependencies

- **enrichment pipeline** — rawinput processing depends on ollama models for extraction/embedding
- **transaction domain** — enrichment status board (on transaction board view) queries ollama status as part of live status bar
