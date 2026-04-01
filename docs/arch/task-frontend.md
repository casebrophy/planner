# Task System (Frontend)

> Task management domain: create, filter, view, and update tasks. Tasks have status (todo/in_progress/done/cancelled), priority, energy level, optional due date, optional context assignment, and optional tags. The board view shows a filterable list; the detail view shows full task info with tag management.

## Core Types

```ts
// types/task.ts
interface Task {
  id: string
  contextId?: string
  title: string
  description: string
  status: TaskStatus        // 'todo' | 'in_progress' | 'done' | 'cancelled'
  priority: TaskPriority    // 'low' | 'medium' | 'high' | 'urgent'
  energy: TaskEnergy        // 'low' | 'medium' | 'high'
  durationMin?: number
  dueDate?: string
  scheduledAt?: string
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
}

interface TaskFilter {
  status?: TaskStatus
  priority?: TaskPriority
  contextId?: string
  startDueDate?: string
  endDueDate?: string
}
```

```ts
// types/enums.ts (task-related)
const TaskStatus = { Todo: 'todo', InProgress: 'in_progress', Done: 'done', Cancelled: 'cancelled' } as const
type TaskStatus = (typeof TaskStatus)[keyof typeof TaskStatus]

const TaskPriority = { Low: 'low', Medium: 'medium', High: 'high', Urgent: 'urgent' } as const
type TaskPriority = (typeof TaskPriority)[keyof typeof TaskPriority]

const TaskEnergy = { Low: 'low', Medium: 'medium', High: 'high' } as const
type TaskEnergy = (typeof TaskEnergy)[keyof typeof TaskEnergy]

const TaskStatusLabels: Record<TaskStatus, string>     // display labels
const TaskPriorityLabels: Record<TaskPriority, string> // display labels
const TaskEnergyLabels: Record<TaskEnergy, string>     // display labels
const StatusColors: Record<string, string>             // hex color map
const PriorityColors: Record<TaskPriority, string>     // hex color map
```

## File Map

### Stores
- `stores/taskStore.ts` — **useTaskStore** — Pinia store wrapping createCRUDStore; adds `tasksByStatus` (computed grouped by status), `hasActiveFilter`, `overdueCount`

### Services
- `services/taskService.ts` — **taskService** — createCRUDService wrapper for `/api/v1/tasks`; maps `contextId→context_id`, `startDueDate→start_due_date`, `endDueDate→end_due_date` in filter

### Composables
- `composables/useTaskBoard.ts` — **useTaskBoard** — board-level composable; wraps taskStore, wires pagination + polling, exposes setFilter/setOrder/setPage/refresh
- `composables/useTaskDetail.ts` — **useTaskDetail(taskId)** — detail composable; loads task + tags in parallel (taskStore + tagStore), exposes update/remove/addTag/removeTag

### Components
- `components/tasks/TaskCard.vue` — **TaskCard** — renders a single task in the board (status badge, priority, energy, due date)
- `components/tasks/TaskFilterBar.vue` — **TaskFilterBar** — filter UI for status, priority, contextId, date range
- `components/tasks/TaskForm.vue` — **TaskForm** — create/edit form for NewTask/UpdateTask fields

### Views
- `views/TaskBoardView.vue` — **TaskBoardView** — uses useTaskBoard; renders TaskFilterBar + TaskCard list + Pagination
- `views/TaskDetailView.vue` — **TaskDetailView** — uses useTaskDetail; renders TaskForm (edit mode), ThreadPanel, tag management

## Impact Callouts

### ⚠ Task (types/task.ts)
Changing this interface shape affects:
- `stores/taskStore.ts` — stores `Task[]` in `items`, computes `tasksByStatus` by iterating `.status`, checks `.dueDate` + `.status` for `overdueCount`
- `composables/useTaskBoard.ts` — exposes `tasks` (the items array), filters via TaskFilter
- `composables/useTaskDetail.ts` — stores current task in `currentTask`, passes to `update(taskId, UpdateTask)`
- `composables/useContextDetail.ts` — filters `taskStore.items` by `.contextId` to build `linkedTasks`
- `composables/useDashboard.ts` — reads `.status`, `.dueDate` to compute taskCounts and overdueTasks
- `composables/useToday.ts` — reads `.dueDate`, `.status` to compute overdueTasks/dueTodayTasks/inProgressTasks
- `composables/useSearch.ts` — matches on `.title` and `.description`
- `services/taskService.ts` — serializes NewTask as POST body, deserializes Task from response
- `components/tasks/TaskCard.vue` — binds all display fields (.title, .status, .priority, .energy, .dueDate)
- `components/tasks/TaskForm.vue` — binds all editable fields

### ⚠ TaskFilter (types/task.ts)
Changing filter field names affects:
- `services/taskService.ts` — mapFilter maps camelCase → snake_case query params; field renames break API query string
- `stores/taskStore.ts` — `hasActiveFilter` reads `.status`, `.priority`, `.contextId`
- `components/tasks/TaskFilterBar.vue` — emits TaskFilter object

### ⚠ TaskStatus / TaskPriority / TaskEnergy (types/enums.ts)
Adding or removing enum values affects:
- `stores/taskStore.ts` — `overdueCount` compares against `TaskStatus.Done` and `TaskStatus.Cancelled`
- `composables/useDashboard.ts` — counts by `TaskStatus.Todo`, `TaskStatus.InProgress`, `TaskStatus.Done`
- `composables/useToday.ts` — filters by `TaskStatus.Done`, `TaskStatus.Cancelled`, `TaskStatus.InProgress`
- `components/shared/StatusBadge.vue` — uses StatusColors map keyed by string value
- `components/shared/PriorityIndicator.vue` — uses PriorityColors map keyed by TaskPriority
- `components/shared/EnergyIndicator.vue` — reads TaskEnergy string values

## Cross-Domain Dependencies

- `stores/tagStore.ts` — useTaskDetail loads tags per task via tagStore; tag add/remove updates taskStore cache
- `stores/contextStore.ts` — useContextDetail reads taskStore.items to show linkedTasks; TaskFilter.contextId scopes list to one context
- `composables/usePolling.ts` — used by useTaskBoard for background refresh
- `composables/usePagination.ts` — used by useTaskBoard for page controls
- `stores/toastStore.ts` — createCRUDStore (used by taskStore) emits toasts on create/update/delete errors
- `stores/captureStore.ts` — calls taskStore.create() when submitting a new task from Capture view
