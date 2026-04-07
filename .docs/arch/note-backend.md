# Note Backend System

The note system provides a note-taking feature that allows users to capture, organize, and retrieve notes with optional context association and source tracking. Notes can be manually created or generated from raw input ingestion. The system follows the layered architecture pattern: handler (noteapp) → business logic (notebus) → store (notedb).

## Core Types

### Business Models

```go
// Note represents a note entity with optional context association
type Note struct {
	ID         uuid.UUID
	ContextID  *uuid.UUID  // Optional: associated context (nil if standalone)
	Content    string      // Required: note text content
	Source     string      // Required: origin (e.g., "manual", "email", "voice")
	RawInputID *uuid.UUID  // Optional: associated raw input record
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// NewNote is the input model for creating a new note
type NewNote struct {
	ContextID  *uuid.UUID  // Optional: associate with a context
	Content    string      // Required: note text
	Source     string      // Optional: defaults to "manual" if omitted
	RawInputID *uuid.UUID  // Optional: link to raw input
}

// UpdateNote allows selective updates to note fields
type UpdateNote struct {
	ContextID *uuid.UUID  // Optional: change context association
	Content   *string     // Optional: change content
	Source    *string     // Optional: change source
}

// QueryFilter allows filtering notes by optional criteria
type QueryFilter struct {
	ContextID *uuid.UUID  // Filter notes by associated context ID
	Source    *string     // Filter notes by source (exact match)
	Search    *string     // Filter notes by content (case-insensitive ILIKE match)
}
```

### HTTP/App Models

```go
// Note is the JSON API representation returned to clients
type Note struct {
	ID         string  `json:"id"`             // UUID as string
	ContextID  *string `json:"contextId,omitempty"`     // Optional context UUID
	Content    string  `json:"content"`        // Note text
	Source     string  `json:"source"`         // Source origin
	RawInputID *string `json:"rawInputId,omitempty"`    // Optional raw input ID
	CreatedAt  string  `json:"createdAt"`      // RFC3339 timestamp
	UpdatedAt  string  `json:"updatedAt"`      // RFC3339 timestamp
}

// NewNote is the JSON request body for creating notes
type NewNote struct {
	ContextID *string `json:"contextId"`  // Optional context UUID string
	Content   string  `json:"content"`    // Required
	Source    string  `json:"source"`     // Required
}

// UpdateNote is the JSON request body for updating notes
type UpdateNote struct {
	ContextID *string `json:"contextId"`
	Content   *string `json:"content"`
	Source    *string `json:"source"`
}
```

### Database Models

```go
// noteDB is the internal database representation
type noteDB struct {
	ID         uuid.UUID  `db:"note_id"`
	ContextID  *uuid.UUID `db:"context_id"`
	Content    string     `db:"content"`
	Source     string     `db:"source"`
	RawInputID *uuid.UUID `db:"raw_input_id"`
	CreatedAt  time.Time  `db:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at"`
}
```

### Order Constants

```go
const (
	OrderByCreatedAt = "created_at"  // Order notes by creation time (internal field)
	OrderByUpdatedAt = "updated_at"  // Order notes by update time (internal field)
)

