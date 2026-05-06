# Task Frontend System

The task domain manages the lifecycle of individual work items, providing a comprehensive system for creating, filtering, and tracking tasks across contexts and time periods. It integrates with contexts for organizational grouping, tags for flexible labeling, and explicit entity links for cross-referencing. The system implements CRUD operations through a factory pattern, maintains pagination and filtering state through Pinia stores, and exposes task data through composables that power views and components.

## Core Types

```typescript
// Enums (types/enums.ts)
export type TaskStatus = 'open' | 'blocked' | 'done' | 'dismissed'
export type TaskPriority = 'low' | 'medium' | 'high' | 'urgent'

// Main types (types/task.ts)
export interface Task {
  id: string
  contextId?: string
  title: string
  description: string
  status: TaskStatus
  priority: TaskPriority
  durationMin?: number
  dueDate?: string
  scheduledAt?: string
  blockedReason?: string
  recurrenceRule?: string
  recurrenceParentId?: string
  createdAt: string
  updatedAt: string
  completedAt?: string
  trackOutcome?: boolean
  unconfirmed?: boolean
}

export interface NewTask {
  title: string
  description: string
  contextId?: string
  priority: TaskPriority
  durationMin?: number
  dueDate?: string
  recurrenceRule?: string
}

export interface UpdateTask {
  title?: string
  description?: string
  contextId?: string
  status?: TaskStatus
  priority?: TaskPriority
  durationMin?: number
  dueDate?: string
  scheduledAt?: string
  blockedReason?: string
  recurrenceRule?: string
  trackOutcome?: boolean
}

export interface TaskFilter {
  status?: TaskStatus
  excludeStatuses?: TaskStatus[]
  priority?: TaskPriority
  contextId?: string
  startDueDate?: string
  endDueDate?: string
  hasRecurrence?: boolean
}
```

## File Map

### Stores
- `stores/taskStore.ts` — **useTaskStore** — Pinia CRUD store built on `createCRUDStore` factory; extends with computeds:
  - `tasksByStatus` — groups all tasks by status into `Record<TaskStatus, Task[]>`
  - `hasActiveFilter` — true if any filter (status/priority/contextId) is set; excludeStatuses is not counted as an active filter
  - `overdueCount` — count of open/blocked tasks with dueDate in past
  - `habits` — ref holding recurring tasks fetched via `fetchHabits()`
  - `habitsLoading` — loading state for habit fetch
  - `fetchHabits()` — fetches tasks with `hasRecurrence: true` filter (excludes Done/Dismissed)
  - Default filter: `{ excludeStatuses: [TaskStatus.Done, TaskStatus.Dismissed] }` — hides completed tasks unless overridden

### Services
- `services/taskService.ts` — **taskService** — CRUD service factory instance for `/api/v1/tasks`:
  - Maps `TaskFilter` fields to query param names (contextId → context_id, startDueDate → start_due_date, excludeStatuses → exclude_status as comma-separated string, hasRecurrence → has_recurrence, etc.)
  - Supports `list(params)`, `getById(id)`, `create(item)`, `update(id, item)`, `delete(id)`
  - Also exports `deleteBatch(ids: string[])` which calls `DELETE /api/v1/tasks/batch` with `{ ids }` body for bulk deletion
  - Also exports `convertTaskToNote(taskId: string)` which calls `POST /api/v1/tasks/{taskId}/convert-to-note` and returns a `Note`; preserves tags and context on the created note

### Composables
- `composables/useTaskBoard.ts` — **useTaskBoard** — Manages paginated task list view:
  - Returns: tasks (items), total, page, rowsPerPage, loading, error, filter, orderBy, hasActiveFilter, pagination, isEmpty
  - Methods: setFilter, setOrder, setPage, refresh
  - Auto-fetches on mount; auto-polls for updates via `usePolling`
- `composables/useTaskDetail.ts` — **useTaskDetail(taskId)** — Single task detail view with tag management:
  - Returns: task (currentTask), tags, loading, update, remove, addTag, removeTag, reload
  - Loads task via `fetchById()` and associated tags via `tagStore.fetchTagsForTask()`
- `composables/useToday.ts` — **useToday** — Dashboard grouping for today's tasks:
  - Computeds: overdueTasks (dueDate < today, not done/dismissed), dueTodayTasks (dueDate today), blockedTasks (status=blocked)
  - Also provides: contextMap (id → title), counts object, loading ref
  - Auto-fetches both taskStore and contextStore on mount; auto-polls

### Components
- `components/tasks/TaskCard.vue` — **TaskCard** — Compact task summary card
  - Props: `{ task: Task }`
  - Displays: title, description, priority (via PriorityIndicator), status (colored badge with dropdown), dueDate (relative format), recurrence icon, clickable context chip
  - Computed: context (via contextStore.contextById), dueLabel, isOverdue, statusColor
  - Emits: `click(taskId)` on card click; `status-change(taskId, status)` on inline status dropdown change; navigates to `/contexts/:id` on context chip click
