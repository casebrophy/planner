# Frontend Views

Route-level views that aggregate data from multiple domains. Views wire composables, stores, and shared components into full page experiences. CRUD mutations flow through domain stores/composables; views orchestrate layout and navigation.

## Route Map

| Route | View | Nesting |
|-------|------|---------|
| / | redirect → /dashboard | — |
| /dashboard | DashboardView | standalone |
| /today | TodayView | standalone |
| /tasks | TaskBoardView | parent |
| /tasks/:id | TaskDetailView | child of TaskBoardView (DrawerPanel) |
| /contexts | ContextBoardView | standalone |
| /contexts/:id | ContextDetailView | standalone (full page) |
| /notes | NotesBoardView | parent |
| /notes/:id | NoteDetailView | child of NotesBoardView (DrawerPanel) |
| /calendar | CalendarView | standalone |
| /events | EventsView | standalone |
| /habits | HabitsView | standalone |
| /transactions | TransactionBoardView | standalone |
| /ingest-queue | RawInputQueueView | parent |
| /ingest-queue/:id | RawInputDetailView | child of RawInputQueueView (DrawerPanel) |
| /clarifications | ClarificationView | standalone |
| /search | SearchView | standalone |
| /settings | SettingsView | standalone |

> `DailyPlanView.vue` exists in views/ but is **NOT registered in the router**. It duplicates much of TodayView's plan mode — appears to be an older standalone plan page that was superseded.

---

## View File Map

### Planning / Today

#### `views/TodayView.vue`
Route: `/today`  
Composables: `useToday`, `useDailyPlan`  
Components: PageHeader, TaskCard, PlanItemCard, PlanGroupHeader, LoadingSpinner, EmptyState, VueDraggable  
Types: `DailyPlanItem` (from `@/types/dailyPlan`)  
Purpose: **Dual-mode Today view.** When a daily plan exists, renders grouped `PlanItemCard` rows with drag-to-reorder, complete, and dismiss (with inline dismiss modal). When no plan exists, falls back to urgency buckets: Overdue / Due Today / Blocked / Needs Classification using `useToday`. Regenerate button triggers `useDailyPlan.regenerate()`.

#### `views/DailyPlanView.vue`
Route: **unregistered**  
Composables: `useDailyPlan`  
Components: PageHeader, PlanItemCard, PlanGroupHeader, LoadingSpinner, EmptyState, VueDraggable  
Types: `DailyPlanItem`  
Purpose: Standalone daily plan page (superseded by TodayView). Contains identical dismiss modal and reorder logic to TodayView. Safe to delete or consolidate.

#### `views/DashboardView.vue`
Route: `/dashboard`  
Composables: `useDashboard`  
Components: PageHeader, LoadingSpinner  
Purpose: **Weekly health view.** Read-only: 4-week completion trend bar chart (from activity logs), growing backlogs by context (open + blocked counts), repeatedly dismissed tasks, active contexts silent for 14+ days. No inline mutations.

---

### Tasks

#### `views/TaskBoardView.vue`
Route: `/tasks` (parent; child route renders `TaskDetailView` in DrawerPanel)  
Composables: `useTaskBoard`  
Stores: `useContextStore` (fetched on mount for grouped view)  
Services: `taskService.create()`  
Components: PageHeader, TaskCard, TaskFilterBar, TaskForm, DrawerPanel, ClassifyDialog, LoadingSpinner, EmptyState, Pagination  
Types: `NewTask`, `UpdateTask`, `ContextKind`, `ContextKindColors`, `ContextKindLabels`  
Purpose: Main task list. Supports **flat** (paginated) and **grouped-by-context** modes (toggled via button; grouped forces rowsPerPage=100). Grouped view shows context-kind color badge per group. Create new task via DrawerPanel → TaskForm. Classify button opens ClassifyDialog for AI task→context assignment.

