# AI model layer

The AI model layer uses the Claude Code CLI (`claude -p`) for all inference, leveraging a Claude Max subscription instead of API keys. Model escalation starts with the cheapest model and bumps up when results are low quality.

---

## Claude CLI client

All AI inference runs through `foundation/claudecli/claudecli.go`, which wraps `exec.CommandContext` calls to `claude -p`.

```go
type Client struct {
    cliPath string        // default "claude"
    models  []string      // escalation chain, e.g. ["haiku", "sonnet", "opus"]
    timeout time.Duration // default 120s
    log     *logger.Logger
}

// RunJSON tries models in escalation order.
// After each successful parse, calls shouldEscalate() — if true, retries with next model.
// Final model's result is always accepted.
func (c *Client) RunJSON(ctx context.Context, prompt string, schema string, dest any, shouldEscalate func() bool) error
```

Each call: `claude -p <prompt> --output-format json --json-schema <schema> --model <model> --bare`

### Model escalation

Models are tried in order (default: haiku → sonnet → opus). Escalation triggers:
- **CLI error** (non-zero exit) → automatic escalation
- **JSON parse failure** → automatic escalation
- **`shouldEscalate()` returns true** (caller-defined quality check) → escalation
- **Last model in chain** → accept whatever we got

Each extractor defines its own quality threshold:
- Email extraction: escalate if zero action items AND context confidence < 0.3
- Thread extraction: escalate if confidence < 0.4

---

## Orchestrator architecture (sidecar)

The sidecar (`zarf/sidecar/`) runs a persistent Claude session (Opus) that acts as an inference dispatcher. Instead of spawning a fresh `claude -p` process per request, the sidecar maintains a session via `--resume` and dispatches work to subagents.

### Session lifecycle

1. **First request** — `claude -p` with `--system-prompt` and `--model opus`. Session ID captured from output.
2. **Subsequent requests** — `--resume <session_id>`. Orchestrator remembers its role and dispatches subagents at the model specified in the request.
3. **Rotation** — session discarded and restarted when: request timeout (180s default), CLI error, input tokens exceed threshold (150k default), or manual rotation.

### How it works

The orchestrator receives a JSON message with `model`, `prompt`, and optional `schema`. It spawns a subagent using the Agent tool at the specified model. The subagent has **read-only** MCP access to the planner API for context queries. The orchestrator returns only the subagent's raw output.

### Observability endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/inference/status` | GET | Current session health: token growth, duration trend, context usage % |
| `/inference/history` | GET | Past session summaries (requests served, peak tokens, end reason) |
| `/inference/tools` | GET | Tool call frequency and avg calls per request |
| `/inference/rotate` | POST | Force session rotation |

### Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `SIDECAR_CONTEXT_MAX` | `150000` | Token threshold for auto-rotation |
| `SIDECAR_REQUEST_TIMEOUT` | `180s` | Per-request timeout |
| `PLANNER_MCP_URL` | `http://localhost:8080/mcp` | MCP endpoint for agent tools |

---

## Sensitivity-tier routing

Financial data (bank transactions) must never leave the local machine. The `TieredRouter` (`business/domain/ingestbus/extractor/router.go`) enforces this policy:

| Data type | Route | Extractor |
|-----------|-------|-----------|
| Transaction (`typeHint="transaction"`) | Local only | `OllamaExtractor` |
| Email | General | `FailoverExtractor` (Claude → Ollama fallback) |
| Voice/text (other) | General | `FailoverExtractor` (Claude → Ollama fallback) |

**Wiring** (`main.go`): When `PLANNER_OLLAMA_ENABLED=true`, creates `TieredRouter(general=FailoverExtractor, localOnly=OllamaExtractor)`. When disabled, uses bare `ClaudeCodeExtractor`.

**Transaction enrichment**: `transactionbus.Business` has an optional `Enricher` interface. When Ollama is enabled, an `ExtractorEnricher` adapter wraps the `Extractor` and enriches transactions asynchronously after CSV import — cleaning merchant names, suggesting categories, and matching contexts.

---

## Extractors

### Email extraction

`business/domain/ingestbus/extractor/claudecli.go` implements the `Extractor` interface.

Called by the ingestion pipeline (`ingestbus.processRawInput()` step 6) to extract structured data from emails: summary, action items, deadlines, sentiment, context matching.

### Thread entry extraction

`business/domain/threadbus/claudecli_extractor.go` implements the `threadbus.Extractor` interface.

Called by `threadbus.AddEntry()` when `Extract` flag is set. Classifies thread entries into kinds (update, blocker, decision, etc.) with sentiment and blocking party detection.

### Shared prompt

