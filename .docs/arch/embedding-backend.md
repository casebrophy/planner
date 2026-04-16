# Embedding Domain Architecture

**Domain:** `business/domain/embeddingbus`  
**Infrastructure:** `foundation/embed` (Embedder interface)  
**Store:** `business/domain/embeddingbus/stores/embeddingdb`  
**Database:** pgvector-enabled PostgreSQL (v1.29 migration)

---

## Overview

The embedding domain provides semantic search across all planner data. It generates vector embeddings from captured content (emails, tasks, notes, events, voice) and enables natural language queries via the `search_semantic` MCP tool.

**Three layers:**
- **Foundation** (`foundation/embed/`): Cross-domain Embedder interface
- **Business** (`embeddingbus`): Embedding operations and vector search
- **Store** (`embeddingdb`): pgvector SQL operations

**Sensitivity routing:** Embeddings of transaction data stay local (OllamaEmbedder). All other content can be embedded with any configured embedder.

---

## File Map

```
foundation/embed/
  ├── embed.go            # Embedder interface
  ├── embed_test.go       # Unit tests (httptest mocking)
  └── ollama.go           # OllamaEmbedder implementation
      └── ollama_test.go  # OllamaEmbedder tests

business/domain/embeddingbus/
  ├── model.go            # Embedding, NewEmbedding, SearchResult
  ├── embeddingbus.go     # Business struct, Storer interface, EmbedAndStore, Search
  ├── filter.go           # QueryFilter (source types, date range)
  ├── order.go            # OrderBy constants
  └── stores/
      └── embeddingdb/
          ├── model.go    # DB struct + conversions (toEmbedding, toDBEmbedding)
          ├── embeddingdb.go  # Storer implementation, pgvector SQL
          ├── filter.go   # applyFilter() → WHERE clauses
          └── order.go    # orderByFields map + orderByClause()
```

---

## Types

### Foundation Layer (`foundation/embed`)

```go
// Embedder generates vector embeddings from text.
type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    Dimensions() int
}
```

### Business Layer (`embeddingbus`)

```go
// Embedding is a stored vector embedding.
type Embedding struct {
    ID         uuid.UUID
    SourceType string      // 'email', 'task', 'note', 'event', 'context', 'voice'
    SourceID   uuid.UUID
    Content    string      // the text that was embedded
    Vector     []float32   // 1024-dimensional vector
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

// NewEmbedding is the input to create an embedding.
type NewEmbedding struct {
    SourceType string
    SourceID   uuid.UUID
    Content    string
    Vector     []float32
}

// SearchResult is an embedding with similarity score.
type SearchResult struct {
    Embedding
    Similarity float64  // cosine similarity [0, 1]
}

// QueryFilter narrows search by source and date.
type QueryFilter struct {
    SourceTypes []string
    CreatedAfter *time.Time
    CreatedBefore *time.Time
}
```

### Store Layer (`embeddingdb`)

```go
// DB struct mirrors the embeddings table.
type Embedding struct {
    ID         string      // UUID
    SourceType string
    SourceID   string      // UUID
    Content    string
    Embedding  pgvector.Vector  // pgvector-go type ([]float32 serialized)
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

// Conversions
func toEmbedding(db Embedding) embeddingbus.Embedding
func toDBEmbedding(emb embeddingbus.NewEmbedding) Embedding
```

---

## Storer Interface

```go
type Storer interface {
    // Create stores a new embedding.
    Create(ctx context.Context, emb NewEmbedding) (Embedding, error)

    // SearchByVector finds embeddings near the query vector using cosine distance.
    // sourceTypes filters by source (nil = all); limit caps results.
    SearchByVector(ctx context.Context, vec []float32, sourceTypes []string, limit int) ([]SearchResult, error)

    // DeleteBySource removes all embeddings for a source entity.
    DeleteBySource(ctx context.Context, sourceType string, sourceID uuid.UUID) error

    // ExistsBySource checks if an embedding exists for a given source entity.
    ExistsBySource(ctx context.Context, sourceType string, sourceID uuid.UUID) (bool, error)

    // Query returns embeddings matching a filter (pagination support for future UI).
    Query(ctx context.Context, filter QueryFilter, order OrderBy, page page.Page) ([]Embedding, error)
}
```

