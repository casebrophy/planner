# Correction Backend System

> The correction domain enables users to reclassify misclassified items (tasks, notes, events) and provides a feedback mechanism for training the classification model. When a user applies a correction, the original item is converted to the correct type and the correction event is recorded for ML model improvement. This system feeds the continuous learning loop that improves the planner's classification accuracy over time.

## Core Types

### Business Types

```go
// Correction represents a recorded classification correction event
type Correction struct {
    ID            uuid.UUID     // Unique identifier
    ClauseText    string        // Text snippet from the original item (title/content)
    PredictedType string        // What the classifier predicted ("task", "note", or "event")
    Confidence    float64       // Confidence score of the original prediction (0.0-1.0)
    ActualType    string        // Correct classification ("task", "note", or "event")
    Source        string        // "clarification_answered" | "correction_applied"
    CreatedAt     time.Time     // When the correction was recorded
}

// NewCorrection is the input for recording a new correction
type NewCorrection struct {
    ClauseText    string  // Text snippet from the original item
    PredictedType string  // Original prediction
    Confidence    float64 // Original confidence
    ActualType    string  // Correct classification
    Source        string  // Source of the correction
}

// QueryFilter allows filtering corrections by various criteria
type QueryFilter struct {
    Source        *string // Filter by source ("clarification_answered" or "correction_applied")
    PredictedType *string // Filter by predicted type
    ActualType    *string // Filter by actual type
}
```

### App Request/Response Types

```go
// CorrectionRequest is the HTTP request body for applying a correction
type CorrectionRequest struct {
    ItemID   string `json:"item_id"`   // UUID of the item to reclassify
    ItemType string `json:"item_type"` // Current type ("task", "note", or "event")
    NewType  string `json:"new_type"`  // Target type ("task", "note", or "event")
}

// CorrectionResult is the HTTP response after a successful correction
type CorrectionResult struct {
    ID   string `json:"id"`   // UUID of the newly created item
    Type string `json:"type"` // New item type
}
```

### Storer Interface

```go
type Storer interface {
    // Create persists a correction record
    Create(ctx context.Context, corr Correction) error
    
    // Query retrieves corrections with filtering, ordering, and pagination
    Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Correction, error)
    
    // Count returns the total number of corrections matching the filter
    Count(ctx context.Context, filter QueryFilter) (int, error)
}
```

## File Map

### Models
- `business/domain/classificationcorrectionbus/model.go` — Correction and NewCorrection structs
- `business/domain/classificationcorrectionbus/stores/correctiondb/model.go` — DB struct + toDBCorrection/toBusCorrection converters

### Handlers
- `app/domain/correctionapp/correctionapp.go` — **correct()** — POST /api/v1/corrections, orchestrates item type conversion **inside a single sqlx.Tx**: fetches source outside tx, then BeginTxx → CreateWithTx target → DeleteWithTx source → RecordWithTx correction → Commit. Holds `db *sqlx.DB` injected via cfg.DB.
- `app/domain/correctionapp/model.go` — Request/response DTOs
- `app/domain/correctionapp/route.go` — Wires `cfg.DB` plus task/note/event/correction buses; registers the POST route
- `app/domain/correctionapp/correctionapp_test.go` — DB-backed tests for all six conversions, NotFound, and explicit assertion that correction rows are persisted (regression for prior silent-swallow bug)

### Business
- `business/domain/classificationcorrectionbus/classificationcorrectionbus.go` — **Record()**, **Query()**, **Count()**, **QueryBySource()** — Core business methods and Storer interface
- `business/domain/classificationcorrectionbus/filter.go` — QueryFilter struct definition
- `business/domain/classificationcorrectionbus/order.go` — OrderByCreatedAt constant, DefaultOrderBy

### Store
- `business/domain/classificationcorrectionbus/stores/correctiondb/correctiondb.go` — **NewStore()**, **Create()**, **Query()**, **Count()** — SQL operations
- `business/domain/classificationcorrectionbus/stores/correctiondb/filter.go` — applyFilter() — builds WHERE clauses from QueryFilter
- `business/domain/classificationcorrectionbus/stores/correctiondb/order.go` — orderByFields map, orderByClause()

## Impact Callouts

