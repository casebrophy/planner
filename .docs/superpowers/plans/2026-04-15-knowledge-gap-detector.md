# Knowledge Gap Detector — Implementation Plan

**Goal:** When entities (tasks, events, notes) are created, the system runs a semantic search for related knowledge, identifies what's missing, and generates clarification cards asking the user to fill gaps. Answers are stored as linked notes. This is Phase 2 of the "second brain" vision — proactive knowledge capture.

**Architecture:** A new `knowledgegapbus` business domain provides gap detection logic. It's called from async post-creation goroutines in taskapp, noteapp, and eventapp (same pattern as embedding generation). Gap detection: embed entity content → semantic search for related entities → analyze what's present vs. what's missing → generate `knowledge_gap` clarification cards. On resolution, answers are stored as notes linked via entity links.

**Key decisions:**
- No new tables — reuses clarification_items, notes, entity_links, embeddings
- Gap analysis is heuristic + Claude-driven (extractor call to identify missing knowledge)
- Fires async (fire-and-forget goroutine) so entity creation latency is unaffected
- Duplicate prevention via clarificationBus.Count() before creating cards
- Unstructured notes + semantic search for extensibility (not structured entity profiles)

---

## File Map

### Backend — new files

| File | Responsibility |
|------|---------------|
| `business/domain/knowledgegapbus/knowledgegapbus.go` | Core detection logic: Detect(ctx, entityType, entityID, content) — semantic search, gap analysis, card generation |
| `business/domain/knowledgegapbus/model.go` | GapDetectionResult, GapCandidate types; gap category constants (missing_contact, missing_location, missing_context, missing_dependency, missing_detail) |

### Backend — modified files

| File | Change |
|------|--------|
| `business/types/clarificationkind/clarificationkind.go` | Add `KnowledgeGap` constant + kinds map + AllKinds + KindWeights |
| `business/domain/clarificationbus/options.go` | Add `KnowledgeGapOptions` struct (gap_category, related_entity_type, related_entity_id, suggested_question, existing_knowledge_summary) |
| `business/sdk/migrate/sql/migrate.sql` | Version 1.36: add `knowledge_gap` to clarification_kind CHECK constraint |
| `app/domain/taskapp/taskapp.go` | Inject knowledgegapbus; fire `go gapBus.Detect()` in post-creation goroutine |
| `app/domain/noteapp/noteapp.go` | Same — fire gap detection after embedding |
| `app/domain/eventapp/eventapp.go` | Same — fire gap detection after embedding |
| `app/domain/clarificationapp/clarificationapp.go` | Add `KnowledgeGap` case in `dispatchResolution` — create note from answer text + entity link to subject |
| `api/services/planner/main.go` | Instantiate knowledgegapbus.Business, inject into taskapp/noteapp/eventapp Routes |

### Frontend — modified files

| File | Change |
|------|--------|
| `api/services/frontend/web/src/types/enums.ts` | Add `knowledge_gap` to ClarificationKind enum + label ("Knowledge Gap") + color |
| `api/services/frontend/web/src/components/shared/ClarificationCard.vue` | Add `v-if` branch for `knowledge_gap` kind — show gap description, existing knowledge summary, textarea for answer |

---

## Tasks

### Task 1: Enum + migration + options struct

Add the `knowledge_gap` clarification kind and supporting types.

**Files:**
- `business/types/clarificationkind/clarificationkind.go` — add `KnowledgeGap` constant, update kinds map, AllKinds, KindWeights (weight ~0.6, lower than operational cards)
- `business/domain/clarificationbus/options.go` — add `KnowledgeGapOptions` struct
- `business/sdk/migrate/sql/migrate.sql` — version 1.36, update CHECK constraint to include `knowledge_gap`

**Pattern:** Follow existing kinds like `EntityLink` and `EventPrep` for weight/structure.

**Tests:** Verify `clarificationkind.Parse("knowledge_gap")` works; verify migration applies cleanly.

### Task 2: knowledgegapbus domain

Create the core gap detection business logic.

**Files:**
- `business/domain/knowledgegapbus/model.go` — types
- `business/domain/knowledgegapbus/knowledgegapbus.go` — Business struct, Detect() method

**Business.Detect(ctx, entityType, entityID, content) flow:**
1. Call `embeddingBus.Search(ctx, content, relevantSourceTypes, limit=10)` to find related entities
2. If no related entities found with similarity > 0.5, skip (nothing to gap-check against)
3. Build a context summary of what the system already knows (related notes, linked entities)
4. Call extractor to analyze: "Given this new entity and what we already know, what useful information is missing?" — returns gap candidates with categories and suggested questions
5. For each high-value gap (confidence > 0.6), check for duplicate via `clarificationBus.Count()` with subject filter
6. Create clarification card via `clarificationBus.Create()` with KnowledgeGapOptions

