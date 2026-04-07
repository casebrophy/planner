# Ingest Backend System

> Email and text ingestion pipeline: SMTP / HTTP → raw input → parse → sanitize → AI extract → context match → task/event/note creation → clarifications. Orchestrated by `ingestbus.Business` (no store layer -- pure orchestrator over other domains). Fed by `smtpbus.Server` (email), `voiceingestapp` HTTP handler (voice/text), and a background `IngestWorker` that retries pending items. AI extraction uses `foundation/claudecli` with model escalation (haiku → sonnet → opus) via `ClaudeCodeExtractor`, a local Ollama instance via `OllamaExtractor`, or a `FailoverExtractor` that tries Claude first and falls back to Ollama on rate-limit / context-limit / connection errors. Clarification `AnswerOptions` JSON is written using typed structs from `clarificationbus/options.go` (`ContextAssignmentOptions`, `NewContextOptions`, `AmbiguousActionOptions`, `AmbiguousDeadlineOptions`).

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

### IngestResult (`business/domain/ingestbus/ingestbus.go`)

```go
type IngestResult struct {
    TaskIDs  []uuid.UUID
    EventIDs []uuid.UUID
    NoteIDs  []uuid.UUID
}
```

### Business (`business/domain/ingestbus/ingestbus.go`)

```go
type Business struct {
    log              *logger.Logger
    rawInputBus      *rawinputbus.Business
    emailBus         *emailbus.Business
    taskBus          *taskbus.Business
    contextBus       *contextbus.Business
    clarificationBus *clarificationbus.Business
    eventBus         *eventbus.Business
    extractor        extractor.Extractor
    noteBus          *notebus.Business
    tagBus           *tagbus.Business
}

func NewBusiness(
    log *logger.Logger,
    rawInputBus *rawinputbus.Business,
    emailBus *emailbus.Business,
    taskBus *taskbus.Business,
    contextBus *contextbus.Business,
    clarificationBus *clarificationbus.Business,
    eventBus *eventbus.Business,
    ext extractor.Extractor,
    noteBus *notebus.Business,
    tagBus *tagbus.Business,
) *Business

func (b *Business) ProcessEmail(ctx context.Context, rawContent string) error
func (b *Business) ProcessText(ctx context.Context, rawContent string) (IngestResult, error)
func (b *Business) Reprocess(ctx context.Context, rawInputID uuid.UUID) error
func (b *Business) EnqueueEmail(ctx context.Context, rawContent string) (uuid.UUID, error)
func (b *Business) EnqueueText(ctx context.Context, rawContent string) (uuid.UUID, error)
func (b *Business) ProcessRawInputByID(ctx context.Context, id uuid.UUID) error
```

**Notes:**
- **ProcessEmail** / **ProcessText** are synchronous; they block until the full pipeline completes.
- **EnqueueEmail** / **EnqueueText** are async queueing methods; they store a raw_input and return its ID immediately.
- **ProcessRawInputByID** is the worker entry point; dispatches to `processRawInput` (email) or `processTextInput` (voice) based on `SourceType`; returns error WITHOUT calling `MarkFailed` -- the caller (worker) decides retry vs. terminal.
- **Reprocess** fetches an existing raw_input by ID, marks it processing, and re-runs the pipeline. On failure it calls `MarkFailed` itself.

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

func parseEmail(rawContent string) (ParsedEmail, error)       // from raw RFC 5322 string
func parseEmailEntity(entity *message.Entity) (ParsedEmail, error)  // from go-message Entity
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
    RetryCount  int
    NextRetryAt *time.Time
    MaxRetries  int
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
    RetryCount  *int
    NextRetryAt *time.Time
}
```

### VoiceIngest App DTOs (`app/domain/voiceingestapp/model.go`)

```go
type ingestRequest struct {
    Text string `json:"text"`
}

type ingestResponse struct {
    RawInputID string `json:"rawInputId"`
}

func (r ingestResponse) Encode() ([]byte, string, error)
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

### Clarification Option Types (`business/domain/clarificationbus/options.go`)

