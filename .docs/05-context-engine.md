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

### Inputs

| Input | Required | Degraded if missing |
|---|---|---|
| Tasks with `duration_min` + `priority` | Yes | — |
| Calendar availability (adapter) | No | Can order tasks, can't propose time slots |
| Deadlines (`due_date`) | No | Urgency sort degrades |
| Energy level per task | No | Morning-slot optimisation skipped |

### Algorithm

1. Sort tasks: urgency (due date approaching) → priority → energy (high-energy first, morning slots)
2. Fit tasks into available calendar slots respecting `duration_min`
3. Insert 15-min buffer between tasks (configurable in settings)
4. Produce `time_blocks` with `confirmed = 0` (proposed)

User reviews and confirms; confirmed blocks optionally sync to calendar.

### Auto-schedule from email

When pipeline extracts a deadline from email:
1. Creates task with implied action + `due_date`
2. Flags for scheduling, notifies user
3. Does **not** auto-schedule without confirmation

## MCP tools

| Tool | Purpose |
|---|---|
| `create_context` | New ongoing context (pipeline or user) |
| `get_context` | Summary + open tasks + recent events for a context |
| `list_contexts` | All active contexts with titles + summaries |
| `update_context` | Rename, re-describe, or close a context |
| `get_schedule` | Proposed + confirmed time blocks for a date range |
| `create_time_block` | Schedule a task into a specific slot |
| `confirm_time_block` | Mark block confirmed; triggers calendar sync if adapter connected |

## Frontend views

- **Context board** — active contexts: title, summary excerpt, open task count, last activity
- **Context detail** — full event timeline, open tasks, linked items, Claude-maintained summary
- **Task board** — flat task list across all contexts; filter by priority/due date
- **Schedule view** — weekly calendar of proposed + confirmed time blocks
- **Transaction review** — triage queue for unassigned transactions
