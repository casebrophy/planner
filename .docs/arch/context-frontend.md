# Context System (Frontend)

Vue 3 + TypeScript implementation of the context management feature. Contexts are organizational units (projects or areas) that group tasks, events, and tags. Contexts support stateful lifecycle management (active/paused/closed) and event threading for maintaining a continuous record of activity and change.

## Core Types

### Context — Main Entity

```typescript
interface Context {
  id: string
  title: string
  description: string
  kind: ContextKind           // 'project' | 'area'
  status: ContextStatus       // 'active' | 'paused' | 'closed'
  summary: string             // High-level summary of context state
  lastEvent?: string          // ISO 8601 timestamp of most recent event
  parentContextId?: string    // UUID of parent context (nullable self-reference)
  createdAt: string           // ISO 8601 creation timestamp
  updatedAt: string           // ISO 8601 last update timestamp
}
```

**File:** `types/context.ts`

### NewContext — Creation Request

Request DTO for creating a context. Omits server-managed fields (`id`, `status`, `summary`, `lastEvent`, timestamps).

```typescript
interface NewContext {
  title: string
  description: string
  kind?: ContextKind          // Defaults to 'project' if omitted
  parentContextId?: string    // Optional parent context UUID
}
```

**File:** `types/context.ts`  
**Used by:** ContextForm (create mode), contextService.create(), contextStore.create()

### UpdateContext — Edit Request

Partial DTO for updating a context. All fields optional; only provided fields are updated.

```typescript
interface UpdateContext {
  title?: string
  description?: string
  kind?: ContextKind
  status?: ContextStatus
  summary?: string
  parentContextId?: string    // Optional parent context UUID (set to null to clear)
}
```

**File:** `types/context.ts`  
**Used by:** ContextForm (edit mode), contextService.update(), contextStore.update(), useContextDetail.update()

### ContextFilter — Query Filter

Declarative filter for listing contexts. Applied by store/service to generate query params.

```typescript
interface ContextFilter {
  status?: ContextStatus      // Filter by status ('active', 'paused', 'closed')
  kind?: ContextKind          // Filter by kind ('project', 'area') — not used in current UI
  title?: string              // Substring search on title field
  parentContextId?: string    // Filter by parent context UUID
}
```

**File:** `types/context.ts`  
**Used by:** ContextFilterBar, contextStore.setFilter(), contextService.queryAll()

### ContextEvent — Event/Thread Entry

Immutable record of activity on a context. Events form an append-only thread. Created via POST `/api/v1/contexts/{id}/events`.

```typescript
interface ContextEvent {
  id: string
  contextId: string
  kind: string                          // Event type (e.g., 'status_change', 'event_added', 'task_linked')
  content: string                       // Human-readable event description
  metadata?: Record<string, unknown>    // Event-specific structured data
  sourceId?: string                     // Optional reference to related entity (task ID, note ID, etc.)
  createdAt: string                     // ISO 8601 creation timestamp
}
```

**File:** `types/event.ts`  
**Used by:** contextStore.events[], contextService.listEvents(), EventTimeline component

### NewEvent — Event Creation Request

Request DTO for adding an event to a context.

```typescript
interface NewEvent {
  kind: string
  content: string
  metadata?: Record<string, unknown>
  sourceId?: string
}
```

**File:** `types/event.ts`  
**Used by:** EventForm component, contextStore.addEvent(), contextService.addEvent(), useContextDetail.addEvent()

### Enum Types

All exported from `types/enums.ts`:

```typescript
const ContextStatus = {
  Active: 'active',
  Paused: 'paused',
  Closed: 'closed',
} as const
type ContextStatus = (typeof ContextStatus)[keyof typeof ContextStatus]

const ContextKind = {
  Project: 'project',  // Time-bounded, can be closed
  Area: 'area',        // Ongoing, always active
} as const
type ContextKind = (typeof ContextKind)[keyof typeof ContextKind]
```

### Enum Labels & Colors

```typescript
const ContextKindLabels: Record<ContextKind, string> = {
  [ContextKind.Project]: 'Project',
  [ContextKind.Area]: 'Area',
}

const ContextKindColors: Record<ContextKind, string> = {
  [ContextKind.Project]: '#3b82f6',    // Blue
  [ContextKind.Area]: '#8b5cf6',       // Purple
}

const ContextStatusLabels: Record<ContextStatus, string> = {
  [ContextStatus.Active]: 'Active',
  [ContextStatus.Paused]: 'Paused',
  [ContextStatus.Closed]: 'Closed',
}

// Status colors are shared across domain (see StatusColors in enums.ts)
// active: '#22c55e', paused: '#eab308', closed: '#6b7280'
```

