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

Calendar-aware scheduling. Extends daily plan with actual time slots.

1. Consume iCal feed for availability
2. Fit daily plan items into available slots respecting `duration_min`
3. Insert 15-min buffer between tasks (configurable)
4. Produce `time_blocks` with `confirmed = false` (proposed)
5. Confirmed blocks optionally sync back to calendar

### Auto-schedule from ingestion

When pipeline extracts a deadline from email/voice:
1. Creates task with implied action + `due_date`
2. Task appears in next daily plan generation
3. Does **not** auto-schedule without user review

## MCP tools

| Tool | Purpose |
|---|---|
| `create_context` | New ongoing context (pipeline or user) |
| `get_context` | Summary + open tasks + recent events for a context |
| `list_contexts` | All active contexts with titles + summaries |
| `update_context` | Rename, re-describe, or close a context |
| `get_daily_plan` | Get today's (or a specific date's) daily plan with items |
| `generate_daily_plan` | Generate or regenerate a daily plan for a date |
| `get_schedule` | Proposed + confirmed time blocks for a date range (Phase 7b) |
| `create_time_block` | Schedule a task into a specific slot (Phase 7b) |
| `confirm_time_block` | Mark block confirmed; triggers calendar sync if adapter connected (Phase 7b) |

## Frontend views

- **Context board** — active contexts: title, summary excerpt, open task count, last activity
- **Context detail** — full event timeline, open tasks, linked items, Claude-maintained summary
- **Task board** — flat task list across all contexts; filter by priority/due date
- **Schedule view** — weekly calendar of proposed + confirmed time blocks
- **Transaction review** — triage queue for unassigned transactions
