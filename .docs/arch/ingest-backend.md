# Ingest Backend System

> Orchestrates document and voice ingestion from multiple sources (email, voice capture, manual re-ingestion via reingest endpoints). Transforms raw input into structured entities (tasks, events, notes) via AI extraction with per-clause cleanup and classification, semantic candidate matching, context linking, and synchronous knowledge gap detection. Manages raw_input state transitions (pending → processing → processed/partial/failed), sanitizes PII before external APIs, generates clarifications for low-confidence matches and ambiguous references, and supports reingest workflows via skip_classify mode to preserve confirmed entity states.

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

### FailureKind & Failure (Phase 4)
```go
type FailureKind string

const (
    PrimaryType      FailureKind = "primary_type"
    TitleContains    FailureKind = "title_contains"
    ContextID        FailureKind = "context_id"
    ContextKind      FailureKind = "context_kind"
    MinActionItems   FailureKind = "min_action_items"
    MaxActionItems   FailureKind = "max_action_items"
    ForbidNotes      FailureKind = "forbid_notes"
    ForbidEvents     FailureKind = "forbid_events"
    MinContextConf   FailureKind = "min_context_confidence"
    MaxContextConf   FailureKind = "max_context_confidence"
)

type Failure struct {
    Kind    FailureKind
    Message string
}
```
Typed failure enumeration for classification eval assertions; replaces string-based failure list to enable structured failure reporting.

## File Map

### Models
- `business/domain/ingestbus/ingestbus.go` (lines 39–135) — IngestResult, PipelineResult, StepResult, GapDetector interface, Business struct, NewBusiness(), WithEmbedder(), WithGapDetector(); clarification source_hash propagation via ComputeSourceHash()
- `business/domain/ingestbus/parse.go` (lines 14–94) — ParsedEmail struct, parseEmail(), parseEmailEntity() (RFC 5322 parsing via go-message)
- `business/domain/ingestbus/extractor/model.go` — Extractor interface, ContextRef (with `Kind` field), ActionItem, Deadline, EmailExtraction, ExtractedEvent, ExtractedNote, AmbiguousReference, EntityMatch, EntityResolution, TextExtraction, ReceiptExtraction, RelatedEntity, GapCandidate, GapAnalysis
  - **ContextRef (new field)** — `Kind: string` (project|area|list) for context type hints in extraction prompts
  - **TextExtraction (new fields Phase 4)** — `ReclassifiedAs: string` (task|event|note override when heuristic incorrect), `SuggestedNewContextKind: string` (project|area|list when suggest_new_context=true)
  - **GapCandidate (new fields Phase 4)** — `Options: []string` for enumerable answer choices, `OptionsConfidence: float64` for confidence in the option set (0 if open-ended)
- `business/domain/ingestbus/classify/classifier.go` — ItemType (TaskType, EventType, NoteType), Classification struct, Classify() (heuristic text classification; broadened imperative patterns: get rid of, clean out/up, throw away/toss, need to, boosted should)
- `business/domain/ingestbus/cleanup/cleanup.go` — ClauseRole enum, Clause struct, StripFillers(), SplitClauses(), expandCommaList(), DetectSubordinateClause(), SplitClausesWithRoles() (Phase 4: clause detection with subordinate/expanded-from-comma-list roles)
- `business/domain/ingestbus/eval/` **(new Phase 4)** — Classification evaluation harness
  - `eval.go` — FailureKind enum (primary_type, title_contains, context_id, context_kind, min/max_action_items, forbid_notes, forbid_events, min/max_context_confidence), Failure struct, FixtureContext, FixtureExpected, FixtureResult, LoadFixtures(), RunSuite(), assert() (typed failures), metric aggregation (pass rate, latency)
  - `metrics.go` — pass/fail scoring against FixtureExpected fields (primary_type, title_contains, context_id, context_kind, min/max action items, forbid_notes, forbid_events, context confidence bounds)
  - `testdata/classification/` — 13 JSON fixtures covering tasks (imperative, passive), events (time-anchored), notes, lists, context assignment, ambiguity

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
  - **Reprocess()** (line 160) — Re-run pipeline on existing raw_input (with optional user correction); routes by source_type: Email → processRawInput, Voice/Manual → processTextInput, else error
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
  - `extractor/model.go` — Extractor interface, ContextRef, ActionItem, Deadline, EmailExtraction, TextExtraction, ReceiptExtraction, EntityMatch, EntityResolution, RelatedEntity, GapCandidate, GapAnalysis
  - `extractor/claudecli.go` — Claude Code sidecar wrapper (calls foundation/claudecli); JSON schemas for emailExtraction, gapAnalysis (lines 37–113)
    - **gapAnalysisSchema (updated)** — Gap candidates now include `options` (array of string) and `options_confidence` (number) fields for enumerable vs open-ended guidance
  - `extractor/ollama.go` — Ollama local model implementation
  - `extractor/mock.go` — Mock implementation for testing
  - `extractor/router.go` — Routes to configured backend (Claude Code / Ollama / mock)
  - `extractor/failover.go` — Failover logic between implementations
  - `extractor/prompt.go` — Prompt builder for extraction (threads typeHint, typeHintConfidence, reclassification locks)
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

