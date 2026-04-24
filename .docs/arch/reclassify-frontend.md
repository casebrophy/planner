# Reclassify Frontend System

The reclassify frontend system enables users to convert items between types (task ↔ note, task → event, note → event) when classification confidence is low or user intent changes. Conversions preserve metadata (tags, context, links) and trigger store cache invalidation to keep detail views and lists synchronized.

## Core Types

```typescript
// Service methods
export interface taskService {
  convertTaskToNote(taskId: string): Promise<Note>
}

export interface noteService {
  convertNoteToTask(noteId: string): Promise<Task>
}

// Conversion flow state (in views)
const converting = ref(false)        // Prevents double-clicks during API call
const showConvertConfirm = ref(false) // Confirmation dialog toggle
```

## File Map

### Services
- `services/taskService.ts` — **taskService.convertTaskToNote** — Calls `POST /api/v1/tasks/{taskId}/convert-to-note`; returns new Note entity
- `services/noteService.ts` — **noteService.convertNoteToTask** — Calls `POST /api/v1/notes/{noteId}/convert-to-task`; returns new Task entity

### Views
- `views/TaskDetailView.vue` — **Convert to Note action**:
  - Button at line 239-243: purple "Convert to Note" button in action bar
  - Confirmation dialog (lines 521-528): ConfirmDialog component with title "Convert to Note", message "Convert this task to a note? Tags and context will be preserved."
  - Handler `handleConvertToNote()` (lines 133-151):
    - Sets `converting.value = true` to disable button during request
    - Calls `taskService.convertTaskToNote(task.value.id)`
    - Removes old task from taskStore via `taskStore.remove(taskId)`
    - Navigates to `note-detail` route with new note ID
    - Preserves `context` query param if present in current route
    - Catches errors via try/finally, resets `converting.value`
  - `showConvertConfirm` ref gate prevents execution when false

- `views/NoteDetailView.vue` — **Convert to Task action**:
  - Button at lines 194-198: emerald "Convert to Task" button in action bar
  - Confirmation dialog (lines 416-423): ConfirmDialog component with title "Convert to Task", message "Convert this note to a task? Tags and context will be preserved."
  - Handler `handleConvertToTask()` (lines 114-132):
    - Sets `converting.value = true` to disable button during request
    - Calls `noteService.convertNoteToTask(note.value.id)`
    - Removes old note from noteStore via `noteStore.remove(noteId)`
    - Navigates to `task-detail` route with new task ID
    - Preserves `context` query param if present in current route
    - Catches errors via try/finally, resets `converting.value`
  - `showConvertConfirm` ref gate prevents execution when false

### Stores
- `stores/taskStore.ts` — Uses `remove(taskId)` to evict converted task from cache after successful conversion
- `stores/noteStore.ts` — Uses `remove(noteId)` to evict converted note from cache after successful conversion

### Tests
- `__tests__/views/TaskDetailView.reclassify.test.ts` — 5 test cases:
  - "shows convert button in the UI" — verifies button renders
  - "opens confirmation dialog when convert button is clicked" — verifies `showConvertConfirm` becomes true
  - "calls taskService.convertTaskToNote on confirm" — mocks service, verifies API call with taskId
  - "navigates to note detail on successful conversion" — checks router.push called with note route
  - "preserves context in navigation query" — verifies query param forwarded to new route
  
- `__tests__/views/NoteDetailView.reclassify.test.ts` — 5 test cases:
  - "shows convert button in the UI" — verifies button renders
  - "opens confirmation dialog when convert button is clicked" — verifies `showConvertConfirm` becomes true
  - "calls noteService.convertNoteToTask on confirm" — mocks service, verifies API call with noteId
  - "navigates to task detail on successful conversion" — checks router.push called with task route
  - "preserves context in navigation query" — verifies query param forwarded to new route

## Impact Callouts

### ⚠ Task / Note Types (`types/task.ts`, `types/note.ts`)
Adding fields affects:
- `taskService.convertTaskToNote()` response — returned Note must match current Note shape
- `noteService.convertNoteToTask()` response — returned Task must match current Task shape
- `handleConvertToNote()` / `handleConvertToTask()` handlers — if new fields require special handling, add post-conversion logic

### ⚠ taskService / noteService
Adding conversion endpoints:
- Both endpoints are called by their respective handlers; if signature changes (params, return type), update corresponding handler calls and test mocks

### ⚠ TaskDetailView / NoteDetailView
Conversion UX spans:
- Button visibility — both buttons are always visible in view mode (no conditional rendering)
- Confirmation dialog — ConfirmDialog component; message text hardcoded; if conversion logic changes, update dialog message
- Navigation — router.push to detail route; if route names change, update navigation paths
- Store cache invalidation — taskStore.remove() / noteStore.remove() called before navigation; if store API changes, update calls
- Context query preservation — if route structure changes, update navTo object construction

### ⚠ Store Cache Invalidation
Conversion requires dual cleanup:
- Old task removed from taskStore after successful `convertTaskToNote()`
- Old note removed from noteStore after successful `convertNoteToTask()`
- Without removal, detail view refetch may read stale cache; board views remain consistent because route changes trigger re-fetch

### ⚠ Navigation & Route Names
Hardcoded route names:
- TaskDetailView uses `name: 'note-detail'` when converting task
- NoteDetailView uses `name: 'task-detail'` when converting note
- If route names change, handlers must be updated; router config is the source of truth

## Cross-Domain Dependencies

- **taskStore / noteStore** — CRUD stores; `remove(id)` called after conversion succeeds to evict from cache
- **taskService / noteService** — HTTP services; `convertTaskToNote` / `convertNoteToTask` methods dispatch to backend API
- **useTaskDetail / useNoteDetail** — composables; provide `task.value` / `note.value` and `task.value.id` / `note.value.id` for conversion handler input
- **useRouter** — Vue Router instance; `router.push()` navigates after conversion with preserved query params
- **useRoute** — Vue Router route; `route.query.context` read to preserve navigation context across conversion
- **ConfirmDialog** — shared component; displays conversion confirmation with `loading` prop bound to `converting` ref

## Flow Diagram

```
User clicks "Convert to Note" button (TaskDetailView)
  ↓
showConvertConfirm = true
  ↓
User confirms in ConfirmDialog
  ↓
handleConvertToNote() runs
  ├─ converting = true (disable button)
  ├─ taskService.convertTaskToNote(taskId) → POST /api/v1/tasks/{taskId}/convert-to-note
  ├─ response: newNote { id, content, tags, context, ... }
  ├─ taskStore.remove(taskId) (cache invalidation)
  ├─ router.push({ name: 'note-detail', params: { id: newNote.id }, query: { context: route.query.context } })
  └─ finally: converting = false

Similar for NoteDetailView → Task conversion with opposite direction and names
```

## Backend API Endpoints

- `POST /api/v1/tasks/{taskId}/convert-to-note` — Returns new Note; implies old Task is deleted
- `POST /api/v1/notes/{noteId}/convert-to-task` — Returns new Task; implies old Note is deleted

Backend handles:
- Data migration (tags, context, explicit links preserved)
- Deletion of source entity
- New entity creation with migrated metadata
