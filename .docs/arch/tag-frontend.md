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
  - Task associations: `getByTask(taskId)`, `addToTask(taskId, tagId)`, `removeFromTask(taskId, tagId)`
  - Context associations: `getByContext(contextId)`, `addToContext(contextId, tagId)`, `removeFromContext(contextId, tagId)`
  - Note associations: `getByNote(noteId)`, `addToNote(noteId, tagId)`, `removeFromNote(noteId, tagId)`

### Stores
- `stores/tagStore.ts` — **useTagStore** — Pinia store extending CRUD with relationship caches:
  - State: `taskTags: Record<string, Tag[]>`, `contextTags: Record<string, Tag[]>`, `noteTags: Record<string, Tag[]>`
  - Methods: `fetchTagsForTask`, `addTagToTask`, `removeTagFromTask`, `fetchTagsForContext`, `addTagToContext`, `removeTagFromContext`, `fetchTagsForNote`, `addTagToNote`, `removeTagFromNote`

### Components
- `components/tags/TagBadge.vue` — **TagBadge** — Displays individual tag with optional remove button; props: `tag: Tag`, `removable: boolean`; emits: `remove(id)`
- `components/tags/TagList.vue` — **TagList** — Renders flex-wrapped collection of TagBadge components with empty state; props: `tags: Tag[]`, `removable: boolean`; emits: `remove(id)`
- `components/tags/TagPicker.vue` — **TagPicker** — Searchable dropdown for selecting or creating tags; props: `selectedIds: string[]`; emits: `add(tagId)`, `create(name)`; features search filtering, auto-fetch on mount, create-from-input

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