### ⚠ GapDetector Interface & JSON Schema (claudecli.go)
Changing affects:
- `knowledgegapbus.Business` — must implement Detect() signature
- Gap detection path (synchronous; runs inline before MarkProcessed in ProcessEmail/ProcessText)
- Clarification generation for gap candidates
- gapAnalysisSchema in claudecli.go must include `options` and `options_confidence` fields
- All Extractor implementations (Claude Code, mock, ollama) must parse and emit options/options_confidence

### ⚠ Extractor Interface
Changing affects:
- `extractor/*` — all implementations (Claude Code sidecar wrapper, mock, ollama, failover, router)
- `ExtractText()` signature (line 8) — now includes `typeHint: string` + `typeHintConfidence: float64` parameters threaded through all implementation layers
- Extraction prompt/output shape (ActionItem, Deadline, EntityResolution, ReclassifiedAs, SuggestedNewContextKind)
- Context matching logic (SuggestedContextID, ContextConfidence, SuggestNewContext, SuggestedContextKind for auto-create)
- ContextRef struct now includes `Kind` field (project|area|list) for context hints in extraction prompts

### ⚠ Classify() Heuristic
Changing patterns/confidence thresholds affects:
- Heuristic match patterns (obligation verbs: need, must, should, have to, get rid of, clean out/up, throw away, toss; temporal anchors: at/on/in + time)
- TypeAssignment clarification triggers (unconfirmed < 0.75)
- Per-clause routing in processTextInput (Task vs Event vs Note creation)
- Downstream entity creation confidence
- Extraction reclassification lock strength (low confidence softens locks, high confidence strengthens them)
- **Type suppression threshold** (line 1076: heuristicSuppressThreshold = 0.7) — gates suppressedTypes behavior in processTextInput: explicit reclassification always suppresses original type; implicit reclassification only suppresses non-heuristic types if confidence ≥ 0.7. Low-confidence hints (< 0.7) don't suppress, allowing LLM to discover missed entities.

### ⚠ Clause Splitting & Role Detection (Phase 4)
Changing SplitClausesWithRoles(), expandCommaList(), or DetectSubordinateClause() affects:
- Per-clause extraction granularity (main vs subordinate vs expanded items)
- Entity creation count per input (single task vs distributed across siblings)
- Sibling index tracking in gapTargets (for multi-entity gap detection)
- Reingest preserve logic: reingestMode=true suppresses unconfirmed flip on clause updates

### ⚠ GapCandidate & Prompt Guidance (Phase 4)
Changing GapCandidate struct or BuildGapAnalysisPrompt affects:
- `business/domain/ingestbus/extractor/prompt.go:386-396` — Options guidance section (enumerable vs open-ended decision logic)
- `business/domain/ingestbus/extractor/claudecli.go:92-113` — gapAnalysisSchema JSON schema with options (array of string) and options_confidence (number) fields
- All Extractor implementations (Claude Code sidecar, mock, ollama) — must produce Options and OptionsConfidence in JSON response
- Clarification generation downstream (gap cards may reference options for user selection)
- Frontend rendering of gap cards — must handle optional options array and confidence score

### ⚠ BuildGapAnalysisPrompt Meta-Question Filtering (Phase 4)
Changing the "Explicitly forbidden" list affects:
- `business/domain/ingestbus/extractor/prompt_test.go:213–255` — TestBuildGapAnalysisPrompt_RejectsMeta_* tests
- Prompt quality and relevance (must avoid meta-questions, duplicates, hygiene observations)
- AI consistency across extraction runs

### ⚠ Reingest Workflow & Source Hash (reingestapp handlers)
Changing reingestTask/Note/Event or resetRawInput affects:
- Stale clarification dismissal logic (raw_input → entity)
- Raw input synthesis for entities without RawInputID
- skip_classify determination (ContextID != nil → skip extraction)
- reingest_mode flag behavior (suppresses unconfirmed flip, preserves user confirmations)
- Clarification source_hash propagation (via ComputeSourceHash in ingestbus.go:200–219) — must remain stable across reingest cycles for clarification dedup

### ⚠ ReclassifiedAs & SuggestedNewContextKind Fields (Phase 4)
Changing TextExtraction.ReclassifiedAs and TextExtraction.SuggestedNewContextKind affects:
- `ingestbus.go:907-908` — context.Kind auto-selection when suggest_new_context=true and SuggestedNewContextKind is non-empty
- `ingestbus.go:981` — auto-created context receives Kind from extraction suggestion, not fixed default
- Extractor prompt guidance (lines in extractor/prompt.go) that instructs AI to emit reclassified_as and suggested_new_context_kind
- All Extractor implementations — must populate these fields when AI output conflicts with heuristic or suggests context kind
- Downstream entity creation — tasks/events/notes are created with suggested context kind if extracted