### ⚠ Correction struct (business/domain/classificationcorrectionbus/model.go)
Changing this struct shape affects:
- `app/domain/correctionapp/correctionapp.go` — passed to correctionBus.Record() for persistence
- `business/domain/classificationcorrectionbus/stores/correctiondb/model.go` — DB mapping and SQL columns (correction_id, clause_text, predicted_type, confidence, actual_type, source, created_at)
- `business/domain/classificationcorrectionbus/stores/correctiondb/correctiondb.go` — SQL INSERT/SELECT column lists in Create() and Query()
- Migration required if any DB column is added/removed/renamed

### ⚠ NewCorrection struct (business/domain/classificationcorrectionbus/model.go)
Changing this struct affects:
- `app/domain/correctionapp/correctionapp.go` — Record() call at line 212 must provide all required fields
- Validation logic in correct() may need updates if fields change

### ⚠ Storer interface (business/domain/classificationcorrectionbus/classificationcorrectionbus.go)
Adding/changing a method affects:
- `business/domain/classificationcorrectionbus/stores/correctiondb/correctiondb.go` — must implement the method
- `app/domain/correctionapp/correctionapp.go` — if new query/mutation, handler must be added or existing handlers updated
- Route registration in `app/domain/correctionapp/route.go` if new endpoints needed

### ⚠ QueryFilter struct (business/domain/classificationcorrectionbus/filter.go)
Adding a filter field affects:
- `business/domain/classificationcorrectionbus/stores/correctiondb/filter.go` — applyFilter() must add WHERE clause logic
- `business/domain/classificationcorrectionbus/stores/correctiondb/correctiondb.go` — Query() passes filter to applyFilter()
- Any future API endpoints that expose filtering must parse and pass the new filter field

## Database Schema

```sql
CREATE TABLE classification_corrections (
    correction_id UUID        NOT NULL DEFAULT gen_random_uuid(),
    clause_text   TEXT        NOT NULL,
    predicted_type TEXT       NOT NULL,
    confidence    FLOAT8      NOT NULL,
    actual_type   TEXT        NOT NULL,
    source        TEXT        NOT NULL CHECK (source IN ('clarification_answered', 'correction_applied')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (correction_id)
);

CREATE INDEX idx_classification_corrections_source ON classification_corrections(source, created_at DESC);
CREATE INDEX idx_classification_corrections_actual_type ON classification_corrections(actual_type);
```

## Routes

| Method | Path | Handler | Permission |
|--------|------|---------|------------|
| POST | /api/v1/corrections | correct() | API Key Auth |

## Cross-Domain Dependencies

### External Domain Interactions
- **taskbus** — Handler queries and deletes tasks; creates new tasks during conversion
- **notebus** — Handler queries and deletes notes; creates new notes during conversion
- **eventbus** — Handler queries and deletes events; creates new events during conversion

### Dependency Details
- `app/domain/correctionapp/correctionapp.go` — Imports taskbus.Business, notebus.Business, eventbus.Business to orchestrate item type conversion
- `app/domain/correctionapp/route.go` — Constructs instances of all three domain buses and passes them to the correction handler
- The correction handler acts as an orchestration layer: it validates the correction request, performs the type conversion (delete old item, create new item), and records the correction event for ML feedback

### Data Flow
1. POST /api/v1/corrections with {item_id, item_type, new_type}
2. Fetch item from appropriate bus (task/note/event) **outside tx** so ErrDBNotFound maps cleanly to 404
3. `BeginTxx` (defer Rollback)
4. **CreateWithTx** replacement item in target bus with converted data
5. **DeleteWithTx** original item from source bus
6. **RecordWithTx** correction event via classificationcorrectionbus (FATAL on failure — was silently swallowed pre-2026-04-28)
7. `Commit`
8. Return new item ID and type to client

## Notes

- **Source field values:** Corrections can originate from two sources:
  - `"correction_applied"` — User manually corrected a misclassified item via the API
  - `"clarification_answered"` — User resolved a clarification question (recorded elsewhere in the ingestion pipeline)
- **Confidence field:** The confidence score from the original classifier is stored with each correction to enable analysis of which confidence ranges produce the most errors
- **Atomic conversion (2026-04-28 planner-ykh0):** create + delete + correction recording all run inside a single `sqlx.Tx`. Any failure rolls back; the user never observes duplicate or lost entities and never observes a successful conversion without a corresponding correction row.
- **Type conversion rules:** When converting between types, data is mapped intelligently:
  - task.title → note.content (with description appended if present)
  - note.content → task.title (truncated to 100 chars) + empty description
  - event.title and description → task.title and description; note.content combines both
  - All conversions preserve the item's context_id and mark new items as unconfirmed
