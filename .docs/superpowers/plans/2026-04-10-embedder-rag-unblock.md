# AI Model Layer — Embedder Interface + Semantic Search (Unblocking Phases 6 & 8)

**Date:** 2026-04-10
**Phases affected:** 6 (Semantic Search/RAG), 8 (Intelligence Layer prerequisites)
**Motivation:** Phases 6 and 8 are architecturally blocked. Phase 6 needs an Embedder interface, pgvector storage, embedding pipeline, and `search_semantic` MCP tool. Phase 8 needs a Python ML service HTTP contract. This plan builds the model layer interfaces and Phase 6 infrastructure so both phases are unblocked.

---

## Current State

- `claudecli.Client` handles all inference with model escalation (haiku → sonnet → opus)
- `TieredRouter` routes by sensitivity tier (transactions → Ollama, else → FailoverExtractor)
- `OllamaExtractor` calls `/api/generate` with 30s timeout HTTP client
- Ollama is in Docker Compose with `llama3` model
- **No Embedder interface, no pgvector, no embeddings table, no `search_semantic` MCP tool**
- Migration version is at 1.28

---

## Design Decisions

1. **Embedder interface lives in `foundation/`** — it's cross-domain infrastructure (like `claudecli`), not tied to a single bus. New package: `foundation/embed/`.

2. **OllamaEmbedder is the first implementation** — calls Ollama `/api/embed` endpoint with `nomic-embed-text` model (768 dimensions). Same HTTP client pattern as `OllamaExtractor`.

3. **pgvector for vector storage** — switch Postgres image to `pgvector/pgvector:pg16`. New `embeddings` table with `vector(768)` column. Index with `ivfflat` for cosine similarity.

4. **Embedding store is its own domain** — `business/domain/embeddingbus/` with its own Storer interface. It's queried by MCP and written to by the ingestion pipeline. Keeps concerns separated from ingestbus.

5. **Embedding generation hooks into existing ingestion pipeline** — after extraction succeeds in `ingestbus.processRawInput()`, call Embedder on the extracted text. Also embed on note/task/event creation (async goroutine in app handlers, same pattern as auto-classify).

6. **Phase 8 Python ML service contract** — define the HTTP API contract and Go client wrapper now. Actual Python service implementation is deferred. The contract lets Phase 8 work begin independently.

7. **Sensitivity routing for embeddings** — embeddings of transaction data stay local (Ollama embedder). All other content can use any embedder. Reuse TieredRouter pattern at the embedder level.

---

## Task 1: Embedder Interface + OllamaEmbedder

**What:** Define the cross-domain Embedder interface and implement it using Ollama's `/api/embed` endpoint.

**Files:**

### CREATE `foundation/embed/embed.go`

```go
package embed

// Embedder generates vector embeddings from text.
type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    Dimensions() int
}
```

### CREATE `foundation/embed/ollama.go`

```go
// OllamaEmbedder implements Embedder using Ollama's /api/embed endpoint.
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

HTTP pattern mirrors `OllamaExtractor` (`extractor/ollama.go`): `http.NewRequestWithContext` + `client.Do()`, 30s timeout, POST JSON body `{"model": "nomic-embed-text", "input": texts}`.

### CREATE `foundation/embed/ollama_test.go`

Unit test with `httptest.NewServer` mocking `/api/embed`. Assert:
- Output dimensions match configured `dims`
- Multiple texts return correct number of vectors
- Non-200 response returns error

---

## Task 2: pgvector Migration + Embedding Store

**What:** Add pgvector extension, `embeddings` table, and the embeddingbus domain.

**Files:**

### MODIFY `business/sdk/migrate/sql/migrate.sql`

Append v1.29:
```sql
-- Version: 1.29
-- Description: Add pgvector extension and embeddings table
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

### MODIFY Docker Compose — switch Postgres image

In `zarf/compose/docker-compose.yml`, change:
```yaml
image: postgres:16  →  image: pgvector/pgvector:pg16
```

Also update `business/sdk/dbtest/` if it references a Postgres Docker image directly.

