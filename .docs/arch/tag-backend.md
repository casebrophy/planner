# Tag Backend System

> Tags enable flexible metadata organization across tasks, contexts, and notes. The tag domain provides CRUD operations on tags themselves, plus association management (add/remove) and querying of tags by their parent resources. Tags are stored in a central `tags` table with junction tables (`task_tags`, `context_tags`, `note_tags`) modeling many-to-many relationships.

## Core Types

### App Layer

```go
type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type NewTag struct {
	Name string `json:"name"`
}
```

### Business Layer

```go
type Tag struct {
	ID   uuid.UUID
	Name string
}

type NewTag struct {
	Name string
}

type QueryFilter struct {
	ID   *uuid.UUID
	Name *string
}

type Storer interface {
	Create(ctx context.Context, tag Tag) error
	Delete(ctx context.Context, tag Tag) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Tag, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	AddToTask(ctx context.Context, taskID, tagID uuid.UUID) error
	RemoveFromTask(ctx context.Context, taskID, tagID uuid.UUID) error
	AddToContext(ctx context.Context, contextID, tagID uuid.UUID) error
	RemoveFromContext(ctx context.Context, contextID, tagID uuid.UUID) error
	QueryByTask(ctx context.Context, taskID uuid.UUID) ([]Tag, error)
	QueryByContext(ctx context.Context, contextID uuid.UUID) ([]Tag, error)
	AddToNote(ctx context.Context, noteID, tagID uuid.UUID) error
	RemoveFromNote(ctx context.Context, noteID, tagID uuid.UUID) error
	QueryByNote(ctx context.Context, noteID uuid.UUID) ([]Tag, error)
	QueryNoteIDsByTag(ctx context.Context, tagID uuid.UUID, pg page.Page) ([]uuid.UUID, error)
}

const (
	OrderByID   = "tag_id"
	OrderByName = "name"
)

var DefaultOrderBy = order.NewBy(OrderByName, order.ASC)
```

### Store Layer

```go
type tagDB struct {
	ID   uuid.UUID `db:"tag_id"`
	Name string    `db:"name"`
}
```

## File Map

### App Layer (app/domain/tagapp/)
- `tagapp.go` — **create**, **delete**, **queryAll**, **addToTask**, **removeFromTask**, **addToContext**, **removeFromContext**, **queryByTask**, **queryByContext**, **addToNote**, **removeFromNote**, **queryByNote** HTTP handlers
- `model.go` — Tag, NewTag DTOs + **toAppTag()**, **toAppTags()**, **toBusNewTag()** converters
- `route.go` — **Routes.Add()** registers 12 endpoints with auth middleware
- `filter.go` — **parseFilter()** maps ?name=X → QueryFilter
- `order.go` — **parseOrder()** maps (id, name) → business constants; defaults to OrderByName ASC

### Business Layer (business/domain/tagbus/)
- `tagbus.go` — All public methods: Create, Delete, Query, Count, AddToTask, RemoveFromTask, AddToContext, RemoveFromContext, QueryByTask, QueryByContext, AddToNote, RemoveFromNote, QueryByNote, QueryNoteIDsByTag
- `model.go` — Tag, NewTag, QueryFilter domain types
- `order.go` — OrderByID, OrderByName constants; DefaultOrderBy = OrderByName ASC

### Store Layer (business/domain/tagbus/stores/tagdb/)
- `tagdb.go` — SQL implementation for all Storer methods; junction table queries join task_tags, context_tags, note_tags
- `model.go` — tagDB struct + **toDBTag()**, **toBusTag()**, **toBusTags()** converters
- `filter.go` — **applyFilter()** WHERE clauses: tag_id = :id (exact), name ILIKE :filter_name (case-insensitive contains)
- `order.go` — orderByFields map constants → SQL columns; **orderByClause()** builds ORDER BY clause

## Impact Callouts

### ⚠ Tag struct (business/domain/tagbus/model.go)
Adding/removing/renaming fields affects:
- `tagapp/model.go` — app DTO + toAppTag() converter
- `tagdb/model.go` — tagDB struct + toDBTag/toBusTag converters
- `tagdb/tagdb.go` — SELECT column lists

### ⚠ QueryFilter (business/domain/tagbus/model.go)
Adding filter fields requires:
- `tagapp/filter.go` — parse from query string
- `tagdb/filter.go` — add to applyFilter() WHERE clause

### ⚠ Order Constants (business/domain/tagbus/order.go)
Adding order fields requires:
- `tagapp/order.go` — add to orderByFields map
- `tagdb/order.go` — add to orderByFields map with SQL column name

### ⚠ Storer Interface (business/domain/tagbus/tagbus.go)
Adding methods requires:
- `tagdb/tagdb.go` — implement the new method

## Routes

| Method | Path | Handler |
|--------|------|---------|
| GET | /api/v1/tags | queryAll — list with optional filter, ordering, pagination |
| POST | /api/v1/tags | create — requires name in body |
| DELETE | /api/v1/tags/{tag_id} | delete |
| POST | /api/v1/tasks/{task_id}/tags/{tag_id} | addToTask |
| DELETE | /api/v1/tasks/{task_id}/tags/{tag_id} | removeFromTask |
| GET | /api/v1/tasks/{task_id}/tags | queryByTask |
| POST | /api/v1/contexts/{context_id}/tags/{tag_id} | addToContext |
| DELETE | /api/v1/contexts/{context_id}/tags/{tag_id} | removeFromContext |
| GET | /api/v1/contexts/{context_id}/tags | queryByContext |
| POST | /api/v1/notes/{note_id}/tags/{tag_id} | addToNote |
| DELETE | /api/v1/notes/{note_id}/tags/{tag_id} | removeFromNote |
| GET | /api/v1/notes/{note_id}/tags | queryByNote |

All routes require `X-API-Key` header authentication.

## Cross-Domain Dependencies

- **Task domain** — tag associations via `task_tags` junction table; task IDs must exist
- **Context domain** — tag associations via `context_tags` junction table; context IDs must exist
- **Note domain** — tag associations via `note_tags` junction table; note IDs must exist; QueryNoteIDsByTag supports reverse lookup (paginated notes by tag)
