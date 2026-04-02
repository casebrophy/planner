# Daily Plan Backend System

> The daily plan domain organizes tasks into AI-generated daily schedules grouped by context or energy level. It supports plan generation (via Claude AI), item management with user overrides, and status tracking (proposed, completed, dismissed). Plans are date-keyed with versioning (generation numbers) to track regenerations. All routes are protected by API-key auth and integrate with task, event, and context domains to build rich planning context for the Claude generator.

---

## Core Types

### Business Layer — `business/domain/dailyplanbus/model.go`

```go
type DailyPlan struct {
	ID         uuid.UUID
	PlanDate   time.Time
	Generation int
	ModelUsed  string
	PromptHash *string
	CreatedAt  time.Time
}

type NewDailyPlan struct {
	PlanDate   time.Time
	Generation int
	ModelUsed  string
	PromptHash *string
}

type DailyPlanItem struct {
	ID               uuid.UUID
	PlanID           uuid.UUID
	TaskID           uuid.UUID
	Position         int
	GroupName        string
	GroupPosition    int
	AIDurationMin    *int
	AIPriorityReason *string
	UserPosition     *int
	UserDurationMin  *int
	Status           string
	DismissReason    *string
	DismissNote      *string
	CompletedAt      *time.Time
	CreatedAt        time.Time
}

type NewDailyPlanItem struct {
	PlanID           uuid.UUID
	TaskID           uuid.UUID
	Position         int
	GroupName        string
	GroupPosition    int
	AIDurationMin    *int
	AIPriorityReason *string
}

type UpdatePlanItem struct {
	UserPosition    *int
	UserDurationMin *int
	Status          *string
	DismissReason   *string
	DismissNote     *string
	CompletedAt     *time.Time
}
```

### Store Layer — `business/domain/dailyplanbus/stores/dailyplandb/model.go`

```go
type dailyPlanDB struct {
	ID         uuid.UUID `db:"plan_id"`
	PlanDate   time.Time `db:"plan_date"`
	Generation int       `db:"generation"`
	ModelUsed  string    `db:"model_used"`
	PromptHash *string   `db:"prompt_hash"`
	CreatedAt  time.Time `db:"created_at"`
}

type dailyPlanItemDB struct {
	ID               uuid.UUID  `db:"item_id"`
	PlanID           uuid.UUID  `db:"plan_id"`
	TaskID           uuid.UUID  `db:"task_id"`
	Position         int        `db:"position"`
	GroupName        string     `db:"group_name"`
	GroupPosition    int        `db:"group_position"`
	AIDurationMin    *int       `db:"ai_duration_min"`
	AIPriorityReason *string    `db:"ai_priority_reason"`
	UserPosition     *int       `db:"user_position"`
	UserDurationMin  *int       `db:"user_duration_min"`
	Status           string     `db:"status"`
	DismissReason    *string    `db:"dismiss_reason"`
	DismissNote      *string    `db:"dismiss_note"`
	CompletedAt      *time.Time `db:"completed_at"`
	CreatedAt        time.Time  `db:"created_at"`
}
```

### App Layer — `app/domain/dailyplanapp/model.go`

```go
type DailyPlan struct {
	ID         string         `json:"id"`
	PlanDate   string         `json:"planDate"`
	Generation int            `json:"generation"`
	ModelUsed  string         `json:"modelUsed"`
	CreatedAt  string         `json:"createdAt"`
	Items      []DailyPlanItem `json:"items"`
}

type DailyPlanItem struct {
	ID               string  `json:"id"`
	PlanID           string  `json:"planId"`
	TaskID           string  `json:"taskId"`
	Position         int     `json:"position"`
	GroupName        string  `json:"groupName"`
	GroupPosition    int     `json:"groupPosition"`
	AIDurationMin    *int    `json:"aiDurationMin,omitempty"`
	AIPriorityReason *string `json:"aiPriorityReason,omitempty"`
	UserPosition     *int    `json:"userPosition,omitempty"`
	UserDurationMin  *int    `json:"userDurationMin,omitempty"`
	Status           string  `json:"status"`
	DismissReason    *string `json:"dismissReason,omitempty"`
	DismissNote      *string `json:"dismissNote,omitempty"`
	CompletedAt      *string `json:"completedAt,omitempty"`
	CreatedAt        string  `json:"createdAt"`
}

type DismissRequest struct {
	Reason string  `json:"reason"` // not_today, blocked, too_long, not_important, other
	Note   *string `json:"note"`
}

type UpdateItemRequest struct {
	UserPosition    *int `json:"userPosition"`
	UserDurationMin *int `json:"userDurationMin"`
}
```

