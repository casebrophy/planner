# Context System (Frontend)

> Manages contexts as ongoing situations or projects. Users organize work into contexts (e.g., "Learning Go", "Q1 Planning") that can be active, paused, or closed. Each context is either a time-bounded project or an ongoing area. Contexts track events (thread entries), metadata, summaries, and last-activity timestamps. The frontend renders context lists in Kanban columns by kind (Project/Area) or by status, and provides forms to create/edit contexts. Event threads appear in detail views.

## Core Types

### Context (types/context.ts)
```typescript
interface Context {
  id: string
  title: string
  description: string
  kind: ContextKind              // 'project' or 'area'
  status: ContextStatus          // 'active', 'paused', or 'closed'
  summary: string                // high-level wrap-up
  lastEvent?: string             // ISO 8601 datetime of most recent event
  createdAt: string
  updatedAt: string
}
```

### NewContext (types/context.ts)
Request DTO for creating a context. Omits `id`, `status`, `summary`, `lastEvent`, timestamps.
```typescript
interface NewContext {
  title: string
  description: string
  kind?: ContextKind  // defaults to 'project' if omitted
}
```

### UpdateContext (types/context.ts)
Partial DTO for editing a context. All fields optional.
```typescript
interface UpdateContext {
  title?: string
  description?: string
  kind?: ContextKind
  status?: ContextStatus
  summary?: string
}
```

### ContextFilter (types/context.ts)
Query filter for listing contexts.
```typescript
interface ContextFilter {
  status?: ContextStatus  // filter by status
  kind?: ContextKind      // filter by kind (not currently used in UI)
  title?: string          // search by title substring
}
```

### ContextEvent (types/event.ts)
Thread entry (event) on a context. Created via /api/v1/contexts/{id}/events POST.
```typescript
interface ContextEvent {
  id: string
  contextId: string
  kind: string                          // event type (e.g., 'update', 'milestone')
  content: string                       // event text
  metadata?: Record<string, unknown>    // extensible data
  sourceId?: string                     // optional reference to originating entity
  createdAt: string
}
```

### NewEvent (types/event.ts)
Request DTO for adding an event to a context.
```typescript
interface NewEvent {
  kind: string
  content: string
  metadata?: Record<string, unknown>
  sourceId?: string
}
```

### Enums (types/enums.ts)
```typescript
// ContextStatus
const ContextStatus = {
  Active: 'active',
  Paused: 'paused',
  Closed: 'closed',
} as const

// ContextKind
const ContextKind = {
  Project: 'project',
  Area: 'area',
} as const

// Labels and colors for UI display
const ContextKindLabels: Record<ContextKind, string> = {
  [ContextKind.Project]: 'Project',
  [ContextKind.Area]: 'Area',
}

const ContextKindColors: Record<ContextKind, string> = {
  [ContextKind.Project]: '#3b82f6',    // blue
  [ContextKind.Area]: '#8b5cf6',       // purple
}

const ContextStatusLabels: Record<ContextStatus, string> = {
  [ContextStatus.Active]: 'Active',
  [ContextStatus.Paused]: 'Paused',
  [ContextStatus.Closed]: 'Closed',
}
```

## File Map

### Stores
- `stores/contextStore.ts` — **useContextStore()** — Pinia store. Wraps CRUD via `createCRUDStore<Context, NewContext, UpdateContext, ContextFilter>()`. Adds `events` ref (ContextEvent[]), `eventsTotal` ref, computed groupers (`contextsByStatus`, `contextsByKind`, `contextById`, `activeCount`, `pausedCount`, `closedCount`), and methods `fetchEvents(contextId, pg)`, `addEvent(contextId, event)`.

### Services
- `services/contextService.ts` — **contextService** — HTTP API client. Wraps CRUD via `createCRUDService<...>()` at `/api/v1/contexts`. Adds `listEvents(contextId, params)` for GET `/api/v1/contexts/{contextId}/events` and `addEvent(contextId, event)` for POST.

### Composables
- `composables/useContextBoard.ts` — **useContextBoard()** — Board view orchestrator. Loads context list on mount, sets up polling refresh, exposes filtering, and refs to computed groupers (`contextsByStatus`, `contextsByKind`, counts). Used by ContextBoard page.
- `composables/useContextDetail.ts` — **useContextDetail(contextId)** — Detail view orchestrator. Loads context by ID, events, linked tags, and tasks. Exposes update, remove, addEvent, addTag, removeTag methods.

### Components
- `components/contexts/ContextForm.vue` — **ContextForm** — Form for create/edit. Props: `context`, `mode: 'create' | 'edit'`. Emits `submit: NewContext | UpdateContext`, `cancel`. In edit mode, shows `status` and `summary` fields. Validates title is non-empty.
- `components/contexts/ContextCard.vue` — **ContextCard** — Single context card. Renders title, kind badge (with color), description, summary, last-event timestamp. Emits `click: id`. Used in ContextKanban.
- `components/contexts/ContextFilterBar.vue` — **ContextFilterBar** — Filter inputs. Props: `filter: ContextFilter`. Emits `update: ContextFilter`. Binds status select and title search, debounced via watch.
- `components/contexts/ContextKanban.vue` — **ContextKanban** — Two-column Kanban (Project | Area). Props: `columns: Record<string, Context[]>`. Renders ContextCard in each column. Emits `select: id`.

## Impact Callouts

