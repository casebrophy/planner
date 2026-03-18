# Phase 4: Frontend Web Shell — Implementation Plan

## Summary

Vue 3 web application providing complete visibility into and management of the planner system. Covers tasks, contexts, context events, tags. Dark-mode-only, touch-first shared component library, kanban context board, nested routes.

**Dependencies:** Go backend running with existing APIs (tasks, contexts, tags, events).

## Decisions

| Question | Decision |
|----------|----------|
| CSS framework | Tailwind CSS 4 |
| Theme | Dark mode only (no light toggle) |
| Date formatting | date-fns (relative time on timelines/cards) |
| State persistence | localStorage for filter state + sidebar collapse |
| Context board | Kanban columns (grouped by status) |
| Form validation | Hand-rolled |
| Testing | Vitest (unit: services, stores, composables) + Playwright (e2e) |
| Component design | Props-heavy unless flexibility demands slots |
| Routing | Nested (task detail renders within task board layout) |
| Polling | 60s interval while tab focused, pause on blur |
| WebSockets | Not needed — polling is sufficient |

## Frontend Architecture — Layer Stack

```
Types       → matches backend DTOs exactly (no logic)
Service     → fetch calls, response marshalling, typed errors (tested)
Store       → orchestrates services, caches with 5min TTL, optimistic CRUD (tested)
Composable  → composes store data for specific view needs (tested)
Component   → props-heavy, renders composable output
```

### Error Handling

Typed error hierarchy in service layer:
```ts
class ApiError extends Error {
  status: number
  code: string
}
class ApiNotFoundError extends ApiError {}       // 404
class ApiValidationError extends ApiError {      // 400
  fields: Record<string, string>
}
class ApiNetworkError extends ApiError {}        // fetch failures
```

Stores catch specific types for different behavior. `useToast` composable surfaces errors as toast notifications.

### Cache Strategy

- 5-minute TTL for all data
- Optimistic CRUD: update local state immediately, rollback on API failure
- Stores track `lastFetchedAt` per query key, skip re-fetch if within TTL
- Manual refresh always bypasses cache

## Tech Stack

- Vue 3.5+ (Composition API, `<script setup>`)
- TypeScript 5.x (strict mode)
- Vite 6.x + `@vitejs/plugin-vue`
- Pinia 2.x
- vue-router 4.x (nested routes)
- Tailwind CSS 4 + `@tailwindcss/vite`
- date-fns
- Vitest + @vue/test-utils
- Playwright

## Directory Structure

```
web/
  index.html
  vite.config.ts
  tsconfig.json
  .env
  src/
    main.ts
    App.vue
    router/
      index.ts                 — route definitions with nesting
    types/
      enums.ts                 — TaskStatus, TaskPriority, TaskEnergy, ContextStatus (as const pattern)
      task.ts                  — Task, NewTask, UpdateTask
      context.ts               — Context, NewContext, UpdateContext
      event.ts                 — ContextEvent, NewEvent
      tag.ts                   — Tag, NewTag
      query.ts                 — QueryResult<T>, ListParams, OrderParams
      errors.ts                — ApiError, ApiNotFoundError, ApiValidationError, ApiNetworkError
    services/
      client.ts                — fetch wrapper (auth, error mapping, query params)
      taskService.ts           — Task API calls (tested)
      contextService.ts        — Context + Event API calls (tested)
      tagService.ts            — Tag API calls (tested)
    stores/
      taskStore.ts             — useTaskStore (tested)
      contextStore.ts          — useContextStore (tested)
      tagStore.ts              — useTagStore (tested)
      captureStore.ts          — useCaptureStore
      toastStore.ts            — useToastStore
    composables/
      useTaskBoard.ts          — composes task data for board view (tested)
      useTaskDetail.ts         — composes task + tags + context for detail (tested)
      useContextBoard.ts       — composes context data grouped by status (tested)
      useContextDetail.ts      — composes context + events + linked tasks (tested)
      useDashboard.ts          — composes summary counts + recent activity (tested)
      useCapture.ts            — composes capture form logic (tested)
      usePagination.ts         — shared pagination logic
      useQueryParams.ts        — sync URL query params with filter state
      usePolling.ts            — 60s poll with tab visibility awareness
      useToast.ts              — toast notification composable
    views/
      DashboardView.vue
      TaskBoardView.vue
      TaskDetailView.vue       — renders within TaskBoardView (nested route)
      ContextBoardView.vue
      ContextDetailView.vue
      CaptureView.vue
    components/
      layout/
        AppShell.vue           — sidebar + main content
        AppSidebar.vue         — nav links, collapsible
        PageHeader.vue         — title + action buttons
      tasks/
        TaskCard.vue           — compact card for list
        TaskForm.vue           — create/edit form
        TaskFilterBar.vue      — status, priority, context, due date filters
      contexts/
        ContextCard.vue        — compact card for list
        ContextForm.vue        — create/edit form
        ContextFilterBar.vue   — status, title filters
        ContextKanban.vue      — three-column kanban (active, paused, closed)
      events/
        EventTimeline.vue      — vertical timeline container
        EventTimelineItem.vue  — single event entry
        EventForm.vue          — add event form
      tags/
        TagBadge.vue           — single tag pill
        TagList.vue            — horizontal tag list
        TagPicker.vue          — search/select/create tags
      shared/
        StatusBadge.vue        — colored badge by status
        PriorityIndicator.vue  — colored dot/icon by priority
        EnergyIndicator.vue    — energy level indicator
        Pagination.vue         — page controls
        EmptyState.vue         — "no items" placeholder
        LoadingSpinner.vue     — loading indicator
        ConfirmDialog.vue      — delete confirmation modal
        DrawerPanel.vue        — slide-in panel (task detail within board)
        ToastContainer.vue     — toast notification display
    __tests__/
      services/               — service layer tests
      stores/                 — store tests
      composables/            — composable tests
    e2e/                      — Playwright tests
```

