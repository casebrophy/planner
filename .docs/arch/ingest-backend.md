# Ingest Backend Architecture

## Overview

The ingest domain is a multi-step orchestration engine that transforms raw input (emails, voice, text) into structured entities (tasks, events, notes) via AI extraction, context matching, and clarification generation. It is **input-agnostic** — all ingestion pipelines are driven by a single business module (`ingestbus.Business`) that reads raw content and orchestrates extraction, classification, embedding, and entity creation.

**Key responsibility:** Receive raw input → sanitize → extract via AI → match context → create entities → generate clarifications → fire knowledge gap detection. All steps operate asynchronously via a background job queue.

## Files Changed (Latest)

- `business/domain/ingestbus/ingestbus.go` — fix gap detection to fire per entity (task/event/note) instead of per raw_input; introduce `GapDetector` interface; add `gapTarget` helper struct
- `business/sdk/migrate/sql/migrate.sql` — migration 1.42 deletes stale `knowledge_gap` rows with `subject_type='raw_input'`

## Core Concepts

### Input Types
- **Email** — RFC 5322 raw message, parsed to extract From, To, Subject, Body (text + HTML), Message-ID
- **Voice / Text** — plain text or voice transcription, split into clauses, per-clause classified (task/note/event) then extracted
- **Manual / Transaction** — synthetic raw inputs created during reingest or transaction ingestion
- **File** — defined in schema but not yet implemented

### Processing Modes
1. **ProcessEmail** — enqueue email for async processing (stores raw_input, then background worker calls ProcessRawInputByID)
2. **ProcessText** — synchronous text ingestion pipeline (voice, manual input), returns created entity IDs immediately
3. **ProcessRawInputByID** — internal method called by background worker; re-executes full pipeline for a raw_input
4. **Reprocess** — re-run pipeline for an existing raw_input (used by user corrections in clarifications)
5. **skip_classify** — reingest branch: load existing entity, regenerate embeddings and gaps WITHOUT re-classifying

### Extraction Routing
The **TieredRouter** (in `extractor/router.go`) dispatches based on data sensitivity:
- General extractors (Claude) → emails, voice, text, gaps
- Local-only extractor (Ollama) → transactions (when configured)
- Fallback routing: general first, local second, skip if both unavailable
- **Gap detection:** Routes exclusively to Claude via sidecar; never routes to Ollama (Ollama timeout is a circuit breaker, not fallback)

## Business Layer

### Path
`business/domain/ingestbus/`

### Main Types

```go
// IngestResult holds the IDs of created tasks, events, and notes from ingestion.
type IngestResult struct {
    TaskIDs  []uuid.UUID
    EventIDs []uuid.UUID
    NoteIDs  []uuid.UUID
}

// PipelineResult tracks per-step pipeline outcomes for observability.
type PipelineResult struct {
    Sanitize     *StepResult `json:"sanitize,omitempty"`
    Extraction   *StepResult `json:"extraction,omitempty"`
    ContextMatch *StepResult `json:"contextMatch,omitempty"`
    Tasks        *StepResult `json:"tasks,omitempty"`
    Events       *StepResult `json:"events,omitempty"`
    Notes        *StepResult `json:"notes,omitempty"`
    GapAnalysis  *StepResult `json:"gapAnalysis,omitempty"`
}

// StepResult records the outcome of a single pipeline step.
type StepResult struct {
    Status string         `json:"status"` // "completed", "failed", "skipped"
    Detail map[string]any `json:"detail,omitempty"`
}

// Business orchestrates the email ingestion pipeline.
type Business struct {
    log              *logger.Logger
    rawInputBus      *rawinputbus.Business      // manage raw_input records
    emailBus         *emailbus.Business         // store/query emails
    taskBus          *taskbus.Business          // create/update tasks
    contextBus       *contextbus.Business       // query active contexts, auto-create new
    clarificationBus *clarificationbus.Business // generate clarifications
    eventBus         *eventbus.Business         // create/update events
    extractor        extractor.Extractor        // AI extraction (router or single impl)
    noteBus          *notebus.Business          // create/update notes
    tagBus           *tagbus.Business           // fetch/create tags for notes
    embeddingBus     *embeddingbus.Business     // optional semantic search + embed+store
    gapBus           *knowledgegapbus.Business  // optional async gap detection
}
```

