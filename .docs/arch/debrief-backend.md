# Debrief Backend System

The Debrief domain is a pure orchestrator — no Storer interface, no HTTP routes, no database table. It generates clarification cards for post-completion review: a `task_debrief` card on every task completion (with an importance/impact rating question), and a 3-card `context_debrief` sequence (outcome, challenge, lesson) when a context is closed. Cards are created via `clarificationbus.Business`. Both methods are idempotent: they query for existing pending and snoozed debrief cards before creating new ones. The package is modeled after `ingestbus` — no DB ownership, no Storer, triggered externally.

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

### WeeklyReviewTask
```go
// Lightweight task reference for the weekly review clarification card.
type WeeklyReviewTask struct {
    ID    uuid.UUID `json:"id"`
    Title string    `json:"title"`
}
```

## File Map

### Business Layer (Core Logic)
- **`business/domain/debriefbus/model.go`** — Input structs: CompletedTask, ClosedContext, WeeklyReviewTask. No output types; debrief results are clarification items owned by clarificationbus.
- **`business/domain/debriefbus/debriefbus.go`** — Business struct and methods:
  - **NewBusiness()** — constructor taking logger, `*clarificationbus.Business`, `*threadbus.Business`
  - **OnTaskCompleted()** — idempotency check (Count pending AND snoozed task_debrief for subject); if none exist, creates one `task_debrief` clarification with priority 0.9. Question defaults to "You completed '{title}'. How important was this?"; if `DurationMin` is set and actual duration > 2× estimate, question notes the overrun. Answer options: High impact / Medium impact / Low impact / Not worth it / Skip
  - **OnContextClosed()** — idempotency check (Count pending + snoozed context_debrief for subject); if none exist, creates 3 `context_debrief` clarification cards all with `snoozed_until = now + 24h`:
    - Card 1 (priority 0.8): "How did it go overall?" — options: Went well / Mixed results / Difficult / Skip debrief
    - Card 2 (priority 0.7): "What was the biggest challenge?" — options: Timeline pressure / Unclear requirements / External dependencies / No major challenges
    - Card 3 (priority 0.6): "Any lessons or insights worth remembering?" — options: Add a lesson / Nothing to add

No filter.go, order.go, or stores/ subdirectory exists — this package owns no data.

## Trigger Points (MCP Layer)

Debrief methods are fired as best-effort goroutines from `app/domain/mcpapp/mcpapp.go`. Errors are logged as warnings (`a.log.Warn`).

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

`debriefbus` depends on `clBus` (`*clarificationbus.Business`) and `thBus` (`*threadbus.Business`). Note: `thBus` is still injected (threadbus is kept in the constructor signature for future use) but is not actively called in the current implementation.

## Impact Callouts

### CompletedTask struct (business/domain/debriefbus/model.go)
All fields are consumed directly in `OnTaskCompleted()`. Adding a field requires updating all three MCP call sites in `mcpapp.go` that construct `debriefbus.CompletedTask`.

### WeeklyReviewTask struct (business/domain/debriefbus/model.go)
Currently unused beyond type definition — reserved for a future weekly review clarification card. When used, callers will need to construct this from taskbus.Task values.

### Trigger logic in OnTaskCompleted
`OnTaskCompleted` fires on **every** task completion. The question text adapts: if `DurationMin` is set and actual elapsed time > 2× estimate, the question notes the overrun. If no estimate or no overrun, the default importance question is used. There is no longer a conditional gate — every completion produces a debrief card (unless one already exists).

### Idempotency scope for OnTaskCompleted
Checks both `pending` and `snoozed` status. This prevents duplicate cards if the task is re-completed after a snoozed card was created.

### Idempotency scope for OnContextClosed
Checks both `pending` and `snoozed` because context_debrief cards are always created pre-snoozed. Without both checks, re-closing a context would create duplicate cards.

### Best-effort fire-and-forget
Errors from `OnTaskCompleted` and `OnContextClosed` are logged as warnings at all three call sites. Failures (e.g., DB unavailable) produce no user-visible error; the debrief card simply does not get created.

### clarificationkind values used
`clarificationkind.TaskDebrief`, `clarificationkind.ContextDebrief`, and `clarificationkind.WeeklyReview` must exist in `business/types/clarificationkind/`. `WeeklyReview` was added in the same commit that introduced `WeeklyReviewTask`. Adding a new kind also requires updating the DB CHECK constraint on the `clarifications` table (migration required).

## Cross-Domain Dependencies

- **clarificationbus** (`business/domain/clarificationbus`) — `Create()` and `Count()` are the only methods called. `clarificationbus.QueryFilter` is used for the idempotency checks (Kind, Status, SubjectType, SubjectID fields).
- **threadbus** (`business/domain/threadbus`) — injected into `Business` struct but not actively used in current implementation (retained for future use).
- **clarificationkind** (`business/types/clarificationkind`) — `TaskDebrief`, `ContextDebrief`, and `WeeklyReview` values must be registered.
- **clarificationstatus** (`business/types/clarificationstatus`) — `Pending` and `Snoozed` values used in idempotency filters.
- **MCP app** (`app/domain/mcpapp`) — sole consumer; no HTTP routes exist for debriefbus.

## Routes

None. debriefbus has no HTTP endpoints. It is triggered exclusively from MCP tool handlers.
