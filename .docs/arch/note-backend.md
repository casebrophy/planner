# Note Backend System

> Personal notes linked to contexts or tasks. Notes are created with content, source, and at least one target (context or task). They support full CRUD with filtering by context, task, source, and full-text search. Optional async classification assigns unlinked notes to contexts based on AI extraction (auto-assign at confidence >= 0.7, clarification at < 0.7).

## Core Types

### App Layer

```go
type Note struct {
	ID         string  `json:"id"`
	ContextID  *string `json:"contextId,omitempty"`
	TaskID     *string `json:"taskId,omitempty"`
	Content    string  `json:"content"`
	Source     string  `json:"source"`
	RawInputID *string `json:"rawInputId,omitempty"`
	Unconfirmed bool   `json:"unconfirmed"`
	CreatedAt  string  `json:"createdAt"`
	UpdatedAt  string  `json:"updatedAt"`
}

type NewNote struct {
	ContextID *string `json:"contextId"`
	TaskID    *string `json:"taskId"`
	Content   string  `json:"content"`
	Source    string  `json:"source"`
}

type UpdateNote struct {
	ContextID *string `json:"contextId"`
	TaskID    *string `json:"taskId"`
	Content   *string `json:"content"`
	Source    *string `json:"source"`
}
```

### Business Layer

```go
type Note struct {
	ID         uuid.UUID
	ContextID  *uuid.UUID
	TaskID     *uuid.UUID
	Content    string
	Source     string
	RawInputID *uuid.UUID
	Unconfirmed bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type NewNote struct {
	ContextID  *uuid.UUID
	TaskID     *uuid.UUID
	Content    string
	Source     string
	RawInputID *uuid.UUID
}

type UpdateNote struct {
	ContextID *uuid.UUID
	TaskID    *uuid.UUID
	Content   *string
	Source    *string
}

type QueryFilter struct {
	ContextID *uuid.UUID
	TaskID    *uuid.UUID
	Source    *string
	Search    *string
}

const (
	OrderByCreatedAt = "created_at"
	OrderByUpdatedAt = "updated_at"
)

var DefaultOrderBy = order.NewBy(OrderByCreatedAt, order.DESC)

type Storer interface {
	Create(ctx context.Context, note Note) error
	Update(ctx context.Context, note Note) error
	Delete(ctx context.Context, note Note) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Note, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (Note, error)
	DeleteByRawInputUnconfirmed(ctx context.Context, rawInputID uuid.UUID) error
}
```

### Store Layer

```go
type noteDB struct {
	ID         uuid.UUID  `db:"note_id"`
	ContextID  *uuid.UUID `db:"context_id"`
	TaskID     *uuid.UUID `db:"task_id"`
	Content    string     `db:"content"`
	Source     string     `db:"source"`
	RawInputID *uuid.UUID `db:"raw_input_id"`
	CreatedAt  time.Time  `db:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at"`
}
```

## File Map

### App Layer (app/domain/noteapp/)
- `noteapp.go` — **create()** validates content required, one of contextId/taskId required; triggers asyncClassify if both nil; fires async fire-and-forget goroutine to **embeddingBus.EmbedAndStore(ctx, "note", id, content)** for vector storage (errors logged internally); **update/delete/queryAll/queryByID** standard CRUD
- `model.go` — App DTOs + **toAppNote()**, **toAppNotes()**, **toBusNewNote()**, **toBusUpdateNote()** converters
- `route.go` — **Routes.Add()** registers 5 endpoints; wires notebus, contextbus, clarificationbus, extractor, embeddingBus
- `filter.go` — **parseFilter()** maps (context_id, task_id, source, search) → QueryFilter
- `order.go` — **parseOrder()** maps (created_at, updated_at) → notebus constants; defaults to created_at DESC

### Business Layer (business/domain/notebus/)
- `notebus.go` — **Create()** uuid.New() + timestamps, defaults source to "manual"; **Update/Delete/Query/Count/QueryByID/DeleteByRawInputUnconfirmed** delegate to storer
- `model.go` — Note, NewNote, UpdateNote domain types
- `filter.go` — QueryFilter struct (ContextID, TaskID, Source, Search)
- `order.go` — OrderByCreatedAt, OrderByUpdatedAt; DefaultOrderBy = created_at DESC

### Store Layer (business/domain/notebus/stores/notedb/)
- `notedb.go` — **Create/Update/Delete/Query/Count/QueryByID/DeleteByRawInputUnconfirmed** with dynamic WHERE via applyFilter and ORDER via orderByClause
- `model.go` — noteDB struct + **toDBNote()**, **toBusNote()**, **toBusNotes()** converters
- `filter.go` — **applyFilter()** WHERE clauses: ContextID/TaskID/Source equality, Search ILIKE on content
- `order.go` — orderByFields map (created_at, updated_at → SQL columns); **orderByClause()** with direction

## Impact Callouts

### ⚠ Note struct (notebus/model.go)
Changing shape affects:
- `noteapp/model.go` — app Note DTO + toAppNote/toBusNewNote/toBusUpdateNote converters
- `notedb/model.go` — noteDB struct + toDBNote/toBusNote converters
- `notedb/notedb.go` — SELECT/INSERT column lists
- Migration SQL for schema changes

### ⚠ QueryFilter struct (notebus/filter.go)
Adding filter fields requires:
- `noteapp/filter.go` — parseFilter() new query param
- `notedb/filter.go` — applyFilter() new WHERE clause

### ⚠ Order constants (notebus/order.go)
Adding order fields requires:
- `noteapp/order.go` — add to orderByFields map
- `notedb/order.go` — add column mapping

### ⚠ Storer interface (notebus/notebus.go)
Adding methods requires:
- `notedb/notedb.go` — implement the method

### ⚠ Async Classification (noteapp/noteapp.go)
Triggered when ContextID == nil && TaskID == nil on create:
- Calls contextbus to fetch active contexts
- Calls extractor.ExtractText() with context refs
- confidence >= 0.7: auto-updates note with suggested context
- confidence < 0.7: creates clarification item via clarificationbus
- Depends on extractor config (Claude CLI or Ollama)

### ⚠ Async Embedding (noteapp/noteapp.go)
Triggered on every create (fire-and-forget goroutine):
- Calls **embeddingBus.EmbedAndStore(ctx, "note", id, content)** if embeddingBus is not nil
- Fire-and-forget: errors are logged internally in EmbedAndStore(); caller does not capture the error
- Uses note ID + content to generate embeddings and store in pgvector
- Enables semantic search / RAG across notes

## Routes

| Method | Path | Handler |
|--------|------|---------|
| POST | /api/v1/notes | create — content required; contextId or taskId required; async-classify if unclassified |
| GET | /api/v1/notes | queryAll — filter (context_id, task_id, source, search); ordering; pagination |
| GET | /api/v1/notes/{note_id} | queryByID — 404 if not found |
| PUT | /api/v1/notes/{note_id} | update — all fields optional |
| DELETE | /api/v1/notes/{note_id} | delete |

## Cross-Domain Dependencies

- **contextbus** — async classification queries active contexts; validates suggested context
- **clarificationbus** — creates clarification items when confidence < 0.7
- **embeddingbus** — creates embeddings on note creation via EmbedAndStore(); enables semantic search
- **ingestbus/extractor** — Claude CLI or Ollama extractor interface
- **raw_inputs** — raw_input_id FK; notes can originate from ingested content
- **tasks** — task_id FK; notes can be attached to tasks
