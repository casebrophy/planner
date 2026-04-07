# Context System

The context system manages high-level organizational containers — Projects (time-bounded, can be closed) and Areas (ongoing, never-closing). Contexts serve as a scoping mechanism for tasks, notes, and other entities. Each context tracks its lifecycle (Active/Paused/Closed) and maintains a thread of events that document changes and activity.

## Core Types

### Context

```typescript
interface Context {
  id: string
  title: string
  description: string
  kind: ContextKind
  status: ContextStatus
  summary: string
  lastEvent?: string
  createdAt: string
  updatedAt: string
}
```

The main context entity. `kind` determines lifecycle rules (Project vs Area). `status` tracks lifecycle state (Active/Paused/Closed). `lastEvent` is a timestamp of the most recent event or thread activity. Used by TaskCard to look up context labels, by NoteForm to allow context assignment, and by other modules for scoping.

### NewContext

```typescript
interface NewContext {
  title: string
  description: string
  kind?: ContextKind
}
```

Creation DTO. Kind defaults to Project if omitted.

### UpdateContext

```typescript
interface UpdateContext {
  title?: string
  description?: string
  kind?: ContextKind
  status?: ContextStatus
  summary?: string
}
```

Partial update DTO. Allows changing status (e.g., Active → Closed) and summary.

### ContextFilter

```typescript
interface ContextFilter {
  status?: ContextStatus
  kind?: ContextKind
  title?: string
}
```

Query filter for board views. Supports filtering by status, kind, and title search.

### ContextEvent

```typescript
interface ContextEvent {
  id: string
  contextId: string
  kind: string
  content: string
  metadata?: Record<string, unknown>
  sourceId?: string
  createdAt: string
}
```

Event thread entry for a context. Captures changes, user actions, and system events with optional metadata (e.g., who changed the status, why it was paused).

### NewEvent

```typescript
interface NewEvent {
  kind: string
  content: string
  metadata?: Record<string, unknown>
  sourceId?: string
}
```

Creation DTO for context events.

### Enums

```typescript
const ContextStatus = {
  Active: 'active',
  Paused: 'paused',
  Closed: 'closed',
} as const
type ContextStatus = typeof ContextStatus[keyof typeof ContextStatus]

const ContextKind = {
  Project: 'project',
  Area: 'area',
} as const
type ContextKind = typeof ContextKind[keyof typeof ContextKind]

const ContextKindLabels: Record<ContextKind, string> = {
  [ContextKind.Project]: 'Project',
  [ContextKind.Area]: 'Area',
}

const ContextKindColors: Record<ContextKind, string> = {
  [ContextKind.Project]: '#3b82f6',   // blue
  [ContextKind.Area]: '#8b5cf6',      // purple
}

const ContextStatusLabels: Record<ContextStatus, string> = {
  [ContextStatus.Active]: 'Active',
  [ContextStatus.Paused]: 'Paused',
  [ContextStatus.Closed]: 'Closed',
}
```

## File Map

### Store
- `stores/contextStore.ts` — **useContextStore** — Pinia store managing context CRUD, filtering, pagination, events, and derived computations (grouping by status/kind, count aggregates)

### Service
- `services/contextService.ts` — **contextService** — REST API client wrapping `/api/v1/contexts` and event endpoints (listEvents, addEvent)

### Composables
- `composables/useContextDetail.ts` — **useContextDetail** — Detail page logic: fetches single context + events + linked tags/tasks, provides update/delete/event/tag-linking methods
- `composables/useContextBoard.ts` — **useContextBoard** — Board/list view logic: fetches contexts with filtering, groups by status/kind, provides filter updates and refresh, auto-polling

### Components
- `components/contexts/ContextCard.vue` — Card display for a single context, shows title, kind badge (colored), status, description excerpt, summary, last event time
- `components/contexts/ContextFilterBar.vue` — Filter inputs: status dropdown, title search, clear button; emits filter updates
- `components/contexts/ContextKanban.vue` — Two-column Kanban layout (Projects | Areas), renders ContextCards in columns with count badges
- `components/contexts/ContextForm.vue` — Create/edit form: title (required), description, kind (Project/Area), status (edit-only), summary (edit-only); emits NewContext or UpdateContext

## Impact Callouts

