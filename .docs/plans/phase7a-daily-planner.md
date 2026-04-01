# Phase 7a — Daily Planner Implementation Plan

## Context

AI-generated daily task plan with smart grouping + events as fixed commitments. The planner becomes the user's primary scheduling tool. Voice ingest already exists — extend it to classify events vs. tasks.

## Implementation Order

Build bottom-up: schema → business layer → API → frontend. Split into 3 milestones.

---

## Milestone 1: Events (backend)

Events are fixed commitments. Need full CRUD before daily plan can reason about them.

### 1.1 Migration — events table

File: `business/sdk/migrate/sql/` (new migration file)

```sql
CREATE TABLE events (
    event_id      UUID        NOT NULL DEFAULT gen_random_uuid(),
    context_id    UUID        REFERENCES contexts(context_id) ON DELETE SET NULL,
    title         TEXT        NOT NULL,
    description   TEXT        NOT NULL DEFAULT '',
    location      TEXT,
    starts_at     TIMESTAMPTZ NOT NULL,
    ends_at       TIMESTAMPTZ NOT NULL,
    all_day       BOOLEAN     NOT NULL DEFAULT FALSE,
    raw_input_id  UUID        REFERENCES raw_inputs(raw_input_id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id)
);
CREATE INDEX idx_events_date ON events(starts_at, ends_at);
CREATE INDEX idx_events_context ON events(context_id);
```

### 1.2 Business domain — eventbus

Follow the standard three-layer pattern. Files:

| File | Contents |
|------|----------|
| `business/domain/eventbus/model.go` | `Event`, `NewEvent`, `UpdateEvent` structs |
| `business/domain/eventbus/eventbus.go` | `Business` struct, `Storer` interface, CRUD methods |
| `business/domain/eventbus/filter.go` | `QueryFilter` — by date range, context_id |
| `business/domain/eventbus/order.go` | `OrderByStartsAt` (default ASC) |
| `business/domain/eventbus/stores/eventdb/model.go` | DB struct + converters |
| `business/domain/eventbus/stores/eventdb/eventdb.go` | SQL queries |
| `business/domain/eventbus/stores/eventdb/filter.go` | `applyFilter()` |
| `business/domain/eventbus/stores/eventdb/order.go` | `orderByFields` map |

### 1.3 App layer — eventapp

| File | Contents |
|------|----------|
| `app/domain/eventapp/model.go` | App DTOs + converters |
| `app/domain/eventapp/eventapp.go` | Handlers: create, update, delete, queryAll, queryByID |
| `app/domain/eventapp/route.go` | Routes with Auth middleware |
| `app/domain/eventapp/filter.go` | `parseFilter()` — date_from, date_to, context_id |
| `app/domain/eventapp/order.go` | `parseOrder()` |

Routes:
- `GET /api/v1/events` — list with date range filter
- `GET /api/v1/events/{event_id}` — get by ID
- `POST /api/v1/events` — create
- `PUT /api/v1/events/{event_id}` — update
- `DELETE /api/v1/events/{event_id}` — delete

### 1.4 MCP tools — events

Add to `app/domain/mcpapp/`:
- `create_event` tool — title, starts_at, ends_at required; location, all_day, description, context_id optional
- `list_events` tool — filter by date range

### 1.5 Voice ingest — event classification

Update extractor to distinguish tasks vs. events:

**`business/domain/ingestbus/extractor/model.go`**
- Add to `TextExtraction`: `Events []ExtractedEvent`

```go
type ExtractedEvent struct {
    Title       string `json:"title"`
    Description string `json:"description"`
    Location    string `json:"location,omitempty"`
    StartsAt    string `json:"starts_at"`        // ISO datetime or natural language
    EndsAt      string `json:"ends_at,omitempty"` // if omitted, assume 1 hour
    AllDay      bool   `json:"all_day"`
    IsAmbiguous bool   `json:"is_ambiguous"`      // generates clarification if true
}
```