```go
type ContextRef struct {
    ID    string `json:"id"`
    Title string `json:"title"`
}

type ContextAssignmentOptions struct {
    SuggestedContext  string       `json:"suggested_context"`
    Confidence        float64      `json:"confidence"`
    AvailableContexts []ContextRef `json:"available_contexts"`
}

type NewContextOptions struct {
    ContextID string `json:"context_id"`
    Title     string `json:"title"`
}

type AmbiguousActionOptions struct {
    Interpretations []string `json:"interpretations"`
}

type AmbiguousDeadlineOptions struct {
    Description string `json:"description"`
    RawDate     string `json:"raw_date"`
}
```

**Note:** `clarificationbus.ContextRef` is structurally identical to `extractor.ContextRef` but is a separate type to avoid import cycles. `ingestbus` builds `[]extractor.ContextRef` for the AI extraction call, then converts to `[]clarificationbus.ContextRef` for writing `AnswerOptions` JSON.

### IngestWorker (`business/sdk/worker/ingestworker.go`)

```go
type RawInputQueuer interface {
    QueryRetryable(ctx context.Context, limit int) ([]rawinputbus.RawInput, error)
    MarkForRetry(ctx context.Context, ri rawinputbus.RawInput, errMsg string) (rawinputbus.RawInput, error)
    MarkFailed(ctx context.Context, ri rawinputbus.RawInput, errMsg string) (rawinputbus.RawInput, error)
}

type RawInputProcessor interface {
    ProcessRawInputByID(ctx context.Context, id uuid.UUID) error
}

type IngestWorker struct {
    log       *logger.Logger
    riBus     RawInputQueuer    // rawinputbus.Business
    igBus     RawInputProcessor // ingestbus.Business
    interval  time.Duration     // 30s
    batchSize int               // 20
}

func NewIngestWorker(log *logger.Logger, riBus RawInputQueuer, igBus RawInputProcessor) *IngestWorker
func (w *IngestWorker) Run(ctx context.Context)          // blocks until ctx cancelled
func (w *IngestWorker) ProcessBatch(ctx context.Context)  // exported for tests
```

## File Map

### App Layer (HTTP Handlers)
- `app/domain/voiceingestapp/voiceingestapp.go` -- **ingest()** -- decodes `ingestRequest`, validates text non-empty, calls `ingestBus.EnqueueText()`, returns `ingestResponse` with raw_input ID
- `app/domain/voiceingestapp/model.go` -- **ingestRequest**, **ingestResponse** -- request/response DTOs with JSON tags and `Encode()` method
- `app/domain/voiceingestapp/route.go` -- **Routes.Add()** -- wires all 8 domain dependencies (rawinput, email, task, context, clarification, event, note, tag), creates `ClaudeCodeExtractor`, registers POST `/api/v1/ingest/voice` with auth middleware

### Business Layer (Pipeline Core)
- `business/domain/ingestbus/ingestbus.go` -- **ProcessEmail()**, **ProcessText()**, **Reprocess()**, **EnqueueEmail()**, **EnqueueText()**, **ProcessRawInputByID()**, **processRawInput()**, **processTextInput()**, **matchContextByKeywords()** -- 10-step pipeline orchestrator; no store layer (orchestrates other domains)
- `business/domain/ingestbus/parse.go` -- **parseEmail()**, **parseEmailEntity()** -- RFC 5322 parsing via `emersion/go-message`; extracts MessageID, From, To, Subject, BodyText, BodyHTML from MIME parts
- `business/domain/ingestbus/ingestbus_test.go` -- **Test_Ingest** -- 6 test cases: empty email extraction, email creates task + raw_input, empty text extraction, text creates task, text with context match, text creates event

