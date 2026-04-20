# Knowledge Gap Backend System

> Knowledge gap detection analyzes task/context content to identify missing information (missing contacts, locations, context, dependencies, details, deadlines, stakeholders, outcomes) and automatically creates clarification cards. The system uses embedding-based semantic search to find related entities, feeds them to an AI analyzer, and filters candidates by confidence threshold before creating duplicate-deduplicated clarification items per category.

## Core Types

### Config (`business/domain/knowledgegapbus/model.go`)
```go
type Config struct {
	ConfidenceThreshold float64 // minimum confidence to create a card (default 0.6)
	EmbeddingLimit      int     // max search results from embedding bus (default 10)
	SimilarityThreshold float64 // minimum similarity to consider a related entity (default 0.5)
}
```

### DetectOptions
```go
type DetectOptions struct {
	DryRun bool // if true, analyze gaps but do not create clarification cards
}
```

### GapCandidate
```go
type GapCandidate struct {
	Category   gapcategory.Category // typed enum from business/types/gapcategory
	Question   string               // e.g. "What is Dr. Smith's phone number?"
	Reasoning  string               // e.g. "You have an appointment but no contact info stored"
	Confidence float64              // 0-1; gaps ≤ ConfidenceThreshold are skipped
	RelatedIDs []string             // IDs of related entities that informed this gap
}
```

### GapDetectionResult
```go
type GapDetectionResult struct {
	CardsCreated int // Clarification cards successfully created
	Skipped      int // Low-confidence gaps skipped
}
```

### GapAnalysis
```go
type GapAnalysis struct {
	Gaps []GapCandidate // Structured response from AI extractor
}
```

### Interfaces

**GapAnalyzer** — implemented by extractorGapAdapter in main.go
```go
type GapAnalyzer interface {
	AnalyzeGaps(ctx context.Context, entityContent string, relatedSummaries []RelatedEntitySummary) (GapAnalysis, error)
}
```

**EmbeddingSearcher** — implemented by embeddingbus
```go
type EmbeddingSearcher interface {
	Search(ctx context.Context, query string, sourceTypes []string, limit int) ([]embeddingbus.SearchResult, error)
}
```

**ClarificationCreator** — implemented by clarificationbus
```go
type ClarificationCreator interface {
	Count(ctx context.Context, filter clarificationbus.QueryFilter) (int, error)
	Create(ctx context.Context, nc clarificationbus.NewClarificationItem) (clarificationbus.ClarificationItem, error)
	Upsert(ctx context.Context, nc clarificationbus.NewClarificationItem) (clarificationbus.ClarificationItem, error)
}
```

### Business
```go
type Business struct {
	log              *logger.Logger
	clarificationBus ClarificationCreator   // Creates clarification cards
	embeddingBus     EmbeddingSearcher      // Semantic search for related entities
	analyzer         GapAnalyzer            // AI-powered gap analyzer
	cfg              Config                 // Tunable thresholds
}

// New(log, clarificationBus, embeddingBus, analyzer, cfg Config) *Business
// Detect(ctx, entityType string, entityID uuid.UUID, content string) (GapDetectionResult, error)
// DetectWithOptions(ctx, entityType string, entityID uuid.UUID, content string, opts DetectOptions) (GapDetectionResult, error)
```

## File Map

### Core Domain
- **`business/domain/knowledgegapbus/model.go`** — Config, defaultConfig, DetectOptions, GapCandidate, GapDetectionResult, GapAnalysis, GapAnalyzer/EmbeddingSearcher/ClarificationCreator interfaces, RelatedEntitySummary
- **`business/domain/knowledgegapbus/knowledgegapbus.go`** — Business struct, New(), Detect(), DetectWithOptions() business logic, helper functions: `dedupeByContent`, `buildSubjectDescription`, `normalizeContentKey`, `buildExistingKnowledgeSummary`

### Integration Points
- **`api/services/planner/main.go`** — wires Business with clarificationBus, embeddingBus, extractorGapAdapter; passes knowledgegapbus.Config{}; extractorGapAdapter.AnalyzeGaps() calls gapcategory.Parse() and skips unknown categories
- **`app/domain/taskapp/taskapp.go`** — calls gapBus.Detect() async on task creation/update with title+description content

## Detect() / DetectWithOptions() Flow