### ⚠ Context (types/context.ts)
Changing this interface shape affects:
- `stores/contextStore.ts` — CRUD operations normalize/denormalize, computed groupers iterate `kind` and `status` fields
- `services/contextService.ts` — request/response serialization
- `components/contexts/ContextCard.vue` — template binds `title`, `kind`, `description`, `summary`, `lastEvent`
- `components/contexts/ContextForm.vue` — form binds `title`, `description`, `kind` (create/edit), `status`, `summary` (edit only)
- `components/contexts/ContextKanban.vue` — passes Context to ContextCard
- `composables/useContextDetail.ts` — reads all fields for detail view
- `composables/useContextBoard.ts` — passes contexts to groupers and return refs

### ⚠ ContextStatus (types/enums.ts)
Changing enum values affects:
- `components/contexts/ContextForm.vue` — status select renders ContextStatus.Active/Paused/Closed options
- `components/contexts/ContextFilterBar.vue` — status select filters
- `stores/contextStore.ts` — contextsByStatus computed initializes groups with all status keys
- `types/context.ts` — UpdateContext.status field type
- Status colors and labels in `StatusBadge` component (shared)

### ⚠ ContextKind (types/enums.ts)
Changing enum values affects:
- `components/contexts/ContextForm.vue` — kind select renders ContextKind.Project/Area options
- `components/contexts/ContextCard.vue` — reads `kind`, looks up ContextKindLabels and ContextKindColors for badge
- `components/contexts/ContextKanban.vue` — column keys hardcoded ('project', 'area') must match ContextKind values
- `stores/contextStore.ts` — contextsByKind computed initializes groups, fallback to Project for unknown kind
- `types/context.ts` — Context.kind and NewContext.kind types

### ⚠ NewContext (types/context.ts)
Changing this interface affects:
- `components/contexts/ContextForm.vue` — satisfies NewContext in create mode emit, fields are title, description, kind
- `services/contextService.ts` — POST /api/v1/contexts request body type
- `stores/contextStore.ts` — wrapped by createCRUDStore, used in create methods

### ⚠ UpdateContext (types/context.ts)
Changing this interface affects:
- `components/contexts/ContextForm.vue` — satisfies UpdateContext in edit mode emit, fields are title, description, kind, status, summary
- `services/contextService.ts` — PATCH /api/v1/contexts/{id} request body type
- `stores/contextStore.ts` — wrapped by createCRUDStore, used in update methods
- `composables/useContextDetail.ts` — update() method parameter type

### ⚠ ContextFilter (types/context.ts)
Changing this interface affects:
- `components/contexts/ContextFilterBar.vue` — props type, emits ContextFilter after binding status/title
- `stores/contextStore.ts` — passed to createCRUDStore, used in setFilter()
- `services/contextService.ts` — mapFilter(f) converter maps to query params

### ⚠ ContextEvent (types/event.ts)
Changing this interface affects:
- `stores/contextStore.ts` — events ref holds array of ContextEvent, unshift on addEvent
- `services/contextService.ts` — listEvents() return type
- Detail views (pages/ContextDetail.vue, not shown) — render event thread with content, kind, timestamps

### ⚠ NewEvent (types/event.ts)
Changing this interface affects:
- `composables/useContextDetail.ts` — addEvent() parameter type
- `stores/contextStore.ts` — addEvent() parameter type
- Detail views — event form constructors

## Cross-Domain Dependencies

### Incoming (Things that depend on Context)
- **taskStore** — tasks have optional `contextId: string` field linking to a context
- **tagStore** — tags can be linked to contexts; `contextTags[contextId]` mapping
- Clarifications — context assignment, context debrief, overlapping contexts kinds reference context IDs
- Pages and composables (`useToday`, `useSearch`, `useDashboard`) — pull contextStore for scoping/grouping

### Outgoing (Context depends on)
- **Shared components** — `StatusBadge` component for rendering context status (reused across task/context/event statuses)
- **Shared composables** — `usePolling` for auto-refresh
- **Shared utilities** — `useToast` for error/success notifications (via crud store)
- **API client** — `/api/v1/contexts` and `/api/v1/contexts/{id}/events` endpoints

## Implementation Notes

### CRUD Pattern
`contextStore` and `contextService` follow a composable factory pattern:
- `createCRUDStore<Context, NewContext, UpdateContext, ContextFilter>()` provides standard list, fetch, create, update, delete with pagination, filtering, ordering
- `createCRUDService<...>()` provides HTTP methods (GET list, GET by ID, POST create, PATCH update, DELETE)
- Service.mapFilter() is called by the store to serialize filter to query params

### Event Thread Pattern
Context events are managed separately:
- `contextStore.fetchEvents(contextId, page?)` — paginated list from `/api/v1/contexts/{contextId}/events?page=X&rows=50`
- `contextStore.addEvent(contextId, event)` — POST to `/api/v1/contexts/{contextId}/events`, updates local array
- Events are stored in separate refs (`events`, `eventsTotal`) not mixed with context CRUD items

### Grouping Computeds
The store exposes two computed groupers:
- `contextsByStatus` — Record<ContextStatus, Context[]> for status-grouped views (Active | Paused | Closed)
- `contextsByKind` — Record<ContextKind, Context[]> for kind-grouped views (Project | Area) with unknown-kind fallback

### Form Modes
ContextForm is dual-purpose:
- **Create mode** — shows title, description, kind (defaults to Project); emits NewContext
- **Edit mode** — adds status and summary fields to the above; emits UpdateContext

### Kanban Column Mapping
ContextKanban hardcodes column keys `'project'` and `'area'` which must match `ContextKind.Project` and `ContextKind.Area` values ('project', 'area'). If enum values change, column keys must be updated in parallel.
