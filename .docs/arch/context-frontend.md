# Context System (Frontend)

> Context management domain: create, filter, and manage contexts (projects/areas). Contexts have status (active/paused/closed), a summary, and a lastEvent timestamp. The board view shows a Kanban by kind (project/area); the detail view shows the context with its event timeline, linked tasks, and tags.

## Core Types

```ts
// types/context.ts
interface Context {
  id: string
  title: string
  description: string
  kind: ContextKind        // 'project' | 'area'
  status: ContextStatus    // 'active' | 'paused' | 'closed'
  summary: string
  lastEvent?: string
  createdAt: string
  updatedAt: string
}

interface NewContext {
  title: string
  description: string
}

interface UpdateContext {
  title?: string
  description?: string
  status?: ContextStatus
  summary?: string
}

interface ContextFilter {
  status?: ContextStatus
  title?: string
}
```

```ts
// types/event.ts
interface ContextEvent {
  id: string
  contextId: string
  kind: string
  content: string
  metadata?: Record<string, unknown>
  sourceId?: string
  createdAt: string
}

interface NewEvent {
  kind: string
  content: string
  metadata?: Record<string, unknown>
  sourceId?: string
}
```

```ts
// types/enums.ts (context-related)
const ContextKind = { Project: 'project', Area: 'area' } as const
type ContextKind = (typeof ContextKind)[keyof typeof ContextKind]

const ContextKindLabels: Record<ContextKind, string>   // 'Project' | 'Area'
const ContextKindColors: Record<ContextKind, string>   // '#3b82f6' (project) | '#8b5cf6' (area)

const ContextStatus = { Active: 'active', Paused: 'paused', Closed: 'closed' } as const
type ContextStatus = (typeof ContextStatus)[keyof typeof ContextStatus]

const ContextStatusLabels: Record<ContextStatus, string>  // display labels
const StatusColors: Record<string, string>                 // includes 'active', 'paused', 'closed'
```

## File Map

### Stores
- `stores/contextStore.ts` — **useContextStore** — Pinia store wrapping createCRUDStore; adds `events` ref (ContextEvent[]), `eventsTotal`, `contextsByStatus` computed (Record<ContextStatus, Context[]>), `contextsByKind` computed (Record<ContextKind, Context[]>), `contextById` computed ((id: string) => Context | undefined), `activeCount`/`pausedCount`/`closedCount`, `fetchEvents(contextId)`, `addEvent(contextId, event)`

### Services
- `services/contextService.ts` — **contextService** — createCRUDService wrapper for `/api/v1/contexts`; extends with `listEvents(contextId, params)` → GET `/api/v1/contexts/:id/events` and `addEvent(contextId, event)` → POST

### Composables
- `composables/useContextBoard.ts` — **useContextBoard** — board-level composable; wraps contextStore, wires polling, exposes `contextsByStatus`/`contextsByKind`/`activeCount`/`pausedCount`/`closedCount` + setFilter/refresh
- `composables/useContextDetail.ts` — **useContextDetail(contextId)** — detail composable; loads context + events + tags + linked tasks in parallel (contextStore + tagStore + taskStore); exposes update/remove/addEvent/addTag/removeTag

### Components
- `components/contexts/ContextCard.vue` — **ContextCard** — single context card (title, kind badge, status, lastEvent, summary preview); kind badge uses ContextKindLabels and ContextKindColors with 20% opacity background
- `components/contexts/ContextFilterBar.vue` — **ContextFilterBar** — filter UI for status and title search
- `components/contexts/ContextForm.vue` — **ContextForm** — create/edit form for NewContext/UpdateContext fields
- `components/contexts/ContextKanban.vue` — **ContextKanban** — two-column Kanban layout (project/area), renders ContextCard per column; `columnDefs` array defines columns with `key`, `label`, `color`, and `emptyLabel`
- `components/events/EventForm.vue` — **EventForm** — form for adding a NewEvent to a context
- `components/events/EventTimeline.vue` — **EventTimeline** — ordered list of ContextEvents
- `components/events/EventTimelineItem.vue` — **EventTimelineItem** — single event in the timeline (kind, content, createdAt)

### Views
- `views/ContextBoardView.vue` — **ContextBoardView** — uses useContextBoard; renders ContextFilterBar + ContextKanban with `contextsByKind` passed as `columns` prop
- `views/ContextDetailView.vue` — **ContextDetailView** — uses useContextDetail; renders ContextForm (edit), EventTimeline + EventForm, tag management, linked tasks list

## Impact Callouts

### ⚠ Context (types/context.ts)
Changing this interface shape affects:
- `stores/contextStore.ts` — stores `Context[]` in `items`; `contextsByStatus` groups by `.status`; `contextsByKind` groups by `.kind`; `contextById` lookups by `.id`; `activeCount`/`pausedCount`/`closedCount` filter by `.status`
- `composables/useContextBoard.ts` — exposes `contexts` (items array), `contextsByStatus`, `contextsByKind`
- `composables/useContextDetail.ts` — exposes `context` (currentItem); passes to `update(contextId, UpdateContext)`
- `composables/useDashboard.ts` — reads `.status` to compute contextCounts; exposes activeContexts
- `composables/useToday.ts` — reads `.id` + `.title` to build contextMap (id→title lookup for task display)
- `composables/useSearch.ts` — matches on `.title` and `.description`
- `components/contexts/ContextCard.vue` — binds .title, .kind, .status, .lastEvent, .summary
- `components/contexts/ContextForm.vue` — binds all editable fields

### ⚠ ContextEvent (types/event.ts)
Changing this interface shape affects:
- `stores/contextStore.ts` — stores `ContextEvent[]` in `events` ref; `addEvent` prepends to array
- `composables/useContextDetail.ts` — exposes `events` and `eventsTotal`
- `services/contextService.ts` — deserializes ContextEvent from listEvents/addEvent responses
- `components/events/EventTimeline.vue` — iterates events array
- `components/events/EventTimelineItem.vue` — binds .kind, .content, .createdAt, .metadata

### ⚠ ContextKind (types/enums.ts)
Adding or removing values affects:
- `stores/contextStore.ts` — `contextsByKind` computed uses ContextKind values as object keys
- `components/contexts/ContextCard.vue` — kind badge reads ContextKindLabels + ContextKindColors
- `components/contexts/ContextKanban.vue` — `columnDefs` must define column per kind value

### ⚠ ContextStatus (types/enums.ts)
Adding or removing values affects:
- `stores/contextStore.ts` — contextsByStatus uses ContextStatus values as object keys; counts filter by enum values
- `composables/useContextBoard.ts` — exposes contextsByStatus with status keys
- `composables/useDashboard.ts` — counts by ContextStatus.Active/Paused/Closed
- `composables/useToday.ts` — no direct enum use (reads status string in contextMap)
- `components/shared/StatusBadge.vue` — uses StatusColors keyed by status string

## Cross-Domain Dependencies

- `stores/tagStore.ts` — useContextDetail loads context tags via tagStore; tag add/remove updates contextStore cache
- `stores/taskStore.ts` — useContextDetail filters taskStore.items by contextId to compute linkedTasks
- `composables/usePolling.ts` — used by useContextBoard for background refresh
- `stores/toastStore.ts` — contextStore calls toast.success/error on addEvent; createCRUDStore handles list/create/update/delete toasts
- `stores/captureStore.ts` — calls contextStore.create() when submitting a new context from Capture view
