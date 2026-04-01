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

## Extractors

### Email extraction

`business/domain/ingestbus/extractor/claudecli.go` implements the `Extractor` interface.

Called by the ingestion pipeline (`ingestbus.processRawInput()` step 6) to extract structured data from emails: summary, action items, deadlines, sentiment, context matching.

### Thread entry extraction

`business/domain/threadbus/claudecli_extractor.go` implements the `threadbus.Extractor` interface.

Called by `threadbus.AddEntry()` when `Extract` flag is set. Classifies thread entries into kinds (update, blocker, decision, etc.) with sentiment and blocking party detection.

### Shared prompt

`business/domain/ingestbus/extractor/prompt.go` contains `BuildEmailExtractionPrompt()` — the shared prompt template used by all email extractor implementations.

### Mock extractor

`business/domain/ingestbus/extractor/mock.go` — used in tests. Returns configured result/error.

---

## Embedder (future — not yet implemented)

```go
type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    Dimensions() int
}
```

Options for future implementation:
- Local Ollama embeddings (e.g. nomic-embed-text, 768 dimensions)
- Or extend Claude CLI client if embedding support is added

---

## RAG: semantic search (future)

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
