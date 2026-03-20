# Email Backend System

The Email domain provides read-only access to parsed email records stored after ingestion through the raw_inputs pipeline. Emails are associated with raw_inputs and optionally linked to a context. The architecture follows the standard layered pattern: HTTP handlers → business logic core → database store, with thin translation layers between each tier.

## Core Types

### Email (Business Layer)
```go
type Email struct {
	ID          uuid.UUID
	RawInputID  uuid.UUID
	MessageID   *string
	FromAddress string
	FromName    *string
	ToAddress   string
	Subject     string
	BodyText    string
	BodyHTML    *string
	ReceivedAt  time.Time
	ContextID   *uuid.UUID
	CreatedAt   time.Time
}
```

### NewEmail (Business Layer)
```go
type NewEmail struct {
	RawInputID  uuid.UUID
	MessageID   *string
	FromAddress string
	FromName    *string
	ToAddress   string
	Subject     string
	BodyText    string
	BodyHTML    *string
	ReceivedAt  time.Time
	ContextID   *uuid.UUID
}
```

### UpdateEmail (Business Layer)
```go
type UpdateEmail struct {
	ContextID *uuid.UUID
}
```

### Email (App Layer)
```go
type Email struct {
	ID          string  `json:"id"`
	RawInputID  string  `json:"rawInputId"`
	MessageID   *string `json:"messageId,omitempty"`
	FromAddress string  `json:"fromAddress"`
	FromName    *string `json:"fromName,omitempty"`
	ToAddress   string  `json:"toAddress"`
	Subject     string  `json:"subject"`
	BodyText    string  `json:"bodyText"`
	BodyHTML    *string `json:"bodyHtml,omitempty"`
	ReceivedAt  string  `json:"receivedAt"`
	ContextID   *string `json:"contextId,omitempty"`
	CreatedAt   string  `json:"createdAt"`
}
```

### emailDB (Store Layer)
```go
type emailDB struct {
	ID          uuid.UUID  `db:"email_id"`
	RawInputID  uuid.UUID  `db:"raw_input_id"`
	MessageID   *string    `db:"message_id"`
	FromAddress string     `db:"from_address"`
	FromName    *string    `db:"from_name"`
	ToAddress   string     `db:"to_address"`
	Subject     string     `db:"subject"`
	BodyText    string     `db:"body_text"`
	BodyHTML    *string    `db:"body_html"`
	ReceivedAt  time.Time  `db:"received_at"`
	ContextID   *uuid.UUID `db:"context_id"`
	CreatedAt   time.Time  `db:"created_at"`
}
```

### QueryFilter
```go
type QueryFilter struct {
	ContextID   *uuid.UUID
	FromAddress *string
	Subject     *string
}
```

### Storer Interface
```go
type Storer interface {
	Create(ctx context.Context, e Email) error
	Update(ctx context.Context, e Email) error
	Delete(ctx context.Context, e Email) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Email, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (Email, error)
	QueryByMessageID(ctx context.Context, messageID string) (Email, error)
}
```

## File Map

### App Layer (HTTP Handlers)
- **`app/domain/emailapp/model.go`** — HTTP DTOs: Email (read-only response) with conversion functions:
  - **toAppEmail()** — emailbus.Email → app Email (UUID fields to strings, times to RFC3339)
  - **toAppEmails()** — slice converter
  - **Email.Encode()** — implements web.Encoder via json.Marshal
- **`app/domain/emailapp/emailapp.go`** — Handler methods:
  - **queryAll()** — GET /api/v1/emails, parses page/rows, filter, orderBy; calls emailBus.Query + emailBus.Count; returns paginated result
  - **queryByID()** — GET /api/v1/emails/{email_id}, parses UUID from path param; checks sqldb.ErrDBNotFound → errs.NotFound
- **`app/domain/emailapp/route.go`** — **Routes.Add()** — registers both endpoints with Auth middleware; instantiates emaildb.Store and emailbus.Business
- **`app/domain/emailapp/filter.go`** — **parseFilter()** — parses query parameters into QueryFilter:
  - `context_id` → uuid.UUID → filter.ContextID
  - `from_address` → string → filter.FromAddress
  - `subject` → string → filter.Subject
