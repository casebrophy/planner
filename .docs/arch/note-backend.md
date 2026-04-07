# Note Backend System

> Personal notes linked to contexts or tasks. Notes are created with content, source, and at least one target (context or task). They support full CRUD operations with filtering by context, task, source, and full-text search. Optional async classification assigns unlinked notes to contexts based on AI extraction.

## Core Types

### Business Layer (`business/domain/notebus/`)

```go
type Note struct {
	ID         uuid.UUID
	ContextID  *uuid.UUID  // Optional: linked context
	TaskID     *uuid.UUID  // Optional: linked task (NEW — v1.21)
	Content    string
	Source     string      // "manual", "api", "email", etc.
	RawInputID *uuid.UUID  // Reference to raw input that generated this note
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type NewNote struct {
	ContextID  *uuid.UUID
	TaskID     *uuid.UUID  // NEW — can link directly to task
	Content    string
	Source     string
	RawInputID *uuid.UUID
}

type UpdateNote struct {
	ContextID *uuid.UUID  // Can update context
	TaskID    *uuid.UUID  // Can update task (NEW — v1.21)
	Content   *string     // Can update content
	Source    *string     // Can update source
}

type QueryFilter struct {
	ContextID *uuid.UUID
	TaskID    *uuid.UUID  // NEW — filter by task
	Source    *string
	Search    *string     // ILIKE full-text search on content
}
```

### Store Layer (`business/domain/notebus/stores/notedb/`)

```go
type noteDB struct {
	ID         uuid.UUID  `db:"note_id"`
	ContextID  *uuid.UUID `db:"context_id"`
	TaskID     *uuid.UUID `db:"task_id"`     // NEW — v1.21 migration
	Content    string     `db:"content"`
	Source     string     `db:"source"`
	RawInputID *uuid.UUID `db:"raw_input_id"`
	CreatedAt  time.Time  `db:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at"`
}
```

### App Layer (`app/domain/noteapp/`)

```go
type Note struct {
	ID         string  `json:"id"`
	ContextID  *string `json:"contextId,omitempty"`
	TaskID     *string `json:"taskId,omitempty"`     // NEW — v1.21
	Content    string  `json:"content"`
	Source     string  `json:"source"`
	RawInputID *string `json:"rawInputId,omitempty"`
	CreatedAt  string  `json:"createdAt"`
	UpdatedAt  string  `json:"updatedAt"`
}

type NewNote struct {
	ContextID *string `json:"contextId"`
	TaskID    *string `json:"taskId"`     // NEW
	Content   string  `json:"content"`
	Source    string  `json:"source"`
}

