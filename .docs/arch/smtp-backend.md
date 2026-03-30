# SMTP Backend System

> `smtpbus` is a pure business-layer SMTP receiver that wraps the `emersion/go-smtp` library. It accepts inbound email over SMTP, validates the recipient domain, reads the raw RFC 5322 message body, and hands the raw content directly to `ingestbus.Business.ProcessEmail()` for the full 10-step ingestion pipeline. There is no HTTP handler or REST route — the server runs as a standalone goroutine alongside the HTTP API, gated by `cfg.SMTP.Enabled`. Graceful shutdown is handled in `main.go` via `smtpSrv.Close()`. Auth is accepted unconditionally (single-user system).

## Core Types

```go
// business/domain/smtpbus/smtpbus.go

// Config holds SMTP server configuration.
type Config struct {
    Addr   string
    Domain string
}

// Server wraps the go-smtp server and connects it to the ingestion pipeline.
// Implements smtp.Backend (NewSession).
type Server struct {
    log       *logger.Logger
    ingestBus *ingestbus.Business
    smtpSrv   *smtp.Server
    domain    string
}

// session implements smtp.Session for a single SMTP transaction.
// Unexported — created per connection by NewSession().
type session struct {
    log       *logger.Logger
    ingestBus *ingestbus.Business
    domain    string
    from      string
    to        string
}
```

```go
// business/domain/ingestbus/extractor/anthropic.go
// (types consumed indirectly via ingestbus.ProcessEmail)

type ContextRef struct {
    ID    string `json:"id"`
    Title string `json:"title"`
}

type ActionItem struct {
    Title           string   `json:"title"`
    Description     string   `json:"description"`
    Priority        string   `json:"priority"`
    Interpretations []string `json:"interpretations,omitempty"`
}

type Deadline struct {
    Description string `json:"description"`
    Date        string `json:"date"`
    IsAmbiguous bool   `json:"is_ambiguous,omitempty"`
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
    ContextConfidence        float64      `json:"context_confidence,omitempty"`
    SuggestNewContext        bool         `json:"suggest_new_context,omitempty"`
    SuggestedContextTitle    string       `json:"suggested_context_title,omitempty"`
}

// Extractor defines the interface for email AI extraction.
type Extractor interface {
    ExtractEmail(ctx context.Context, subject, bodyText, fromAddress string, activeContexts []ContextRef) (EmailExtraction, error)
}
```

## File Map

### Business Layer — smtpbus

- `business/domain/smtpbus/smtpbus.go` — **NewServer()** — constructs `*smtp.Server` (go-smtp) with timeouts, max message size, max recipients; wires `Server` as the `smtp.Backend`
  - **ListenAndServe()** — starts the SMTP listener; blocking call, run in goroutine
  - **Close()** — graceful shutdown; called by `main.go` on SIGINT/SIGTERM
  - **NewSession()** — implements `smtp.Backend`; creates a per-connection `session`
  - **session.AuthPlain()** — accepts all credentials unconditionally
  - **session.Mail()** — captures sender address into `session.from`
  - **session.Rcpt()** — validates recipient domain against `cfg.Domain`; captures `session.to`
  - **session.Data()** — reads full message body into buffer; calls `ingestBus.ProcessEmail()`
  - **session.Reset()** — clears `from`/`to` for SMTP RSET command
  - **session.Logout()** — no-op

### Business Layer — ingestbus

- `business/domain/ingestbus/ingestbus.go` — **NewBusiness()** — constructs pipeline orchestrator wiring rawInputBus, emailBus, taskBus, contextBus, clarificationBus, extractor
  - **ProcessEmail(ctx, rawContent string) error** — 10-step pipeline: store raw_input → parse → dedup → store email → fetch contexts → AI extraction → context match → create tasks → create context event → mark processed
  - **Reprocess(ctx, rawInputID uuid.UUID) error** — re-runs pipeline on existing raw_input
  - **processRawInput()** — internal pipeline execution shared by ProcessEmail and Reprocess
  - **matchContextByKeywords()** — fuzzy keyword match against active context titles

- `business/domain/ingestbus/parse.go` — **parseEmail(rawContent string) (ParsedEmail, error)** — parses RFC 5322 message using `go-message/mail`; extracts MessageID, From, To, Subject, BodyText, BodyHTML
  - **parseEmailEntity()** — parses from a `*message.Entity` (alternate entry point)

### Business Layer — extractor

- `business/domain/ingestbus/extractor/anthropic.go` — defines `Extractor` interface, `ContextRef`, `ActionItem`, `Deadline`, `EmailExtraction` types
  - **NewAnthropicExtractor(apiKey, model string) *AnthropicExtractor** — constructs client
  - **ExtractEmail(ctx, subject, bodyText, fromAddress, activeContexts) (EmailExtraction, error)** — calls Anthropic Messages API with structured prompt; parses JSON response

- `business/domain/ingestbus/extractor/mock.go` — **MockExtractor** — test double implementing `Extractor`; returns configured `Result`/`Err`

### Wiring — main.go

