# Ingest Backend System

> Email and text ingestion pipeline: SMTP / HTTP → raw input → parse → sanitize → AI extract → context match → task/event/note creation → vector embedding (optional) → clarifications. Orchestrated by `ingestbus.Business` (no store layer -- pure orchestrator over other domains). Fed by `smtpbus.Server` (email), `voiceingestapp` HTTP handler (voice/text), and a background `IngestWorker` that retries pending items. AI extraction uses a `TieredRouter` that routes by sensitivity tier: financial data (transactions) → local Ollama only, everything else → `FailoverExtractor` (Claude primary via `ClaudeCodeExtractor` with model escalation haiku → sonnet → opus, Ollama fallback on rate-limit / context-limit / connection errors). After creating tasks, events, and notes, embeddings are generated and stored via optional `embeddingBus.EmbedAndStore()` (non-fatal on error). Clarification `AnswerOptions` JSON is written using typed structs from `clarificationbus/options.go` (`ContextAssignmentOptions`, `NewContextOptions`, `AmbiguousActionOptions`, `AmbiguousDeadlineOptions`).

## Core Types

### Extractor Interface (`business/domain/ingestbus/extractor/model.go`)

```go
type Extractor interface {
    ExtractEmail(ctx context.Context, subject, bodyText, fromAddress, userCorrection string, activeContexts []ContextRef) (EmailExtraction, error)
    ExtractText(ctx context.Context, text, userCorrection string, activeContexts []ContextRef, typeHint string) (TextExtraction, error)
    ExtractReceipt(ctx context.Context, ocrText string) (ReceiptExtraction, error)
    AnalyzeGaps(ctx context.Context, entityType, entityContent string, relatedEntities []RelatedEntity) (GapAnalysis, error)
}

// userCorrection (when non-empty) prepends a high-priority preamble to the AI prompt:
// "IMPORTANT — The user has provided a correction for this input: '{userCorrection}'. Treat this as the authoritative interpretation."
// Used for overriding AI extraction with user-provided corrections.

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
    Summary                  string             `json:"summary"`
    SenderName               string             `json:"sender_name"`
    SenderDomain             string             `json:"sender_domain"`
    ActionItems              []ActionItem       `json:"action_items"`
    Deadlines                []Deadline         `json:"deadlines"`
    SuggestedContextKeywords []string           `json:"suggested_context_keywords"`
    Sentiment                string             `json:"sentiment"`           // positive|neutral|negative|mixed
    SuggestedContextID       *string            `json:"suggested_context_id,omitempty"`
    ContextConfidence        float64            `json:"context_confidence,omitempty"`
    SuggestNewContext        bool               `json:"suggest_new_context,omitempty"`
    SuggestedContextTitle    string             `json:"suggested_context_title,omitempty"`
    EntityResolutions        []EntityResolution `json:"entity_resolutions,omitempty"`
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

type AmbiguousReference struct {
    OriginalText  string `json:"original_text"`
    ReferenceType string `json:"reference_type"` // pronoun, vague_noun, implicit
}

type EntityMatch struct {
    ID         string  `json:"id"`
    SourceType string  `json:"source_type"` // "event", "task", "note"
    Title      string  `json:"title"`
    Content    string  `json:"content"`
    Similarity float64 `json:"similarity"`
}

// EntityResolution is Claude's decision about whether the input references an existing entity.
type EntityResolution struct {
    Action      string  `json:"action"`                // "update", "create", "ambiguous"
    MatchedID   string  `json:"matched_id,omitempty"`   // UUID of matched entity
    MatchedType string  `json:"matched_type,omitempty"` // "event", "task", "note"
    Confidence  float64 `json:"confidence"`             // 0-1
    Reasoning   string  `json:"reasoning"`              // Why this decision
}

type TextExtraction struct {
    Summary                  string                 `json:"summary"`
    ActionItems              []ActionItem           `json:"action_items"`
    Deadlines                []Deadline             `json:"deadlines"`
    Events                   []ExtractedEvent       `json:"events"`
    Notes                    []ExtractedNote        `json:"notes"`
    AmbiguousReferences      []AmbiguousReference   `json:"ambiguous_references,omitempty"`
    SuggestedContextKeywords []string               `json:"suggested_context_keywords"`
    SuggestedContextID       *string                `json:"suggested_context_id,omitempty"`
    ContextConfidence        float64                `json:"context_confidence,omitempty"`
    SuggestNewContext        bool                   `json:"suggest_new_context,omitempty"`
    SuggestedContextTitle    string                 `json:"suggested_context_title,omitempty"`
    EntityResolutions        []EntityResolution     `json:"entity_resolutions,omitempty"`
}

// ReceiptExtraction holds structured data extracted from OCR'd receipt text.
type ReceiptExtraction struct {
    Merchant string            `json:"merchant"`
    Date     string            `json:"date"`     // YYYY-MM-DD
    Total    int               `json:"total"`    // cents
    Tax      int               `json:"tax"`      // cents
    Subtotal int               `json:"subtotal"` // cents
    Items    []ReceiptLineItem `json:"items"`
    Notes    string            `json:"notes,omitempty"`
}

type ReceiptLineItem struct {
    Description string `json:"description"`
    Amount      int    `json:"amount"`   // cents
    Quantity    int    `json:"quantity"`
}

type RelatedEntity struct {
    ID         string `json:"id"`
    SourceType string `json:"source_type"` // "task", "event", "note"
    Title      string `json:"title"`
    Content    string `json:"content"`
}

type GapCandidate struct {
    Category   string   `json:"category"`    // missing_contact, missing_location, missing_detail, missing_dependency, missing_context
    Question   string   `json:"question"`    // e.g. "What is Dr. Smith's phone number?"
    Reasoning  string   `json:"reasoning"`   // e.g. "You have an appointment but no contact info stored"
    Confidence float64  `json:"confidence"`  // 0-1
    RelatedIDs []string `json:"related_ids"` // IDs of related entities that informed this gap
}

type GapAnalysis struct {
    Gaps []GapCandidate `json:"gaps"`
}
```

