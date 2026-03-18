# Tag Backend System

The tag system provides a tagging feature for tasks and contexts, enabling flexible organization and filtering of work items. Tags are independent entities that can be associated with both tasks and contexts through many-to-many relationships. The system follows the layered architecture pattern: handler (tagapp) → business logic (tagbus) → store (tagdb).

## Core Types

### Business Models

```go
// Tag represents a tag entity with unique identifier and name
type Tag struct {
	ID   uuid.UUID
	Name string
}

// NewTag is the input model for creating a new tag
type NewTag struct {
	Name string
}

// QueryFilter allows filtering tags by optional criteria
type QueryFilter struct {
	ID   *uuid.UUID  // Filter by specific tag ID
	Name *string     // Filter by name (case-insensitive ILIKE match)
}
```

### HTTP/App Models

```go
// Tag is the JSON API representation returned to clients
type Tag struct {
	ID   string `json:"id"`      // UUID as string
	Name string `json:"name"`    // Tag name
}

// NewTag is the JSON request body for creating tags
type NewTag struct {
	Name string `json:"name"`
}
```

### Database Models

```go
// tagDB is the internal database representation
type tagDB struct {
	ID   uuid.UUID `db:"tag_id"`
	Name string    `db:"name"`
}
```

### Order Constants

```go
const (
	OrderByID   = "tag_id"   // Order tags by ID (internal field)
	OrderByName = "name"     // Order tags by name (internal field)
)

var DefaultOrderBy = order.NewBy(OrderByName, order.ASC)
```

### Storer Interface

```go
// Storer defines all database operations for tags
type Storer interface {
	// CRUD operations on tags
	Create(ctx context.Context, tag Tag) error
	Delete(ctx context.Context, tag Tag) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Tag, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)

	// Task-tag associations
	AddToTask(ctx context.Context, taskID, tagID uuid.UUID) error
	RemoveFromTask(ctx context.Context, taskID, tagID uuid.UUID) error
	QueryByTask(ctx context.Context, taskID uuid.UUID) ([]Tag, error)

	// Context-tag associations
	AddToContext(ctx context.Context, contextID, tagID uuid.UUID) error
	RemoveFromContext(ctx context.Context, contextID, tagID uuid.UUID) error
	QueryByContext(ctx context.Context, contextID uuid.UUID) ([]Tag, error)
}
```

## File Map

### Models / Types

- **`business/domain/tagbus/model.go`** — Core domain models: `Tag`, `NewTag`, `QueryFilter`
- **`business/domain/tagbus/order.go`** — Ordering constants: `OrderByID`, `OrderByName`, `DefaultOrderBy`
- **`app/domain/tagapp/model.go`** — HTTP API models: `Tag` (app), `NewTag` (app); conversion functions `toAppTag()`, `toAppTags()`, `toBusNewTag()`

### App (Handlers)

- **`app/domain/tagapp/tagapp.go`**
  - **`(*app) create()`** — POST /api/v1/tags; creates a new tag, validates name is required
  - **`(*app) delete()`** — DELETE /api/v1/tags/{tag_id}; removes a tag by ID
  - **`(*app) queryAll()`** — GET /api/v1/tags; lists tags with pagination, filtering, and ordering
  - **`(*app) addToTask()`** — POST /api/v1/tasks/{task_id}/tags/{tag_id}; associates tag with task
  - **`(*app) removeFromTask()`** — DELETE /api/v1/tasks/{task_id}/tags/{tag_id}; disassociates tag from task
  - **`(*app) addToContext()`** — POST /api/v1/contexts/{context_id}/tags/{tag_id}; associates tag with context
  - **`(*app) removeFromContext()`** — DELETE /api/v1/contexts/{context_id}/tags/{tag_id}; disassociates tag from context
  - **`(*app) queryByTask()`** — GET /api/v1/tasks/{task_id}/tags; retrieves all tags for a task
  - **`(*app) queryByContext()`** — GET /api/v1/contexts/{context_id}/tags; retrieves all tags for a context

- **`app/domain/tagapp/filter.go`**
  - **`parseFilter()`** — HTTP query parameter → `tagbus.QueryFilter`; extracts optional name filter