### Extractor Implementations
- `business/domain/ingestbus/extractor/model.go` -- **Extractor** interface, **ContextRef**, **ActionItem**, **Deadline**, **EmailExtraction**, **ExtractedEvent**, **ExtractedNote**, **TextExtraction** types
- `business/domain/ingestbus/extractor/claudecli.go` -- **ClaudeCodeExtractor** -- production implementation using Claude CLI with model escalation and JSON schema validation; escalation callback: escalates if zero action items AND confidence < 0.3 (email) or zero action items (text)
- `business/domain/ingestbus/extractor/ollama.go` -- **OllamaExtractor** -- local Ollama fallback; POSTs to `/api/generate` with `format:"json"` and 30s timeout; drains body on non-200; fixes `ContextConfidence=0.85` (local models cannot reliably self-report)
- `business/domain/ingestbus/extractor/prompt.go` -- **BuildEmailExtractionPrompt()**, **BuildTextExtractionPrompt()** -- shared prompt templates; text prompt includes current time, timezone, and UTC conversion instructions
- `business/domain/ingestbus/extractor/failover.go` -- **FailoverExtractor** -- wraps `*ClaudeCodeExtractor` (primary) + `*OllamaExtractor` (fallback); `isFallbackError()` triggers on "429", "context"+"limit", "connection", "timeout", "refused"; `newFailoverExtractorForTest()` package-private helper accepts interfaces
- `business/domain/ingestbus/extractor/mock.go` -- **MockExtractor** -- returns configured `Result` (email) or `TextResult` (text) or `Err` for tests
- `business/domain/ingestbus/extractor/failover_test.go` -- 7 tests: Claude success (sentinel ensures Ollama not called), 429 triggers fallback, context-limit triggers fallback, connection-refused triggers fallback, 400 does NOT trigger fallback, both fail returns Ollama error, ExtractText fallback works
- `business/domain/ingestbus/extractor/ollama_test.go` -- 4 tests: successful email/text extraction via httptest server, HTTP 500 error, malformed inner JSON

### Background Worker
- `business/sdk/worker/ingestworker.go` -- **IngestWorker** -- polls every 30s for retryable raw_inputs (batch of 20); dispatches each in a goroutine with 3-minute timeout; on failure: if `RetryCount+1 >= MaxRetries` calls `MarkFailed`, else calls `MarkForRetry`

### SMTP Server
- `business/domain/smtpbus/smtpbus.go` -- **Server**, **session** -- SMTP server implementing `smtp.Backend`; `session.Data()` reads email body and calls `ingestBus.ProcessEmail()`; accepts email even on pipeline failure (stored as failed raw_input); validates recipient domain; 10MB max message, 5 max recipients

### Foundation
- `foundation/claudecli/claudecli.go` -- **Client.RunJSON()** -- wraps `claude -p` with `--output-format json --json-schema --bare`; tries models in escalation order, calls `shouldEscalate()` callback after each parse

### Wiring
- `api/services/planner/main.go` -- constructs `igBus` with all 8 domain deps + extractor; passes `igBus` to `smtpbus.NewServer()` and `worker.NewIngestWorker()`; worker runs in background goroutine

## Impact Callouts

### -- EmailExtraction (`business/domain/ingestbus/extractor/model.go`)
Changing this struct shape affects:
- `extractor/claudecli.go` -- `emailExtractionSchema` JSON schema constant must match struct fields exactly; `shouldEscalate` reads `.ActionItems` length and `.ContextConfidence`
- `extractor/ollama.go` -- `json.Unmarshal` into this struct; hardcodes `.ContextConfidence = 0.85` post-parse
- `extractor/prompt.go` -- `BuildEmailExtractionPrompt` instructs Claude to return JSON matching this schema
- `extractor/mock.go` -- `MockExtractor.Result` field is this type
- `extractor/failover.go` -- delegates and returns this type from `ExtractEmail()`
- `ingestbus/ingestbus.go:processRawInput` -- reads `.SuggestedContextID`, `.ContextConfidence`, `.SuggestNewContext`, `.SuggestedContextTitle`, `.ActionItems[].Title/Description/Priority/Interpretations`, `.Deadlines[].IsAmbiguous/Date/Description`, `.SuggestedContextKeywords`, `.Sentiment`, `.Summary`

### -- TextExtraction (`business/domain/ingestbus/extractor/model.go`)
Changing this struct shape affects:
- `extractor/claudecli.go` -- `textExtractionSchema` JSON schema constant must match; `shouldEscalate` reads `.ActionItems` length
- `extractor/ollama.go` -- `json.Unmarshal` into this struct; hardcodes `.ContextConfidence = 0.85`
- `extractor/prompt.go` -- `BuildTextExtractionPrompt` instructs Claude to return JSON matching this schema
- `extractor/mock.go` -- `MockExtractor.TextResult` field is this type
- `extractor/failover.go` -- delegates and returns this type from `ExtractText()`
- `ingestbus/ingestbus.go:processTextInput` -- reads all fields from `EmailExtraction` callout above, plus `.Events[].Title/Description/Location/StartsAt/EndsAt/AllDay`, `.Notes[].Content/SuggestedTags`
- `app/domain/noteapp/noteapp.go` -- calls `extractor.ExtractText()` for note auto-tag/context suggestion
- `app/domain/eventapp/eventapp.go` -- calls `extractor.ExtractText()` for event auto-extraction from text
- `app/domain/classifyapp/classifyapp.go` -- calls `extractor.ExtractText()` for task classification
- `app/domain/mcpapp/mcpapp.go` -- calls `extractor.ExtractText()` in background goroutine for MCP classify

