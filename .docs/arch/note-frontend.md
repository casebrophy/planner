# Note Frontend System

The note domain provides note creation, browsing, detail views, and task-scoped note management with context association, source tracking, tag management, and entity linking.

## Core Types

```typescript
export interface Note {
  id: string
  contextId?: string
  taskId?: string
  content: string
  source: string
  rawInputId?: string
  createdAt: string
  updatedAt: string
  unconfirmed?: boolean
}

export interface NewNote {
  contextId?: string
  taskId?: string
  content: string
  source: string
}

export interface UpdateNote {
  contextId?: string
  taskId?: string
  content?: string
  source?: string
}

export interface NoteFilter {
  contextId?: string
  taskId?: string
  source?: string
  search?: string
}
```

## File Map

### Stores
- `stores/noteStore.ts` — **useNoteStore** — Pinia store wrapping CRUD operations for notes via createCRUDStore; exposes `hasActiveFilter` computed (checks contextId, source, search)

### Services
- `services/noteService.ts` — **noteService** — CRUD service for `/api/v1/notes`; maps filter fields `contextId` → `context_id`, `taskId` → `task_id`; includes `convertToTask(noteId: string, taskData: NewTask)` method for converting notes to tasks

### Composables
- `composables/useNoteBoard.ts` — **useNoteBoard** — Board-level logic: fetches note list on mount, polls for updates, wraps create/setFilter/setOrder/setPage/refresh; returns `notes`, `pagination`, `isEmpty`, `hasActiveFilter`
- `composables/useNoteDetail.ts` — **useNoteDetail(noteId)** — Single note loading with associated tags; provides update/remove/addTag/removeTag; fetches both note and tags on mount
- `composables/useTaskNotes.ts` — **useTaskNotes(taskId)** — Task-scoped note list; sets `NoteFilter.taskId` on store, fetches on mount; provides addNote/updateNote/deleteNote/reload

### Components
- `components/notes/NoteCard.vue` — **NoteCard** — Clickable card showing note content (line-clamp-3), source badge with color coding, relative time via date-fns; props: `{ note: Note }`; emits `click(id: string)`; shows amber "Unconfirmed" badge when `note.unconfirmed === true`
- `components/notes/NoteFilterBar.vue` — **NoteFilterBar** — Filter controls for search text, source dropdown, context dropdown; props: `{ filter: NoteFilter }`; emits `update(filter: NoteFilter)`; fetches context list on mount
- `components/notes/NoteForm.vue` — **NoteForm** — Create/edit form for note content, source, and context; props: `{ note?: Note | null, mode: 'create' | 'edit' }`; emits `submit(data: NewNote | UpdateNote)` and `cancel()`
- `components/notes/NoteList.vue` — **NoteList** — Compact list rendering with content truncated to 100 chars, source label, formatted date, edit (✎) and delete (×) action buttons; props: `{ notes: Note[], loading?: boolean }`; emits `edit(note: Note)` and `delete(note: Note)`; shows empty state when notes.length === 0
- `components/shared/NoteItem.vue` — **NoteItem** — Thread-entry display for activity/thread contexts; uses local `ThreadEntry` interface (not `Note`); shows source icon (user/claude/system/email/default), content, timestamp, sentiment, requiresAction badge

### Views
- `views/NotesBoardView.vue` — Route `/notes` — Note grid using `useNoteBoard`; NoteCard → open detail drawer; NoteFilterBar; create drawer with NoteForm; nested route renders NoteDetailView in second drawer
- `views/NoteDetailView.vue` — Route `/notes/:id` — Full note metadata view using `useNoteDetail`; edit/delete with ConfirmDialog; TagPicker/TagList; ActivityLogButton/StreakDisplay/ActivityHistory; ThreadPanel; Related Items panel (explicit entity links via entityLinkStore); uses correctionService
  - Unconfirmed banner: shown when `note.unconfirmed` is true; offers "Move to Task" button calling `correctionService.correct(id, 'note', 'task')` then navigating; guarded by `correcting` ref
  - Convert to task action: "Convert to task" button triggers `convertToTask()` method with confirm dialog; calls `noteService.convertToTask()` to create task from note content and navigate to new task detail

## Impact Callouts

