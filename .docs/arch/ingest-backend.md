# Ingest Backend System

> Email and text ingestion pipeline: SMTP → raw input → parse → sanitize → AI extract → context match → task creation → clarifications. Orchestrated by `ingestbus.Business`, fed by `smtpbus.Server`, queried/reprocessed via `rawinputapp` HTTP handlers. AI extraction uses `foundation/claudecli` with model escalation (haiku → sonnet → opus) via `ClaudeCodeExtractor`, or a local Ollama instance via `OllamaExtractor`.

## Core Types

### Extractor Interface (`business/domain/ingestbus/extractor/model.go`)

```go
type Extractor interface {
    ExtractEmail(ctx context.Context, subject, bodyText, fromAddress string, activeContexts []ContextRef) (EmailExtraction, error)
    ExtractText(ctx context.Context, text string, activeContexts []ContextRef) (TextExtraction, error)
}

type ContextRef struct {
    ID    string `json:"id"`
    Title string `json:"title"`
}

type ActionItem struct {
    Title           string   `json:"title"`
    Description     string   `json:"description"`
    Priority        string   `json:"priority"`          // low|medium|high|urgent
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
    Sentiment                string       `json:"sentiment"`           // positive|neutral|negative|mixed
    SuggestedContextID       *string      `json:"suggested_context_id,omitempty"`
    ContextConfidence        float64      `json:"context_confidence,omitempty"`
    SuggestNewContext        bool         `json:"suggest_new_context,omitempty"`
    SuggestedContextTitle    string       `json:"suggested_context_title,omitempty"`
}

type ExtractedEvent struct {
    Title       string `json:"title"`
    Description string `json:"description,omitempty"`
    Location    string `json:"location,omitempty"`
    StartsAt    string `json:"starts_at"`
    EndsAt      string `json:"ends_at,omitempty"`
    AllDay      bool   `json:"all_day"`
    IsAmbiguous bool   `json:"is_ambiguous"`
}

type ExtractedNote struct {
    Content       string   `json:"content"`
    SuggestedTags []string `json:"suggested_tags,omitempty"`
}

type TextExtraction struct {
    Summary                  string           `json:"summary"`
    ActionItems              []ActionItem     `json:"action_items"`
    Deadlines                []Deadline       `json:"deadlines"`
    Events                   []ExtractedEvent `json:"events"`
    Notes                    []ExtractedNote  `json:"notes"`
    SuggestedContextKeywords []string         `json:"suggested_context_keywords"`
    SuggestedContextID       *string          `json:"suggested_context_id,omitempty"`
    ContextConfidence        float64          `json:"context_confidence,omitempty"`
    SuggestNewContext        bool             `json:"suggest_new_context,omitempty"`
    SuggestedContextTitle    string           `json:"suggested_context_title,omitempty"`
}
```

### Claude CLI Client (`foundation/claudecli/claudecli.go`)

```go
type Client struct {
    cliPath string        // default "claude"
    models  []string      // escalation chain, e.g. ["haiku", "sonnet", "opus"]
    timeout time.Duration // default 120s
    log     *logger.Logger
}

func NewClient(log *logger.Logger, cliPath string, models []string) *Client
func (c *Client) RunJSON(ctx context.Context, prompt string, schema string, dest any, shouldEscalate func() bool) error
```

### ParsedEmail (`business/domain/ingestbus/parse.go`)

```go
type ParsedEmail struct {
    MessageID   string
    FromAddress string
    FromName    string
    ToAddress   string
    Subject     string
    BodyText    string
    BodyHTML    string
}
```

### RawInput (`business/domain/rawinputbus/model.go`)

```go
type RawInput struct {
    ID          uuid.UUID
    SourceType  rawinputsource.Source
    Status      rawinputstatus.Status
    RawContent  string
    ProcessedAt *time.Time
    Error       *string
    CreatedAt   time.Time
}

type NewRawInput struct {
    SourceType rawinputsource.Source
    RawContent string
}

type UpdateRawInput struct {
    Status      *rawinputstatus.Status
    ProcessedAt *time.Time
    Error       *string
}
```

### Email (`business/domain/emailbus/model.go`)

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
```

### Sanitize (`business/sdk/sanitize/sanitize.go`)

```go
type Finding struct {
    Kind  string // SSN, PHONE, CREDIT_CARD, ROUTING_NUMBER, BANK_ACCOUNT
    Count int
}

