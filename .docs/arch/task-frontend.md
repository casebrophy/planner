# Task Frontend System

The task domain manages the lifecycle of individual work items, providing a comprehensive system for creating, filtering, and tracking tasks across contexts and time periods. It integrates with contexts for organizational grouping, tags for flexible labeling, and explicit entity links for cross-referencing. The system implements CRUD operations through a factory pattern, maintains pagination and filtering state through Pinia stores, and exposes task data through composables that power views and components.

## Core Types

```typescript
// Enums (types/enums.ts)
export type TaskStatus = 'open' | 'blocked' | 'done' | 'dismissed'
export type TaskPriority = 'low' | 'medium' | 'high' | 'urgent'
export type TaskEnergy = 'low' | 'medium' | 'high'

// Main types (types/task.ts)
export interface Task {
  id: string
  contextId?: string
  title: string
  description: string
  status: TaskStatus
  priority: TaskPriority
  energy: TaskEnergy
  durationMin?: number
  dueDate?: string
  scheduledAt?: string
  blockedReason?: string
  recurrenceRule?: string
  recurrenceParentId?: string
  createdAt: string
  updatedAt: string
  completedAt?: string
}

export interface NewTask {
  title: string
  description: string
  contextId?: string
  priority: TaskPriority
  energy: TaskEnergy
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
  energy?: TaskEnergy
  durationMin?: number
  dueDate?: string
  scheduledAt?: string
  blockedReason?: string
  recurrenceRule?: string
}

export interface TaskFilter {
  status?: TaskStatus
  priority?: TaskPriority
  contextId?: string
  startDueDate?: string
  endDueDate?: string
}
```

## File Map

### Stores
- `stores/taskStore.ts` — **useTaskStore** — Pinia CRUD store built on `createCRUDStore` factory; extends with computeds:
  - `tasksByStatus` — groups all tasks by status into `Record<TaskStatus, Task[]>`
  - `hasActiveFilter` — true if any filter (status/priority/contextId) is set
  - `overdueCount` — count of open/blocked tasks with dueDate in past

### Services
- `services/taskService.ts` — **taskService** — CRUD service factory instance for `/api/v1/tasks`:
  - Maps `TaskFilter` fields to query param names (contextId → context_id, startDueDate → start_due_date, etc.)
  - Supports `list(params)`, `getById(id)`, `create(item)`, `update(id, item)`, `delete(id)`

### Composables
- `composables/useTaskBoard.ts` — **useTaskBoard** — Manages paginated task list view:
  - Returns: tasks (items), total, page, rowsPerPage, loading, error, filter, orderBy, hasActiveFilter, pagination, isEmpty
  - Methods: setFilter, setOrder, setPage, refresh
  - Auto-fetches on mount; auto-polls for updates via `usePolling`
- `composables/useTaskDetail.ts` — **useTaskDetail(taskId)** — Single task detail view with tag management:
  - Returns: task (currentTask), tags, loading, update, remove, addTag, removeTag, reload
  - Loads task via `fetchById()` and associated tags via `tagStore.fetchTagsForTask()`
- `composables/useTaskNotes.ts` — **useTaskNotes(taskId)** — Notes scoped to a specific task:
  - Accepts `taskId` as `string | Ref<string>` (resolves reactively via computed)
  - Sets `noteStore.filter = { taskId }` then calls `fetchList(true)` on mount to load server-filtered notes
  - Returns: notes (items from noteStore), loading, addNote, updateNote, deleteNote, reload
  - `addNote(data: NewNote)` delegates to `noteStore.create(data)`; caller is responsible for setting taskId on the note at the API/service layer
- `composables/useToday.ts` — **useToday** — Dashboard grouping for today's tasks:
  - Computeds: overdueTasks (dueDate < today, not done/dismissed), dueTodayTasks (dueDate today), blockedTasks (status=blocked)
  - Also provides: contextMap (id → title), counts object, loading ref
  - Auto-fetches both taskStore and contextStore on mount; auto-polls

### Components
- `components/tasks/TaskCard.vue` — **TaskCard** — Compact task summary card
  - Props: `{ task: Task }`
  - Displays: title, description, priority (via PriorityIndicator), energy (via EnergyIndicator), status (via StatusBadge), dueDate (relative format), recurrence icon, clickable context chip
  - Computed: context (via contextStore.contextById), dueLabel, isOverdue
  - Emits: `click(taskId)` on card click; navigates to `/contexts/:id` on context chip click
- `components/tasks/TaskForm.vue` — **TaskForm** — Create/edit form for all task fields
  - Props: `{ task?: Task | null, mode: 'create' | 'edit' }`
  - Fields: title, description, status (edit-only dropdown), priority, energy, context (select from contextStore.items), dueDate, recurrenceRule (presets: None, Daily, Weekly, Weekdays, Monthly)
  - Emits: `submit(NewTask | UpdateTask)`, `cancel()`
  - Validation: title required; constructs NewTask (create mode) or UpdateTask (edit mode) with ISO dueDate
