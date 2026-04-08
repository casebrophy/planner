# Debrief Backend System

The Debrief domain is a pure orchestrator — no Storer interface, no HTTP routes, no database table. It generates clarification cards for post-completion review: a `task_debrief` card on every task completion (with an importance/impact rating question), a 3-card `context_debrief` sequence (outcome, challenge, lesson) when a context is closed, and a `weekly_review` card every Sunday at 18:00 listing that week's completed tasks for impact ranking. Cards are created via `clarificationbus.Business`. All methods are idempotent: they query for existing pending and snoozed cards before creating new ones. The package is modeled after `ingestbus` — no DB ownership, no Storer, triggered externally.

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
  - **GenerateWeeklyReview(ctx, weekID string, tasks []CompletedTask) error** — idempotency check (Count pending + snoozed weekly_review where SubjectType="week" and SubjectID=deterministic UUID from weekID); if none exist, creates one `weekly_review` clarification card (priority 0.8) with question "You completed N tasks this week. Which had the most impact?" and AnswerOptions JSON `{"tasks": [{id, title}, ...]}`. SubjectID is `uuid.NewSHA1(uuid.NameSpaceURL, "planner:weekly-review:"+weekID)` for deterministic dedup. Returns nil immediately if `tasks` is empty.

No filter.go, order.go, or stores/ subdirectory exists — this package owns no data.

## Trigger Points

### MCP Layer

Debrief methods are fired as best-effort goroutines from `app/domain/mcpapp/mcpapp.go`. Errors are logged as warnings (`a.log.Warn`).

| MCP Tool | Trigger Condition | Method Called |
|----------|-------------------|---------------|
| `toolCompleteTask` | `updated.CompletedAt != nil` after status set to Done | `debriefBus.OnTaskCompleted()` |
| `toolUpdateTask` | `updated.CompletedAt != nil` after any task update | `debriefBus.OnTaskCompleted()` |
| `toolUpdateContext` | `input.Status == "closed"` | `debriefBus.OnContextClosed()` |

All three callers use `go func() { ... }()` with a fresh `context.Background()`.

### Background Scheduler (main.go)

`GenerateWeeklyReview` is called from a ticker goroutine in `api/services/planner/main.go`. It fires once per week on Sunday at 18:00 local time.

```
Ticker interval: 1 minute
Guard: lastGenWeek string — skips if weekID already generated this week
Fire condition: now.Weekday() == time.Sunday && now.Format("15:04") == "18:00"
weekID format: "2026-W15" (ISOWeek year + zero-padded week number)
Task query: taskbus.QueryFilter{Status: &taskstatus.Done}, page.New(1, 200)
Filter: tasks where CompletedAt != nil && CompletedAt.After(now - 7 days)
```

The scheduler instantiates `debriefBus` directly in `main.go` (not via mcpapp wiring) using the shared `clarBus` and a dedicated `threadBus` instance:
```go
threadStore := threaddb.NewStore(log, db)
threadBus  := threadbus.NewBusiness(log, threadStore)
debriefBus := debriefbus.NewBusiness(log, clarBus, threadBus)
```

## Wiring

### MCP routes (app/domain/mcpapp/route.go)

```go
dbBus := debriefbus.NewBusiness(cfg.Log, clBus, thBus)
// injected into app struct as debriefBus *debriefbus.Business
```

### main.go (api/services/planner/main.go)

A second `debriefBus` instance is constructed at startup for the weekly review scheduler:

```go
threadStore := threaddb.NewStore(log, db)
threadBus  := threadbus.NewBusiness(log, threadStore)
debriefBus := debriefbus.NewBusiness(log, clarBus, threadBus)
```

`clarBus` is the shared top-level instance. `threadBus` here is a dedicated instance for the scheduler goroutine.

Note: `thBus`/`threadBus` is injected into `Business` struct but not actively called in the current implementation (retained for future use).

## Impact Callouts

### CompletedTask struct (business/domain/debriefbus/model.go)
All fields are consumed directly in `OnTaskCompleted()`. Adding a field requires updating all three MCP call sites in `mcpapp.go` that construct `debriefbus.CompletedTask`.