### ⚠ Context (types/context.ts)
The core context entity. Changing this interface affects:
- `stores/contextStore.ts` — stores Context items in CRUD store, contextById lookup, status/kind grouping
- `services/contextService.ts` — CRUD service serializes to/from API, filters on status/kind/title
- `composables/useContextDetail.ts` — currentItem binding, update payload construction
- `composables/useContextBoard.ts` — items filtering and grouping by status/kind
- `components/contexts/ContextCard.vue` — template binds title, description, summary, kind, status, lastEvent
- `components/contexts/ContextForm.vue` — form binds to context fields for create/edit
- `components/tasks/TaskCard.vue` — contextById lookup to display context label on task cards
- `components/notes/NoteForm.vue` — context selection dropdown in note creation/edit
- Any other modules that reference contextId (scope filtering)

### ⚠ ContextFilter (types/context.ts)
Query filter shape used by the board. Changing this interface affects:
- `components/contexts/ContextFilterBar.vue` — emits ContextFilter objects with status/kind/title
- `composables/useContextBoard.ts` — receives and applies filters via store.setFilter()
- `stores/contextStore.ts` — CRUD store's filter state and fetchList() behavior
- `services/contextService.ts` — mapFilter() translates to API query params

### ⚠ ContextEvent (types/event.ts)
Event thread entries for context activity. Changing this interface affects:
- `stores/contextStore.ts` — events array binding, listEvents/addEvent payload
- `services/contextService.ts` — listEvents/addEvent serialization
- `composables/useContextDetail.ts` — events display, addEvent() method
- Thread UI components (if they exist) — render event list with kind/content/metadata

### ⚠ ContextStatus (types/enums.ts)
Enum of context lifecycle states. Changing this affects:
- `stores/contextStore.ts` — contextsByStatus grouping, count computations
- `services/contextService.ts` — filter mapping
- `components/contexts/ContextFilterBar.vue` — status dropdown options
- `components/contexts/ContextForm.vue` — status select options (edit-only)
- `components/shared/StatusBadge.vue` — renders status color/label (if it knows ContextStatus)
- Any downstream modules that dispatch based on status (e.g., can't add tasks to Closed contexts)

### ⚠ ContextKind (types/enums.ts)
Enum of context types (Project vs Area). Changing this affects:
- `stores/contextStore.ts` — contextsByKind grouping, fallback logic
- `services/contextService.ts` — filter mapping
- `components/contexts/ContextForm.vue` — kind select options
- `components/contexts/ContextCard.vue` — renders kind badge with ContextKindLabels and ContextKindColors
- `components/contexts/ContextKanban.vue` — column definitions hardcoded for 'project' and 'area' keys
- Business logic: Areas are always active; Projects can be closed

## Cross-Domain Dependencies

### Depends On (imports from other domains)
- `stores/tagStore.ts` — useContextDetail calls fetchTagsForContext() and reads contextTags[contextId]
- `stores/taskStore.ts` — useContextDetail calls fetchList() and filters tasks by contextId
- `components/shared/StatusBadge.vue` — ContextCard renders status badges
- `composables/usePolling.ts` — useContextBoard uses auto-polling for real-time updates
- `services/client.ts` — contextService uses HTTP client for API calls

### Used By (imported by other domains)
- `components/tasks/TaskCard.vue` — imports useContextStore, contextById() to look up context label for display
- `components/notes/NoteForm.vue` — imports useContextStore, fetchList() for context selection dropdown, binds contextId
- `components/calendar-events/CalendarEventForm.vue` — likely uses contextId for event scoping (check if calendar events are context-scoped)
- `stores/taskStore.ts` — Task interface has contextId; tasks are scoped to contexts
- `stores/noteStore.ts` — Note interface has contextId; notes are scoped to contexts
- `services/tagService.ts` — supports fetchTagsForContext() and contextTags grouping
- `views/ContextBoardView.vue` — renders the context Kanban board (not listed in Glob but implied)
- `views/TaskBoardView.vue` — may use context filter/scoping
- `views/TodayView.vue` — may display context-scoped tasks
- `components/layout/AppSidebar.vue` — references contexts in navigation (likely shows context list or active count)
- `components/clarifications/ClarificationCard.vue` — may reference contextId for scope

## Notes

- **Lifecycle Rules:** Projects can transition Active → Paused → Closed. Areas stay Active (UI enforces this).
- **Events Thread:** Every context maintains an ordered event log. Events capture why a context was paused/closed, task additions, tag updates, etc. Use for audit trail and context history.
- **Scoping:** Tasks, notes, and other entities reference contextId. Filtering by context is fundamental to the UI (board views, daily plan, etc.).
- **Status Badges:** Contexts reuse the shared StatusBadge component — if status badge colors or labels change, context cards update automatically.
