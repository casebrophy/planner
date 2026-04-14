# Ollama Backend System

Ollama integration provides HTTP endpoints to check the status of an external Ollama service and trigger model pulls. This is a thin handler layer with no business/store logic — it acts as a proxy and status monitor for the Ollama API. The Ollama service itself is consumed by the ingestion pipeline (extractor, embedder) configured in `main.go`.

## Core Types

### Response Types (Handlers)

```go
// OllamaStatus is the response from the GET /api/v1/ollama/status endpoint.
type OllamaStatus struct {
    Reachable    bool      `json:"reachable"`      // Whether Ollama service is reachable
    ExtractModel ModelInfo `json:"extractModel"`   // Status of extraction model
    EmbedModel   ModelInfo `json:"embedModel"`     // Status of embedding model
    AllModels    []string  `json:"allModels"`      // All models available on Ollama
}

// ModelInfo describes one expected model and whether it's available.
type ModelInfo struct {
    Name      string `json:"name"`      // Model name (e.g., "qwen3.5:0.8b")
    Available bool   `json:"available"` // True if model is downloaded/available
}
```

### Internal Request/Response Types

```go
// pullRequest is sent to Ollama /api/pull endpoint.
type pullRequest struct {
    Name   string `json:"name"`   // Model name to pull
    Stream bool   `json:"stream"` // Streaming flag (always false)
}

// pullResult is the response from Ollama pull request.
type pullResult struct {
    Status string `json:"status"` // Status message
}

// ollamaTagsResponse matches the Ollama /api/tags response shape.
type ollamaTagsResponse struct {
    Models []struct {
        Name string `json:"name"`
    } `json:"models"`
}
```

### Configuration (from `app/sdk/mux.Config`)

```go
type Config struct {
    // Ollama configuration
    OllamaURL        string // Base URL of Ollama service (e.g., "http://localhost:11434")
    OllamaModel      string // Model name for extraction (e.g., "qwen3.5:0.8b")
    OllamaEmbedModel string // Model name for embeddings (e.g., "qwen3-embed:0.6b")
    OllamaEnabled    bool   // Whether Ollama is enabled (true if URL is set and feature enabled)
}
```

## File Map

### Handlers
- `app/domain/ollamaapp/ollamaapp.go` — **status()**, **pull()** — HTTP handler methods
- `app/domain/ollamaapp/route.go` — **Routes.Add()** — HTTP route registration

## Handler Functions

| Handler | Method | Route | Purpose |
|---------|--------|-------|---------|
| status | GET | /api/v1/ollama/status | Check Ollama service reachability and model availability |
| pull | POST | /api/v1/ollama/pull/{model} | Trigger a model pull from Ollama registry |

### status() — `app/domain/ollamaapp/ollamaapp.go:50`

**HTTP:** GET /api/v1/ollama/status

**Purpose:** Report the status of the Ollama service and whether required models are available.

**Behavior:**
1. Returns pre-configured extract/embed model names (from config)
2. If Ollama is disabled or URL not set, returns status with `Reachable=false`
3. Probes Ollama's `/api/tags` endpoint (3-second timeout) to list available models
4. Compares configured model names against available models (handles model tags like `name:tag`)
5. Returns `OllamaStatus` with reachability and availability info
6. Uses 3-second timeout probeClient (lightweight)

**Returns:** `OllamaStatus` (HTTP 200 JSON)

### pull() — `app/domain/ollamaapp/ollamaapp.go:120`

**HTTP:** POST /api/v1/ollama/pull/{model}

**Purpose:** Trigger download/cache of a model from Ollama registry.

**Validation:**
- Returns 400 InvalidArgument if Ollama disabled or URL not set
- Only allows pulling the configured `extractModel` or `embedModel` (security: prevent pulling arbitrary models)

**Behavior:**
1. Validates {model} path param matches configured extract or embed model
2. Marshals a pullRequest with `Stream=false` to Ollama's `/api/pull` endpoint
3. Uses 10-minute timeout (model pulls can take time)
4. Polls Ollama until completion
5. Parses response status and returns it

**Returns:** `pullResult` with status message (HTTP 200 JSON)

**Error Cases:**
- 400 InvalidArgument: Ollama disabled, unknown model requested
- 500 Internal: HTTP/marshal/decode errors from Ollama

## Routes

| Method | Path | Handler | Auth | Purpose |
|--------|------|---------|------|---------|
| GET | /api/v1/ollama/status | status() | API Key | Status probe |
| POST | /api/v1/ollama/pull/{model} | pull() | API Key | Model pull trigger |

## Cross-Domain Dependencies

### Ingestion Pipeline Integration (main.go)

Ollama is wired into the ingestion pipeline at startup, **but the handlers do not directly call into business logic:**

- **Extractor:** `main.go` creates `extractor.NewOllamaExtractor()` and configures a tiered router (Claude Code → Ollama failover)
- **Embedder:** `main.go` creates `embed.NewOllamaEmbedder()` for embedding operations
- **Configuration:** Models and URL passed via `mux.Config` to `Routes.Add()`

### No Direct Store Dependencies

The ollama handlers are pure HTTP proxies — they do not read from or write to any application store. They forward requests to the external Ollama service and report its status.

## Notes

- **External Service Dependency:** Ollama is an external service; handlers gracefully degrade if unreachable (no error on status probe, validation errors on pull request)
- **Model Tag Handling:** The `stripTag()` helper removes Ollama model tags (e.g., `qwen3.5:0.8b` → `qwen3.5`) for availability comparison
- **No Streaming:** Pull requests explicitly set `Stream=false`, so the client waits for completion
- **Timeouts:** Status probes use 3s timeout (fail-fast), pull operations use 10m timeout (allow long downloads)
