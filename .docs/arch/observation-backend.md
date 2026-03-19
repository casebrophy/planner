# Observation Backend System

The Observation domain records and queries outcome observations against a polymorphic subject — either a task or a context. Observations are append-only (no update or delete), identified by a `subject_type` + `subject_id` pair, and classified by a `Kind` enum (`duration_accuracy`, `blocker_profile`, `timeline_profile`, `lesson`, `completion_pattern`, `scope_change`, `cost_profile`). The free-form `data` column is a JSONB payload whose schema is determined by the `kind`; the domain does not validate its structure beyond requiring it to be present. The system is the feedback/intelligence layer: observations accumulate over time and are consumed by AI features to learn patterns (duration accuracy, blocker profiles, etc.). The table is `outcome_observations`.

## Core Types

### Observation (Business Layer)
```go
type Observation struct {
	ID          uuid.UUID
	SubjectType string
	SubjectID   uuid.UUID
	Kind        observationkind.Kind
	Data        json.RawMessage
	Source      string
	Confidence  float32
	Weight      float32
	CreatedAt   time.Time
}
```

### NewObservation (Business Layer)
```go
type NewObservation struct {
	SubjectType string
	SubjectID   uuid.UUID
	Kind        observationkind.Kind
	Data        json.RawMessage
	Source      string
	Confidence  float32
	Weight      float32
}
```
Note: No UpdateObservation exists. Observations are immutable after creation.

### Observation (App Layer)
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
```

### NewObservation (App Layer)
```go
type NewObservation struct {
	SubjectType string          `json:"subjectType"`
	SubjectID   string          `json:"subjectId"`
	Kind        string          `json:"kind"`
	Data        json.RawMessage `json:"data"`
	Source      string          `json:"source"`
	Confidence  *float32        `json:"confidence"`
	Weight      *float32        `json:"weight"`
}
```
Defaults applied in `toBusNewObservation()`: `Source` → `"user"`, `Confidence` → `1.0`, `Weight` → `1.0`.

### observationDB (Store Layer)
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

### QueryFilter
```go
type QueryFilter struct {
	SubjectType *string
	SubjectID   *uuid.UUID
	Kind        *observationkind.Kind
}
```

### Storer Interface
```go
type Storer interface {
	Create(ctx context.Context, obs Observation) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Observation, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
}
```

### observationkind.Kind Enum
```go
type Kind struct {
	value string
}

var (
	DurationAccuracy  = Kind{"duration_accuracy"}
	BlockerProfile    = Kind{"blocker_profile"}
	TimelineProfile   = Kind{"timeline_profile"}
	Lesson            = Kind{"lesson"}
	CompletionPattern = Kind{"completion_pattern"}
	ScopeChange       = Kind{"scope_change"}
	CostProfile       = Kind{"cost_profile"}
)

