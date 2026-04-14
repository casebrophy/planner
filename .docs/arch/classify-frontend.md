# Classify Frontend System

The classify domain implements automatic task-to-context assignment through an AI-driven matching workflow. A modal dialog presents a confirmation flow, triggers the classification operation via a service layer, and displays results or errors to the user. The system integrates with the task domain to manage unlinked tasks and depends on backend classification logic that generates suggestions and clarification cards for low-confidence matches.

## Core Types

```typescript
// Service types (services/classifyService.ts)
export interface ClassifyAccepted {
  message: string
  unlinkedCount: number
}

export const classifyService = {
  async classify(): Promise<ClassifyAccepted>
}
```

## File Map

### Services
- `services/classifyService.ts` — **classifyService** — Single-method classification service:
  - `classify(): Promise<ClassifyAccepted>` — POST to `/api/v1/tasks/classify`; returns confirmation message and count of unlinked tasks processed
  - Returns `ClassifyAccepted` with `message` (human-readable confirmation) and `unlinkedCount` (count of tasks that received classifications or clarification cards)

### Components
- `components/tasks/ClassifyDialog.vue` — **ClassifyDialog** — Modal dialog for triggering auto-classification
  - Props: `{ open: boolean }` — controls visibility
  - Refs: `running` (boolean) — shows during API call; disables close backdrop click; changes button label to "Classifying..."; `result` (ClassifyAccepted | null) — holds successful response; `error` (string | null) — holds error message text
  - Methods:
    - `classify()` — async; sets running=true, clears error, calls `classifyService.classify()`, updates result, catches errors into error ref, always sets running=false
    - `close()` — resets result and error, emits `close` event
  - Template states:
    - **Confirm state** (no result, no error) — shows title "Classify Unlinked Tasks", description text, Cancel and Classify buttons
    - **Running state** — Classify button disabled, shows "Classifying..." label
    - **Error state** (error ref set) — shows "Classification Failed" title, error message, Close button; triggered by catch block in classify()
    - **Results state** (result ref set) — shows "Classification Started", result.message, Done button; result is the ClassifyAccepted response from the service
  - Emits: `close()` on Cancel, Close (error), or Done (results) button click; also emitted when backdrop clicked during non-running state
  - Transitions: CSS fade animation (opacity 0.15s) on enter/leave

## Impact Callouts

### ⚠ ClassifyAccepted (`services/classifyService.ts`)
Changing the response interface affects:
- `components/tasks/ClassifyDialog.vue` — `result` ref typed as `ClassifyAccepted | null`; template binds to `result.message`; if shape changes, message binding breaks
- `services/classifyService.ts` — return type must match interface; changes cascade to all callers

### ⚠ classifyService.classify() (`services/classifyService.ts`)
Changing the method signature, endpoint, or error behavior affects:
- `components/tasks/ClassifyDialog.vue` — calls `classifyService.classify()` directly; if method returns different type or throws different errors, catch block in classify() must be updated
- Task board integration — TaskBoardView invokes ClassifyDialog; if service changes, dialog behavior may break task list auto-refresh expectations

### ⚠ ClassifyDialog.open prop (`components/tasks/ClassifyDialog.vue`)
Controlling dialog visibility affects:
- Parent components (TaskBoardView) — must pass boolean open prop; must listen to close event and update their own showClassify ref
- Modal visibility lifecycle — open=true mounts Teleport and Transition; open=false triggers fade-out and unmounts

### ⚠ ClassifyDialog state machine (`components/tasks/ClassifyDialog.vue`)
The three-state template structure (confirm, error, results) is tightly coupled:
- Initial render shows confirm state (no result, no error)
- Setting error ref shows error state (error template branch)
- Setting result ref shows results state (result template branch)
- Both running and error must be carefully managed; if logic in classify() is changed (e.g., not catching certain errors), state machine breaks

## Cross-Domain Dependencies

- **taskService / taskStore** — ClassifyDialog triggers classification that affects task→context assignments in the background; TaskBoardView watches for completion and may refresh task list after close event to pick up context assignments
- **taskDomain (Task, TaskFilter)** — classification addresses unlinked tasks (tasks with `contextId == null` or unset); the operation modifies tasks' contextId field server-side; frontend sees changes on next fetch
- **clarificationStore** — backend generates clarification cards for low-confidence matches; these appear in the clarifications panel after classification completes; users see them as "classify these tasks?" cards
- **Modal pattern** — ClassifyDialog uses Teleport to body, Transition for fade animation, fixed positioning and z-50; integrates with the global modal z-stack alongside other dialogs (TaskForm, TaskDebriefDialog, etc.)
