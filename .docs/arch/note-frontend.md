# Note Frontend System

The note domain provides note creation, browsing, and detail views with context association, source tracking, tag management, activity tracking, and entity linking.

## Core Types

```typescript
export interface Note {
  id: string
  contextId?: string
  content: string
  source: string
  rawInputId?: string
  createdAt: string
  updatedAt: string
}

export interface NewNote {
  content: string
  source: string
  contextId?: string
}

export interface UpdateNote {
  content?: string
  contextId?: string
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
- `stores/noteStore.ts` — **useNoteStore** — Pinia store wrapping CRUD operations for notes

### Services
- `services/noteService.ts` — **noteService** — CRUD service for `/api/v1/notes`

### Composables
- `composables/useNoteDetail.ts` — **useNoteDetail** — Single note loading with associated tags; provides update/remove/tag management

### Components
- `components/notes/NoteForm.vue` — **NoteForm** — Create/edit form for note content and context; props: `{ note?: Note | null, mode: 'create' | 'edit' }`

### Views
- `views/NotesBoardView.vue` — Route `/notes` — Note list with filtering and create drawer
- `views/NoteDetailView.vue` — Route `/notes/:id` — Full note metadata, edit/delete, tag management, activity tracking, thread panel, Related Items panel (explicit entity links via entityLinkStore)

## Impact Callouts

### ⚠ Note (`types/note.ts`)
Changing the Note interface shape affects:
- `composables/useNoteDetail.ts` — display and edit form bind all optional fields
- `components/notes/NoteForm.vue` — binds to `content`, `contextId`, `source`

### ⚠ UpdateNote / NewNote (`types/note.ts`)
Changing these shapes affects:
- `components/notes/NoteForm.vue` — payload shape must match
- `stores/noteStore.ts` — `create()` uses NewNote, `update()` uses UpdateNote

## Cross-Domain Dependencies

- **contextStore** — NoteDetailView displays context name; NoteForm includes context picker
- **tagStore** — useNoteDetail loads and manages note tags; NoteDetailView renders TagPicker and TagList
- **entityLinkStore** — NoteDetailView fetches and manages explicit entity links; provides `fetchLinks`, `getLinks`, `createLink`, `deleteLink`
- **shared components** — NoteDetailView uses `ThreadPanel`, `StreakDisplay`, `ActivityLogButton`, `ActivityHistory`, `TagList`, `TagPicker`, `ConfirmDialog`, `LoadingSpinner`