type Result struct {
    Text     string
    Findings []Finding
}

func Sanitize(text string) Result
```

## File Map

### Extractor
- `business/domain/ingestbus/extractor/model.go` — `Extractor` interface, `EmailExtraction`, `TextExtraction`, `ActionItem`, `Deadline`, `ExtractedEvent`, `ExtractedNote`, `ContextRef` types
- `business/domain/ingestbus/extractor/claudecli.go` — **ClaudeCodeExtractor** — production implementation using Claude CLI with model escalation; escalates if zero action items AND confidence < 0.3
- `business/domain/ingestbus/extractor/ollama.go` — **OllamaExtractor** — local Ollama fallback; POSTs to `/api/generate` with `format:"json"`; drains body before returning on non-200 to allow connection reuse; sets `ContextConfidence=0.85` as a fixed policy (local models cannot reliably self-report confidence)
- `business/domain/ingestbus/extractor/prompt.go` — **BuildEmailExtractionPrompt()**, **BuildTextExtractionPrompt()** — shared prompt templates for email and text extraction
- `business/domain/ingestbus/extractor/mock.go` — **MockExtractor** — returns configured result/error for tests

### Foundation
- `foundation/claudecli/claudecli.go` — **Client.RunJSON()** — wraps `claude -p` with `--output-format json --json-schema --bare`; tries models in escalation order, calls `shouldEscalate()` callback after each parse

### Pipeline Core
- `business/domain/ingestbus/ingestbus.go` — **ProcessEmail()**, **Reprocess()** — 10-step pipeline: store → parse → dedup → persist email → fetch contexts → sanitize → AI extract → context match → create tasks/clarifications → mark processed
- `business/domain/ingestbus/parse.go` — **parseEmail()** — RFC 5322 parsing via `go-message`

### Business (dependencies)
- `business/domain/rawinputbus/rawinputbus.go` — **Create()**, **MarkProcessing()**, **MarkProcessed()**, **MarkFailed()**, **Query()**, **QueryByID()** — raw input lifecycle
- `business/domain/rawinputbus/model.go` — `RawInput`, `NewRawInput`, `UpdateRawInput`
- `business/domain/emailbus/emailbus.go` — **Create()**, **QueryByMessageID()** — email persistence + dedup
- `business/domain/emailbus/model.go` — `Email`, `NewEmail`, `UpdateEmail`

### Store
- `business/domain/rawinputbus/stores/rawinputdb/rawinputdb.go` — SQL INSERT/UPDATE/SELECT on `raw_inputs`
- `business/domain/emailbus/stores/emaildb/emaildb.go` — SQL INSERT/UPDATE/SELECT on `emails`

### Handlers
- `app/domain/rawinputapp/rawinputapp.go` — **queryAll()**, **queryByID()**, **reprocess()** — HTTP handlers
- `app/domain/rawinputapp/route.go` — **Routes.Add()** — wires all dependencies, creates `ClaudeCodeExtractor` from `cfg.ClaudeCLI`
- `app/domain/rawinputapp/model.go` — app-layer `RawInput` DTO + `toAppRawInput()` converter
- `app/domain/rawinputapp/filter.go` — `parseFilter()` for `status`, `source_type`
- `app/domain/rawinputapp/order.go` — `parseOrder()` for `created_at`, `status`

### SMTP
- `business/domain/smtpbus/smtpbus.go` — **NewServer()**, **ListenAndServe()**, **Close()** — SMTP server feeding `ingestBus.ProcessEmail()`

### Sanitize
- `business/sdk/sanitize/sanitize.go` — **Sanitize()** — PII redaction (SSN, phone, credit card, routing, bank account)

## Impact Callouts

### ⚠ EmailExtraction (`extractor/model.go`)
Changing this struct shape affects:
- `extractor/claudecli.go` — JSON schema constant must match struct fields; `shouldEscalate` reads `ActionItems` and `ContextConfidence`
- `extractor/prompt.go` — prompt instructs Claude to return JSON matching this schema
- `extractor/mock.go` — `MockExtractor.Result` is this type
- `ingestbus/ingestbus.go` — reads `SuggestedContextID`, `ContextConfidence`, `SuggestNewContext`, `SuggestedContextTitle`, `ActionItems`, `Deadlines`, `SuggestedContextKeywords` to drive context matching + task/clarification creation

### ⚠ Extractor interface (`extractor/model.go`)
Adding/changing a method affects:
- `extractor/claudecli.go` — must implement
- `extractor/ollama.go` — must implement
- `extractor/mock.go` — must implement
- `ingestbus/ingestbus.go` — calls `ExtractEmail()` in step 6
- `rawinputapp/route.go` — constructs the extractor
- `api/services/planner/main.go` — constructs the extractor for SMTP path

### ⚠ claudecli.Client (`foundation/claudecli/claudecli.go`)
Changing the Client API affects:
- `extractor/claudecli.go` — calls `client.RunJSON()`
- `threadbus/claudecli_extractor.go` — calls `client.RunJSON()`
- `app/sdk/mux/mux.go` — `Config.ClaudeCLI` field carries the client
- `rawinputapp/route.go` — reads `cfg.ClaudeCLI`
- `api/services/planner/main.go` — constructs with `claudecli.NewClient()`

### ⚠ RawInput (`rawinputbus/model.go`)
Changing this struct shape affects:
- `rawinputbus/rawinputbus.go` — all CRUD methods
- `rawinputdb/rawinputdb.go` — SQL columns, `Scan()` field list
- `rawinputdb/model.go` — DB struct + `toBusRawInput()` converter
- `rawinputapp/model.go` — app DTO + `toAppRawInput()` converter
- `ingestbus/ingestbus.go` — creates and updates raw inputs through pipeline
- Migration required if DB column added/removed

### ⚠ Email (`emailbus/model.go`)
Changing this struct shape affects:
- `emailbus/emailbus.go` — all CRUD methods
- `emaildb/emaildb.go` — SQL columns, `Scan()` field list
- `emaildb/model.go` — DB struct + `toBusEmail()` converter
- `emailapp/model.go` — app DTO + `toAppEmail()` converter
- `ingestbus/ingestbus.go` — creates emails with parsed fields + context assignment
- Migration required if DB column added/removed

### ⚠ rawinputbus.Storer interface (`rawinputbus/rawinputbus.go`)
Adding/changing a method affects:
- `rawinputdb/rawinputdb.go` — must implement
- `rawinputbus/rawinputbus.go` — calls the method
- `ingestbus/ingestbus.go` — uses rawinputbus.Business which wraps Storer

## Routes

| Method | Path | Handler | Auth |
|--------|------|---------|------|
| GET | `/api/v1/raw-inputs` | queryAll | API key |
| GET | `/api/v1/raw-inputs/{raw_input_id}` | queryByID | API key |
| POST | `/api/v1/raw-inputs/{raw_input_id}/reprocess` | reprocess | API key |

## Cross-Domain Dependencies

- **taskbus** — `ingestbus` creates tasks from extracted action items
- **contextbus** — `ingestbus` queries active contexts for AI prompt, matches emails to contexts, auto-creates new contexts
- **clarificationbus** — `ingestbus` creates `context_assignment`, `ambiguous_action`, `ambiguous_deadline`, `new_context` clarifications
- **sanitize** — PII redaction before sending to Claude CLI
- **claudecli** — foundation package wrapping `claude -p` for inference
- **smtpbus** — SMTP server calls `ingestbus.ProcessEmail()` for incoming mail
- **go-message** — RFC 5322 email parsing
- **go-smtp** — SMTP server library

## Configuration

| Env var | Default | Purpose |
|---------|---------|---------|
| `PLANNER_SMTP_ENABLED` | `false` | Enable SMTP listener |
| `PLANNER_SMTP_ADDR` | `:2525` | SMTP listen address |
| `PLANNER_SMTP_DOMAIN` | `localhost` | Domain for RCPT TO validation |
| `PLANNER_CLAUDE_CLI_PATH` | `claude` | Path to Claude CLI binary |
| `PLANNER_CLAUDE_MODELS` | `haiku,sonnet,opus` | Model escalation chain |

## Pipeline Steps

1. Store raw content → `raw_inputs` (status: pending)
2. Mark processing → status: processing
3. Parse RFC 5322 headers/body
4. Dedup check via `emails.message_id` unique index
5. Store parsed email → `emails`
6. Fetch active contexts for AI prompt
7. Sanitize subject/body (PII redaction)
8. AI extraction via Claude CLI → `EmailExtraction`
9. Context matching: by suggested UUID → keyword fuzzy → auto-create if suggested
10. Create tasks from action items; create clarifications for ambiguities
11. Mark processed or failed
