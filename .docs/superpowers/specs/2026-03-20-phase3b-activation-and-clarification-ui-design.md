# Phase 3b Activation + Clarification Queue UI

**Date:** 2026-03-20
**Status:** Approved
**Approach:** Two-track parallel (1 backend agent + 1 frontend agent in isolated worktrees)

## Summary

Complete Phase 3b backend activation so the clarification queue fills naturally, while simultaneously building the clarification queue frontend view as the centerpiece of Phase 4. Two parallel worktree agents — one per track — with no shared file overlap.

## Track 1: Backend Activation

One agent, one branch, sequential execution. All work in Go backend (`app/`, `business/`, `api/`).

### Step 1 — Resolution Dispatcher

**Files:**
- `app/domain/clarificationapp/clarificationapp.go` — add dispatch logic in `resolve` handler
- `app/domain/clarificationapp/route.go` — inject additional bus dependencies into app struct

Fill in the existing TODO placeholder in the `resolve` handler. Map `kind + answer` → side-effect using existing enum values from `clarificationkind`:

| Kind (enum value) | Side-effect |
|------|-------------|
| `context_assignment` | Update email's `context_id` to resolved value |
| `inactivity_prompt` | Update task/context based on answer (close, extend, note) |
| `ambiguous_action` | Create task from clarified action |
| `new_context` | Confirm or merge the auto-created context |
| `stale_task` | Update task status or add thread note |
| `ambiguous_deadline` | Set correct due date on task |
| `voice_reference` | Resolve ambiguous voice capture reference |
| `context_debrief` | Store outcome observation via `observationbus` |

**Dependency injection:** The `clarificationapp.app` struct currently only holds `clarificationBus`. Expand it to also hold `emailBus`, `taskBus`, `contextBus`, `observationBus`, and `rawinputBus` so the dispatcher can delegate side-effects. Update `route.go` to instantiate and inject these dependencies (following the same pattern as other app packages).

Resolution calls go through the business layer — dispatcher lives in the app layer and delegates to the appropriate bus method.

### Step 2 — Clarification Generation in Ingestion

**Files:**
- `business/domain/ingestbus/ingestbus.go` — modify pipeline steps to generate clarification items
- `business/domain/ingestbus/ingestbus.go` — add `clarificationBus` to `Business` struct and `NewBusiness` constructor
- `api/services/planner/main.go` — wire `clarificationBus` into `ingestbus.NewBusiness`
- `business/domain/ingestbus/extractor/anthropic.go` — ensure extraction response includes confidence scores

After AI extraction in the ingestion pipeline, create clarification items when:

- Context match confidence < 0.7 → `context_assignment` item with candidate contexts in metadata
- Action items are ambiguous (multiple interpretations) → `ambiguous_action` item
- A new context was auto-created → `new_context` item

**Confidence scoring:** The `Extractor` interface response must include a `ContextConfidence float64` field (0.0–1.0) for the matched context. If the current `Extractor` response struct doesn't include this, add it and update the Anthropic implementation to request a confidence score in the extraction prompt.

Requires `clarificationbus.Business` as a new dependency of `ingestbus.Business`. Update the struct, constructor, and all wiring in `main.go`.

### Step 3 — Inactivity Detection Job

**Files:** new `business/domain/inactivitybus/` package + `main.go` goroutine

Background goroutine running every 15 minutes:

1. Query tasks where `now() - last_thread_at > expected_update_days` (or default 7 days if null)
2. Query contexts with same staleness check
3. For each stale item, check if an `inactivity_prompt` clarification already exists (avoid duplicates)
4. Create `inactivity_prompt` clarification items for new stale items

**Approach:** Create a dedicated `inactivitybus` package with its own store that queries directly across tasks/contexts (avoids polluting existing bus packages). The store runs a single SQL query joining tasks/contexts with their latest thread entry timestamps.

