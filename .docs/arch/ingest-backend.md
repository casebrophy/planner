# Ingest Business System

> The ingestion pipeline receives raw input (emails, voice, manual text) and transforms it into structured entities (tasks, events, notes) via AI extraction, semantic matching, and knowledge gap detection. The system manages raw_input state transitions (pending → processing → processed/partial/failed), sanitizes PII before external APIs, performs context matching and clarification generation, and fires asynchronous gap detection to populate the clarification queue.

## Core Types

### IngestResult
```go
type IngestResult struct {
    TaskIDs  []uuid.UUID
    EventIDs []uuid.UUID
    NoteIDs  []uuid.UUID
}
```
Holds the IDs of created tasks, events, and notes from a single ingestion run (used by ProcessText).

### PipelineResult
```go
type PipelineResult struct {
    Sanitize     *StepResult `json:"sanitize,omitempty"`
    Extraction   *StepResult `json:"extraction,omitempty"`
    ContextMatch *StepResult `json:"contextMatch,omitempty"`
    Tasks        *StepResult `json:"tasks,omitempty"`
    Events       *StepResult `json:"events,omitempty"`
    Notes        *StepResult `json:"notes,omitempty"`
    GapAnalysis  *StepResult `json:"gapAnalysis,omitempty"`
}
```
Tracks per-step pipeline outcomes for observability; stored in raw_input.result JSON.

### StepResult
```go
type StepResult struct {
    Status string         `json:"status"` // "completed", "failed", "skipped"
    Detail map[string]any `json:"detail,omitempty"`
}
```
Records the outcome of a single pipeline step (status + optional detail map for counts, IDs, errors).

### GapDetector Interface
```go
type GapDetector interface {
    Detect(ctx context.Context, entityType string, entityID uuid.UUID, content string) (knowledgegapbus.GapDetectionResult, error)
}
```
Abstracts knowledge gap detection; plugged via WithGapDetector().