- `components/tasks/TaskForm.vue` — **TaskForm** — Create/edit form for all task fields
  - Props: `{ task?: Task | null, mode: 'create' | 'edit' }`
  - Fields: title, description, status (edit-only dropdown), priority, context (select from contextStore.items), dueDate, recurrenceRule (presets: None, Daily, Weekly, Weekdays, Monthly), trackOutcome checkbox (edit-only)
  - Emits: `submit(NewTask | UpdateTask)`, `cancel()`
  - Validation: title required; constructs NewTask (create mode) or UpdateTask (edit mode) with ISO dueDate; includes `trackOutcome` in edit-mode payload
  - Note: Energy field is NOT present (removed in batch)
- `components/tasks/TaskFilterBar.vue` — **TaskFilterBar** — Status and priority filter controls
  - Props: `{ filter: TaskFilter }`
  - Two dropdowns: status (all statuses), priority (all priorities)
  - "Show completed" / "Hide completed" toggle button: sets/clears `excludeStatuses`; always visible
  - Clear button (shown if status or priority filter active); resets to default excludeStatuses on clear
  - Selecting Done/Dismissed status clears excludeStatuses to allow those tasks through
  - Emits: `update(TaskFilter)` on change via watch
- `components/habits/HabitGrid.vue` — **HabitGrid** — GitHub-style habit tracking grid
  - Props: `{ habits: Task[], habitGrid: HabitGridMap, days: Date[] }`
  - Renders: table with habit names as rows, dates as columns, colored cells for completed days
  - Empty state when no habits exist
- `components/tasks/ClassifyDialog.vue` — **ClassifyDialog** — Modal for auto-classification workflow
  - Props: `{ open: boolean }`
  - States: confirm (initial), running, error, results
  - Calls `classifyService.classify()` to trigger AI task→context matching
  - Emits: `close()`

### Tests
- Test files for TaskDetailView exist and cover view rendering, user interactions, and component integration (exact counts and details depend on test implementation)

### Views
- `views/TaskBoardView.vue` — Route `/tasks` — Main task list view with filtering, creation, classification
  - Uses: useTaskBoard, useContextStore, useRouter, useRoute
  - Refs: showCreateForm, showClassify, groupByContext (toggle grouped/flat view)
  - Computed: groupedTasks (groups by contextId via contextStore.contextById); groups display context title/kind with color
  - Sets task store orderBy to `context_id` when grouped, `created_at` when flat — keeps tasks for the same context contiguous so pagination doesn't split groups arbitrarily. Initial setOrder runs synchronously in setup so the first fetch uses the right ordering; watch on groupByContext re-applies + refreshes on toggle
  - Shows: filter bar, task cards, pagination (in both grouped and flat modes when totalPages > 1), create/edit drawers, classify modal
- `views/TaskDetailView.vue` — Route `/tasks/:id` — Full task metadata and management
  - Uses: useTaskDetail, useTagStore, useEntityLinkStore, correctionService, taskService, useRelatedByContext
  - Displays: task form (edit mode), tags (TagList + TagPicker for add/remove), activity log, streaks, recurrence parent link, explicit entity links
  - Loads task via useTaskDetail; fetchLinks('task', taskId) via entityLinkStore.watchEffect
  - Supports: update, remove, addTag, removeTag, addLink, deleteLink (via entityLinkStore)
  - Unconfirmed banner: shown when `task.unconfirmed` is true; offers "Move to Note" and "Move to Event" buttons calling `correctionService.correct(id, 'task', newType)` then navigating away; guarded by `correcting` ref
  - Convert to Note button: "Convert to Note" button (always visible in view mode) opens ConfirmDialog; on confirm calls `taskService.convertTaskToNote(taskId)`, removes task from taskStore, and navigates to new note detail page, preserving context query param if present. See `.docs/arch/reclassify-frontend.md` for full reclassification workflow details
- `views/HabitsView.vue` — Route `/habits` — Habit tracking grid view
  - Uses: useTaskStore (fetchHabits, habits), useActivityLogStore (fetchHabitGrid, habitGrid)
  - Refs: dayRange (30 or 90 days), loading
  - Computed: days (array of Date objects for selected range)
  - Renders: HabitGrid component with habits, habitGrid, and days
  - Day range toggle: 30d/90d buttons with purple active state
- `views/TodayView.vue` — Route `/today` — Dashboard view grouping tasks by urgency
  - Uses: useToday for overdueTasks, dueTodayTasks, blockedTasks, contextMap, counts
  - Displays: three sections (overdue, due-today, blocked) with counts and task cards showing context labels
  - Auto-refreshes via useToday polling

## Impact Callouts

