# Tag System (Frontend)

> Tag management domain: create and manage global tags, then associate them with tasks or contexts. Tags are simple (id + name). The store maintains a global tag list plus per-entity caches (`taskTags[taskId]` and `contextTags[contextId]`). There is no dedicated tag view — tags are managed inline from TaskDetail and ContextDetail.

## Core Types

```ts
// types/tag.ts
interface Tag {
  id: string
  name: string
}

interface NewTag {
  name: string
}
```

## File Map

### Stores
- `stores/tagStore.ts` — **useTagStore** — Pinia store wrapping createCRUDStore (list/create/update/delete global tags); adds `taskTags: Record<string, Tag[]>` and `contextTags: Record<string, Tag[]>` caches; exposes `fetchTagsForTask`, `addTagToTask`, `removeTagFromTask`, `fetchTagsForContext`, `addTagToContext`, `removeTagFromContext`

### Services
- `services/tagService.ts` — **tagService** — createCRUDService wrapper for `/api/v1/tags`; extends with `getByTask(taskId)` → GET `/api/v1/tasks/:id/tags`, `addToTask`/`removeFromTask` → POST/DELETE `/api/v1/tasks/:id/tags/:tagId`, `getByContext(contextId)` → GET `/api/v1/contexts/:id/tags`, `addToContext`/`removeFromContext` → POST/DELETE `/api/v1/contexts/:id/tags/:tagId`

### Components
- `components/tags/TagBadge.vue` — **TagBadge** — renders a single tag as a pill/badge (name)
- `components/tags/TagList.vue` — **TagList** — renders an array of Tag objects as TagBadges; optionally shows remove buttons
- `components/tags/TagPicker.vue` — **TagPicker** — dropdown/search UI for selecting from the global tag list; emits select + create-new

## Impact Callouts

### ⚠ Tag (types/tag.ts)
Changing this interface shape affects:
- `stores/tagStore.ts` — `items` (global list), `taskTags[id]` and `contextTags[id]` caches all store Tag[]; addTagToTask/Context looks up `.id` in items
- `services/tagService.ts` — deserializes Tag[] from all getBy* responses; Tag used as request/response body for CRUD
- `composables/useTaskDetail.ts` — reads `tagStore.taskTags[taskId]` (Tag[]) for `tags` computed
- `composables/useContextDetail.ts` — reads `tagStore.contextTags[contextId]` (Tag[]) for `tags` computed
- `composables/useSearch.ts` — matches on `.name`
- `components/tags/TagBadge.vue` — binds `.name` for display
- `components/tags/TagList.vue` — iterates Tag[] array, passes each to TagBadge
- `components/tags/TagPicker.vue` — renders Tag[] from global store, uses `.id` for selection

## Cross-Domain Dependencies

- `stores/taskStore.ts` — TagStore.addTagToTask/removeTagFromTask operates on tasks; taskTags cache is keyed by task ID
- `stores/contextStore.ts` — TagStore.addTagToContext/removeTagFromContext operates on contexts; contextTags cache is keyed by context ID
- `composables/useTaskDetail.ts` — uses tagStore directly for per-task tags
- `composables/useContextDetail.ts` — uses tagStore directly for per-context tags
- `composables/useSearch.ts` — fetches tagStore.items to search tag names
- `stores/toastStore.ts` — tagStore (via createCRUDStore) emits toasts; fetchTagsForTask/Context also calls toast.error on failure
