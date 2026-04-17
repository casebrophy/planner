# Knowledge Gap Detection — Routing Fix

**Date:** 2026-04-17
**Status:** planned

## Problem

Backend log:
```
ERROR "knowledge gap detection failed" entity_kind=task
error="ollama: do request: Post \"http://ollama:11434/api/generate\": context deadline exceeded"
```

Gap detection is routed to Ollama (`qwen3.5:0.8b`), whose `/api/generate` call exceeds the 180s per-request timeout in `foundation/ollamaclient`. The goroutine at `business/domain/ingestbus/ingestbus.go:640` logs the error and moves on, so ingest still succeeds — but gap detection silently produces nothing.

## Root cause

1. `business/domain/ingestbus/extractor/router.go:59-64` prefers `localOnly` (Ollama) for `AnalyzeGaps`. Ollama is too slow for this reasoning task at our hardware tier; a single `/api/generate` call exceeds 180s.
2. `foundation/ollamaclient/client.go:86-111` caps per-request execution at 180s (post-dequeue — queue wait is unbounded). Raising the cap would just let bad calls hang longer; the cap is working as a circuit breaker.

## Secondary bug (same path)

`api/services/planner/main.go:640-648` — the `extractorGapAdapter.AnalyzeGaps` builds `extractor.RelatedEntity` values but only copies `ID` and `SourceType`. `Title` and `Content` stay empty. The prompt at `business/domain/ingestbus/extractor/prompt.go:329` then renders:

```
1. [TASK] id=<uuid> title=""
   <empty>
```

…for every related entity. The model receives a nearly useless prompt. This predates the current bug but becomes more expensive once we route to Claude (paid tokens for no signal).

`embeddingbus.SearchResult` embeds `Embedding`, which already carries `Content` — no store lookup required to fix.

## Design choice: why not sanitize and route everything to Claude?

The `business/sdk/sanitize` package is regex-only (SSN, phone, credit card, routing, account number). It does not redact merchant names, amounts, or memo text, so a task like *"Pay $500 to Chase Sapphire"* would pass through unredacted. Since the user's `financial-data-privacy` preference is that Claude should not see bank/transaction data, regex sanitize is not a sufficient boundary for transactions. A categorical source-based filter is.

LLM-based sanitize (use Ollama to redact before sending to Claude) was considered and rejected: it doubles latency on the same FIFO queue we're escaping, introduces a probabilistic trust layer, and adds failure modes — not worth it as a primary defense.

## Fix (Option B)

1. **Skip gap detection when `ri.SourceType == rawinputsource.Transaction`** at the three call sites in `ingestbus.go` (498, 640, 1115). Financial content never leaves the local side.
2. **Flip `router.AnalyzeGaps` to prefer `general` (Claude)** over `localOnly`. Non-financial gap analysis gets better reasoning and avoids the Ollama bottleneck.
3. **Sanitize `entityContent` at the adapter boundary** (`main.go:640`) via `sanitize.Sanitize(...).Text`. Belt-and-suspenders for incidental PII in non-financial entities.
4. **Fix empty related-entity content** — add `Content` to `knowledgegapbus.RelatedEntitySummary`, populate from `SearchResult.Content` in `knowledgegapbus.Detect`, thread into `extractor.RelatedEntity.Content` in the adapter.
5. **No timeout change.** The 180s cap stays as a circuit breaker. Once gap analysis no longer uses Ollama, the remaining Ollama workload (embeds, transaction text extraction) is well under 180s.

## Files touched

- `business/domain/ingestbus/ingestbus.go` — 3 call sites, add Transaction-source guard
- `business/domain/ingestbus/extractor/router.go` — flip `AnalyzeGaps` preference
- `api/services/planner/main.go` — adapter: sanitize + thread content
- `business/domain/knowledgegapbus/model.go` — add `Content` field to `RelatedEntitySummary`
- `business/domain/knowledgegapbus/knowledgegapbus.go` — populate `Content` from `SearchResult`
- `business/domain/knowledgegapbus/knowledgegapbus_test.go` — adapter content passthrough
- `business/domain/ingestbus/extractor/router_test.go` (or new) — `AnalyzeGaps` → general
- `business/domain/ingestbus/ingestbus_test.go` — Transaction source skips gap detection
- `.docs/arch/ingest-backend.md` — refresh routing description

## Out of scope

- Changing `ollamaclient` timeout semantics
- Making gap-analyzer timeout configurable per-call
- LLM-based sanitization layer
- Fetching full titles from taskBus/noteBus/eventBus (Content from the embedding is sufficient; Title can stay empty or fall back to first line of Content)