**File:** `types/enums.ts`  
**Used by:** ContextCard (kind badge), ContextForm (select options), ContextFilterBar (status select), shared StatusBadge component

## File Map

### Types

| File | Purpose | Exports |
|------|---------|---------|
| **types/context.ts** | Core context entities | `Context`, `NewContext`, `UpdateContext`, `ContextFilter` interfaces |
| **types/event.ts** | Event threading | `ContextEvent`, `NewEvent` interfaces |
| **types/enums.ts** | Enum values and labels | `ContextStatus`, `ContextKind`, `ContextKindLabels`, `ContextKindColors`, `ContextStatusLabels` |
| **types/index.ts** | Public exports | Re-exports all context types from above |

### Services

| File | Purpose | Key Exports |
|------|---------|-------------|
| **services/contextService.ts** | HTTP API client for context CRUD and events | `contextService` object with methods: `create()`, `update()`, `remove()`, `queryAll()`, `queryByID()`, `listEvents()`, `addEvent()` |

Uses `createCRUDService<Context, NewContext, UpdateContext, ContextFilter>()` generic factory with base path `/api/v1/contexts`. Implements `mapFilter()` to translate `ContextFilter` to query params (`?status=X&kind=Y&title=Z&parent_context_id=UUID`).

### Stores

| File | Purpose | Key State & Methods |
|------|---------|-------------------|
| **stores/contextStore.ts** | Pinia store for context state management | State: `items: Ref<Context[]>`, `total: Ref<number>`, `loading: Ref<boolean>`, `error: Ref<string \| null>`, `filter: Ref<ContextFilter>`, `currentItem: Ref<Context \| null>` (via CRUD mixin) + `events: Ref<ContextEvent[]>`, `eventsTotal: Ref<number>` |

Uses `createCRUDStore<Context, NewContext, UpdateContext, ContextFilter>()` mixin providing CRUD methods. Extends with event-specific state and methods:
- **Computed:** `contextsByStatus` (grouped by active/paused/closed), `contextsByKind` (grouped by project/area), `contextById(id)`, `activeCount`, `pausedCount`, `closedCount`
- **Actions:** `fetchEvents(contextId, page?)`, `addEvent(contextId, event)`
- **Error Handling:** All async operations show success/error toast via `useToastStore()`

### Composables

| File | Purpose | Arguments | Returns |
|------|---------|-----------|---------|
| **composables/useContextBoard.ts** | Board/list view orchestrator | (none) | Object with refs: `contexts`, `total`, `loading`, `error`, `filter`, `contextsByStatus`, `contextsByKind`, `activeCount`, `pausedCount`, `closedCount`, `isEmpty`; methods: `setFilter(f)`, `refresh()` |
| **composables/useContextDetail.ts** | Detail view orchestrator | `contextId: string` | Object with refs: `context`, `events`, `eventsTotal`, `tags`, `linkedTasks`, `loading`; methods: `update()`, `remove()`, `addEvent()`, `addTag()`, `removeTag()`, `reload()` |

**useContextBoard:**
- Wraps `useContextStore()` and re-exports computed groupers + count properties
- Adds `isEmpty` computed (true if not loading and contexts empty)
- Auto-loads on mount via `onMounted()`
- Sets up polling via `usePolling()` for auto-refresh
- Filter changes trigger `setFilter()` → `store.setFilter()` → `store.fetchList(true)` pipeline

**useContextDetail:**
- Accepts `contextId` parameter to scope all operations to a single context
- Dependencies: reads from `contextStore`, `tagStore` (for context tags), `taskStore` (for linked tasks)
- Parallel load on mount: `Promise.all([fetchContext, fetchEvents, fetchTags, fetchTasks])`
- Filter linked tasks by `task.contextId === contextId`
- Tag operations delegate to `tagStore.addTagToContext()`, `removeTagFromContext()`

### Components

