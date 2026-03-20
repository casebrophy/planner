# Clarification Backend Architecture

> The clarification domain manages a queue of questions the system cannot resolve autonomously — ambiguous context assignments, stale tasks, unclear deadlines, etc. Items are surfaced as a swipeable review deck. Users resolve, snooze, or dismiss items; resolution triggers side-effects (not yet implemented). Priority scoring weights item age and kind importance. An `UnsnoozeExpired` store method supports background re-queuing of snoozed items.
>
> **Note:** Routes are registered in `clarificationapp/route.go` but the domain **may not yet be wired into `main.go`** — verify before assuming endpoints are live.

---

## Core Types

### App Layer — `app/domain/clarificationapp/model.go`

```go
// Response DTO — returned by all read and action handlers.
type ClarificationItem struct {
    ID            string          `json:"id"`
    Kind          string          `json:"kind"`
    Status        string          `json:"status"`
    SubjectType   string          `json:"subjectType"`
    SubjectID     string          `json:"subjectId"`
    Question      string          `json:"question"`
    ClaudeGuess   json.RawMessage `json:"claudeGuess,omitempty"`
    Reasoning     *string         `json:"reasoning,omitempty"`
    AnswerOptions json.RawMessage `json:"answerOptions"`
    Answer        json.RawMessage `json:"answer,omitempty"`
    PriorityScore float32         `json:"priorityScore"`
    SnoozedUntil  *string         `json:"snoozedUntil,omitempty"`
    CreatedAt     string          `json:"createdAt"`
    ResolvedAt    *string         `json:"resolvedAt,omitempty"`
}

// Request body for POST /api/v1/clarifications/{id}/resolve.
type ResolveInput struct {
    Answer json.RawMessage `json:"answer"`
}

// Request body for POST /api/v1/clarifications/{id}/snooze.
type SnoozeInput struct {
    Hours int `json:"hours"` // defaults to 24 if <= 0
}

// Response for GET /api/v1/clarifications/count.
type CountResponse struct {
    Count int `json:"count"`
}
```

### Business Layer — `business/domain/clarificationbus/model.go`

```go
type ClarificationItem struct {
    ID            uuid.UUID
    Kind          clarificationkind.Kind
    Status        clarificationstatus.Status
    SubjectType   string
    SubjectID     uuid.UUID
    Question      string
    ClaudeGuess   *json.RawMessage
    Reasoning     *string
    AnswerOptions json.RawMessage
    Answer        *json.RawMessage
    PriorityScore float32
    SnoozedUntil  *time.Time
    CreatedAt     time.Time
    ResolvedAt    *time.Time
}

type NewClarificationItem struct {
    Kind          clarificationkind.Kind
    SubjectType   string
    SubjectID     uuid.UUID
    Question      string
    ClaudeGuess   *json.RawMessage
    Reasoning     *string
    AnswerOptions json.RawMessage
    PriorityScore float32
    SnoozedUntil  *time.Time
}

type ResolveClarificationItem struct {
    Answer json.RawMessage
}
```

### Business Layer — `business/domain/clarificationbus/filter.go`

```go
type QueryFilter struct {
    Status      *clarificationstatus.Status
    Kind        *clarificationkind.Kind
    SubjectType *string
    SubjectID   *uuid.UUID
}
```

### Business Layer — `business/domain/clarificationbus/clarificationbus.go`

```go
type Storer interface {
    Create(ctx context.Context, item ClarificationItem) error
    Update(ctx context.Context, item ClarificationItem) error
    Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]ClarificationItem, error)
    Count(ctx context.Context, filter QueryFilter) (int, error)
    QueryByID(ctx context.Context, id uuid.UUID) (ClarificationItem, error)
    UnsnoozeExpired(ctx context.Context, now time.Time) (int, error)
}
```

### Store Layer — `business/domain/clarificationbus/stores/clarificationdb/model.go`

```go
type clarificationDB struct {
    ID            uuid.UUID        `db:"clarification_id"`
    Kind          string           `db:"kind"`
    Status        string           `db:"status"`
    SubjectType   string           `db:"subject_type"`
    SubjectID     uuid.UUID        `db:"subject_id"`
    Question      string           `db:"question"`
    ClaudeGuess   *json.RawMessage `db:"claude_guess"`
    Reasoning     *string          `db:"reasoning"`
    AnswerOptions json.RawMessage  `db:"answer_options"`
    Answer        *json.RawMessage `db:"answer"`
    PriorityScore float32          `db:"priority_score"`
    SnoozedUntil  *time.Time       `db:"snoozed_until"`
    CreatedAt     time.Time        `db:"created_at"`
    ResolvedAt    *time.Time       `db:"resolved_at"`
}
```

