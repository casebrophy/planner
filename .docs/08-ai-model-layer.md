# AI model layer

The AI model layer makes the system model-agnostic, routing inference, embedding, and sanitization requests to the right backend (Anthropic API or local Ollama) based on sensitivity tier and task type.

---

## Core interfaces

```go
// Inferencer generates text from a prompt.
// Used for extraction, classification, summarization, and context reasoning.
type Inferencer interface {
    // Infer sends a prompt and returns the model's response.
    // systemPrompt may be empty. temperature is 0.0–1.0.
    Infer(ctx context.Context, req InferRequest) (InferResponse, error)
}

type InferRequest struct {
    SystemPrompt string
    UserPrompt   string
    Temperature  float32   // default 0.0 for extraction, 0.3 for summarization
    MaxTokens    int       // default 1000
    Format       string    // "json" enforces structured output where supported
}

type InferResponse struct {
    Content    string
    TokensUsed int
    Model      string    // which model actually handled it (for logging)
    Latency    time.Duration
}

// Embedder converts text to a fixed-dimension vector.
// Used for semantic search indexing and query embedding.
type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    Dimensions() int    // vector size — must match the index
}

// ModelRouter selects the appropriate Inferencer and Embedder
// for a given sensitivity tier and task type.
type ModelRouter interface {
    InferencerFor(tier SensitivityTier, task TaskType) Inferencer
    EmbedderFor(tier SensitivityTier) Embedder
}

// TaskType guides model selection beyond just tier —
// some tasks benefit from a larger/more capable model
// even within the same tier.
type TaskType string

const (
    TaskExtraction    TaskType = "extraction"    // structured JSON from raw content
    TaskClassification TaskType = "classification" // routing decisions
    TaskSummarization TaskType = "summarization"  // context summary rewrites
    TaskReasoning     TaskType = "reasoning"      // planning, cross-context queries
    TaskSanitization  TaskType = "sanitization"   // PII removal (always local)
)
```

---

## Concrete implementations

```go
type AnthropicInferencer struct {
    APIKey  string
    Model   string    // e.g. "claude-sonnet-4-20250514"
    client  *http.Client
}

type OllamaInferencer struct {
    BaseURL string    // default: http://ollama:11434
    Model   string    // e.g. "llama3.2", "mistral", "phi4-mini"
    client  *http.Client
}

type OllamaEmbedder struct {
    BaseURL string
    Model   string    // default: "nomic-embed-text"
    dims    int       // set on first call, validated against stored embeddings
}
```

---

## Model router

```go
type DefaultRouter struct {
    apiInferencer   Inferencer  // Tier 1 + promoted Tier 2
    apiEmbedder     Embedder
    localInferencer Inferencer  // Tier 2 sanitization + Tier 3
    localEmbedder   Embedder
}

func (r *DefaultRouter) InferencerFor(tier SensitivityTier, task TaskType) Inferencer {
    switch {
    case tier == Tier3:
        return r.localInferencer
    case task == TaskSanitization:
        return r.localInferencer    // sanitization is ALWAYS local, regardless of tier
    case tier == Tier2 && task == TaskExtraction:
        return r.localInferencer    // Tier 2 extraction is local before promotion
    default:
        return r.apiInferencer
    }
}

func (r *DefaultRouter) EmbedderFor(tier SensitivityTier) Embedder {
    if tier == Tier3 {
        return r.localEmbedder
    }
    return r.localEmbedder    // default to local embeddings for all tiers
    // swap to r.apiEmbedder for higher quality Tier 1 embeddings if desired
}
```

Hard rule: `TaskSanitization` always routes to the local inferencer — not configurable.

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

Each chunk carries: `ID`, `SourceType` ("email"|"context_event"|"task_note"|"task"|"voice"), `SourceID` (FK), `Content`, `Tier`, `CreatedAt`.

### Vector storage DDL

```sql
CREATE VIRTUAL TABLE embeddings USING vec0(
    id          TEXT PRIMARY KEY,
    source_type TEXT,
    source_id   TEXT,
    content     TEXT,
    tier        INTEGER,
    created_at  TEXT,
    embedding   FLOAT[768]    -- dimensions must match the embeddings model
);
```

Similarity search: `SELECT id, source_type, source_id, content, vec_distance_cosine(embedding, ?) AS distance FROM embeddings WHERE tier <= ? ORDER BY distance LIMIT 20;`

### Retrieval pipeline

1. Embed query using `EmbedderFor(Tier1)` (queries always Tier 1)
2. Vector similarity search with tier ceiling (never return Tier 3 to API Claude)
3. Post-filters: date range, source type, context ID
4. Re-rank with heuristic scorer
5. Fetch full records for top N; return to Claude

Re-ranking formula:
```
final_score = cosine_similarity × recency_boost(created_at) × status_penalty(status) × context_boost(context_id)
```
Recency boost: last 7 days = 1.3×, older than 90 days = 0.7×.

### MCP tool: `search_semantic`

```json
{
  "name": "search_semantic",
  "description": "Search across all your data using natural language. Use for questions about commitments, past decisions, information you've captured, or anything that requires understanding meaning rather than matching keywords. Combine with date filters for time-bounded queries.",
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
ANTHROPIC_API_KEY=required
ANTHROPIC_MODEL=claude-sonnet-4-20250514

OLLAMA_BASE_URL=http://ollama:11434
OLLAMA_EXTRACTION_MODEL=llama3.2
OLLAMA_SANITIZATION_MODEL=llama3.2
OLLAMA_REASONING_MODEL=llama3.2

OLLAMA_EMBEDDING_MODEL=nomic-embed-text
EMBEDDING_DIMENSIONS=768    # must match model; changing requires re-indexing all embeddings
```

Changing `EMBEDDING_DIMENSIONS` requires dropping and rebuilding the `embeddings` virtual table.