| File | Purpose | Props | Emits |
|------|---------|-------|-------|
| **components/contexts/ContextForm.vue** | Unified create/edit form | `context?: Context \| null`, `mode: 'create' \| 'edit'` | `submit: NewContext \| UpdateContext`, `cancel` |
| **components/contexts/ContextCard.vue** | Single context card display | `context: Context` | `click: string` (context ID) |
| **components/contexts/ContextFilterBar.vue** | Filter UI (status, title search) | `filter: ContextFilter` | `update: ContextFilter` |
| **components/contexts/ContextKanban.vue** | Two-column Kanban (projects/areas) | `columns: Record<string, Context[]>` | `select: string` (context ID) |

**ContextForm Behavior:**
- **Create mode:** Renders title (required), description, kind select (defaults to Project). Emits `NewContext` (omits status/summary).
- **Edit mode:** Adds status select and summary textarea. Emits `UpdateContext` (includes status/summary).
- Validation: Title must be non-empty; submit button disabled if invalid
- CSS: Tailwind dark theme (bg-gray-800, text-gray-100, etc.)

**ContextCard Features:**
- Displays title, kind badge (color-coded per `ContextKindColors`), description, summary (if present)
- Shows last event time using `formatDistanceToNow()` from `date-fns`
- Emits `click` with context ID on card click
- Used as direct child of ContextKanban

**ContextFilterBar Features:**
- Title input (text search, real-time)
- Status select (all statuses + "All statuses" option)
- Kind is not exposed in UI (filter field exists but unused)
- Clear button shown when any filter active
- Uses `watch` to emit updates; no explicit debounce

**ContextKanban Layout:**
- Hardcoded to two columns: Projects (key='project', color blue) | Areas (key='area', color purple)
- Column definitions include `key`, `label`, `color`, `emptyLabel`
- Renders ContextCard for each context in column
- Empty state message per column
- Responsive: `grid-cols-1 md:grid-cols-2`

### Views

| File | Purpose | Key Composables | Key Patterns |
|------|---------|-----------------|---------------|
| **views/ContextBoardView.vue** | List/browse all contexts with filtering and kanban layout | `useContextBoard()`, `useContextStore()` | CRUD store pattern, polling refresh, drawer form pattern |
| **views/ContextDetailView.vue** | Single context detail: events, tags, linked tasks, observations, thread | `useContextDetail()`, `useTagStore()`, `observationService` | Orchestrator composable, multi-store coordination, observation lookups |

**ContextBoardView:**
- Displays title/subtitle with total count
- Filter bar controls which contexts are shown
- Kanban by kind (projects/areas)
- Refresh button force-reloads list
- New Context button opens drawer with `ContextForm` (create mode)
- Click card → navigate to context-detail route with ID
- Loading spinner during fetch; empty state if no contexts

**ContextDetailView:**
- Renders full context detail with context ID from route params
- Edit mode toggles form in-place; cancel reverts to read mode
- Delete button prompts confirmation before removal
- Content sections: Status & Summary, Events (with add form), Tags (with picker), Linked Tasks, Thread, Observations
- Events displayed via `EventTimeline` component; new events via `EventForm`
- Tags displayed via `TagList` component; add via `TagPicker`
- Linked tasks displayed via `TaskCard` component; click navigates to task detail
- Observations loaded via `observationService.queryBySubject('context', contextId)` on mount
- Thread panel via `ThreadPanel` component for context activity history

## Impact Callouts

### ⚠ Context Interface — All Domain Operations

**Affects:**
- **types/context.ts** — Main entity definition
- **services/contextService.ts** — HTTP request/response serialization
- **stores/contextStore.ts** — CRUD state management; computed properties group by `status` and `kind`
- **composables/useContextBoard.ts** — Passes context list to grouping logic
- **composables/useContextDetail.ts** — Loads and displays all context fields
- **components/contexts/ContextCard.vue** — Renders title, kind, description, summary, lastEvent in template
- **components/contexts/ContextForm.vue** — Binds form fields to context properties (title, description, kind, status, summary)
- **views/ContextBoardView.vue** — Displays contexts via composable
- **views/ContextDetailView.vue** — Full context display and edit

**Pattern:** Adding a field requires updates to:
1. Interface definition (context.ts)
2. Form bindings if user-editable (ContextForm.vue)
3. UI display if rendered (ContextCard.vue, ContextDetailView.vue)
4. Computed properties if affects grouping (contextsByStatus, contextsByKind)
5. mapFilter() in contextService.ts if the field is filterable