- **`app/domain/tagapp/order.go`**
  - **`parseOrder()`** — HTTP query parameter → `order.By`; parses orderBy field with validation

- **`app/domain/tagapp/route.go`**
  - **`(Routes) Add()`** — Registers all tag endpoints with router; creates Store and Business instances

### Business (Core)

- **`business/domain/tagbus/tagbus.go`**
  - **`NewBusiness()`** — Factory for Business; requires Logger and Storer
  - **`(*Business) Create()`** — Generates UUID, delegates to store; wraps errors
  - **`(*Business) Delete()`** — Delegates to store; wraps errors
  - **`(*Business) Query()`** — Delegates to store with filter/order/pagination; wraps errors
  - **`(*Business) Count()`** — Returns matching tag count; wraps errors
  - **`(*Business) AddToTask()`** — Delegates task-tag association to store; wraps errors
  - **`(*Business) RemoveFromTask()`** — Removes task-tag association; wraps errors
  - **`(*Business) AddToContext()`** — Delegates context-tag association to store; wraps errors
  - **`(*Business) RemoveFromContext()`** — Removes context-tag association; wraps errors
  - **`(*Business) QueryByTask()`** — Retrieves all tags for a task; wraps errors
  - **`(*Business) QueryByContext()`** — Retrieves all tags for a context; wraps errors

### Store

- **`business/domain/tagbus/stores/tagdb/tagdb.go`**
  - **`New()`** — Factory for Store; requires Logger and *sqlx.DB
  - **`(*Store) Create()`** — INSERT INTO tags; named parameters `:tag_id`, `:name`
  - **`(*Store) Delete()`** — DELETE FROM tags WHERE tag_id = :tag_id
  - **`(*Store) Query()`** — SELECT from tags with WHERE 1=1 + optional filters, ORDER BY, LIMIT/OFFSET
  - **`(*Store) Count()`** — SELECT COUNT(*) FROM tags with optional filters
  - **`(*Store) AddToTask()`** — INSERT INTO task_tags (task_id, tag_id)
  - **`(*Store) RemoveFromTask()`** — DELETE FROM task_tags WHERE task_id = :task_id AND tag_id = :tag_id
  - **`(*Store) AddToContext()`** — INSERT INTO context_tags (context_id, tag_id)
  - **`(*Store) RemoveFromContext()`** — DELETE FROM context_tags WHERE context_id = :context_id AND tag_id = :tag_id
  - **`(*Store) QueryByTask()`** — JOIN tags with task_tags; filters by task_id; ordered by name ASC
  - **`(*Store) QueryByContext()`** — JOIN tags with context_tags; filters by context_id; ordered by name ASC

- **`business/domain/tagbus/stores/tagdb/filter.go`**
  - **`applyFilter()`** — Appends SQL WHERE clauses for QueryFilter; supports ID exact match and Name ILIKE (case-insensitive)

- **`business/domain/tagbus/stores/tagdb/order.go`**
  - **`orderByClause()`** — Converts `order.By` field to SQL column name and direction; validates against allowed fields

- **`business/domain/tagbus/stores/tagdb/model.go`**
  - **`toDBTag()`** — Converts `tagbus.Tag` → `tagDB`
  - **`toBusTag()`** — Converts `tagDB` → `tagbus.Tag`
  - **`toBusTags()`** — Bulk conversion `[]tagDB` → `[]tagbus.Tag`

## Impact Callouts

### ⚠ Tag (`business/domain/tagbus/model.go`)
Changing the Tag struct affects:
- `tagapp/model.go` — `toAppTag()` and `toAppTags()` conversion functions must be updated
- `tagdb/model.go` — `toDBTag()` and `toBusTag()` conversion functions must be updated
- API contract: ID must remain `uuid.UUID`, Name must remain `string`

### ⚠ NewTag (`business/domain/tagbus/model.go`)
Changing the NewTag struct affects:
- `tagapp/model.go` — `toBusNewTag()` conversion must be updated
- `tagapp/tagapp.go` — `create()` handler validation logic must be updated
- HTTP request body schema changes (breaking API change)