### Generator — `business/domain/dailyplanbus/generator/generator.go`

```go
type TaskRef struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Priority    string  `json:"priority"`
	Energy      string  `json:"energy"`
	DurationMin *int    `json:"duration_min,omitempty"`
	DueDate     *string `json:"due_date,omitempty"`
	Context     *string `json:"context,omitempty"`
	Status      string  `json:"status"`
}

type EventRef struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	StartsAt string  `json:"starts_at"`
	EndsAt   string  `json:"ends_at"`
	Location *string `json:"location,omitempty"`
	AllDay   bool    `json:"all_day"`
}

type CarryoverItem struct {
	TaskID string `json:"task_id"`
	Title  string `json:"title"`
	Reason string `json:"reason"` // why it wasn't completed (dismissed reason or "not completed")
}

type PlanOutput struct {
	Groups []PlanGroup `json:"groups"`
}

type PlanGroup struct {
	Name   string     `json:"name"`
	Reason string     `json:"reason"`
	Items  []PlanItem `json:"items"`
}

type PlanItem struct {
	TaskID         string `json:"task_id"`
	AIDurationMin  int    `json:"ai_duration_min"`
	PriorityReason string `json:"priority_reason"`
}

type Generator struct {
	client *claudecli.Client
}
```

---

## File Map

### Handlers — `app/domain/dailyplanapp/`

- **dailyplanapp.go** — Five HTTP handlers:
  - `getPlan(ctx, r)` — GET /api/v1/daily-plan (date query param, defaults to today)
  - `generate(ctx, r)` — POST /api/v1/daily-plan/generate (calls Claude to build new plan)
  - `updateItem(ctx, r)` — PUT /api/v1/daily-plan/items/{item_id} (user position/duration overrides)
  - `completeItem(ctx, r)` — POST /api/v1/daily-plan/items/{item_id}/complete (marks done, sets CompletedAt)
  - `dismissItem(ctx, r)` — POST /api/v1/daily-plan/items/{item_id}/dismiss (rejects item, stores reason)
- **model.go** — Request/response DTOs (DailyPlan, DailyPlanItem, DismissRequest, UpdateItemRequest) + converters (toAppPlan, toAppItem, toAppItems)
- **route.go** — Routes.Add() — wires business, store, and generator dependencies; registers five routes with auth middleware
- **filter.go** — Empty (no filtering implemented)
- **order.go** — Empty (no HTTP ordering implemented)

### Business Logic — `business/domain/dailyplanbus/`

- **dailyplanbus.go** — Business struct (Storer interface, logger); five operations:
  - `Create(ctx, ne NewDailyPlan)` — Creates DailyPlan record
  - `AddItem(ctx, ni NewDailyPlanItem)` — Creates DailyPlanItem (status = "proposed")
  - `GetByDate(ctx, date)` — Queries plan + all items by date (most recent generation)
  - `UpdateItem(ctx, item, update)` — Applies UpdatePlanItem fields and persists
  - `QueryItemByID(ctx, itemID)` — Single item lookup
  - `DeleteItemsByPlan(ctx, planID)` — Cascades when regenerating
- **model.go** — Business-layer types (DailyPlan, NewDailyPlan, DailyPlanItem, NewDailyPlanItem, UpdatePlanItem)
- **order.go** — Constants for ordering: OrderByGroupPosition, OrderByPosition, OrderByCreatedAt; DefaultOrderBy = group_position ASC
- **filter.go** — Empty (no query filters)

