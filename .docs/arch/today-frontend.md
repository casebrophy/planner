# Today System (Frontend)

> The Today view provides a focused daily dashboard showing three task categories: overdue tasks (past due date, not done/dismissed), tasks due today, and blocked tasks. Data is fetched from `taskStore` and `contextStore` on mount and refreshed via a 60-second visibility-aware polling interval. Context titles are mapped client-side for display beneath each task card.

---

## Core Types

### Task (`types/task.ts`)

```ts
interface Task {
  id: string
  contextId?: string        // FK to Context — used by TodayView for contextMap lookup
  title: string
  description: string
  status: TaskStatus        // filters: Done/Dismissed excluded from overdue; Blocked = blocked list
  priority: TaskPriority    // displayed via PriorityIndicator
  energy: TaskEnergy        // displayed via EnergyIndicator
  durationMin?: number
  dueDate?: string          // ISO string — primary filter for overdue/dueToday buckets
  scheduledAt?: string
  blockedReason?: string
  recurrenceRule?: string   // renders recurring icon in TaskCard
  recurrenceParentId?: string
  createdAt: string
  updatedAt: string
  completedAt?: string
}

interface NewTask {
  title: string
  description: string
  contextId?: string
  priority: TaskPriority
  energy: TaskEnergy
  durationMin?: number
  dueDate?: string
  recurrenceRule?: string
}

interface UpdateTask {
  title?: string
  description?: string
  contextId?: string
  status?: TaskStatus
  priority?: TaskPriority
  energy?: TaskEnergy
  durationMin?: number
  dueDate?: string
  scheduledAt?: string
  blockedReason?: string
  recurrenceRule?: string
}

interface TaskFilter {
  status?: TaskStatus
  priority?: TaskPriority
  contextId?: string
  startDueDate?: string
  endDueDate?: string
}
```

### Context (`types/context.ts`)

```ts
interface Context {
  id: string
  title: string             // used in contextMap: id → title for display in TodayView
  description: string
  kind: ContextKind
  status: ContextStatus
  summary: string
  lastEvent?: string
  createdAt: string
  updatedAt: string
}

interface NewContext { title: string; description: string; kind?: ContextKind }
interface UpdateContext { title?: string; description?: string; kind?: ContextKind; status?: ContextStatus; summary?: string }
interface ContextFilter { status?: ContextStatus; kind?: ContextKind; title?: string }
```

### TaskStatus / TaskPriority / TaskEnergy (`types/enums.ts`)

```ts
const TaskStatus = { Open: 'open', Blocked: 'blocked', Done: 'done', Dismissed: 'dismissed' } as const
type TaskStatus = (typeof TaskStatus)[keyof typeof TaskStatus]

const TaskPriority = { Low: 'low', Medium: 'medium', High: 'high', Urgent: 'urgent' } as const
type TaskPriority = (typeof TaskPriority)[keyof typeof TaskPriority]

const TaskEnergy = { Low: 'low', Medium: 'medium', High: 'high' } as const
type TaskEnergy = (typeof TaskEnergy)[keyof typeof TaskEnergy]

// Color maps used by StatusBadge, PriorityIndicator
const StatusColors: Record<string, string>       // open/blocked/done/dismissed/active/paused/closed
const PriorityColors: Record<TaskPriority, string>
```

---

## File Map

### Views
- `views/TodayView.vue` — **TodayView** — three-section layout (Overdue, Due Today, Blocked); destructures `useToday()`; routes to `task-detail` on card click; shows `contextMap[task.contextId]` label beneath each card

### Composables
- `composables/useToday.ts` — **useToday** — primary data layer; computes `overdueTasks`, `dueTodayTasks`, `blockedTasks`, `contextMap`, `counts`; calls `taskStore.fetchList(true)` + `contextStore.fetchList(true)` on mount; delegates polling to `usePolling`
- `composables/usePolling.ts` — **usePolling** — generic 60s interval poller; pauses when `document.hidden`, triggers immediate refetch on tab re-focus; used by `useToday`

### Stores (consumed, not owned by today)
- `stores/taskStore.ts` — **useTaskStore** — `items: Task[]` reactive list; `fetchList(force?)` fetches all tasks
- `stores/contextStore.ts` — **useContextStore** — `items: Context[]` reactive list; `fetchList(force?)` fetches all contexts

