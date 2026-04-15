# Input Reference Resolution

**Goal:** When a new raw input arrives (e.g., "the wedding got moved to June"), use vector similarity search to find related existing entities and have the extraction step decide whether to UPDATE an existing entity vs. CREATE a new one. Ambiguous matches generate clarifications for user resolution.

**Status:** Planning

---

## Overview

The ingestion pipeline currently always creates new entities. This feature adds a "semantic pre-fetch" step before extraction that searches for existing entities similar to the incoming input text. The matched candidates are injected into the extraction prompt so Claude can make an informed create-vs-update decision.

Flow:
```
Raw input text
  → embed + vector search existing entities (NEW)
  → extraction prompt (enriched with candidate matches) (MODIFIED)
  → extraction result includes entity_resolution decision (MODIFIED)
  → route: UPDATE existing entity | CREATE new entity | AMBIGUOUS → clarification (NEW)
```

---

## Task 1: Pre-Extraction Semantic Lookup

**Files to modify:**
- `business/domain/ingestbus/ingestbus.go` — add semantic lookup before extraction call

**What to do:**

In `processTextInput()` (line 527) and `processRawInput()` (line 160), before calling the extractor, add a semantic search step:

```go
// Pre-extraction: find candidate entity matches
candidates, err := b.embeddingBus.Search(ctx, rawContent, []string{"event", "task", "note"}, 5)
if err != nil {
    b.log.Info(ctx, "semantic pre-fetch failed, continuing without candidates", "error", err)
    // Non-fatal — proceed without candidates
}
```

Pass `candidates` into the extraction functions. The `embeddingBus` is already wired into `ingestbus.Business` (used post-extraction for embedding storage).

**Key decisions:**
- Search limit: 5 candidates (enough for disambiguation, not overwhelming)
- Source types: `["event", "task", "note"]` — all entity types
- Failure is non-fatal: log and proceed with empty candidates (Ollama might be down)
- Similarity threshold constant: `const minCandidateSimilarity = 0.70` — filter out weak matches before passing to Claude

**Dependencies:** None — this is the foundation step.

---

## Task 2: Enriched Extraction Prompt + Schema

**Files to modify:**
- `business/domain/ingestbus/extractor/model.go` — add entity resolution types
- `business/domain/ingestbus/extractor/prompt.go` — inject candidates into prompt
- `business/domain/ingestbus/extractor/claudecli.go` — update JSON schemas

### 2a: New types in model.go

Add to `model.go`:

```go
// EntityMatch represents a candidate entity found via semantic search.
type EntityMatch struct {
    ID         string  `json:"id"`
    SourceType string  `json:"source_type"` // "event", "task", "note"
    Title      string  `json:"title"`
    Content    string  `json:"content"`
    Similarity float64 `json:"similarity"`
}

// EntityResolution is Claude's decision about whether the input references an existing entity.
type EntityResolution struct {
    Action      string  `json:"action"`                // "update", "create", "ambiguous"
    MatchedID   string  `json:"matched_id,omitempty"`   // UUID of matched entity (for update/ambiguous)
    MatchedType string  `json:"matched_type,omitempty"` // "event", "task", "note"
    Confidence  float64 `json:"confidence"`             // 0-1
    Reasoning   string  `json:"reasoning"`              // Why this decision
}
```

Add `EntityResolutions []EntityResolution` to both `TextExtraction` and `EmailExtraction` structs. This is an array because a single input could reference multiple existing entities (e.g., "move Ethan's wedding to June and cancel the dentist appointment").

### 2b: Prompt enrichment in prompt.go

Add a new function `BuildCandidateBlock(candidates []EntityMatch) string` that formats candidates as a numbered list:

```
## Existing Entities (potential matches)
The following existing entities were found to be semantically similar to this input.
If the input is clearly referring to or updating one of these, set entity_resolutions
with action "update" and the matched_id. If ambiguous between multiple, use "ambiguous".
If this is genuinely new content, use "create".

1. [EVENT] id=abc-123 title="Ethan's Wedding" content="Wedding ceremony in May"
2. [TASK] id=def-456 title="Buy wedding gift" content="Get something for Ethan"
```

Inject this block into `BuildEmailExtractionPrompt()` and `BuildTextExtractionPrompt()` after the active contexts section.

### 2c: Schema updates in claudecli.go

Add `entity_resolutions` to both `emailExtractionSchema` and `textExtractionSchema`:

```json
"entity_resolutions": {
  "type": "array",
  "items": {
    "type": "object",
    "properties": {
      "action": {"type": "string", "enum": ["update", "create", "ambiguous"]},
      "matched_id": {"type": "string"},
      "matched_type": {"type": "string", "enum": ["event", "task", "note"]},
      "confidence": {"type": "number"},
      "reasoning": {"type": "string"}
    },
    "required": ["action", "confidence", "reasoning"]
  }
}
```

**Dependencies:** Task 1 (needs candidates to pass into prompt).

---

## Task 3: Update Routing in Pipeline

**Files to modify:**
- `business/domain/ingestbus/ingestbus.go` — route extraction results to update vs. create

**What to do:**

After extraction returns, before the existing create-entities step, process `EntityResolutions`:

