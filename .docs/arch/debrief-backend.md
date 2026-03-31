# Debrief Backend System

The Debrief domain is a pure orchestrator — no Storer interface, no HTTP routes, no database table. It generates clarification cards for post-completion review: a `task_debrief` card when a completed task had a duration overrun (actual > 2× estimated) or blocker thread entries, and a 3-card `context_debrief` sequence (outcome, challenge, lesson) when a context is closed. Cards are created via `clarificationbus.Business` and the blocker check is performed via `threadbus.Business`. Both methods are idempotent: they query for existing pending/snoozed debrief cards before creating new ones. The package is modeled after `ingestbus` — no DB ownership, no Storer, triggered externally.

## Core Types

### CompletedTask
```go
type CompletedTask struct {
    ID          uuid.UUID
    Title       string
    DurationMin *int   // estimated duration; nil means no estimate
    CreatedAt   int64  // unix timestamp
    CompletedAt int64  // unix timestamp
}
```

### ClosedContext
```go
type ClosedContext struct {
    ID    uuid.UUID
    Title string
}
```

## File Map

### Business Layer (Core Logic)
- **`business/domain/debriefbus/model.go`** — Input structs: CompletedTask, ClosedContext. No output types; debrief results are clarification items owned by clarificationbus.
- **`business/domain/debriefbus/debriefbus.go`** — Business struct and methods:
  - **NewBusiness()** — constructor taking logger, `*clarificationbus.Business`, `*threadbus.Business`
  - **OnTaskCompleted()** — idempotency check (Count pending task_debrief for subject), blocker check via `hasBlockerEntries()`, duration overrun check (actual minutes > 2× estimated); if either trigger fires, creates one `task_debrief` clarification with priority 0.9 and answer options: "Add a lesson", "Nothing notable", "Snooze"
  - **OnContextClosed()** — idempotency check (Count pending + snoozed context_debrief for subject); if none exist, creates 3 `context_debrief` clarification cards all with `snoozed_until = now + 24h`:
    - Card 1 (priority 0.8): "How did it go overall?" — options: Went well / Mixed results / Difficult / Skip debrief
    - Card 2 (priority 0.7): "What was the biggest challenge?" — options: Timeline pressure / Unclear requirements / External dependencies / No major challenges
    - Card 3 (priority 0.6): "Any lessons or insights worth remembering?" — options: Add a lesson / Nothing to add
  - **hasBlockerEntries()** (unexported) — queries `threadbus.Count()` with filter for subject type/ID and `threadentrykind.Blocker`; returns `true` if count > 0

No filter.go, order.go, or stores/ subdirectory exists — this package owns no data.

## Trigger Points (MCP Layer)

Debrief methods are fired as best-effort goroutines from `app/domain/mcpapp/mcpapp.go`. Errors are silently discarded (`_ = err`).

| MCP Tool | Trigger Condition | Method Called |
|----------|-------------------|---------------|
| `toolCompleteTask` | `updated.CompletedAt != nil` after status set to Done | `debriefBus.OnTaskCompleted()` |
| `toolUpdateTask` | `updated.CompletedAt != nil` after any task update | `debriefBus.OnTaskCompleted()` |
| `toolUpdateContext` | `input.Status == "closed"` | `debriefBus.OnContextClosed()` |

All three callers use `go func() { ... }()` with a fresh `context.Background()`.

## Wiring (route.go)

`app/domain/mcpapp/route.go` constructs the debrief business in the MCP routes setup:

```go
dbBus := debriefbus.NewBusiness(cfg.Log, clBus, thBus)
// injected into app struct as debriefBus *debriefbus.Business
```

`debriefbus` depends on `clBus` (`*clarificationbus.Business`) and `thBus` (`*threadbus.Business`), both of which are already constructed for other MCP tools.

## Impact Callouts

### CompletedTask struct (business/domain/debriefbus/model.go)
All fields are consumed directly in `OnTaskCompleted()`. Adding a field requires updating all three MCP call sites in `mcpapp.go` that construct `debriefbus.CompletedTask`.

### Trigger logic in OnTaskCompleted
The 2× duration overrun threshold is hardcoded. `DurationMin` being nil or zero skips the duration check entirely — only the blocker check applies in that case.

### Idempotency scope for OnTaskCompleted
Only checks for `pending` status (not `snoozed`). If a task_debrief card is snoozed and the task is re-completed (e.g., reopened and completed again), a second card will be created. This is intentional: the pending check is sufficient for the common path, and snoozed cards represent a prior debrief that was deferred, not resolved.

### Idempotency scope for OnContextClosed
Checks both `pending` and `snoozed` because context_debrief cards are always created pre-snoozed. Without both checks, re-closing a context would create duplicate cards.

### Best-effort fire-and-forget
Errors from `OnTaskCompleted` and `OnContextClosed` are silently discarded at all three call sites. Failures (e.g., DB unavailable) produce no user-visible error; the debrief card simply does not get created.

### clarificationkind values used
`clarificationkind.TaskDebrief` and `clarificationkind.ContextDebrief` must exist in `business/types/clarificationkind/` and in the DB CHECK constraint on the `clarifications` table. Adding a new kind requires updating the enum and a migration.

## Cross-Domain Dependencies

- **clarificationbus** (`business/domain/clarificationbus`) — `Create()` and `Count()` are the only methods called. `clarificationbus.QueryFilter` is used for the idempotency checks (Kind, Status, SubjectType, SubjectID fields).
- **threadbus** (`business/domain/threadbus`) — `Count()` is called in `hasBlockerEntries()` with a `QueryFilter` on SubjectType, SubjectID, and Kind = `threadentrykind.Blocker`.
- **clarificationkind** (`business/types/clarificationkind`) — `TaskDebrief` and `ContextDebrief` values must be registered.
- **clarificationstatus** (`business/types/clarificationstatus`) — `Pending` and `Snoozed` values used in idempotency filters.
- **threadentrykind** (`business/types/threadentrykind`) — `Blocker` value used in the thread filter.
- **MCP app** (`app/domain/mcpapp`) — sole consumer; no HTTP routes exist for debriefbus.

## Routes

None. debriefbus has no HTTP endpoints. It is triggered exclusively from MCP tool handlers.
