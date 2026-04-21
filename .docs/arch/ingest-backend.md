# Ingest Backend System

> Orchestrates document and voice ingestion from multiple sources (email, voice capture, manual re-ingestion via reingest endpoints). Transforms raw input into structured entities (tasks, events, notes) via AI extraction with per-clause cleanup and classification, semantic candidate matching, context linking, and asynchronous knowledge gap detection. Manages raw_input state transitions (pending → processing → processed/partial/failed), sanitizes PII before external APIs, generates clarifications for low-confidence matches and ambiguous references, and supports reingest workflows via skip_classify mode to preserve confirmed entity states.

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
- `business/domain/ingestbus/ingestbus.go` (lines 39–135) — IngestResult, PipelineResult, StepResult, GapDetector interface, Business struct, NewBusiness(), WithEmbedder(), WithGapDetector()
- `business/domain/ingestbus/parse.go` (lines 14–94) — ParsedEmail struct, parseEmail(), parseEmailEntity() (RFC 5322 parsing via go-message)
- `business/domain/ingestbus/extractor/model.go` — Extractor interface, ContextRef, ActionItem, Deadline, EmailExtraction, ExtractedEvent, ExtractedNote, AmbiguousReference, EntityMatch, EntityResolution, TextExtraction, ReceiptExtraction, RelatedEntity, GapCandidate, GapAnalysis
  - **GapCandidate (new fields Phase 4)** — `Options: []string` for enumerable answer choices, `OptionsConfidence: float64` for confidence in the option set (0 if open-ended)
- `business/domain/ingestbus/classify/classifier.go` — ItemType (TaskType, EventType, NoteType), Classification struct, Classify() (heuristic text classification)
- `business/domain/ingestbus/cleanup/cleanup.go` — ClauseRole enum, Clause struct, StripFillers(), SplitClauses(), expandCommaList(), DetectSubordinateClause(), SplitClausesWithRoles() (Phase 4: clause detection with subordinate/expanded-from-comma-list roles)

### Handlers
- `app/domain/reingestapp/reingestapp.go` — **reingestTask()**, **reingestNote()**, **reingestEvent()** (per-entity reingest with optional raw_input synthesis), **reingestBulk()** (bulk reingest for tasks/notes/events)
  - Helper: **synthesizeRawInputForTask/Note/Event()** (creates raw_input from entity content)
  - Helper: **dismissStaleClarifications()** (dismisses pending/snoozed clarifications on reingest)
  - Helper: **resetRawInput()** (calls ResetForReingest or ResetForReprocess + sets reingest_mode)
- `app/domain/voiceingestapp/voiceingestapp.go` — **ingest()** (enqueues voice text input, returns raw_input ID for async processing)

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

### ⚠ Clause Splitting & Role Detection (Phase 4)
Changing SplitClausesWithRoles(), expandCommaList(), or DetectSubordinateClause() affects:
- Per-clause extraction granularity (main vs subordinate vs expanded items)
- Entity creation count per input (single task vs distributed across siblings)
- Sibling index tracking in gapTargets (for multi-entity gap detection)
- Reingest preserve logic: reingestMode=true suppresses unconfirmed flip on clause updates

### ⚠ GapCandidate & Prompt Guidance (Phase 4)
Changing GapCandidate struct or BuildGapAnalysisPrompt affects:
- `business/domain/ingestbus/extractor/prompt.go:386-396` — Options guidance section (enumerable vs open-ended decision logic)
- `business/domain/ingestbus/extractor/prompt.go:399-411` — JSON schema that includes options and options_confidence fields
- All Extractor implementations (Claude Code sidecar, mock, ollama) — must produce Options and OptionsConfidence in JSON response
- Clarification generation downstream (gap cards may reference options for user selection)
- Frontend rendering of gap cards — must handle optional options array and confidence score

### ⚠ BuildGapAnalysisPrompt Meta-Question Filtering (Phase 4)
Changing the "Explicitly forbidden" list affects:
- `business/domain/ingestbus/extractor/prompt_test.go:213–255` — TestBuildGapAnalysisPrompt_RejectsMeta_* tests
- Prompt quality and relevance (must avoid meta-questions, duplicates, hygiene observations)
- AI consistency across extraction runs

### ⚠ Reingest Workflow (reingestapp handlers)
Changing reingestTask/Note/Event or resetRawInput affects:
- Stale clarification dismissal logic (raw_input → entity)
- Raw input synthesis for entities without RawInputID
- skip_classify determination (ContextID != nil → skip extraction)
- reingest_mode flag behavior (suppresses unconfirmed flip, preserves user confirmations)

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
- **app/domain/reingestapp/route.go** — Routes that call reingestTask(), reingestNote(), reingestEvent(), reingestBulk()
- **app/domain/voiceingestapp/route.go** — Routes that call ingest() (EnqueueText internally)
- **Background worker** (business/sdk/worker/ingestworker.go) — Calls ProcessRawInputByID() to process queued raw_inputs
- **smtpbus** — SMTP server integration that calls ingestBus.ProcessEmail() for incoming emails

