# Tag System

Tags are simple name-only labels that can be attached to tasks, contexts, and notes. The store maintains three per-entity tag maps (taskTags, contextTags, noteTags) keyed by entity ID. All backend tag-by-entity endpoints return `QueryResult<Tag>` (not a plain array) — services must unwrap `.items`.

## Core Types

```typescript
// types/tag.ts
export interface Tag {
  id: string
  name: string
}

export interface NewTag {
  name: string
}
```

`QueryResult<Tag>` (from `types/query.ts`) is the response shape for all `getBy*` service calls:
```typescript
interface QueryResult<T> {
  items: T[]
  total: number
  page: number
  rowsPerPage: number
}
```

## File Map

### Services
- `services/tagService.ts` — **tagService** — CRUD + relationship methods (getByTask, addToTask, removeFromTask, getByContext, addToContext, removeFromContext, getByNote, addToNote, removeFromNote). All `getBy*` calls unwrap `QueryResult<Tag>.items`.

### Stores
- `stores/tagStore.ts` — **useTagStore** — Extends createCRUDStore with three per-entity tag maps: `taskTags: Record<string, Tag[]>`, `contextTags: Record<string, Tag[]>`, `noteTags: Record<string, Tag[]>`. Provides fetchTagsFor*, addTagTo*, removeTagFrom* for each entity type.

### Composables
- `composables/useTaskDetail.ts` — **useTaskDetail** — Reads `tagStore.taskTags[taskId]`, exposes `tags`, `addTag`, `removeTag` for task detail views.
- `composables/useContextDetail.ts` — **useContextDetail** — Reads `tagStore.contextTags[contextId]`, exposes `tags`, `addTag`, `removeTag` for context detail views.
- `composables/useNoteDetail.ts` — **useNoteDetail** — Reads `tagStore.noteTags[noteId]`, exposes `tags`, `addTag`, `removeTag` for note detail views.
- `composables/useSearch.ts` — **useSearch** — Reads `tagStore.items` for global search across tasks, contexts, and tags.

### Components
- `components/tags/TagBadge.vue` — **TagBadge** — Displays a single tag pill; optional `removable` prop emits `remove(id)`.
- `components/tags/TagList.vue` — **TagList** — Renders a list of TagBadge pills; propagates `remove` emit.
- `components/tags/TagPicker.vue` — **TagPicker** — Searchable dropdown to add existing tags or create new ones. Calls `tagStore.fetchList()` on mount. Emits `add(tagId)` and `create(name)`.

### Views
- `views/TaskDetailView.vue` — **TaskDetailView** — Uses `useTaskDetail` + direct `useTagStore` for tag creation. Renders TagList and TagPicker.
- `views/ContextDetailView.vue` — **ContextDetailView** — Uses `useContextDetail` + direct `useTagStore` for tag creation. Renders TagList and TagPicker.
- `views/NoteDetailView.vue` — **NoteDetailView** — Uses `useNoteDetail` + direct `useTagStore` for tag creation. Renders TagList and TagPicker.
- `views/SearchView.vue` — **SearchView** — Uses `useSearch`, displays filtered tags alongside tasks/contexts.

## Impact Callouts

### ⚠ Tag (types/tag.ts)
Changing this interface shape affects:
- `stores/tagStore.ts` — typed as `Tag[]` in taskTags/contextTags/noteTags maps; find() and filter() operate on Tag fields
- `services/tagService.ts` — `request<QueryResult<Tag>>` type param; `.items` unwrap returns `Tag[]`
- `composables/useTaskDetail.ts` — `tags` computed is `Tag[]`; returned to view
- `composables/useContextDetail.ts` — `tags` computed is `Tag[]`; returned to view
- `composables/useNoteDetail.ts` — `tags` computed is `Tag[]`; returned to view
- `composables/useSearch.ts` — filters `tagStore.items` by `t.name`
- `components/tags/TagBadge.vue` — prop `tag: Tag`; template binds `tag.name` and emits `tag.id` on remove
- `components/tags/TagList.vue` — prop `tags: Tag[]`; passes each to TagBadge
- `components/tags/TagPicker.vue` — filters `tagStore.items` by `t.name`; passes `t.id` on select
- `views/TaskDetailView.vue` — calls `tags.map(t => t.id)` for TagPicker `:selected-ids`
- `views/NoteDetailView.vue` — calls `tags.map(t => t.id)` for TagPicker `:selected-ids`
- `views/ContextDetailView.vue` — calls `(tags || []).map(t => t.id)` for TagPicker `:selected-ids`

### ⚠ QueryResult envelope (backend response shape)
All three backend tag-by-entity endpoints (`/api/v1/tasks/:id/tags`, `/api/v1/contexts/:id/tags`, `/api/v1/notes/:id/tags`) return `QueryResult<Tag>` not a plain array. **Every `getBy*` in tagService must unwrap `.items`.** Failing to do so causes `taskTags[id]` to hold the envelope object, breaking any `.map()` call downstream (e.g., TagPicker `:selected-ids`, useTaskDetail `tags` computed).

## Cross-Domain Dependencies

- `stores/createCRUDStore.ts` — base CRUD store factory used by useTagStore
- `services/createCRUDService.ts` — base CRUD service factory used by tagService
- `stores/taskStore.ts` — consumed by useTaskDetail alongside useTagStore
- `stores/contextStore.ts` — consumed by useContextDetail alongside useTagStore
- `stores/noteStore.ts` — consumed by useNoteDetail alongside useTagStore
- `stores/toastStore.ts` — error toasts in tagStore fetch methods