### Store — `business/domain/dailyplanbus/stores/dailyplandb/`

- **dailyplandb.go** — Store struct (logger, DB); Storer interface implementation:
  - `CreatePlan(ctx, plan)` — INSERT INTO daily_plans (7 fields)
  - `CreateItem(ctx, item)` — INSERT INTO daily_plan_items (10 fields)
  - `UpdateItem(ctx, item)` — UPDATE daily_plan_items (user position/duration/status/dismiss reason/completed time)
  - `QueryPlanByDate(ctx, date)` — Selects most recent generation for date (ORDER BY generation DESC LIMIT 1)
  - `QueryItemsByPlan(ctx, planID)` — Selects all items for plan (ORDER BY group_position, position)
  - `QueryItemByID(ctx, itemID)` — Single item by ID
  - `DeleteItemsByPlan(ctx, planID)` — DELETE WHERE plan_id (used when regenerating)
- **model.go** — DB structs (dailyPlanDB, dailyPlanItemDB) with db tags; converters (toDBPlan, toBusPlan, toDBItem, toBusItem, toBusItems)

### Generator — `business/domain/dailyplanbus/generator/`

- **generator.go** — Generator struct (claudecli.Client); operations:
  - `NewGenerator(client)` — Constructor
  - `Generate(ctx, tasks, events, carryover)` — Calls Claude with prompt + schema, returns PlanOutput (groups of tasks with AI estimates and reasoning)
  - Includes planSchema — JSON schema for Claude structured output (groups → items with task_id, ai_duration_min, priority_reason)
- **prompt.go** — buildPlanPrompt(tasks, events, carryover) — Constructs rich system prompt for Claude:
  - Encodes tasks, events, carryover as JSON
  - Instructions to group by context/energy, order by urgency → priority → energy, estimate durations
  - Constraints on task count, prerequisite reasoning, carryover handling

---

## Impact Callouts

### DailyPlan Struct

- **Versioning**: Generation field enables multi-attempt planning. On regenerate, old items are deleted and a new plan created with incremented generation.
- **PlanDate**: Date key — allows querying "what's my plan for 2025-04-15?". Combined with generation DESC to fetch most recent.
- **PromptHash**: Optional — intended for deduplication (has Claude already generated with this task set?). Not yet used in generate flow.
- **ModelUsed**: Hardcoded to "haiku" in generate handler (TODO: make configurable).

### DailyPlanItem Struct

- **AI vs User Fields**: Four AI fields (AIDurationMin, AIPriorityReason, Position, GroupPosition) are set by Claude and immutable. User fields (UserPosition, UserDurationMin) allow client to override estimates. Status field tracks proposal → completed/dismissed progression.
- **Status Values**: "proposed" (from AI), "completed" (user marked done), "dismissed" (user rejected). No explicit "active" — relies on client-side filtering.
- **CompletedAt**: Nullable timestamp set by complete handler. Used for analytics (when did user finish tasks?).
- **DismissReason + DismissNote**: Capture why user rejected item. Reason is enum-ish (not_today, blocked, too_long, not_important, other); Note is free text. Enables feedback loop to refine future prompts.

### Storer Interface

- **Separation**: Store methods operate on business types (DailyPlan, DailyPlanItem). DB struct (dailyPlanDB, dailyPlanItemDB) hidden. Converters cross the boundary.
- **No Pagination/Filtering**: Unlike task/event domains, daily plan doesn't support paging. Plans are keyed by date + generation; items are fetched all-at-once.
- **DeleteItemsByPlan**: Cascades when regenerating. No orphaned items.

### Generator

