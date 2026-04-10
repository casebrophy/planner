# AI Model Layer — Sensitivity-Tier Routing + Transaction Enrichment

**Date:** 2026-04-10
**Phases affected:** 5 (transaction AI enrichment), cross-cutting (model routing)
**Motivation:** User is not comfortable with Claude seeing bank data. Financial data must use local Ollama inference exclusively. Currently Ollama exists only as an error-fallback; needs policy-based routing.

---

## Current State

- `OllamaExtractor` exists (`extractor/ollama.go`) — fully functional, calls `/api/generate`
- `FailoverExtractor` exists (`extractor/failover.go`) — Claude primary, Ollama on error only
- `main.go:201` creates only `ClaudeCodeExtractor`, ignoring Ollama config entirely
- `main.go:204` computes `ollamaEnabled` but only passes it to muxCfg (unused by extractor)
- Ollama config parsed (`main.go:117-121`) — `PLANNER_OLLAMA_URL`, `PLANNER_OLLAMA_MODEL`, `PLANNER_OLLAMA_ENABLED`
- Transaction domain is CRUD-only (CSV import) — no AI enrichment path
- `rawinputsource.Transaction` exists but no extraction pipeline uses it
- No Docker Compose service for Ollama

## Design Decisions

1. **TieredRouter** — new `Extractor` implementation that routes by source type. Transaction source → Ollama only (never Claude). Everything else → FailoverExtractor (Claude primary, Ollama fallback on error). Composable: `TieredRouter(general=FailoverExtractor, financial=OllamaExtractor)`.

2. **No new Extractor interface methods** — transaction enrichment uses `ExtractText()` with `typeHint="transaction"` and a transaction-specific prompt. The OllamaExtractor already accepts typeHint (currently ignores it); we add transaction prompt support.

3. **Transaction enrichment is opt-in** — when `PLANNER_OLLAMA_ENABLED=true`, CSV import triggers async enrichment (CleanName, Category, ContextID suggestion). When disabled, transactions store as-is (current behavior). Enrichment failure logs + skips, never blocks.

4. **Ollama Docker service** — added to compose, internal network only (not exposed on host in production). Dev uses `localhost:11434`.

5. **Embedder is out of scope** — that's Phase 6 (RAG). This plan covers sensitivity routing + transaction enrichment only.

---

## Task 1: Wire TieredRouter extractor in main.go

**What:** Create `TieredRouter` and wire it as the primary extractor, replacing the bare `ClaudeCodeExtractor`.

**Files:**

### CREATE `business/domain/ingestbus/extractor/router.go`

```go
// TieredRouter routes extraction calls based on sensitivity tier.
// Financial data (transactions) → localOnly extractor (Ollama).
// Everything else → general extractor (FailoverExtractor or Claude).
type TieredRouter struct {
    log       *logger.Logger
    general   Extractor  // Claude (with Ollama failover)
    localOnly Extractor  // Ollama only — for sensitive data
}

func NewTieredRouter(log *logger.Logger, general Extractor, localOnly Extractor) *TieredRouter

// ExtractEmail always routes to general (emails are not sensitive in this system).
func (r *TieredRouter) ExtractEmail(...) (EmailExtraction, error)

// ExtractText routes based on typeHint:
//   typeHint == "transaction" → localOnly
//   everything else → general
func (r *TieredRouter) ExtractText(...) (TextExtraction, error)
```

The routing key is `typeHint` on `ExtractText`. This avoids changing the Extractor interface.

### CREATE `business/domain/ingestbus/extractor/router_test.go`

Test that:
- `ExtractText` with `typeHint="transaction"` calls localOnly mock, never general mock
- `ExtractText` with any other typeHint calls general mock
- `ExtractEmail` always calls general mock
- When localOnly is nil (Ollama disabled), transaction calls return a sentinel "enrichment skipped" (not an error)

### MODIFY `api/services/planner/main.go` (lines 198-202)

Replace:
```go
ext := extractor.NewClaudeCodeExtractor(cli)
```

With:
```go
claudeExt := extractor.NewClaudeCodeExtractor(cli)

var ext extractor.Extractor
if ollamaEnabled {
    ollamaExt := extractor.NewOllamaExtractor(cfg.Ollama.URL, cfg.Ollama.Model)
    failover := extractor.NewFailoverExtractor(log, claudeExt, ollamaExt)
    ext = extractor.NewTieredRouter(log, failover, ollamaExt)
} else {
    ext = claudeExt
}
```

This gives us: emails/voice → FailoverExtractor (Claude → Ollama on error), transactions → OllamaExtractor directly.