## Routes

| Method | Path | Handler | Notes |
|--------|------|---------|-------|
| POST | `/api/v1/tasks/{task_id}/reingest` | reingestapp.reingestTask() | Re-run pipeline on task, optionally skip_classify if confirmed |
| POST | `/api/v1/notes/{note_id}/reingest` | reingestapp.reingestNote() | Re-run pipeline on note |
| POST | `/api/v1/events/{event_id}/reingest` | reingestapp.reingestEvent() | Re-run pipeline on event |
| POST | `/api/v1/reingest/bulk` | reingestapp.reingestBulk() | Bulk reingest tasks/notes/events, filtered by entity type and optional context |
| POST | `/api/v1/ingest/voice` | voiceingestapp.ingest() | Enqueue voice/text input for async processing |

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
4. **Cleanup** (Phase 4) — StripFillers() + SplitClausesWithRoles() → per-clause processing with role metadata
   - Detects subordinate clauses (when/while/if + subject pronoun + action verb)
   - Expands comma lists with leading verbs ("buy milk, bread, eggs" → ["buy milk", "buy bread", "buy eggs"])
   - Returns Clause[] with Role (main/subordinate/expanded_from_comma_list) and SiblingIdx
5. **Per-clause classify + extract** — For each Clause:
   - Classify() heuristic (Task/Event/Note confidence)
   - Pre-extraction semantic search (candidates)
   - ExtractText(typeHint=string(cl.Type)) → TextExtraction (action items, events, notes, entity resolutions)
   - Collects clauseWithIdx for role-aware processing
6. **Merge clause results** — Aggregate action items, keywords, best context, entity resolutions across all clauses
7. **Context matching** — Similar to email pipeline (best context ID + confidence, keyword fallback, auto-create)
8. **Low-confidence type clarification** (Phase 4) — Upsert TypeAssignment clarification if:
   - Clause confidence < 0.75 AND
   - reingestMode=false (skipped during reingest to preserve confirmed state)
9. **Entity resolution** — Apply "update" or "ambiguous" decisions per clause
10. **Create tasks, events, notes** (Phase 4) — Per clause, create entities with:
    - matched context
    - description fallback: if item.Description empty and matchedContextID exists, use context.Description
    - unconfirmed flag: true if clause confidence < 0.75 (unless reingestMode=true)
    - collect gapTargets: (entityType, entityID, content) tuples
11. **Embed created entities** — Store embeddings for tasks, events, notes
12. **Generate clarifications** — Ambiguous actions, deadlines, entity matches (Phase 4), voice references
13. **Knowledge gap detection** — Async goroutine: Detect() per created entity, update raw_input.result with gap analysis
14. **Mark raw_input** — Mark processed or partial

### Reingest Path (skip_classify=true) (Phase 4)
Triggered via reingestapp handlers when entity already has ContextID (skip_classify=true):
1. **Dismiss stale clarifications** — Find all pending/snoozed clarifications tied to raw_input or entity, dismiss them
2. **Synthesize raw_input** (if needed) — If entity has no RawInputID, create one from entity content (task title+desc, event title+desc, note content)
3. **Set reingest_mode=true** — Either via ResetForReingest() or ResetForReprocess() + explicit update
4. **Call processSkipClassify()** — Load entity by kind, delete old embeddings, regenerate embeddings, fire gap detection
5. **Mark processed** — Don't extract/classify; preserve confirmed entity state; gap detection runs async in background

### Async Gap Detection
- Runs in background (context.Background(), not tied to request lifetime)
- Per-entity: Detect(ctx, entityType, entityID, content)
- Collects results: totalCardsCreated, totalSkipped, errors
- Merges with existing PipelineResult in raw_input.result
- Does NOT fire for Transaction source_type

## Gap Analysis & Options (Phase 4)

Gap detection via BuildGapAnalysisPrompt follows strict meta-question filtering and options guidance:

### Meta-Question Filtering
- **Forbidden:** "Should we consolidate?", "Why multiple copies?", "Is this a duplicate?", "Consider recurrence pattern", "May be redundant"
- **Forbidden:** Observations about data quality, system hygiene, organizational structure
- **Allowed:** References to related entities for dependencies/stakeholders/context (e.g., "Could Task A depend on Task B?")

### Options Guidance
Each GapCandidate must classify its question as enumerable or open-ended:

| Type | Example | Options | OptionsConfidence |
|------|---------|---------|------------------|
| Enumerable | "Is the project timeline...?" | ["flexible", "fixed deadline", "unknown"] | 0.9 |
| Enumerable | "Where is the meeting?" | ["building A", "building B", "remote", "unknown"] | 0.8 |
| Open-ended | "What is the project budget?" | [] | 0 |
| Open-ended | "Who are all the stakeholders?" | [] | 0 |

The prompt (lines 386–396) guides the AI to populate options and options_confidence; downstream clarifications may render as multiple-choice when options are present.