### CREATE `business/domain/embeddingbus/model.go`

```go
type Embedding struct {
    ID         uuid.UUID
    SourceType string
    SourceID   uuid.UUID
    Content    string
    Vector     []float32
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type NewEmbedding struct {
    SourceType string
    SourceID   uuid.UUID
    Content    string
    Vector     []float32
}

type SearchResult struct {
    Embedding
    Similarity float64
}
```

### CREATE `business/domain/embeddingbus/embeddingbus.go`

```go
type Storer interface {
    Create(ctx context.Context, emb NewEmbedding) (Embedding, error)
    SearchByVector(ctx context.Context, vec []float32, sourceTypes []string, limit int) ([]SearchResult, error)
    DeleteBySource(ctx context.Context, sourceType string, sourceID uuid.UUID) error
}

type Business struct {
    log      *logger.Logger
    storer   Storer
    embedder embed.Embedder
}

// EmbedAndStore generates an embedding and stores it.
func (b *Business) EmbedAndStore(ctx context.Context, sourceType string, sourceID uuid.UUID, content string) error

// Search embeds the query text, then searches by vector similarity.
func (b *Business) Search(ctx context.Context, query string, sourceTypes []string, limit int) ([]SearchResult, error)
```

### CREATE `business/domain/embeddingbus/stores/embeddingdb/embeddingdb.go`

Store implementation using pgvector operators. `SearchByVector` uses `<=>` (cosine distance):
```sql
SELECT *, 1 - (embedding <=> $1) AS similarity
FROM embeddings
WHERE ($2::text[] IS NULL OR source_type = ANY($2))
ORDER BY embedding <=> $1
LIMIT $3
```

### CREATE `business/domain/embeddingbus/stores/embeddingdb/model.go`

DB struct with pgvector type handling. Use `pgvector-go` package for `vector` column serialization.

---

## Task 3: Embedding Pipeline Integration

**What:** Wire embedding generation into the ingestion pipeline and entity creation handlers.

**Files:**

### MODIFY `business/domain/ingestbus/ingestbus.go`

Add optional `embed.Embedder` field (via `WithEmbedder()` option). In `processRawInput()`, after extraction succeeds (step 6), call embeddingbus to embed the extracted summary text. This is a new step 7.

### MODIFY `app/domain/noteapp/noteapp.go`

In the `create` handler, after note is created, fire async goroutine to embed note content (same pattern as auto-classify goroutine).

### MODIFY `app/domain/taskapp/taskapp.go`

In the `create` handler, fire async goroutine to embed task title + description.

### MODIFY `app/domain/eventapp/eventapp.go`

In the `create` handler, fire async goroutine to embed event content.

### MODIFY `api/services/planner/main.go`

Wire the embedding pipeline:
```go
var embedder embed.Embedder
if ollamaEnabled {
    embedder = embed.NewOllamaEmbedder(cfg.Ollama.URL, "nomic-embed-text", 768)
}

embeddingStore := embeddingdb.NewStore(log, db)
embeddingBus := embeddingbus.NewBusiness(log, embeddingStore, embedder)

// Pass to ingestbus
ingestBus = ingestbus.NewBusiness(...).WithEmbedder(embedder)

// Pass to app handlers that need async embedding
// Pass embeddingBus to noteapp, taskapp, eventapp via Routes.Add()
```

---

## Task 4: `search_semantic` MCP Tool

**What:** Add semantic search MCP tool that embeds the query and searches by vector similarity.

**Files:**

### MODIFY `app/domain/mcpapp/tools.go`

Add `search_semantic` tool definition:
```go
{
    Name: "search_semantic",
    Description: "Search across all your data using natural language.",
    InputSchema: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "query":        map[string]any{"type": "string", "description": "Natural language search query"},
            "source_types": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Limit to: email, task, note, event, context, voice"},
            "limit":        map[string]any{"type": "integer", "description": "Max results (default 10, max 25)"},
        },
        "required": []string{"query"},
    },
}
```

### MODIFY `app/domain/mcpapp/mcpapp.go`