### Enum Types

`business/types/clarificationkind/` — values: `context_assignment`, `stale_task`, `ambiguous_deadline`, `new_context`, `overlapping_contexts`, `ambiguous_action`, `voice_reference`, `inactivity_prompt`, `context_debrief`

Kind weights (used in priority scoring):
| Kind | Weight |
|------|--------|
| `new_context` | 0.9 |
| `ambiguous_action` | 0.8 |
| `context_debrief` | 0.8 |
| `context_assignment` | 0.7 |
| `voice_reference` | 0.7 |
| `stale_task` | 0.6 |
| `overlapping_contexts` | 0.6 |
| `inactivity_prompt` | 0.6 |
| `ambiguous_deadline` | 0.5 |

`business/types/clarificationstatus/` — values: `pending`, `snoozed`, `resolved`, `dismissed`

All enums expose `Parse(s string) (T, error)`, `MustParse(s string) T`, `String() string`, and text marshaling.

### Database Schema — `business/sdk/migrate/sql/migrate.sql` (version 1.07)

```sql
CREATE TABLE clarification_items (
    clarification_id UUID        NOT NULL DEFAULT gen_random_uuid(),
    kind             TEXT        NOT NULL CHECK (kind IN (
        'context_assignment', 'stale_task', 'ambiguous_deadline',
        'new_context', 'overlapping_contexts', 'ambiguous_action',
        'voice_reference', 'inactivity_prompt', 'context_debrief'
    )),
    status           TEXT        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'snoozed', 'resolved', 'dismissed')),
    subject_type     TEXT        NOT NULL CHECK (subject_type IN ('task', 'context', 'email', 'raw_input')),
    subject_id       UUID        NOT NULL,
    question         TEXT        NOT NULL,
    claude_guess     JSONB,
    reasoning        TEXT,
    answer_options   JSONB       NOT NULL,
    answer           JSONB,
    priority_score   REAL        NOT NULL DEFAULT 0.0,
    snoozed_until    TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at      TIMESTAMPTZ,
    PRIMARY KEY (clarification_id)
);
CREATE INDEX idx_clarification_pending ON clarification_items(status, priority_score DESC) WHERE status = 'pending';
CREATE INDEX idx_clarification_snoozed ON clarification_items(snoozed_until) WHERE status = 'snoozed';
CREATE INDEX idx_clarification_subject ON clarification_items(subject_type, subject_id);
```

---

## File Map

### App Layer (`app/domain/clarificationapp/`)

- `clarificationapp.go` — **queryQueue()**, **queryByID()**, **resolve()**, **snooze()**, **dismiss()**, **countPending()** — HTTP handlers; queryQueue defaults status filter to `pending` if not specified; resolve validates non-empty answer; snooze defaults to 24h
- `model.go` — **toAppClarification()**, **toAppClarifications()** — type converters; `ClarificationItem.Encode()`, `CountResponse.Encode()` implement `web.Encoder`
- `filter.go` — **parseFilter()** — maps query params (`status`, `kind`, `subject_type`, `subject_id`) to `clarificationbus.QueryFilter`
- `order.go` — **parseOrder()** — maps `orderBy` query param to `order.By` via `orderByFields` map; falls back to `clarificationbus.DefaultOrderBy` (`priority_score DESC`)
- `route.go` — **Routes.Add()** — instantiates `clarificationdb.NewStore` and `clarificationbus.NewBusiness`; registers six endpoints with `mid.Auth` middleware

### Business Layer (`business/domain/clarificationbus/`)

- `clarificationbus.go` — **NewBusiness()**, **Create()**, **Resolve()**, **Snooze()**, **Dismiss()**, **Query()**, **Count()**, **QueryByID()**, **UnsnoozeExpired()**, **RecalculatePriority()** — `Create` computes priority as `age_hours * 0.4 + kind_weight * 0.6`; `Resolve`/`Snooze`/`Dismiss` mutate status and call `storer.Update()`; defines `Storer` interface
- `model.go` — `ClarificationItem`, `NewClarificationItem`, `ResolveClarificationItem` — domain structs
- `filter.go` — `QueryFilter` — shared filter struct
- `order.go` — order field constants and `DefaultOrderBy` (`priority_score DESC`)

### Store Layer (`business/domain/clarificationbus/stores/clarificationdb/`)

- `clarificationdb.go` — **NewStore()**, **Create()**, **Update()**, **Query()**, **Count()**, **QueryByID()**, **UnsnoozeExpired()** — SQL implementations; `UnsnoozeExpired` bulk-updates snoozed items past their `snoozed_until` timestamp back to `pending`
- `model.go` — `clarificationDB` (unexported), **toDBClarification()**, **toBusClarification()**, **toBusClarifications()** — sqlx-tagged struct; enums serialized to strings
- `filter.go` — **applyFilter()** — appends `AND` clauses for status, kind, subject_type, subject_id
- `order.go` — `orderByFields` map (business constant → SQL column name); **orderByClause()**