### Core Methods

#### ProcessEmail(ctx context.Context, rawContent string) error
- **Step 1:** Create raw_input with source_type=Email, status=pending
- **Step 2:** Parse RFC 5322 email → FromAddress, Subject, BodyText, BodyHTML, MessageID
- **Step 3:** Dedup check via emailbus.QueryByMessageID; skip if exists
- **Step 4:** Store email record, FK to raw_input
- **Step 5:** Fetch active contexts (status=active, limit 50)
- **Step 5b:** Sanitize subject + body text via `business/sdk/sanitize`
- **Step 5c:** Semantic pre-fetch (if embeddingBus available) — search for candidates with similarity >= 0.70
- **Step 6:** AI extraction via extractor.ExtractEmail() → summary, action_items, deadlines, suggested_context, entity_resolutions
- **Step 7:** Embed email content if embeddingBus available
- **Step 8:** Context matching (priority: explicit context_id, then keyword fuzzy match, then auto-create)
- **Step 8b:** Fire async knowledge gap detection (skip for transactions)
- **Step 9:** Create clarifications for low-confidence context matches, ambiguous actions, ambiguous deadlines, entity resolutions, new_context (skip for transactions)
- **Step 10:** Create tasks from action_items with priority parsing
- **Step 11:** Mark raw_input processed or partial (if task creation failed)
- On failure: Mark raw_input failed with error message

#### ProcessText(ctx context.Context, rawContent string) (IngestResult, error)
- Synchronous version of ProcessRawInputByID for voice/manual input
- Returns created entity IDs immediately
- Steps 1–10 same as email, but per-clause:
  - Step 3: Fetch active contexts
  - Step 4: Sanitize + cleanup (strip fillers, split clauses)
  - Step 5: Per-clause classify (type hint: task/note/event) + extract
  - Step 6: Semantic pre-fetch candidates per clause
  - Step 7: Create tasks, events, notes (with Unconfirmed flag for low-confidence classifications)
  - Step 8: Generate TypeAssignment clarifications for low-confidence clauses (skip for transactions)
  - Step 9: Generate clarifications for ambiguous actions, deadlines, voice references (skip for transactions)
  - Step 10: Mark processed or partial

#### ProcessRawInputByID(ctx context.Context, id uuid.UUID) error
- Internal: called by background worker to process an existing raw_input
- **Phase 1:** Load raw_input, mark processing
- **Phase 2:** Branch on skip_classify flag
  - If skip_classify=true + SourceEntityID is set: call processSkipClassify (load entity, regenerate embeddings + gaps only)
  - Otherwise: proceed to Phase 3
- **Phase 3:** Branch on SourceType
  - Email → call processRawInput (email pipeline)
  - Voice/Manual → call processTextInput (text pipeline)
- Returns error without marking raw_input failed (caller decides retry/terminal)

#### EnqueueEmail(ctx context.Context, rawContent string) (uuid.UUID, error)
- Store raw_input with source_type=Email, return ID
- Background worker will async call ProcessRawInputByID
- Used by SMTP/email ingest endpoints

#### EnqueueText(ctx context.Context, rawContent string) (uuid.UUID, error)
- Store raw_input with source_type=Voice, return ID
- Used by voice/text ingest endpoints

#### Reprocess(ctx context.Context, rawInputID uuid.UUID) error
- Mark raw_input as processing + reset internal state
- Re-run processRawInput or processTextInput with user-corrected content
- Called when user provides `UserCorrection` in a clarification

#### applyEntityUpdate(ctx context.Context, res extractor.EntityResolution, ri rawinputbus.RawInput) error
- Apply an "update" action from AI entity resolution
- Sets Unconfirmed=true on the matched entity (task/event)
- Logs the update for observability

#### createAmbiguousMatchClarification(ctx context.Context, res extractor.EntityResolution, ri rawinputbus.RawInput) error
- Create AmbiguousEntityMatch clarification when AI cannot decide update vs. create
- Stores options (use_existing / create_new) for user decision

