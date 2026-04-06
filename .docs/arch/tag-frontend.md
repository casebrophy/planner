# Tag Frontend System

The tag system provides comprehensive CRUD operations for creating, managing, and associating tags with tasks, contexts, and notes. It implements a reusable CRUD pattern across stores and services, with a picker component for tag selection and badge display components.

## Core Types

```typescript
export interface Tag {
  id: string
  name: string
}

export interface NewTag {
  name: string
}

// From types/query.ts — used by tagService for association endpoints
export interface QueryResult<T> {
  items: T[]
  total: number
  page: number
  rowsPerPage: number
}
```

## File Map

### Types
- `types/tag.ts` — **Tag, NewTag** — Core tag domain interfaces

### Services
- `services/tagService.ts` — **tagService** — Tag CRUD operations plus relationship management:
  - CRUD: `list()`, `getById(id)`, `create(item)`, `update(id, item)`, `delete(id)`
  - Task associations: `getByTask(taskId): Promise<Tag[]>` — fetches `QueryResult<Tag>` from `/api/v1/tasks/:id/tags`, returns `.items`; `addToTask(taskId, tagId)`, `removeFromTask(taskId, tagId)`
  - Context associations: `getByContext(contextId): Promise<Tag[]>` — fetches `QueryResult<Tag>` from `/api/v1/contexts/:id/tags`, returns `.items`; `addToContext(contextId, tagId)`, `removeFromContext(contextId, tagId)`
  - Note associations: `getByNote(noteId): Promise<Tag[]>` — fetches `QueryResult<Tag>` from `/api/v1/notes/:id/tags`, returns `.items`; `addToNote(noteId, tagId)`, `removeFromNote(noteId, tagId)`
  - **Note:** `getByTask`, `getByContext`, `getByNote` unwrap paginated `QueryResult<Tag>` — the API returns `{ items, total, page, rowsPerPage }`, not a bare array. Do not change the response typing back to `Tag[]`.

### Stores
- `stores/tagStore.ts` — **useTagStore** — Pinia store extending CRUD with relationship caches:
  - State: `taskTags: Record<string, Tag[]>`, `contextTags: Record<string, Tag[]>`, `noteTags: Record<string, Tag[]>`
  - Methods: `fetchTagsForTask`, `addTagToTask`, `removeTagFromTask`, `fetchTagsForContext`, `addTagToContext`, `removeTagFromContext`, `fetchTagsForNote`, `addTagToNote`, `removeTagFromNote`

### Components
- `components/tags/TagBadge.vue` — **TagBadge** — Displays individual tag with optional remove button; props: `tag: Tag`, `removable: boolean`; emits: `remove(id)`
- `components/tags/TagList.vue` — **TagList** — Renders flex-wrapped collection of TagBadge components with empty state; props: `tags: Tag[]`, `removable: boolean`; emits: `remove(id)`
- `components/tags/TagPicker.vue` — **TagPicker** — Searchable dropdown for selecting or creating tags; props: `selectedIds: string[]`; emits: `add(tagId)`, `create(name)`; features search filtering, auto-fetch on mount, create-from-input

### Tests

- `__tests__/services/tagService.test.ts` — Tests for all 6 tag service methods. The `getByTask` and `getByContext` tests mock the fetch response as `{ items: tags, total: tags.length }` (a `QueryResult<Tag>` envelope), not a bare array — the service unwraps `.items` before returning. Do not change mocks back to bare arrays.
- `__tests__/stores/tagStore.test.ts` — Unit tests for store CRUD and relationship cache methods.
- `__tests__/components/tags/TagBadge.test.ts` — Render and emit tests for TagBadge.
- `__tests__/components/tags/TagList.test.ts` — Render and empty-state tests for TagList.
- `__tests__/components/tags/TagPicker.test.ts` — Search, select, and create-from-input tests for TagPicker.

## Impact Callouts

### ⚠ Tag (`types/tag.ts`)
Changing the Tag interface shape affects:
- `services/tagService.ts` — All CRUD and relationship methods assume `id` and `name` properties
- `stores/tagStore.ts` — Relies on `Tag.id` for filtering and lookups in taskTags/contextTags/noteTags maps
- `components/tags/TagBadge.vue` — displays `tag.name`, passes `tag.id` on remove emit
- `components/tags/TagPicker.vue` — filters by `tag.name`, passes `tag.id` on add emit

### ⚠ tagService relationship methods
Adding new entity-type associations (e.g., `getByEvent`) affects:
- `stores/tagStore.ts` — must add corresponding cache map and fetch/add/remove methods
- Components in that entity domain — must integrate TagPicker and TagList

## Cross-Domain Dependencies

- **services/client.ts** — tagService uses `request()` for HTTP; handles auth headers and base URL
- **stores/toastStore.ts** — createCRUDStore (base of tagStore) calls `toast.error()` / `toast.success()`
- **services/createCRUDService.ts** — tagService spreads CRUD methods from this factory
- **stores/createCRUDStore.ts** — tagStore uses this factory for base CRUD state and methods
- **Task domain** — TagPicker/TagList used in TaskDetailView for task tag management
- **Context domain** — TagPicker/TagList used in ContextDetailView for context tag management
- **Note domain** — TagPicker/TagList used in NoteDetailView for note tag management
