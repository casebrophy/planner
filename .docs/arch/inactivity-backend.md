# Inactivity Backend System

The Inactivity domain is a background detection service — it has no HTTP routes. It scans periodically for stale tasks and contexts and for overlapping context pairs, then generates clarification cards so the user is prompted to review them. It does not own a domain table; it reads from the `tasks` and `contexts` tables via dedicated read-only queries and writes via `clarificationbus`. The job runs as a goroutine in `main.go` every 15 minutes.

## Core Types

### StaleItem (Business Layer)
```go
type StaleItem struct {
    SubjectType   string    // "task" or "context"
    SubjectID     uuid.UUID
    Title         string
    Priority      string    // e.g. "urgent", "high", "medium", "low"
    LastUpdated   time.Time
    ThresholdDays float64
}
```

### OverlapPair (Business Layer)
```go
type OverlapPair struct {
    ContextID1 uuid.UUID
    Title1     string
    ContextID2 uuid.UUID
    Title2     string
    SharedTags int
}
```

### Storer Interface
```go
type Storer interface {
    QueryStaleTasks(ctx context.Context) ([]StaleItem, error)
    QueryStaleContexts(ctx context.Context) ([]StaleItem, error)
    QueryOverlappingContexts(ctx context.Context) ([]OverlapPair, error)
}
```

## File Map

### Business Layer (Core Logic)
- **`business/domain/inactivitybus/model.go`** — StaleItem and OverlapPair types
- **`business/domain/inactivitybus/inactivitybus.go`** — Business struct and methods:
  - **NewBusiness()** — constructor taking logger, Storer, and `*clarificationbus.Business`
  - **CheckAll()** — entry point called by scheduler; queries stale tasks, stale contexts, then delegates to CheckOverlaps(); logs summary counts; errors per-item are logged but do not abort the run
  - **CheckOverlaps()** — queries overlapping context pairs; calls createOverlapPrompt() for each; safe to call independently
  - **createInactivityPrompt()** — idempotency check via `clarificationBus.Count()` for an existing `inactivity_prompt` with `pending` status on the same SubjectType+SubjectID; if none, calls `clarificationBus.Create()` with kind=`inactivity_prompt`, a human-readable question, and a JSON `AnswerOptions` payload containing type/priority/last_updated/threshold_days
  - **createOverlapPrompt()** — idempotency check via `clarificationBus.Count()` for an existing `overlapping_contexts` with `pending` status on ContextID1; if none, calls `clarificationBus.Create()` with kind=`overlapping_contexts`, a question naming both context titles and shared tag count, and a JSON `AnswerOptions` array of three actions: `keep`, `merge`, `dismiss`

### Store Layer (Database)
- **`business/domain/inactivitybus/stores/inactivitydb/inactivitydb.go`** — Store struct and methods (no separate model.go or filter.go; all DB types are defined inline):
  - **NewStore()** — constructor taking logger and `*sqlx.DB`
  - **QueryStaleTasks()** — SELECT from `tasks` WHERE status IN ('todo','in_progress') AND `COALESCE(last_thread_at, updated_at) < NOW() - INTERVAL '1 day' * <threshold>`; threshold is a CASE on priority (urgent=1, high=2, medium=5, low=14, else=7); returns `[]StaleItem` with subject_type hardcoded to `'task'`
  - **QueryStaleContexts()** — SELECT from `contexts` WHERE status = 'active' AND `COALESCE(last_event, last_thread_at, updated_at) < NOW() - INTERVAL '7 days'`; priority hardcoded to `'medium'`, threshold_days to `7`; returns `[]StaleItem` with subject_type hardcoded to `'context'`
  - **QueryOverlappingContexts()** — self-join on `context_tags` (ct1.tag_id = ct2.tag_id AND ct1.context_id < ct2.context_id), inner-joined to `contexts` c1 and c2 both filtered to status='active'; GROUP BY pair, HAVING COUNT(*) >= 2; ORDER BY shared_tags DESC LIMIT 10; returns `[]OverlapPair`
  - **dbStaleItem** — internal DB struct with `db` tags; **toBusStaleItem() / toBusStaleItems()** convert to business types; `dbOverlapPair` is defined as a local struct inside `QueryOverlappingContexts()`