**Staleness thresholds by priority** (per planning doc `11-feedback-loop.md`):
- Urgent: 1 day
- High: 2 days
- Medium: 5 days
- Low: 14 days
- Contexts: use `expected_update_days` column (default 7 days)

**Duplicate prevention:** Before creating, check if a pending `inactivity_prompt` clarification already exists for the same subject.

Goroutine pattern: `go func() { ticker := time.NewTicker(15 * time.Minute); for range ticker.C { ... } }()` in main.go, with graceful shutdown via the existing `signal.NotifyContext` shutdown pattern.

### Step 4 — Context Debrief Flow

**Files:** `app/domain/contextapp/contextapp.go` or `business/domain/contextbus/contextbus.go`

When a context's status transitions to `closed` (detected in the update handler):

1. Set `debrief_status` = `pending`
2. Create 3 pre-snoozed clarification items (snoozed for 24h):
   - `context_debrief` kind: "What was the outcome?"
   - `context_debrief` kind: "What was the biggest challenge?"
   - `context_debrief` kind: "What would you do differently?"
   - (4th card — "Any cost/time observations?" — deferred to Phase 5 when transaction data exists)
3. Each card's metadata includes `context_id` and `debrief_question` field

Detection: compare old status vs new status in the update path. If old != closed and new == closed, trigger debrief.

### Step 5 — Unsnooze Expired Job

**Files:** `main.go`

Background goroutine running every 5 minutes:

1. Call `clarificationbus.UnsnoozeExpired(ctx)` (method already exists)
2. This transitions snoozed items past their `snooze_until` back to `pending`

Same goroutine pattern as inactivity job. Can share the shutdown context.

### Step 6 — MCP Tools

**Files:**
- `app/domain/mcpapp/mcpapp.go` — add tool handlers and expand `app` struct
- `app/domain/mcpapp/route.go` — inject `clarificationbus`, `threadbus`, `observationbus` dependencies

**Dependency injection:** The `mcpapp.app` struct currently holds `taskBus`, `contextBus`, and `emailBus`. Expand to also hold `clarificationBus`, `threadBus`, and `observationBus`. Update `route.go` to instantiate stores and bus instances for all three new domains (follow the existing wiring pattern in that file).

New MCP tools following existing patterns:

| Tool | Method | Parameters |
|------|--------|------------|
| `get_clarification_queue` | `clarificationbus.Query` | `status` (default pending), `page`, `rows_per_page` |
| `resolve_clarification` | `clarificationbus.Resolve` | `id`, `answer` |
| `snooze_clarification` | `clarificationbus.Snooze` | `id`, `duration_hours` |
| `dismiss_clarification` | `clarificationbus.Dismiss` | `id` |
| `add_thread_entry` | `threadbus.AddEntry` | `subject_type`, `subject_id`, `kind`, `content`, `source` |
| `record_outcome` | `observationbus.Record` | `subject_type`, `subject_id`, `kind`, `content` |

Each tool: add to `tools/list` response, add case in `tools/call` handler. Follow the existing pattern for parameter parsing and error handling.

## Track 2: Frontend — Clarification Queue + Polish

One agent, one branch, sequential execution. All work in `web/src/`.

### Step 1 — Foundation

**New files:**
- `web/src/services/clarificationService.ts` — API client wrapping existing REST endpoints
- `web/src/stores/clarificationStore.ts` — Pinia store for queue state
- `web/src/types/clarification.ts` — TypeScript types and enums

Service methods: `queryQueue()`, `queryByID(id)`, `countPending()`, `resolve(id, answer)`, `snooze(id, hours)`, `dismiss(id)`

Store state: `items`, `total`, `currentIndex`, `pendingCount`, `loading`, `error`

### Step 2 — ClarificationCard Component

**New file:** `web/src/components/clarifications/ClarificationCard.vue`

Renders a single clarification item. Behavior varies by `kind`:

- `context_assignment`: Show question + context option buttons (from metadata candidates)
- `inactivity_prompt`: Show stale item info + action options (close, extend, add note)
- `ambiguous_action`: Show interpretations + pick correct one
- `new_context`: Show auto-created context + confirm/merge/dismiss
- `stale_task`: Show task info + action options (update, deprioritize, close)
- `ambiguous_deadline`: Show deadline options + date picker
- `voice_reference`: Show ambiguous reference + resolution options
- `context_debrief`: Show debrief question + free-text answer input

Common elements: kind badge, source info, age, snooze/dismiss buttons.

Emits: `resolve(answer)`, `snooze(hours)`, `dismiss()`

### Step 3 — ClarificationSession Component

**New file:** `web/src/components/clarifications/ClarificationSession.vue`

Manages the review deck:
- Fetches pending queue on mount
- Displays current ClarificationCard
- On resolve/snooze/dismiss → calls store action → advances to next card
- Progress indicator: "1 of N" + dots
- Transition animation between cards
- "All caught up" state when queue empty

### Step 4 — ClarificationView + Routing

**New files:**
- `web/src/views/ClarificationView.vue` — wraps ClarificationSession
- `web/src/composables/useClarification.ts` — composable for the view

**Modified files:**
- `web/src/router/index.ts` — add `/clarifications` route
- `web/src/components/layout/AppSidebar.vue` — add nav item with pending count badge

Sidebar badge: poll `countPending()` every 60 seconds, show amber badge when > 0.

### Step 5 — Polish Existing Views

**New files:**
- `web/src/services/threadService.ts` — wrap thread REST endpoints
- `web/src/services/observationService.ts` — wrap observation REST endpoints
- `web/src/components/shared/ThreadPanel.vue` — reusable thread/activity timeline component

**Modified files:**
- `web/src/views/TaskDetailView.vue` — embed ThreadPanel (subject_type="task")
- `web/src/views/ContextDetailView.vue` — embed ThreadPanel (subject_type="context") + observations list

ThreadPanel component: accepts `subjectType` and `subjectId` props, fetches and displays a chronological list of thread entries. Uses existing `GET /api/v1/threads/{subject_type}/{subject_id}`.

Observation display: list of observations on detail views. Uses existing `GET /api/v1/observations/{subject_type}/{subject_id}`.

Empty states: add EmptyState usage to TaskBoardView, ContextBoardView, DashboardView when no data loaded.

## Parallel Execution Strategy

```
┌─────────────────────────────────┐  ┌─────────────────────────────────┐
│  Backend Agent (worktree)       │  │  Frontend Agent (worktree)      │
│                                 │  │                                 │
│  1. Resolution dispatcher       │  │  1. Service + store + types     │
│  2. Clarification in ingestion  │  │  2. ClarificationCard           │
│  3. Inactivity detection job    │  │  3. ClarificationSession        │
│  4. Context debrief flow        │  │  4. View + routing + sidebar    │
│  5. Unsnooze expired job        │  │  5. Polish (threads, empty      │
│  6. MCP tools                   │  │     states, observations)       │
│                                 │  │                                 │
│  Merge → main                   │  │  Merge → main                   │
└─────────────────────────────────┘  └─────────────────────────────────┘
```

No file overlap between tracks. Backend touches `app/`, `business/`, `api/`. Frontend touches `web/src/`. Safe to merge independently.

## Review Checkpoints

After each agent completes:
- Backend: `make lint && make test` must pass
- Frontend: `npm run lint && npm run build` must pass
- Code review agent validates against this spec
- Merge to main one branch at a time (backend first recommended, since frontend can be tested against mock data)

## What's Explicitly Out of Scope

- Mobile shell (Phase 4b)
- Search view, Settings view (lower priority)
- SMTP receiver changes (already implemented)
- Transaction ingestion (Phase 5)
- Semantic search / ML service
- Frontend tests (can be added later)