### -- Extractor interface (`business/domain/ingestbus/extractor/model.go`)
Adding/changing a method affects:
- `extractor/claudecli.go` -- must implement
- `extractor/ollama.go` -- must implement
- `extractor/failover.go` -- must implement (delegates primary then fallback)
- `extractor/mock.go` -- must implement
- `ingestbus/ingestbus.go` -- calls `ExtractEmail()` and `ExtractText()` via interface
- `voiceingestapp/route.go` -- constructs `ClaudeCodeExtractor` and stores as `extractor.Extractor`
- `noteapp/route.go` -- constructs `ClaudeCodeExtractor` and stores as `extractor.Extractor`
- `eventapp/route.go` -- constructs `ClaudeCodeExtractor` and stores as `extractor.Extractor`
- `classifyapp/route.go` -- constructs `ClaudeCodeExtractor` and stores as `extractor.Extractor`
- `mcpapp/route.go` -- constructs `ClaudeCodeExtractor` and stores as `extractor.Extractor`
- `api/services/planner/main.go` -- constructs extractor for SMTP path

### -- FailoverExtractor (`business/domain/ingestbus/extractor/failover.go`)
Changing fallback trigger logic (`isFallbackError`) affects:
- `extractor/failover_test.go` -- 7 tests cover exact trigger conditions; update test cases if trigger rules change
- `ingestbus/ingestbus.go` -- soft-failure behaviour: extraction errors that don't trigger fallback are swallowed (pipeline continues without AI features)
- `NewFailoverExtractor` accepts `*ClaudeCodeExtractor` and `*OllamaExtractor` (concrete types) to prevent accidental nesting -- wiring must pass concrete pointers, not interface values

### -- clarificationbus option types (`business/domain/clarificationbus/options.go`)
Changing any option struct field affects:
- `ingestbus/ingestbus.go` -- both `processRawInput` (email path) and `processTextInput` (voice/text path) marshal `ContextAssignmentOptions`, `NewContextOptions`, `AmbiguousActionOptions`, `AmbiguousDeadlineOptions` into `AnswerOptions` JSON; field renames silently produce wrong JSON keys
- `app/domain/classifyapp/classifyapp.go` -- marshals `ContextAssignmentOptions` for low-confidence task classification
- `app/domain/mcpapp/mcpapp.go` -- marshals `ContextAssignmentOptions` in background goroutine for MCP classify tool
- Frontend `ClarificationCard` component -- deserializes `answer_options` JSON per clarification kind; JSON field renames break the UI

### -- IngestResult (`business/domain/ingestbus/ingestbus.go`)
Changing this struct affects:
- `ingestbus/ingestbus.go:processTextInput` -- builds and returns result with `TaskIDs`, `EventIDs`, `NoteIDs`
- `ingestbus/ingestbus_test.go` -- asserts on `result.TaskIDs`, `result.EventIDs` lengths

### -- RawInput (`business/domain/rawinputbus/model.go`)
Changing this struct shape affects:
- `rawinputbus/rawinputbus.go` -- all CRUD methods including `MarkProcessing()`, `MarkProcessed()`, `MarkFailed()`, `MarkForRetry()`, `QueryRetryable()`
- `rawinputdb/rawinputdb.go` -- SQL columns, `Scan()` field list
- `rawinputdb/model.go` -- DB struct + `toBusRawInput()` converter
- `rawinputapp/model.go` -- app DTO + `toAppRawInput()` converter
- `ingestbus/ingestbus.go` -- creates and updates raw inputs throughout pipeline; reads `ri.RawContent`, `ri.ID`, `ri.SourceType`, `ri.RetryCount`, `ri.MaxRetries`
- `worker/ingestworker.go` -- reads `ri.ID`, `ri.RetryCount`, `ri.MaxRetries` for retry logic
- Migration SQL required if DB column added/removed

