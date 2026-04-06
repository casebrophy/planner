# Task Frontend System

The task domain manages the lifecycle of individual work items, providing a comprehensive system for creating, filtering, and tracking tasks across contexts and time periods. It integrates with contexts for organizational grouping, tags for flexible labeling, and time-blocking for scheduling. The system implements CRUD operations through a base pattern factory, maintains pagination and filtering state through Pinia stores, and exposes task data through composables that power views and components.

## Core Types

```typescript
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

export type TaskStatus = 'open' | 'blocked' | 'done' | 'dismissed'
export type TaskPriority = 'low' | 'medium' | 'high' | 'urgent'
export type TaskEnergy = 'low' | 'medium' | 'high'
```

## File Map

### Stores
- `stores/taskStore.ts` — **useTaskStore** — Pinia store wrapping CRUD operations; computeds: `tasksByStatus` (group by status), `overdueCount` (open/blocked past dueDate)

### Services
- `services/taskService.ts` — **taskService** — CRUD service for `/api/v1/tasks`:
  - `list(params)`, `getById(id)`, `create(item)`, `update(id, item)`, `delete(id)`

### Composables
- `composables/useTaskBoard.ts` — **useTaskBoard** — Manages paginated task list with filtering, sorting, polling; returns tasks, pagination, filter state, setter methods
- `composables/useTaskDetail.ts` — **useTaskDetail** — Single task loading with associated tags; provides update/remove/tag management and reload
- `composables/useToday.ts` — **useToday** — Groups tasks by due date/status (overdue, dueToday, blocked); builds contextMap from contextStore; auto-polls

### Components
- `components/tasks/TaskCard.vue` — **TaskCard** — Task summary card; props: `{ task: Task }`; shows title, description, priority, energy, dueDate, recurrenceRule indicator; emits `click`
- `components/tasks/TaskForm.vue` — **TaskForm** — Create/edit form for all task fields; props: `{ task?: Task | null, mode: 'create' | 'edit' }`; includes status (edit-only), context picker, due date, recurrence presets
- `components/tasks/TaskFilterBar.vue` — **TaskFilterBar** — Filter controls for status and priority; props: `{ filter: TaskFilter }`; emits `update:filter`
- `components/tasks/ClassifyDialog.vue` — **ClassifyDialog** — Modal for triggering AI task classification; props: `{ open: boolean }`; shows classified count and clarifications created

### Views
- `views/TaskBoardView.vue` — Route `/tasks` — Task list grid with filtering/sorting/pagination, create/edit drawers, classify button
- `views/TaskDetailView.vue` — Route `/tasks/:id` — Full task metadata, edit/delete, tag management, thread panel, recurrence parent link
- `views/TodayView.vue` — Route `/today` — Grouped dashboard: overdue / due-today / blocked sections with counts and context labels

## Impact Callouts

### ⚠ Task (`types/task.ts`)
Changing the Task interface shape affects:
- `stores/taskStore.ts` — `tasksByStatus` computed uses `status`; `overdueCount` uses `dueDate` and `status`
- `composables/useTaskBoard.ts` — rendering and filtering depend on all fields
- `composables/useTaskDetail.ts` — display and edit form bind all optional fields
- `composables/useToday.ts` — `overdueTasks`, `dueTodayTasks`, `blockedTasks` all filter on `dueDate` and `status`
- `components/tasks/TaskCard.vue` — renders `title`, `description`, `priority`, `energy`, `dueDate`, `recurrenceRule`, `status`
- `components/tasks/TaskForm.vue` — binds to `title`, `description`, `status`, `priority`, `energy`, `contextId`, `dueDate`, `recurrenceRule`
- `components/tasks/TaskFilterBar.vue` — filters on `status` and `priority`

### ⚠ TaskStatus (`types/enums.ts`)
Changing status values affects:
- `stores/taskStore.ts` — `tasksByStatus` grouping; Done/Dismissed exclusions in `overdueCount`
- `components/tasks/TaskForm.vue` — status dropdown options in edit mode
- `components/tasks/TaskFilterBar.vue` — status filter dropdown options
- `composables/useToday.ts` — Done/Dismissed exclusions, Blocked status filtering
- `components/shared/StatusBadge.vue` — renders status label and color for both task and context types

### ⚠ TaskPriority (`types/enums.ts`)
Changing priority values affects:
- `components/tasks/TaskForm.vue` — priority dropdown options
- `components/tasks/TaskFilterBar.vue` — priority filter options
- `components/shared/PriorityIndicator.vue` — renders priority label and colored dot

### ⚠ TaskEnergy (`types/enums.ts`)
Changing energy values affects:
- `components/tasks/TaskForm.vue` — energy dropdown options
- `components/shared/EnergyIndicator.vue` — renders energy bars and label

### ⚠ TaskFilter (`types/task.ts`)
Changing filter shape affects:
- `services/taskService.ts` — `mapFilter` must translate all fields to API query params
- `composables/useTaskBoard.ts` — `setFilter` passes filter to store
- `components/tasks/TaskFilterBar.vue` — emits updated filter object

### ⚠ NewTask / UpdateTask (`types/task.ts`)
Changing these shapes affects:
- `components/tasks/TaskForm.vue` — payload shape must match (create mode uses NewTask, edit mode uses UpdateTask)
- `stores/taskStore.ts` — `create()` uses NewTask, `update()` uses UpdateTask

## Cross-Domain Dependencies

- **contextStore** — TaskForm imports for context picker; useToday builds contextMap for display labels
- **tagStore** — useTaskDetail loads and manages task tags; TaskDetailView renders TagPicker and TagList
- **classifyService** — ClassifyDialog triggers AI task→context classification workflow
- **shared components** — TaskCard uses `StatusBadge`, `PriorityIndicator`, `EnergyIndicator`; TaskDetailView uses `ThreadPanel`, `StreakDisplay`, `ActivityLogButton`, `ActivityHistory`
- **usePolling** — useTaskBoard and useToday use for interval-based auto-refresh
- **usePagination** — useTaskBoard uses for totalPages/hasNext/hasPrev computed state
- **services/client** — taskService and classifyService use `request()` for API calls