#### processSkipClassify(ctx context.Context, ri rawinputbus.RawInput, skipClassify, reingestMode bool) error
- **Phase 4 branch (reingest):** Load entity by kind, delete old embeddings, regenerate embeddings + gap detection
- Skips classify and extraction steps entirely
- Used when reingest is triggered from frontend with "keep confirmed changes"

### Cross-Domain Calls (Outbound)

| Domain | Method | Usage |
|--------|--------|-------|
| **rawinputbus** | Create, QueryByID, MarkProcessing, MarkProcessed, MarkPartial, MarkFailed, Update, ResetForReingest, ResetForReprocess | Manage raw_input lifecycle, update with UserCorrection, set skip_classify/reingest_mode flags |
| **emailbus** | Create, QueryByMessageID, Update | Store parsed email, dedup check, associate with context |
| **taskbus** | Create, QueryByID, Update, DeleteByRawInputUnconfirmed | Create tasks from action_items, set Unconfirmed flag, delete on reingest |
| **contextbus** | Query (filter status=active), QueryByID, Create | Fetch active contexts for matching, verify suggested contexts, auto-create new contexts |
| **clarificationbus** | Upsert | Generate clarifications for low-confidence matches, ambiguous actions, deadlines, new contexts, entity mismatches, voice references |
| **eventbus** | Create, QueryByID, Update, DeleteByRawInputUnconfirmed | Create events from extracted events, set Unconfirmed flag |
| **noteBus** | Create, QueryByID, Update, DeleteByRawInputUnconfirmed | Create notes from extracted notes |
| **tagbus** | Query, Create, AddToNote | Fetch/create tags suggested by extraction, link to notes |
| **embeddingbus** (optional) | Search, EmbedAndStore, DeleteBySource | Pre-fetch semantic candidates (similarity filter 0.70), embed email/task/note/event content, clear old embeddings on reingest |
| **knowledgegapbus** (optional) | Detect | Fire async gap detection for email/raw_input content (skip for transactions) |

**Key pattern:** All cross-domain calls happen at orchestration points (after extraction, entity creation, etc.). None are bidirectional — ingestbus never receives calls from other domains.

## Extractor Subdomain

### Path
`business/domain/ingestbus/extractor/`

### Interface & Implementations

```go
// Extractor defines the interface for AI extraction.
type Extractor interface {
    ExtractEmail(ctx context.Context, subject, bodyText, fromAddress, userCorrection string, activeContexts []ContextRef) (EmailExtraction, error)
    ExtractText(ctx context.Context, text, userCorrection string, activeContexts []ContextRef, typeHint string) (TextExtraction, error)
    ExtractReceipt(ctx context.Context, ocrText string) (ReceiptExtraction, error)
    AnalyzeGaps(ctx context.Context, entityType, entityContent string, relatedEntities []RelatedEntity) (GapAnalysis, error)
}
```

**Implementations:**
- **TieredRouter** (router.go) — dispatches based on data sensitivity (general vs. local-only)
- **FailoverExtractor** (failover.go) — tries ClaudeExtractor first, falls back to OllamaExtractor
- **ClaudeExtractor** (claudecli.go) — calls Claude Code sidecar via HTTP + structured outputs
- **OllamaExtractor** (ollama.go) — calls local Ollama API (transaction enrichment, gap analysis)
- **MockExtractor** (mock.go) — for testing

### Key Types