**`business/domain/ingestbus/extractor/prompt.go`**
- Update `BuildTextExtractionPrompt` to include events in the schema and rules:
  - "Distinguish between tasks (things to do) and events (fixed commitments with a specific date/time)"
  - "Examples: 'dentist at 2pm Thursday' = event; 'wash the dishes' = task; 'wedding on June 15 in Napa' = event with location"

**`business/domain/ingestbus/ingestbus.go`**
- Update `ProcessText` to create events from `extraction.Events` (in addition to tasks)
- Generate clarifications for ambiguous events (e.g., "road trip this weekend" — which days exactly?)
- Return event IDs alongside task IDs

### 1.6 Tests

- `business/domain/eventbus/` — CRUD unit tests (unitest.Table pattern)
- `app/domain/eventapp/tests/eventapi/` — API tests (apitest.Table pattern): 200, 400, 401
- `business/domain/ingestbus/ingestbus_test.go` — add `processTextCreatesEvent` test case

---

## Milestone 2: Daily Plan (backend)

### 2.1 Migration — daily_plans + daily_plan_items tables

```sql
CREATE TABLE daily_plans (
    plan_id       UUID        NOT NULL DEFAULT gen_random_uuid(),
    plan_date     DATE        NOT NULL,
    generation    INTEGER     NOT NULL DEFAULT 1,
    model_used    TEXT        NOT NULL,
    prompt_hash   TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (plan_id)
);
CREATE INDEX idx_daily_plans_date ON daily_plans(plan_date DESC);

CREATE TABLE daily_plan_items (
    item_id             UUID        NOT NULL DEFAULT gen_random_uuid(),
    plan_id             UUID        NOT NULL REFERENCES daily_plans(plan_id) ON DELETE CASCADE,
    task_id             UUID        NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    position            INTEGER     NOT NULL,
    group_name          TEXT        NOT NULL DEFAULT 'ungrouped',
    group_position      INTEGER     NOT NULL DEFAULT 0,
    ai_duration_min     INTEGER,
    ai_priority_reason  TEXT,
    user_position       INTEGER,
    user_duration_min   INTEGER,
    status              TEXT        NOT NULL DEFAULT 'proposed' CHECK (status IN ('proposed', 'accepted', 'completed', 'dismissed')),
    dismiss_reason      TEXT        CHECK (dismiss_reason IN ('not_today', 'blocked', 'too_long', 'not_important', 'other')),
    dismiss_note        TEXT,
    completed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (item_id)
);
CREATE INDEX idx_daily_plan_items_plan ON daily_plan_items(plan_id, group_position, position);
```

### 2.2 Business domains

**`business/domain/dailyplanbus/`** — standard three-layer:
- `model.go` — `DailyPlan`, `DailyPlanItem`, `NewDailyPlan`, `UpdatePlanItem` structs
- `dailyplanbus.go` — `Business` struct with `Storer` interface, methods:
  - `Generate(ctx, date) (DailyPlan, error)` — orchestrates plan generation
  - `GetByDate(ctx, date) (DailyPlan, []DailyPlanItem, error)`
  - `UpdateItem(ctx, itemID, UpdatePlanItem) error` — reorder, dismiss, complete
  - `Regenerate(ctx, date) (DailyPlan, error)` — increments generation counter
- `stores/dailyplandb/` — SQL queries

**`business/domain/dailyplanbus/generator/`** — plan generation logic:
- `generator.go` — `Generate(ctx, tasks, events, previousPlan) (PlanOutput, error)`
  - Fetches open tasks + today's events
  - Builds Claude prompt with: tasks (title, priority, energy, duration, due_date, context), events (title, time, location), yesterday's incomplete items
  - Claude returns: grouped + ordered task list with AI duration estimates and priority reasoning
  - Parses response into `DailyPlan` + `[]DailyPlanItem`

