# Correction Frontend System

> The correction domain provides a lightweight reclassification API for items created with low classifier confidence. It exposes a single service (`correctionService`) consumed by TaskDetailView and NoteDetailView to let users override the AI's initial type assignment. A correction deletes the original item and recreates it as the target type on the backend.

## Core Types

```typescript
// Internal to correctionService.ts (not exported)
interface CorrectionResult {
  id: string    // ID of the newly created item
  type: string  // type of the newly created item ('task' | 'note' | 'event')
}
```

## File Map

### Services
- `services/correctionService.ts` — **correctionService** — Posts to `POST /api/v1/corrections` with `{ item_id, item_type, new_type }`; returns `CorrectionResult { id, type }`. Uses `request()` from `services/client.ts`. No store; stateless.

## Impact Callouts

### ⚠ correctionService (services/correctionService.ts)
Changing this service's signature or endpoint path affects:
- `views/TaskDetailView.vue` — calls `correctionService.correct(task.id, 'task', 'note' | 'event')` from `handleDemote()`; guards with `correcting` ref; navigates to `notes` or `tasks` route after success
- `views/NoteDetailView.vue` — calls `correctionService.correct(note.id, 'note', 'task')` from `handlePromote()`; guards with `correcting` ref; navigates to `tasks` route after success

## Cross-Domain Dependencies

- `services/client.ts` — `request<T>()` function used for the HTTP POST call
- `views/TaskDetailView.vue` — shows unconfirmed banner when `task.unconfirmed === true`; the banner includes "Move to Note" / "Move to Event" demote buttons
- `views/NoteDetailView.vue` — shows unconfirmed banner when `note.unconfirmed === true`; the banner includes a "Move to Task" promote button
- `types/task.ts` — `Task.unconfirmed?: boolean` controls banner visibility in TaskDetailView
- `types/note.ts` — `Note.unconfirmed?: boolean` controls banner visibility in NoteDetailView
