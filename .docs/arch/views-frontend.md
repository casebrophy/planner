# Frontend Views

Route-level views that aggregate data from multiple domains. Views wire composables, stores, and shared components into full page experiences. Most are read/aggregate views; CRUD mutations flow through domain stores.

## Views

### TodayView (`views/TodayView.vue`)
Route: `/today`
Uses: `useToday`, TaskCard, LoadingSpinner, EmptyState
Purpose: Grouped task dashboard — overdue / due-today / blocked sections with counts and context labels per task.

### DashboardView (`views/DashboardView.vue`)
Route: `/dashboard`
Uses: `useDashboard`, TaskCard, ContextCard, LoadingSpinner, EmptyState
Purpose: Overview with task/context summary stats, recent tasks, overdue tasks, active contexts; auto-refreshes.

### CaptureView (`views/CaptureView.vue`)
Route: `/capture`
Uses: `useCapture`
Purpose: Quick task/context creation with mode toggle, title/description/priority/energy fields, form validation.

### SearchView (`views/SearchView.vue`)
Route: `/search`
Uses: `useSearch`, SearchBar, TaskCard, ContextCard, TagBadge, LoadingSpinner, EmptyState
Purpose: Multi-tab search (all/tasks/contexts/tags) across domains; 2-char minimum, result counts.

### SettingsView (`views/SettingsView.vue`)
Route: `/settings`
Uses: `useSettings`, `useServerMonitor`
Purpose: User preferences (API URL, polling interval, rows per page, sidebar), server monitoring (containers, inference, logs, Claude instances, timers).

### RawInputQueueView (`views/RawInputQueueView.vue`)
Route: `/ingest-queue`
Uses: rawinputStore, ProcessingStatus
Purpose: Admin view for ingest queue — status filter, retry/reprocess buttons, 15s poll interval.

### CalendarView (`views/CalendarView.vue`)
Route: `/calendar`
Uses: `useCalendar`, DrawerPanel, TimeBlockForm
Purpose: Week-based calendar grid with time block creation; click-to-create slots, time block confirmation.

### ClarificationView (`views/ClarificationView.vue`)
Route: `/clarifications`
Uses: `useClarification`, ClarificationSession
Purpose: AI clarification queue for ambiguous inputs; displays pending count, session-based interaction.

### EventsView (`views/EventsView.vue`)
Route: `/events`
Uses: `useEventBoard`, calendarEventStore, CalendarEventCard, CalendarEventForm, DrawerPanel
Purpose: CRUD for calendar events (appointments/trips); split into upcoming/past sections.

### TaskBoardView (`views/TaskBoardView.vue`)
Route: `/tasks`
Uses: `useTaskBoard`, TaskCard, TaskForm, TaskFilterBar, ClassifyDialog, DrawerPanel, Pagination
Purpose: Main task list with filtering/sorting/pagination, create/edit drawers, classify action.

### TaskDetailView (`views/TaskDetailView.vue`)
Route: `/tasks/:id`
Uses: `useTaskDetail`, TaskForm, TagPicker, TagList, ThreadPanel, StreakDisplay, ActivityLogButton, ActivityHistory
Purpose: Full task detail — metadata, edit/delete, tag management, thread, recurrence parent link.

### ContextBoardView (`views/ContextBoardView.vue`)
Route: `/contexts`
Uses: `useContextBoard`, ContextCard, ContextForm, DrawerPanel, Pagination
Purpose: Context board with filtering by status/kind, create/edit drawers.

### ContextDetailView (`views/ContextDetailView.vue`)
Route: `/contexts/:id`
Uses: `useContextDetail`, ContextForm, TagPicker, TagList, ThreadPanel, ActivityHistory
Purpose: Full context detail — metadata, events timeline, tag management, debrief status.

### NotesBoardView (`views/NotesBoardView.vue`)
Route: `/notes`
Uses: `useNoteBoard`, NoteCard, NoteForm, DrawerPanel, Pagination
Purpose: Note list with filtering, create/edit drawers.