## TypeScript Types

### Enums (as const pattern)

```ts
export const TaskStatus = {
  Todo: 'todo',
  InProgress: 'in_progress',
  Done: 'done',
  Cancelled: 'cancelled',
} as const
export type TaskStatus = (typeof TaskStatus)[keyof typeof TaskStatus]

export const TaskPriority = { Low: 'low', Medium: 'medium', High: 'high', Urgent: 'urgent' } as const
export type TaskPriority = (typeof TaskPriority)[keyof typeof TaskPriority]

export const TaskEnergy = { Low: 'low', Medium: 'medium', High: 'high' } as const
export type TaskEnergy = (typeof TaskEnergy)[keyof typeof TaskEnergy]

export const ContextStatus = { Active: 'active', Paused: 'paused', Closed: 'closed' } as const
export type ContextStatus = (typeof ContextStatus)[keyof typeof ContextStatus]
```

### Domain Types

Match Go app-layer JSON DTOs exactly. See `types/task.ts`, `types/context.ts`, `types/event.ts`, `types/tag.ts` in directory structure above.

### Query Types

```ts
export interface QueryResult<T> {
  items: T[]
  total: number
  page: number
  rowsPerPage: number
}

export interface ListParams {
  page?: number
  rows?: number
  orderBy?: string
}
```

## Service Layer

Thin fetch wrapper in `client.ts`. Domain services (`taskService.ts`, etc.) use client and return typed responses.

```ts
// client.ts — key features:
// - Base URL from VITE_API_BASE_URL (empty in dev, Vite proxies /api)
// - X-API-Key header from VITE_API_KEY
// - Response → typed error mapping (404 → ApiNotFoundError, 400 → ApiValidationError, network → ApiNetworkError)
// - Query params: undefined values silently dropped
```

Each service is an object of functions, not a class. Easily mockable for store tests.

## Pinia Stores

### Common pattern

```ts
interface StoreState<T> {
  items: T[]
  total: number
  page: number
  rowsPerPage: number
  loading: boolean
  error: string | null
  lastFetchedAt: Record<string, number>  // query key → timestamp for TTL
  filter: Record<string, unknown>
  orderBy: string
}
```

### taskStore

- State: tasks, currentTask, filter (status, priority, contextId, dueDate range), orderBy
- Actions: fetchTasks, fetchTask, createTask (optimistic), updateTask (optimistic), deleteTask (optimistic), setFilter, setPage, setOrder
- Getters: tasksByStatus (for potential board views), hasActiveFilter, overdueCount

### contextStore

- State: contexts, currentContext, events, eventsTotal, filter (status, title), orderBy
- Actions: fetchContexts, fetchContext, createContext, updateContext, deleteContext, fetchEvents, addEvent
- Getters: contextsByStatus (for kanban), activeCount, pausedCount, closedCount

### tagStore

- State: tags
- Actions: fetchTags, createTag, deleteTag, addTagToTask, removeTagFromTask, addTagToContext, removeTagFromContext, fetchTagsForTask, fetchTagsForContext

### toastStore

- State: toasts (array of {id, message, type, duration})
- Actions: success, error, info, dismiss

## Composables

Each composable orchestrates stores for a specific view:

- `useTaskBoard` — fetches tasks, manages filter/sort state, provides paginated list, handles URL query param sync
- `useTaskDetail(id)` — fetches single task + tags, provides update/delete handlers
- `useContextBoard` — fetches contexts grouped by status for kanban, filter state
- `useContextDetail(id)` — fetches context + events + linked tasks + tags
- `useDashboard` — fetches summary counts, recent tasks, active contexts, overdue tasks
- `useCapture` — manages capture form state, submit handler
- `usePolling(fn, interval)` — calls fn every interval ms, pauses when tab not visible
- `useToast` — wraps toastStore for components

## Routes (nested)

