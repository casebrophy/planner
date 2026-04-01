# Cross-Cutting Views (Frontend)

> Read-only views that aggregate data from multiple domains without owning any domain entities: Dashboard (summary stats + recent items), Today (time-scoped task buckets), Search (client-side fuzzy search across tasks/contexts/tags), and Settings (app configuration). None of these views create or mutate domain data.

## File Map

### Composables

- `composables/useDashboard.ts` — **useDashboard** — loads tasks + contexts in parallel on mount; computes `taskCounts` (total/todo/inProgress/done/overdue), `contextCounts` (total/active/paused/closed), `recentTasks` (5 most recently updated), `overdueTasks`, `activeContexts`; polls via usePolling

- `composables/useToday.ts` — **useToday** — loads tasks + contexts on mount; computes `overdueTasks` (dueDate before today, not done/cancelled), `dueTodayTasks` (dueDate = today), `inProgressTasks`; builds `contextMap` (id→title for task display); `counts` summary; polls via usePolling

- `composables/useSearch.ts` — **useSearch** — client-side search (no dedicated API endpoint); watches `query` ref with 300ms debounce; triggers fetch of all tasks/contexts/tags from their stores when query ≥ 2 chars; computes `filteredTasks` (matches title+description), `filteredContexts` (matches title+description), `filteredTags` (matches name); `activeTab` for all/tasks/contexts/tags result tabs

- `composables/useSettings.ts` — **useSettings** — wraps settingsStore; exposes apiBaseUrl, pollingIntervalMs, rowsPerPage, sidebarCollapsed; `save()` sets a 2s `saved` flash ref; `reset()` delegates to settingsStore.reset()

### Views

- `views/DashboardView.vue` — **DashboardView** — uses useDashboard; renders count cards for tasks/contexts, overdue task list, active context list, recent tasks
- `views/TodayView.vue` — **TodayView** — uses useToday; renders three sections: overdue, due today, in progress; shows context title via contextMap
- `views/SearchView.vue` — **SearchView** — uses useSearch; renders SearchBar + tab switcher + result lists for tasks/contexts/tags
- `views/SettingsView.vue` — **SettingsView** — uses useSettings; form for each settings field with save/reset actions

## Impact Callouts

### ⚠ Task.status / Task.dueDate (types/task.ts)
These fields drive all time-scoped views:
- `composables/useDashboard.ts` — counts by TaskStatus values; overdueTasks checks dueDate < now AND status not Done/Cancelled
- `composables/useToday.ts` — overdueTasks checks dueDate < startOfToday; dueTodayTasks checks dueDate within today's range; inProgressTasks checks status = InProgress
- Both composables import `TaskStatus` enum directly and compare string values

### ⚠ Context.status / Context.id+title (types/context.ts)
- `composables/useDashboard.ts` — contextCounts groups by ContextStatus values; activeContexts filters by ContextStatus.Active
- `composables/useToday.ts` — contextMap builds id→title lookup; if Context.id or .title renames, task display breaks

### ⚠ Task.title / Task.description / Context.title / Context.description / Tag.name (search targets)
- `composables/useSearch.ts` — all three filtered computed values match on these exact field names; field renames silently break search results

### ⚠ Settings fields (stores/settingsStore.ts)
- `composables/useSettings.ts` — exposes all four fields as reactive refs; adding a new setting requires updating the Settings interface, defaults object, persist function, and useSettings composable
- `composables/usePolling.ts` — pollingIntervalMs is passed as intervalMs argument; not read reactively from store (polling interval only changes if consumer re-creates the composable)

## Cross-Domain Dependencies

- `stores/taskStore.ts` — useDashboard, useToday, useSearch all read taskStore.items
- `stores/contextStore.ts` — useDashboard, useToday, useSearch all read contextStore.items
- `stores/tagStore.ts` — useSearch reads tagStore.items
- `composables/usePolling.ts` — used by useDashboard and useToday for background refresh
- `stores/settingsStore.ts` — useSettings is a thin wrapper; SettingsView is the only mutation path for stored settings