---

## Business Methods

```go
type Business struct {
    log      *logger.Logger
    storer   Storer
    embedder embed.Embedder  // optional; nil when embedding is disabled
}

// EmbedAndStore generates an embedding and stores it.
// Returns early (no error) if embedder is nil.
func (b *Business) EmbedAndStore(ctx context.Context, sourceType string, sourceID uuid.UUID, content string) error

// DeleteBySource removes embeddings for a given source entity (wrapper).
// Called during entity updates (skip_classify path) to regenerate embeddings.
// Phase 4 addition: used by ingestbus to clear old embeddings before reprocessing.
func (b *Business) DeleteBySource(ctx context.Context, sourceType string, sourceID uuid.UUID) error

// Search embeds the query text, then searches by vector similarity.
// Returns embeddings ranked by cosine similarity (highest first).
func (b *Business) Search(ctx context.Context, query string, sourceTypes []string, limit int) ([]SearchResult, error)
```

---

## SQL Operations

### Create

```sql
INSERT INTO embeddings (embedding_id, source_type, source_id, content, embedding)
VALUES ($1, $2, $3, $4, $5::vector)
RETURNING *
```

### Search (cosine similarity)

```sql
SELECT 
    embedding_id, source_type, source_id, content, embedding,
    1 - (embedding <=> $1::vector) AS similarity,
    created_at, updated_at
FROM embeddings
WHERE ($2::text[] IS NULL OR source_type = ANY($2::text[]))
ORDER BY embedding <=> $1::vector
LIMIT $3
```

The `<=>` operator is pgvector's cosine distance. We compute `1 - distance` to get similarity [0, 1].
`$2` is wrapped with `pq.Array()` in the Go code to properly encode the sourceTypes slice as a PostgreSQL text array.

### Delete by source

```sql
DELETE FROM embeddings
WHERE source_type = $1 AND source_id = $2::uuid
```

---

## Integration Points

### Embedding Generation

**On entity creation:** noteapp, taskapp, eventapp fire async goroutines (fire-and-forget):
```go
go func() {
    embeddingBus.EmbedAndStore(
        context.Background(),
        "note",
        note.ID,
        note.Content,
    )
}()
```
Errors are logged internally by `EmbedAndStore()`; callers do not capture or handle the error.

**On email ingestion:** ingestbus calls embeddingbus after extraction:
```go
embeddingBus.EmbedAndStore(ctx, "email", rawInputID, extractedSummary)
```

### Embedding Regeneration (Phase 4)

**skip_classify path in ingestbus:** When a raw_input has `skip_classify=true` and `source_entity_id` is set (pointing to an existing entity), ingestbus calls:
```go
embeddingBus.DeleteBySource(ctx, entityKind, entityID)  // Delete old embeddings
embeddingBus.EmbedAndStore(ctx, entityKind, entityID, updatedContent)  // Store new embeddings
```
This allows users to provide a corrected content and have the embeddings regenerated without re-classifying the entity type.

### Cleanup

When an entity is deleted (task, note, event, context), call:
```go
embeddingBus.DeleteBySource(ctx, "task", taskID)
```

---

## MCP Tool: search_semantic

```json
{
  "name": "search_semantic",
  "description": "Search across all your data using natural language.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "query":        {"type": "string",  "description": "Natural language search query"},
      "source_types": {"type": "array",   "items": {"type": "string"}, "description": "Limit to: email, task, note, event, context, voice"},
      "limit":        {"type": "integer", "description": "Max results (default 10, max 25)"}
    },
    "required": ["query"]
  }
}
```

**Implementation:** `app/domain/mcpapp/mcpapp.go` → `toolSearchSemantic()` handler.
Unmarshal params → call `embeddingBus.Search()` → return ranked results.

---

## Configuration

| Env Var | Default | Notes |
|---------|---------|-------|
| `PLANNER_OLLAMA_ENABLED` | `true` | If false, embedder is nil and embedding operations are no-ops |
| `PLANNER_OLLAMA_URL` | `http://localhost:11434` | Ollama endpoint |

