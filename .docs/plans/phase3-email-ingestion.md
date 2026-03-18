# Phase 3: Email Ingestion — Implementation Plan

## Summary

Forward an email, system extracts tasks and updates contexts automatically. Delivers: SMTP receiver (embedded in Go binary), email parsing, AI extraction via Anthropic API, two new database tables, two new CRUD domains, ingestion pipeline orchestrator, REST endpoints, MCP tools.

**Dependencies:** Phase 2 (Contexts) complete. No frontend dependency.
**New Go deps:** `github.com/emersion/go-smtp`, `github.com/emersion/go-message`, `github.com/anthropics/anthropic-sdk-go`

## Decisions

| Question | Decision |
|----------|----------|
| Anthropic SDK vs raw HTTP | Official `anthropic-sdk-go` SDK |
| SMTP port | TBD — need to check VPS inbound port 25 access |
| STARTTLS | Yes, reuse nginx wildcard cert |
| Sync vs async processing | Synchronous in SMTP handler |
| Context match confidence | Trust Claude's suggestion; Phase 3b clarification queue catches misses |
| Extraction model | Configurable via `PLANNER_ANTHROPIC_MODEL`, default `claude-sonnet-4-20250514` |

## Database Migrations

### Version 1.05 — raw_inputs

```sql
CREATE TABLE raw_inputs (
    raw_input_id  UUID        NOT NULL DEFAULT gen_random_uuid(),
    source_type   TEXT        NOT NULL CHECK (source_type IN ('email', 'transaction', 'voice', 'file')),
    status        TEXT        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'processed', 'failed')),
    raw_content   TEXT        NOT NULL,
    processed_at  TIMESTAMPTZ,
    error         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (raw_input_id)
);
CREATE INDEX idx_raw_inputs_status ON raw_inputs(status, created_at);
```

### Version 1.06 — emails

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

## New Enum Types

- `business/types/rawinputstatus/` — Pending, Processing, Processed, Failed
- `business/types/rawinputsource/` — Email, Transaction, Voice, File

## New Go Packages

### rawinputbus (standard CRUD domain)

```
business/domain/rawinputbus/
  model.go        — RawInput, NewRawInput, UpdateRawInput
  rawinputbus.go  — Storer interface + Business (Create, Update, Query, Count, QueryByID, MarkProcessing, MarkProcessed, MarkFailed)
  filter.go       — QueryFilter (Status, SourceType)
  order.go        — OrderByCreatedAt (default DESC), OrderByStatus
  stores/rawinputdb/
    model.go      — DB struct + converters
    rawinputdb.go — Store implementation
    filter.go     — applyFilter
    order.go      — orderByFields
```

### emailbus (standard CRUD domain)

```
business/domain/emailbus/
  model.go       — Email, NewEmail, UpdateEmail
  emailbus.go    — Storer interface + Business (Create, Update, Delete, Query, Count, QueryByID, QueryByMessageID)
  filter.go      — QueryFilter (ContextID, FromAddress, Subject ILIKE)
  order.go       — OrderByReceivedAt (default DESC), OrderBySubject, OrderByCreatedAt
  stores/emaildb/
    model.go     — DB struct + converters
    emaildb.go   — Store implementation
    filter.go    — applyFilter
    order.go     — orderByFields
```

### ingestbus (pipeline orchestrator — NOT standard CRUD)

```
business/domain/ingestbus/
  ingestbus.go   — Business struct, ProcessEmail(), Reprocess()
  parse.go       — parseEmail() using go-message
  extractor/
    anthropic.go — AnthropicExtractor implementing Extractor interface
```

**Extractor interface:**
```go
type Extractor interface {
    ExtractEmail(ctx context.Context, subject, bodyText, fromAddress string, activeContexts []ContextRef) (EmailExtraction, error)
}

type EmailExtraction struct {
    Summary                  string       `json:"summary"`
    SenderName               string       `json:"sender_name"`
    SenderDomain             string       `json:"sender_domain"`
    ActionItems              []ActionItem `json:"action_items"`
    Deadlines                []Deadline   `json:"deadlines"`
    SuggestedContextKeywords []string     `json:"suggested_context_keywords"`
    Sentiment                string       `json:"sentiment"`
    SuggestedContextID       *string      `json:"suggested_context_id,omitempty"`
}
```

### smtpbus (SMTP receiver)

```
business/domain/smtpbus/
  smtpbus.go — Server struct using emersion/go-smtp, implements Backend + Session interfaces
```

Embedded in Go binary alongside HTTP server. Listens on configurable port (default `:2525`). Validates recipient domain. Calls `ingestbus.ProcessEmail()` on DATA receipt.

## Pipeline Flow (10 steps)

1. Store raw_input (source_type=email, status=pending)
2. Parse email (headers + body via go-message)
3. Dedup check (Message-ID exists in emails table?)
4. Store email record
5. Fetch active contexts for AI prompt
6. AI extraction (Anthropic API → EmailExtraction)
7. Context matching (SuggestedContextID or keyword fuzzy match)
8. Create tasks (one per ActionItem, linked to matched context)
9. Create context event (kind=email, content=summary, source_id=email_id)
10. Mark raw_input processed

On failure at any step: mark raw_input failed with error message.

## API Endpoints

| Method | Path | Handler |
|--------|------|---------|
| GET | `/api/v1/emails` | queryAll (filters: context_id, from_address, subject) |
| GET | `/api/v1/emails/{email_id}` | queryByID |
| GET | `/api/v1/raw-inputs` | queryAll (filters: status, source_type) |
| GET | `/api/v1/raw-inputs/{raw_input_id}` | queryByID |
| POST | `/api/v1/raw-inputs/{raw_input_id}/reprocess` | reprocess |

## MCP Tools

- `list_emails` — filter by context_id, from, paginated
- `get_email` — full email detail including extraction results

## Config

```go
SMTP struct {
    Addr    string `conf:"default::2525"`
    Domain  string `conf:"default:localhost"`
    Enabled bool   `conf:"default:false"`
}
Anthropic struct {
    APIKey string `conf:"mask"`
    Model  string `conf:"default:claude-sonnet-4-20250514"`
}
```

## Docker Changes

No new container. SMTP runs inside backend container. Add port mapping:
```yaml
backend:
  ports:
    - "0.0.0.0:25:2525"
    - "0.0.0.0:587:2525"
  environment:
    PLANNER_SMTP_ENABLED: "true"
    PLANNER_SMTP_ADDR: ":2525"
    PLANNER_SMTP_DOMAIN: "mail.yourdomain.com"
    PLANNER_ANTHROPIC_API_KEY: "${ANTHROPIC_API_KEY}"
    PLANNER_ANTHROPIC_MODEL: "claude-sonnet-4-20250514"
```

DNS: A record for mail subdomain, MX record, SPF TXT record. Port 25 must be open inbound.

## Implementation Order

1. Database migrations (1.05 + 1.06)
2. Enum types (rawinputstatus, rawinputsource)
3. Raw input domain (full three-layer stack)
4. Email domain (full three-layer stack)
5. REST endpoints (rawinputapp + emailapp, register in main.go)
6. Email parsing (go-message, unit tests with sample RFC 5322)
7. AI extractor (anthropic-sdk-go, Extractor interface)
8. Ingestion pipeline orchestrator (ingestbus.ProcessEmail + Reprocess)
9. SMTP receiver (smtpbus, go-smtp)
10. Wire into main.go (config, goroutine, shutdown)
11. MCP tools (list_emails, get_email)
12. Integration test (forward real email end-to-end)
