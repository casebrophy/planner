# Thread Backend System

> Thread entries are conversation-like records (updates, blocker reports, decisions, observations, emails, etc.) attached to a subject (task or context). Supports rich metadata, optional AI extraction classification (kind/sentiment/requires_action), and flexible querying by subject with pagination. Entries are append-only (no update or delete).

## Core Types

### Business Layer

```go
type ThreadEntry struct {
	ID             uuid.UUID
	SubjectType    string                // "task" or "context"
	SubjectID      uuid.UUID
	Kind           threadentrykind.Kind
	Content        string
	Metadata       *json.RawMessage
	Source         threadsource.Source   // user, voice, email, transaction, system, claude
	SourceID       *uuid.UUID            // optional reference to source entity
	Sentiment      *string
	RequiresAction bool
	CreatedAt      time.Time
}

type NewThreadEntry struct {
	SubjectType    string
	SubjectID      uuid.UUID
	Kind           threadentrykind.Kind
	Content        string
	Metadata       *json.RawMessage
	Source         threadsource.Source
	SourceID       *uuid.UUID
	Sentiment      *string
	RequiresAction bool
	Extract        bool   // when true, run AI extraction to classify kind/sentiment/requiresAction
}

type QueryFilter struct {
	SubjectType    *string
	SubjectID      *uuid.UUID
	Kind           *threadentrykind.Kind
	RequiresAction *bool
}

const OrderByCreatedAt = "created_at"
var DefaultOrderBy = order.NewBy(OrderByCreatedAt, order.DESC)
```

### Extractor Interface

```go
type Extractor interface {
	ExtractThreadEntry(ctx context.Context, content string, subjectType string) (ExtractionResult, error)
}

type ExtractionResult struct {
	Kind              string  `json:"kind"`
	Sentiment         *string `json:"sentiment"`
	BlockingParty     *string `json:"blocking_party"`
	TimelineDeltaDays *int    `json:"timeline_delta_days"`
	RequiresAction    bool    `json:"requires_action"`
	Confidence        float64 `json:"confidence"`
}
```

### Storer Interface

```go
type Storer interface {
	Create(ctx context.Context, entry ThreadEntry) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]ThreadEntry, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (ThreadEntry, error)
}
```

### App Layer DTOs

```go
type ThreadEntry struct {
	ID             string          `json:"id"`
	SubjectType    string          `json:"subjectType"`
	SubjectID      string          `json:"subjectId"`
	Kind           string          `json:"kind"`
	Content        string          `json:"content"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	Source         string          `json:"source"`
	SourceID       *string         `json:"sourceId,omitempty"`
	Sentiment      *string         `json:"sentiment,omitempty"`
	RequiresAction bool            `json:"requiresAction"`
	CreatedAt      string          `json:"createdAt"`
}

type NewThreadEntry struct {
	SubjectType    string          `json:"subjectType"`
	SubjectID      string          `json:"subjectId"`
	Kind           string          `json:"kind"`
	Content        string          `json:"content"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	Source         string          `json:"source"`
	SourceID       *string         `json:"sourceId"`
	Sentiment      *string         `json:"sentiment"`
	RequiresAction bool            `json:"requiresAction"`
}
```

### Store Layer

```go
type threadEntryDB struct {
	ID             uuid.UUID        `db:"entry_id"`
	SubjectType    string           `db:"subject_type"`
	SubjectID      uuid.UUID        `db:"subject_id"`
	Kind           string           `db:"kind"`
	Content        string           `db:"content"`
	Metadata       *json.RawMessage `db:"metadata"`
	Source         string           `db:"source"`
	SourceID       *uuid.UUID       `db:"source_id"`
	Sentiment      *string          `db:"sentiment"`
	RequiresAction bool             `db:"requires_action"`
	CreatedAt      time.Time        `db:"created_at"`
}
```

## File Map

### App Layer (app/domain/threadapp/)
- `threadapp.go` — **addEntry()** POST handler; **queryThread()** GET handler
- `model.go` — ThreadEntry, NewThreadEntry DTOs + **toAppThreadEntry()**, **toBusNewThreadEntry()** converters
- `route.go` — **Routes.Add()** wires Store → Business → Handlers with auth middleware

### Business Layer (business/domain/threadbus/)
- `threadbus.go` — **AddEntry()** creates entry, optionally runs extraction (confidence >= 0.6 threshold for applying results); **QueryBySubject()**, **CountBySubject()**, **QueryByID()**, **Query()**, **Count()** delegate to storer; **WithExtractor()** registers optional AI extractor
- `model.go` — ThreadEntry, NewThreadEntry domain types
- `filter.go` — QueryFilter struct (SubjectType, SubjectID, Kind, RequiresAction)
- `order.go` — OrderByCreatedAt; DefaultOrderBy = created_at DESC
- `extractor.go` — Extractor interface + ExtractionResult struct

### Store Layer (business/domain/threadbus/stores/threaddb/)
- `threaddb.go` — **Create/Query/Count/QueryByID** SQL methods
- `model.go` — threadEntryDB struct + **toDBThreadEntry()**, **toBusThreadEntry()** converters
- `filter.go` — **applyFilter()** WHERE clauses for SubjectType, SubjectID, Kind, RequiresAction
- `order.go` — orderByFields map; **orderByClause()** maps constants → SQL columns

## Impact Callouts

### ⚠ ThreadEntry (all layers)
Adding a field to ThreadEntry requires:
- Business model.go — add field to ThreadEntry/NewThreadEntry structs
- `threaddb/model.go` — add db tag to threadEntryDB + update toDBThreadEntry/toBusThreadEntry converters
- `threadapp/model.go` — add JSON tag to app DTO + update toAppThreadEntry/toBusNewThreadEntry converters
- `threaddb/threaddb.go` — update INSERT and SELECT SQL
- Migration SQL — add column to thread_entries table

### ⚠ Extraction Pipeline (threadbus/threadbus.go AddEntry)
The Extract flag in NewThreadEntry triggers optional AI extraction. If extractor != nil and Extract=true, extraction result (if confidence >= 0.6) is applied to kind, sentiment, requiresAction, and metadata. Silent fallback to defaults if extraction fails.

### ⚠ QueryFilter (threadbus/filter.go)
Adding filter fields requires:
- `threaddb/filter.go` — add applyFilter() WHERE clause
- Consider adding HTTP query param parsing if exposing via API

### ⚠ threadentrykind / threadsource enums (business/types/)
Changing enum values affects DB CHECK constraints and all string ↔ enum conversions in converters.

## Routes

| Method | Path | Handler |
|--------|------|---------|
| POST | /api/v1/threads | addEntry — create thread entry; optional AI extraction via Extract flag |
| GET | /api/v1/threads/{subject_type}/{subject_id} | queryThread — paginated entries for subject; ?page=N&rows=M |

Both routes require `X-API-Key` header (mid.Auth middleware).

## Cross-Domain Dependencies

- **taskbus** — SubjectType="task" references tasks via SubjectID
- **contextbus** — SubjectType="context" references contexts via SubjectID
- **business/types/threadentrykind** — Kind enum
- **business/types/threadsource** — Source enum
- **Extractor interface** — optional AI classification; can be nil (skips extraction when not configured)