- `api/services/planner/main.go` — conditionally constructs and starts `smtpbus.Server` when `cfg.SMTP.Enabled == true`; wires full dependency chain (rawinputdb → rawinputbus → emaildb → emailbus → taskdb → taskbus → contextdb → contextbus → extractor → ingestbus → smtpbus); runs `ListenAndServe()` in goroutine; calls `Close()` on shutdown

## Impact Callouts

### ⚠ smtpbus.Config (business/domain/smtpbus/smtpbus.go)
Changing this struct shape affects:
- `api/services/planner/main.go` — passes `smtpbus.Config{Addr: cfg.SMTP.Addr, Domain: cfg.SMTP.Domain}` to `NewServer()`
- Config env vars: `PLANNER_SMTP_ADDR` (default `:2525`) and `PLANNER_SMTP_DOMAIN` (default `localhost`) feed this struct

### ⚠ smtpbus.Server (business/domain/smtpbus/smtpbus.go)
Changing the Server struct or its public method signatures affects:
- `api/services/planner/main.go` — calls `NewServer()`, `ListenAndServe()`, `Close()`
- `smtpSrv.MaxMessageBytes` (10MB) and `smtpSrv.MaxRecipients` (5) are hardcoded in `NewServer()` — changing these requires no cross-file update but does affect email acceptance policy

### ⚠ session.Rcpt() domain validation (business/domain/smtpbus/smtpbus.go)
The domain check (`parts[1] != s.domain`) only fires when `domain != "" && domain != "localhost"`:
- Changing this logic affects which emails are accepted/rejected before reaching `ingestbus`
- No other file depends on this logic, but it is the only inbound filter before the pipeline fires

### ⚠ ingestbus.Business.ProcessEmail() (business/domain/ingestbus/ingestbus.go)
This is the single integration point between smtpbus and the rest of the system. Changes to its signature affect:
- `business/domain/smtpbus/smtpbus.go` — `session.Data()` calls `s.ingestBus.ProcessEmail(ctx, rawContent)`
- Any future callers (e.g., a webhook ingestion path) would also call this method

### ⚠ extractor.Extractor interface (business/domain/ingestbus/extractor/anthropic.go)
Adding or changing `ExtractEmail()` signature affects:
- `business/domain/ingestbus/ingestbus.go` — calls `b.extractor.ExtractEmail()`
- `business/domain/ingestbus/extractor/anthropic.go` — must implement the updated signature
- `business/domain/ingestbus/extractor/mock.go` — must implement the updated signature
- `api/services/planner/main.go` — passes `AnthropicExtractor` as `Extractor` to `ingestbus.NewBusiness()`

### ⚠ extractor.EmailExtraction (business/domain/ingestbus/extractor/anthropic.go)
Changing this struct affects:
- `business/domain/ingestbus/ingestbus.go` — reads all fields: `ActionItems`, `Deadlines`, `SuggestedContextID`, `ContextConfidence`, `SuggestNewContext`, `SuggestedContextTitle`, `SuggestedContextKeywords`, `Sentiment`, `Summary`
- `business/domain/ingestbus/extractor/anthropic.go` — Anthropic prompt schema must stay in sync with struct fields (JSON tags)
- `business/domain/ingestbus/extractor/mock.go` — `MockExtractor.Result` field is `EmailExtraction`
- Adding or renaming a field requires updating both the Go struct and the JSON schema embedded in the Anthropic prompt string

### ⚠ ingestbus.Business constructor (business/domain/ingestbus/ingestbus.go)
`NewBusiness()` takes 7 parameters. Adding a new dependency (e.g., a new bus) affects:
- `api/services/planner/main.go` — must instantiate and pass the new dependency in the SMTP wiring block

## Routes

No HTTP routes. `smtpbus` is a raw TCP server, not an HTTP handler.

| Protocol | Address | Config Key | Default |
|----------|---------|------------|---------|
| SMTP (TCP) | `cfg.SMTP.Addr` | `PLANNER_SMTP_ADDR` | `:2525` |

The server is started only when `PLANNER_SMTP_ENABLED=true`. When disabled, `smtpSrv` is `nil` and no SMTP goroutine runs.

## Cross-Domain Dependencies

- **ingestbus** — primary downstream; `session.Data()` calls `ingestBus.ProcessEmail()` which orchestrates the full pipeline
- **rawinputbus** — ingestbus stores every received email as a `raw_input` record before processing
- **emailbus** — ingestbus creates an `email` record after parsing; dedup check via `QueryByMessageID`
- **taskbus** — ingestbus creates tasks from AI-extracted action items
- **contextbus** — ingestbus queries active contexts for AI matching; may auto-create contexts; adds context events
- **clarificationbus** — ingestbus creates clarification items for low-confidence context matches, new contexts, ambiguous action items, and ambiguous deadlines
- **extractor.AnthropicExtractor** — calls Anthropic Messages API (claude-sonnet-4-20250514 default); requires `PLANNER_ANTHROPIC_API_KEY` env var
- **go-smtp** (`github.com/emersion/go-smtp`) — provides `smtp.Server`, `smtp.Backend`, `smtp.Session` interfaces
- **go-message** (`github.com/emersion/go-message`) — RFC 5322 email parser used in `ingestbus/parse.go`
- **foundation/logger** — structured logging in both `smtpbus` and `ingestbus`
