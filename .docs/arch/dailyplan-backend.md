# DailyPlan Backend System

> AI-generated daily task plans. The user triggers generation (async) which calls Claude via `claudecli` to group open/blocked tasks into energy-based time-of-day slots. Plans are versioned by `generation` (each re-generate increments the counter and replaces items). Items track AI suggestions (position, duration, reason) and user overrides (position, duration, status). Handlers live in `dailyplanapp`; business logic in `dailyplanbus`; SQL in `dailyplandb`.

---

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
    Position         int
    GroupName        string
    GroupPosition    int
    AIDurationMin    *int
    AIPriorityReason *string
    UserPosition     *int
    UserDurationMin  *int
    Status           string       // proposed | accepted | completed | dismissed
    DismissReason    *string      // not_today | blocked | too_long | not_important | other
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

### DB Structs (`business/domain/dailyplanbus/stores/dailyplandb/model.go`)

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
    Reason string `json:"reason"`
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

---

## File Map

### Models
- `business/domain/dailyplanbus/model.go` — `DailyPlan`, `NewDailyPlan`, `DailyPlanItem`, `NewDailyPlanItem`, `UpdatePlanItem`
- `business/domain/dailyplanbus/stores/dailyplandb/model.go` — `dailyPlanDB`, `dailyPlanItemDB`; converters `toDBPlan`, `toBusPlan`, `toDBItem`, `toBusItem`, `toBusItems`
- `app/domain/dailyplanapp/model.go` — `DailyPlan`, `DailyPlanItem`, `GenerateAccepted`, `DismissRequest`, `UpdateItemRequest`; converters `toAppPlan`, `toAppItem`, `toAppItems`

### Handlers (`app/domain/dailyplanapp/dailyplanapp.go`)
- **`getPlan()`** — `GET /api/v1/daily-plan?date=YYYY-MM-DD` — fetches plan + items for a date; returns empty plan (not 404) when none exists
- **`generate()`** — `POST /api/v1/daily-plan/generate?date=YYYY-MM-DD` — returns `{status:"generating"}` immediately; spawns goroutine that calls LLM, creates/replaces plan and items
- **`updateItem()`** — `PUT /api/v1/daily-plan/items/{item_id}` — updates `userPosition` / `userDurationMin`
- **`completeItem()`** — `POST /api/v1/daily-plan/items/{item_id}/complete` — sets status=completed, completedAt=now; also marks the underlying task as `done`
- **`dismissItem()`** — `POST /api/v1/daily-plan/items/{item_id}/dismiss` — sets status=dismissed with reason + optional note

### Core (`business/domain/dailyplanbus/dailyplanbus.go`)
- **`NewBusiness(log, storer)`** — constructor
- **`Create(ctx, NewDailyPlan)`** — creates plan record with new UUID
- **`AddItem(ctx, NewDailyPlanItem)`** — creates item with status=proposed
- **`GetByDate(ctx, date)`** — queries plan by date then fetches its items
- **`UpdateItem(ctx, item, UpdatePlanItem)`** — applies partial update to item, persists
- **`QueryItemByID(ctx, itemID)`** — single item lookup
- **`DeleteItemsByPlan(ctx, planID)`** — bulk delete all items for a plan (called before re-generate)

### Generator (`business/domain/dailyplanbus/generator/`)
- `generator.go` — **`NewGenerator(claudecli)`**, **`Generate(ctx, tasks, events, carryover)`** → calls `claudecli.RunJSON` with JSON schema, returns `PlanOutput` + model name via `client.LastModel()`
- `prompt.go` — **`buildPlanPrompt(tasks, events, carryover)`** — builds the LLM prompt; energy→time-of-day mapping rules embedded here

### Store (`business/domain/dailyplanbus/stores/dailyplandb/dailyplandb.go`)
- **`NewStore(log, db)`** — constructor
- **`CreatePlan(ctx, DailyPlan)`** — INSERT into `daily_plans`
- **`CreateItem(ctx, DailyPlanItem)`** — INSERT into `daily_plan_items`
- **`UpdateItem(ctx, DailyPlanItem)`** — UPDATE user_position, user_duration_min, status, dismiss_reason, dismiss_note, completed_at WHERE item_id
- **`QueryPlanByDate(ctx, date)`** — SELECT latest generation for date (ORDER BY generation DESC LIMIT 1)
- **`QueryItemsByPlan(ctx, planID)`** — SELECT items ORDER BY group_position, position
- **`QueryItemByID(ctx, itemID)`** — SELECT single item
- **`DeleteItemsByPlan(ctx, planID)`** — DELETE all items for a plan

### Routes
- `app/domain/dailyplanapp/route.go` — wires `dailyplandb`, `taskdb`, `eventdb`, `contextdb`, `generator`; registers all 5 endpoints with `mid.Auth`

### Order
- `business/domain/dailyplanbus/order.go` — `OrderByGroupPosition`, `OrderByPosition`, `OrderByCreatedAt`; `DefaultOrderBy = group_position ASC`
- `business/domain/dailyplanbus/stores/dailyplandb/order.go` — empty (store queries use literal ORDER BY clauses, not the order SDK)

### Tests
- `business/domain/dailyplanbus/dailyplanbus_test.go` — business layer unit tests
- `app/domain/dailyplanapp/tests/dailyplanapi/query_test.go` — API integration tests
- `app/domain/dailyplanapp/tests/dailyplanapi/dailyplan_test.go` — API integration tests