```go
for _, res := range extraction.EntityResolutions {
    switch res.Action {
    case "update":
        if err := b.applyEntityUpdate(ctx, res, extraction, ri); err != nil {
            b.log.Error(ctx, "entity update failed", "matched_id", res.MatchedID, "error", err)
            // Fall through to create path
        }
    case "ambiguous":
        if err := b.createAmbiguousMatchClarification(ctx, res, ri); err != nil {
            b.log.Error(ctx, "clarification creation failed", "error", err)
        }
    case "create":
        // Fall through to existing create path
    }
}
```

`applyEntityUpdate()` dispatches to the appropriate bus:
- `res.MatchedType == "event"` → `eventbus.QueryByID()` then `eventbus.Update()` with fields from extraction
- `res.MatchedType == "task"` → `taskbus.QueryByID()` then `taskbus.Update()` with fields from extraction
- Keep `Unconfirmed: true` on updated entities until user confirms

`createAmbiguousMatchClarification()` creates a clarification with the new `AmbiguousEntityMatch` kind.

**Key decisions:**
- Update failure is non-fatal — log and fall through to create path
- Updates preserve `Unconfirmed: true` for user review
- The extraction's `ActionItems`/`Events`/`Notes` arrays still get processed for any items that don't have a corresponding `EntityResolution` with action=update

**Dependencies:** Task 2 (needs entity resolutions from extraction result).

---

## Task 4: Ambiguous Entity Match Clarification Kind

**Files to modify:**
- `business/types/clarificationkind/clarificationkind.go` — add `AmbiguousEntityMatch` constant
- `business/domain/clarificationbus/options.go` — add options struct
- `app/domain/clarificationapp/clarificationapp.go` — add resolution dispatch case
- `business/sdk/migrate/sql/migrate.sql` — migration 1.34 for CHECK constraint

### 4a: New kind constant

In `clarificationkind.go`:
- Add `AmbiguousEntityMatch = MustParse("ambiguous_entity_match")`
- Add to `AllKinds` slice
- Add to `KindWeights` map with weight 8 (high-friction — user must decide)

### 4b: Options struct

In `options.go`:
```go
type AmbiguousEntityMatchOptions struct {
    CandidateID   string `json:"candidate_id"`
    CandidateType string `json:"candidate_type"` // "event", "task", "note"
    CandidateTitle string `json:"candidate_title"`
    Similarity    float64 `json:"similarity"`
    Choices       []string `json:"choices"` // ["use_existing", "create_new"]
}
```

### 4c: Resolution dispatch

In `clarificationapp.go` `dispatchResolution()`:
```go
case clarificationkind.AmbiguousEntityMatch:
    // If user chose "use_existing", apply the update to the matched entity
    // If user chose "create_new", proceed with normal creation from raw input
```

The resolution handler needs to:
- Parse the answer options to get the candidate entity ID and type
- If "use_existing": query + update the matched entity using the extraction data stored in `raw_inputs.result`
- If "create_new": re-run the create path for the raw input (or mark the clarification resolved and let user handle)

### 4d: Migration 1.34

Add to `migrate.sql`:
```sql
-- Version: 1.34
-- Description: Add ambiguous_entity_match clarification kind
ALTER TABLE clarification_items DROP CONSTRAINT IF EXISTS clarification_items_kind_check;
ALTER TABLE clarification_items ADD CONSTRAINT clarification_items_kind_check
    CHECK (kind IN ('context_assignment', 'type_assignment', 'voice_reference',
                    'entity_link', 'event_prep_implication', ..., 'ambiguous_entity_match'));
```

**Dependencies:** Task 3 (needs the routing logic that creates these clarifications).

---

## Task 5: Tests

**Files to modify/create:**
- Tests alongside each modified file following existing patterns

### Test scenarios:

1. **Semantic pre-fetch** (ingestbus): given an existing event "Ethan's Wedding in May" that's been embedded, when processing input "the wedding got moved to June", assert `embeddingBus.Search()` returns the wedding event above threshold
2. **Prompt enrichment** (extractor): table-driven tests — given N candidates with known titles/similarities, assert the prompt contains the expected candidate block
3. **Entity resolution routing** (ingestbus): mock extraction returning `action: "update"` with matched_id → assert `eventbus.Update()` called with correct fields
4. **Ambiguous clarification creation** (ingestbus): extraction returns `action: "ambiguous"` → assert clarification created with kind `AmbiguousEntityMatch`
5. **Threshold boundary** (ingestbus): candidate at 0.69 similarity filtered out; candidate at 0.71 passed to prompt
6. **Clarification resolution** (clarificationapp): resolve with "use_existing" → assert entity updated; resolve with "create_new" → assert new entity created

**Dependencies:** Tasks 1-4 (tests cover all prior tasks).

---

## Execution Order

```
Task 1 (semantic lookup)
  → Task 2 (prompt + schema)
    → Task 3 (update routing)
      → Task 4 (clarification kind)
        → Task 5 (tests)
```

Strictly sequential — each task depends on the prior. Single-domain changes throughout (ingestbus + extractor + clarificationkind), so no parallelization opportunity.

---

## Out of Scope

- Frontend changes to RawInputDetailView step visualization (can be a follow-up)
- Hybrid search (combining vector + SQL filters for better precision)
- Reranking (second-pass Claude scoring)
- Auto-confirm high-confidence updates without user review
- Note update path (notes are typically append-only; reference resolution for notes is less clear)