### Business
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
    embeddingBus     *embeddingbus.Business
    gapBus           GapDetector
}
```
Main orchestrator. Composes multiple domain buses and the Extractor interface.

### ParsedEmail
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
Parsed RFC 5322 email message components extracted via go-message MIME parsing.

## File Map

### Models
- `ingestbus.go` (lines 39–135) — IngestResult, PipelineResult, StepResult, GapDetector interface, Business struct, NewBusiness(), WithEmbedder(), WithGapDetector()
- `parse.go` (lines 14–94) — ParsedEmail struct, parseEmail(), parseEmailEntity() (RFC 5322 parsing via go-message)
- `extractor/model.go` — Extractor interface, ContextRef, ActionItem, Deadline, EmailExtraction, ExtractedEvent, ExtractedNote, AmbiguousReference, EntityMatch, EntityResolution, TextExtraction, ReceiptExtraction, RelatedEntity, GapCandidate, GapAnalysis
- `classify/classifier.go` — ItemType (TaskType, EventType, NoteType), Classification struct, Classify() (heuristic text classification)

### Core Logic
- `ingestbus.go`:
  - **ProcessEmail()** (line 137) — Entry point: stores raw_input, calls processRawInput
  - **ProcessText()** (line 608) — Entry point: stores raw_input, calls processTextInput, returns IngestResult
  - **EnqueueEmail()** (line 633) — Async enqueue: stores raw_input, returns ID; background worker processes later
  - **EnqueueText()** (line 646) — Async enqueue: stores raw_input, returns ID; background worker processes later
  - **Reprocess()** (line 160) — Re-run pipeline on existing raw_input (with optional user correction)
  - **ProcessRawInputByID()** (line 747) — Called by background worker; routes to email or text pipeline based on source_type
  - **processRawInput()** (line 182) — Email pipeline: parse → dedup → store → sanitize → extract → embed → context match → create tasks → gap detect → mark processed/partial
  - **processTextInput()** (line 783) — Text/voice pipeline: sanitize → cleanup → per-clause classify+extract → context match → create tasks/events/notes → gap detect → mark processed/partial
  - **processSkipClassify()** (line 659) — Reingest path: loads entity by kind, regenerates embeddings and gaps (skips extraction/classify)
  - **applyEntityUpdate()** (line 1423) — Updates task/event when extraction suggests update (marks Unconfirmed=true)
  - **createAmbiguousMatchClarification()** (line 1464) — Creates AmbiguousEntityMatch clarification when entity resolution is uncertain
  - **matchContextByKeywords()** (line 1490) — Fallback context matching via keyword substring search
- `parse.go`:
  - **parseEmail()** (line 26) — RFC 5322 MIME parsing using go-message; returns ParsedEmail (FromAddress, Subject, BodyText, BodyHTML, MessageID)
  - **parseEmailEntity()** (line 96) — Parse go-message.Entity into ParsedEmail
- `extractor/*` — AI extraction layer (Claude Code sidecar via extractor.Extractor interface); not defined in ingestbus, abstracted as dependency
- `classify/classifier.go`:
  - **Classify()** (line 61) — Heuristic classification of text clause into Task/Event/Note with confidence 0–1; uses obligation verbs, temporal anchors, reference patterns (phone, email, address)
- `cleanup/cleanup.go` — Text preprocessing: StripFillers(), SplitClauses()

## Impact Callouts

### ⚠ Business.ProcessEmail / ProcessText
Changing these affects:
- `app/domain/ingestapp/` — HTTP handlers that call these methods
- `zarf/` — Background worker that calls ProcessRawInputByID()
- All downstream buses (taskBus, emailBus, eventBus, noteBus, contextBus, clarificationBus)

### ⚠ PipelineResult
Changing affects:
- `raw_input.result` JSON schema — altering fields breaks observability
- Frontend dashboard that visualizes pipeline execution traces
- Any monitoring/alerting on pipeline step outcomes

### ⚠ GapDetector Interface
Changing affects:
- `knowledgegapbus.Business` — must implement Detect() signature
- Gap detection callback path (async goroutine in ProcessEmail/ProcessText)
- Clarification generation for gap candidates

### ⚠ Extractor Interface
Changing affects:
- `extractor/*` — all implementations (Claude Code sidecar wrapper, mock, etc.)
- Extraction prompt/output shape (ActionItem, Deadline, EntityResolution)
- Context matching logic (SuggestedContextID, ContextConfidence, SuggestNewContext)

### ⚠ Classify() Heuristic
Changing patterns/confidence thresholds affects:
- TypeAssignment clarification triggers (unconfirmed < 0.75)
- Per-clause routing in processTextInput (Task vs Event vs Note creation)
- Downstream entity creation confidence

## Cross-Domain Dependencies

### Outbound
- **rawinputbus.Business** — Create(), MarkProcessing(), MarkProcessed(), MarkPartial(), MarkFailed(), QueryByID(), Update()
- **emailbus.Business** — Create(), Update(), QueryByMessageID()
- **taskbus.Business** — Create(), Update(), QueryByID()
- **eventbus.Business** — Create(), Update(), QueryByID()
- **notebus.Business** — Create(), QueryByID()
- **contextbus.Business** — Query() (active contexts), QueryByID(), Create() (auto-create)
- **tagbus.Business** — Query(), Create(), AddToNote()
- **clarificationbus.Business** — Upsert() (generates clarification items for context match, ambiguous actions, deadlines, entity resolution, new context, type assignment, voice references)
- **embeddingbus.Business** — Search() (pre-extraction semantic lookup), EmbedAndStore() (post-creation embeddings), DeleteBySource() (reingest cleanup)
- **knowledgegapbus.Business** (GapDetector) — Detect() (async knowledge gap detection per created entity)
- **extractor.Extractor** — ExtractEmail(), ExtractText() (AI extraction via Claude Code sidecar)
- **sanitize** SDK — Sanitize() (PII detection/redaction before external APIs)
- **cleanup** — StripFillers(), SplitClauses() (text preprocessing)
- **classify** — Classify() (per-clause heuristic classification)

### Inbound
- **app/domain/ingestapp/** — Routes that call ProcessEmail(), ProcessText(), EnqueueEmail(), EnqueueText(), Reprocess()
- **Background worker** (zarf/) — Calls ProcessRawInputByID() to process queued raw_inputs
- **MCP server** — May expose ProcessEmail/ProcessText via /mcp routes

## Processing Pipeline

### Email Pipeline (ProcessEmail / processRawInput)
1. **Store raw_input** — Create raw_input record with status=pending
2. **Parse email** — RFC 5322 MIME parsing → ParsedEmail (FromAddress, Subject, BodyText, BodyHTML, MessageID)
3. **Dedup check** — Query emailBus.QueryByMessageID(); skip if already exists
4. **Store email record** — Create email record linked to raw_input
5. **Fetch active contexts** — Load all active contexts for AI prompt context
6. **Sanitize PII** — Redact before external API (subject, body)
7. **Pre-extraction semantic lookup** — Search embeddings for candidate entities; filter by 0.70 similarity
8. **AI extraction** — Call extractor.ExtractEmail() → EmailExtraction (summary, action items, deadlines, suggested context)
9. **Embed extracted content** — Store embeddings for the email
10. **Context matching** — Check suggested context ID, fallback to keyword fuzzy match, optionally auto-create context
11. **Clarification generation** — Upsert clarifications for low-confidence context, ambiguous actions, ambiguous deadlines, entity resolutions
12. **Entity resolution** — Apply "update" or "ambiguous" resolution decisions; mark affected entities Unconfirmed=true
13. **Create tasks** — Per action item, create task with matched context, collect gapTargets
14. **Knowledge gap detection** — Async goroutine: Detect() per created entity, update raw_input.result with gap analysis
15. **Mark raw_input** — Mark processed or partial (if task creation failures)

### Text/Voice Pipeline (ProcessText / processTextInput)
1. **Store raw_input** — Create raw_input record with status=pending
2. **Fetch active contexts** — Load all active contexts for AI prompt context
3. **Sanitize PII** — Redact before external API
4. **Cleanup** — StripFillers() + SplitClauses() → per-clause processing
5. **Per-clause classify + extract** — For each clause:
   - Classify() heuristic (Task/Event/Note confidence)
   - Pre-extraction semantic search (candidates)
   - ExtractText() → TextExtraction (action items, events, notes, entity resolutions)
6. **Merge clause results** — Aggregate action items, keywords, best context, entity resolutions
7. **Context matching** — Similar to email pipeline
8. **Low-confidence type clarification** — Upsert TypeAssignment clarification if clause confidence < 0.75 (unless reingestMode=true)
9. **Entity resolution** — Apply "update" or "ambiguous" decisions
10. **Create tasks, events, notes** — Per clause, create entities with matched context, collect gapTargets
11. **Embed created entities** — Store embeddings for tasks, events, notes
12. **Generate clarifications** — Ambiguous actions, deadlines, entity matches, voice references
13. **Knowledge gap detection** — Async goroutine: Detect() per created entity, update raw_input.result with gap analysis
14. **Mark raw_input** — Mark processed or partial

### Reingest Path (skip_classify=true)
- Skip extraction/classify entirely
- Load entity by kind (task, event, note)
- Delete old embeddings, regenerate new embeddings
- Fire knowledge gap detection
- Mark processed

### Async Gap Detection
- Runs in background (context.Background(), not tied to request lifetime)
- Per-entity: Detect(ctx, entityType, entityID, content)
- Collects results: totalCardsCreated, totalSkipped, errors
- Merges with existing PipelineResult in raw_input.result
- Does NOT fire for Transaction source_type
