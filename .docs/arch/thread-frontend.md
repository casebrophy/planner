# Thread System (Activity & History)

> Displays a time-ordered activity log (thread) for any entity — task, context, or note. Entries show user actions, Claude updates, system events, and external inputs (email, etc.). The thread is passive display-only; no mutations or stores. Each entry tracks source, kind, sentiment, and whether action is required.

## Core Types

### ThreadEntry (web/src/services/threadService.ts)

```typescript
export interface ThreadEntry {
  id: string
  subjectType: string          // "task" | "context" | "note" (polymorphic subject)
  subjectId: string            // UUID of the subject entity
  kind: string                 // "note" | "status_change" | "update" | "decision" | etc.
  content: string              // Human-readable entry text
  source: string               // "user" | "claude" | "system" | "email"
  sentiment?: string           // Optional sentiment label (e.g., "positive", "negative")
  requiresAction: boolean      // Whether this entry flagged as requiring follow-up
  sourceId?: string            // Optional external ID (email message ID, etc.)
  metadata?: Record<string, unknown>  // Extensible data for specialized renderers
  createdAt: string            // ISO 8601 timestamp
}
```

## File Map

### Services
- `web/src/services/threadService.ts` — **threadService** — Queries thread entries by subject (task/context/note). Single method `queryBySubject(subjectType, subjectId)` fetches paginated entries from `/api/v1/threads/{subjectType}/{subjectId}`.

### Components
- `web/src/components/shared/ThreadPanel.vue` — **ThreadPanel** — Container component that loads and renders all entries for a subject. Shows loading state, empty state, and scrollable list of entries. Imports from `threadService`.
- `web/src/components/shared/NoteItem.vue` — **NoteItem** — Displays a single `ThreadEntry` with source icon (user/claude/system/email), timestamp, sentiment badge, and action flag. Renders entry content with proper whitespace handling.

## Impact Callouts

### ⚠ ThreadEntry (web/src/services/threadService.ts)
Changing this interface shape affects:
- `web/src/components/shared/ThreadPanel.vue` — iterates over `ThreadEntry[]`, accesses `.id` (key), `.kind`, `.source`, `.createdAt`, `.content`, `.requiresAction` in template
- `web/src/components/shared/NoteItem.vue` — destructures via `defineProps<{entry: ThreadEntry}>`, renders `.source`, `.content`, `.createdAt`, `.requiresAction`, `.sentiment` in template
- `web/src/views/TaskDetailView.vue` — imports ThreadPanel, passes `subject-type="task"` and `:subject-id="taskId"`
- `web/src/views/ContextDetailView.vue` — imports ThreadPanel, passes `subject-type="context"` and `:subject-id="contextId"`
- `web/src/views/NoteDetailView.vue` — imports ThreadPanel, passes `subject-type="note"` and `:subject-id="noteId"`

### ⚠ threadService.queryBySubject() (web/src/services/threadService.ts)
Changing signature or response shape affects:
- `web/src/components/shared/ThreadPanel.vue` — calls `threadService.queryBySubject(props.subjectType, props.subjectId)` in `load()`, expects `ThreadEntry[]` result

## Cross-Domain Dependencies

**No Pinia stores.** Thread is display-only with no state mutation.

**Uses shared components:**
- `components/shared/LoadingSpinner.vue` — shown while fetching entries
- `date-fns` — `formatDistanceToNow()` for relative timestamps in both ThreadPanel and NoteItem

**Consumed by:**
- **task domain** — TaskDetailView renders `<ThreadPanel subject-type="task" :subject-id="taskId" />`
- **context domain** — ContextDetailView renders `<ThreadPanel subject-type="context" :subject-id="contextId" />`
- **note domain** — NoteDetailView renders `<ThreadPanel subject-type="note" :subject-id="noteId" />`

## Notes

- **Type duplication:** NoteItem.vue redefines `ThreadEntry` locally instead of importing from `threadService`. Consider consolidating to import from service to keep types in sync.
- **Kind icons and source colors:** ThreadPanel defines hardcoded mappings for `kind` (note→N, status_change→S, update→U, decision→D) and `source` (user→blue, claude→purple, system→gray, email→amber). Adding new kinds/sources requires template updates in both ThreadPanel and NoteItem.
- **No mutations:** Thread is read-only. Add endpoints and UI for mutations (update kind, remove entry, etc.) when needed.