**Model:** `qwen3-embedding:0.6b` (1024 dimensions, configurable via `PLANNER_OLLAMA_EMBED_MODEL`).  
**Pull it:** `make ollama-pull-embed` or `docker exec planner-ollama ollama pull qwen3-embedding:0.6b`.

**Backfill:** Use `admin backfill-embeddings` to generate embeddings for entities created before v1.29. Uses Ollama locally (configurable via `PLANNER_OLLAMA_URL`, default `http://localhost:11434`).

---

## Migration

**Version:** 1.29  
**Description:** Add pgvector extension and embeddings table

```sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE embeddings (
    embedding_id  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    source_type   TEXT        NOT NULL,
    source_id     UUID        NOT NULL,
    content       TEXT        NOT NULL,
    embedding     vector(1024) NOT NULL,
    created_at    TIMESTAMP   NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_embeddings_source ON embeddings(source_type, source_id);
CREATE INDEX idx_embeddings_vector ON embeddings USING hnsw (embedding vector_cosine_ops);
```

**Docker:** Switch Postgres to pgvector-enabled image:
```yaml
db:
  image: pgvector/pgvector:pg17  # replaces postgres:16
```

---

## Cross-Domain Dependencies

- **ingestbus** → Phase 4: calls `embeddingBus.DeleteBySource()` then `EmbedAndStore()` in skip_classify path for embedding regeneration
- **ingestbus** → calls `embeddingBus.EmbedAndStore()` after extraction (standard path)
- **noteapp** → fires async embedding on create/update
- **taskapp** → fires async embedding on create/update
- **eventapp** → fires async embedding on create/update
- **mcpapp** → provides `search_semantic` MCP tool
- **taskbus/notebus/eventbus** → need to call embedding cleanup on delete

---

## Cascade Rules

### Creating a new indexed entity

1. Entity created (note, task, event, email via ingest)
2. App handler fires async goroutine: `EmbedAndStore(type, id, content)`
3. Embedder generates vector (1024-dim)
4. Store saves to `embeddings` table

### Updating indexed entity (Phase 4: skip_classify path)

1. User provides corrected content via raw_input with `skip_classify=true` and `source_entity_id` set
2. ingestbus calls `DeleteBySource(type, id)` to remove old embeddings
3. ingestbus calls `EmbedAndStore(type, id, correctedContent)` to generate new embedding
4. New vector is stored; old one is gone

### Deleting an indexed entity

1. Entity deleted (note, task, event)
2. App handler or domain cleanup calls `DeleteBySource(type, id)`
3. All rows with matching source_type/source_id are deleted

### Sensitivity routing for transactions

Transactions only embed locally (OllamaEmbedder, never Claude API):
- Use `ingestbus` with `WithEmbedder()` for local-only embeddings
- Or call `OllamaEmbedder` directly if needed

---

## Testing

**Store layer** (`embeddingdb_test.go`): Use `dbtest` for real Postgres.
- Test Create, SearchByVector, DeleteBySource with fixture embeddings
- Assert cosine similarity ranking
- Verify ivfflat index works (large vector search is performant)

**Business layer** (`embeddingbus_test.go`): Mock Storer.
- Test EmbedAndStore calls Storer.Create
- Test Search calls Storer.SearchByVector
- Test nil embedder handling (graceful no-op)

**Foundation layer** (`embed/ollama_test.go`): Mock `/api/embed` endpoint.
- Verify HTTP POST format and dimensions
- Test timeout, non-200 responses

**Integration** (`apitest`): Test `search_semantic` MCP tool.
- Create entities → embed → search by query
- Assert results ranked by similarity

---

## Future Enhancements

1. **Hybrid search:** Combine semantic + SQL filters (e.g. "urgent tasks from last week")
2. **Reranking:** Second-pass scoring using Claude API for high-precision results
3. **Context summary rewrite:** Auto-update context summaries when new semantic content emerges
4. **Multi-modal embeddings:** If Ollama adds image embedding support
5. **Embedding streaming:** For large documents (currently embeds whole content at once)
6. **Vector clustering:** Find "islands" of related embeddings to surface patterns