var DefaultOrderBy = order.NewBy(OrderByCreatedAt, order.DESC)
```

### Storer Interface

```go
// Storer defines all database operations for notes
type Storer interface {
	Create(ctx context.Context, note Note) error
	Update(ctx context.Context, note Note) error
	Delete(ctx context.Context, note Note) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Note, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (Note, error)
}
```

## File Map

### Models / Types

- **`business/domain/notebus/model.go`** — Core domain models: `Note`, `NewNote`, `UpdateNote`, `QueryFilter`
- **`business/domain/notebus/order.go`** — Ordering constants: `OrderByCreatedAt`, `OrderByUpdatedAt`, `DefaultOrderBy`
- **`app/domain/noteapp/model.go`** — HTTP API models: `Note` (app), `NewNote` (app), `UpdateNote` (app); conversion functions `toAppNote()`, `toAppNotes()`, `toBusNewNote()`, `toBusUpdateNote()`

### App (Handlers)

- **`app/domain/noteapp/noteapp.go`**
  - **`(*app) create()`** — POST /api/v1/notes; creates a new note, validates content is required, defaults source to "manual"; if `ContextID == nil` after creation, fires `asyncClassify` goroutine
  - **`(*app) update()`** — PUT /api/v1/notes/{note_id}; updates existing note fields (context, content, source), retrieves note first to ensure exists
  - **`(*app) delete()`** — DELETE /api/v1/notes/{note_id}; removes a note by ID, retrieves first to ensure exists
  - **`(*app) queryAll()`** — GET /api/v1/notes; lists notes with pagination, filtering, and ordering
  - **`(*app) queryByID()`** — GET /api/v1/notes/{note_id}; retrieves a single note by ID
  - **`(*app) asyncClassify()`** — background goroutine; queries active contexts, calls extractor.ExtractText, then either directly links the note (confidence >= 0.7) or creates a clarification card

- **`app/domain/noteapp/filter.go`**
  - **`parseFilter()`** — HTTP query parameter → `notebus.QueryFilter`; extracts optional context_id, source, and search filters

- **`app/domain/noteapp/order.go`**
  - **`parseOrder()`** — HTTP query parameter → `order.By`; parses orderBy field (created_at, updated_at) with validation against allowed fields

- **`app/domain/noteapp/route.go`**
  - **`(Routes) Add()`** — Registers all note endpoints with router; creates Store and Business instances; also wires contextbus, clarificationbus, and extractor (Claude/Ollama failover) for auto-classification

### Business (Core)

- **`business/domain/notebus/notebus.go`**
  - **`NewBusiness()`** — Factory for Business; requires Logger and Storer
  - **`(*Business) Create()`** — Generates UUID, timestamps, applies default source ("manual"), delegates to store; wraps errors
  - **`(*Business) Update()`** — Applies selective field updates, updates timestamp, delegates to store; wraps errors
  - **`(*Business) Delete()`** — Delegates to store; wraps errors
  - **`(*Business) Query()`** — Delegates to store with filter/order/pagination; wraps errors
  - **`(*Business) Count()`** — Returns matching note count; wraps errors
  - **`(*Business) QueryByID()`** — Retrieves single note by ID; wraps errors

### Store

- **`business/domain/notebus/stores/notedb/notedb.go`**
  - **`NewStore()`** — Factory for Store; requires Logger and *sqlx.DB
  - **`(*Store) Create()`** — INSERT INTO notes; named parameters `:note_id`, `:context_id`, `:content`, `:source`, `:raw_input_id`, `:created_at`, `:updated_at`
  - **`(*Store) Update()`** — UPDATE notes SET context_id, content, source, updated_at WHERE note_id
  - **`(*Store) Delete()`** — DELETE FROM notes WHERE note_id = :note_id
  - **`(*Store) Query()`** — SELECT from notes with WHERE 1=1 + optional filters, ORDER BY, LIMIT/OFFSET pagination
  - **`(*Store) Count()`** — SELECT COUNT(*) FROM notes with optional filters
  - **`(*Store) QueryByID()`** — SELECT single note by note_id; returns sqldb.ErrDBNotFound if not found

- **`business/domain/notebus/stores/notedb/filter.go`**
  - **`applyFilter()`** — Appends SQL WHERE clauses for QueryFilter; supports context_id (exact match), source (exact match), and search (ILIKE case-insensitive content match)

- **`business/domain/notebus/stores/notedb/order.go`**
  - **`orderByClause()`** — Converts `order.By` field to SQL column name and direction; validates against allowed fields (created_at, updated_at)

- **`business/domain/notebus/stores/notedb/model.go`**
  - **`toDBNote()`** — Converts `notebus.Note` → `noteDB`
  - **`toBusNote()`** — Converts `noteDB` → `notebus.Note`
  - **`toBusNotes()`** — Bulk conversion `[]noteDB` → `[]notebus.Note`

## Impact Callouts

### ⚠ Note (`business/domain/notebus/model.go`)
Changing the Note struct affects:
- `noteapp/model.go` — `toAppNote()` and `toAppNotes()` conversion functions must be updated
- `notedb/model.go` — `toDBNote()` and `toBusNote()` conversion functions must be updated
- Database migration required if adding/removing fields
- API contract: ID must remain `uuid.UUID`, timestamps must remain `time.Time`

### ⚠ NewNote (`business/domain/notebus/model.go`)
Changing the NewNote struct affects:
- `noteapp/model.go` — `toBusNewNote()` conversion must be updated
- `noteapp/noteapp.go` — `create()` handler validation logic must be updated (e.g., required field checks)
- HTTP POST request body schema changes (breaking API change)

### ⚠ UpdateNote (`business/domain/notebus/model.go`)
Changing the UpdateNote struct affects:
- `noteapp/model.go` — `toBusUpdateNote()` conversion must be updated
- `noteapp/noteapp.go` — `update()` handler merge logic must be updated
- HTTP PUT request body schema changes (breaking API change)

### ⚠ QueryFilter (`business/domain/notebus/model.go`)
Changing the QueryFilter struct affects:
- `noteapp/filter.go` — `parseFilter()` must be updated to parse new query parameter fields
- `notedb/filter.go` — `applyFilter()` must generate SQL WHERE clauses for new fields
- Query capabilities and HTTP query parameter schema

### ⚠ Storer Interface (`business/domain/notebus/notebus.go`)
Adding/changing a Storer method affects:
- `notebus/notebus.go` — Business methods must call the store method
- `notedb/notedb.go` — Store struct must implement the method with matching signature
- `noteapp/noteapp.go` — May need new handlers if new query method is added
- Contract breaking change if existing method signatures are modified

### ⚠ notedb.Store (`business/domain/notebus/stores/notedb/notedb.go`)
Changing Store methods affects:
- All Storer interface methods must maintain same signature as declared in `business/domain/notebus/notebus.go`
- SQL queries must handle all filter combinations and order-by fields
- Database schema (notes table) must match query structure and column names

## Routes

| Method | Path | Handler | Notes |
|--------|------|---------|-------|
| GET | `/api/v1/notes` | `queryAll()` | List all notes; supports `context_id` (filter), `source` (filter), `search` (content ILIKE), `orderBy` (created_at/updated_at), `page`, `rows` |
| GET | `/api/v1/notes/{note_id}` | `queryByID()` | Retrieve single note by ID; returns 404 if not found |
| POST | `/api/v1/notes` | `create()` | Create note; JSON body: `{"content":"...", "source":"...", "contextId":"..."}` |
| PUT | `/api/v1/notes/{note_id}` | `update()` | Update note fields; JSON body with optional fields: `{"content":"...", "source":"...", "contextId":"..."}` |
| DELETE | `/api/v1/notes/{note_id}` | `delete()` | Delete note by ID; returns 204 No Content on success |

All routes require authentication via API key middleware (`mid.Auth(cfg.APIKey)`).

## Database Schema

### notes table
```sql
CREATE TABLE notes (
    note_id UUID NOT NULL DEFAULT gen_random_uuid(),
    context_id UUID REFERENCES contexts(context_id) ON DELETE SET NULL,
    content TEXT NOT NULL,
    source TEXT NOT NULL,
    raw_input_id UUID REFERENCES raw_inputs(raw_input_id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    PRIMARY KEY (note_id)
);

CREATE INDEX idx_notes_context_id ON notes(context_id);
CREATE INDEX idx_notes_source ON notes(source);
```
- `note_id`: Primary key, auto-generated UUID
- `context_id`: Foreign key to contexts table (nullable, cascade on context delete)
- `content`: Full text of note (indexed for ILIKE search)
- `source`: Origin/type of note (e.g., "manual", "email", "voice")
- `raw_input_id`: Optional link to raw input record (nullable)
- `created_at`: Timestamp of note creation
- `updated_at`: Timestamp of last update

## Cross-Domain Dependencies

### Context Domain
- Notes can be associated with contexts via the `context_id` foreign key
- Deleting a context sets `context_id` to NULL for all associated notes (SET NULL cascade)
- Context handlers may call note business methods to manage context notes

### Raw Input Domain
- Notes can be generated from raw input ingestion via the `raw_input_id` foreign key
- Deleting raw input sets `raw_input_id` to NULL for all associated notes (SET NULL cascade)
- Raw input handlers may call note business methods to create notes from captured input

### Extractor / Classify Pipeline
- `asyncClassify()` fires in a goroutine after note creation when `ContextID == nil`
- Queries active contexts, calls `extractor.ExtractText()`, applies 0.7 confidence threshold
- High confidence → direct `notebus.Update()` with matched context
- Low confidence → creates `clarificationbus.ClarificationItem` of kind `ContextAssignment`
- Dependencies: `business/domain/contextbus`, `business/domain/clarificationbus`, `business/domain/ingestbus/extractor`, `business/types/clarificationkind`

### SDK Dependencies
- `business/sdk/order` — Order.By type for sorting notes
- `business/sdk/page` — Page type for pagination
- `business/sdk/sqldb` — Named SQL query execution helpers
- `foundation/logger` — Logger for Store operations
- `foundation/web` — HTTP encoding/decoding framework