### ⚠ Task (`types/task.ts`)
Changing the Task interface shape affects:
- `stores/taskStore.ts` — tasksByStatus computed groups by `status`; overdueCount filters on `dueDate` and `status`
- `composables/useTaskBoard.ts` — all fields may be rendered in task list; filtering/sorting depends on full shape
- `composables/useTaskDetail.ts` — display form binds all optional fields for editing; currentTask returned for template
- `composables/useToday.ts` — overdueTasks/dueTodayTasks/blockedTasks all filter on `dueDate` and `status`
- `components/tasks/TaskCard.vue` — renders title, description, priority, status, dueDate, recurrenceRule; reads contextId; shows amber "Unconfirmed" badge when `unconfirmed === true`
- `components/tasks/TaskForm.vue` — binds to title, description, status, priority, contextId, dueDate, recurrenceRule, trackOutcome
- `components/tasks/TaskFilterBar.vue` — filters on status and priority fields
- `views/TaskBoardView.vue` — task cards displayed throughout; grouping by contextId
- `views/TaskDetailView.vue` — full task form, entity link source entity; `unconfirmed` drives the classification correction banner

### ⚠ TaskStatus (`types/enums.ts`)
Changing status values or labels affects:
- `stores/taskStore.ts` — tasksByStatus grouping keys; Done/Dismissed exclusions in overdueCount
- `composables/useToday.ts` — Blocked status filtering; Done/Dismissed exclusions in overdue/due-today counts
- `components/tasks/TaskForm.vue` — status dropdown options (edit mode only)
- `components/tasks/TaskFilterBar.vue` — status filter dropdown options
- `components/shared/StatusBadge.vue` — renders status label and color mapping
- `views/TaskBoardView.vue` — task status display in cards

### ⚠ TaskPriority (`types/enums.ts`)
Changing priority values or labels affects:
- `components/tasks/TaskForm.vue` — priority dropdown options in both create/edit modes
- `components/tasks/TaskFilterBar.vue` — priority filter dropdown options
- `components/shared/PriorityIndicator.vue` — renders priority indicator with label and color

### ⚠ TaskEnergy (REMOVED)
Energy field has been removed from the Task domain. The following are NO LONGER PRESENT:
- `TaskEnergy` type in `types/enums.ts`
- Energy field in Task, NewTask, UpdateTask interfaces
- Energy dropdown in `TaskForm.vue`
- `EnergyIndicator.vue` component (deleted)
- Energy display in `TaskDetailPanel.vue` and `TaskDetailView.vue`
- Energy filtering in `useToday.ts`

### ⚠ TaskFilter (`types/task.ts`)
Changing filter shape affects:
- `services/taskService.ts` — mapFilter function must translate all fields to API query param names (snake_case); excludeStatuses joins to comma-separated string for exclude_status param
- `composables/useTaskBoard.ts` — setFilter updates store; filter is returned for template binding
- `components/tasks/TaskFilterBar.vue` — emits updated filter object on change; manages excludeStatuses via toggle button
- `stores/taskStore.ts` — filter state managed via setFilter; passed to service.list(); defaultFilter sets initial excludeStatuses
- `stores/createCRUDStore.ts` — supports `defaultFilter?: Partial<TFilter>` option; initializes filter ref from defaultFilter

### ⚠ NewTask / UpdateTask (`types/task.ts`)
Changing these shapes affects:
- `components/tasks/TaskForm.vue` — payload shape must match mode (NewTask for create, UpdateTask for edit)
- `stores/taskStore.ts` — create() accepts NewTask; update() accepts UpdateTask
- `composables/useTaskDetail.ts` — update(data: UpdateTask) passed to store.update()

## Cross-Domain Dependencies

- **contextStore** — TaskCard uses `contextById(contextId)` computed to resolve context title; TaskForm imports items for context picker; TaskBoardView imports ContextKindColors/ContextKindLabels for styled group headers; useToday builds contextMap for label display
- **tagStore** — useTaskDetail loads/manages task tags via `fetchTagsForTask`, `addTagToTask`, `removeTagFromTask`; TaskDetailView renders TagList/TagPicker
- **entityLinkStore** — TaskDetailView fetches explicit links via `fetchLinks('task', taskId)` in watchEffect; displays links via getLinks(); supports add/remove via `createLink` / `deleteLink`
- **classifyService** — ClassifyDialog triggers `classify()` to auto-assign unlinked tasks to contexts
- **correctionService** — TaskDetailView calls `correct()` when user selects "Move to Note" or "Move to Event" from unconfirmed banner
- **taskService.convertTaskToNote** — TaskDetailView calls `convertTaskToNote(taskId)` to convert task to note via backend API; preserves tags and context
- **shared components** — TaskCard uses StatusBadge, PriorityIndicator; TaskDetailView uses StreakDisplay, ActivityLogButton, ActivityHistory
- **usePolling** — useTaskBoard and useToday use to auto-refresh list; interval-based fetching
- **usePagination** — useTaskBoard uses for totalPages/hasNext/hasPrev computed state
- **useRelatedByContext** — TaskDetailView uses to fetch tasks and notes in the same context for the "Related Items" panel
- **router** — TaskCard navigates to /contexts/:id on context chip click; TaskBoardView navigates on task select to /tasks/:id; TaskDetailView converts to note and navigates to /notes/:id
- **reclassification workflow** — See `.docs/arch/reclassify-frontend.md` for full details on task→note/event conversion and auto-classification correction features

