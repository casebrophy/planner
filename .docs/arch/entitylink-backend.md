# EntityLink Backend System

> The entitylink domain represents directional semantic links between two entities (tasks, notes, events). Links can be created manually (confidence=1.0, kind="manual") or by the AI classify pipeline (kind="ai_suggested", confidence 0.0–1.0). The system follows the layered architecture pattern: handler (entitylinkapp) → business logic (entitylinkbus) → store (entitylinkdb).

---

## Core Types

### Business Layer — `business/domain/entitylinkbus/model.go`

```go
// EntityLink is a directional semantic link between two entities.
// At query time, always fetch both directions (QueryBySource + QueryByTarget).
type EntityLink struct {
    ID         uuid.UUID
    SourceType string    // "task" | "note" | "event"
    SourceID   uuid.UUID
    TargetType string    // "task" | "note" | "event"
    TargetID   uuid.UUID
    Confidence float64   // 1.0 for manual; 0.0–1.0 for AI-suggested
    Kind       string    // "manual" | "ai_suggested"
    CreatedAt  time.Time
}

// NewEntityLink is the input for creating an entity link.
type NewEntityLink struct {
    SourceType string
    SourceID   uuid.UUID
    TargetType string
    TargetID   uuid.UUID
    Confidence float64
    Kind       string
}
```

### Store Layer — `business/domain/entitylinkbus/stores/entitylinkdb/model.go`

```go
type entityLinkDB struct {
    ID         uuid.UUID `db:"link_id"`
    SourceType string    `db:"source_type"`
    SourceID   uuid.UUID `db:"source_id"`
    TargetType string    `db:"target_type"`
    TargetID   uuid.UUID `db:"target_id"`
    Confidence float64   `db:"confidence"`
    Kind       string    `db:"kind"`
    CreatedAt  time.Time `db:"created_at"`
}
```

Converters:
- `toDBEntityLink(entitylinkbus.EntityLink) entityLinkDB` — business → store layer
- `toBusEntityLink(entityLinkDB) entitylinkbus.EntityLink` — store → business layer

### App Layer — `app/domain/entitylinkapp/model.go`

```go
// EntityLink is the JSON response DTO.
type EntityLink struct {
    ID         string  `json:"id"`
    SourceType string  `json:"sourceType"`
    SourceID   string  `json:"sourceId"`
    TargetType string  `json:"targetType"`
    TargetID   string  `json:"targetId"`
    Confidence float64 `json:"confidence"`
    Kind       string  `json:"kind"`
    CreatedAt  string  `json:"createdAt"`
}

// EntityLinks is the list response.
type EntityLinks struct {
    Items []EntityLink `json:"items"`
    Total int          `json:"total"`
}

// NewEntityLink is the request body for POST /api/v1/entity-links.
type NewEntityLink struct {
    SourceType string `json:"sourceType"`
    SourceID   string `json:"sourceId"`
    TargetType string `json:"targetType"`
    TargetID   string `json:"targetId"`
}
```

Converters:
- `toAppEntityLink(entitylinkbus.EntityLink) EntityLink` — business → app layer (UUIDs to strings, timestamp to RFC3339)
- `toBusNewEntityLink(NewEntityLink) (entitylinkbus.NewEntityLink, error)` — app → business layer (parses UUIDs, sets kind="manual", confidence=1.0)

---

## File Map

### App (Handlers) — `app/domain/entitylinkapp/`

- **`entitylinkapp.go`** — `queryByEntity(ctx, r) web.Encoder` (GET /api/v1/entity-links) — requires `entity_type` and `entity_id` query params; calls `QueryByEntity` which combines source+target results; returns `EntityLinks`
- **`entitylinkapp.go`** — `create(ctx, r) web.Encoder` (POST /api/v1/entity-links) — decodes `NewEntityLink`, converts to bus type (manual/1.0), creates link, returns `EntityLink`; duplicate constraint propagates as Internal error
- **`entitylinkapp.go`** — `delete(ctx, r) web.Encoder` (DELETE /api/v1/entity-links/{link_id}) — pre-flight QueryByID (returns 404 if not found), then deletes; returns 204 NoResponse
- **`route.go`** — `Routes.Add(a *web.App, cfg mux.Config)` — instantiates entitylinkdb.Store and entitylinkbus.Business; registers three routes with API-key auth middleware

### Business (Core Logic) — `business/domain/entitylinkbus/`