```ts
const routes = [
  { path: '/', redirect: '/dashboard' },
  { path: '/dashboard', name: 'dashboard', component: DashboardView },
  {
    path: '/tasks',
    name: 'tasks',
    component: TaskBoardView,
    children: [
      { path: ':id', name: 'task-detail', component: TaskDetailView }
    ]
  },
  {
    path: '/contexts',
    name: 'contexts',
    component: ContextBoardView,
    children: [
      { path: ':id', name: 'context-detail', component: ContextDetailView }
    ]
  },
  { path: '/capture', name: 'capture', component: CaptureView },
]
```

Task detail renders in a `DrawerPanel` inside the TaskBoardView. Context detail is full-page (more content to show).

## Layout

```
+------------------+----------------------------------------+
|   AppSidebar     |   PageHeader                           |
|   (240px, dark)  |----------------------------------------|
|                  |                                        |
|  [Dashboard]     |   Main content (router-view)           |
|  [Tasks]         |                                        |
|  [Contexts]      |   +-- DrawerPanel (task detail) ──+    |
|  [Capture]       |   |                               |    |
|                  |   +───────────────────────────────+    |
+------------------+----------------------------------------+
```

Responsive: sidebar collapses to icon-only at 768px, hamburger below 640px. Sidebar collapse state persisted to localStorage.

## Tailwind Theme (dark only)

```ts
colors: {
  status: {
    todo: '#6b7280',
    'in-progress': '#3b82f6',
    done: '#22c55e',
    cancelled: '#ef4444',
    active: '#22c55e',
    paused: '#eab308',
    closed: '#6b7280',
  },
  priority: {
    low: '#6b7280',
    medium: '#3b82f6',
    high: '#f97316',
    urgent: '#ef4444',
  },
}
```

Background: `bg-gray-950`. Cards: `bg-gray-900`. Borders: `border-gray-800`. Text: `text-gray-100` / `text-gray-400`.

## Docker Integration

### Development

`make web-dev` runs Vite dev server on :5173. Proxy `/api` → `http://localhost:8080`.

### Production

`zarf/docker/Dockerfile.frontend`:
- Build: `node:22-alpine`, `npm ci`, `npm run build`
- Serve: `nginx:alpine`, copy `dist/`, proxy `/api` to backend

```yaml
# docker-compose.yml
frontend:
  build:
    context: ../..
    dockerfile: zarf/docker/Dockerfile.frontend
  ports:
    - "127.0.0.1:5173:80"
  depends_on:
    - backend
```

## Implementation Order

### Foundation (Steps 1-3)

1. **Project scaffolding** — Vite + Vue 3 + TS + Pinia + vue-router + Tailwind 4 + date-fns + Vitest + Playwright. ESLint + Prettier config. Vite proxy. Verify: `npm run lint && npm run build`
2. **TypeScript types** — all files in `src/types/` (enums, task, context, event, tag, query, errors)
3. **Service layer + tests** — `client.ts`, `taskService.ts`, `contextService.ts`, `tagService.ts`. Unit tests with mocked fetch.

### Data Layer (Steps 4-6)

4. **Pinia stores + tests** — taskStore, contextStore, tagStore, toastStore. Unit tests with mocked services.
5. **Shared composables** — usePagination, useQueryParams, usePolling, useToast
6. **View composables + tests** — useTaskBoard, useTaskDetail, useContextBoard, useContextDetail, useDashboard, useCapture

### Skeleton (Steps 7-8)

7. **Shell layout** — AppShell, AppSidebar, PageHeader. Router with all routes (stub views). Dark theme base styles. Sidebar collapse + localStorage persistence.
8. **Shared components** — StatusBadge, PriorityIndicator, EnergyIndicator, Pagination, EmptyState, LoadingSpinner, ConfirmDialog, DrawerPanel, ToastContainer

### Views (Steps 9-13)

9. **Task board** — TaskCard, TaskFilterBar, TaskBoardView. Filters synced to URL params. Pagination.
10. **Task detail** — TaskForm, TaskDetailView (nested route, renders in DrawerPanel). Full CRUD. Tag management with TagBadge, TagList, TagPicker.
11. **Context board** — ContextCard, ContextFilterBar, ContextKanban, ContextBoardView. Kanban columns by status.
12. **Context detail** — EventTimeline, EventTimelineItem, EventForm, ContextForm, ContextDetailView. Events + linked tasks + tags.
13. **Dashboard + Capture** — DashboardView (summary cards, recent activity, overdue). CaptureView (quick create toggle).

### Polish (Steps 14-15)

14. **Polling + error handling** — Wire usePolling into board views. Error toasts on API failures. Loading states. Empty states.
15. **Docker + e2e tests** — Dockerfile.frontend, nginx config, docker-compose update. Playwright e2e tests for core flows (create task, view context, capture).

## Makefile Additions (repo root)

```makefile
web-dev:
	cd web && npm run dev

web-build:
	cd web && npm run build

web-lint:
	cd web && npm run lint && npm run build

web-test:
	cd web && npm run test

web-e2e:
	cd web && npx playwright test
```