`business/domain/ingestbus/extractor/prompt.go` contains shared prompt templates: `BuildEmailExtractionPrompt()` for emails, `BuildTextExtractionPrompt()` which dispatches to type-specific prompts (task, event, note, transaction) based on `typeHint`, and `buildTransactionExtractionPrompt()` for merchant name cleanup and category suggestion.

### Mock extractor

`business/domain/ingestbus/extractor/mock.go` — used in tests. Returns configured result/error.

### Composite MCP tool: `get_inference_context`

A single-call tool that returns pre-assembled context for each inference pipeline, reducing multi-tool round trips by subagents.

| Use Case | Data Returned |
|----------|---------------|
| `daily_plan` | Open + in-progress tasks, today's events, active contexts |
| `email_extraction` | Active contexts for matching |
| `text_extraction` | Active contexts + today's events |
| `thread_classification` | Thread history (last 10), subject details |

Implemented in `app/domain/mcpapp/mcpapp.go` alongside existing MCP tools.

---

## Embedder

Cross-domain interface lives in `foundation/embed/`. Generates vector embeddings from text for semantic search (Phase 6).

```go
type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    Dimensions() int
}
```

### OllamaEmbedder

Default implementation: `foundation/embed/ollama.go`. Calls Ollama's `/api/embed` endpoint with the `nomic-embed-text` model (768 dimensions).

```go
type OllamaEmbedder struct {
    url    string
    model  string
    client *http.Client
    dims   int
}

func NewOllamaEmbedder(url, model string, dims int) *OllamaEmbedder
func (e *OllamaEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error)
func (e *OllamaEmbedder) Dimensions() int
```

HTTP pattern mirrors `OllamaExtractor`: `http.NewRequestWithContext` + `client.Do()`, 30s timeout, POST JSON body `{"model": "nomic-embed-text", "input": texts}`.

### Storage

Embeddings are stored in the `embeddings` table (pgvector extension, v1.29 migration):

```sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE embeddings (
    embedding_id  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    source_type   TEXT        NOT NULL,  -- 'email', 'task', 'note', 'event', 'context', 'voice'
    source_id     UUID        NOT NULL,
    content       TEXT        NOT NULL,  -- the text that was embedded
    embedding     vector(768) NOT NULL,
    created_at    TIMESTAMP   NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_embeddings_source ON embeddings(source_type, source_id);
CREATE INDEX idx_embeddings_vector ON embeddings USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
```

### Domain: embeddingbus

Business logic lives in `business/domain/embeddingbus/`. The Storer interface provides:

- `Create(ctx context.Context, emb NewEmbedding) (Embedding, error)` — store a new embedding
- `SearchByVector(ctx context.Context, vec []float32, sourceTypes []string, limit int) ([]SearchResult, error)` — cosine similarity search using pgvector `<=>` operator
- `DeleteBySource(ctx context.Context, sourceType string, sourceID uuid.UUID) error` — clean up embeddings when a source is deleted

The Business struct holds the Embedder, Storer, and logger. Public methods:

- `EmbedAndStore()` — generate embedding and store it
- `Search()` — embed the query text, then search by vector similarity

---

## RAG: semantic search

### What gets embedded

| Content | Chunking | Indexed fields |
|---------|----------|----------------|
| Email body | Paragraph-level | Sanitized summary + action items |
| Context events | One chunk per event | Event content |
| Task notes | One chunk per note | Note content |
| Task title + description | Single chunk | Title + description combined |
| Context summary | Single chunk | Full summary |
| Voice captures | Sentence-level | Transcript |
| Transactions | Not embedded | SQL handles these |
| Raw email HTML | Not embedded | Plain text only |

### Vector storage DDL

```sql
CREATE VIRTUAL TABLE embeddings USING vec0(
    id          TEXT PRIMARY KEY,
    source_type TEXT,
    source_id   TEXT,
    content     TEXT,
    tier        INTEGER,
    created_at  TEXT,
    embedding   FLOAT[768]
);
```

### MCP tool: `search_semantic`

```json
{
  "name": "search_semantic",
  "description": "Search across all your data using natural language.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "query":        {"type": "string",  "description": "Natural language search query"},
      "after":        {"type": "string",  "description": "ISO date — only return results after this date"},
      "before":       {"type": "string",  "description": "ISO date — only return results before this date"},
      "source_types": {"type": "array",   "items": {"type": "string"}, "description": "Limit to: email, context_event, task_note, task, voice"},
      "context_id":   {"type": "string",  "description": "Limit results to a specific context"},
      "limit":        {"type": "integer", "description": "Max results (default 10, max 25)"}
    },
    "required": ["query"]
  }
}
```

---

## Configuration reference

```bash
PLANNER_CLAUDE_CLI_PATH=claude           # path to claude binary
PLANNER_CLAUDE_MODELS=haiku,sonnet,opus  # escalation chain (tried in order)
```

No API keys required — uses the Claude Max subscription via the CLI.