**Example:** Adding `parentContextId?: string` field (already done):
- Updated `Context`, `NewContext`, `UpdateContext`, `ContextFilter` interfaces
- Added `parent_context_id: f.parentContextId` to contextService.mapFilter()
- Backend accepts `parent_context_id` query param

**Example:** Adding `color: string` field:
- Update `Context` interface
- Add color input to ContextForm (both create/edit modes)
- Pass color to ContextCard, render as border/background
- No store grouping changes needed (doesn't group by color)

### ⚠ ContextStatus Enum — Lifecycle & Filtering

**Affects:**
- **types/enums.ts** — Enum values, labels, colors
- **types/context.ts** — UpdateContext.status field type
- **components/contexts/ContextForm.vue** — Edit mode status select renders all status options
- **components/contexts/ContextFilterBar.vue** — Status filter select options
- **stores/contextStore.ts** — `contextsByStatus` computed iterates hardcoded status keys; fallback: none
- **components/shared/StatusBadge.vue** — Shared component looks up status colors/labels

**Pattern:** Enum values must match across all select options and grouping logic. Adding a 4th status (e.g., 'archived') requires:
1. Add to ContextStatus enum
2. Add label/color to ContextStatusLabels/StatusColors
3. Update ContextForm status select options
4. Update ContextFilterBar status select options
5. Update contextsByStatus computed to initialize `archived: []` group
6. Update StatusBadge if needed for new styling

**Critical:** If status enum changes but select options aren't updated, form can submit invalid values; if grouping isn't updated, new status contexts won't appear in `contextsByStatus`.

### ⚠ ContextKind Enum — Kanban Layout & Grouping

**Affects:**
- **types/enums.ts** — Enum values, ContextKindLabels, ContextKindColors
- **types/context.ts** — Context.kind, NewContext.kind types
- **components/contexts/ContextForm.vue** — Create/edit mode kind select renders Project/Area options
- **components/contexts/ContextCard.vue** — Reads `context.kind`, looks up label in ContextKindLabels, color in ContextKindColors for badge styling
- **components/contexts/ContextKanban.vue** — Hardcoded column definitions with keys 'project', 'area'; only 2 columns rendered
- **stores/contextStore.ts** — `contextsByKind` computed initializes groups; fallback for unknown kind is Project

**Pattern:** Kind is fundamental to layout. Changing existing values (e.g., renaming 'area' to 'zone') breaks hardcoded Kanban column keys. Adding a 3rd kind (e.g., 'goal') requires:
1. Add to ContextKind enum
2. Add label to ContextKindLabels
3. Add color to ContextKindColors
4. Update ContextForm kind select to include new option
5. Update ContextKanban columnDefs to add 3rd column
6. Update contextsByKind computed to initialize new group

**Critical:** Kanban layout hardcodes column keys. Mismatch between enum values and column keys causes contexts to not display.

### ⚠ NewContext Interface — Create Form & Request

**Affects:**
- **components/contexts/ContextForm.vue** — Create mode collects title, description, kind; satisfies NewContext on emit
- **services/contextService.ts** — POST /api/v1/contexts request body type
- **stores/contextStore.ts** — CRUD mixin's create() parameter type

**Pattern:** Create payload is minimal; server assigns id, timestamps, status (default active), summary (empty). Adding optional field to NewContext:
1. Add to interface with optional marker (?)
2. Add form input to ContextForm create mode
3. Handle undefined/null on server

Adding required field breaks backward compatibility and existing creates.

### ⚠ UpdateContext Interface — Edit Form & Request

**Affects:**
- **components/contexts/ContextForm.vue** — Edit mode collects title, description, kind, status, summary; satisfies UpdateContext on emit
- **services/contextService.ts** — PATCH /api/v1/contexts/{id} request body type
- **stores/contextStore.ts** — CRUD mixin's update() parameter type
- **composables/useContextDetail.ts** — update() method parameter type

**Pattern:** Update payload is partial; only specified fields are changed. All fields are optional. Form only includes edit-mode fields (not create-mode defaults). Adding field:
1. Add to interface as optional
2. Add form input to ContextForm edit mode
3. Server validates and applies updates

### ⚠ ContextFilter Interface — Filtering Pipeline

**Affects:**
- **components/contexts/ContextFilterBar.vue** — Filter UI collects status, title; emits ContextFilter
- **stores/contextStore.ts** — `setFilter()` applies via service; `filter` ref holds current state
- **services/contextService.ts** — `mapFilter()` converts interface to query params (status, kind, title → ?status=X&kind=Y&title=Z)

**Pattern:** Bidirectional flow: UI input → ContextFilterBar → `setFilter()` → `store.setFilter()` → service → API. Adding filterable field:
1. Add to ContextFilter interface (optional)
2. Add UI input to ContextFilterBar if desired (not mandatory; can be programmatic-only)
3. Update contextService.mapFilter() to include new field in query params (e.g., `parent_context_id: f.parentContextId`)
4. Backend API must accept and apply the parameter

Removing field breaks filter pipeline; UI won't render, service won't pass param, API doesn't filter.

### ⚠ ContextEvent Interface — Event Threading

**Affects:**
- **types/event.ts** — Event entity definition
- **stores/contextStore.ts** — `events: Ref<ContextEvent[]>` holds thread; `addEvent()` unshifts new events
- **services/contextService.ts** — `listEvents()` return type, `addEvent()` request type
- **views/ContextDetailView.vue** — Passes events to EventTimeline component
- **components/events/EventTimeline.vue** — Renders event content, kind, timestamps

**Pattern:** Events are immutable append-only logs. Changes affect storage/display. Adding field:
1. Update ContextEvent interface
2. EventTimeline component renders if field is included (or update component)
3. Backend populates field on creation

Removing field: Events lose data; components should handle gracefully (e.g., metadata as catch-all).

### ⚠ NewEvent Interface — Event Creation

**Affects:**
- **types/event.ts** — Creation request DTO
- **composables/useContextDetail.ts** — `addEvent(event: NewEvent)` parameter type
- **stores/contextStore.ts** — `addEvent(contextId, event: NewEvent)` parameter type
- **components/events/EventForm.vue** — Form collects kind, content, metadata, sourceId

**Pattern:** NewEvent requires kind and content; metadata and sourceId are optional. Adding required field breaks existing callers. Adding optional field extends event creation without breaking changes.

### ⚠ Computed Properties (contextsByStatus, contextsByKind) — Grouping Logic

**Affects:**
- **stores/contextStore.ts** — Compute contexts grouped by status or kind
- **composables/useContextBoard.ts** — Returns refs to these computed properties
- **views/ContextBoardView.vue** — Passes contextsByKind to ContextKanban

**Pattern:** Hardcoded grouping keys. contextsByStatus iterates `[Active, Paused, Closed]`; contextsByKind iterates `[Project, Area]` with fallback. Changing enum values must update grouping keys in parallel. Example: If ContextStatus.Active becomes 'in-progress', contextsByStatus['active'] returns undefined.

### ⚠ useContextBoard Composable — Polling & Filtering

**Affects:**
- **views/ContextBoardView.vue** — Calls useContextBoard(), uses returned refs and methods
- **stores/contextStore.ts** — setFilter(), fetchList() called by composable

**Pattern:** Assumes store provides contextsByStatus, contextsByKind, count properties. Store changes break composable. Example: If count properties are removed, `activeCount` becomes undefined in template.

### ⚠ useContextDetail Composable — Multi-Store Orchestration

**Affects:**
- **views/ContextDetailView.vue** — Calls useContextDetail(), uses returned refs and methods
- **stores/contextStore.ts** — fetchById(), fetchEvents(), update(), remove(), addEvent()
- **stores/tagStore.ts** — fetchTagsForContext(), addTagToContext(), removeTagFromContext()
- **stores/taskStore.ts** — fetchList() to load tasks, then filters by contextId

**Pattern:** Assumes task domain has `contextId` field, tag domain has context linking methods. Cross-domain changes break composable. Example: If taskStore removes contextId field, linked tasks filtering fails silently.

### ⚠ ContextForm Component — Mode Behavior

**Affects:**
- **components/contexts/ContextForm.vue** — Dual-mode form (create/edit)
- **views/ContextBoardView.vue** — Uses create mode
- **views/ContextDetailView.vue** — Uses edit mode

**Pattern:** Create mode ignores status and summary (server assigns); edit mode includes them. Emits NewContext vs UpdateContext depending on mode. Changing mode logic affects consumers. Example: If create mode adds status field, server-side creation logic must handle/ignore it.

### ⚠ ContextKanban Layout — Hardcoded Columns

**Affects:**
- **components/contexts/ContextKanban.vue** — Renders exactly 2 columns with keys 'project', 'area'
- **views/ContextBoardView.vue** — Passes contextsByKind (grouped by kind) to Kanban

**Pattern:** Column keys hardcoded. If ContextKind enum changes or 3rd kind added, layout breaks. Example: If ContextKind.Area value changes to 'ongoing', column key 'area' won't match, area contexts won't render in Kanban.

**Critical:** Mismatch between enum values and column keys causes silent rendering failure (empty column).

## Cross-Domain Dependencies

### Inbound Dependencies (Domains That Depend on Context)

**Task Domain (taskStore)**
- Tasks have optional `contextId: string` field linking to a parent context
- Used by: useContextDetail filters tasks where `task.contextId === contextId`
- Impact: If taskStore removes contextId field, linked tasks feature breaks
- Query: taskStore.items filtered by contextId; assumes field exists and is string

**Tag Domain (tagStore)**
- Tags can be linked to contexts via tagStore.contextTags mapping
- Methods: `fetchTagsForContext(contextId)`, `addTagToContext(contextId, tagId)`, `removeTagFromContext(contextId, tagId)`
- Used by: useContextDetail loads tags via these methods
- Impact: If tagStore removes these methods, tag management in context detail breaks
- Assumptions: tagStore provides context-scoped tag operations

**Clarification Domain (clarificationStore)**
- Clarification kinds that reference context: context_assignment, new_context, overlapping_contexts, context_debrief
- Impact: Context ID changes affect clarification resolution side-effects
- Example: Resolving 'new_context' clarification links task to newly assigned context

**Other Views & Composables**
- useToday, useSearch, useDashboard, other pages consume contextStore
- Pull context data for scoping/filtering other domains
- Impact: Context store changes ripple across app

### Outbound Dependencies (Context Depends On)

**Shared Components**
- **StatusBadge.vue** — Renders context.status with color/label lookup
  - Depends on: ContextStatusLabels, StatusColors from enums.ts
  - Impact: Status enum changes affect StatusBadge rendering
- **StatusBadge.vue** — Also used for task.status; shared component
  - Context status colors in StatusColors: active='#22c55e', paused='#eab308', closed='#6b7280'

**Shared Composables**
- **usePolling()** — Auto-refresh polling used by useContextBoard
  - Called on mount; triggers store.fetchList(true) at interval
  - Impact: Polling logic changes affect context board refresh behavior
- **useToastStore()** — Toast notifications used by contextStore for all async operations
  - contextStore.create/update/remove/fetchEvents/addEvent all call toast.success() or toast.error()
  - Impact: If useToastStore is removed, users get no feedback on async operations

**Shared Utilities**
- **date-fns** — formatDistanceToNow() used by ContextCard to display lastEvent
  - Impact: Removing date-fns breaks event timestamp display
- **Tailwind CSS** — All components styled with Tailwind classes (bg-gray-800, text-gray-100, etc.)
  - Impact: Tailwind config changes affect component styling

**HTTP Client**
- **services/client.ts** — request() function used by contextService
  - API endpoints: GET /api/v1/contexts, GET /api/v1/contexts/{id}, POST /api/v1/contexts, PATCH /api/v1/contexts/{id}, DELETE /api/v1/contexts/{id}
  - Event endpoints: GET /api/v1/contexts/{id}/events, POST /api/v1/contexts/{id}/events
  - Impact: Backend API changes affect all context operations

**Vue Router**
- **router/index.ts** — Routes: 'contexts' (board), 'context-detail' (detail with :id param)
- Used by: ContextBoardView navigates to context-detail on card click, back button in ContextDetailView
- Impact: Route name changes break navigation

**Other Services (Used by ContextDetailView)**
- **observationService** — queryBySubject('context', contextId) to load observations
- **tagService** — (indirectly via tagStore)
- **taskService** — (indirectly via taskStore)

## Key Patterns

### CRUD Store & Service Factory Pattern

Both `contextStore` and `contextService` use generic factories for boilerplate reduction:

**contextService:**
```typescript
const crud = createCRUDService<Context, NewContext, UpdateContext, ContextFilter>({
  basePath: '/api/v1/contexts',
  mapFilter: (f) => ({
    status: f.status,
    kind: f.kind,
    title: f.title,
    parent_context_id: f.parentContextId,
  }),
})

export const contextService = {
  ...crud,  // create(), update(), remove(), queryAll(), queryByID()
  async listEvents(...) { ... },
  async addEvent(...) { ... },
}
```

Factory provides:
- `create(data: NewContext): Promise<Context>`
- `update(id: string, data: UpdateContext): Promise<Context>`
- `remove(id: string): Promise<void>`
- `queryAll(filter: ContextFilter, page: number, rowsPerPage: number): Promise<QueryResult<Context>>`
- `queryByID(id: string): Promise<Context>`

**contextStore:**
```typescript
const crud = createCRUDStore<Context, NewContext, UpdateContext, ContextFilter>({
  name: 'context',
  service: contextService,
  defaultOrderBy: 'last_event',
  defaultRowsPerPage: 50,
})

export const useContextStore = defineStore('context', () => {
  const { items, total, loading, error, filter, currentItem, ...methods } = crud
  // Extend with domain-specific logic
  const events = ref<ContextEvent[]>([])
  const eventsTotal = ref(0)
  // ... custom methods
  return { ...crud, events, eventsTotal, ... }
})
```

Factory provides state refs and methods; store extends with event management.

### Event Threading Pattern

Events form an append-only thread separate from CRUD operations:

```typescript
// Load events paginated
async function fetchEvents(contextId: string, pg = 1) {
  const result = await contextService.listEvents(contextId, { page: pg, rows: 50 })
  events.value = result.items
  eventsTotal.value = result.total
}

// Add event (prepend, show newest first)
async function addEvent(contextId: string, event: NewEvent) {
  const created = await contextService.addEvent(contextId, event)
  events.value.unshift(created)  // Prepend for chronological display
  eventsTotal.value++
}
```

Separate refs (`events`, `eventsTotal`) from context CRUD state; events are immutable and append-only.

### Reactive Filter Pipeline

Unidirectional data flow from UI to store to service:

```
ContextFilterBar (user input)
  ↓ emit @update
useContextBoard.setFilter()
  ↓ calls
contextStore.setFilter(filter)
  ↓ calls
contextStore.fetchList(true)  // refresh
  ↓ calls
contextService.queryAll(filter, ...)
  ↓ HTTP GET /api/v1/contexts?status=...&title=...
API response
  ↓ updates
contextStore.items
  ↓ reactive
Components re-render
```

Filter is reactive; clearing inputs resets to `{}` (no filters).

### Grouping Computed Properties

Two computed groupers organize contexts by different dimensions:

```typescript
const contextsByStatus = computed(() => {
  const groups: Record<ContextStatus, Context[]> = {
    [ContextStatus.Active]: [],
    [ContextStatus.Paused]: [],
    [ContextStatus.Closed]: [],
  }
  for (const ctx of items.value) {
    groups[ctx.status]?.push(ctx)
  }
  return groups
})

const contextsByKind = computed(() => {
  const groups: Record<ContextKind, Context[]> = {
    [ContextKind.Project]: [],
    [ContextKind.Area]: [],
  }
  for (const ctx of items.value) {
    const bucket = groups[ctx.kind]
    if (bucket) {
      bucket.push(ctx)
    } else {
      groups[ContextKind.Project]!.push(ctx)  // fallback for unknown kind
    }
  }
  return groups
})
```

Both re-compute whenever `items` changes; used by composable and views for grouped rendering.

### Dual-Mode Form Pattern

ContextForm handles both create and edit in single component:

```typescript
// Create mode
if (mode === 'create') {
  emit('submit', {
    title: title.value.trim(),
    description: description.value.trim(),
    kind: kind.value,
  } satisfies NewContext)
}

// Edit mode (includes status and summary)
else {
  emit('submit', {
    title: title.value.trim(),
    description: description.value.trim(),
    kind: kind.value,
    status: status.value,
    summary: summary.value.trim(),
  } satisfies UpdateContext)
}
```

Emits type-specific payloads (NewContext vs UpdateContext) based on mode. Template conditionally renders status/summary fields.

### Kanban Column Hardcoding

ContextKanban uses hardcoded column definitions:

```typescript
const columnDefs = [
  { key: 'project', label: 'Projects', color: 'bg-blue-500', emptyLabel: 'No projects yet' },
  { key: 'area', label: 'Areas', color: 'bg-purple-500', emptyLabel: 'No areas yet' },
]
```

Column keys ('project', 'area') must match `ContextKind` enum values. Layout only renders 2 columns; adding a 3rd kind requires explicit column definition update.

### Multi-Store Orchestration in useContextDetail

Detail view composable coordinates multiple stores on mount:

```typescript
async function load() {
  await Promise.all([
    contextStore.fetchById(contextId),
    contextStore.fetchEvents(contextId),
    tagStore.fetchTagsForContext(contextId),
    taskStore.fetchList(true),  // load all tasks, then filter by contextId
  ])
}
```

Parallel loading minimizes latency; filter tasks by `contextId` on client side.

## File Inventory

### Core Context Files (Non-Test)

| # | Type | File | Purpose |
|---|------|------|---------|
| 1 | Type | types/context.ts | Context, NewContext, UpdateContext, ContextFilter interfaces |
| 2 | Type | types/event.ts | ContextEvent, NewEvent interfaces |
| 3 | Type | types/enums.ts | ContextStatus, ContextKind enums + labels/colors |
| 4 | Type | types/index.ts | Public exports barrel |
| 5 | Service | services/contextService.ts | HTTP API client for CRUD and events |
| 6 | Store | stores/contextStore.ts | Pinia store for context state + events |
| 7 | Composable | composables/useContextBoard.ts | List view orchestrator |
| 8 | Composable | composables/useContextDetail.ts | Detail view orchestrator |
| 9 | Component | components/contexts/ContextForm.vue | Create/edit form |
| 10 | Component | components/contexts/ContextCard.vue | Single context card |
| 11 | Component | components/contexts/ContextFilterBar.vue | Filter UI |
| 12 | Component | components/contexts/ContextKanban.vue | Two-column Kanban layout |
| 13 | View | views/ContextBoardView.vue | List/browse page |
| 14 | View | views/ContextDetailView.vue | Detail page with events, tags, tasks, thread |

**Total Core Files: 14**

### Test Files

| # | File | Covers |
|---|------|--------|
| 1 | __tests__/stores/contextStore.test.ts | Store CRUD, events, grouping, counts |
| 2 | __tests__/services/contextService.test.ts | Service HTTP methods |
| 3 | __tests__/composables/useContextBoard.test.ts | Board composable filtering, refresh, polling |
| 4 | __tests__/composables/useContextDetail.test.ts | Detail composable loading, updates, events, tags, tasks |
| 5 | __tests__/components/contexts/ContextForm.test.ts | Form create/edit modes, validation, emit |
| 6 | __tests__/components/contexts/ContextCard.test.ts | Card rendering, event timestamp, kind badge |
| 7 | __tests__/components/contexts/ContextFilterBar.test.ts | Filter UI interactions, clear, watch |
| 8 | __tests__/components/contexts/ContextKanban.test.ts | Kanban layout, columns, empty states |
| 9 | __tests__/views/ContextBoardView.test.ts | Board page integration: page header, kanban, empty states, new button |
| 10 | __tests__/views/ContextDetailView.test.ts | Detail page integration: page header, loading state, edit toggle, delete confirm, event form, tag picker, task navigation, observations |

**Total Test Files: 10**

**Grand Total: 24 files**

## Testing Notes

Frontend tests use:
- **Unit tests** for composables (useContextBoard, useContextDetail) and services (contextService)
- **Component tests** for all Vue components (ContextForm, ContextCard, ContextFilterBar, ContextKanban)
- **Integration tests** for views (ContextBoardView, ContextDetailView) ensuring composables + components work together

**Key test patterns:**
- Mock contextService for unit tests
- Mock stores via vi.mock() where needed
- Use test factories for sample context objects (see __tests__/helpers/testFactories.ts)
- Test both success and error paths for async operations (fetch, create, update, delete, events)
- Test computed properties (contextsByStatus, contextsByKind, counts)
- Test form validation (ContextForm) and filter interactions (ContextFilterBar)
- Test component rendering with sample data

## Notes

- All components styled with Tailwind CSS dark theme (bg-gray-900, text-gray-100, etc.)
- All async operations show toast notifications for user feedback
- Polling refresh every ~10s on board view (configurable via usePolling)
- Task linking filters by `task.contextId === contextId`; assumes taskStore maintains this field
- Tag linking delegated to tagStore; assumes context-scoped tag methods exist
- Event threads are immutable; new events only (no edit/delete)
