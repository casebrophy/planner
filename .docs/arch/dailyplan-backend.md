# DailyPlan Backend System

> Generates and manages AI-created daily plans that group open/blocked tasks by priority and map them against the user's calendar. Plans include AI-estimated durations and reasoning, and support user overrides (position, duration) plus item-level status tracking (proposed, completed, dismissed). Plan generation is async—POST /generate returns 202 immediately while a goroutine builds and stores the plan.

## Core Types

### App Layer

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
	Reason string  `json:"reason"`
	Note   *string `json:"note"`
}

type UpdateItemRequest struct {
	UserPosition    *int `json:"userPosition"`
	UserDurationMin *int `json:"userDurationMin"`
}
```

### Business Layer

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

### Storer Interface

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

### Generator Types (business/domain/dailyplanbus/generator/)

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

### Store Layer

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

## File Map

### App Layer (app/domain/dailyplanapp/)
- `dailyplanapp.go` — **getPlan()** fetches plan by date (defaults today), returns empty plan if none exists; **generate()** spawns goroutine for async AI plan generation, returns 202; **updateItem()** updates user-provided position/duration; **completeItem()** sets status="completed" + CompletedAt; **dismissItem()** sets status="dismissed" + reason/note
- `model.go` — **toAppPlan()**, **toAppItem()**, **toAppItems()** converts business → JSON response DTOs
- `route.go` — **Routes.Add()** registers 5 endpoints, wires dailyplanbus, taskbus, eventbus, contextbus, and generator

### Business Layer (business/domain/dailyplanbus/)
- `dailyplanbus.go` — **Create()** allocates plan ID + timestamp; **AddItem()** allocates item ID, status="proposed"; **GetByDate()** queries plan + items; **UpdateItem()** patches user fields; **QueryItemByID()** single item; **DeleteItemsByPlan()** removes all items for a plan
- `model.go` — DailyPlan, NewDailyPlan, DailyPlanItem, NewDailyPlanItem, UpdatePlanItem
- `generator/generator.go` — **NewGenerator()** creates generator with Claude CLI client; **Generate()** dispatches prompt to Claude via sidecar, parses JSON response into PlanOutput, returns (PlanOutput, modelUsed string, error)

### Store Layer (business/domain/dailyplanbus/stores/dailyplandb/)
- `dailyplandb.go` — **CreatePlan()** INSERT into daily_plans; **CreateItem()** INSERT into daily_plan_items; **UpdateItem()** UPDATE item fields; **QueryPlanByDate()** SELECT by date, DESC by generation LIMIT 1 (latest); **QueryItemsByPlan()** SELECT ordered by group_position, position; **QueryItemByID()** single item; **DeleteItemsByPlan()** DELETE all items for plan
- `model.go` — **toDBPlan()**, **toBusPlan()**, **toDBItem()**, **toBusItem()**, **toBusItems()** bidirectional converters

## Impact Callouts

### ⚠ DailyPlan / NewDailyPlan (business/domain/dailyplanbus/model.go)
Changing plan structure affects:
- `dailyplanapp/model.go` — **toAppPlan()** must map all fields
- `dailyplandb/model.go` — **toDBPlan()**, **toBusPlan()** converters
- Database schema `daily_plans` table + SQL queries

### ⚠ DailyPlanItem / NewDailyPlanItem / UpdatePlanItem (business/domain/dailyplanbus/model.go)
Changing item structure affects:
- `dailyplanapp/model.go` — **toAppItem()** must serialize all fields
- `dailyplandb/model.go` — **toDBItem()**, **toBusItem()** converters
- Database schema `daily_plan_items` table + all SQL queries
- `dailyplanapp/dailyplanapp.go` — **updateItem()**, **completeItem()**, **dismissItem()** that construct UpdatePlanItem

### ⚠ Storer interface (business/domain/dailyplanbus/dailyplanbus.go)
Adding or changing a method affects:
- `dailyplandb/dailyplandb.go` — must implement the new method
- Business methods that call the storer
- App-layer handlers
- Background plan generation job in `api/services/planner/main.go`

### ⚠ PlanOutput / PlanGroup / PlanItem (business/domain/dailyplanbus/generator/)
Changing generator output schema affects:
- JSON schema constant in `generator.go` — must match struct field names
- Claude AI prompt — must guide output to match expected structure
- Item creation loop in `dailyplanapp/dailyplanapp.go`
- Background plan generation loop in `api/services/planner/main.go`

### ⚠ Background Plan Job (api/services/planner/main.go)
Runs every 1 minute, checks if scheduled time (default 07:00, PLANNER_DAILY_PLAN_TIME) has arrived. Generates once per day (tracked via lastGenDate). Logs errors but does not retry.

## Routes

| Method | Path | Handler |
|--------|------|---------|
| GET | /api/v1/daily-plan | getPlan — fetch plan for ?date=YYYY-MM-DD (defaults today); 200 with empty plan if none exists |
| POST | /api/v1/daily-plan/generate | generate — async plan generation; returns 202 immediately |
| PUT | /api/v1/daily-plan/items/{item_id} | updateItem — update user position and/or duration |
| POST | /api/v1/daily-plan/items/{item_id}/complete | completeItem — mark as completed with timestamp |
| POST | /api/v1/daily-plan/items/{item_id}/dismiss | dismissItem — mark as dismissed with reason/note |

## Cross-Domain Dependencies

- **taskbus** — generate() fetches open+blocked tasks; background job queries tasks for plan creation
- **eventbus** — generate() fetches events for plan date; background job queries today's events
- **contextbus** — generate() enriches task refs with context names; background job builds context lookup
- **claudecli** (foundation/claudecli) — Generator.Generate() dispatches prompt via sidecar via Client.RunJSON(); requires double-envelope unwrapping and auth key alignment (PLANNER_AUTH_API_KEY); Client.LastModel() returns model used in most recent RunJSON call
- **sqldb** (business/sdk/sqldb) — store layer uses NamedExecContext, NamedQueryStruct, NamedQuerySlice