- `components/tasks/TaskFilterBar.vue` — **TaskFilterBar** — Status and priority filter controls
  - Props: `{ filter: TaskFilter }`
  - Two dropdowns: status (all statuses), priority (all priorities)
  - Clear button (shown if either filter active)
  - Emits: `update(TaskFilter)` on change via watch
- `components/tasks/ClassifyDialog.vue` — **ClassifyDialog** — Modal for auto-classification workflow
  - Props: `{ open: boolean }`
  - States: confirm (initial), running, error, results
  - Calls `classifyService.classify()` to trigger AI task→context matching
  - Emits: `close()`

### Views
- `views/TaskBoardView.vue` — Route `/tasks` — Main task list view with filtering, creation, classification
  - Uses: useTaskBoard, useContextStore, useRouter, useRoute
  - Refs: showCreateForm, showClassify, groupByContext (toggle grouped/flat view)
  - Computed: groupedTasks (groups by contextId via contextStore.contextById); groups display context title/kind with color
  - Watch on groupByContext: sets rowsPerPage to 100 (grouped) or 20 (flat); triggers refresh
  - Shows: filter bar, task cards (paginated in flat mode, all in grouped mode), pagination (flat mode only), create/edit drawers, classify modal
- `views/TaskDetailView.vue` — Route `/tasks/:id` — Full task metadata and management
  - Uses: useTaskDetail, useTagStore, useEntityLinkStore
  - Displays: task form (edit mode), tags (TagList + TagPicker for add/remove), thread panel, activity log, streaks, recurrence parent link, explicit entity links
  - Loads task via useTaskDetail; fetchLinks('task', taskId) via entityLinkStore.watchEffect
  - Supports: update, remove, addTag, removeTag, addLink, deleteLink (via entityLinkStore)
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
- `components/tasks/TaskCard.vue` — renders title, description, priority, energy, status, dueDate, recurrenceRule; reads contextId
- `components/tasks/TaskForm.vue` — binds to title, description, status, priority, energy, contextId, dueDate, recurrenceRule
- `components/tasks/TaskFilterBar.vue` — filters on status and priority fields
- `views/TaskBoardView.vue` — task cards displayed throughout; grouping by contextId
- `views/TaskDetailView.vue` — full task form and entity link source entity

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

### ⚠ TaskEnergy (`types/enums.ts`)
Changing energy values or labels affects:
- `components/tasks/TaskForm.vue` — energy dropdown options in both create/edit modes
- `components/shared/EnergyIndicator.vue` — renders energy indicator (bars) with label and color

### ⚠ TaskFilter (`types/task.ts`)
Changing filter shape affects:
- `services/taskService.ts` — mapFilter function must translate all fields to API query param names (snake_case)
- `composables/useTaskBoard.ts` — setFilter updates store; filter is returned for template binding
- `components/tasks/TaskFilterBar.vue` — emits updated filter object on change
- `stores/taskStore.ts` — filter state managed via setFilter; passed to service.list()

### ⚠ NewTask / UpdateTask (`types/task.ts`)
Changing these shapes affects:
- `components/tasks/TaskForm.vue` — payload shape must match mode (NewTask for create, UpdateTask for edit)
- `stores/taskStore.ts` — create() accepts NewTask; update() accepts UpdateTask
- `composables/useTaskDetail.ts` — update(data: UpdateTask) passed to store.update()

## Cross-Domain Dependencies

- **contextStore** — TaskCard uses `contextById(contextId)` computed to resolve context title; TaskForm imports items for context picker; TaskBoardView imports ContextKindColors/ContextKindLabels for styled group headers; useToday builds contextMap for label display
- **tagStore** — useTaskDetail loads/manages task tags via `fetchTagsForTask`, `addTagToTask`, `removeTagFromTask`; TaskDetailView renders TagList/TagPicker
- **noteStore / noteService** — useTaskNotes filters noteStore by taskId via `setFilter({ taskId })` + `fetchList(true)`; delegates CRUD to noteStore.create/update/remove; NoteFilter.taskId is the server-side filter param
- **entityLinkStore** — TaskDetailView fetches explicit links via `fetchLinks('task', taskId)` in watchEffect; displays links via getLinks(); supports add/remove via `createLink` / `deleteLink`
- **classifyService** — ClassifyDialog triggers `classify()` to auto-assign unlinked tasks to contexts
- **shared components** — TaskCard uses StatusBadge, PriorityIndicator, EnergyIndicator; TaskDetailView uses ThreadPanel, StreakDisplay, ActivityLogButton, ActivityHistory
- **usePolling** — useTaskBoard and useToday use to auto-refresh list; interval-based fetching
- **usePagination** — useTaskBoard uses for totalPages/hasNext/hasPrev computed state
- **router** — TaskCard navigates to /contexts/:id on context chip click; TaskBoardView navigates on task select to /tasks/:id