1. **Semantic search**: calls `embeddingBus.Search(content, nil, cfg.EmbeddingLimit)`
2. **Filter by similarity**: keeps only results > `cfg.SimilarityThreshold`; early-exit if none
3. **Deduplicate by content**: calls `dedupeByContent(filtered)` to collapse near-identical entities (e.g., recurring task siblings) by normalized content-prefix key, keeping highest-similarity result per key
4. **Build resultByID map**: UUID string → SearchResult for per-candidate entity lookup
5. **Build summaries**: converts SearchResult to RelatedEntitySummary list
6. **AI analysis**: calls `analyzer.AnalyzeGaps(content, summaries)` → GapAnalysis with GapCandidate list
7. **For each gap**:
   - Skip if Confidence ≤ `cfg.ConfidenceThreshold`
   - Pick best related entity: iterate `gap.RelatedIDs`, use first match from resultByID; fallback to filtered[0]
   - Build KnowledgeGapOptions: `ExistingKnowledgeSummary` is now set via `buildExistingKnowledgeSummary(summaries)` — a formatted multiline summary of ALL related entities (deduplicated search results), not just the best match
   - Populate SubjectDescription via `buildSubjectDescription(entityType, content)` — "{entityType}: {first-line}", truncated to 120 runes
   - If DryRun: increment cardsCreated but skip Upsert
   - Upsert clarification card with `GapCategory: gap.Category.String()`
8. **Return**: GapDetectionResult with counts

### RelatedEntitySummary
```go
type RelatedEntitySummary struct {
	SourceType string  // Entity type from embedding search (e.g. "task", "context", "note")
	SourceID   string  // UUID of the related entity
	Similarity float64 // Semantic similarity score from embedding search
	Content    string  // Extracted content/summary from the entity; passed to AI prompt for context
}
```

## Impact Callouts

### ⚠ GapCandidate.Category type (model.go)
Changed from `string` to `gapcategory.Category`. Affects:
- `extractorGapAdapter.AnalyzeGaps()` in main.go — must call `gapcategory.Parse(g.Category)` and skip invalid
- Test fixtures — must use `gapcategory.MissingContact` etc. instead of string constants

### ⚠ Config (model.go)
Added to control thresholds:
- `api/services/planner/main.go` — passes `knowledgegapbus.Config{}` (uses defaults)
- Tests can pass custom Config to New() to test boundary conditions

### ⚠ GapCategory field on NewClarificationItem
Upsert now sets `GapCategory: gap.Category.String()` which feeds the updated dedup constraint `(kind, subject_type, subject_id, gap_category)`. Each gap category creates its own independent card per entity.

### ⚠ RelatedEntitySummary (model.go)
Adding/changing fields affects:
- `extractor.AnalyzeGaps()` — receives list of summaries; adding fields expands AI prompt context
- `DetectWithOptions()` loop — builds summaries from SearchResult; must map new fields from embedding results
- `buildExistingKnowledgeSummary()` — formats summaries into clarification card text; adding fields may require updating the summary template (e.g., adding a new percentile or metadata field would change the bullet-point format)

### ⚠ GapAnalyzer interface (model.go)
Adding/changing method affects:
- `api/services/planner/main.go` — extractorGapAdapter must implement the new signature

### ⚠ DetectWithOptions() method signature (knowledgegapbus.go)
Changing affects:
- `app/domain/taskapp/taskapp.go` — calls Detect(ctx, "task", taskID, titleAndDesc) async on Create/Update handlers
- `app/domain/noteapp/noteapp.go` — can call Detect() async on Create/Update handlers
- `app/domain/eventapp/eventapp.go` — can call Detect() async on Create/Update handlers

## Routes

No dedicated HTTP routes — knowledgegapbus is purely internal. Triggered via:
| Source | Trigger | Flow |
|--------|---------|------|
| taskapp | Create task | Calls gapBus.Detect(ctx, "task", id, titleAndDesc) async in goroutine, ignores errors |
| noteapp | Create note | Calls gapBus.Detect(ctx, "note", id, content) async in goroutine, ignores errors |
| eventapp | Create event | Calls gapBus.Detect(ctx, "event", id, titleAndDesc) async in goroutine, ignores errors |
| gap-backfill CLI | Manual backfill | Calls gapBus.DetectWithOptions(ctx, entityType, id, content, opts) for existing entities (supports DryRun mode); filters by entity-type and limit |

