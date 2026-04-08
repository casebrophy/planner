# Observation Backend System

> Immutable outcome observation tracking system that records domain-specific learnings and metrics (duration accuracy, blocker profiles, timelines, lessons, completion patterns, scope changes, cost profiles, and debriefs) across tasks and contexts. Each observation stores typed data as JSON, source attribution, and confidence/weight scores for validation pipelines and machine learning. Observations are append-only (no update or delete).

## Core Types

### observationkind.Kind (business/types/observationkind/)

Valid values: `DurationAccuracy`, `BlockerProfile`, `TimelineProfile`, `Lesson`, `CompletionPattern`, `ScopeChange`, `CostProfile`, `Debrief`.

```go
type Kind struct{ value string }
func Parse(s string) (Kind, error)
func MustParse(s string) Kind
func (k Kind) String() string
func (k Kind) MarshalText() ([]byte, error)
func (k *Kind) UnmarshalText(data []byte) error
```

### Business Layer

```go
type Observation struct {
	ID          uuid.UUID
	SubjectType string               // "task" or "context"
	SubjectID   uuid.UUID
	Kind        observationkind.Kind
	Data        json.RawMessage      // typed domain data; schema determined by kind
	Source      string               // "user" or "inferred"
	Confidence  float32              // 0.0–1.0
	Weight      float32              // 1.0 default importance weight
	CreatedAt   time.Time
}

type NewObservation struct {
	SubjectType string
	SubjectID   uuid.UUID
	Kind        observationkind.Kind
	Data        json.RawMessage
	Source      string               // defaults to "user"
	Confidence  float32              // defaults to 1.0
	Weight      float32              // defaults to 1.0
}

type QueryFilter struct {
	SubjectType *string
	SubjectID   *uuid.UUID
	Kind        *observationkind.Kind
}

const OrderByCreatedAt = "created_at"
var DefaultOrderBy = order.NewBy(OrderByCreatedAt, order.DESC)
```

### Storer Interface

```go
type Storer interface {
	Create(ctx context.Context, obs Observation) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Observation, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
}
```

### App Layer DTO

```go
type Observation struct {
	ID          string          `json:"id"`
	SubjectType string          `json:"subjectType"`
	SubjectID   string          `json:"subjectId"`
	Kind        string          `json:"kind"`
	Data        json.RawMessage `json:"data"`
	Source      string          `json:"source"`
	Confidence  float32         `json:"confidence"`
	Weight      float32         `json:"weight"`
	CreatedAt   string          `json:"createdAt"`
}

type NewObservation struct {
	SubjectType string          `json:"subjectType"`
	SubjectID   string          `json:"subjectId"`
	Kind        string          `json:"kind"`
	Data        json.RawMessage `json:"data"`
	Source      string          `json:"source"`
	Confidence  *float32        `json:"confidence"` // optional, defaults to 1.0
	Weight      *float32        `json:"weight"`     // optional, defaults to 1.0
}
```

### Store Layer

```go
type observationDB struct {
	ID          uuid.UUID       `db:"observation_id"`
	SubjectType string          `db:"subject_type"`
	SubjectID   uuid.UUID       `db:"subject_id"`
	Kind        string          `db:"kind"`
	Data        json.RawMessage `db:"data"`
	Source      string          `db:"source"`
	Confidence  float32         `db:"confidence"`
	Weight      float32         `db:"weight"`
	CreatedAt   time.Time       `db:"created_at"`
}
```

## File Map

### App Layer (app/domain/observationapp/)
- `observationapp.go` — **record()** POST handler; validates input, converts to business types; **queryBySubject()** GET handler with pagination
- `model.go` — Observation, NewObservation DTOs + **toAppObservation()**, **toBusNewObservation()** converters
- `route.go` — **Routes.Add()** wires Store → Business → Handlers with auth middleware

### Business Layer (business/domain/observationbus/)
- `observationbus.go` — **Record()** creates observation with UUID + timestamp; **QueryBySubject()**, **QueryByKind()**, **Query()**, **Count()** delegate to storer
- `model.go` — Observation, NewObservation domain types
- `filter.go` — QueryFilter struct (SubjectType, SubjectID, Kind)
- `order.go` — OrderByCreatedAt constant; DefaultOrderBy = created_at DESC

### Store Layer (business/domain/observationbus/stores/observationdb/)
- `observationdb.go` — **Create()** INSERT into outcome_observations; **Query()** SELECT with filter + ordering + pagination; **Count()** COUNT(*) with filter
- `model.go` — observationDB struct + **toDBObservation()**, **toBusObservation()** converters (typed Kind ↔ string)
- `filter.go` — **applyFilter()** WHERE clauses for SubjectType, SubjectID, Kind
- `order.go` — orderByFields map; **orderByClause()** maps OrderByCreatedAt → "created_at"

## Impact Callouts

### ⚠ observationkind.Kind (business/types/observationkind/)
Adding or removing observation kinds affects:
- `observationkind.go` — add Kind constant and update kinds map
- Database migration — add/remove CHECK constraint value in outcome_observations.kind
- API clients — must parse new kind string values

### ⚠ Observation struct (business/domain/observationbus/model.go)
Adding fields requires:
- `observationdb/model.go` — update observationDB struct + toDBObservation/toBusObservation converters
- `observationapp/model.go` — update app DTO + toAppObservation converter
- Migration SQL — add column to outcome_observations table

### ⚠ QueryFilter (business/domain/observationbus/filter.go)
Adding filter fields requires:
- `observationdb/filter.go` — add WHERE clause in applyFilter()
- Handler validation if new field needs input validation

### ⚠ Storer interface (business/domain/observationbus/observationbus.go)
No update or delete operations exist by design (append-only). Adding a new method requires:
- `observationdb/observationdb.go` — implementation

## Routes

| Method | Path | Handler |
|--------|------|---------|
| POST | /api/v1/observations | record — validates subjectType, subjectId, kind, data; defaults Source="user", Confidence=1.0, Weight=1.0 |
| GET | /api/v1/observations/{subject_type}/{subject_id} | queryBySubject — paginated query; supports ?page=N&rows=M |

Both routes require `X-API-Key` header authentication.

## Cross-Domain Dependencies

- **debriefbus** — creates observations as part of debrief recording
- **taskbus / contextbus** — observations record retroactive learnings about task/context performance
- **business/types/observationkind** — Kind enum
- **business/sdk/page** — pagination support
- **business/sdk/order** — ordering support