type UpdateNote struct {
	ContextID *string `json:"contextId"`
	TaskID    *string `json:"taskId"`     // NEW
	Content   *string `json:"content"`
	Source    *string `json:"source"`
}
```

### Storer Interface (`business/domain/notebus/notebus.go`)

```go
type Storer interface {
	Create(ctx context.Context, note Note) error
	Update(ctx context.Context, note Note) error
	Delete(ctx context.Context, note Note) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Note, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (Note, error)
}
```

## File Map

### Models
- `business/domain/notebus/model.go` — Note, NewNote, UpdateNote (business)
- `business/domain/notebus/stores/notedb/model.go` — noteDB, toDBNote(), toBusNote()
- `app/domain/noteapp/model.go` — Note, NewNote, UpdateNote DTOs; converters

### Business & Store
- `business/domain/notebus/notebus.go` — Storer interface; Create/Update/Delete/Query/Count/QueryByID methods
- `business/domain/notebus/stores/notedb/notedb.go` — Store implementation

### Filtering & Ordering
- `business/domain/notebus/filter.go` — QueryFilter
- `business/domain/notebus/stores/notedb/filter.go` — applyFilter()
- `business/domain/notebus/order.go` — constants
- `business/domain/notebus/stores/notedb/order.go` — orderByClause()
- `app/domain/noteapp/filter.go` — parseFilter()
- `app/domain/noteapp/order.go` — parseOrder()

### HTTP
- `app/domain/noteapp/noteapp.go` — create(), update(), delete(), queryAll(), queryByID(), asyncClassify()
- `app/domain/noteapp/route.go` — Routes.Add()

## Impact Callouts

### ⚠ Note struct (business/domain/notebus/model.go)

Changes affect:
- `notedb/model.go` — toDBNote(), toBusNote()
- `notedb/notedb.go` — CREATE/UPDATE/SELECT columns
- `noteapp/model.go` — toAppNote(), app DTO
- **Migration required for new DB columns**

**IMPORTANT:** TaskID added v1.21. Must propagate through all converters and queries.

### ⚠ NewNote struct (business/domain/notebus/model.go)

Changes affect:
- `noteapp/model.go` — app DTO, toBusNewNote()
- `noteapp/noteapp.go` — create() handler validation
- `notebus.go` — Create() assignment

**IMPORTANT:** TaskID added v1.21. App-layer create() validates: **contextId OR taskId required** (matches DB CHECK constraint).

### ⚠ UpdateNote struct (business/domain/notebus/model.go)

Changes affect:
- `noteapp/model.go` — app DTO, toBusUpdateNote()
- `notebus.go` — Update() if-clause
- `notedb/notedb.go` — UPDATE SET clause

**IMPORTANT:** TaskID added v1.21, is mutable. UPDATE SET includes `task_id = :task_id`.

### ⚠ QueryFilter struct (business/domain/notebus/filter.go)

Changes affect:
- `noteapp/filter.go` — parseFilter()
- `notedb/filter.go` — applyFilter()

**IMPORTANT:** TaskID filter added v1.21. Supports `?task_id=<uuid>` query param.

### ⚠ noteDB struct (business/domain/notebus/stores/notedb/model.go)

Column name mismatches break scans:
- Struct tags must match DB column names and named params
- SELECT scans use tags; CREATE/UPDATE use named params

**IMPORTANT:** TaskID added v1.21. Tag `db:"task_id"` must match migration column and `:task_id` named param.

### ⚠ CREATE & UPDATE queries (notedb/notedb.go)

Column lists and SET clause must stay in sync:
- v1.21 added `task_id` to INSERT columns/VALUES and UPDATE SET
- Omitting a column makes field unpatchable/unpopulated

### ⚠ SELECT queries (notedb/notedb.go, lines 84 & 128)

Omitting a column leaves that field zero-value (nil for pointers):
- v1.21 added `task_id` to both Query and QueryByID SELECT lists
- Without it, taskId always returns nil in responses

### ⚠ asyncClassify() condition (noteapp/noteapp.go, line 51)

Now checks both targets:
- Old: `if note.ContextID == nil`
- New: `if note.ContextID == nil && note.TaskID == nil`
- Only classifies notes with neither link set

## Routes

| Method | Path | Handler | Notes |
|--------|------|---------|-------|
| GET | /api/v1/notes | queryAll | params: context_id, task_id (NEW), source, search, page, rows, orderBy |
| GET | /api/v1/notes/{note_id} | queryByID | — |
| POST | /api/v1/notes | create | **requires contextId OR taskId** (NEW validation) |
| PUT | /api/v1/notes/{note_id} | update | can set taskId (NEW) |
| DELETE | /api/v1/notes/{note_id} | delete | — |

## Database Schema (v1.21+)

```sql
notes (
  note_id UUID PK,
  context_id UUID REFS contexts ON DELETE SET NULL,
  task_id UUID REFS tasks ON DELETE SET NULL,  -- NEW
  content TEXT NOT NULL,
  source VARCHAR NOT NULL,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL,
  CONSTRAINT notes_has_target CHECK (context_id IS NOT NULL OR task_id IS NOT NULL),
  INDEX idx_notes_context_id,
  INDEX idx_notes_task_id  -- NEW
)
```

**CRITICAL:** CHECK constraint requires **at least one of context_id/task_id non-NULL**. App must validate before insert.

## Cross-Domain Dependencies

- **contextbus** — asyncClassify() fetches active contexts
- **clarificationbus** — asyncClassify() creates clarification items
- **taskbus** — task_id is FK reference
- **ingestbus/extractor** — Claude Code for async classification
