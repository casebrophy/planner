# Daily Plan Backend System

> Manages AI-generated daily task plans. For a given date, the system fetches open tasks and calendar events, invokes Claude (haiku) asynchronously to produce a grouped, prioritized plan with duration estimates, and persists the result as a versioned `daily_plan` + `daily_plan_items` pair. Users can then accept, complete, or dismiss individual items; dismissals carry a structured reason. Regeneration increments the `generation` counter on a new plan row and deletes the previous generation's items.

## Core Types

### Business Layer (`business/domain/dailyplanbus/model.go`)

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
    Position         int        // global sort order across all groups
    GroupName        string
    GroupPosition    int        // sort order within the group
    AIDurationMin    *int
    AIPriorityReason *string
    UserPosition     *int       // user override
    UserDurationMin  *int       // user override
    Status           string     // proposed | accepted | completed | dismissed
    DismissReason    *string    // not_today | blocked | too_long | not_important | other
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

### Storer Interface (`business/domain/dailyplanbus/dailyplanbus.go`)

```go
type Storer interface {
    CreatePlan(ctx context.Context, plan DailyPlan) error
    CreateItem(ctx context.Context, item DailyPlanItem) error
    UpdateItem(ctx context.Context, item DailyPlanItem) error
    QueryPlanByDate(ctx context.Context, date time.Time) (DailyPlan, error)
    QueryItemsByPlan(ctx context.Context, planID uuid.UUID) ([]DailyPlanItem, error)
    QueryItemByID(ctx context.Context, itemID uuid.UUID) (DailyPlanItem, error)
    DeleteItemsByPlan(ctx context.Context, planID uuid.UUID) error
}
```

### App Layer DTOs (`app/domain/dailyplanapp/model.go`)

```go
type DailyPlan struct {
    ID         string          `json:"id"`
    PlanDate   string          `json:"planDate"`
    Generation int             `json:"generation"`
    ModelUsed  string          `json:"modelUsed"`
    CreatedAt  string          `json:"createdAt"`
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

type GenerateAccepted struct {
    Status string `json:"status"`
}

type DismissRequest struct {
    Reason string  `json:"reason"` // not_today | blocked | too_long | not_important | other
    Note   *string `json:"note"`
}

type UpdateItemRequest struct {
    UserPosition    *int `json:"userPosition"`
    UserDurationMin *int `json:"userDurationMin"`
}
```

### Store DB Structs (`business/domain/dailyplanbus/stores/dailyplandb/model.go`)

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