- **AI-First Design**: Generator takes rich context (all open tasks, today's events, carryover) and returns structured PlanOutput. Handler orchestrates: fetch context → call generator → store result.
- **Task/Event Aggregation**: generate handler pulls data from taskbus (Query all todo/in_progress, filter in Go), eventbus (range query by DateFrom/DateTo), contextbus (name lookup for task contexts). Generator doesn't query directly — it's data-driven.
- **Carryover**: Computed from yesterday's plan (TODO: not yet implemented). When done, will check yesterday's items with status != "completed", extract as CarryoverItem input.
- **Schema-Driven**: Prompt + JSON schema sent to Claude. Response unmarshaled into PlanOutput. No post-hoc validation — trust Claude's adherence to schema.

---

## Routes

| Method | Path | Handler | Auth | Purpose |
|--------|------|---------|------|---------|
| GET | /api/v1/daily-plan | getPlan | APIKey | Fetch plan for date (default today). Returns empty plan if not found. |
| POST | /api/v1/daily-plan/generate | generate | APIKey | Build new plan via Claude. Handles versioning (increment generation), cascade delete old items, persist new plan+items. |
| PUT | /api/v1/daily-plan/items/{item_id} | updateItem | APIKey | Override user position/duration. Updates item, persists, returns updated item. |
| POST | /api/v1/daily-plan/items/{item_id}/complete | completeItem | APIKey | Mark item as completed. Sets status="completed", CompletedAt=now. |
| POST | /api/v1/daily-plan/items/{item_id}/dismiss | dismissItem | APIKey | Reject item. Sets status="dismissed", DismissReason, optional DismissNote. |

---

## Cross-Domain Dependencies

### Inbound (Domains that call Daily Plan)

- None yet. Daily plan is a leaf domain in the read direction (doesn't expose data to other domains). The MCP handler may expose for assistant integration.

### Outbound (Domains this depends on)

- **taskbus** (`business/domain/taskbus`) — generate handler queries all tasks (Query with empty filter, page 1-1000), filters for todo/in_progress, extracts title/priority/energy/duration/duedate/context for TaskRef. If ContextID set, calls contextbus to get context name.
  
- **eventbus** (`business/domain/eventbus`) — generate handler queries today's events (DateFrom/DateTo range filter). Converts to EventRef (ID, title, time bounds, location, all_day). Events are treated as time constraints in Claude prompt.
  
- **contextbus** (`business/domain/contextbus`) — generate handler, during task loop, looks up context by ID to get Title. Used to populate TaskRef.Context field for Claude.
  
- **claudecli** (`foundation/claudecli`) — Generator client sends prompt + schema to Claude API, receives structured JSON response. Used in generate → Generator.Generate.

### Related Components

- **app/sdk/errs** — Error codes (InvalidArgument, NotFound, Internal) for HTTP responses.
- **app/sdk/mid** — Auth middleware (APIKey) applied to all five routes.
- **app/sdk/mux** — Config struct (Log, DB, APIKey, ClaudeCLI) injected in route wiring.
- **foundation/web** — Web framework (App, Handle, HandlerFunc, Respond).
- **foundation/logger** — Structured logging.
- **business/sdk/sqldb** — SQL helpers (NamedExecContext, NamedQueryStruct, NamedQuerySlice).
- **business/sdk/page** — Pagination (used in generate handler to fetch tasks/events in batches).

---

## Database Schema (Inferred)

```sql
CREATE TABLE daily_plans (
    plan_id UUID PRIMARY KEY,
    plan_date DATE NOT NULL,
    generation INT NOT NULL,
    model_used VARCHAR NOT NULL,
    prompt_hash VARCHAR,
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE daily_plan_items (
    item_id UUID PRIMARY KEY,
    plan_id UUID NOT NULL REFERENCES daily_plans(plan_id),
    task_id UUID NOT NULL,
    position INT NOT NULL,
    group_name VARCHAR NOT NULL,
    group_position INT NOT NULL,
    ai_duration_min INT,
    ai_priority_reason VARCHAR,
    user_position INT,
    user_duration_min INT,
    status VARCHAR NOT NULL,
    dismiss_reason VARCHAR,
    dismiss_note VARCHAR,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL
);
```