## Cross-Domain Dependencies

- **`clarificationbus`** — creates NewClarificationItem (Kind: KnowledgeGap, GapCategory: string, AnswerOptions: JSON)
- **`embeddingbus`** — semantic search (Search result returns SourceType, SourceID, Similarity, Content)
- **`extractor`** — AI-powered gap analyzer (AnalyzeGaps returns GapAnalysis with high-confidence candidates)
- **`business/types/gapcategory`** — typed category enum (MissingContact, MissingLocation, MissingContext, MissingDependency, MissingDetail, MissingDeadline, MissingStakeholder, MissingOutcome)
- **`business/types/clarificationkind`** — KnowledgeGap enum value in CHECK constraint (migration v1.36)

## Database

**Migration v1.36** — adds 'knowledge_gap' to clarification_items.kind CHECK constraint.

**Migration v1.40** — adds `gap_category TEXT NOT NULL DEFAULT ''` to clarification_items and updates `uq_clarification_dedup` to `UNIQUE (kind, subject_type, subject_id, gap_category)` enabling per-category dedup.

No dedicated tables; clarification_items table (owned by clarificationbus) stores gap-driven cards.

## Implementation Notes

- **Async trigger (entity creation)**: taskapp, noteapp, and eventapp each call Detect() in a goroutine, discard result/error (fire-and-forget). Errors are logged but don't block entity creation. Context is Background() to ensure operation completes even if request is cancelled.
- **Backfill CLI trigger**: `gap-backfill` admin command iterates all/filtered entities (supports `--entity-type task|event|note|context` and `--limit N`), calling `DetectWithOptions(ctx, entityType, id, content, opts)` synchronously. Returns counts of cards created/skipped per entity type. Supports `--dry-run` to analyze without persisting cards. Implemented in `api/tooling/admin/commands/gapbackfill.go`.
- **Upsert semantics**: Detect()/DetectWithOptions() call `clarificationBus.Upsert()` with GapCategory set; the updated dedup constraint allows one card per (kind, subject_type, subject_id, gap_category) tuple. Each gap category creates an independent card for the same entity.
- **Confidence filter**: gaps with Confidence ≤ cfg.ConfidenceThreshold (default 0.6) are skipped (≤, not <, so exactly 0.6 is skipped).
- **Per-candidate entity selection**: each gap candidate's RelatedIDs are looked up in a resultByID map; first match wins; fallback to filtered[0] if no match.
- **DryRun mode**: DetectWithOptions with DryRun=true counts cardsCreated without calling Upsert; useful for analysis or testing without side-effects.
- **Unknown category handling**: extractorGapAdapter silently skips gaps where gapcategory.Parse fails (unknown string from AI).
- **Content enrichment**: RelatedEntitySummary.Content comes from embedding search results and is passed to the AI analyzer for richer gap detection.
- **Multi-gap emission**: DetectWithOptions loops over all GapCandidates from AnalyzeGaps(), creating separate clarification cards per gap category. No collapsing or bundling — one card per category per entity.
- **SubjectDescription now populated**: previously hardcoded `""`, now set via `buildSubjectDescription()` to `"{entityType}: {first-line-of-content}"` (120-rune max). Affects clarification card UI display — users now see human-readable titles on gap cards.
- **Deduplication helper**: `dedupeByContent()` collapses near-identical entities by normalized content-prefix key (first 200 runes, trimmed/lowercased/whitespace-collapsed), keeping highest-similarity result per key. Prevents analyzer bias from N copies of same content (e.g., recurring tasks). Runs after similarity filter, before gap analysis.
- **Content normalization**: `normalizeContentKey()` supports dedupe by producing stable lowercase, whitespace-collapsed prefix keys; used internally by `dedupeByContent()`.
- **Existing knowledge summary**: `buildExistingKnowledgeSummary()` formats the full list of related entities (from the deduplicated search results) into a human-readable multiline summary for the clarification card's ExistingKnowledgeSummary field. Format: `"Found N related items:\n• type (XX% match): \"snippet...\"\n..."`. Snippet truncation is ~60 runes. Called on every gap to provide full context, not just the best matching entity.