### WeeklyReviewTask struct (business/domain/debriefbus/model.go)
Used by `GenerateWeeklyReview()` to build the `AnswerOptions` JSON payload for the weekly review clarification card. The scheduler in `main.go` constructs these from `taskbus.Task` values filtered to the past 7 days. Adding a field here requires updating the JSON marshal call in `GenerateWeeklyReview()` and any frontend components that parse the `answer_options` JSON for `weekly_review` kind cards.

### Trigger logic in OnTaskCompleted
`OnTaskCompleted` fires on **every** task completion. The question text adapts: if `DurationMin` is set and actual elapsed time > 2× estimate, the question notes the overrun. If no estimate or no overrun, the default importance question is used. There is no longer a conditional gate — every completion produces a debrief card (unless one already exists).

### Idempotency scope for OnTaskCompleted
Checks both `pending` and `snoozed` status. This prevents duplicate cards if the task is re-completed after a snoozed card was created.

### Idempotency scope for OnContextClosed
Checks both `pending` and `snoozed` because context_debrief cards are always created pre-snoozed. Without both checks, re-closing a context would create duplicate cards.

### GenerateWeeklyReview dedup key
The SubjectID for weekly review cards is a deterministic UUID: `uuid.NewSHA1(uuid.NameSpaceURL, []byte("planner:weekly-review:"+weekID))`. This ensures the same week always maps to the same UUID, making the idempotency check reliable across restarts. Changing the seed string would break dedup for any week that hasn't yet been reviewed.

### Weekly review scheduler (api/services/planner/main.go)
The scheduler guards via `lastGenWeek string` in goroutine local state — it resets to empty on server restart, so if the server restarts after 18:00 on a Sunday before the review was generated, it will re-fire once the minute ticker hits 18:00 again (but the idempotency check inside `GenerateWeeklyReview` prevents duplicates). The scheduler uses `taskstatus.Done` filter + 7-day cutoff — tasks with `CompletedAt == nil` are excluded regardless of status.

### Best-effort fire-and-forget
Errors from `OnTaskCompleted` and `OnContextClosed` are logged as warnings at all three MCP call sites. Errors from `GenerateWeeklyReview` are logged via `log.Error` in the scheduler goroutine. Failures produce no user-visible error; the debrief card simply does not get created.

### clarificationkind values used
`clarificationkind.TaskDebrief`, `clarificationkind.ContextDebrief`, and `clarificationkind.WeeklyReview` must exist in `business/types/clarificationkind/`. `WeeklyReview` was added in the same commit that introduced `WeeklyReviewTask`. Adding a new kind also requires updating the DB CHECK constraint on the `clarifications` table (migration required).

## Cross-Domain Dependencies

- **clarificationbus** (`business/domain/clarificationbus`) — `Create()` and `Count()` are the only methods called. `clarificationbus.QueryFilter` is used for the idempotency checks (Kind, Status, SubjectType, SubjectID fields).
- **threadbus** (`business/domain/threadbus`) — injected into `Business` struct but not actively used in current implementation (retained for future use).
- **clarificationkind** (`business/types/clarificationkind`) — `TaskDebrief`, `ContextDebrief`, and `WeeklyReview` values must be registered.
- **clarificationstatus** (`business/types/clarificationstatus`) — `Pending` and `Snoozed` values used in idempotency filters.
- **MCP app** (`app/domain/mcpapp`) — consumer for `OnTaskCompleted` and `OnContextClosed`; no HTTP routes exist for debriefbus.
- **main.go** (`api/services/planner/main.go`) — consumer for `GenerateWeeklyReview` via background scheduler goroutine; also imports `taskbus`, `taskstatus`, `threadbus`, `threaddb`, and `page` for the scheduler.
- **taskbus** (`business/domain/taskbus`) — the scheduler queries `taskbus.QueryFilter{Status: &taskstatus.Done}` to get completed tasks for the weekly review.

## Routes

None. debriefbus has no HTTP endpoints. It is triggered exclusively from MCP tool handlers.