```go
// ContextRef is a lightweight reference to an active context for the AI prompt.
type ContextRef struct {
    ID    string `json:"id"`
    Title string `json:"title"`
}

// ActionItem represents a task extracted from an email.
type ActionItem struct {
    Title            string   `json:"title"`
    Description      string   `json:"description"`
    Priority         string   `json:"priority"`                // "low", "medium", "high", "urgent"
    Interpretations  []string `json:"interpretations,omitempty"`
}

// Deadline represents a deadline mentioned in an email.
type Deadline struct {
    Description string `json:"description"`
    Date        string `json:"date"`
    IsAmbiguous bool   `json:"is_ambiguous,omitempty"`
}

// EmailExtraction holds the AI-extracted data from an email.
type EmailExtraction struct {
    Summary                  string               `json:"summary"`
    SenderName               string               `json:"sender_name"`
    SenderDomain             string               `json:"sender_domain"`
    ActionItems              []ActionItem         `json:"action_items"`
    Deadlines                []Deadline           `json:"deadlines"`
    SuggestedContextKeywords []string             `json:"suggested_context_keywords"`
    Sentiment                string               `json:"sentiment"` // "positive", "neutral", "negative", "mixed"
    SuggestedContextID       *string              `json:"suggested_context_id,omitempty"`
    ContextConfidence        float64              `json:"context_confidence,omitempty"`
    SuggestNewContext        bool                 `json:"suggest_new_context,omitempty"`
    SuggestedContextTitle    string               `json:"suggested_context_title,omitempty"`
    EntityResolutions        []EntityResolution   `json:"entity_resolutions,omitempty"`
}

// ExtractedEvent represents an event extracted from text input.
type ExtractedEvent struct {
    Title       string `json:"title"`
    Description string `json:"description,omitempty"`
    Location    string `json:"location,omitempty"`
    StartsAt    string `json:"starts_at"` // RFC3339 or YYYY-MM-DD
    EndsAt      string `json:"ends_at,omitempty"`
    AllDay      bool   `json:"all_day"`
    IsAmbiguous bool   `json:"is_ambiguous"`
}

// ExtractedNote represents a note extracted from text input.
type ExtractedNote struct {
    Content       string   `json:"content"`
    SuggestedTags []string `json:"suggested_tags,omitempty"`
}

// AmbiguousReference represents a vague reference in voice input.
type AmbiguousReference struct {
    OriginalText  string `json:"original_text"`
    ReferenceType string `json:"reference_type"` // "pronoun", "vague_noun", "implicit"
}

// EntityMatch represents a candidate entity found via semantic search.
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

// TextExtraction holds the AI-extracted data from a voice capture or text input.
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
    SuggestedContextTitle    string               `json:"suggested_context_title,omitempty"`
    EntityResolutions        []EntityResolution     `json:"entity_resolutions,omitempty"`
}

// ReceiptExtraction holds structured data extracted from OCR'd receipt text.
type ReceiptExtraction struct {
    Merchant string            `json:"merchant"`
    Date     string            `json:"date"` // YYYY-MM-DD
    Total    int               `json:"total"` // cents
    Tax      int               `json:"tax"`   // cents
    Subtotal int               `json:"subtotal"` // cents
    Items    []ReceiptLineItem `json:"items"`
    Notes    string            `json:"notes,omitempty"`
}

// ReceiptLineItem is a single line item on a receipt.
type ReceiptLineItem struct {
    Description string `json:"description"`
    Amount      int    `json:"amount"`   // cents
    Quantity    int    `json:"quantity"`
}

// RelatedEntity is a lightweight summary of an entity related to a new entity (gap analysis).
type RelatedEntity struct {
    ID         string `json:"id"`
    SourceType string `json:"source_type"` // "task", "event", "note"
    Title      string `json:"title"`
    Content    string `json:"content"`
}

// GapCandidate is a single gap identified by the AI.
type GapCandidate struct {
    Category   string   `json:"category"` // "missing_contact", "missing_location", "missing_context", "missing_dependency", "missing_detail", "missing_deadline", "missing_stakeholder", "missing_outcome"
    Question   string   `json:"question"` // e.g. "What is Dr. Smith's phone number?"
    Reasoning  string   `json:"reasoning"`
    Confidence float64  `json:"confidence"` // 0-1
    RelatedIDs []string `json:"related_ids"` // IDs of related entities
}

// GapAnalysis holds the AI-identified gaps for a new entity.
type GapAnalysis struct {
    Gaps []GapCandidate `json:"gaps"`
}
```

### TieredRouter (router.go)

Routes extraction calls based on sensitivity:
- ExtractEmail → general (Claude)
- ExtractText with typeHint="transaction" → localOnly (Ollama), or skip if unavailable
- ExtractText without typeHint → general (Claude)
- ExtractReceipt → general (Claude)
- AnalyzeGaps → general (Claude) only; skipped entirely for Transaction source type (financial data local-only)

