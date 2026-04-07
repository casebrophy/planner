# EntityLink Frontend System

> The entitylink feature provides directional semantic linking between entities (tasks, notes, events). Links can be created manually or suggested by the AI classify pipeline. The frontend fetches, caches, creates, and deletes links via a Pinia store, with integration into NoteDetailView and TaskDetailView to display related items.

## Core Types

### `types/entityLink.ts`

```typescript
export type EntityKind = 'task' | 'note' | 'event'
export type LinkKind = 'manual' | 'ai_suggested'

export interface EntityLink {
  id: string
  sourceType: EntityKind
  sourceId: string
  targetType: EntityKind
  targetId: string
  confidence: number    // 1.0 for manual; 0.0–1.0 for AI-suggested
  kind: LinkKind
  createdAt: string
}

export interface NewEntityLink {
  sourceType: EntityKind
  sourceId: string
  targetType: EntityKind
  targetId: string
}

export interface EntityLinkFilter {
  entityType?: EntityKind
  entityId?: string
}
```

## File Map

### Services
- **`services/entityLinkService.ts`** — `entityLinkService` object with three async methods
  - `listByEntity(entityType: string, entityId: string): Promise<EntityLink[]>` — GET /api/v1/entity-links with query params, returns items array
  - `create(link: NewEntityLink): Promise<EntityLink>` — POST /api/v1/entity-links, returns created link
  - `delete(id: string): Promise<void>` — DELETE /api/v1/entity-links/{id}
  - **Imports:** `request` from client, `EntityLink` and `NewEntityLink` types

### Stores
- **`stores/entityLinkStore.ts`** — `useEntityLinkStore` Pinia store
  - **State:**
    - `linksByEntity: Ref<Record<string, EntityLink[]>>` — cache keyed by `"${entityType}:${entityId}"`
    - `loading: Ref<boolean>` — loading state during fetch
  - **Functions:**
    - `fetchLinks(entityType, entityId, force?)` — fetches links for entity; caches result; invalidates on create/delete; toast on error
    - `getLinks(entityType, entityId): EntityLink[]` — returns cached links (empty array if not cached)
    - `createLink(link: NewEntityLink): Promise<EntityLink | null>` — creates link, invalidates both source+target caches, toast on error
    - `deleteLink(link: EntityLink): Promise<void>` — deletes link, invalidates both source+target caches, toast on error
  - **Imports:** `entityLinkService`, `useToastStore`

### Views
- **`views/NoteDetailView.vue`** — Note detail page with Related Items section
  - Imports `useEntityLinkStore`; calls `fetchLinks('note', note.id)` on mount
  - `relatedItems` computed reads `getLinks('note', note.id)` and returns `EntityLink[]`
  - UI for creating link: calls `createLink({sourceType: 'note', sourceId: note.id, targetType: ..., targetId: ...})`
  - UI for deleting link: calls `deleteLink(link)` on click

- **`views/TaskDetailView.vue`** — Task detail page with Related Items section
  - Imports `useEntityLinkStore`; calls `fetchLinks('task', task.id)` on mount
  - `relatedItems` computed reads `getLinks('task', task.id)` and returns `EntityLink[]`
  - UI for creating link: calls `createLink({sourceType: 'task', sourceId: task.id, targetType: ..., targetId: ...})`
  - UI for deleting link: calls `deleteLink(link)` on click

---

## Impact Callouts

### ⚠ EntityLink (`types/entityLink.ts`)
Changing this interface shape affects:
- `services/entityLinkService.ts` — listByEntity() response parsing, create() return type, type annotations in function signatures
- `stores/entityLinkStore.ts` — linksByEntity `Record<string, EntityLink[]>` type, createLink/deleteLink methods work with this type, cache stores arrays of EntityLink
- `views/NoteDetailView.vue` — relatedItems computed property destructures and displays EntityLink fields (sourceType, targetType, sourceId, targetId, confidence, kind); link deletion expects EntityLink
- `views/TaskDetailView.vue` — same as NoteDetailView

### ⚠ NewEntityLink (`types/entityLink.ts`)
Changing this interface shape affects:
- `services/entityLinkService.ts` — create() param type annotation
- `stores/entityLinkStore.ts` — createLink() param type, passed to entityLinkService.create()
- `views/NoteDetailView.vue` — link creation form builds this object with sourceType='note', sourceId from note.id, targetType/targetId from user selection
- `views/TaskDetailView.vue` — same as NoteDetailView

### ⚠ EntityKind type (`types/entityLink.ts`)
Changing the union of allowed entity types affects:
- `views/NoteDetailView.vue` — link creation form entity-type selector must match allowed values
- `views/TaskDetailView.vue` — same
- `stores/entityLinkStore.ts` — cache key function accepts entityType; backend validation ensures only valid kinds are stored

### ⚠ useEntityLinkStore (`stores/entityLinkStore.ts`)
Changes to store API affect:
- `views/NoteDetailView.vue` — imports store, calls fetchLinks(), getLinks(), createLink(), deleteLink()
- `views/TaskDetailView.vue` — imports store, calls fetchLinks(), getLinks(), createLink(), deleteLink()

---

## Routes & Integration

The feature integrates into two detail views:
- `/notes/:id` — NoteDetailView fetches and displays entity links
- `/tasks/:id` — TaskDetailView fetches and displays entity links

Both views:
1. Call `fetchLinks(entityType, entityId)` on mount to load related items
2. Compute `relatedItems` from `getLinks()` to display in template
3. Provide UI to create/delete links, calling `createLink()/deleteLink()`

---

## Cross-Domain Dependencies

**External dependencies:**
- `services/client` — HTTP request abstraction (used by entityLinkService)
- `stores/toastStore` — user notifications on error/success
- Views depend on their respective detail models (note, task)

**Used by:**
- NoteDetailView — to display related tasks/events/notes
- TaskDetailView — to display related tasks/events/notes

---

## API Contract

```
GET /api/v1/entity-links?entity_type={task|note|event}&entity_id={uuid}
  → { items: EntityLink[], total: number }

POST /api/v1/entity-links
  Body: { sourceType, sourceId, targetType, targetId }
  → EntityLink (kind set to "manual", confidence set to 1.0 by backend)

DELETE /api/v1/entity-links/{link_id}
  → 204 No Content
```

All endpoints require API key auth (configured in request client).