---

## Task 2: Transaction-specific extraction prompt

**What:** Add a transaction extraction prompt to OllamaExtractor so `ExtractText(typeHint="transaction")` returns useful enrichment.

**Files:**

### MODIFY `business/domain/ingestbus/extractor/prompt.go`

Add `BuildTransactionExtractionPrompt(description string, amount float64, activeContexts []byte) string`. The prompt asks the model to:
- Clean up the merchant name (strip card numbers, transaction codes)
- Suggest a spending category (groceries, dining, transport, utilities, etc.)
- Suggest a context ID from active contexts if relevant
- Return JSON: `{"clean_name": "...", "category": "...", "suggested_context_id": "...", "context_confidence": 0.0}`

### MODIFY `business/domain/ingestbus/extractor/ollama.go`

In `ExtractText()`, check `typeHint == "transaction"`. If so, use the transaction-specific prompt instead of the generic text prompt. Parse result into `TextExtraction` with the enrichment data mapped to the existing fields (Summary = clean_name, SuggestedContextKeywords = [category], SuggestedContextID, ContextConfidence).

### CREATE `business/domain/ingestbus/extractor/ollama_test.go`

Test transaction prompt generation and response parsing (mock HTTP server).

---

## Task 3: Transaction enrichment pipeline

**What:** After CSV import creates transactions, trigger async Ollama enrichment for each transaction that lacks a CleanName or Category.

**Files:**

### MODIFY `business/domain/transactionbus/transactionbus.go`

Add an optional `Enricher` field (set via `WithEnricher()` functional option pattern, same as `threadbus.WithExtractor()`):

```go
type Enricher interface {
    EnrichTransaction(ctx context.Context, description string, amount float64, activeContexts []extractor.ContextRef) (TransactionEnrichment, error)
}

type TransactionEnrichment struct {
    CleanName          string
    Category           string
    SuggestedContextID *uuid.UUID
    ContextConfidence  float64
}
```

On `CreateBatch()` (the CSV import path), if enricher is set, fire a goroutine that enriches each transaction and calls `Update()` with the results. Log + skip on failure.

### CREATE `business/domain/transactionbus/enricher.go`

Adapter that wraps `extractor.Extractor` and implements `Enricher`:
- Calls `ExtractText(ctx, description, activeContexts, "transaction")`
- Maps TextExtraction fields → TransactionEnrichment
- Queries active contexts (needs `contextbus.Business` or receives `[]ContextRef` as param)

### MODIFY `api/services/planner/main.go`

Wire the enricher into transactionbus when Ollama is enabled:
```go
if ollamaEnabled {
    txEnricher := transactionbus.NewExtractorEnricher(ext, ctxBus)
    txBus = txBus.WithEnricher(txEnricher)
}
```

---

## Task 4: Ollama Docker Compose service

**What:** Add Ollama to the dev Docker Compose so `make dev-up` brings it up automatically.

**Files:**

### MODIFY `zarf/compose/docker-compose.yml`

Add service:
```yaml
ollama:
  image: ollama/ollama:latest
  ports:
    - "11434:11434"
  volumes:
    - ollama-data:/root/.ollama
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:11434/api/tags"]
    interval: 30s
    timeout: 10s
    retries: 3

volumes:
  ollama-data:
```

### MODIFY `Makefile`

Add `ollama-pull` target: `docker exec -it planner-ollama ollama pull llama3` (one-time model download after first `make dev-up`).

---

## Task 5: Update docs + roadmap

**What:** Update arch docs and roadmap to reflect the new sensitivity-tier routing.

**Files:**
- MODIFY `.docs/arch/ingest-backend.md` — add TieredRouter to extractor section
- MODIFY `.docs/08-ai-model-layer.md` — document sensitivity decision, remove Ollama as "future"
- MODIFY `.docs/07-roadmap.md` — update Phase 5 status

---

## Build Sequence

```
Task 1 (TieredRouter + wiring)
  ↓
Task 2 (transaction prompt)  ←── can start after Task 1 merges
  ↓
Task 3 (enrichment pipeline) ←── depends on Task 2
  
Task 4 (Docker Compose)      ←── independent, can parallel with Task 1
Task 5 (docs)                ←── after Task 3
```

## Verification

After all tasks:
1. `make test` passes
2. `make dev-up` starts Ollama container + pulls model
3. CSV import triggers enrichment (check logs for Ollama calls)
4. Non-transaction ingestion still routes through Claude (verify with email/voice test)
5. With `PLANNER_OLLAMA_ENABLED=false`, everything works as before (no regression)