### TieredRouter (`business/domain/ingestbus/extractor/router.go`)

```go
type TieredRouter struct {
    log       *logger.Logger
    general   Extractor  // FailoverExtractor (Claude → Ollama on error)
    localOnly Extractor  // OllamaExtractor — for sensitive data (transactions)
}

func NewTieredRouter(log *logger.Logger, general Extractor, localOnly Extractor) *TieredRouter
func (r *TieredRouter) ExtractEmail(ctx, subject, bodyText, fromAddress string, activeContexts []ContextRef) (EmailExtraction, error)
func (r *TieredRouter) ExtractText(ctx, text string, activeContexts []ContextRef, typeHint string) (TextExtraction, error)
func (r *TieredRouter) ExtractReceipt(ctx context.Context, ocrText string) (ReceiptExtraction, error)
func (r *TieredRouter) AnalyzeGaps(ctx context.Context, entityType, entityContent string, relatedEntities []RelatedEntity) (GapAnalysis, error)
```

**Routing rules:**
- `typeHint == "transaction"` → `localOnly` (Ollama only, never sends to Claude)
- All other typeHints → `general` (FailoverExtractor: Claude primary, Ollama fallback)
- `ExtractEmail` → always `general`
- `ExtractReceipt` → always `general` (receipt OCR text is not sensitive financial data)
- `AnalyzeGaps` → `localOnly` if available (gap analysis uses entity summaries, no raw PII), falls back to `general`
- When `localOnly` is nil (Ollama disabled), transaction requests return zero-value `TextExtraction`

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
    embeddingBus     *embeddingbus.Business    // optional embedder for vector storage
    gapBus           *knowledgegapbus.Business // optional knowledge gap detector
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

func (b *Business) WithEmbedder(emb *embeddingbus.Business) *Business
func (b *Business) WithGapDetector(gap *knowledgegapbus.Business) *Business
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
- **WithGapDetector** (optional) attaches a knowledge gap detector; after email processing completes, Step 10b asynchronously fires `gapBus.Detect()` with raw_input entity context.

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
    ID             uuid.UUID
    SourceType     rawinputsource.Source
    Status         rawinputstatus.Status
    RawContent     string
    ProcessedAt    *time.Time
    Error          *string
    RetryCount     int
    NextRetryAt    *time.Time
    MaxRetries     int
    Result         json.RawMessage
    CreatedAt      time.Time
    UserCorrection *string          // user-provided correction for extraction override
}

type NewRawInput struct {
    SourceType rawinputsource.Source
    RawContent string
}