- **`app/domain/emailapp/order.go`** — **parseOrder()** — maps request orderBy field names to emailbus constants:
  - `"received_at"` → `emailbus.OrderByReceivedAt`
  - `"subject"` → `emailbus.OrderBySubject`
  - `"created_at"` → `emailbus.OrderByCreatedAt`

### Business Layer (Core Logic)
- **`business/domain/emailbus/model.go`** — Business models: Email, NewEmail, UpdateEmail
- **`business/domain/emailbus/emailbus.go`** — Business struct, Storer interface definition, and methods:
  - **NewBusiness()** — constructor taking logger and Storer
  - **Create()** — generates UUID, sets CreatedAt to now, calls storer.Create
  - **Update()** — applies partial update (only ContextID), calls storer.Update
  - **Delete()** — calls storer.Delete
  - **Query()** — delegates to storer with filter/order/pagination
  - **Count()** — delegates to storer to count filtered emails
  - **QueryByID()** — delegates to storer by UUID
  - **QueryByMessageID()** — delegates to storer by MIME message-id string (used during ingestion for deduplication)
- **`business/domain/emailbus/filter.go`** — QueryFilter struct (ContextID, FromAddress, Subject)
- **`business/domain/emailbus/order.go`** — Order field constants and DefaultOrderBy:
  - `OrderByReceivedAt = "received_at"` (DefaultOrderBy: DESC)
  - `OrderBySubject = "subject"`
  - `OrderByCreatedAt = "created_at"`

### Store Layer (Database)
- **`business/domain/emailbus/stores/emaildb/model.go`** — emailDB internal struct (all db tags), conversion functions:
  - **toDBEmail()** — emailbus.Email → emailDB
  - **toBusEmail()** — emailDB → emailbus.Email
  - **toBusEmails()** — slice converter
- **`business/domain/emailbus/stores/emaildb/emaildb.go`** — Store struct and methods:
  - **NewStore()** — constructor taking logger and *sqlx.DB
  - **Create()** — INSERT INTO emails with all 12 columns via named query using `:email_id, :raw_input_id, :message_id, :from_address, :from_name, :to_address, :subject, :body_text, :body_html, :received_at, :context_id, :created_at`
  - **Update()** — UPDATE emails SET `context_id = :context_id` WHERE `email_id = :email_id`
  - **Delete()** — DELETE FROM emails WHERE `email_id = :email_id`
  - **Query()** — SELECT all columns with WHERE 1=1 base, applies applyFilter, ORDER BY clause, OFFSET/FETCH pagination
  - **Count()** — SELECT COUNT(*) FROM emails with same filter applied
  - **QueryByID()** — SELECT all columns WHERE `email_id = :email_id`
  - **QueryByMessageID()** — SELECT all columns WHERE `message_id = :message_id`
- **`business/domain/emailbus/stores/emaildb/filter.go`** — **applyFilter()** — builds WHERE clauses:
  - `filter.ContextID` → `AND context_id = :filter_context_id` (exact match)
  - `filter.FromAddress` → `AND from_address = :filter_from_address` (exact match)
  - `filter.Subject` → `AND subject ILIKE :filter_subject` (case-insensitive substring, wrapped in `%...%`)
- **`business/domain/emailbus/stores/emaildb/order.go`** — orderByFields map (business constants → SQL column names), **orderByClause()** — validates field exists and formats `column direction` string

## Database Schema

```sql
CREATE TABLE emails (
    email_id      UUID        NOT NULL DEFAULT gen_random_uuid(),
    raw_input_id  UUID        NOT NULL REFERENCES raw_inputs(raw_input_id),
    message_id    TEXT,
    from_address  TEXT        NOT NULL,
    from_name     TEXT,
    to_address    TEXT        NOT NULL,
    subject       TEXT        NOT NULL,
    body_text     TEXT        NOT NULL,
    body_html     TEXT,
    received_at   TIMESTAMPTZ NOT NULL,
    context_id    UUID        REFERENCES contexts(context_id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (email_id)
);

CREATE INDEX idx_emails_raw_input ON emails(raw_input_id);
CREATE INDEX idx_emails_context ON emails(context_id);
CREATE INDEX idx_emails_received ON emails(received_at DESC);
CREATE UNIQUE INDEX idx_emails_message_id ON emails(message_id) WHERE message_id IS NOT NULL;
```