### -- IngestWorker interfaces (`business/sdk/worker/ingestworker.go`)
Changing `RawInputQueuer` or `RawInputProcessor` affects:
- `rawinputbus/rawinputbus.go` -- must satisfy `RawInputQueuer` (provides `QueryRetryable`, `MarkForRetry`, `MarkFailed`)
- `ingestbus/ingestbus.go` -- must satisfy `RawInputProcessor` (provides `ProcessRawInputByID`)
- `api/services/planner/main.go` -- passes `riBus` and `igBus` to `NewIngestWorker()`

### -- claudecli.Client (`foundation/claudecli/claudecli.go`)
Changing the Client API affects:
- `extractor/claudecli.go` -- calls `client.RunJSON()`
- `app/sdk/mux/mux.go` -- `Config.ClaudeCLI` field carries the client
- `voiceingestapp/route.go` -- reads `cfg.ClaudeCLI`
- `noteapp/route.go` -- reads `cfg.ClaudeCLI`
- `eventapp/route.go` -- reads `cfg.ClaudeCLI`
- `classifyapp/route.go` -- reads `cfg.ClaudeCLI`
- `mcpapp/route.go` -- reads `cfg.ClaudeCLI`
- `api/services/planner/main.go` -- constructs with `claudecli.NewClient()`

## Routes

| Method | Path | Handler | Auth |
|--------|------|---------|------|
| POST | `/api/v1/ingest/voice` | `voiceingestapp.ingest` | API key |

**Non-HTTP entry points:**
- SMTP listener (`smtpbus.Server`) calls `ingestBus.ProcessEmail()` on incoming mail
- Background `IngestWorker` calls `ingestBus.ProcessRawInputByID()` every 30s for pending/retryable items

## Cross-Domain Dependencies

| Domain | How ingestbus uses it |
|--------|-----------------------|
| **rawinputbus** | Create raw_input (Step 1), MarkProcessing/MarkProcessed/MarkFailed lifecycle |
| **emailbus** | Store parsed email record, dedup via `QueryByMessageID()`, update email with matched context |
| **taskbus** | Create tasks from `extraction.ActionItems` with priority/status/energy/context |
| **contextbus** | Query active contexts for AI prompt, verify suggested context exists, auto-create new contexts, add context events |
| **clarificationbus** | Create `context_assignment`, `ambiguous_action`, `ambiguous_deadline`, `new_context` clarifications using typed option structs |
| **eventbus** | Create events from `extraction.Events` (text pipeline only) with parsed start/end times and location |
| **notebus** | Create notes from `extraction.Notes` (text pipeline only) with content, source, raw_input_id, context |
| **tagbus** | Query existing tags by name, create new tags, link tags to notes via `AddToNote()` (text pipeline only) |
| **smtpbus** | SMTP server calls `ProcessEmail()` for incoming mail |
| **sanitize** | PII redaction (SSN, phone, credit card, routing, bank account) before sending to external AI |
| **claudecli** | Foundation package wrapping `claude -p` for inference with model escalation |
| **go-message** | RFC 5322 email parsing (`emersion/go-message`) |
| **go-smtp** | SMTP server library (`emersion/go-smtp`) |

## Configuration

| Env var | Default | Purpose |
|---------|---------|---------|
| `PLANNER_SMTP_ENABLED` | `false` | Enable SMTP listener |
| `PLANNER_SMTP_ADDR` | `:2525` | SMTP listen address |
| `PLANNER_SMTP_DOMAIN` | `localhost` | Domain for RCPT TO validation |
| `PLANNER_CLAUDE_CLI_PATH` | `claude` | Path to Claude CLI binary |
| `PLANNER_CLAUDE_MODELS` | `haiku,sonnet,opus` | Model escalation chain |
| `PLANNER_OLLAMA_URL` | (empty) | Ollama server URL for fallback extraction |
| `PLANNER_OLLAMA_MODEL` | (empty) | Ollama model name (e.g. `llama3`) |
| `PLANNER_OLLAMA_ENABLED` | `false` | Enable Ollama fallback |

## Pipeline Steps