**Dependencies:** embeddingbus, clarificationbus, extractor interface (for gap analysis prompt)

**Constructor:** `New(log, clarificationBus, embeddingBus, extractor)`

**Tests:** Unit test with mock embeddingbus and clarificationbus — verify cards generated for gaps, skipped when no related entities, duplicates prevented.

### Task 3: Extractor gap analysis method

Add gap analysis capability to the extractor interface and implementations.

**Files:**
- `business/domain/ingestbus/extractor/model.go` — add `GapAnalysis` response type, `AnalyzeGaps` to Extractor interface
- `business/domain/ingestbus/extractor/ollama.go` — implement AnalyzeGaps (local-only, no PII concern)
- `business/domain/ingestbus/extractor/claudecli.go` — implement AnalyzeGaps
- `business/domain/ingestbus/extractor/failover.go` — implement AnalyzeGaps
- `business/domain/ingestbus/extractor/router.go` — route AnalyzeGaps through TieredRouter (prefer local)
- `business/domain/ingestbus/extractor/mock.go` — add mock

**GapAnalysis response type:**
```go
type GapAnalysis struct {
    Gaps []GapCandidate `json:"gaps"`
}
type GapCandidate struct {
    Category    string  `json:"category"`     // missing_contact, missing_location, missing_detail, etc.
    Question    string  `json:"question"`      // "What is Dr. Smith's phone number?"
    Reasoning   string  `json:"reasoning"`     // "You have an appointment but no contact info stored"
    Confidence  float64 `json:"confidence"`    // 0-1
    RelatedIDs  []string `json:"related_ids"`  // IDs of entities that informed this gap
}
```

**Prompt design:** Given the new entity content + summaries of related entities found via search, identify what information would be useful to know but is currently missing. Focus on actionable gaps (contact info, locations, deadlines, dependencies, context).

### Task 4: Wire into app layer + main.go

Hook gap detection into entity creation and wire resolution side-effects.

**Files:**
- `app/domain/taskapp/taskapp.go` — inject knowledgegapbus into Routes struct; add `go gapBus.Detect(detachedCtx, "task", taskID, content)` after embedding goroutine
- `app/domain/noteapp/noteapp.go` — same pattern
- `app/domain/eventapp/eventapp.go` — same pattern
- `app/domain/clarificationapp/clarificationapp.go` — add `case clarificationkind.KnowledgeGap:` in dispatchResolution:
  1. Parse answer text from resolution
  2. Create note via noteBus.Create() with content=answer, source="clarification"
  3. Create entity link via entityLinkBus.Create() from note → subject entity
  4. Optionally create entity links to related entities mentioned in KnowledgeGapOptions
- `api/services/planner/main.go` — instantiate knowledgegapbus.New(), pass to taskapp/noteapp/eventapp Routes

**Goroutine pattern:** Use `context.Background()` with timeout (30s), NOT the request context. Log errors internally. Follow exact pattern from `taskapp.go:66-74`.

**Tests:** API integration test — create a task, verify clarification card generated (requires seeded embedding data for related entities).

### Task 5: Frontend — card rendering + resolution

Add the knowledge_gap card type to the frontend.

**Files:**
- `api/services/frontend/web/src/types/enums.ts` — add `KnowledgeGap = 'knowledge_gap'` to ClarificationKind, add label "Knowledge Gap" + color (blue/teal for informational)
- `api/services/frontend/web/src/components/shared/ClarificationCard.vue` — add template branch:
  - Show gap description (from question field)
  - Show "What I already know" collapsed section (from reasoning/claude_guess)
  - Textarea for user's answer
  - "Save" button → resolves with `{ answer_text: string, create_note: true }`
  - "Not useful" → resolves with `{ dismissed: true }`

**Pattern:** Follow existing card branches. The `entity_link` kind's confirm/reject pattern is closest. Use `resolveWithValue()` for resolution.

**Tests:** Frontend unit test — render knowledge_gap card, verify textarea + save button, verify resolution payload.

---

## Ordering Constraints

```
Task 1 (enum + migration) → Task 2 (knowledgegapbus) → Task 3 (extractor) → Task 4 (wiring) → Task 5 (frontend)
```

Task 1 is a prerequisite for everything. Tasks 2 and 3 could theoretically parallel but share the extractor interface change, so sequential is safer. Task 5 (frontend) can start after Task 1 if working in isolation, but integration requires Task 4.

---

## What This Does NOT Include

- **Behavioral observation / SQL aggregations** — that's a separate feature (original 5b Layer 1)
- **Pattern_observations table** — deferred; gap detection uses existing clarification + notes infrastructure
- **MCP tools for patterns** — not needed for gap detection
- **Scheduled/batch gap analysis** — Phase 1 is reactive (on entity creation only); batch analysis of existing entities is a future enhancement
- **Gap analysis for existing entities** — only new entities trigger detection; backfill would be a separate task
