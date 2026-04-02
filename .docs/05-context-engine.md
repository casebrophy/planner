# Context engine & scheduling

The context engine is the set of operations on the data model that collectively produce context-aware behaviour — no discrete service or binary. Claude + MCP tools + the data model work together.

## Context operations

| Operation | Trigger | What happens |
|---|---|---|
| **create_context** | Pipeline finds unmatched ongoing content, or user request | Claude sets title + initial description |
| **update_context** | New content linked to a context | Appends event, rewrites `contexts.summary` |
| **get_context** | User asks "what's happening with X?" | Returns summary + open tasks for Claude to reason over |
| **list_contexts** | Broad planning ("what do I need this week?") | Returns all active contexts + summaries |

## Summary rewrite rules

`contexts.summary` is working memory — compresses the full event log into ~200–400 words. Always preserve:
- Core: what the context is about
- Current status: what's happening right now
- Outstanding items: what's unresolved
- Key facts: names, numbers, dates that matter
- Recent events: last 2–3 significant things

Older superseded detail may be dropped.

## Context lifecycle

```
active    ← default; appears in all queries and planning
paused    ← temporarily dormant (waiting on someone else)
closed    ← resolved; excluded from planning, but searchable
```

Contexts are never deleted. Event log and linked tasks/transactions/emails are preserved permanently.

## Scheduling

Two layers: **daily plan** (Phase 7a, grouped task list) and **time blocks** (Phase 7b, calendar slots).

### Daily Plan (Phase 7a)

AI-generated prioritized, grouped task list. No time slots — just "do these things today" in this order.

**Generation inputs:**

| Input | Required | Degraded if missing |
|---|---|---|
| Tasks with `priority` + `status=todo` | Yes | — |
| `duration_min` on tasks | No | AI estimates at plan time; stored as `ai_duration_min` |
| Deadlines (`due_date`) | No | Urgency sort degrades |
| Energy level per task | No | Grouping by energy mode skipped |
| Active contexts | No | Context-based grouping skipped |
| Yesterday's plan results | No | Carryover logic skipped |

**Generation algorithm:**
1. Collect open tasks (todo + in_progress) across all active contexts
2. Claude groups by context/errand-type/energy-mode (e.g. "errands", "deep work", "admin calls", or context title)
3. Within each group, order by: urgency (due date) → priority → energy
4. Estimate duration for tasks missing `duration_min`
5. Carry forward incomplete tasks from yesterday's plan (if any)
6. Produce `daily_plan` + `daily_plan_items` with `status = proposed`

**Triggers:**
- Morning batch job (configurable time, default 7am)
- On-demand regeneration via API or frontend

**User interactions (all captured for training data):**
- Drag reorder → saves `user_position`
- Override duration → saves `user_duration_min`
- Dismiss → saves `dismiss_reason` (structured) + `dismiss_note` (freeform)
- Complete task → saves `completed_at` on plan item

**Dismiss reasons (structured):**

| Value | Meaning |
|---|---|
| `not_today` | Generic defer |
| `blocked` | Waiting on someone/something |
| `too_long` | Duration estimate too high for today |
| `not_important` | Priority was wrong |
| `other` | See `dismiss_note` |

### Time Blocks (Phase 7b)

Self-contained time-slotted scheduling. Any task can be assigned to a time block — not limited to daily plan items.

1. User manually assigns tasks to time slots via calendar view or API
2. Events from `events` table are the availability constraints (fixed commitments)
3. 15-min buffer between tasks (configurable)
4. Blocks created with `confirmed = false` (proposed); user confirms when committed
5. No external calendar integration — the planner is the calendar

### Auto-schedule from ingestion

When pipeline extracts a deadline from email/voice:
1. Creates task with implied action + `due_date`
2. Task appears in next daily plan generation
3. Does **not** auto-schedule without user review

## MCP tools

| Tool | Purpose |
|---|---|
| `create_task` | Create a new task (title, status, priority, energy, context, due date, duration) |
| `list_tasks` | Query tasks with filters (status, priority, context) |
| `get_task` | Get a single task by ID |
| `update_task` | Update task fields (title, status, priority, context, etc.) |
| `complete_task` | Mark a task as done (sets status=done, completed_at=now) |
| `create_context` | New ongoing context (pipeline or user) |
| `get_context` | Summary + open tasks + recent events for a context |
| `list_contexts` | All active contexts with titles + summaries |
| `update_context` | Rename, re-describe, or close a context |
| `list_emails` | Query ingested emails |
| `get_email` | Get a single email by ID |
| `get_clarification_queue` | Pending clarification items (filterable by kind) |
| `resolve_clarification` | Resolve a clarification item with an answer |
| `snooze_clarification` | Snooze a clarification item |
| `add_thread_entry` | Append a note/update to a task or context thread |
| `get_thread` | Get thread entries for a subject (task or context) |
| `record_outcome` | Record an outcome observation for a task or context |
| `get_outcome_observations` | Query observations for a subject |
| `create_event` | Create a calendar event (title, start/end, location, context) |
| `list_events` | Query events with date range and context filters |
| `get_event` | Get a single event by ID |
| `update_event` | Update event fields |
| `delete_event` | Delete an event |
| `get_daily_plan` | Get today's (or a specific date's) daily plan with items |
| `generate_daily_plan` | Generate or regenerate a daily plan for a date |
| `get_schedule` | Events + time blocks merged for a date range (Phase 7b) |
| `create_time_block` | Schedule any task into a specific time slot (Phase 7b) |
| `confirm_time_block` | Mark a proposed block as confirmed (Phase 7b) |

## Frontend views

- **Context board** — active contexts: title, summary excerpt, open task count, last activity
- **Context detail** — full event timeline, open tasks, linked items, Claude-maintained summary
- **Task board** — flat task list across all contexts; filter by priority/due date
- **Schedule view** — weekly calendar of proposed + confirmed time blocks
- **Transaction review** — triage queue for unassigned transactions