// Functions:
// Parse(s string) (Kind, error)
// MustParse(s string) Kind
// (k Kind) String() string
// (k Kind) MarshalText() ([]byte, error)
// (k *Kind) UnmarshalText(data []byte) error
// (k Kind) EqualString(v string) bool
```

## File Map

### Type Definitions
- **`business/types/observationkind/observationkind.go`** — Kind enum (duration_accuracy, blocker_profile, timeline_profile, lesson, completion_pattern, scope_change, cost_profile) with Parse/MustParse and text marshaling

### App Layer (HTTP Handlers)
- **`app/domain/observationapp/model.go`** — HTTP DTOs: Observation, NewObservation; converter functions:
  - **toAppObservation()** — observationbus.Observation → app Observation (UUIDs to strings, time to RFC3339)
  - **toAppObservations()** — slice converter
  - **toBusNewObservation()** — app NewObservation → observationbus.NewObservation (parses SubjectID UUID, parses Kind, applies Source/Confidence/Weight defaults)
- **`app/domain/observationapp/observationapp.go`** — Handler methods:
  - **record()** — POST /api/v1/observations; validates subjectType, subjectId, kind, data are all present; calls `observationBus.Record()`; returns the created Observation
  - **queryBySubject()** — GET /api/v1/observations/{subject_type}/{subject_id}; parses subject_id as UUID, parses page/rows; calls `observationBus.QueryBySubject()` then `observationBus.Count()` for pagination envelope
- **`app/domain/observationapp/route.go`** — **Routes.Add()** — constructs observationdb.Store, observationbus.Business, registers two endpoints with Auth middleware

### Business Layer (Core Logic)
- **`business/domain/observationbus/model.go`** — Business models: Observation, NewObservation
- **`business/domain/observationbus/observationbus.go`** — Business struct and methods:
  - **NewBusiness()** — constructor taking logger and Storer
  - **Record()** — generates UUID, stamps CreatedAt, calls storer.Create; returns populated Observation
  - **QueryBySubject()** — builds filter for SubjectType+SubjectID, delegates to storer.Query with DefaultOrderBy
  - **QueryByKind()** — builds filter for Kind, delegates to storer.Query with DefaultOrderBy
  - **Query()** — generic pass-through to storer.Query with caller-supplied filter/orderBy/pagination
  - **Count()** — delegates to storer.Count with supplied filter
- **`business/domain/observationbus/filter.go`** — QueryFilter struct (SubjectType, SubjectID, Kind — all optional pointers)
- **`business/domain/observationbus/order.go`** — Order constant `OrderByCreatedAt = "created_at"` and `DefaultOrderBy = order.NewBy(OrderByCreatedAt, order.DESC)`

### Store Layer (Database)
- **`business/domain/observationbus/stores/observationdb/model.go`** — observationDB internal struct (db tags), converter functions:
  - **toDBObservation()** — business Observation → observationDB (Kind enum to string)
  - **toBusObservation()** — observationDB → business Observation (string to Kind via MustParse — panics on unknown values)
  - **toBusObservations()** — slice converter
- **`business/domain/observationbus/stores/observationdb/observationdb.go`** — Store struct and methods:
  - **NewStore()** — constructor taking logger and *sqlx.DB
  - **Create()** — INSERT INTO outcome_observations with all 9 fields via named query
  - **Query()** — SELECT all columns FROM outcome_observations WHERE 1=1; applies applyFilter, orderByClause, OFFSET/FETCH pagination
  - **Count()** — SELECT COUNT(*) FROM outcome_observations WHERE 1=1; applies applyFilter
- **`business/domain/observationbus/stores/observationdb/filter.go`** — **applyFilter()** — appends WHERE clauses for SubjectType (`subject_type = :filter_subject_type`), SubjectID (`subject_id = :filter_subject_id`), Kind (`kind = :filter_kind`)
- **`business/domain/observationbus/stores/observationdb/order.go`** — orderByFields map (`OrderByCreatedAt → "created_at"`), **orderByClause()** — validates field and formats SQL ORDER BY fragment

## Database Schema

```sql
-- Version: 1.10
CREATE TABLE outcome_observations (
    observation_id   UUID        NOT NULL DEFAULT gen_random_uuid(),
    subject_type     TEXT        NOT NULL CHECK (subject_type IN ('task', 'context')),
    subject_id       UUID        NOT NULL,
    kind             TEXT        NOT NULL CHECK (kind IN (
        'duration_accuracy', 'blocker_profile', 'timeline_profile',
        'lesson', 'completion_pattern', 'scope_change', 'cost_profile'
    )),
    data             JSONB       NOT NULL,
    source           TEXT        NOT NULL CHECK (source IN ('user', 'inferred')),
    confidence       REAL        NOT NULL DEFAULT 1.0,
    weight           REAL        NOT NULL DEFAULT 1.0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (observation_id)
);
CREATE INDEX idx_observations_subject ON outcome_observations(subject_type, subject_id);
CREATE INDEX idx_observations_kind ON outcome_observations(kind, created_at DESC);
```

`subject_id` has no foreign key constraint — the subject is resolved at the application level. There is no ON DELETE cascade.

## Impact Callouts

### Observation struct (business/domain/observationbus/model.go)
Adding or removing a field cascades across all layers:
- `business/domain/observationbus/stores/observationdb/model.go` — toDBObservation() and toBusObservation() must be updated
- `app/domain/observationapp/model.go` — App Observation struct and toAppObservation() converter must be updated
- `business/domain/observationbus/stores/observationdb/observationdb.go` — SQL INSERT column list and SELECT column list must match
- `business/sdk/migrate/sql/migrate.sql` — column must be added/removed in `outcome_observations` table

### Storer interface (business/domain/observationbus/observationbus.go)
Adding or changing a Storer method requires:
- `business/domain/observationbus/stores/observationdb/observationdb.go` — Store must implement the new/changed method
- Any business method that calls the storer must be updated

Note: `QueryByKind()` and generic `Query()` are implemented on the business layer but are NOT currently exposed via HTTP routes. If they are wired up in route.go, handler methods must be added to observationapp.go.

### QueryFilter struct (business/domain/observationbus/filter.go)
Adding a filter field requires updates at every layer:
- `business/domain/observationbus/stores/observationdb/filter.go` — applyFilter() must add a new WHERE clause branch
- If the filter is to be exposed via HTTP, app-layer parsing logic must be added to observationapp.go (currently there is no standalone filter.go in observationapp; filter params are built inline in queryBySubject)

### observationkind.Kind enum (business/types/observationkind/observationkind.go)
Adding a new kind value requires:
- `business/sdk/migrate/sql/migrate.sql` — UPDATE the CHECK constraint on `outcome_observations.kind` to include the new value (requires a migration, not just an ALTER)
- toBusObservation() in the store uses `MustParse()` — any row with an unknown kind string will panic at query time; all existing data must be valid before deploying a removal
- `business/types/observationkind/observationkind.go` — add to the `kinds` map and declare a new package-level var

### source field constraint
The DB enforces `source IN ('user', 'inferred')`. The app layer defaults to `"user"` and passes through whatever the caller supplies; the DB will reject any other value. There is no enum type for source in the business layer — it is a bare string. If additional source values are needed, the CHECK constraint in migrate.sql must be updated with a new migration.

### toBusObservation uses MustParse (panics on bad data)
`observationdb/model.go` calls `observationkind.MustParse(o.Kind)` — if the database contains a `kind` value not in the enum map, this will panic at query time. This is a risk when removing a kind value or if rows were inserted outside the application.

## Routes

| Method | Path | Handler | Notes |
|--------|------|---------|-------|
| POST | /api/v1/observations | record | Body: NewObservation; validates subjectType, subjectId, kind, data required; Source defaults to "user"; Confidence/Weight default to 1.0 |
| GET | /api/v1/observations/{subject_type}/{subject_id} | queryBySubject | Path params: subject_type (string), subject_id (UUID); query params: page, rows; returns paginated result envelope |

All endpoints require Auth middleware (API key via `X-API-Key` header).

Business methods `QueryByKind()` and `Query()` are implemented but not currently registered as routes.

## Cross-Domain Dependencies

- **Task Domain** — `subject_type = 'task'` observations reference task UUIDs. There is no DB-level FK, so observations for deleted tasks are orphaned silently.
- **Context Domain** — `subject_type = 'context'` observations reference context UUIDs. Same orphan caveat applies.
- **observationkind package** (`business/types/observationkind`) — Kind enum used in business layer, parsed from strings at app and store boundaries
- **Page SDK** (`business/sdk/page`) — queryBySubject uses page.Parse() and page.Page for pagination
- **Order SDK** (`business/sdk/order`) — Query uses order.By; only `created_at DESC` is currently mapped
- **sqldb utilities** (`foundation/sqldb`) — Store uses NamedExecContext, NamedQuerySlice, NamedQueryStruct helpers
- **Error handling** (`app/sdk/errs`) — Handlers return errs.InvalidArgument for bad input, errs.Internal for store failures; no errs.NotFound path exists (observations are never fetched by ID)
- **HTTP web framework** (`foundation/web`) — Handlers implement web.Encoder pattern; queryBySubject uses query.NewResult for the paginated envelope