### Email Path (`ProcessEmail` / `processRawInput`)
1. **Store raw_input** -- `rawinputbus.Create(Email, rawContent)` -- status: pending
2. **Mark processing** -- `rawinputbus.MarkProcessing()` -- status: processing
3. **Parse RFC 5322** -- `parseEmail(rawContent)` -- extracts headers + MIME body parts
4. **Dedup check** -- `emailbus.QueryByMessageID()` -- if found, mark processed and return
5. **Store email** -- `emailbus.Create()` -- persists parsed fields
6. **Fetch active contexts** -- `contextbus.Query(Status=Active, limit 50)` -- build `[]ContextRef` for AI
7. **Sanitize** -- `sanitize.Sanitize(subject)` + `sanitize.Sanitize(body)` -- PII redaction
8. **AI extraction** -- `extractor.ExtractEmail()` -- returns `EmailExtraction`; on error, marks processed and returns (soft failure)
9. **Context matching** -- suggested UUID first, keyword fuzzy match fallback, auto-create context if `SuggestNewContext=true`; creates `new_context` clarification for auto-created contexts; creates `context_assignment` clarification if confidence < 0.7
10. **Create tasks** -- one task per `ActionItem` with mapped priority; creates `ambiguous_action` clarification for items with multiple interpretations
11. **Create deadline clarifications** -- `ambiguous_deadline` clarification for `Deadline.IsAmbiguous=true`
12. **Update email context** -- `emailbus.Update()` with matched context ID
13. **Create context event** -- `contextbus.AddEvent(kind="email")` with email metadata
14. **Mark processed** -- `rawinputbus.MarkProcessed()`

### Text Path (`ProcessText` / `processTextInput`)
1. **Store raw_input** -- `rawinputbus.Create(Voice, rawContent)` -- status: pending
2. **Mark processing** -- status: processing
3. **Fetch active contexts** -- same as email path
4. **Sanitize** -- `sanitize.Sanitize(rawContent)`
5. **AI extraction** -- `extractor.ExtractText()` -- returns `TextExtraction` (includes events, notes)
6. **Context matching** -- same logic as email path (UUID, keywords, auto-create)
7. **Create tasks** -- same as email path; collects created task IDs
8. **Create events** -- one event per `ExtractedEvent`; parses RFC3339 or YYYY-MM-DD for start/end; defaults to 1hr duration; links to raw_input and context
9. **Create notes** -- one note per `ExtractedNote` with `source="voice"`, raw_input_id, context; auto-creates and links tags via `tagbus.Query()` + `tagbus.Create()` + `tagbus.AddToNote()`
10. **Create context event** -- `contextbus.AddEvent(kind="voice")` with raw_input metadata
11. **Create clarifications** -- same ambiguous action/deadline logic as email path
12. **Mark processed** -- returns `IngestResult{TaskIDs, EventIDs, NoteIDs}`

### Async Queue Path (`EnqueueEmail` / `EnqueueText` -> `IngestWorker` -> `ProcessRawInputByID`)
1. **Enqueue** (HTTP handler fast path): Store raw_input with appropriate source type, return ID immediately
2. **Worker polls** every 30s: `QueryRetryable(limit=20)` fetches pending/retryable items
3. **Process** (background goroutine per item, 3-min timeout): Fetch raw_input by ID, mark processing, dispatch to email or text pipeline based on `SourceType`
4. **Retry logic**: if `RetryCount+1 >= MaxRetries` -> `MarkFailed`; else -> `MarkForRetry` (increments retry count, sets next retry time)

## Test Coverage

| Test | File | What it verifies |
|------|------|------------------|
| `processEmailEmptyExtraction` | `ingestbus_test.go` | ProcessEmail succeeds with empty extraction (no action items) |
| `processEmailCreatesTask` | `ingestbus_test.go` | ProcessEmail creates task from action item + stores raw_input |
| `processTextEmptyExtraction` | `ingestbus_test.go` | ProcessText returns empty IngestResult when no items extracted |
| `processTextCreatesTask` | `ingestbus_test.go` | ProcessText creates task, returns task ID, stores raw_input with source=voice |
| `processTextWithContextMatch` | `ingestbus_test.go` | ProcessText matches task to pre-created context via keyword fuzzy match |
| `processTextCreatesEvent` | `ingestbus_test.go` | ProcessText creates event from extracted event data |
| `TestFailover_*` (7 tests) | `extractor/failover_test.go` | Claude success, 429 triggers fallback, context-limit triggers, connection-refused triggers, 400 does NOT trigger, both fail, ExtractText fallback |
| `TestOllama*` (4 tests) | `extractor/ollama_test.go` | Successful email/text extraction, HTTP 500 error, malformed JSON |