### Generator Types (`business/domain/dailyplanbus/generator/generator.go`)

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
    Reason string `json:"reason"` // dismissed reason or "not completed"
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
```

## File Map

### Models
- `business/domain/dailyplanbus/model.go` — `DailyPlan`, `NewDailyPlan`, `DailyPlanItem`, `NewDailyPlanItem`, `UpdatePlanItem`
- `business/domain/dailyplanbus/stores/dailyplandb/model.go` — `dailyPlanDB`, `dailyPlanItemDB`; converters `toDBPlan`, `toBusPlan`, `toDBItem`, `toBusItem`, `toBusItems`
- `app/domain/dailyplanapp/model.go` — app-layer DTOs; converters `toAppPlan`, `toAppItem`, `toAppItems`
- `business/domain/dailyplanbus/generator/generator.go` — `TaskRef`, `EventRef`, `CarryoverItem`, `PlanOutput`, `PlanGroup`, `PlanItem`; `planSchema` JSON Schema const

### Handlers (`app/domain/dailyplanapp/`)
- `dailyplanapp.go` — **getPlan()** `GET /api/v1/daily-plan` — returns empty plan on not-found instead of 404
- `dailyplanapp.go` — **generate()** `POST /api/v1/daily-plan/generate` — queries tasks/events, spawns goroutine for LLM + DB writes, returns 202 `{"status":"generating"}`
- `dailyplanapp.go` — **updateItem()** `PUT /api/v1/daily-plan/items/{item_id}` — user position/duration overrides
- `dailyplanapp.go` — **completeItem()** `POST /api/v1/daily-plan/items/{item_id}/complete` — sets status=completed + completed_at=now
- `dailyplanapp.go` — **dismissItem()** `POST /api/v1/daily-plan/items/{item_id}/dismiss` — sets status=dismissed + reason/note
- `route.go` — wires dailyplandb → dailyplanbus, taskdb → taskbus, eventdb → eventbus, contextdb → contextbus, generator; registers all 5 routes under auth middleware
- `filter.go` — placeholder (empty)
- `order.go` — placeholder (empty)

### Core (`business/domain/dailyplanbus/`)
- `dailyplanbus.go` — **Create()** creates plan row; **AddItem()** inserts item with status="proposed"; **GetByDate()** fetches latest plan + items; **UpdateItem()** applies UpdatePlanItem patch; **QueryItemByID()** single item lookup; **DeleteItemsByPlan()** bulk delete for regeneration
- `order.go` — `OrderByGroupPosition` (default ASC), `OrderByPosition`, `OrderByCreatedAt`
- `filter.go` — placeholder (empty)
- `generator/generator.go` — **Generate()** builds prompt, calls `claudecli.Client.RunJSON()` with JSON schema validation, escalates (higher model) if groups empty
- `generator/prompt.go` — **buildPlanPrompt()** constructs system prompt with task/event/carryover context; instructs grouping by context/energy, ordering by urgency→priority→energy, duration estimation

### Store (`business/domain/dailyplanbus/stores/dailyplandb/`)
- `dailyplandb.go` — **CreatePlan()** INSERT into `daily_plans`; **CreateItem()** INSERT into `daily_plan_items`; **UpdateItem()** UPDATE user overrides + status fields; **QueryPlanByDate()** SELECT latest generation for date (ORDER BY generation DESC LIMIT 1); **QueryItemsByPlan()** SELECT all items ORDER BY group_position, position; **QueryItemByID()** SELECT by item_id; **DeleteItemsByPlan()** DELETE all items for plan
- `filter.go` — placeholder (empty)
- `order.go` — placeholder (empty)

### Tests
- `app/domain/dailyplanapp/tests/dailyplanapi/dailyplan_test.go` — `getPlan200()`, `getPlan401()`
- `app/domain/dailyplanapp/tests/dailyplanapi/query_test.go` — API test table for GET /api/v1/daily-plan
- `business/domain/dailyplanbus/dailyplanbus_test.go` — `createPlan()`, `createAndQueryItems()`, `updateItem()`, `dismissItem()`

## Impact Callouts

### ⚠ DailyPlan (`business/domain/dailyplanbus/model.go`)
Changing this struct shape affects:
- `business/domain/dailyplanbus/dailyplanbus.go` — assembled in `Create()`, passed to `Storer.CreatePlan()`
- `business/domain/dailyplanbus/stores/dailyplandb/model.go` — `toDBPlan()`/`toBusPlan()` field mapping
- `business/domain/dailyplanbus/stores/dailyplandb/dailyplandb.go` — INSERT column list in `CreatePlan()`, SELECT column list in `QueryPlanByDate()`
- `app/domain/dailyplanapp/model.go` — `toAppPlan()` field mapping to app DTO
- `app/domain/dailyplanapp/dailyplanapp.go` — `generate()` goroutine accesses `existingPlan.Generation`, `existingPlan.ID`
- Migration required if DB column added/removed

### ⚠ DailyPlanItem (`business/domain/dailyplanbus/model.go`)
Changing this struct shape affects:
- `business/domain/dailyplanbus/dailyplanbus.go` — assembled in `AddItem()`, mutated in `UpdateItem()`; passed to all Storer item methods
- `business/domain/dailyplanbus/stores/dailyplandb/model.go` — `toDBItem()`/`toBusItem()` mapping (all 15 fields)
- `business/domain/dailyplanbus/stores/dailyplandb/dailyplandb.go` — INSERT columns in `CreateItem()`, UPDATE SET fields in `UpdateItem()`, SELECT columns in `QueryItemsByPlan()` and `QueryItemByID()`
- `app/domain/dailyplanapp/model.go` — `toAppItem()` optional-field forwarding logic
- `app/domain/dailyplanapp/dailyplanapp.go` — `updateItem()`, `completeItem()`, `dismissItem()` all construct `UpdatePlanItem` from this
- Migration required if DB column added/removed

### ⚠ UpdatePlanItem (`business/domain/dailyplanbus/model.go`)
Changing this struct shape affects:
- `business/domain/dailyplanbus/dailyplanbus.go` — `UpdateItem()` applies each field conditionally (nil = no-op)
- `app/domain/dailyplanapp/dailyplanapp.go` — constructed in `updateItem()`, `completeItem()`, `dismissItem()`
- `app/domain/dailyplanapp/model.go` — `UpdateItemRequest` and `DismissRequest` map to this struct

### ⚠ Storer interface (`business/domain/dailyplanbus/dailyplanbus.go`)
Adding/changing a method affects:
- `business/domain/dailyplanbus/stores/dailyplandb/dailyplandb.go` — must implement the method
- `business/domain/dailyplanbus/dailyplanbus.go` — all Business methods delegate to Storer
- `app/domain/dailyplanapp/route.go` — store constructor unchanged unless signature changes
- Any Storer test doubles must be updated

### ⚠ PlanOutput / PlanGroup / PlanItem (`business/domain/dailyplanbus/generator/generator.go`)
Changing these types or the JSON schema affects:
- `business/domain/dailyplanbus/generator/generator.go` — `planSchema` const must stay in sync with struct fields and `required` list
- `app/domain/dailyplanapp/dailyplanapp.go` — `generate()` goroutine iterates `planOutput.Groups`, accesses `group.Name`, `item.TaskID`, `item.AIDurationMin`, `item.PriorityReason`

### ⚠ Generator (`business/domain/dailyplanbus/generator/generator.go`)
Changing `Generate()` signature or `NewGenerator()` constructor affects:
- `app/domain/dailyplanapp/route.go` — `generator.NewGenerator(cfg.ClaudeCLI)`
- `app/domain/dailyplanapp/dailyplanapp.go` — `a.generator.Generate(bgCtx, taskRefs, eventRefs, carryover)`

## Routes

| Method | Path | Handler | Auth |
|--------|------|---------|------|
| GET | `/api/v1/daily-plan` | `getPlan` | API key |
| POST | `/api/v1/daily-plan/generate` | `generate` | API key |
| PUT | `/api/v1/daily-plan/items/{item_id}` | `updateItem` | API key |
| POST | `/api/v1/daily-plan/items/{item_id}/complete` | `completeItem` | API key |
| POST | `/api/v1/daily-plan/items/{item_id}/dismiss` | `dismissItem` | API key |

## Cross-Domain Dependencies

- **taskbus** — `generate()` queries all tasks, filters for `Open`/`Blocked` status (`business/types/taskstatus`)
- **contextbus** — `generate()` resolves context name for each task's optional `ContextID`
- **eventbus** — `generate()` queries events with date-range filter (today 00:00–24:00) for blocking time slots
- **claudecli** (`foundation/claudecli`) — `generator.Generator` calls `client.RunJSON()` to invoke Claude via CLI; escalates to higher model if response has empty groups
- **MCP** (`app/domain/mcpapp`) — holds `dailyplanbus.Business` reference; exposes `get_daily_plan` and `generate_daily_plan` MCP tools (generate not yet fully wired in MCP)

## DB Schema

```sql
CREATE TABLE daily_plans (
    plan_id     UUID PRIMARY KEY,
    plan_date   DATE NOT NULL,
    generation  INTEGER DEFAULT 1,
    model_used  TEXT NOT NULL,
    prompt_hash TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_daily_plans_date ON daily_plans(plan_date DESC);

CREATE TABLE daily_plan_items (
    item_id             UUID PRIMARY KEY,
    plan_id             UUID NOT NULL REFERENCES daily_plans(plan_id) ON DELETE CASCADE,
    task_id             UUID NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    position            INTEGER NOT NULL,
    group_name          TEXT DEFAULT 'ungrouped',
    group_position      INTEGER DEFAULT 0,
    ai_duration_min     INTEGER,
    ai_priority_reason  TEXT,
    user_position       INTEGER,
    user_duration_min   INTEGER,
    status              TEXT CHECK (status IN ('proposed','accepted','completed','dismissed')),
    dismiss_reason      TEXT CHECK (dismiss_reason IN ('not_today','blocked','too_long','not_important','other')),
    dismiss_note        TEXT,
    completed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_daily_plan_items_plan ON daily_plan_items(plan_id, group_position, position);
```