#### `views/TaskDetailView.vue`
Route: `/tasks/:id` (child, rendered inside TaskBoardView's DrawerPanel)  
Composables: `useTaskDetail(taskId)`, `useTaskNotes(taskId)`, `useRelatedByContext(contextId, 'task', taskId)`  
Stores: `useTagStore`, `useEntityLinkStore`  
Components: TaskForm, TaskDebriefDialog, TagList, TagPicker, NoteList, NoteForm, ThreadPanel, ConfirmDialog, ActivityLogButton, StreakDisplay, ActivityHistory, LoadingSpinner  
Types: `UpdateTask`, `EntityLink`, `Note`, `NewNote`, `UpdateNote`  
Purpose: Full task detail panel. Sections: metadata (status/priority/energy/dueDate/recurrence), tags, notes (inline create/edit), activity tracking (streak + history + log button), activity thread (ThreadPanel), related items (same-context tasks/notes via `useRelatedByContext`, explicit entity links via `useEntityLinkStore`). On status→done with `trackOutcome=true`, triggers TaskDebriefDialog.

---

### Contexts

#### `views/ContextBoardView.vue`
Route: `/contexts`  
Composables: `useContextBoard`  
Stores: `useContextStore` (for create)  
Components: PageHeader, ContextFilterBar, ContextKanban, ContextForm, DrawerPanel, LoadingSpinner, EmptyState  
Types: `NewContext`, `UpdateContext`  
Purpose: Context kanban board filtered by status/kind. Create via DrawerPanel; on create, navigates to the new context's detail page.

#### `views/ContextDetailView.vue`
Route: `/contexts/:id` (standalone full-page route)  
Composables: `useContextDetail(contextId)`  
Stores: `useTagStore`, `useNoteStore`, `useCalendarEventStore`  
Services: `observationService.queryBySubject()`, `contextService.list()` (sub-contexts), `contextService.create()`, `taskService.create()`  
Components: PageHeader, ContextForm, TaskCard, CalendarEventCard, CalendarEventForm, NoteList, TagList, TagPicker, ThreadPanel, StatusBadge, DrawerPanel, ConfirmDialog, LoadingSpinner  
Types: `UpdateContext`, `Note`, `Context`, `CalendarEvent`, `Task`, `NewTask`, `UpdateTask`, `ContextKind`  
Purpose: **Kind-aware context hub** — renders entirely different layout for Project vs Area:

- **Project hub**: Status + progress bar (done/total tasks), combined timeline (tasks + events sorted by date), collapsible ThreadPanel, sidebar with tags/events/observations/notes. Create task via DrawerPanel.
- **Area hub**: Status summary, sub-projects list (fetched via `contextService.list({parentContextId})`; create sub-project inline), floating tasks list, collapsible thread, sidebar with tags/events/notes.

Both hubs: edit via inline ContextForm, delete with ConfirmDialog, event CRUD via CalendarEventForm drawer, tag management, observations panel (from observationService).

---

### Notes

#### `views/NotesBoardView.vue`
Route: `/notes` (parent; child route renders `NoteDetailView` in DrawerPanel)  
Composables: `useNoteBoard`  
Components: PageHeader, NoteCard, NoteFilterBar, NoteForm, DrawerPanel, LoadingSpinner, EmptyState, Pagination  
Types: `NewNote`, `UpdateNote`  
Purpose: Note list with filtering and pagination. Create via DrawerPanel. Child route (`notes/:id`) rendered inside second DrawerPanel.

#### `views/NoteDetailView.vue`
Route: `/notes/:id` (child of NotesBoardView)  
Composables: `useNoteDetail(noteId)`, `useRelatedByContext(contextId, 'note', noteId)`  
Stores: `useTagStore`, `useContextStore`, `useEntityLinkStore`  
Components: NoteForm, TagList, TagPicker, ThreadPanel, ConfirmDialog, ActivityLogButton, StreakDisplay, ActivityHistory, LoadingSpinner  
Types: `UpdateNote`, `EntityLink`  
Purpose: Full note detail panel. Source badge (manual/voice/email color-coded). Sections: content, metadata (source/context/timestamps), tags, activity tracking (streak + history + log button), ThreadPanel, related items (same-context tasks/notes + explicit entity links). Context name resolved from `contextStore.items`.

---

### Calendar & Events

#### `views/CalendarView.vue`
Route: `/calendar`  
Composables: `useCalendar`  
Components: PageHeader, WeekGrid, TimeBlockForm, DrawerPanel, LoadingSpinner  
Types: `ScheduleItem`, `NewTimeBlock`  
Purpose: Week-view calendar. WeekGrid renders timed + all-day items. Click slot → opens TimeBlockForm drawer with pre-filled start/end. Click unconfirmed time block → confirms it via `confirmTimeBlock()`. Week nav (prev/next/today).

#### `views/EventsView.vue`
Route: `/events`  
Composables: `useEventBoard`  
Components: PageHeader, CalendarEventCard, CalendarEventForm, DrawerPanel, LoadingSpinner, EmptyState  
Types: `NewCalendarEvent`, `UpdateCalendarEvent`  
Purpose: CRUD list for calendar events split into Upcoming/Past sections. Create/edit via single DrawerPanel (edit pre-fills form via `editingEvent`). Delete via `useEventBoard.remove()`.

---

### Habits

#### `views/HabitsView.vue`
Route: `/habits`  
Stores: `useTaskStore` (habits), `useActivityLogStore`  
Components: HabitGrid  
Purpose: Habit tracking grid. Fetches tasks with `taskStore.fetchHabits()` and activity data with `activityLogStore.fetchHabitGrid(habitIds, dayRange)`. Day range toggle: 30d / 90d. HabitGrid renders the heatmap.

---

### Ingest / Admin

#### `views/RawInputQueueView.vue`
Route: `/ingest-queue` (parent; child renders `RawInputDetailView` in DrawerPanel)  
Stores: `useRawInputStore`  
Components: DrawerPanel  
Purpose: Admin ingest queue table. Status filter tabs (All/Pending/Processing/Processed/Failed). Sortable columns (status, created_at). Reprocess button per row. Polls every 15s via `setInterval`. Child route rendered in DrawerPanel.

#### `views/RawInputDetailView.vue`
Route: `/ingest-queue/:id` (child of RawInputQueueView)  
Stores: `useRawInputStore`  
Components: LoadingSpinner  
Types: `StepResult` (from `@/types/rawinput`)  
Purpose: Pipeline step inspector. Renders 6 steps (sanitize → extraction → contextMatch → tasks → events → notes) with status icons (✓/✗/—). Shows step detail counts and errors. Expandable raw content section. Reprocess button.

---

### Utility

#### `views/ClarificationView.vue`
Route: `/clarifications`  
Composables: `useClarification`  
Components: PageHeader, ClarificationSession  
Purpose: AI clarification queue; shows pending count, delegates session interaction to ClarificationSession component.

#### `views/SearchView.vue`
Route: `/search`  
Composables: `useSearch`  
Components: PageHeader, SearchBar, TaskCard, ContextCard, TagBadge, LoadingSpinner, EmptyState  
Purpose: Multi-tab search (All/Tasks/Contexts/Tags). Minimum 2 chars. Result counts per tab. Navigates to task-detail or context-detail on click.

#### `views/TransactionBoardView.vue`
Route: `/transactions`  
Stores: `useTransactionStore`  
Components: PageHeader, TransactionImport, TransactionFilterBar, TransactionRow, LoadingSpinner, EmptyState, Pagination  
Purpose: Financial transaction list. TransactionImport handles CSV upload. Inline review via `store.markReviewed`. Summary subtitle shows total, spend, unreviewed count.

#### `views/SettingsView.vue`
Route: `/settings`  
Composables: `useSettings`, `useServerMonitor`  
Components: PageHeader  
Purpose: Two sections: (1) User preferences — API base URL, polling interval, rows per page, sidebar collapsed; save/reset. (2) Server monitoring (only shown when `available`) — tabbed: containers, inference (session stats, token usage bar, history, tool frequency), logs (per-service selector; sidecar logs show structured entries), Claude instances, timers.

---

## Impact Callouts

### ⚠ `useDailyPlan` composable
Changing its returned shape affects:
- `views/TodayView.vue` — destructures `plan`, `groupedItems`, `taskMap`, `completedCount`, `totalCount`, `generating`, `completeItem`, `dismissItem`, `reorderItem`
- `views/DailyPlanView.vue` — same destructure pattern (unregistered duplicate)

### ⚠ `useToday` composable
- `views/TodayView.vue` — destructures `overdueTasks`, `dueTodayTasks`, `blockedTasks`, `unschedulableTasks`, `contextMap`, `counts`

### ⚠ `useContextDetail` composable
- `views/ContextDetailView.vue` — destructures `context`, `tags`, `linkedTasks`, `loading`, `update`, `remove`, `addTag`, `removeTag`, `reload`; `context.kind` drives the entire Project/Area layout split

### ⚠ `ContextKind` enum (`@/types/enums`)
- `views/ContextDetailView.vue` — `context.kind === ContextKind.Project` / `ContextKind.Area` branches render completely different UIs
- `views/TaskBoardView.vue` — reads `ctx?.kind` for color badge in grouped view using `ContextKindColors` / `ContextKindLabels`

### ⚠ `useRelatedByContext` composable
- `views/TaskDetailView.vue` — `{ tasks: sameContextTasks, notes: sameContextNotes } = useRelatedByContext(computed(() => task.value?.contextId), 'task', taskId)`
- `views/NoteDetailView.vue` — same pattern with `'note'` entity type

### ⚠ `useEntityLinkStore`
- `views/TaskDetailView.vue` — `fetchLinks('task', id)`, `getLinks('task', id)`, `createLink()`, `deleteLink()`
- `views/NoteDetailView.vue` — same pattern with `'note'`

### ⚠ `DailyPlanItem` type (`@/types/dailyPlan`)
- `views/TodayView.vue` — `handleReorder(groupItems: DailyPlanItem[])`, reads `item.userPosition ?? item.position`
- `views/DailyPlanView.vue` — same

### ⚠ `StepResult` type (`@/types/rawinput`)
- `views/RawInputDetailView.vue` — `pipelineSteps` computed reads `r.sanitize`, `r.extraction`, `r.contextMatch`, `r.tasks`, `r.events`, `r.notes` each typed as `StepResult | undefined`

---

## Cross-Domain Dependencies

- **contextStore** used in TaskBoardView (grouped view labels), NoteDetailView (resolves context name from `contextStore.items`)
- **tagStore** used in TaskDetailView, NoteDetailView, ContextDetailView for inline tag management
- **noteStore + calendarEventStore** directly accessed in ContextDetailView (with filter set/cleared in onMounted/onUnmounted)
- **observationService** called in ContextDetailView for context observations
- **activityLogStore** in HabitsView (`fetchHabitGrid`), indirectly via ActivityHistory/StreakDisplay components in TaskDetailView and NoteDetailView
- **ThreadPanel** component used in TaskDetailView, NoteDetailView, ContextDetailView — manages its own threadService calls
- **DrawerPanel pattern**: TaskBoardView, NotesBoardView, RawInputQueueView all use `<router-view />` inside DrawerPanel for nested child routes; `drawerOpen` computed from `!!route.params.id`
- **Shared UI**: all views use PageHeader, LoadingSpinner, EmptyState from `@/components/layout` and `@/components/shared`