### NoteDetailView (`views/NoteDetailView.vue`)
Route: `/notes/:id`
Uses: `useNoteDetail`, NoteForm, TagPicker, TagList, ThreadPanel
Purpose: Full note detail — content, tag management, thread.

### TransactionBoardView (`views/TransactionBoardView.vue`)
Route: `/transactions`
Uses: transactionStore, TransactionCard, TransactionForm, DrawerPanel, Pagination
Purpose: Transaction list with date/category filtering, create/edit drawers.

### DailyPlanView (`views/DailyPlanView.vue`)
Route: `/plan`
Uses: `useDailyPlan`, dailyPlanStore, TaskCard, TimeBlockForm
Purpose: Daily plan generation and time block management for the current day.

## Route Map

| Route | View | Auth |
|-------|------|------|
| / → redirect | DashboardView | No |
| /dashboard | DashboardView | No |
| /today | TodayView | No |
| /plan | DailyPlanView | No |
| /events | EventsView | No |
| /calendar | CalendarView | No |
| /tasks | TaskBoardView | No |
| /tasks/:id | TaskDetailView | No |
| /contexts | ContextBoardView | No |
| /contexts/:id | ContextDetailView | No |
| /notes | NotesBoardView | No |
| /notes/:id | NoteDetailView | No |
| /transactions | TransactionBoardView | No |
| /ingest-queue | RawInputQueueView | No |
| /capture | CaptureView | No |
| /clarifications | ClarificationView | No |
| /search | SearchView | No |
| /settings | SettingsView | No |

## View Composable Dependencies

| View | Primary Composable | Stores Used | Key Services |
|------|--------------------|-------------|--------------|
| TodayView | useToday | taskStore, contextStore | — |
| DashboardView | useDashboard | taskStore, contextStore | — |
| CaptureView | useCapture | taskStore, contextStore | — |
| SearchView | useSearch | taskStore, contextStore, tagStore | — |
| SettingsView | useSettings, useServerMonitor | settingsStore | server monitoring API |
| RawInputQueueView | — | rawinputStore | rawinputService |
| CalendarView | useCalendar | taskStore, timeBlockStore | calendarEventService, timeBlockService |
| ClarificationView | useClarification | clarificationStore | clarificationService |
| EventsView | useEventBoard | calendarEventStore | calendarEventService |
| TaskBoardView | useTaskBoard | taskStore | taskService |
| TaskDetailView | useTaskDetail | taskStore, tagStore | taskService, threadService |
| ContextBoardView | useContextBoard | contextStore | contextService |
| ContextDetailView | useContextDetail | contextStore, tagStore | contextService, threadService |
| NotesBoardView | useNoteBoard | noteStore | noteService |
| NoteDetailView | useNoteDetail | noteStore, tagStore | noteService, threadService |
| TransactionBoardView | — | transactionStore | transactionService |
| DailyPlanView | useDailyPlan | dailyPlanStore, taskStore | dailyPlanService |

## Cross-Domain Dependencies

- **Task + Context**: TodayView and DashboardView both read taskStore and contextStore to correlate tasks with their context names
- **Tag integration**: TaskDetailView, ContextDetailView, NoteDetailView all use tagStore for inline tag management
- **Thread integration**: Detail views (Task, Context, Note) use ThreadPanel which calls threadService independently
- **Activity integration**: Task/Context detail views use ActivityLogButton, ActivityHistory, StreakDisplay which call activityLogStore
- **Polling**: TodayView, DashboardView, RawInputQueueView use usePolling with settingsStore.pollingIntervalMs
- **Classification**: TaskBoardView uses ClassifyDialog → classifyService for AI task→context classification
- **Shared UI**: All views use LoadingSpinner, EmptyState, DrawerPanel, Pagination from shared components
- **Toast**: All mutation operations trigger toasts via useToast → toastStore
