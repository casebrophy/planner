# Shared Frontend Components & Utilities

Cross-cutting utilities and UI components used across all feature domains: generic CRUD factory (service + store), HTTP client, shared composables (polling, pagination, query params, toast, settings), and shared UI components. Every domain store and service is built on top of these primitives.

## Components

### StatusBadge (`components/shared/StatusBadge.vue`)
Props: `status: string`, `type?: 'task' | 'context'`
Purpose: Colored badge with status label and dot indicator; supports both task and context status enums.

### PriorityIndicator (`components/shared/PriorityIndicator.vue`)
Props: `priority: TaskPriority`
Purpose: Displays priority label with colored dot, maps to TaskPriorityLabels enum.

### EnergyIndicator (`components/shared/EnergyIndicator.vue`)
Props: `energy: TaskEnergy` ('low' | 'medium' | 'high')
Purpose: Visual energy requirement indicator with 1-3 bars and label.

### ConfirmDialog (`components/shared/ConfirmDialog.vue`)
Props: `open: boolean`, `title: string`, `message: string`, `confirmLabel?: string`, `cancelLabel?: string`
Emits: `confirm`, `cancel`
Purpose: Teleported modal for destructive action confirmation.

### DrawerPanel (`components/shared/DrawerPanel.vue`)
Props: `open: boolean`, `title?: string`
Emits: `close`
Purpose: Slide-in right drawer with header slot, closes on click-outside.

### Pagination (`components/shared/Pagination.vue`)
Props: `page: number`, `totalPages: number`, `hasNext: boolean`, `hasPrev: boolean`
Emits: `next`, `prev`, `goto(page: number)`
Purpose: Page navigation showing current page and prev/next buttons.

### SearchBar (`components/shared/SearchBar.vue`)
Props: `modelValue: string`, `placeholder?: string`, `debounceMs?: number`
Emits: `update:modelValue`
Purpose: Search input with recent searches saved to localStorage, debounced emit, click-outside close.

### ToastContainer (`components/shared/ToastContainer.vue`)
Purpose: Fixed-position container for toast notifications; uses useToast, auto-dismisses.

### LoadingSpinner (`components/shared/LoadingSpinner.vue`)
Props: `size?: 'sm' | 'md' | 'lg'`
Purpose: Centered animated spinner, default md size.

### EmptyState (`components/shared/EmptyState.vue`)
Props: `title: string`, `message?: string`, `actionLabel?: string`
Emits: `action`
Purpose: Centered empty state with icon, title, optional message and action button.

### NoteItem (`components/shared/NoteItem.vue`)
Props: `entry: ThreadEntry` (id, subjectType, subjectId, kind, content, source, metadata, sourceId, sentiment, requiresAction, createdAt)
Purpose: Displays a single thread entry with source-specific icon, timestamp, and action indicator.

### ThreadPanel (`components/shared/ThreadPanel.vue`)
Props: `subjectType: string`, `subjectId: string`
Purpose: Loads and displays thread entries with kind-specific icon and source color coding.

### ActivityLogButton (`components/shared/ActivityLogButton.vue`)
Props: `subjectType: string`, `subjectId: string`
Purpose: Toggleable button to quickly log an activity entry with optional value.

### ActivityHistory (`components/shared/ActivityHistory.vue`)
Props: `subjectType: string`, `subjectId: string`
Purpose: Loads and displays activity log entries for a subject with relative timestamps; auto-loads on mount.

### StreakDisplay (`components/shared/StreakDisplay.vue`)
Props: `subjectType: string`, `subjectId: string`
Purpose: Shows current/longest/total activity streak; fetches from activityLogStore on mount.

### ProcessingStatus (`components/shared/ProcessingStatus.vue`)
Props: `rawInputId: string`
Purpose: Visual step progress indicator (pending → processing → processed) for raw input queue items.

## Composables

### useToast (`composables/useToast.ts`)
Returns: `{ toasts, success(msg, duration?), error(msg, duration?), info(msg, duration?), dismiss(id) }`
Purpose: Wrapper around toastStore for easy toast notifications with auto-dismiss.

### usePagination (`composables/usePagination.ts`)
Returns: `{ totalPages, hasNextPage, hasPrevPage, nextPage(), prevPage(), goToPage(p) }`
Purpose: Computed pagination state and navigation from page/rowsPerPage/total refs.