### Components (consumed by TodayView)
- `components/tasks/TaskCard.vue` — **TaskCard** — renders task title, description, status badge, priority, energy, due label (relative via `date-fns`), recurring icon; emits `click(id)`
- `components/layout/PageHeader.vue` — **PageHeader** — title/subtitle header with `#actions` slot (used for Refresh button)
- `components/shared/LoadingSpinner.vue` — **LoadingSpinner** — shown while `loading` ref is true
- `components/shared/EmptyState.vue` — **EmptyState** — shown when all three counts are 0
- `components/shared/StatusBadge.vue` — **StatusBadge** — renders `task.status` with color; used by TaskCard
- `components/shared/PriorityIndicator.vue` — **PriorityIndicator** — renders `task.priority`; used by TaskCard
- `components/shared/EnergyIndicator.vue` — **EnergyIndicator** — renders `task.energy`; used by TaskCard

### Tests
- `__tests__/composables/useToday.test.ts` — unit tests for `useToday`: overdue/dueToday/blocked filtering, context mapping, load(), cancelled task exclusion
- `__tests__/views/TodayView.test.ts` — component tests for TodayView: loading spinner, empty state, section rendering

---

## Impact Callouts

### ⚠ Task (`types/task.ts`)
Changing this interface shape affects:

- `composables/useToday.ts` — filters on `t.dueDate` (overdue/dueToday buckets), `t.status` (Done/Dismissed exclusion, Blocked list)
- `views/TodayView.vue` — reads `task.contextId` for contextMap lookup; iterates overdueTasks/dueTodayTasks/blockedTasks arrays
- `components/tasks/TaskCard.vue` — binds `task.title`, `task.description`, `task.dueDate`, `task.status`, `task.priority`, `task.energy`, `task.recurrenceRule`
- `stores/taskStore.ts` — `items: Task[]` — store items type
- `__tests__/composables/useToday.test.ts` — `makeTask()` factories depend on Task shape for filter assertions
- `__tests__/views/TodayView.test.ts` — renders tasks via makeTask factory

### ⚠ TaskStatus (`types/enums.ts`)
Changing values or adding/removing members affects:

- `composables/useToday.ts` — explicit checks against `TaskStatus.Done`, `TaskStatus.Dismissed` (overdue filter), `TaskStatus.Blocked` (blocked filter)
- `components/tasks/TaskCard.vue` — `isOverdue` computed checks `status !== 'done' && status !== 'dismissed'` (literal strings, not enum — **risk: enum drift**)
- `components/shared/StatusBadge.vue` — maps status string to color via `StatusColors`

### ⚠ Context (`types/context.ts`)
Changing this interface shape affects:

- `composables/useToday.ts` — iterates `contexts.value`, reads `ctx.id` and `ctx.title` to build `contextMap`
- `views/TodayView.vue` — looks up `contextMap[task.contextId]` using Context `id` as key and `title` as display value
- `stores/contextStore.ts` — `items: Context[]`

### ⚠ useToday return shape
Changing the composable's return object affects:

- `views/TodayView.vue` — destructures `{ loading, overdueTasks, dueTodayTasks, blockedTasks, contextMap, counts, refresh }` — all seven must stay present
- `__tests__/composables/useToday.test.ts` — accesses all returned values directly

---

## Cross-Domain Dependencies

| Dependency | Used by | How |
|---|---|---|
| `stores/taskStore.ts` | `useToday` | `fetchList(true)`, `items` via `storeToRefs` |
| `stores/contextStore.ts` | `useToday` | `fetchList(true)`, `items` via `storeToRefs` |
| `services/taskService.ts` | taskStore (indirect) | underlying HTTP calls for task CRUD |
| `services/contextService.ts` | contextStore (indirect) | underlying HTTP calls for context CRUD |
| `vue-router` | TodayView | `router.push({ name: 'task-detail', params: { id } })` on card click |
| `date-fns` | TaskCard | `formatDistanceToNow` for relative due date label |
| `pinia/storeToRefs` | useToday | reactive destructuring of store item refs |

---

## Filtering Logic Reference

```
overdueTasks  = tasks where dueDate < startOfToday AND status NOT IN (Done, Dismissed)
dueTodayTasks = tasks where dueDate IN [startOfToday, endOfToday]
blockedTasks  = tasks where status === Blocked  (no date filter — any blocked task appears)
```

Note: A task can appear in **both** `dueTodayTasks` and `blockedTasks` if it is due today and blocked. The view renders each section independently with no deduplication.

---

## Known Risks

- **TaskCard uses literal status strings** (`'done'`, `'dismissed'`) rather than the `TaskStatus` enum in its `isOverdue` computed. If enum values change, `TaskCard` will silently break while `useToday` correctly uses the enum.
- **No deduplication across sections** — a task due today with `Blocked` status appears in both the Due Today and Blocked sections simultaneously.
