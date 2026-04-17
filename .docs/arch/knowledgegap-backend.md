# Knowledge Gap Backend System

> Knowledge gap detection analyzes task/context content to identify missing information (missing contacts, locations, context, dependencies, details) and automatically creates clarification cards. The system uses embedding-based semantic search to find related entities, feeds them to an AI analyzer, and filters candidates by confidence threshold before creating duplicate-deduplicated clarification items.

## Core Types

### Constants (`business/domain/knowledgegapbus/model.go`)
```go
const (
	CategoryMissingContact    = "missing_contact"
	CategoryMissingLocation   = "missing_location"
	CategoryMissingContext    = "missing_context"
	CategoryMissingDependency = "missing_dependency"
	CategoryMissingDetail     = "missing_detail"
)
```

### GapCandidate
```go
type GapCandidate struct {
	Category   string    // One of CategoryMissing* constants
	Question   string    // e.g. "What is Dr. Smith's phone number?"
	Reasoning  string    // e.g. "You have an appointment but no contact info stored"
	Confidence float64   // 0-1; gaps ≤ 0.6 are skipped
	RelatedIDs []string  // IDs of related entities that informed this gap
}
```

### GapDetectionResult
```go
type GapDetectionResult struct {
	CardsCreated int // Clarification cards successfully created
	Skipped      int // Duplicates or low-confidence gaps skipped
}
```

### GapAnalysis
```go
type GapAnalysis struct {
	Gaps []GapCandidate // Structured response from AI extractor
}
```

### Interfaces

**GapAnalyzer** — implemented by extractor package
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
}