### ⚠ Extractor TypeHint & Confidence Parameters (Phase 4)
Changing ExtractText(typeHint, typeHintConfidence) affects:
- `extractor/claudecli.go`, `extractor/ollama.go`, `extractor/mock.go` — all implementations must accept and thread these parameters to prompt building
- `extractor/router.go`, `extractor/failover.go` — routing logic must pass typeHint/typeHintConfidence through to underlying implementations
- Prompt guidance: typeHint is the heuristic-classified type (task|event|note), typeHintConfidence is the heuristic score (0–1)
- Reclassification lock strength: low confidence (<0.5) softens reclassification locks in prompts; high confidence (>0.8) prevents reclassification

### ⚠ Failure Type Structure (eval/ Phase 4)
Changing Failure struct or FailureKind enum affects:
- `eval/eval.go:assert()` — must populate Kind and Message for each failure
- `eval/metrics.go` — scoring logic that may enumerate FailureKind values
- Fixture test results reporting (structured failures enable categorized failure analysis)
- Any tooling that consumes FixtureResult.Failures (must switch from string parsing to Kind enum)

### ⚠ Classification Evaluation (eval/ package)
Adding/modifying fixture assertions affects:
- `eval/metrics.go` — scoring logic for pass/fail decisions (must enumerate all FixtureExpected fields)
- `make eval-classification` target — coverage of extraction quality across test cases
- CI/CD classification regression detection — if evaluation harness is wired to pre-commit hooks or CI

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
14. **Knowledge gap detection** — Synchronous: Detect() per created entity, merge result into raw_input.result before marking processed (was previously a goroutine; ran async raced with MarkProcessed)
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
   - ExtractText(typeHint=string(cl.Type), typeHintConfidence=cl.Confidence, candidates, contextAnnotations) → TextExtraction (action items, events, notes, entity resolutions, ReclassifiedAs, SuggestedNewContextKind)
   - Reclassification lock: softened if typeHintConfidence < 0.5, locked if > 0.8
   - Collects clauseWithIdx for role-aware processing
6. **Merge clause results** — Aggregate action items, keywords, best context, entity resolutions across all clauses
7. **Context matching** — Similar to email pipeline (best context ID + confidence, keyword fallback, auto-create)
8. **Low-confidence type clarification** (Phase 4) — Upsert TypeAssignment clarification if:
   - Clause confidence < 0.75 AND
   - reingestMode=false (skipped during reingest to preserve confirmed state)
9. **Entity resolution** — Apply "update" or "ambiguous" decisions per clause
10. **Create tasks, events, notes** (Phase 4) — Per clause, create entities with:
    - matched context (including Kind from extraction if SuggestedNewContextKind is non-empty)
    - description fallback: if item.Description empty and matchedContextID exists, use context.Description
    - unconfirmed flag: true if clause confidence < 0.75 (unless reingestMode=true)
    - auto-create context: if suggest_new_context=true and matchedContextID is nil, create context with Kind from SuggestedNewContextKind
    - collect gapTargets: (entityType, entityID, content) tuples
11. **Embed created entities** — Store embeddings for tasks, events, notes
12. **Generate clarifications** — Ambiguous actions, deadlines, entity matches (Phase 4), voice references
13. **Knowledge gap detection** — Synchronous: Detect() per created entity; gapResult is assigned to pr.GapAnalysis before the final pipeline-result write (was previously async goroutine; race could clobber pipeline result)
14. **Mark raw_input** — Mark processed or partial

### Reingest Path (skip_classify=true) (Phase 4)
Triggered via reingestapp handlers when entity already has ContextID (skip_classify=true):
1. **Dismiss stale clarifications** — Find all pending/snoozed clarifications tied to raw_input or entity, dismiss them
2. **Synthesize raw_input** (if needed) — If entity has no RawInputID, create one from entity content (task title+desc, event title+desc, note content)
3. **Set reingest_mode=true** — Either via ResetForReingest() or ResetForReprocess() + explicit update
4. **Call processSkipClassify()** — Load entity by kind, delete old embeddings, regenerate embeddings, run gap detection synchronously
5. **Mark processed** — Don't extract/classify; preserve confirmed entity state; gap detection completes before MarkProcessed

### Synchronous Gap Detection
- Runs inline on the request context (was previously a goroutine on context.Background(); the async path raced with MarkProcessed and the final pipeline-result write, dropping/overwriting gap data on raw_input.result — see planner-8co3)
- Implemented via `runGapDetection(ctx, targets) *StepResult` helper
- Per-entity: Detect(ctx, entityType, entityID, content)
- Collects results: totalCardsCreated, totalSkipped, errors
- Email path: writes a fresh PipelineResult update before MarkProcessed
- Text/voice path: assigns to `pr.GapAnalysis` directly; the existing pipeline-result write picks it up (no merge-back read needed)
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

The prompt (lines 386–396) guides the AI to populate options and options_confidence; the gapAnalysisSchema in claudecli.go (lines 92–113) defines the JSON structure for option fields; downstream clarifications may render as multiple-choice when options are present.
