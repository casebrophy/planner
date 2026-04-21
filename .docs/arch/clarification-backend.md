# Clarification Domain — Backend Architecture

**Status:** Core multi-kind clarification engine — feedback collection, user correction, and side-effect dispatch.

**Location:**
- App layer: `app/domain/clarificationapp/`
- Business layer: `business/domain/clarificationbus/` + `business/domain/clarificationbus/stores/clarificationdb/`

---

## Core Models

### ClarificationItem (Business)
Represents a single clarification card in the queue — a structured question the user must answer before extraction can proceed.

```go
type ClarificationItem struct {
    ID                 uuid.UUID
    Kind               clarificationkind.Kind
    Status             clarificationstatus.Status  // pending, snoozed, resolved, dismissed
    SubjectType        string                      // task, context, email, raw_input, note, event
    SubjectID          uuid.UUID
    SubjectDescription string
    GapCategory        string
    Question           string
    ClaudeGuess        *json.RawMessage
    Reasoning          *string
    AnswerOptions      json.RawMessage
    Answer             *json.RawMessage
    PriorityScore      float32
    SnoozedUntil       *time.Time
    SuppressUntil      *time.Time
    CreatedAt          time.Time
    ResolvedAt         *time.Time
}
```

**Key fields:**
- **Kind:** Enum — context_assignment, stale_task, ambiguous_deadline, new_context, overlapping_contexts, ambiguous_action, voice_reference, inactivity_prompt, context_debrief, entity_link, task_debrief, weekly_review, type_assignment, event_prep, ambiguous_entity_match, knowledge_gap.
- **Status:** pending → snoozed (auto-unsnoozed when SnoozedUntil expires) → resolved/dismissed.
- **SubjectType:** Polymorphic subject — the entity being clarified (task, context, email, raw_input, note, event).
- **AnswerOptions:** JSON schema defining available choices (structured per Kind via Options types below).
- **Answer:** User's submitted answer (JSON, Kind-specific shape).
- **PriorityScore:** Float32 computed at creation: `age_hours * 0.4 + kind_weight * 0.6`.
- **SuppressUntil:** For dismissed knowledge_gap items, set to 7 days after dismissal to prevent re-surfacing.
- **GapCategory:** Dedup key for knowledge_gap items (missing_contact, missing_location, etc.). Unique constraint: (kind, subject_type, subject_id, gap_category).

### NewClarificationItem
Constructor input struct — validated before DB insert.

```go
type NewClarificationItem struct {
    Kind               clarificationkind.Kind
    SubjectType        string
    SubjectID          uuid.UUID
    SubjectDescription string
    GapCategory        string
    Question           string
    ClaudeGuess        *json.RawMessage
    Reasoning          *string
    AnswerOptions      json.RawMessage  // JSON schema (Kind-specific Options type)
    PriorityScore      float32
    SnoozedUntil       *time.Time
    SuppressUntil      *time.Time
}
```

**Validation:** Enforces required fields (Kind, SubjectType, SubjectID, SubjectDescription, Question, AnswerOptions). Returns `ErrInvalidClarification` on missing fields.

### ResolveClarificationItem
Minimal input struct for resolution.

```go
type ResolveClarificationItem struct {
    Answer json.RawMessage
}
```

### QueryFilter
Filterable fields for queue queries.

```go
type QueryFilter struct {
    Status       *clarificationstatus.Status
    Kind         *clarificationkind.Kind
    SubjectType  *string
    SubjectID    *uuid.UUID
    CreatedSince *time.Time
}
```

### Storer Interface

```go
type Storer interface {
    Create(ctx context.Context, item ClarificationItem) error
    Upsert(ctx context.Context, item ClarificationItem) (ClarificationItem, error)
    Update(ctx context.Context, item ClarificationItem) error
    Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]ClarificationItem, error)
    Count(ctx context.Context, filter QueryFilter) (int, error)
    QueryByID(ctx context.Context, id uuid.UUID) (ClarificationItem, error)
    UnsnoozeExpired(ctx context.Context, now time.Time) (int, error)
}
```

---

## Answer Options (JSON Schemas)

Each Kind has a corresponding Options struct that validates the `AnswerOptions` JSON:

| Kind | Options Struct | Key Fields |
|------|---|---|
| context_assignment | ContextAssignmentOptions | suggested_context (str), confidence (0-1), available_contexts ([]ContextRef) |
| new_context | NewContextOptions | context_id (UUID str), title |
| ambiguous_action | AmbiguousActionOptions | interpretations ([]str) |
| ambiguous_deadline | AmbiguousDeadlineOptions | description, raw_date |
| entity_link | EntityLinkOptions | source_type, source_id (UUID), target_type, target_id (UUID), confidence |
| type_assignment | TypeAssignmentOptions | clause_text, predicted_type, confidence, options ([]str — "task", "note", "event") |
| voice_reference | VoiceReferenceOptions | original_text, reference_type, clause_text |
| event_prep | EventPrepOptions | event_id (UUID), event_title, event_starts_at (RFC3339), prep_task_ids ([]UUID), prep_task_titles ([]str) |
| ambiguous_entity_match | AmbiguousEntityMatchOptions | candidate_id (UUID), candidate_type, candidate_title, similarity (0-1), choices ([]str) |
| knowledge_gap | KnowledgeGapOptions | gap_category (str), related_entity_type, related_entity_id (UUID), suggested_question, existing_knowledge_summary, confidence (0-1), options ([]str — user selectable choices) |

---

## Business Methods

### Create(ctx, NewClarificationItem) → ClarificationItem
Creates a new clarification card. Validates input, computes priority score, sets initial status (Pending or Snoozed if SnoozedUntil is set), then stores.

### Upsert(ctx, NewClarificationItem) → ClarificationItem
Idempotent create — used by ingestbus when re-processing extractions. Deduplicates by (kind, subject_type, subject_id, gap_category).

### Resolve(ctx, item, ResolveClarificationItem) → ClarificationItem
Sets Status → Resolved, stores the Answer, sets ResolvedAt ← now. Caller must invoke `dispatchResolution()` for side-effects.

### Snooze(ctx, item, until time.Time) → ClarificationItem
Sets Status → Snoozed, SnoozedUntil ← until.

### Dismiss(ctx, item) → ClarificationItem
Sets Status → Dismissed, ResolvedAt ← now, SuppressUntil ← now + 7 days. Used for knowledge_gap items to prevent re-surfacing.

### Query, Count, QueryByID
Standard CRUD read operations with filtering and pagination.

### UnsnoozeExpired(ctx) → count int
Background task — finds all Snoozed items where SnoozedUntil ≤ now, reverts Status → Pending.

### RecalculatePriority(item) → float32
Stateless helper — recomputes priority score: `age_hours * 0.4 + kind_weight * 0.6`.

---

## HTTP Endpoints (App Layer)

All endpoints require `X-API-Key` header (auth middleware).

### GET /api/v1/clarifications
Query the clarification queue with optional filtering and pagination.

**Query params:**
- `status` (optional) — filter by status; defaults to "pending"
- `kind` (optional) — filter by kind
- `subject_type` (optional) — filter by subject_type
- `subject_id` (optional) — filter by subject_id
- `page`, `rows` — pagination
- `order_by`, `order_dir` — sorting

**Response:** `{items: ClarificationItem[], total: int, page: int, rows: int}`

### GET /api/v1/clarifications/count
Count pending clarifications.

**Response:** `{count: int}`

### GET /api/v1/clarifications/{id}
Fetch a single clarification by ID.

**Response:** `ClarificationItem`

### POST /api/v1/clarifications/{id}/resolve
Resolve a clarification with user's answer.

**Body:** `{answer: json.RawMessage}`

**Response:** `ClarificationItem`

**Side-effect:** Dispatches `dispatchResolution()` based on Kind and Answer content (see Resolution Dispatch section below).

### POST /api/v1/clarifications/{id}/snooze
Snooze a clarification.

**Body:** `{hours: int}` (default 24)

**Response:** `ClarificationItem`

### POST /api/v1/clarifications/{id}/dismiss
Dismiss a clarification (sets suppress_until).

**Response:** `ClarificationItem`

---

## Resolution Dispatch (dispatchResolution)

Invoked after `.resolve()` succeeds. Maps Kind + Answer → side-effects. Errors logged but non-fatal (user sees resolution confirmed).

