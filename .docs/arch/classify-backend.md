# Classify Backend

> The classify domain provides a single endpoint that asynchronously assigns open, unlinked tasks to active contexts using AI (Claude or Ollama fallback). High-confidence matches are applied directly; low-confidence matches create clarification cards for user review. The handler returns immediately after enqueuing work — the LLM calls run in a background goroutine.

---

## Core Types

### App Layer — `app/domain/classifyapp/model.go`

```go
// ClassifyAccepted is returned immediately when classification is enqueued.
type ClassifyAccepted struct {
    Message       string `json:"message"`
    UnlinkedCount int    `json:"unlinkedCount"`
}
```

---

## File Map

### Handlers — `app/domain/classifyapp/`

- **classifyapp.go** — One HTTP handler:
  - `classify(ctx, r)` — POST /api/v1/tasks/classify (async: queries open unlinked tasks + active contexts synchronously, spawns goroutine for LLM classification + DB writes, returns ClassifyAccepted immediately)
- **model.go** — Response DTO (ClassifyAccepted with Encode())
- **route.go** — Routes.Add() — wires taskbus, contextbus, clarificationbus, extractor (Claude or Ollama failover); registers route with auth middleware

### Classification Logic (goroutine)

For each unlinked task, the goroutine:
1. Calls `extractor.ExtractText(bgCtx, "Task: <title>\nDescription: <desc>", ctxRefs)`
2. Skips if no suggested context or unparseable UUID
3. Verifies suggested context exists via `contextBus.QueryByID`
4. If confidence >= 0.7: directly updates task ContextID via `taskBus.Update`
5. If confidence < 0.7: creates a clarification card via `clarificationBus.Create` with kind=context_assignment, the suggested context, confidence score, and available contexts as answer options

---

## Impact Callouts

### Async Pattern

- DB queries for tasks and contexts happen synchronously (before goroutine) to catch errors early and return fast accepted response
- LLM calls (expensive, slow) and subsequent DB writes happen in `go func()` with `context.Background()`
- No result channel — errors in the goroutine are silently skipped (same pattern as MCP tool handlers)
- Caller should poll or use polling composables to observe classification effects

### Extractor Failover

- Configured in route.go: if `OllamaEnabled && OllamaURL != ""`, wraps ClaudeCodeExtractor with OllamaExtractor as failover
- Failover is transparent to the classify handler

---

## Routes

| Method | Path | Handler | Auth | Purpose |
|--------|------|---------|------|---------|
| POST | /api/v1/tasks/classify | classify | APIKey | Enqueue async classification of all open unlinked tasks. Returns ClassifyAccepted{message, unlinkedCount} immediately. |

---

## Cross-Domain Dependencies

### Outbound (Domains this depends on)

- **taskbus** (`business/domain/taskbus`) — Queries open tasks (status=open, page 1-200), filters for nil ContextID in Go. Updates task ContextID on high-confidence match.
- **contextbus** (`business/domain/contextbus`) — Queries active contexts (status=active, page 1-50) to build ContextRef list for extractor. Also verifies suggested context exists via QueryByID.
- **clarificationbus** (`business/domain/clarificationbus`) — Creates clarification items (kind=context_assignment) for low-confidence matches.
- **ingestbus/extractor** (`business/domain/ingestbus/extractor`) — ExtractText call: sends task text + context refs, returns SuggestedContextID + ContextConfidence.

### Related Components

- **app/sdk/errs** — Error codes for synchronous validation phase (Internal errors from DB queries).
- **app/sdk/mid** — Auth middleware (APIKey) applied to the route.
- **foundation/web** — HandlerFunc + Encoder pattern.