Note: `message_id` has a partial unique index (only on non-NULL rows) to support MIME message-id deduplication during ingestion.

## Impact Callouts

### Email struct (business/domain/emailbus/model.go)
Adding or removing a field affects:
- `business/domain/emailbus/stores/emaildb/model.go` — toDBEmail() and toBusEmail() must include the new field; emailDB struct must add the corresponding `db:` tag
- `app/domain/emailapp/model.go` — app Email struct and toAppEmail() must include the new field with its `json:` tag
- `business/domain/emailbus/stores/emaildb/emaildb.go` — SQL INSERT column list and VALUES named params in Create() must be updated; SELECT column list in Query(), QueryByID(), QueryByMessageID() must be updated; UPDATE SET clause in Update() must be updated if the field is mutable
- `business/sdk/migrate/sql/migrate.sql` — must add/remove columns and update constraints or indexes

### NewEmail struct (business/domain/emailbus/model.go)
Adding a field affects:
- `business/domain/emailbus/emailbus.go` — Create() must assign the new field when constructing the Email
- Caller code (e.g. ingestbus or any future app-layer create handler) must populate the new field

### UpdateEmail struct (business/domain/emailbus/model.go)
Currently only contains ContextID. Adding a field affects:
- `business/domain/emailbus/emailbus.go` — Update() must apply the new field to the Email before calling storer.Update
- `business/domain/emailbus/stores/emaildb/emaildb.go` — SQL UPDATE SET clause must include the new column

### Storer interface (business/domain/emailbus/emailbus.go)
Adding or changing a method affects:
- `business/domain/emailbus/stores/emaildb/emaildb.go` — Store struct must implement the new method with the matching signature
- Any mock storer used in tests must implement the new method
- Business methods in emailbus.go that delegate to storer must call new methods as appropriate

### QueryFilter struct (business/domain/emailbus/filter.go)
Adding a filter field affects:
- `business/domain/emailbus/stores/emaildb/filter.go` — applyFilter() must add a new conditional block with the WHERE clause fragment and named param key
- `app/domain/emailapp/filter.go` — parseFilter() must parse the new query parameter from r.URL.Query() and set the filter field

### Order constants (business/domain/emailbus/order.go)
Adding a new OrderBy constant affects:
- `business/domain/emailbus/stores/emaildb/order.go` — orderByFields map must add the constant → SQL column name entry
- `app/domain/emailapp/order.go` — orderByFields map must add the request field string → business constant entry

## Routes

| Method | Path | Handler | Notes |
|--------|------|---------|-------|
| GET | /api/v1/emails | queryAll | Query params: `page`, `rows`, `context_id`, `from_address`, `subject`, `orderBy` (received_at, subject, created_at); default order: received_at DESC |
| GET | /api/v1/emails/{email_id} | queryByID | Fetches single email by UUID; returns 404 if not found |

All endpoints require Auth middleware (X-API-Key header validation). No write endpoints are exposed via HTTP — Create/Update/Delete are only called internally by the ingestion pipeline via emailbus.Business.

## Cross-Domain Dependencies

- **RawInput Domain** — Every email has a required foreign key `raw_input_id` referencing `raw_inputs(raw_input_id)`; the ingestion pipeline creates the raw_input record before creating the email record
- **Context Domain** — Emails have an optional `context_id` foreign key to `contexts(context_id)` with ON DELETE SET NULL; UpdateEmail allows linking or re-linking an email to a context after creation
- **Page SDK** (`business/sdk/page`) — queryAll uses page.Page for pagination (Offset, RowsPerPage, Number)
- **Order SDK** (`business/sdk/order`) — Query uses order.By with Field constant and Direction (ASC/DESC)
- **sqldb utilities** (`foundation/sqldb`) — Store uses NamedExecContext, NamedQuerySlice, NamedQueryStruct helpers; ErrDBNotFound (= sql.ErrNoRows) is returned by QueryByID/QueryByMessageID when no row matches and must be checked by callers
- **Error handling** (`app/sdk/errs`) — queryByID checks errors.Is(err, sqldb.ErrDBNotFound) and maps to errs.NotFound (HTTP 404); other errors map to errs.Internal (HTTP 500)
- **HTTP web framework** (`foundation/web`) — Handlers implement web.HandlerFunc signature and return web.Encoder; Email.Encode() satisfies the interface via json.Marshal