### Free-Text Override (Special)
If Answer contains `{free_text: string}`, the entire clarification is cleared and the raw_input is reset for reprocessing with the user's correction. Delete order: notes → tasks → events (notes may reference tasks).

### context_assignment
Answer: `{context_id: UUID}`
- Fetches subject entity (task, note, event, email) by SubjectID.
- Updates entity's context_id.

### ambiguous_deadline
Answer: `{due_date: "2006-01-02" | RFC3339}`
- Parses date; currently no side-effect implemented (stub).

### ambiguous_action
Answer: `{is_task: bool, title: str, description: str, context_id: UUID}`
- If is_task, creates a new Task with the provided title, description, context_id.

### new_context
Answer: `{action: "confirm"|"merge", title?: str, description?: str, merge_target_id?: UUID}`
- If "confirm": updates the context's title/description if provided.
- If "merge": deletes the context (manual merge handling).

### inactivity_prompt
Answer: `{action: "completed"|other, note?: str}`
- Adds a ThreadEntry to the subject.
- If action="completed" and subject is task: sets task Status → Done.
- If action="completed" and subject is context: sets context Status → Closed.

### context_debrief
Answer: `{response: str}`
- Records an Observation (Kind=Debrief, Data={response, question}).
- Checks if all debrief clarifications for this context are resolved.
- If all resolved: sets context DebriefStatus → Done.

### stale_task
Answer: `{status: str}`
- Parses status (e.g., "done"), updates task Status.

### entity_link
Answer: `{confirmed: bool}`
- If confirmed: parses AnswerOptions (EntityLinkOptions), creates EntityLink between source and target.

### task_debrief
Answer: `{value: str}`
- Records an Observation (Kind=Debrief, Data={importance, question}) if value != "skip".

### weekly_review
Answer: `{selected_task_ids: []UUID}`
- For each selected task, records an Observation (Kind=Debrief, Data={importance: "high", source: "weekly_review"}).

### type_assignment
Answer: `{actual_type: "task"|"note"|"event"}`
- Logs a ClassificationCorrection (predicted vs actual type).
- Clears Unconfirmed flag on subject entity.

### event_prep
Answer: (none used)
- No side-effect — user acknowledges but manually schedules.

### ambiguous_entity_match
Answer: `{choice: "use_existing"|"create_new"}`
- If "use_existing": deletes the unconfirmed duplicate extracted from raw_input.

### knowledge_gap
Answer: `{answer_text?: str, selected_option?: str, dismissed?: bool}`
- **Precedence:** dismissed > selected_option > answer_text.
- If **dismissed=true**: calls `Dismiss()` (sets suppress_until ← now + 7 days).
- If **selected_option or answer_text**: creates a Note with the answer content (selected_option takes precedence if both present), links it to subject via EntityLink (kind="knowledge_gap_answer").
- If both selected_option and answer_text are empty: returns without side-effect.

---

## Database Schema

**Table:** `clarification_items`

| Column | Type | Nullable | Constraints |
|--------|------|----------|-------------|
| clarification_id | UUID | NO | PK, DEFAULT gen_random_uuid() |
| kind | TEXT | NO | CHECK: one of 17 Kind values |
| status | TEXT | NO | DEFAULT 'pending', CHECK: pending/snoozed/resolved/dismissed |
| subject_type | TEXT | NO | CHECK: task/context/email/raw_input/note/event |
| subject_id | UUID | NO | |
| subject_description | TEXT | NO | DEFAULT '' |
| gap_category | TEXT | NO | DEFAULT '' |
| question | TEXT | NO | |
| claude_guess | JSONB | YES | |
| reasoning | TEXT | YES | |
| answer_options | JSONB | NO | |
| answer | JSONB | YES | |
| priority_score | REAL | NO | DEFAULT 0.0 |
| snoozed_until | TIMESTAMPTZ | YES | |
| suppress_until | TIMESTAMPTZ | YES | (for dismissed gap suppression) |
| created_at | TIMESTAMPTZ | NO | DEFAULT NOW() |
| resolved_at | TIMESTAMPTZ | YES | |

