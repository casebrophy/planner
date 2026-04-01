# Shared Infrastructure (Frontend)

> Cross-cutting utilities: generic CRUD factory (service + store), HTTP client, shared composables (polling, pagination, query params, toast, settings), and shared UI components. Every domain store and service is built on top of these primitives.

## Core Types

```ts
// types/query.ts
interface QueryResult<T> {
  items: T[]
  total: number
  page: number
  rowsPerPage: number
}

interface ListParams {
  page?: number
  rows?: number
  orderBy?: string
}
```

```ts
// types/errors.ts
class ApiError extends Error { status: number; code: string }
class ApiNotFoundError extends ApiError  // status 404, code 'not_found'
class ApiValidationError extends ApiError { fields: Record<string, string> }  // status 400
class ApiNetworkError extends ApiError   // status 0, code 'network_error'
```

```ts
// stores/toastStore.ts
interface Toast {
  id: number
  message: string
  type: 'success' | 'error' | 'info'
  duration: number
}
```

```ts
// stores/settingsStore.ts (internal Settings interface)
interface Settings {
  apiBaseUrl: string        // VITE_API_BASE_URL default
  pollingIntervalMs: number // default 60000
  rowsPerPage: number       // default 20
  sidebarCollapsed: boolean // default false
}
```

```ts
// services/createCRUDService.ts
interface CRUDServiceConfig<TFilter> {
  basePath: string
  mapFilter?: (filter: TFilter) => Record<string, string | number | undefined>
}

interface CRUDService<T, TNew, TUpdate, TFilter> {
  list(params?: ListParams & { filter?: TFilter }): Promise<QueryResult<T>>
  getById(id: string): Promise<T>
  create(item: TNew): Promise<T>
  update(id: string, item: TUpdate): Promise<T>
  delete(id: string): Promise<void>
}
```

```ts
// stores/createCRUDStore.ts
interface CRUDStoreConfig<T, TNew, TUpdate, TFilter> {
  name: string
  service: CRUDService<T, TNew, TUpdate, TFilter>
  defaultOrderBy?: string
  defaultRowsPerPage?: number
}
// Returns: { items, total, page, rowsPerPage, loading, error, filter, orderBy, currentItem,
//            fetchList, fetchById, create, update, remove, setFilter, setPage, setOrder }
// CACHE_TTL = 5 minutes; cache key = JSON(filter + orderBy + page)
// update/remove use optimistic updates with rollback on error
```

## File Map

### Services
- `services/client.ts` — **request\<T\>** — base HTTP client; reads VITE_API_BASE_URL + VITE_API_KEY env vars; maps 404 → ApiNotFoundError, 400 → ApiValidationError, network failure → ApiNetworkError; handles 204 (returns undefined)
- `services/createCRUDService.ts` — **createCRUDService** — factory for standard CRUD services; uses `request` internally; `mapFilter` converts camelCase filter fields to snake_case query params

### Stores
- `stores/createCRUDStore.ts` — **createCRUDStore** — factory for standard CRUD stores; 5-minute TTL cache per (filter+orderBy+page) key; optimistic update/remove with rollback; delegates all API calls to CRUDService
- `stores/toastStore.ts` — **useToastStore** — global toast queue; auto-dismisses after duration (success: 4s, error: 6s); used by createCRUDStore and every domain store for user feedback
- `stores/settingsStore.ts` — **useSettingsStore** — persists app settings to localStorage under key 'planner-settings'; watchers auto-save on any change; `reset()` restores defaults

### Composables
- `composables/usePolling.ts` — **usePolling(fn, intervalMs=60000)** — setInterval-based background refresh; pauses when tab is hidden (document.hidden); immediate re-fetch on tab re-focus via visibilitychange; auto-cleanup on unmount
- `composables/usePagination.ts` — **usePagination(page, rowsPerPage, total)** — computed totalPages/hasNextPage/hasPrevPage; nextPage/prevPage/goToPage mutate the passed page Ref directly
- `composables/useQueryParams.ts` — **useQueryParams(params, onUpdate?)** — initializes Refs from URL query params on mount; watches Refs and syncs back to URL via router.replace
- `composables/useSettings.ts` — **useSettings** — thin composable wrapping settingsStore; adds `saved` flash state for save confirmation UI
- `composables/useToast.ts` — thin composable wrapping useToastStore (if present); toasts are typically called directly via store in domain code