### useMediaQuery (`composables/useMediaQuery.ts`)
Returns: `Ref<boolean>` (matches)
Purpose: Reactive media query listener; updates on window.matchMedia changes, cleans up on unmount.

### usePolling (`composables/usePolling.ts`)
Returns: `{ active: Ref<boolean>, start(), stop() }`
Purpose: Polls a function every intervalMs; pauses when document is hidden, immediate fetch on tab re-focus.

### useSearch (`composables/useSearch.ts`)
Returns: `{ query, activeTab, loading, hasSearched, filteredTasks, filteredContexts, filteredTags, totalResults, search(), setTab() }`
Purpose: Multi-tab search across tasks/contexts/tags; 2-char minimum, debounced fetch, tab filtering.

### useQueryParams (`composables/useQueryParams.ts`)
Returns: void (syncs refs to URL params bidirectionally)
Purpose: Bidirectional sync of Vue refs to router query params; initializes from URL on mount.

### useSettings (`composables/useSettings.ts`)
Returns: settings state refs and persist/reset methods
Purpose: Wrapper around settingsStore for user preferences.

### useServerMonitor (`composables/useServerMonitor.ts`)
Returns: server status (containers, inference, logs, Claude instances, timers), polling
Purpose: Admin server monitoring via API polling.

## CRUD Factory Patterns

### createCRUDService (`services/createCRUDService.ts`)
Factory for domain services. Returns: `{ list(params), getById(id), create(item), update(id, item), delete(id) }`
All domain services (taskService, contextService, tagService, etc.) spread this factory and add domain-specific methods.

### createCRUDStore (`stores/createCRUDStore.ts`)
Factory for Pinia stores. State: `items`, `total`, `page`, `rowsPerPage`, `loading`, `error`, `filter`, `orderBy`, `currentItem`, `lastFetchedAt`
Methods: `fetchList`, `fetchById`, `create`, `update`, `remove`, `setFilter`, `setPage`, `setOrder`
Features: 5-minute cache validation; optimistic updates with rollback on error.

## Stores

### toastStore (`stores/toastStore.ts`)
State: `toasts: Toast[]` (id, message, type: 'success'|'error'|'info', duration)
Actions: `add(msg, type, duration)`, `success(msg, duration?)`, `error(msg, duration?)`, `info(msg, duration?)`, `dismiss(id)`
Purpose: Global toast notification management with auto-expiry.

### settingsStore (`stores/settingsStore.ts`)
State: `apiBaseUrl`, `pollingIntervalMs`, `rowsPerPage`, `sidebarCollapsed`
Actions: `persist()`, `reset()`
Purpose: User preferences synced to localStorage; watches all state for auto-persist.

## Impact Callouts

### ⚠ StatusBadge (`components/shared/StatusBadge.vue`)
The `type` prop determines which enum set to render:
- All task detail/board views — pass `type="task"` with TaskStatus values
- All context detail/board views — pass `type="context"` with ContextStatus values
- Adding a new entity type requires updating the component's enum lookup

### ⚠ createCRUDStore state shape
Changing the base store state structure affects ALL domain stores:
- `taskStore`, `contextStore`, `tagStore`, `noteStore`, `clarificationStore`, `timeBlockStore`, `transactionStore`, `activityLogStore`, `calendarEventStore`, `rawinputStore`, `dailyPlanStore` — all extend this factory

### ⚠ createCRUDService return type
Changing the base service interface affects ALL domain services:
- `taskService`, `contextService`, `tagService`, `noteService`, `clarificationService`, `timeBlockService`, `transactionService`, `activityLogService`, `calendarEventService`, `rawinputService`, `dailyPlanService` — all spread this factory

## Cross-Domain Dependencies

- **useToast** → toastStore
- **ToastContainer** → useToast + toastStore
- **ActivityLogButton / ActivityHistory** → activityLogStore (activitylog domain)
- **StreakDisplay** → activityLogStore (activitylog domain)
- **ThreadPanel** → threadService (thread domain)
- **SearchBar** → localStorage directly
- **useSearch** → taskStore + contextStore + tagStore
- **usePolling** → document.visibilitychange API
- **useQueryParams** → Vue Router
- **ProcessingStatus** → request service (rawinput domain)
- **settingsStore.pollingIntervalMs** → consumed by useToday, useTaskBoard, useDashboard for poll intervals