**Indexes:**
- `idx_clarification_pending(status, priority_score DESC)` WHERE status = 'pending'
- `idx_clarification_snoozed(snoozed_until)` WHERE status = 'snoozed'
- `idx_clarification_subject(subject_type, subject_id)`
- Unique constraint: `(kind, subject_type, subject_id, gap_category)` for dedup

---

## Cross-Domain Dependencies

### Inbound (Who calls clarificationbus)

| Caller | How | Purpose |
|--------|-----|---------|
| **ingestbus** | Create(ctx, NewClarificationItem) | Generate clarification cards during ingestion (type_assignment, context_assignment, etc.) |
| **knowledgegapbus** | Create(ctx, NewClarificationItem) | Generate knowledge_gap clarifications from extracted gaps |
| **debriefbus** | Create(ctx, NewClarificationItem) | Generate context_debrief, task_debrief, weekly_review |
| **inactivitybus** | Create(ctx, NewClarificationItem) | Generate inactivity_prompt clarifications |
| **classifyapp** | Query (read-only) | Display clarification queue in UI |
| **mcpapp** | Query, Resolve (via MCP methods) | MCP interface for external agents |
| **clarificationapp.dispatchResolution** | Create (observation, entity_link), Update (task, note, event, context, email), Delete (task, note, event) | Side-effect execution |

### Outbound (clarificationbus calls)

| Target | Method | Purpose |
|--------|--------|---------|
| **taskbus** | QueryByID, Update, DeleteByRawInputUnconfirmed | Side-effect dispatch (context_assignment, inactivity_prompt, stale_task, type_assignment, knowledge_gap) |
| **notebus** | QueryByID, Update, Create, DeleteByRawInputUnconfirmed | Side-effect dispatch (context_assignment, knowledge_gap) |
| **eventbus** | QueryByID, Update, DeleteByRawInputUnconfirmed | Side-effect dispatch (context_assignment, knowledge_gap) |
| **contextbus** | QueryByID, Update | Side-effect dispatch (new_context, inactivity_prompt, context_debrief) |
| **emailbus** | QueryByID, Update | Side-effect dispatch (context_assignment) |
| **observationbus** | Record | Side-effect dispatch (context_debrief, task_debrief, weekly_review) |
| **rawinputbus** | QueryByID, Update, ResetForReprocess | Free-text override dispatch |
| **threadbus** | AddEntry | Side-effect dispatch (inactivity_prompt) |
| **entitylinkbus** | Create | Side-effect dispatch (entity_link, knowledge_gap) |
| **classificationcorrectionbus** | Record | Side-effect dispatch (type_assignment) |

---

## Impact Callouts

### Adding a new Kind
1. Add to `business/types/clarificationkind/clarificationkind.go` (enum).
2. Update migration SQL: CHECK constraint on `kind` column.
3. Create corresponding Options struct in `business/domain/clarificationbus/options.go`.
4. Add case in `dispatchResolution()` (even if no side-effect, stub with comment).
5. Add tests in `app/domain/clarificationapp/tests/clarificationapi/resolve_*_test.go`.

### Adding a new Answer field (per Kind)
1. Update the Options struct in `options.go`.
2. Update dispatchResolution case to unmarshal the new field.
3. Invoke new side-effect (e.g., call another bus, update a domain).

### Adding a QueryFilter field
1. Add field to `business/domain/clarificationbus/filter.go`.
2. Implement `applyFilter()` in `business/domain/clarificationbus/stores/clarificationdb/filter.go`.
3. Add parsing in `app/domain/clarificationapp/filter.go`.

### Changing the dedup key (gap_category)
- Unique constraint is `(kind, subject_type, subject_id, gap_category)`.
- If a Kind should not dedupe on gap_category, use empty string (gap_category DEFAULT '').
- Knowledge_gap items use non-empty gap_category for dedup (e.g., "missing_contact").

### ⚠ KnowledgeGapOptions ({path: business/domain/clarificationbus/options.go})
The `Options []string` and `Confidence float64` fields enable structured answer choices.

