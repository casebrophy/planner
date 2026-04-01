# Capture System (Frontend)

> Quick-entry domain: create a new task or context from a single unified form. The user toggles between 'task' and 'context' mode. On submit, the item is created and the router navigates to the new item's detail view. Capture does not have its own list or filter — it delegates creation directly to taskStore and contextStore.

## Core Types

```ts
// stores/captureStore.ts (local type)
type CaptureMode = 'task' | 'context'
```

Types used from other domains:
- `NewTask` (types/task.ts) — submitted when mode = 'task'
- `NewContext` (types/context.ts) — submitted when mode = 'context'

## File Map

### Stores
- `stores/captureStore.ts` — **useCaptureStore** — mode ref (CaptureMode), submitting flag; `submitTask(task: NewTask)` delegates to taskStore.create; `submitContext(ctx: NewContext)` delegates to contextStore.create; `defaultTask()` and `defaultContext()` return fresh form objects with sensible defaults (priority: medium, energy: medium)

### Composables
- `composables/useCapture.ts` — **useCapture** — manages local form refs (taskForm, contextForm); validates (title must be non-empty); `submit()` calls the appropriate store method then navigates to the created item's detail route; `reset()` reinitializes both forms to defaults

### Views
- `views/CaptureView.vue` — **CaptureView** — uses useCapture; renders mode toggle tabs + TaskForm (when mode=task) or ContextForm (when mode=context); submit button calls useCapture.submit()

## Impact Callouts

### ⚠ NewTask / NewContext (types/task.ts, types/context.ts)
These are the submission payload types — changing required/optional fields affects:
- `stores/captureStore.ts` — `defaultTask()` must include all required NewTask fields; `defaultContext()` must include all required NewContext fields
- `composables/useCapture.ts` — `isValid` only checks title; other required fields must have defaults
- `views/CaptureView.vue` — form must collect all required fields before submission is valid

### ⚠ CaptureMode (stores/captureStore.ts)
Adding a third mode (e.g. 'note') affects:
- `composables/useCapture.ts` — `isValid` and `submit()` switch on mode value
- `views/CaptureView.vue` — mode toggle tabs and conditional form rendering

## Cross-Domain Dependencies

- `stores/taskStore.ts` — captureStore.submitTask delegates to taskStore.create; on success, item appears in task list
- `stores/contextStore.ts` — captureStore.submitContext delegates to contextStore.create; on success, item appears in context list
- `components/tasks/TaskForm.vue` — rendered in task mode
- `components/contexts/ContextForm.vue` — rendered in context mode
- `router/index.ts` — useCapture calls router.push to 'task-detail' or 'context-detail' after successful creation