---

## Impact Callouts

### ⚠ ClarificationItem (`business/domain/clarificationbus/model.go`)

Adding, removing, or renaming a field affects:

- `clarificationbus/clarificationbus.go` — `Create()` builds item from `NewClarificationItem`; `Resolve()`/`Snooze()`/`Dismiss()` mutate item fields before `Update()`
- `clarificationdb/model.go` — `toDBClarification()` and `toBusClarification()` converters must be kept in sync
- `clarificationdb/clarificationdb.go` — SQL INSERT column list in `Create()`; UPDATE SET clause in `Update()`; SELECT column list in `Query()` and `QueryByID()`
- `clarificationapp/model.go` — `toAppClarification()` maps `clarificationbus.ClarificationItem` → `app.ClarificationItem`; add field to DTO and converter

### ⚠ Storer interface (`business/domain/clarificationbus/clarificationbus.go`)

Adding or changing a method signature affects:

- `clarificationdb/clarificationdb.go` — `*Store` must implement the new/changed method
- Any future test doubles or mock implementations

### ⚠ QueryFilter (`business/domain/clarificationbus/filter.go`)

Adding a filter field affects:

- `clarificationdb/filter.go` — `applyFilter()` must add `AND` clause and data map key
- `clarificationapp/filter.go` — `parseFilter()` must parse the new query param

### ⚠ Order constants (`business/domain/clarificationbus/order.go`)

Adding a new `OrderBy*` constant affects:

- `clarificationdb/order.go` — `orderByFields` map must add mapping
- `clarificationapp/order.go` — `orderByFields` map must add mapping

### ⚠ Enum values (`business/types/clarificationkind`, `clarificationstatus`)

Adding a new value affects:

- `business/sdk/migrate/sql/migrate.sql` — `CHECK` constraint must include the new value
- `clarificationkind` — if adding a kind, also add its weight to `KindWeights` map
- Converters using `MustParse`/`Parse` will panic or error on unknown values until updated

### ⚠ Resolution side-effects (TODO — `clarificationapp.go:~108`)

Resolution dispatcher is not yet implemented. When added, `resolve()` handler will map `kind + answer → side-effect`:
- `context_assignment` → update entity's `context_id`
- `ambiguous_action` → create task or mark as no-task
- `new_context` → edit context / merge
- `inactivity_prompt` → update thread / block / deprioritize
- `ambiguous_deadline` → update task `due_date`
- `context_debrief` → create `outcome_observation`, update context `debrief_status`

---

## Routes

| Method | Path | Handler | Auth |
|--------|------|---------|------|
| GET | `/api/v1/clarifications` | `queryQueue` | X-API-Key |
| GET | `/api/v1/clarifications/count` | `countPending` | X-API-Key |
| GET | `/api/v1/clarifications/{id}` | `queryByID` | X-API-Key |
| POST | `/api/v1/clarifications/{id}/resolve` | `resolve` | X-API-Key |
| POST | `/api/v1/clarifications/{id}/snooze` | `snooze` | X-API-Key |
| POST | `/api/v1/clarifications/{id}/dismiss` | `dismiss` | X-API-Key |

Query params for `GET /api/v1/clarifications`: `page`, `rows`, `orderBy` (priority_score/created_at), `status` (defaults to `pending`), `kind`, `subject_type`, `subject_id`.

`GET /api/v1/clarifications/count` — returns `{"count": N}` for pending items only.

---

## Cross-Domain Dependencies

- **tasks** — `subject_type='task'` references `tasks.task_id` (polymorphic, no FK constraint)
- **contexts** — `subject_type='context'` references `contexts.context_id`
- **emails** — `subject_type='email'` references `emails.email_id`
- **raw_inputs** — `subject_type='raw_input'` references `raw_inputs.raw_input_id`
- **inactivity_checks** — `inactivity_checks.clarification_id` FK references `clarification_items.clarification_id`
- **observationbus** (future) — resolution of `context_debrief` kind will create outcome observations
- **page SDK** (`business/sdk/page`) — `queryQueue` uses `page.Parse` for pagination
- **order SDK** (`business/sdk/order`) — `Query` uses `order.By`; default is `priority_score DESC`
- **sqldb** (`foundation/sqldb`) — store uses `NamedExecContext`, `NamedQuerySlice`, `NamedQueryStruct`; returns `sqldb.ErrDBNotFound` on missing rows