Changing this struct shape affects:
- `app/domain/clarificationapp/clarificationapp.go:696-698` — dispatchResolution unmarshals SelectedOption from Answer and checks precedence (selected_option > answer_text).
- `app/domain/clarificationapp/model.go` — may need JSON binding updates if new fields added.
- Frontend schema validation in `api/services/frontend/web/src/components/clarifications/` — types auto-imported via tygo will update automatically.

**Answer precedence (runtime logic):**
- `{dismissed: true}` → unconditional dismiss, sets suppress_until ← now + 7 days (no note created).
- `{selected_option: "..."} && answer_text == ""` → create Note from selected_option.
- `{selected_option: "" && answer_text: "..."}` → create Note from answer_text.
- Both empty → return without side-effect.

---

## File Map

| File | Role |
|------|------|
| `business/domain/clarificationbus/model.go` | ClarificationItem, NewClarificationItem, ResolveClarificationItem structs + Validate() |
| `business/domain/clarificationbus/clarificationbus.go` | Storer interface, Business type, CRUD + side-effect logic |
| `business/domain/clarificationbus/filter.go` | QueryFilter struct |
| `business/domain/clarificationbus/order.go` | OrderBy constants (priority_score, created_at, resolved_at, snoozed_until) |
| `business/domain/clarificationbus/options.go` | All Options structs (ContextAssignmentOptions, KnowledgeGapOptions, etc.) |
| `business/domain/clarificationbus/stores/clarificationdb/model.go` | DB struct (clarificationDB), toDBClarification, toBusClarification converters |
| `business/domain/clarificationbus/stores/clarificationdb/clarificationdb.go` | Store implementation (Create, Upsert, Update, Query, Count, QueryByID, UnsnoozeExpired) |
| `business/domain/clarificationbus/stores/clarificationdb/filter.go` | applyFilter() — builds WHERE clauses |
| `business/domain/clarificationbus/stores/clarificationdb/order.go` | orderByFields map, orderByClause() — SQL column mapping |
| `app/domain/clarificationapp/model.go` | ClarificationItem (app DTO), ResolveInput, SnoozeInput, toAppClarification converters |
| `app/domain/clarificationapp/clarificationapp.go` | Handlers: queryQueue, queryByID, resolve, snooze, dismiss, countPending; **dispatchResolution()** |
| `app/domain/clarificationapp/filter.go` | parseFilter() — maps query params to QueryFilter |
| `app/domain/clarificationapp/order.go` | parseOrder() — maps request fields to OrderBy |
| `app/domain/clarificationapp/route.go` | Routes.Add() — wires endpoints, instantiates stores + buses |
| `business/sdk/migrate/sql/migrate.sql` | CREATE TABLE clarification_items + ALTERs for subject_description, gap_category, suppress_until |

---

## Testing

Tests use real Postgres via `business/sdk/dbtest`.

**Store tests:** `business/domain/clarificationbus/clarificationbus_test.go`
- Create, Query, Count, QueryByID, UnsnoozeExpired.

**API tests:** `app/domain/clarificationapp/tests/clarificationapi/`
- `clarification_test.go` — Query queue, count, fetch by ID.
- `resolve_test.go` — Resolve with side-effects (ContextAssignment, InactivityPrompt, etc.).
- `resolve_freetext_test.go` — Free-text override dispatch.
- `snooze_dismiss_test.go` — Snooze and dismiss.
- `query_test.go` — Filter and order.
- `seed_test.go` — Test fixtures.

**Key assertions:**
- Create validates input; returns `ErrInvalidClarification` if missing fields.
- Resolve sets Status → Resolved, stores Answer, invokes dispatchResolution.
- dispatchResolution updates cross-domain entities (tasks, contexts, observations).
- Snooze sets Status → Snoozed, SnoozedUntil.
- Dismiss sets Status → Dismissed, SuppressUntil ← now + 7 days.

---

## Version History

| Version | Date | Change |
|---------|------|--------|
| 1.07 | Initial | Create clarification_items table, Storer interface, CRUD methods |
| 1.11 | Later | Add subject_description column (for user-facing context) |
| 1.19 | Later | Add gap_category column + unique constraint for dedup; add suppress_until for dismissed gaps; add CreatedSince to QueryFilter |
| Current | 2026-04 | Document all 16+ Kind values, Options types, dispatchResolution precedence (dismissed > selected_option > answer_text) |
