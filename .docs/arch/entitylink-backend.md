# EntityLink Backend System

> Directional semantic linking system that enables cross-entity relationships (task-to-note, task-to-event, note-to-event, etc.) with confidence scoring and link kind classification. Supports both manual user-created links (confidence=1.0, kind="manual") and AI-suggested links (kind="ai_suggested", confidence 0.0–1.0). Query operations always fetch both link directions (source and target) to reconstruct full relationship graphs.

## Core Types

### App Layer

```go
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

type EntityLinks struct {
	Items []EntityLink `json:"items"`
	Total int          `json:"total"`
}

type NewEntityLink struct {
	SourceType string `json:"sourceType"`
	SourceID   string `json:"sourceId"`
	TargetType string `json:"targetType"`
	TargetID   string `json:"targetId"`
}
```

### Business Layer

```go
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

type NewEntityLink struct {
	SourceType string
	SourceID   uuid.UUID
	TargetType string
	TargetID   uuid.UUID
	Confidence float64
	Kind       string
}

type Storer interface {
	Create(ctx context.Context, link EntityLink) error
	Delete(ctx context.Context, id uuid.UUID) error
	QueryByID(ctx context.Context, id uuid.UUID) (EntityLink, error)
	QueryBySource(ctx context.Context, sourceType string, sourceID uuid.UUID) ([]EntityLink, error)
	QueryByTarget(ctx context.Context, targetType string, targetID uuid.UUID) ([]EntityLink, error)
}
```

### Store Layer

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

## File Map

### App Layer (app/domain/entitylinkapp/)
- `entitylinkapp.go` — **queryByEntity()**, **create()**, **delete()** HTTP handlers
- `model.go` — App DTOs + **toAppEntityLink()**, **toBusNewEntityLink()** converters
- `route.go` — **Routes.Add()** wires Store → Business → Handlers; registers 3 endpoints

### Business Layer (business/domain/entitylinkbus/)
- `entitylinkbus.go` — **Create()** defaults Kind="manual" Confidence=1.0; **Delete()**, **QueryByID()**, **QueryByEntity()** (combines QueryBySource + QueryByTarget for bidirectional lookup)
- `model.go` — EntityLink, NewEntityLink domain types

### Store Layer (business/domain/entitylinkbus/stores/entitylinkdb/)
- `entitylinkdb.go` — **Create/Delete/QueryByID/QueryBySource/QueryByTarget** with parameterized SQL
- `model.go` — entityLinkDB struct + **toDBEntityLink()**, **toBusEntityLink()** converters

## Impact Callouts

### ⚠ EntityLink / NewEntityLink (business/domain/entitylinkbus/model.go)
Changing these types cascades to:
- `entitylinkapp/model.go` — app DTO + toAppEntityLink() and toBusNewEntityLink() converters
- `entitylinkdb/model.go` — DB struct + toDBEntityLink() and toBusEntityLink() converters
- `entitylinkbus.go` — Create() defaults logic
- `entitylinkdb/entitylinkdb.go` — all SQL queries must reference new columns

### ⚠ Storer Interface (business/domain/entitylinkbus/entitylinkbus.go)
Adding methods requires:
- `entitylinkdb/entitylinkdb.go` — implementation must be added

### ⚠ Entity Type Strings
Valid entity_type values: "task", "note", "event" (hardcoded in callers). No enum type exists—validation is implicit. Adding new entity types requires updating all callers.

### ⚠ QueryByEntity() (business/domain/entitylinkbus/entitylinkbus.go)
Always use QueryByEntity() (not QueryBySource/QueryByTarget alone) when querying links for an entity, to avoid missing reverse links.

### ⚠ Database constraints
- Unique pair index on (source_type, source_id, target_type, target_id) prevents duplicate links
- Self-link CHECK constraint: NOT (source_type = target_type AND source_id = target_id)
- Source/target indexes optimize bidirectional queries

## Routes

| Method | Path | Handler |
|--------|------|---------|
| GET | /api/v1/entity-links?entity_type={type}&entity_id={id} | queryByEntity — returns all links where entity is source or target |
| POST | /api/v1/entity-links | create — manual link; Kind="manual", Confidence=1.0 |
| DELETE | /api/v1/entity-links/{link_id} | delete — removes link by ID |

All routes require `X-API-Key` header authentication.

## Cross-Domain Dependencies

- **clarificationapp** — creates entity links as side-effect of EntityLink clarification resolution
- Links reference external entities (task, note, event) by UUID but do NOT import taskbus/notebus/eventbus
- No referential integrity enforced in code—callers must validate entity existence before creating links
