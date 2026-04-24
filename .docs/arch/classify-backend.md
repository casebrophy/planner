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

### Classifier Logic — `business/domain/ingestbus/classify/classifier.go`

```go
// ItemType represents the classification of an item.
type ItemType string

const (
	TaskType  ItemType = "task"
	EventType ItemType = "event"
	NoteType  ItemType = "note"
)

// Classification holds the result of classifying a clause.
type Classification struct {
	Type       ItemType
	Confidence float64 // 0.0 - 1.0
}

// Classify(clause string) Classification — heuristic classification using:
// - obligationVerbs: task indicators ("need to", "should", "must", "book", "get rid of", etc.)
// - timePatterns: event indicators (dates, times, day names, durations)
// - reference patterns: note indicators (phone, email, address, facts)
```

---

## File Map

### Handlers — `app/domain/classifyapp/`

- **classifyapp.go** — HTTP handlers:
  - `classify(ctx, r)` — Routes to entity-specific classifier based on `?entity_type` param (default: "task")
  - `classifyTasks(ctx)` — Queries open unlinked tasks, spawns goroutine for classification
  - `classifyNotes(ctx)` — Queries unlinked notes, spawns goroutine for classification
  - `classifyEvents(ctx)` — Queries unlinked events, spawns goroutine for classification
  - `fetchContextRefs(ctx)` — Shared helper: queries active contexts, builds []extractor.ContextRef with ID, Title, and Kind fields
  - `classifyEntity(ctx, entityType, entityID, text, ctxRefs)` — Shared goroutine worker: calls extractor.ExtractText with full signature (typeHint, typeHintConfidence, candidates, contextAnnotations), routes high/low confidence outcomes
- **model.go** — Response DTO (ClassifyAccepted with Encode())
- **route.go** — Routes.Add() — wires taskbus, notebus, eventbus, contextbus, clarificationbus, extractor (Claude or Ollama failover); registers routes with auth middleware

### Classification Logic (goroutine, per entity)

**In app/domain/classifyapp/classifyapp.go:**

For each unlinked entity, `classifyEntity` goroutine:
1. Calls `extractor.ExtractText(bgCtx, "<EntityType>: <title/content>\nDescription: <desc>", "", ctxRefs, "", 0, nil, nil)` — no typeHint, typeHintConfidence, candidates, or contextAnnotations
2. Skips if no suggested context; logs via `a.log.Error` on extractor failure or unparseable UUID
3. Verifies suggested context exists via `contextBus.QueryByID`; logs on failure
4. If confidence >= 0.7: directly updates entity ContextID via `taskBus.Update` / `noteBus.Update` / `eventBus.Update`; logs on entity lookup failure
5. If confidence < 0.7: creates a clarification card via `clarificationBus.Upsert` with kind=context_assignment, the suggested context, confidence score, and available contexts as answer options

**In business/domain/ingestbus/ingestbus.go:**

As part of the ingestion pipeline (processing raw voice input or email), each clause is:
1. Classified via `classify.Classify(clause)` — returns ItemType (task/event/note) + confidence [0.0-1.0] using heuristic patterns
2. Used as a typeHint to the extractor: `extractor.ExtractText(..., typeHint, typeHintConfidence, candidates, contextAnnotations)`
3. The heuristic hints guide the LLM but can be overridden via the TextExtraction.ReclassifiedAs field

---

## Heuristic Classification Details

The classifier uses regex patterns and keyword lists to assign task/event/note type hints before LLM extraction:

**Obligation Verbs** (task): "need to", "have to", "should", "must", "book", "buy", "send", "finish", "fix", "get rid of", "clean out", "clean up", "throw away", etc.

**Time Patterns** (event): Day names, months, relative dates ("tomorrow", "next week"), times ("2pm", "at 3:30"), time expressions ("in 2 days", "morning")

**Reference Patterns** (note): Phone numbers, email addresses, street addresses, fact statements ("X is Y"), high-confidence notes

**Confidence Scoring:**
- Pure reference (phone/email/address) → Note at 0.95
- Obligation without temporal → Task at 0.9
- Temporal without obligation → Event at 0.9
- Both obligation + temporal → Task at 0.6 (ambiguous)
- Fact + obligation → Task at 0.65
- Fact + temporal → Event at 0.65
- No matches → Note at 0.5

These hints are passed as `typeHint` + `typeHintConfidence` to the extractor, which can override them via `TextExtraction.ReclassifiedAs`.

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
- **contextbus** (`business/domain/contextbus`) — Queries active contexts (status=active, page 1-50) to build ContextRef list (with Kind field) for extractor. Also verifies suggested context exists via QueryByID.
- **clarificationbus** (`business/domain/clarificationbus`) — Creates clarification items (kind=context_assignment) for low-confidence matches via Upsert.
- **ingestbus/extractor** (`business/domain/ingestbus/extractor`) — ExtractText call with full signature: sends entity text, context refs, typeHint, typeHintConfidence, candidates, contextAnnotations. Returns TextExtraction with SuggestedContextID + ContextConfidence.
- **ingestbus/classify** (`business/domain/ingestbus/classify`) — Heuristic classifier embedded in ingestbus pipeline; used by ingestion to generate typeHint before extraction.

### Inbound (Domains that depend on this)

- **ingestbus** (`business/domain/ingestbus`) — Uses `classify.Classify(clause)` to generate typeHint + typeHintConfidence for each clause extracted from voice/email, fed into ExtractText call.
- **mcpapp** (`app/domain/mcpapp`) — Exposes "classify_tasks" MCP tool, routes to POST /api/v1/classify?entity_type=task
- **reingestapp** (`app/domain/reingestapp`) — Respects skip_classify flag to bypass classification on entity re-ingestion.

### Related Components

- **app/sdk/errs** — Error codes for synchronous validation phase (Internal errors from DB queries, InvalidArgument for unknown entity_type).
- **app/sdk/mid** — Auth middleware (APIKey) applied to the routes.
- **foundation/web** — HandlerFunc + Encoder pattern.
- **foundation/logger** — `*logger.Logger` stored on `app` struct; used to log errors from background goroutine in `classifyEntity`.
