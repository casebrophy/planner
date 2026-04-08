# Email Backend System

> Read-only email domain for querying and linking ingested emails to contexts. Emails are created through the raw input ingestion pipeline via `raw_inputs` table, then extracted and stored as structured email records with optional context associations. Updates are limited to context linking only.

## Core Types

### Business Layer

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

type UpdateEmail struct {
	ContextID *uuid.UUID
}

type QueryFilter struct {
	ContextID   *uuid.UUID
	FromAddress *string
	Subject     *string  // ILIKE substring match
}

const (
	OrderByReceivedAt = "received_at"
	OrderBySubject    = "subject"
	OrderByCreatedAt  = "created_at"
)

var DefaultOrderBy = order.NewBy(OrderByReceivedAt, order.DESC)
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

### App Layer DTO

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

### Store Layer Model

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

## File Map

### App Layer (app/domain/emailapp/)
- `emailapp.go` — **queryAll()** paginated list with filters/ordering; **queryByID()** fetch single email
- `model.go` — Email DTO + **toAppEmail()**, **toAppEmails()** converters
- `route.go` — **Routes.Add()** registers GET /api/v1/emails and GET /api/v1/emails/{email_id}
- `filter.go` — **parseFilter()** parses (context_id, from_address, subject) → QueryFilter
- `order.go` — **parseOrder()** maps (received_at, subject, created_at) → order.By

### Business Layer (business/domain/emailbus/)
- `emailbus.go` — **Create/Update/Delete/Query/Count/QueryByID/QueryByMessageID** + Storer interface
- `model.go` — Email, NewEmail, UpdateEmail domain types
- `filter.go` — QueryFilter struct (ContextID, FromAddress, Subject)
- `order.go` — Order constants + DefaultOrderBy (received_at DESC)

### Store Layer (business/domain/emailbus/stores/emaildb/)
- `emaildb.go` — SQL implementation for all Storer methods
- `model.go` — emailDB struct + **toDBEmail()**, **toBusEmail()**, **toBusEmails()** converters
- `filter.go` — **applyFilter()** WHERE clauses: context_id =, from_address =, subject ILIKE %?%
- `order.go` — orderByFields map; **orderByClause()** maps constants → SQL columns

## Impact Callouts

### ⚠ Email struct (business/domain/emailbus/model.go)
Changing Email fields affects:
- `emailapp/model.go` — app DTO + toAppEmail() converter
- `emaildb/model.go` — emailDB struct tags + toDBEmail/toBusEmail converters
- `emaildb/emaildb.go` — INSERT/SELECT SQL column lists
- Migration SQL for schema changes

### ⚠ QueryFilter struct (business/domain/emailbus/filter.go)
Adding filter fields requires:
- `emailapp/filter.go` — new query param parsing
- `emaildb/filter.go` — new WHERE clause logic

### ⚠ Order constants (business/domain/emailbus/order.go)
Adding order fields requires:
- `emailbus/order.go` — new constant
- `emaildb/order.go` — new entry in orderByFields map (const → SQL column)
- `emailapp/order.go` — new entry in orderByFields map (request param → const)

### ⚠ Storer interface (business/domain/emailbus/emailbus.go)
Adding new methods requires:
- `emaildb/emaildb.go` — implementation
- Handlers in emailapp.go if API exposure needed

### ⚠ QueryByMessageID (business/domain/emailbus/emailbus.go)
Unique constraint on message_id allows deduplication. MessageID is immutable (set at creation, never updated). Callers must use this before creating to prevent duplicates.

## Routes

| Method | Path | Handler |
|--------|------|---------|
| GET | /api/v1/emails | queryAll — paginated; filters: context_id, from_address, subject; orderBy: received_at, subject, created_at |
| GET | /api/v1/emails/{email_id} | queryByID — returns 404 if not found |

All routes require `X-API-Key` header authentication.

## Cross-Domain Dependencies

- **clarificationapp** — updates email context_id when resolving context assignment clarifications
- **mcpapp** — lists emails via Query (MCP tool endpoint)
- **raw_inputs** — emailbus.Email.RawInputID links to raw_inputs(raw_input_id) [immutable FK]
- **contexts** — emailbus.Email.ContextID optionally links to contexts(context_id) [nullable FK, SET NULL on context delete]