**Impact:** When Ollama is disabled and transaction is received, TextExtraction is empty (no enrichment). Gap analysis routes exclusively to Claude via sidecar; the 180s Ollama timeout serves as a circuit breaker for Claude timeouts, not a fallback target.

### ClaudeExtractor (claudecli.go)

Calls Claude Code sidecar (HTTP) with structured outputs. Key features:
- **Prompts:** Inline prompts for email, text (with type hint), receipt, gap analysis
- **Active contexts:** Serialized into prompt so Claude can reference existing contexts
- **User correction:** Included in prompt to guide re-extraction
- **Semantic candidates:** Serialized into prompt (up to 5) for entity resolution; Content field populated from embeddingbus.SearchResult.Content
- **Structured output:** JSON schema validation
- **Timeout:** 60s per call
- **Sanitization (gap analysis only):** entityContent is sanitized via `business/sdk/sanitize.Sanitize` (regex-only PII scrub: SSN, phone, card, routing, account numbers) before being sent to Claude for gap analysis
- **Gap categories (8 total):** missing_contact, missing_location, missing_context, missing_dependency, missing_detail, missing_deadline, missing_stakeholder, missing_outcome

### OllamaExtractor (ollama.go)

Calls local Ollama API. Recent change (refactored JSON parsing):
- **parseOllamaJSON()** — improved parsing to handle markdown code fences (```json ... ```)
- Strips opening fence (`````json` or ` ``` `), uses `strings.CutSuffix()` for safer closing fence removal
- Handles whitespace robustly (leading/trailing), both ```json and ``` formats
- Better error messages with truncated raw response for debugging
- Used for transaction enrichment and gap analysis fallback

### Email/Text Parser (parse.go)

```go
// ParsedEmail holds the parsed components of an email message.
type ParsedEmail struct {
    MessageID   string
    FromAddress string
    FromName    string
    ToAddress   string
    Subject     string
    BodyText    string
    BodyHTML    string
}

func parseEmail(rawContent string) (ParsedEmail, error) // RFC 5322 parser
func parseEmailEntity(entity *message.Entity) (ParsedEmail, error) // go-message entity parser
```

**Impact:** ParsedEmail is input to AI extraction (subject + bodyText only; BodyHTML is fallback).

### Cleanup & Classify (cleanup/, classify/)

- **cleanup.StripFillers()** — remove filler words from text before classification
- **cleanup.SplitClauses()** — split text into grammatical clauses for per-clause processing
- **classify.Classify()** — classify clause as task, note, or event with confidence

**Impact:** Text ingestion is per-clause; low-confidence classifications generate TypeAssignment clarifications.

## App Layer

### Path
`app/domain/voiceingestapp/` and `app/domain/reingestapp/`

### HTTP Handlers

**voiceingestapp** (`app/domain/voiceingestapp/voiceingestapp.go`):
- **POST /ingest/voice** — enqueue voice text for async processing
  - Request: `{ "text": "..." }`
  - Response: `{ "raw_input_id": "uuid" }`
  - Calls: ingestbus.EnqueueText() → returns raw_input.ID

**reingestapp** (`app/domain/reingestapp/reingestapp.go`):
- **POST /reingest/task/{task_id}** — reingest a task
  - Queries task, synthesizes raw_input if needed, sets skip_classify based on context_id, resets raw_input for processing
  - Response: `{ "raw_input_id": "uuid", "skip_classify": bool, "enqueued": true }`
- **POST /reingest/note/{note_id}** — reingest a note
  - Response: `{ "raw_input_id": "uuid", "skip_classify": bool, "enqueued": true }`
- **POST /reingest/event/{event_id}** — reingest an event
  - Response: `{ "raw_input_id": "uuid", "skip_classify": bool, "enqueued": true }`
- **POST /reingest/bulk** — bulk reingest tasks, notes, or events
  - Request: `{ "entityType": "task|note|event", "contextId": "uuid|empty" }`
  - Response: `{ "queued": int }`

### Request/Response DTOs

**voiceingestapp:**
```go
type ingestRequest struct {
    Text string `json:"text"`
}

type ingestResponse struct {
    RawInputID string `json:"raw_input_id"`
}
```

**reingestapp:**
```go
type ReingestResponse struct {
    RawInputID  string `json:"raw_input_id"`
    SkipClassify bool  `json:"skip_classify"`
    Enqueued    bool  `json:"enqueued"`
}

type BulkReingestRequest struct {
    EntityType string `json:"entityType"` // "task", "note", "event"
    ContextID  string `json:"contextId,omitempty"`
}

type BulkReingestResponse struct {
    Queued int `json:"queued"`
}
```

### Helper Methods (reingestapp)

- **synthesizeRawInputForTask/Note/Event()** — create a raw_input from entity content, update entity with raw_input_id
- **buildTaskContent() / buildEventContent()** — combine title + description for raw_input
- **resetRawInput()** — call ResetForReingest (skip_classify=true) or ResetForReprocess (skip_classify=false)
- **queryTasksForBulkReingest() / queryNotesForBulkReingest() / queryEventsForBulkReingest()** — fetch entities by context_id (optional)

## Database Schema

### raw_inputs Table

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

**Extended columns** (added in rawinputbus):
- user_correction TEXT — user-provided correction from clarifications
- skip_classify BOOLEAN — skip extraction on reingest (set by ResetForReingest)
- reingest_mode BOOLEAN — preserve confirmed state during reingest
- source_entity_id UUID — for synthetic raw inputs (task, note, event being reingestd)
- source_entity_kind TEXT — "task", "note", "event" (for skip_classify path)
- result JSONB — PipelineResult (per-step observability)

### emails Table

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
CREATE UNIQUE INDEX idx_emails_message_id ON emails(message_id) WHERE message_id IS NOT NULL;
```

## Impact Callouts

### ProcessEmail & ProcessText: Context Matching
- Fetches ALL active contexts (limit 50) on every email/text ingestion
- Matches by explicit context_id from extraction, then keyword fuzzy match, then auto-creates new
- **Impact on contextbus:** High read volume; consider caching active contexts or pagination if >50 contexts exist

### ProcessEmail & ProcessText: Semantic Search
- Calls embeddingbus.Search() with clause/email content + types=["event", "task", "note"]
- Filters candidates by similarity >= 0.70 (minCandidateSimilarity const)
- **Impact on embeddingbus:** One search per email, one per clause for text (may be many)
- **Impact on failure:** Logs warning but continues without candidates; non-fatal

### Task/Event/Note Creation: Unconfirmed Flag
- Tasks/events created from text ingestion with low-classification-confidence are marked Unconfirmed=true
- Tasks/notes from email ingestion are marked Unconfirmed=false (emails are trusted)
- **Impact on frontend:** Unconfirmed entities should be highlighted for user review

### Entity Resolution: Update vs. Create
- AI's EntityResolution can return "update" (change existing), "create" (new entity), or "ambiguous" (unclear)
- applyEntityUpdate() sets Unconfirmed=true on updated entities → user must confirm changes
- **Impact on tasks/events:** Entity updates are non-destructive; user sees them as pending confirmation

### Clarifications: Volume
- Low-confidence context matches → ContextAssignment clarification
- Ambiguous action items → AmbiguousAction clarification (per action item)
- Ambiguous deadlines → AmbiguousDeadline clarification (per deadline)
- New context auto-creation → NewContext clarification
- Entity resolutions → AmbiguousEntityMatch clarification
- Type assignments (text) → TypeAssignment clarification (per low-confidence clause)
- Voice references → VoiceReference clarification (per ambiguous reference)
- **Skipped entirely for Transaction source type** (sensitive financial data stays local-only, matches gap detection behavior)
- **Impact on clarificationbus:** High volume of Upsert calls; clarifications are keyed by kind+subject to avoid duplicates

### Knowledge Gap Detection: Async
- Fires after all entities are created (background goroutine)
- Skipped entirely for Transaction source type (sensitive financial data stays local-only)
- For other sources, routes exclusively to Claude via sidecar; entityContent is sanitized before submission
- Related-entity Content populated from semantic search results (embeddingbus.SearchResult.Content)
- Calls knowledgegapbus.Detect(context.Background(), ...) per entity and collects results (CardsCreated, Skipped)
- Updates raw_input.result.GapAnalysis with StepResult: Status ("completed"/"partial"/"failed"), Detail (total_cards_created, total_skipped, entity_count, errors if any)
- **Impact:** Gap analysis outcome is tracked in pipeline result for observability; user sees gaps later via clarifications

### ⚠ GapAnalysis Field in PipelineResult (ingestbus.go:67)
Changing this field affects:
- `ingestbus.go:521-584` (email gap detection goroutine) — collects gap detection results and updates raw_input.result
- `ingestbus.go:1214-1277` (text gap detection goroutine) — same logic, queries raw_input and merges existing result before updating
- `rawinputbus.Update()` call in both goroutines — must pass Result field as raw json.RawMessage
- Tests verifying gap analysis status and detail fields are persisted in raw_input.result

### Reingest: skip_classify Path
- When task/note/event is reingested with existing context, skip_classify=true
- processSkipClassify() loads entity, deletes old embeddings, regenerates embeddings + gaps only
- Avoids re-extracting / re-classifying confirmed entities
- **Impact on embeddings:** If embeddings have changed (gap detection side effects), gaps are regenerated

## Routes

### Wired in `api/services/planner/main.go`

```go
// Lines 242, 288, 300
igBus := ingestbus.NewBusiness(log, riBus, emBus, taskBus, ctxBus, clarBus, evtBus, ext, noteBus, tgBus)

Routes: {
    voiceingestapp.Routes{},  // POST /ingest/voice
    reingestapp.Routes{},      // POST /reingest/{task,note,event}
}
```

**Note:** Email ingestion is async via background worker (SMTP ingest handler calls EnqueueEmail). No HTTP endpoint for email — emails are pushed via SMTP integration.

## Cross-Domain Dependencies

| Domain | Used In | Purpose |
|--------|---------|---------|
| **rawinputbus** | ProcessEmail, ProcessText, ProcessRawInputByID, Reprocess | Central queue for all ingestion; persist raw content and status |
| **emailbus** | ProcessEmail | Parse + store email records, dedup, associate with context |
| **taskbus** | ProcessText, reingestapp | Create tasks from action items, mark Unconfirmed, delete on reingest |
| **eventbus** | ProcessText, reingestapp | Create events, mark Unconfirmed, delete on reingest |
| **notebus** | ProcessText, reingestapp | Create notes, mark Unconfirmed, delete on reingest |
| **contextbus** | ProcessEmail, ProcessText | Query active contexts, verify suggested contexts, auto-create |
| **clarificationbus** | ProcessEmail, ProcessText | Generate low-confidence match/action/deadline/reference clarifications |
| **tagbus** | ProcessText | Fetch/create tags suggested by extraction |
| **embeddingbus** (optional) | ProcessEmail, ProcessText | Semantic search for candidates, embed + store content, delete on reingest |
| **knowledgegapbus** (optional) | ProcessEmail, ProcessText | Fire async gap detection |

**Inbound callers:**
- **SMTP handler** → EnqueueEmail() (background worker)
- **Voice endpoint** (voiceingestapp) → EnqueueText()
- **Reingest endpoints** (reingestapp) → Reprocess or ProcessRawInputByID
- **Clarification feedback** (clarificationbus) → calls Reprocess with UserCorrection
- **Background worker** → ProcessRawInputByID (called by job queue)

## Testing

- `business/domain/ingestbus/ingestbus_test.go` — integration tests using real Postgres (dbtest)
- `app/domain/voiceingestapp/tests/voiceingestapi/voiceingest_test.go` — HTTP API tests (apitest)
- `app/domain/reingestapp/tests/reingestapi/reingest_test.go` — HTTP API tests (apitest)
- `business/domain/ingestbus/extractor/` — per-extractor tests (mock, Ollama JSON parsing, router)

## Frontend Integration

Tygo exports ingest types via `business/domain/ingestbus/extractor/model.go`. Frontend auto-generates:
- `ActionItem`, `Deadline`, `EmailExtraction`, `TextExtraction`, `ReceiptExtraction`, `GapAnalysis`, etc.

Used in:
- Clarification UI (display extracted content, multiple interpretations)
- Reingest UI (trigger reingest, view skip_classify mode)
- Pipeline observability (view per-step results from PipelineResult)