- Add `embeddingBus *embeddingbus.Business` field to app struct (line ~43-59)
- Add `"search_semantic"` case in `callTool()` switch
- Implement `toolSearchSemantic()` handler: unmarshal params, call `embeddingBus.Search()`, return results

### MODIFY `app/domain/mcpapp/route.go`

Pass `embeddingBus` into app struct construction.

---

## Task 5: Python ML Service Contract (Phase 8 Prep)

**What:** Define the HTTP API contract and Go client for the future Python ML service. No Python code yet — just the Go side of the interface.

**Files:**

### CREATE `foundation/mlclient/mlclient.go`

```go
package mlclient

// Client wraps HTTP calls to the Python ML service.
type Client struct {
    url    string
    client *http.Client
    log    *logger.Logger
}

func NewClient(url string, log *logger.Logger) *Client

// ClusterTasks sends task history and returns cluster assignments.
func (c *Client) ClusterTasks(ctx context.Context, tasks []TaskInput) ([]ClusterResult, error)

// FindSimilarSituations returns historically similar contexts.
func (c *Client) FindSimilarSituations(ctx context.Context, contextID uuid.UUID, embeddings [][]float32) ([]SimilarMatch, error)

// Health checks if the ML service is reachable.
func (c *Client) Health(ctx context.Context) error
```

### CREATE `foundation/mlclient/model.go`

Define `TaskInput`, `ClusterResult`, `SimilarMatch` structs matching the planned Python API contract.

This is a stub client — methods return `ErrServiceUnavailable` until the Python service exists. But the interface is real, allowing Phase 8 work to begin coding against it.

---

## Task 6: Infrastructure + Docs

**What:** Docker, Makefile, and documentation updates.

**Files:**

### MODIFY `zarf/compose/docker-compose.yml`

- Change Postgres image: `postgres:16` → `pgvector/pgvector:pg16`
- Add Makefile target: `ollama-pull-embed` → `docker exec planner-ollama ollama pull nomic-embed-text`

### MODIFY `Makefile`

Add `ollama-pull-embed` target.

### MODIFY `.docs/08-ai-model-layer.md`

Add Embedder section (no longer "future — not yet implemented"). Document OllamaEmbedder, dimensions, model choice.

### MODIFY `.docs/07-roadmap.md`

Update Phase 6 status to "In Progress". Note that Embedder interface and pgvector are done, remaining work is pipeline tuning and SKILL.md updates.

### CREATE `.docs/arch/embedding-backend.md`

New arch doc covering embeddingbus domain: file map, types, Storer interface, cascade rules.

---

## Build Sequence

```
Task 1 (Embedder interface + OllamaEmbedder)
  ↓
Task 2 (pgvector migration + embeddingbus domain)  ←── depends on Task 1 for Embedder type
  ↓
Task 3 (Pipeline integration)                       ←── depends on Task 2 for embeddingbus
  ↓
Task 4 (search_semantic MCP tool)                   ←── depends on Task 2 for embeddingbus

Task 5 (Python ML client contract)                  ←── independent, can parallel with Tasks 1-4
Task 6 (Infrastructure + docs)                      ←── after Tasks 1-4
```

## Verification

After all tasks:
1. `make test` passes (pgvector image in dbtest)
2. `make dev-up` starts pgvector-enabled Postgres + pulls embedding model
3. Creating a note/task/event triggers async embedding (check logs)
4. Email ingestion embeds extracted summary after extraction
5. `search_semantic` MCP tool returns ranked results for natural language queries
6. `mlclient.Health()` returns `ErrServiceUnavailable` (expected — no Python service yet)
7. With `PLANNER_OLLAMA_ENABLED=false`, embedding is skipped gracefully (nil Embedder guard)

## Dependencies

- **Go package:** `github.com/pgvector/pgvector-go` — pgvector type support for `database/sql`
- **Docker image:** `pgvector/pgvector:pg16` replaces `postgres:16`
- **Ollama model:** `nomic-embed-text` (768 dimensions, ~274MB)