// New(log, clarificationBus, embeddingBus, analyzer) *Business
// Detect(ctx, entityType string, entityID uuid.UUID, content string) (GapDetectionResult, error)
```

## File Map

### Core Domain
- **`business/domain/knowledgegapbus/model.go`** — gap constants, GapCandidate, GapDetectionResult, GapAnalysis, analyzer/searcher/creator interfaces
- **`business/domain/knowledgegapbus/knowledgegapbus.go`** — Business struct, New(), Detect() business logic

### Integration Points
- **`api/services/planner/main.go`** — wires Business with clarificationBus, embeddingBus, extractorGapAdapter; exposes via mux.Config.KnowledgeGapBus
- **`app/domain/taskapp/taskapp.go`** — calls gapBus.Detect() async on task creation/update with title+description content

## Detect() Flow

1. **Semantic search**: calls `embeddingBus.Search(content)` (limit 10 results)
2. **Filter by similarity**: keeps only results > 0.5 threshold; early-exit if none
3. **Build summaries**: converts SearchResult to RelatedEntitySummary list
4. **AI analysis**: calls `analyzer.AnalyzeGaps(content, summaries)` → GapAnalysis with GapCandidate list
5. **For each gap**:
   - Skip if Confidence ≤ 0.6
   - Check for duplicates: `clarificationBus.Count(filter: Kind=KnowledgeGap, SubjectType=entityType, SubjectID=entityID)`
   - Skip if duplicate exists
   - Marshal GapCandidate → clarificationbus.KnowledgeGapOptions JSON
   - Upsert clarification card: `clarificationBus.Upsert(NewClarificationItem{ Kind: KnowledgeGap, ... })`
6. **Return**: GapDetectionResult with counts

### RelatedEntitySummary
```go
type RelatedEntitySummary struct {
	SourceType string  // Entity type from embedding search (e.g. "task", "context", "note")
	SourceID   string  // UUID of the related entity
	Similarity float64 // Semantic similarity score from embedding search
	Content    string  // Extracted content/summary from the entity; passed to AI prompt for context
}
```
Built by `Detect()` from `embeddingbus.SearchResult` (line 68-75 in knowledgegapbus.go). Content comes from embedding search results and provides the AI analyzer with related entity context (e.g., full task description, note text, context summary).

## Impact Callouts

### ⚠ GapCandidate (model.go)
Changing this struct affects:
- `extractor.AnalyzeGaps()` — must populate all fields; Confidence is critical for filtering
- `Detect()` loop — iterates candidates, checks Confidence, marshals to KnowledgeGapOptions

### ⚠ RelatedEntitySummary (model.go)
Adding/changing fields affects:
- `extractor.AnalyzeGaps()` — receives list of summaries; adding fields expands AI prompt context
- `Detect()` loop — builds summaries from SearchResult; must map new fields from embedding results
- **Content field**: Fetched from embedding search results; enables AI analyzer to see full entity text, not just IDs

### ⚠ GapAnalyzer interface (model.go)
Adding/changing method affects:
- `api/services/planner/main.go` — extractorGapAdapter must implement the new signature
- `knowledgegapbus.Business.analyzer` — must be injected at New()

### ⚠ Detect() method signature (knowledgegapbus.go)
Changing affects:
- `app/domain/taskapp/taskapp.go` — calls Detect(ctx, "task", taskID, titleAndDesc) async on Create/Update handlers
- `app/domain/noteapp/noteapp.go` — can call Detect(ctx, "note", noteID, content) async on Create/Update handlers
- `app/domain/eventapp/eventapp.go` — can call Detect(ctx, "event", eventID, content) async on Create/Update handlers

### ⚠ Clarification Integration (knowledgegapbus.go)
Changing ClarificationCreator contract or KnowledgeGapOptions struct affects:
- `business/domain/clarificationbus/` — NewClarificationItem must accept Kind, options marshaling, reasoning
- `business/types/clarificationkind/` — must define KnowledgeGap enum value (wired in migration)

## Routes

No dedicated HTTP routes — knowledgegapbus is purely internal. Triggered via:
| Source | Trigger | Flow |
|--------|---------|------|
| taskapp | Create task | Calls gapBus.Detect(ctx, "task", id, titleAndDesc) async in goroutine, ignores errors |
| noteapp | Create note | Calls gapBus.Detect(ctx, "note", id, content) async in goroutine, ignores errors |
| eventapp | Create event | Calls gapBus.Detect(ctx, "event", id, titleAndDesc) async in goroutine, ignores errors |

## Cross-Domain Dependencies

- **`clarificationbus`** — creates NewClarificationItem (Kind: KnowledgeGap, AnswerOptions: JSON), counts existing cards by Kind/SubjectType/SubjectID
- **`embeddingbus`** — semantic search (Search result returns SourceType, SourceID, Similarity)
- **`extractor`** — AI-powered gap analyzer (AnalyzeGaps returns GapAnalysis with high-confidence candidates)
- **`business/types/clarificationkind`** — KnowledgeGap enum value in CHECK constraint (migration v1.36)

## Database

**Migration v1.36** — updates clarification_items.kind CHECK constraint to include 'knowledge_gap':
```sql
ALTER TABLE clarification_items ADD CONSTRAINT clarification_items_kind_check CHECK (kind IN (
    ..., 'knowledge_gap'
));
```

No dedicated tables; clarification_items table (owned by clarificationbus) stores gap-driven cards.

## Implementation Notes

- **Async trigger**: taskapp, noteapp, and eventapp each call Detect() in a goroutine, discard result/error (fire-and-forget). Errors are logged but don't block entity creation. Context is Background() to ensure operation completes even if request is cancelled.
- **Upsert semantics**: Detect() calls `clarificationBus.Upsert()` to create or update clarification cards (duplicates are upserted, not checked first; the duplicate-detection flow was simplified from earlier versions).
- **Confidence filter**: gaps with Confidence ≤ 0.6 are skipped; this threshold is hardcoded in Detect() (not configurable).
- **Related entity selection**: uses first related entity from filtered search results for KnowledgeGapOptions (index [0]); could be randomized or improved in future.
- **Content enrichment**: RelatedEntitySummary.Content comes from embedding search results and is passed to the AI analyzer for richer gap detection (e.g., AI can reference full task description, not just task ID).
