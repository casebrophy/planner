# Classify Backend

> The classify domain provides endpoints that asynchronously assign unlinked tasks, notes, and events to active contexts using AI (Claude or Ollama fallback). High-confidence matches are applied directly; low-confidence matches create clarification cards for user review. The handler returns immediately after enqueuing work — the LLM calls run in a background goroutine. Entity type is selected via `?entity_type=task|note|event` query param (defaults to "task" for backward compatibility).

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

- **classifyapp.go** — HTTP handlers:
  - `classify(ctx, r)` — Routes to entity-specific classifier based on `?entity_type` param (default: "task")
  - `classifyTasks(ctx)` — Queries open unlinked tasks, spawns goroutine for classification
  - `classifyNotes(ctx)` — Queries unlinked notes, spawns goroutine for classification
  - `classifyEvents(ctx)` — Queries unlinked events, spawns goroutine for classification
  - `fetchContextRefs(ctx)` — Shared helper: queries active contexts, builds []extractor.ContextRef
  - `classifyEntity(ctx, entityType, entityID, text, ctxRefs)` — Shared goroutine worker: calls extractor, routes high/low confidence outcomes
- **model.go** — Response DTO (ClassifyAccepted with Encode())
- **route.go** — Routes.Add() — wires taskbus, notebus, eventbus, contextbus, clarificationbus, extractor (Claude or Ollama failover); registers routes with auth middleware

### Classification Logic (goroutine, per entity)

For each unlinked entity, `classifyEntity` goroutine:
1. Calls `extractor.ExtractText(bgCtx, "<EntityType>: <title/content>\nDescription: <desc>", ctxRefs)`
2. Skips if no suggested context; logs via `a.log.Error` on extractor failure or unparseable UUID
3. Verifies suggested context exists via `contextBus.QueryByID`; logs on failure
4. If confidence >= 0.7: directly updates entity ContextID via `taskBus.Update` / `noteBus.Update` / `eventBus.Update`; logs on entity lookup failure
5. If confidence < 0.7: creates a clarification card via `clarificationBus.Create` with kind=context_assignment, the suggested context, confidence score, and available contexts as answer options

---

## Impact Callouts

### Async Pattern

- DB queries for entities and contexts happen synchronously (before goroutine) to catch errors early and return fast accepted response
- LLM calls (expensive, slow) and subsequent DB writes happen in `go func()` with `context.Background()`
- No result channel — errors in the goroutine are logged via `a.log.Error` and then skipped (extractor failure, UUID parse failure, context/entity lookup failure)
- Caller should poll or use polling composables to observe classification effects

### Extractor Failover

- Configured in route.go: if `OllamaEnabled && OllamaClient != nil`, wraps ClaudeCodeExtractor with OllamaExtractor as failover
- OllamaExtractor is constructed from the shared `*ollamaclient.Client` on `mux.Config` so all Ollama calls (extract + embed) share one FIFO-serialized queue
- Failover is transparent to the classify handler

---

## Routes

| Method | Path | Handler | Auth | Purpose |
|--------|------|---------|------|---------|
| POST | /api/v1/tasks/classify | classify | APIKey | Backward-compatible: classify open unlinked tasks (entity_type defaults to "task") |
| POST | /api/v1/classify | classify | APIKey | Unified: classify entities by type — `?entity_type=task\|note\|event`. Returns ClassifyAccepted{message, unlinkedCount} immediately. |

---

## Cross-Domain Dependencies

### Outbound (Domains this depends on)

- **taskbus** (`business/domain/taskbus`) — Queries open tasks (status=open, page 1-200), filters for nil ContextID in Go. Updates task ContextID on high-confidence match.
- **notebus** (`business/domain/notebus`) — Queries notes (page 1-200), filters for nil ContextID in Go. Updates note ContextID on high-confidence match.
- **eventbus** (`business/domain/eventbus`) — Queries events (page 1-200), filters for nil ContextID in Go. Updates event ContextID on high-confidence match.
- **contextbus** (`business/domain/contextbus`) — Queries active contexts (status=active, page 1-50) to build ContextRef list for extractor. Also verifies suggested context exists via QueryByID.
- **clarificationbus** (`business/domain/clarificationbus`) — Creates clarification items (kind=context_assignment) for low-confidence matches.
- **ingestbus/extractor** (`business/domain/ingestbus/extractor`) — ExtractText call: sends entity text + context refs, returns SuggestedContextID + ContextConfidence.

### Related Components

- **app/sdk/errs** — Error codes for synchronous validation phase (Internal errors from DB queries, InvalidArgument for unknown entity_type).
- **app/sdk/mid** — Auth middleware (APIKey) applied to the routes.
- **foundation/web** — HandlerFunc + Encoder pattern.
- **foundation/logger** — `*logger.Logger` stored on `app` struct; used to log errors from background goroutine in `classifyEntity`.