### Wiring (main.go)
- **`api/services/planner/main.go`** — constructs `inactivitydb.NewStore` and `inactivitybus.NewBusiness` (passing `clarBus`); starts a goroutine with a 15-minute ticker that calls `inactBus.CheckAll()`; errors are logged, not fatal

## Stale Thresholds

| Priority | Threshold |
|----------|-----------|
| urgent   | 1 day     |
| high     | 2 days    |
| medium   | 5 days    |
| low      | 14 days   |
| (else)   | 7 days    |

Contexts use a flat 7-day threshold regardless of any priority field (contexts do not have a priority column).

Activity is measured by `COALESCE(last_thread_at, updated_at)` for tasks and `COALESCE(last_event, last_thread_at, updated_at)` for contexts.

## Clarification Cards Produced

| Card kind | Trigger | Subject | AnswerOptions |
|-----------|---------|---------|---------------|
| `inactivity_prompt` | Task or context exceeds stale threshold | SubjectType + SubjectID of stale item | JSON object: type, priority, last_updated, threshold_days |
| `overlapping_contexts` | Two active contexts share 2+ tags | ContextID1 of the pair | JSON array: [{keep}, {merge}, {dismiss}] |

Both are idempotent: if a `pending` card of the same kind already exists for the subject, no duplicate is created.

## Impact Callouts

### Adding a new stale detection query
A new query method must be added to the Storer interface in `inactivitybus.go`, implemented in `inactivitydb.go`, and called from `CheckAll()` or a new exported method. No other layers are affected (no HTTP routes, no app layer).

### Changing priority threshold values
Thresholds are hardcoded as SQL CASE expressions inside `QueryStaleTasks()` in `inactivitydb.go`. They are also reflected in the `threshold_days` column of the result and embedded in the `AnswerOptions` JSON payload written by `createInactivityPrompt()`. Changing a threshold requires updating the SQL CASE in `QueryStaleTasks()` only — no migration needed.

### Changing the inactivity context threshold (7 days)
The value is hardcoded twice in `QueryStaleContexts()`: in the WHERE clause interval and in the `7 AS threshold_days` SELECT expression. Both must be updated together.

### Overlap detection is keyword-based (Phase 6 caveat)
`QueryOverlappingContexts()` uses tag co-occurrence as a proxy for semantic similarity. Two contexts with completely different purposes can share tags by coincidence. True overlap detection requires embeddings (planned for Phase 6). The LIMIT 10 caps noise but does not eliminate false positives.

### Storer interface change
Adding or renaming a Storer method requires:
- `business/domain/inactivitybus/inactivitybus.go` — update interface definition
- `business/domain/inactivitybus/stores/inactivitydb/inactivitydb.go` — implement the new/changed method

### clarificationbus dependency
`createInactivityPrompt()` and `createOverlapPrompt()` call `clarificationBus.Count()` and `clarificationBus.Create()`. Changes to `clarificationbus.QueryFilter` fields used here (Kind, Status, SubjectType, SubjectID) or to `NewClarificationItem` fields will require updates in `inactivitybus.go`. The `clarificationkind` values used (`InactivityPrompt`, `OverlappingContexts`) must exist in `business/types/clarificationkind/`.

## Cross-Domain Dependencies

- **clarificationbus** (`business/domain/clarificationbus`) — direct dependency; inactivitybus calls `Count()` for idempotency and `Create()` to emit cards; the clarificationbus.Business pointer is injected at construction
- **clarificationkind** (`business/types/clarificationkind`) — `InactivityPrompt` and `OverlappingContexts` kind values must be defined
- **clarificationstatus** (`business/types/clarificationstatus`) — `Pending` status value used in idempotency filter
- **Task domain** — `QueryStaleTasks()` reads from the `tasks` table directly; no Go import of taskbus; schema dependency on columns `task_id`, `title`, `priority`, `status`, `last_thread_at`, `updated_at`
- **Context domain** — `QueryStaleContexts()` and `QueryOverlappingContexts()` read from `contexts` and `context_tags` directly; schema dependency on columns `context_id`, `title`, `status`, `last_event`, `last_thread_at`, `updated_at` and join table `context_tags(context_id, tag_id)`
- **sqldb utilities** (`business/sdk/sqldb`) — Store uses `NamedQuerySlice` helper
- **logger** (`foundation/logger`) — structured logging in both business and store layers

## Routes

None. This is a background job only. There are no HTTP endpoints for inactivity detection. Output is consumed via the clarification queue (GET /api/v1/clarifications).