type UpdateRawInput struct {
    Status         *rawinputstatus.Status
    ProcessedAt    *time.Time
    Error          *string
    RetryCount     *int
    NextRetryAt    *time.Time
    Result         *json.RawMessage
    UserCorrection *string
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

type VoiceReferenceOptions struct {
    OriginalText  string `json:"original_text"`
    ReferenceType string `json:"reference_type"`
    ClauseText    string `json:"clause_text"`
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
- `business/domain/ingestbus/ingestbus.go` -- **ProcessEmail()**, **ProcessText()**, **Reprocess()**, **EnqueueEmail()**, **EnqueueText()**, **ProcessRawInputByID()**, **processRawInput()**, **processTextInput()**, **applyEntityUpdate()**, **createAmbiguousMatchClarification()**, **matchContextByKeywords()** -- pipeline orchestrator; text path runs cleanup → per-clause classify+extract → create items per clause with unconfirmed flag → generate voice_reference clarifications for ambiguous references; entity resolution routing: "update" action calls applyEntityUpdate (sets Unconfirmed=true on matched event/task), "ambiguous" action calls createAmbiguousMatchClarification; after email processing, Step 10b asynchronously fires knowledge gap detection (if gapBus attached) on raw_input content; no store layer (orchestrates other domains)
- `business/domain/ingestbus/parse.go` -- **parseEmail()**, **parseEmailEntity()** -- RFC 5322 parsing via `emersion/go-message`; extracts MessageID, From, To, Subject, BodyText, BodyHTML from MIME parts
- `business/domain/ingestbus/ingestbus_test.go` -- **Test_Ingest** -- 9 test cases: empty email extraction, email creates task + raw_input, empty text extraction, text creates task, text with context match, text creates event, compound input splits into two tasks, low-confidence clause creates unconfirmed task, ambiguous reference creates voice_reference clarification

### Classify Package
- `business/domain/ingestbus/classify/classifier.go` -- **Classify()** -- rule-based type classifier: obligation verbs → task (0.9), temporal anchor alone → event (0.9), reference patterns → note (0.95), obligation+temporal → task (0.6), ambiguous → note (0.5); returns `Classification{Type, Confidence}`
- `business/domain/ingestbus/classify/classifier_test.go` -- table-driven: clear tasks, events, notes, ambiguous cases

### Cleanup Package
- `business/domain/ingestbus/cleanup/cleanup.go` -- **StripFillers()**, **SplitClauses()** -- StripFillers removes transcription noise (um, uh, you know, etc.); SplitClauses splits on conjunctions (" and ", " also ", " oh and ") and sentence-ending punctuation
- `business/domain/ingestbus/cleanup/cleanup_test.go` -- filler removal, clause splitting, edge cases

### Extractor Implementations
- `business/domain/ingestbus/extractor/model.go` -- **Extractor** interface, **ContextRef**, **ActionItem**, **Deadline**, **EmailExtraction**, **ExtractedEvent**, **ExtractedNote**, **EntityMatch**, **EntityResolution**, **TextExtraction**, **RelatedEntity**, **GapCandidate**, **GapAnalysis** types
- `business/domain/ingestbus/extractor/claudecli.go` -- **ClaudeCodeExtractor** -- production implementation using Claude CLI with model escalation and JSON schema validation; escalation callback: escalates if zero action items AND confidence < 0.3 (email) or zero action items (text)
- `business/domain/ingestbus/extractor/ollama.go` -- **OllamaExtractor** -- local Ollama fallback; POSTs to `/api/generate` with `format:"json"` and 30s timeout; drains body on non-200; fixes `ContextConfidence=0.85` (local models cannot reliably self-report)
- `business/domain/ingestbus/extractor/prompt.go` -- **BuildCandidateBlock()** -- formats semantic candidate entities for prompt injection; **BuildEmailExtractionPrompt()**, **BuildTextExtractionPrompt()** -- shared prompt templates (now accept `candidates []EntityMatch` parameter); text prompt includes current time, timezone, and UTC conversion instructions; **BuildGapAnalysisPrompt()** -- builds gap analysis prompt from entity content and related entities
- `business/domain/ingestbus/extractor/failover.go` -- **FailoverExtractor** -- wraps `*ClaudeCodeExtractor` (primary) + `*OllamaExtractor` (fallback); `isFallbackError()` triggers on "429", "context"+"limit", "connection", "timeout", "refused"; `newFailoverExtractorForTest()` package-private helper accepts interfaces
- `business/domain/ingestbus/extractor/mock.go` -- **MockExtractor** -- returns configured `Result` (email), `TextResult` (text), `ReceiptResult` (receipt), `GapResult` (gap analysis), or `Err` for tests
- `business/domain/ingestbus/extractor/failover_test.go` -- 7 tests: Claude success (sentinel ensures Ollama not called), 429 triggers fallback, context-limit triggers fallback, connection-refused triggers fallback, 400 does NOT trigger fallback, both fail returns Ollama error, ExtractText fallback works
- `business/domain/ingestbus/extractor/ollama_test.go` -- 4 tests: successful email/text extraction via httptest server, HTTP 500 error, malformed inner JSON

### Background Worker
- `business/sdk/worker/ingestworker.go` -- **IngestWorker** -- polls every 30s for retryable raw_inputs (batch of 20); dispatches each in a goroutine with 3-minute timeout; on failure: if `RetryCount+1 >= MaxRetries` calls `MarkFailed`, else calls `MarkForRetry`

### SMTP Server
- `business/domain/smtpbus/smtpbus.go` -- **Server**, **session** -- SMTP server implementing `smtp.Backend`; `session.Data()` reads email body and calls `ingestBus.ProcessEmail()`; accepts email even on pipeline failure (stored as failed raw_input); validates recipient domain; 10MB max message, 5 max recipients

### Foundation
- `foundation/claudecli/claudecli.go` -- **Client.RunJSON()** -- wraps `claude -p` with `--output-format json --json-schema --bare`; tries models in escalation order, calls `shouldEscalate()` callback after each parse

### Wiring
- `api/services/planner/main.go` -- constructs `igBus` with all 8 domain deps + extractor; attaches `embBus` (embedder) via `WithEmbedder()` if available; optionally attaches `gapBus` (knowledge gap detector) via `WithGapDetector()`; passes `igBus` to `smtpbus.NewServer()` and `worker.NewIngestWorker()`; worker runs in background goroutine

## Impact Callouts

### -- EmailExtraction (`business/domain/ingestbus/extractor/model.go`)
Changing this struct shape affects:
- `extractor/claudecli.go` -- `emailExtractionSchema` JSON schema constant must match struct fields exactly; `shouldEscalate` reads `.ActionItems` length and `.ContextConfidence`
- `extractor/ollama.go` -- `json.Unmarshal` into this struct; hardcodes `.ContextConfidence = 0.85` post-parse
- `extractor/prompt.go` -- `BuildEmailExtractionPrompt` instructs Claude to return JSON matching this schema
- `extractor/mock.go` -- `MockExtractor.Result` field is this type
- `extractor/failover.go` -- delegates and returns this type from `ExtractEmail()`
- `ingestbus/ingestbus.go:processRawInput` -- reads `.SuggestedContextID`, `.ContextConfidence`, `.SuggestNewContext`, `.SuggestedContextTitle`, `.ActionItems[].Title/Description/Priority/Interpretations`, `.Deadlines[].IsAmbiguous/Date/Description`, `.SuggestedContextKeywords`, `.Sentiment`, `.Summary`, `.EntityResolutions[].Action/MatchedID/MatchedType/Confidence/Reasoning` for entity update/create routing

### -- TextExtraction (`business/domain/ingestbus/extractor/model.go`)
Changing this struct shape affects:
- `extractor/claudecli.go` -- `textExtractionSchema` JSON schema constant must match; `shouldEscalate` reads `.ActionItems` length
- `extractor/ollama.go` -- `json.Unmarshal` into this struct; hardcodes `.ContextConfidence = 0.85`
- `extractor/prompt.go` -- `BuildTextExtractionPrompt` instructs Claude to return JSON matching this schema
- `extractor/mock.go` -- `MockExtractor.TextResult` field is this type
- `extractor/failover.go` -- delegates and returns this type from `ExtractText()`
- `ingestbus/ingestbus.go:processTextInput` -- reads all fields from `EmailExtraction` callout above, plus `.Events[].Title/Description/Location/StartsAt/EndsAt/AllDay`, `.Notes[].Content/SuggestedTags`, `.AmbiguousReferences[].OriginalText/ReferenceType` for voice_reference clarification generation, `.EntityResolutions[].Action/MatchedID/MatchedType/Confidence/Reasoning` for entity update/create routing
- `app/domain/noteapp/noteapp.go` -- calls `extractor.ExtractText()` for note auto-tag/context suggestion
- `app/domain/eventapp/eventapp.go` -- calls `extractor.ExtractText()` for event auto-extraction from text
- `app/domain/classifyapp/classifyapp.go` -- calls `extractor.ExtractText()` for task classification
- `app/domain/mcpapp/mcpapp.go` -- calls `extractor.ExtractText()` in background goroutine for MCP classify

### -- EntityMatch / EntityResolution (`business/domain/ingestbus/extractor/model.go`)
New types supporting entity resolution (semantic matching of input to existing entities).
- `extractor/prompt.go` -- **BuildCandidateBlock()** formats `[]EntityMatch` into prompt injection block; **BuildEmailExtractionPrompt()** and **BuildTextExtractionPrompt()** both accept `candidates []EntityMatch` parameter and call `BuildCandidateBlock()` to inject into prompt
- `extractor/claudecli.go` -- `emailExtractionSchema` and `textExtractionSchema` must include `entity_resolutions` field in JSON schema
- `ingestbus/ingestbus.go` -- reads `EntityResolutions` from both `EmailExtraction` and `TextExtraction`; per resolution: if `action=="update"` calls `applyEntityUpdate()` (marks entity Unconfirmed=true), if `action=="ambiguous"` calls `createAmbiguousMatchClarification()` with resolution details
- `extractor/mock.go` -- `MockExtractor.Result` and `TextResult` must populate `EntityResolutions` field to match test scenarios
- Callers must provide `[]EntityMatch` candidates to extraction methods. Currently:
  - `voiceingestapp/route.go` -- TODO: integrate semantic search to fetch candidates before calling `ExtractText()`
  - `ingestbus.go` -- TODO: integrate semantic search in `processRawInput()` and `processTextInput()` before calling extractor

### -- Extractor interface (`business/domain/ingestbus/extractor/model.go`)
Adding/changing a method or its signature affects:
- `extractor/claudecli.go` -- implements both methods; builds correctionPreamble and prepends to prompts; must update if signature changes
- `extractor/ollama.go` -- implements both methods; builds correctionPreamble and prepends to prompts; must update if signature changes
- `extractor/failover.go` -- implements both methods (delegates to primary then fallback); must update if signature changes
- `extractor/router.go` -- implements both methods (routes by typeHint/sensitivity); passes userCorrection through; must update if signature changes
- `extractor/mock.go` -- implements both methods (returns configured result); must update if signature changes
- `extractor/prompt.go` -- `BuildEmailExtractionPrompt()` and `BuildTextExtractionPrompt()` both accept `userCorrection`; when non-empty, `correctionPreamble()` prepends high-priority instruction to prompt
- `ingestbus/ingestbus.go` -- calls `ExtractEmail()` and `ExtractText()` via interface, passing `ri.UserCorrection` from the raw_input; must update all call sites if signature changes
- `voiceingestapp/route.go` -- constructs `ClaudeCodeExtractor` and stores as `extractor.Extractor`; ingest handler passes empty string (no user correction at HTTP layer)
- `noteapp/route.go` -- constructs `ClaudeCodeExtractor`; calls in noteapp handler pass empty string
- `eventapp/route.go` -- constructs `ClaudeCodeExtractor`; calls pass empty string
- `classifyapp/route.go` -- constructs `ClaudeCodeExtractor`; calls pass empty string
- `mcpapp/route.go` -- constructs `ClaudeCodeExtractor`; calls pass empty string
- `transactionbus/enricher.go` -- calls extractor for transaction enrichment; passes empty string
- `api/services/planner/main.go` -- constructs extractor for SMTP path

### -- FailoverExtractor (`business/domain/ingestbus/extractor/failover.go`)
Changing fallback trigger logic (`isFallbackError`) affects:
- `extractor/failover_test.go` -- 7 tests cover exact trigger conditions; update test cases if trigger rules change
- `ingestbus/ingestbus.go` -- soft-failure behaviour: extraction errors that don't trigger fallback are swallowed (pipeline continues without AI features)
- `NewFailoverExtractor` accepts `*ClaudeCodeExtractor` and `*OllamaExtractor` (concrete types) to prevent accidental nesting -- wiring must pass concrete pointers, not interface values

### -- clarificationbus option types (`business/domain/clarificationbus/options.go`)
Changing any option struct field affects:
- `ingestbus/ingestbus.go:processRawInput` -- marshals `ContextAssignmentOptions`, `NewContextOptions`, `AmbiguousActionOptions`, `AmbiguousDeadlineOptions`; field renames silently produce wrong JSON keys
- `ingestbus/ingestbus.go:processTextInput` -- additionally marshals `TypeAssignmentOptions` for low-confidence clauses (confidence < 0.75); marshals `VoiceReferenceOptions` for ambiguous references; marshals `AmbiguousEntityMatchOptions` for ambiguous entity resolution
- `ingestbus/ingestbus.go:createAmbiguousMatchClarification` -- marshals `AmbiguousEntityMatchOptions` for entity resolution clarifications; field renames silently produce wrong JSON keys
- `app/domain/classifyapp/classifyapp.go` -- marshals `ContextAssignmentOptions` for low-confidence task classification
- `app/domain/mcpapp/mcpapp.go` -- marshals `ContextAssignmentOptions` in background goroutine for MCP classify tool
- Frontend `ClarificationCard` component -- deserializes `answer_options` JSON per clarification kind; JSON field renames break the UI; `type_assignment` kind needs its own branch

### -- gapBus field in Business (`business/domain/ingestbus/ingestbus.go`)
Optional knowledge gap detector attachment affects:
- `ingestbus.go:WithGapDetector()` -- option method to attach `knowledgegapbus.Business`
- `ingestbus.go:processRawInput()` -- Step 10b: if `gapBus != nil`, fires async `gapBus.Detect(context.Background(), "raw_input", ri.ID, ri.RawContent)` in a goroutine with background context; non-blocking, failures logged implicitly by gap detector
- `api/services/planner/main.go` -- wiring: optional call to `igBus.WithGapDetector(gapBus)` after constructing ingest business; if gap bus not available, detection step silently skipped

### -- classify.Classification (`business/domain/ingestbus/classify/classifier.go`)
Changing this type or `Classify()` logic affects:
- `ingestbus/ingestbus.go:processTextInput` -- calls `Classify(clause)` per clause; uses `.Confidence < 0.75` threshold for `Unconfirmed` flag and `TypeAssignment` clarification creation; `.Type` passed as type hint to `ExtractText()`
- Confidence threshold (0.75) matches the context confidence cutoff — tune together if recalibrating

### -- IngestResult (`business/domain/ingestbus/ingestbus.go`)
Changing this struct affects:
- `ingestbus/ingestbus.go:processTextInput` -- builds and returns result with `TaskIDs`, `EventIDs`, `NoteIDs`
- `ingestbus/ingestbus_test.go` -- asserts on `result.TaskIDs`, `result.EventIDs` lengths

### -- RawInput (`business/domain/rawinputbus/model.go`)
Changing this struct shape affects:
- `rawinputbus/rawinputbus.go` -- all CRUD methods including `MarkProcessing()`, `MarkProcessed()`, `MarkPartial()`, `MarkFailed()`, `MarkForRetry()`, `QueryRetryable()`, `Update()`
- `rawinputdb/rawinputdb.go` -- SQL columns, `Scan()` field list, INSERT/UPDATE statements
- `rawinputdb/model.go` -- DB struct (with sql tags) + `toBusRawInput()` converter
- `rawinputapp/model.go` -- app DTO + `toAppRawInput()` converter
- `ingestbus/ingestbus.go` -- creates and updates raw inputs throughout pipeline; reads `ri.RawContent`, `ri.ID`, `ri.SourceType`, `ri.RetryCount`, `ri.MaxRetries`, `ri.UserCorrection` (passed to extractor); reassigns `ri` after `Update()` so pipeline sees latest version (including `Result`)
- `worker/ingestworker.go` -- reads `ri.ID`, `ri.RetryCount`, `ri.MaxRetries` for retry logic
- **UserCorrection field:** Passed from `UpdateRawInput` to both `ExtractEmail()` and `ExtractText()` calls; when non-empty, overrides AI extraction with user-provided interpretation
- **Phase 4 additions:** `SourceEntityID`, `SourceEntityKind`, `SkipClassify`, `ReingestMode` fields enable lightweight reprocessing path; `ProcessRawInputByID` branches on `SkipClassify` to call `processSkipClassify()` instead of full pipeline
- Migration SQL required if DB column added/removed

### -- ProcessRawInputByID (`business/domain/ingestbus/ingestbus.go`) — Phase 4 skip_classify branch
`ProcessRawInputByID` now branches on `ri.SkipClassify` flag:
- If `skipClassify=true` AND `SourceEntityID != nil`: calls `processSkipClassify(ctx, ri, skipClassify, reingestMode)` for lightweight reprocessing (delete old embeddings → regenerate → fire gap detection)
- If `skipClassify=true` but `SourceEntityID=nil`: logs warning, falls through to full pipeline
- Otherwise: dispatches to `processRawInput()` (email) or `processTextInput()` (voice) based on `SourceType`
- **reingestMode flag:** Passed to `processTextInput()` to suppress `unconfirmed` flip during re-ingestion; preserves user-confirmed entity state on clarification follow-up

### -- processSkipClassify (`business/domain/ingestbus/ingestbus.go`) — Phase 4 new method
```go
func (b *Business) processSkipClassify(ctx context.Context, ri rawinputbus.RawInput, skipClassify bool, reingestMode bool) error
```
Handles skip_classify path: load entity by kind + ID, delete old embeddings, regenerate from entity text, fire knowledge gap detection, mark processed. Non-blocking; failures logged, pipeline continues. Called only when `skip_classify=true` and `SourceEntityID` is set.

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
| **rawinputbus** | Create raw_input (Step 1), MarkProcessing/MarkProcessed/MarkPartial/MarkFailed lifecycle |
| **emailbus** | Store parsed email record, dedup via `QueryByMessageID()`, update email with matched context |
| **taskbus** | Create tasks from `extraction.ActionItems` with priority/status/energy/context/raw_input_id |
| **contextbus** | Query active contexts for AI prompt, verify suggested context exists, auto-create new contexts, add context events |
| **clarificationbus** | Create `context_assignment`, `ambiguous_action`, `ambiguous_deadline`, `new_context`, `type_assignment`, `voice_reference` clarifications using typed option structs |
| **eventbus** | Create events from `extraction.Events` (text pipeline only) with parsed start/end times and location |
| **notebus** | Create notes from `extraction.Notes` (text pipeline only) with content, source, raw_input_id, context |
| **tagbus** | Query existing tags by name, create new tags, link tags to notes via `AddToNote()` (text pipeline only) |
| **embeddingbus** | (Optional, attached via `WithEmbedder()`) Embed and store vectors for: emails (Step 9, email path), tasks (Step 8, email path; Step 9, text path), events (Step 9, text path), and notes (Step 9, text path) via `EmbedAndStore(ctx, entityType, entityID, content)`; non-fatal on error, logged and pipeline continues |
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

### Skip Classify Path (`ProcessRawInputByID` → `processSkipClassify`)

**Phase 4 specialization:** When `skip_classify=true` and `SourceEntityID` is set, branches to lightweight reprocessing path (no AI extraction, no entity creation).

1. **Load entity** -- by `SourceEntityKind` (task|note|event) and `SourceEntityID`; extract entity text (title+description for task/event, content for note)
2. **Delete old embeddings** -- if `embeddingBus != nil`, call `DeleteBySource(entityKind, entityID)` to clear vector store
3. **Regenerate embeddings** -- if `embeddingBus != nil`, call `EmbedAndStore(ctx, entityKind, entityID, entityText)` with refreshed entity content
4. **Fire gap detection** -- if `gapBus != nil`, fire async `gapBus.Detect(context.Background(), entityKind, entityID, entityText)` in background goroutine
5. **Mark processed** -- `rawinputbus.MarkProcessed()`

**Use case:** User corrects/clarifies an entity's raw_input; system re-runs embedding + gap detection without re-classifying. Avoids `Unconfirmed` flip on subsequent ingests (preserves user-confirmed state).

### Email Path (`ProcessEmail` / `processRawInput`)
1. **Store raw_input** -- `rawinputbus.Create(Email, rawContent)` -- status: pending
2. **Mark processing** -- `rawinputbus.MarkProcessing()` -- status: processing
3. **Parse RFC 5322** -- `parseEmail(rawContent)` -- extracts headers + MIME body parts
4. **Dedup check** -- `emailbus.QueryByMessageID()` -- if found, mark processed and return
5. **Store email** -- `emailbus.Create()` -- persists parsed fields
6. **Fetch active contexts** -- `contextbus.Query(Status=Active, limit 50)` -- build `[]ContextRef` for AI
7. **Sanitize** -- `sanitize.Sanitize(subject)` + `sanitize.Sanitize(body)` -- PII redaction
8. **AI extraction** -- `extractor.ExtractEmail()` -- returns `EmailExtraction`; on error, marks partial and returns (soft failure)
9. **Embed email content** -- if `embeddingBus != nil`, calls `embeddingBus.EmbedAndStore(ctx, "email", email.ID, content)` with extraction summary (or body if summary empty); non-fatal on error, error logged and pipeline continues
10. **Context matching** -- suggested UUID first, keyword fuzzy match fallback, auto-create context if `SuggestNewContext=true`; creates `new_context` clarification for auto-created contexts; creates `context_assignment` clarification if confidence < 0.7
11. **Create tasks** -- one task per `ActionItem` with mapped priority + raw_input_id link; after creating each task, if `embeddingBus != nil`, calls `embeddingBus.EmbedAndStore(ctx, "task", task.ID, actionItem.Description)`; creates `ambiguous_action` clarification for items with multiple interpretations
12. **Create deadline clarifications** -- `ambiguous_deadline` clarification for `Deadline.IsAmbiguous=true`
13. **Update email context** -- `emailbus.Update()` with matched context ID
14. **Mark processed or partial** -- `rawinputbus.MarkProcessed()` or `MarkPartial()` if entity creation failures occurred

### Text Path (`ProcessText` / `processTextInput`)
1. **Store raw_input** -- `rawinputbus.Create(Voice, rawContent)` -- status: pending
2. **Mark processing** -- status: processing
3. **Fetch active contexts** -- same as email path
4. **Sanitize** -- `sanitize.Sanitize(rawContent)`
5. **Cleanup** -- `cleanup.StripFillers()` removes transcription noise; `cleanup.SplitClauses()` splits on conjunctions/punctuation; falls back to full text if no clauses
6. **Per-clause classify + extract** -- for each clause: `classify.Classify(clause)` → type + confidence; `extractor.ExtractText(clause, typeHint)` with type hint; skip clause if extraction fails; bail out with empty result if all clauses fail
7. **Context matching** -- merges `SuggestedContextKeywords` from all clauses; picks highest-confidence `SuggestedContextID`; same UUID verify → keyword fuzzy → auto-create logic as email path
8. **Create items per clause** -- for each clause: `unconfirmed = confidence < 0.75` (Phase 4 guard: if `reingestMode=true`, skip unconfirmed flip to preserve confirmed state); create `TypeAssignment` clarification if unconfirmed; create tasks, events, notes from that clause's extraction with `Unconfirmed` flag set + raw_input_id link
9. **Embed created items** -- after creating each task, if `embeddingBus != nil`, calls `embeddingBus.EmbedAndStore(ctx, "task", task.ID, actionItem.Description)`; after creating each event, if `embeddingBus != nil`, calls `embeddingBus.EmbedAndStore(ctx, "event", event.ID, event.Description)` with event details; after creating each note, if `embeddingBus != nil`, calls `embeddingBus.EmbedAndStore(ctx, "note", note.ID, note.Content)` with note content
10. **Create notes** -- one note per `ExtractedNote` with `source="voice"`, raw_input_id, context; auto-creates and links tags
11. **Create clarifications** -- ambiguous action/deadline clarifications across all clause items; voice_reference clarifications for ambiguous references (pronouns, vague nouns, implicit refs)
12. **Save pipeline result** -- `rawInputBus.Update(ri, {Result: json})` -- captures updated `ri` so `MarkProcessed`/`MarkPartial` uses the latest version (with Result populated)
13. **Mark processed or partial** -- `MarkPartial()` if any entity creation failures occurred; otherwise `MarkProcessed()`; returns `IngestResult{TaskIDs, EventIDs, NoteIDs}`

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
| `processTextCompoundInput` | `ingestbus_test.go` | Compound input split on "and" creates one task per clause |
| `processTextLowConfidenceUnconfirmed` | `ingestbus_test.go` | Clause with obligation+temporal (confidence 0.6) creates task with Unconfirmed=true |
| `processTextAmbiguousReference` | `ingestbus_test.go` | Ambiguous reference in extraction creates voice_reference clarification card |
| `TestFailover_*` (7 tests) | `extractor/failover_test.go` | Claude success, 429 triggers fallback, context-limit triggers, connection-refused triggers, 400 does NOT trigger, both fail, ExtractText fallback |
| `TestOllama*` (4 tests) | `extractor/ollama_test.go` | Successful email/text extraction, HTTP 500 error, malformed JSON |