- **`entitylinkbus.go`** — `Business` struct holds storer + logger
- **`entitylinkbus.go`** — `NewBusiness(log, storer) *Business` — constructor
- **`entitylinkbus.go`** — `Create(ctx, NewEntityLink) (EntityLink, error)` — defaults kind to "manual" if empty, defaults confidence to 1.0 for manual links; generates UUID + timestamp; delegates to storer; wraps `ErrDBDuplicatedEntry` from store layer
- **`entitylinkbus.go`** — `QueryByID(ctx, id) (EntityLink, error)` — delegates to storer
- **`entitylinkbus.go`** — `Delete(ctx, id) error` — delegates to storer
- **`entitylinkbus.go`** — `QueryByEntity(ctx, entityType, entityID) ([]EntityLink, error)` — fetches both directions (QueryBySource + QueryByTarget) and concatenates results
- **`entitylinkbus.go`** — `Storer` interface defines contract: Create, Delete, QueryByID, QueryBySource, QueryByTarget

### Store (Database) — `business/domain/entitylinkbus/stores/entitylinkdb/`

- **`entitylinkdb.go`** — `Store` struct holds logger + sqlx.ExtContext
- **`entitylinkdb.go`** — `NewStore(log, db) *Store` — constructor
- **`entitylinkdb.go`** — `Create(ctx, link) error` — INSERT into entity_links; checks `sqldb.ErrDBDuplicatedEntry` (23505) and wraps with "entity link already exists"
- **`entitylinkdb.go`** — `QueryByID(ctx, id) (EntityLink, error)` — SELECT by link_id
- **`entitylinkdb.go`** — `Delete(ctx, id) error` — DELETE WHERE link_id
- **`entitylinkdb.go`** — `QueryBySource(ctx, sourceType, sourceID) ([]EntityLink, error)` — SELECT WHERE source_type AND source_id ORDER BY created_at DESC
- **`entitylinkdb.go`** — `QueryByTarget(ctx, targetType, targetID) ([]EntityLink, error)` — SELECT WHERE target_type AND target_id ORDER BY created_at DESC

---

## Impact Callouts

### EntityLink struct (`business/domain/entitylinkbus/model.go`)

**Used by:**
- `entitylinkbus.go` — all business methods accept or return EntityLink
- `entitylinkdb/model.go` — toDBEntityLink() and toBusEntityLink() converters
- `entitylinkapp/model.go` — toAppEntityLink() converter
- `entitylinkapp/entitylinkapp.go` — all handlers work with EntityLink (via Business methods)

**If modified:** Update converters in model.go files (entitylinkdb + entitylinkapp), update SQL INSERT/SELECT clauses.

### Storer interface (`business/domain/entitylinkbus/entitylinkbus.go`)

**Implemented by:**
- `business/domain/entitylinkbus/stores/entitylinkdb/entitylinkdb.go` — Store struct implements all five methods

**If modified:** Add method to entitylinkdb.Store, update Business methods that call storer.

---

## Routes

| Method | Path | Handler | Auth |
|--------|------|---------|------|
| GET | /api/v1/entity-links | queryByEntity | API key |
| POST | /api/v1/entity-links | create | API key |
| DELETE | /api/v1/entity-links/{link_id} | delete | API key |

All routes registered in `app/domain/entitylinkapp/route.go` → `Routes.Add()`. Auth middleware applied via `mid.Auth(cfg.APIKey)`.

---

## Cross-Domain Dependencies

**Imports from shared infrastructure:**
- `github.com/casebrophy/planner/business/sdk/sqldb` — NamedExecContext, NamedQuerySlice, NamedQueryStruct helpers; `ErrDBDuplicatedEntry` for unique constraint detection
- `github.com/casebrophy/planner/foundation/logger` — structured logging
- `github.com/casebrophy/planner/foundation/web` — HTTP framework (App, Handle, Encoder, Param)
- `github.com/casebrophy/planner/app/sdk/errs` — error codes (InvalidArgument, NotFound, Internal)
- `github.com/casebrophy/planner/app/sdk/mid` — middleware (Auth)
- `github.com/casebrophy/planner/app/sdk/mux` — mux config (Log, DB, APIKey)

**Used by classify pipeline:**
- `classifyapp` and `asyncClassify` goroutines in `noteapp`/`eventapp` create entity links when AI finds related entities

---

## Database Table

```sql
CREATE TABLE entity_links (
    link_id     UUID PRIMARY KEY,
    source_type TEXT NOT NULL,
    source_id   UUID NOT NULL,
    target_type TEXT NOT NULL,
    target_id   UUID NOT NULL,
    confidence  DOUBLE PRECISION NOT NULL,
    kind        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL
);
```