### ⚠ QueryFilter (`business/domain/tagbus/model.go`)
Changing the QueryFilter struct affects:
- `tagapp/filter.go` — `parseFilter()` must be updated to parse new fields
- `tagdb/filter.go` — `applyFilter()` must generate SQL for new fields
- Query capabilities and HTTP query parameter schema

### ⚠ Storer Interface (`business/domain/tagbus/tagbus.go`)
Adding/changing a Storer method affects:
- `tagbus/tagbus.go` — Business struct must call the method (e.g., `(*Business) Create()` calls `b.storer.Create()`)
- `tagdb/tagdb.go` — Store struct must implement the method
- `tagapp/route.go` — May need route registration if new handler is added
- Contract breaking change if existing methods are modified

### ⚠ tagdb.Store (`business/domain/tagbus/stores/tagdb/tagdb.go`)
Changing Store methods affects:
- All Storer interface methods must maintain same signature as declared in `business/domain/tagbus/tagbus.go`
- SQL queries must handle all filter combinations and order-by fields
- Database schema (tags, task_tags, context_tags tables) must match query structure

## Routes

| Method | Path | Handler | Notes |
|--------|------|---------|-------|
| GET | `/api/v1/tags` | `queryAll()` | List all tags; supports `name` (filter), `orderBy` (id/name), `page`, `rows` |
| POST | `/api/v1/tags` | `create()` | Create tag; JSON body: `{"name":"..."}` |
| DELETE | `/api/v1/tags/{tag_id}` | `delete()` | Delete tag by ID |
| POST | `/api/v1/tasks/{task_id}/tags/{tag_id}` | `addToTask()` | Associate tag with task |
| DELETE | `/api/v1/tasks/{task_id}/tags/{tag_id}` | `removeFromTask()` | Disassociate tag from task |
| GET | `/api/v1/tasks/{task_id}/tags` | `queryByTask()` | List all tags for a task |
| POST | `/api/v1/contexts/{context_id}/tags/{tag_id}` | `addToContext()` | Associate tag with context |
| DELETE | `/api/v1/contexts/{context_id}/tags/{tag_id}` | `removeFromContext()` | Disassociate tag from context |
| GET | `/api/v1/contexts/{context_id}/tags` | `queryByContext()` | List all tags for a context |

All routes require authentication via API key middleware (`mid.Auth(cfg.APIKey)`).

## Database Schema

### tags table
```sql
CREATE TABLE tags (
    tag_id UUID NOT NULL DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    PRIMARY KEY (tag_id)
);
```
- `tag_id`: Primary key, auto-generated UUID
- `name`: Unique tag name (case-sensitive in DB, searched case-insensitive via ILIKE)

### task_tags table (junction)
```sql
CREATE TABLE task_tags (
    task_id UUID NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(tag_id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, tag_id)
);
```
- Composite primary key prevents duplicate associations
- Cascade deletes: removing task or tag removes association

### context_tags table (junction)
```sql
CREATE TABLE context_tags (
    context_id UUID NOT NULL REFERENCES contexts(context_id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(tag_id) ON DELETE CASCADE,
    PRIMARY KEY (context_id, tag_id)
);
```
- Composite primary key prevents duplicate associations
- Cascade deletes: removing context or tag removes association

## Cross-Domain Dependencies

### Task Domain
- `AddToTask()`, `RemoveFromTask()`, `QueryByTask()` associate tags with tasks via `task_tags` junction table
- Task deletion cascades to remove all task-tag associations
- Task domain handlers may call tag business methods to manage task tags

### Context Domain
- `AddToContext()`, `RemoveFromContext()`, `QueryByContext()` associate tags with contexts via `context_tags` junction table
- Context deletion cascades to remove all context-tag associations
- Context domain handlers may call tag business methods to manage context tags

### SDK Dependencies
- `business/sdk/order` — Order.By type for sorting tags
- `business/sdk/page` — Page type for pagination
- `foundation/logger` — Logger for Store operations
- `foundation/web` — HTTP encoding/decoding framework
- `foundation/sqldb` — Named SQL query execution helpers