### ⚠ Note (`types/note.ts`)
Changing the Note interface shape affects:
- `composables/useNoteDetail.ts` — exposes `note` ref; update/remove/tag ops use noteId; all fields bound in NoteDetailView template
- `composables/useNoteBoard.ts` — `notes` is `items` from store (Note[])
- `composables/useTaskNotes.ts` — `notes` computed from store items (Note[])
- `components/notes/NoteCard.vue` — renders `note.content` (line-clamp), `note.source` (badge + color), `note.createdAt` (relative time), emits `note.id`; `unconfirmed` drives amber badge display
- `components/notes/NoteForm.vue` — binds to `note.content`, `note.contextId`, `note.source` for edit pre-fill
- `components/notes/NoteList.vue` — renders `note.content` (truncated), `note.source`, `note.createdAt` (formatted), emits full Note on edit/delete
- `views/NoteDetailView.vue` — reads `note.id`, `note.contextId`, `note.source`, `note.content`, `note.createdAt`, `note.updatedAt`; `unconfirmed` drives the classification correction banner

### ⚠ NoteFilter (`types/note.ts`)
Changing the filter interface affects:
- `services/noteService.ts` — maps `contextId` → `context_id`, `taskId` → `task_id`, `source`, `search` as query parameters
- `stores/noteStore.ts` — `filter.value` exposes current filter; `hasActiveFilter` checks `contextId`, `source`, `search` (NOT `taskId`)
- `composables/useNoteBoard.ts` — `setFilter(f: NoteFilter)` passed directly to store
- `composables/useTaskNotes.ts` — sets `{ taskId: resolvedTaskId.value }` on store filter
- `components/notes/NoteFilterBar.vue` — local refs mirror `filter.source`, `filter.search`, `filter.contextId`; emits merged NoteFilter on change

### ⚠ TaskId Field (Note, NewNote, UpdateNote)
Adding/removing taskId affects:
- `composables/useTaskNotes.ts` — task-scoped note operations now distinguish notes by taskId
- `components/notes/NoteForm.vue` — task-scoped form must optionally bind `taskId` prop
- `views/NoteDetailView.vue` — may display task linkage if taskId is set
- `services/noteService.ts` — `taskId` serialized in request bodies and query params
- `composables/useTaskNotes.ts` — creates notes with taskId when in task context

### ⚠ UpdateNote / NewNote (`types/note.ts`)
Changing these shapes affects:
- `components/notes/NoteForm.vue` — builds and emits `NewNote` (create) or `UpdateNote` (edit); contextId conditionally appended
- `stores/noteStore.ts` — `create()` uses `NewNote`, `update()` uses `UpdateNote`
- `composables/useNoteBoard.ts` — `create(data: NewNote)` delegates to store then refetches
- `composables/useNoteDetail.ts` — `update(data: UpdateNote)` delegates to store
- `composables/useTaskNotes.ts` — `addNote(data: NewNote)` and `updateNote(noteId, data: UpdateNote)` delegate to store

### ⚠ NoteDetailView Convert to Task Action
The "Convert to task" feature (`convertToTask()` method, confirm dialog, button in template) affects:
- `services/noteService.ts` — `convertToTask(noteId, taskData)` implements API call to create task
- `__tests__/views/NoteDetailView.reclassify.test.ts` — test file covering convert action, dialog confirmation, task creation, and navigation

## Cross-Domain Dependencies

- **contextStore** — NoteDetailView reads context name via `contextStore.items`; NoteFilterBar and NoteForm fetch and list contexts for pickers
- **tagStore** — useNoteDetail calls `tagStore.fetchTagsForNote`, `addTagToNote`, `removeTagFromNote`; NoteDetailView renders TagPicker and TagList; `noteStore.noteTags[noteId]` provides note-specific tags
- **entityLinkStore** — NoteDetailView fetches and manages explicit entity links; provides `fetchLinks`, `getLinks`, `createLink`, `deleteLink`
- **taskService** — NoteDetailView's `convertToTask()` calls `noteService.convertToTask()`, which creates a task via taskService
- **reclassification frontend** — see `.docs/arch/reclassify-frontend.md` for full cross-feature documentation on note-to-task conversion interaction with the reclassification system
- **usePolling** — useNoteBoard wires up polling for background refresh
- **usePagination** — useNoteBoard wraps pagination state (page, rowsPerPage, total) for NotesBoardView
- **createCRUDStore / createCRUDService** — noteStore and noteService delegate all CRUD + pagination + filter state to these shared factories
- **shared components** — NoteDetailView uses `ThreadPanel`, `StreakDisplay`, `ActivityLogButton`, `ActivityHistory`, `TagList`, `TagPicker`, `ConfirmDialog`, `LoadingSpinner`; NotesBoardView uses `PageHeader`, `DrawerPanel`, `LoadingSpinner`, `EmptyState`, `Pagination`
