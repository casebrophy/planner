# Debrief Auto-Learn System (Frontend)

> Task completion debrief flow: when a user marks a task done, the TaskDebriefDialog prompts "How did it go?" and records the response as an observation. Skip observations are also recorded with task metadata (contextId, priority, energy) so the auto-skip heuristic can identify patterns of consistent skips across similar tasks.

## Core Types

### Observation (observationService.ts)
```typescript
interface Observation {
  id: string
  subjectType: string        // 'task', 'context', etc.
  subjectId: string          // task id, context id, etc.
  kind: string               // 'debrief', etc.
  data: Record<string, unknown>  // outcome, note, contextId, priority, energy
  source: string
  confidence: number
  weight: number
  createdAt: string
}
```

### Task (types/task.ts)
```typescript
interface Task {
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
  trackOutcome?: boolean
}
```

## File Map

### Services
- `services/observationService.ts` — **observationService** — record observations, query by subject or by kind (for auto-skip heuristic)

### Components
- `components/tasks/TaskDebriefDialog.vue` — **TaskDebriefDialog** — dialog prompted on task completion; records submit or skip observations with task metadata

### Views
- `views/TaskDetailView.vue` — **TaskDetailView** — wires TaskDebriefDialog; shows dialog when task is marked done

### Composables (indirect)
- `composables/useTaskDetail.ts` — **useTaskDetail** — manages task state; used by TaskDetailView

## Impact Callouts

### ⚠ Observation (services/observationService.ts)
Changing the `Observation` interface or `data` field shape affects:
- `components/tasks/TaskDebriefDialog.vue` — calls `observationService.record()` with `data: { outcome, note, contextId, priority, energy }`
- `views/TaskDetailView.vue` — would be affected if observation query filtering changes (future auto-skip heuristic implementation)

### ⚠ Task (types/task.ts)
Changing Task interface affects:
- `components/tasks/TaskDebriefDialog.vue` — receives full Task prop; reads `contextId`, `priority`, `energy` for observation metadata
- `views/TaskDetailView.vue` — passes Task to TaskDebriefDialog; reads `status` in `handleUpdate()`

### ⚠ TaskDebriefDialog Props
**Inputs:** `open` (boolean), `taskId` (string), `task` (Task)
**Emits:** `close`

Changing props or emit signature affects:
- `views/TaskDetailView.vue` — binds `:open="showDebrief"`, `:task-id="taskId"`, `:task="task!"`, `@close="showDebrief = false"`

## Cross-Domain Dependencies

Files outside the debrief domain that this feature depends on:
- `services/client.ts` — HTTP client for observationService API calls
- `types/task.ts` — Task type definition
- `types/enums.ts` — TaskStatus, TaskPriority, TaskEnergy enums
- `composables/useTaskDetail.ts` — provides task state and update method in TaskDetailView
- `stores/tagStore.ts`, `stores/entityLinkStore.ts` — used by TaskDetailView but not part of debrief feature