**Claude prompt for plan generation:**
- Input: open tasks, today's events, yesterday's carryover
- Output schema:
```json
{
  "groups": [
    {
      "name": "Errands",
      "reason": "These tasks involve going out",
      "items": [
        {
          "task_id": "uuid",
          "position": 1,
          "ai_duration_min": 30,
          "priority_reason": "Windshield wipers needed before road trip Saturday"
        }
      ]
    }
  ]
}
```
- Rules: consider events as constraints, group by context/errand-type/energy, surface prerequisite relationships, carry forward yesterday's incomplete tasks

### 2.3 Scheduled job — morning batch

In `api/services/planner/main.go`:
- Add a goroutine that runs daily at configurable time (env: `PLANNER_DAILY_PLAN_TIME`, default `07:00`)
- Calls `dailyplanbus.Generate(ctx, today)`
- Same pattern as the existing inactivity check goroutine

### 2.4 App layer — dailyplanapp

| File | Contents |
|------|----------|
| `app/domain/dailyplanapp/model.go` | App DTOs for plan + items |
| `app/domain/dailyplanapp/dailyplanapp.go` | Handlers |
| `app/domain/dailyplanapp/route.go` | Route registration |

Routes:
- `GET /api/v1/daily-plan?date=2026-04-01` — get plan for date (today if omitted)
- `POST /api/v1/daily-plan/generate?date=2026-04-01` — generate/regenerate plan
- `PUT /api/v1/daily-plan/items/{item_id}` — update item (reorder, dismiss, override duration)
- `POST /api/v1/daily-plan/items/{item_id}/complete` — mark item completed
- `POST /api/v1/daily-plan/items/{item_id}/dismiss` — dismiss with reason + note

### 2.5 MCP tools

Add to `app/domain/mcpapp/`:
- `get_daily_plan` — returns today's plan with grouped items
- `generate_daily_plan` — triggers plan generation

### 2.6 Tests

- `business/domain/dailyplanbus/` — unit tests with mock generator
- `app/domain/dailyplanapp/tests/` — API tests: get plan, update item, dismiss
- Generator tests with mock Claude extractor

---

## Milestone 3: Frontend

### 3.1 Stores

- `src/stores/events.ts` — Pinia store for event CRUD
- `src/stores/dailyPlan.ts` — Pinia store for daily plan (get, generate, update items)

### 3.2 Components

- `PlanItemCard` — task card with drag handle, AI duration badge, dismiss/complete actions
- `PlanGroupHeader` — group name + collapse toggle
- `EventCard` — event display (time, title, location)
- `EventForm` — create/edit event form
- `DismissModal` — structured reason picker + freeform note input

### 3.3 Views

- `DailyPlanView` — the main view:
  - Today's events at top (fixed, not reorderable)
  - Grouped task list below (drag-reorderable within and between groups)
  - Each item: PlanItemCard with swipe-to-dismiss on mobile
  - "Regenerate" button
  - Empty state: "No plan yet — generate one?"
- `EventsView` — list/create/edit events (simple CRUD view)

### 3.4 Navigation

- Add "Plan" to web sidebar (between Dashboard and Tasks)
- Replace "Today" in mobile tab bar with "Plan" (or make "Today" show the daily plan)

---

## Env vars

| Var | Default | Purpose |
|-----|---------|---------|
| `PLANNER_DAILY_PLAN_TIME` | `07:00` | When morning batch runs (local time) |
| `PLANNER_DAILY_PLAN_ENABLED` | `true` | Enable/disable morning batch |

## Verification

1. `make migrate` — events + daily plan tables created
2. Create events via API: `POST /api/v1/events`
3. Voice ingest creates event: `curl -X POST .../ingest/voice -d '{"text": "dentist appointment Thursday at 2pm"}'`
4. Generate daily plan: `POST /api/v1/daily-plan/generate`
5. Plan includes events as constraints, groups tasks intelligently
6. Dismiss item with reason: `POST /api/v1/daily-plan/items/{id}/dismiss`
7. Frontend: daily plan view renders, drag reorder works, dismiss captures reason