### Components
- `components/shared/ConfirmDialog.vue` — **ConfirmDialog** — modal confirmation dialog; emits confirm/cancel
- `components/shared/DrawerPanel.vue` — **DrawerPanel** — slide-in panel for detail views on mobile
- `components/shared/EmptyState.vue` — **EmptyState** — placeholder shown when a list is empty
- `components/shared/EnergyIndicator.vue` — **EnergyIndicator** — visual badge for TaskEnergy values
- `components/shared/LoadingSpinner.vue` — **LoadingSpinner** — spinner overlay
- `components/shared/NoteItem.vue` — **NoteItem** — renders a single note/observation entry
- `components/shared/Pagination.vue` — **Pagination** — page controls; uses usePagination internally or accepts page/total props
- `components/shared/PriorityIndicator.vue` — **PriorityIndicator** — color-coded badge for TaskPriority; uses PriorityColors map
- `components/shared/ProcessingStatus.vue` — **ProcessingStatus** — shows async processing state (e.g. for raw input ingestion)
- `components/shared/SearchBar.vue` — **SearchBar** — controlled input with debounce; emits search event
- `components/shared/StatusBadge.vue` — **StatusBadge** — color-coded badge for any status string; uses StatusColors map
- `components/shared/ThreadPanel.vue` — **ThreadPanel** — shows thread entries for a subject (task or context)
- `components/shared/ToastContainer.vue` — **ToastContainer** — renders active toasts from toastStore; positioned fixed overlay
- `components/layout/AppShell.vue` — **AppShell** — root shell component; renders AppSidebar + router-view
- `components/layout/AppSidebar.vue` — **AppSidebar** — navigation sidebar; polls clarificationStore.fetchPendingCount every 60s; reads settingsStore.sidebarCollapsed
- `components/layout/PageHeader.vue` — **PageHeader** — per-view header with title + action slots

## Impact Callouts

### ⚠ CRUDService interface (services/createCRUDService.ts)
Changing the interface shape or method signatures affects every domain service:
- `services/taskService.ts`, `contextService.ts`, `tagService.ts`, `transactionService.ts` — all use createCRUDService; contextService and tagService extend the returned object with extra methods
- `stores/createCRUDStore.ts` — calls service.list, .getById, .create, .update, .delete with specific signatures

### ⚠ createCRUDStore return shape (stores/createCRUDStore.ts)
Changing what createCRUDStore returns affects every domain store:
- `stores/taskStore.ts`, `contextStore.ts`, `tagStore.ts`, `transactionStore.ts` — all spread `...crud` into their return; field renames break store consumers

### ⚠ QueryResult\<T\> (types/query.ts)
Changing this interface affects:
- `services/createCRUDService.ts` — `.list()` returns `Promise<QueryResult<T>>`
- `services/contextService.ts` — listEvents returns `QueryResult<ContextEvent>`
- `services/tagService.ts` — getByTask/getByContext returns `QueryResult<Tag>`
- `services/clarificationService.ts` — queryQueue maps to QueryResult shape manually
- `stores/createCRUDStore.ts` — destructures .items and .total from result

### ⚠ Settings interface (stores/settingsStore.ts)
Changing settings fields affects:
- `composables/usePolling.ts` — reads pollingIntervalMs (passed as arg from consumers, not directly from store)
- `composables/useSettings.ts` — exposes all four settings fields
- `services/client.ts` — reads VITE_API_BASE_URL at module load time (not reactive to runtime changes)
- `components/layout/AppSidebar.vue` — reads sidebarCollapsed

### ⚠ Toast interface (stores/toastStore.ts)
Changing Toast shape affects:
- `components/shared/ToastContainer.vue` — renders toasts array; binds .message, .type, .id
- All domain stores that call toast.success/error/info

## Cross-Domain Dependencies

- All domain stores depend on `createCRUDStore` and `useToastStore`
- All domain services depend on `createCRUDService` and `client.request`
- `usePolling` is used by `useTaskBoard`, `useContextBoard`, `useDashboard`, `useToday`
- `usePagination` is used by `useTaskBoard`
- `useQueryParams` is available for views that want URL-synced filter state
- `components/layout/AppSidebar.vue` is the global poller for clarification pending count