---

## Impact Callouts

### ⚠ DailyPlan (business/domain/dailyplanbus/model.go)
Changing this struct shape affects:
- `business/domain/dailyplanbus/dailyplanbus.go` — constructed in `Create()`, returned from `GetByDate()`
- `business/domain/dailyplanbus/stores/dailyplandb/model.go` — `toDBPlan` / `toBusPlan` field mapping
- `business/domain/dailyplanbus/stores/dailyplandb/dailyplandb.go` — `CreatePlan` INSERT columns; `QueryPlanByDate` SELECT columns
- `app/domain/dailyplanapp/model.go` — `toAppPlan()` field mapping
- `app/domain/dailyplanapp/dailyplanapp.go` — `generate()` reads `existingPlan.Generation` for increment
- Migration required if DB column added/removed (table: `daily_plans`)

### ⚠ DailyPlanItem (business/domain/dailyplanbus/model.go)
Changing this struct shape affects:
- `business/domain/dailyplanbus/dailyplanbus.go` — constructed in `AddItem()`, mutated in `UpdateItem()`
- `business/domain/dailyplanbus/stores/dailyplandb/model.go` — `toDBItem` / `toBusItem` field mapping
- `business/domain/dailyplanbus/stores/dailyplandb/dailyplandb.go` — `CreateItem` INSERT columns; `UpdateItem` SET columns; `QueryItemsByPlan` / `QueryItemByID` SELECT columns
- `app/domain/dailyplanapp/model.go` — `toAppItem()` field mapping
- `app/domain/dailyplanapp/dailyplanapp.go` — `completeItem()`, `dismissItem()`, `updateItem()` all build `UpdatePlanItem` from item fields
- Migration required if DB column added/removed (table: `daily_plan_items`); `status` and `dismiss_reason` have DB CHECK constraints

### ⚠ UpdatePlanItem (business/domain/dailyplanbus/model.go)
Changing this struct affects:
- `business/domain/dailyplanbus/dailyplanbus.go` — `UpdateItem()` reads all fields; nil = no-change semantics
- `app/domain/dailyplanapp/dailyplanapp.go` — `updateItem()`, `completeItem()`, `dismissItem()` all construct this struct

### ⚠ Storer interface (business/domain/dailyplanbus/dailyplanbus.go)
Adding/changing a method affects:
- `business/domain/dailyplanbus/stores/dailyplandb/dailyplandb.go` — must implement the method
- `business/domain/dailyplanbus/dailyplanbus.go` — `Business` calls the method
- `app/domain/dailyplanapp/dailyplanapp.go` — may need new handler if new query path
- `app/domain/dailyplanapp/route.go` — may need new route

### ⚠ Generator.Generate() / PlanOutput shape (business/domain/dailyplanbus/generator/generator.go)
Changing inputs or `PlanOutput`/`PlanGroup`/`PlanItem` struct shape affects:
- `app/domain/dailyplanapp/dailyplanapp.go` — `generate()` calls `a.generator.Generate(...)` and iterates `planOutput.Groups[].Items[]`
- `business/domain/dailyplanbus/generator/prompt.go` — `buildPlanPrompt` must reflect any new input types
- `planSchema` JSON string inside `generator.go` must stay in sync with struct tags

### ⚠ item_id status CHECK constraint (migration SQL)
Status values `proposed | accepted | completed | dismissed` are enforced by DB CHECK on `daily_plan_items.status`. Adding a new value requires:
- New migration SQL altering the CHECK constraint
- Update status string literals in `dailyplanbus.go` (`AddItem` hardcodes `"proposed"`) and `dailyplanapp.go` (`"completed"`, `"dismissed"`)

---

## Routes

| Method | Path | Handler | Auth |
|--------|------|---------|------|
| GET | `/api/v1/daily-plan` | `getPlan` | API key |
| POST | `/api/v1/daily-plan/generate` | `generate` | API key |
| PUT | `/api/v1/daily-plan/items/{item_id}` | `updateItem` | API key |
| POST | `/api/v1/daily-plan/items/{item_id}/complete` | `completeItem` | API key |
| POST | `/api/v1/daily-plan/items/{item_id}/dismiss` | `dismissItem` | API key |

---

## Cross-Domain Dependencies

- **taskbus** — `generate()` calls `taskBus.Query()` (open/blocked filter applied in Go); `completeItem()` calls `taskBus.Update()` to mark task done; wired via `taskdb.NewStore` + `taskdb.NewDependencyStore` in `route.go`
- **eventbus** — `generate()` calls `eventBus.Query()` with date-range filter to fetch today's events; wired via `eventdb.NewStore` in `route.go`
- **contextbus** — `generate()` calls `contextBus.QueryByID()` to resolve context names for task refs; wired via `contextdb.NewStore` in `route.go`
- **claudecli** (`foundation/claudecli`) — `generator.Generator` uses `client.RunJSON()` with `shouldEscalate` callback; `client.LastModel()` returns the model name stored in `DailyPlan.ModelUsed`
- **mid.Auth** — all 5 routes protected by API key middleware
- **sqldb.ErrDBNotFound** — `getPlan` returns empty DailyPlan struct (not 404) when no plan exists for the date; `updateItem`/`completeItem`/`dismissItem` return 404
