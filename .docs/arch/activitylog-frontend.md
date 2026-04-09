# Activity Log System

> Generic activity tracking domain that logs timestamped entries against any entity (`subjectType` + `subjectId`). Powers two surfaces: (1) per-entity history + streak display on TaskDetailView and NoteDetailView, and (2) bulk habit-grid rendering on HabitsView/DashboardView. The store extends `createCRUDStore` and adds streak and habitGrid caches keyed by `subjectType:subjectId`.

## Core Types

```ts
// web/src/types/activityLog.ts

export interface ActivityLog {
  id: string
  subjectType: string   // e.g. "task", "note", "context"
  subjectId: string
  value?: string        // optional free-text payload
  loggedAt: string      // ISO timestamp
}

export interface NewActivityLog {
  subjectType: string
  subjectId: string
  value?: string
}

export interface ActivityLogFilter {
  subjectType?: string
  subjectId?: string
  startDate?: string    // ISO timestamp
  endDate?: string      // ISO timestamp
}

export interface StreakInfo {
  current: number
  longest: number
  totalCount: number
  lastLogged?: string   // ISO timestamp
}

export interface BulkLogsResponse {
  items: Record<string, ActivityLog[]>  // key = subjectId
}
```

```ts
// web/src/stores/activityLogStore.ts (exported type alias)

export type HabitGridMap = Record<string, string[]>
// key = subjectId, value = array of ISO date strings (YYYY-MM-DD) when logged
```

## File Map

### Stores
- `stores/activityLogStore.ts` — **useActivityLogStore** — CRUD store extended with `streaks` (StreakInfo keyed by `subjectType:subjectId`), `habitGrid` (HabitGridMap), `fetchStreaks()`, `fetchHabitGrid()`, `logActivity()`

### Services
- `services/activityLogService.ts` — **activityLogService** — CRUD service for `/api/v1/activity-logs` plus two custom endpoints: `getStreaks(subjectType, subjectId)` and `getBulkLogs(subjectType, subjectIds[], from, to)`

### Composables
- `composables/useDashboard.ts` — **useDashboard** — calls `activityLogService.list()` directly (bypasses store) to fetch last 4 weeks of `task` logs; computes `completionTrend` (WeekBucket[]) and `inactiveContexts` from `activityLogs` ref

### Components
- `components/shared/ActivityLogButton.vue` — **ActivityLogButton** — inline log button with optional value input; calls `store.logActivity(subjectType, subjectId, value)`
- `components/shared/ActivityHistory.vue` — **ActivityHistory** — fetches and displays paginated log entries directly via `activityLogService.list()`; exposes `reload()` ref
- `components/shared/StreakDisplay.vue` — **StreakDisplay** — reads `store.streaks[key]`; triggers `store.fetchStreaks()` on mount; renders current/longest/total counts
- `components/habits/HabitGrid.vue` — **HabitGrid** — pure display component; receives `habits: Task[]`, `habitGrid: HabitGridMap`, `days: Date[]` as props; uses `isCompleted(habitId, day, grid)` to color cells
- `components/habits/HabitRow.vue` — **HabitRow** — row within habit grid; calls `activityLogStore.logActivity('task', habit.id, dateStr)` on cell click; computes `currentStreak` locally from `completedDates` prop

### Views
- `views/HabitsView.vue` — calls `taskStore.fetchHabits()` + `activityLogStore.fetchHabitGrid(habitIds, dayRange)`, passes results to `HabitGrid`
- `views/TaskDetailView.vue` — mounts `ActivityLogButton`, `StreakDisplay`, `ActivityHistory` for `subjectType="task"`
- `views/NoteDetailView.vue` — mounts `ActivityLogButton`, `StreakDisplay`, `ActivityHistory` for `subjectType="note"`

## Impact Callouts

### ⚠ ActivityLog (web/src/types/activityLog.ts)
Changing this interface shape affects:
- `stores/activityLogStore.ts` — CRUD list items typed as `ActivityLog[]`; `fetchHabitGrid` iterates `response.items` entries and reads `.loggedAt`
- `services/activityLogService.ts` — generic type param for `createCRUDService<ActivityLog, ...>`; deserializes response JSON directly to this shape
- `components/shared/ActivityHistory.vue` — `entries` ref typed as `ActivityLog[]`; template reads `.id`, `.value`, `.loggedAt`
- `composables/useDashboard.ts` — `activityLogs` ref typed as `ActivityLog[]`; `completionTrend` reads `.subjectType`, `.loggedAt`; `inactiveContexts` reads `.subjectType`, `.subjectId`, `.loggedAt`

### ⚠ StreakInfo (web/src/types/activityLog.ts)
Changing this interface shape affects:
- `stores/activityLogStore.ts` — `streaks` ref typed as `Record<string, StreakInfo>`; populated by `activityLogService.getStreaks()`
- `components/shared/StreakDisplay.vue` — reads `streak.current`, `streak.longest`, `streak.totalCount` for display

### ⚠ HabitGridMap (web/src/stores/activityLogStore.ts)
Changing this type alias (currently `Record<string, string[]>`) affects:
- `components/habits/HabitGrid.vue` — receives as `habitGrid` prop; calls `isCompleted(habitId, day, grid)` which checks `grid[habitId].includes(dateKey(day))`
- `components/habits/HabitRow.vue` — receives `completedDates: string[]` (a single row extracted from the map by the parent); `isCompleted()` uses `.includes()`
- `views/HabitsView.vue` — passes `activityLogStore.habitGrid` directly to `HabitGrid`

### ⚠ ActivityLogFilter (web/src/types/activityLog.ts)
Changing this interface shape affects:
- `services/activityLogService.ts` — `mapFilter` converts camelCase fields to snake_case query params (`subject_type`, `subject_id`, `start_date`, `end_date`)
- `components/shared/ActivityHistory.vue` — passes `{ subjectType, subjectId }` as filter to `activityLogService.list()`
- `composables/useDashboard.ts` — passes `{ subjectType: 'task', startDate, endDate }` to `activityLogService.list()`

### ⚠ BulkLogsResponse (web/src/types/activityLog.ts)
Changing this interface shape affects:
- `services/activityLogService.ts` — return type of `getBulkLogs()`
- `stores/activityLogStore.ts` — `fetchHabitGrid()` calls `getBulkLogs()`, iterates `response.items` entries to build `HabitGridMap`

## Cross-Domain Dependencies

- `stores/taskStore.ts` — `HabitsView` calls `taskStore.fetchHabits()` to get habit task IDs before calling `fetchHabitGrid()`; `HabitGrid` and `HabitRow` receive `Task` objects as props (use `.id`, `.title`)
- `stores/toastStore.ts` — `useActivityLogStore.fetchStreaks()` calls `toast.error()` on failure
- `services/createCRUDService.ts` — `activityLogService` is built on top of this factory
- `stores/createCRUDStore.ts` — `useActivityLogStore` delegates CRUD operations to this factory
- `composables/useDashboard.ts` — consumes `activityLogService` directly (not the store) for dashboard-level aggregation across all tasks
- `views/TaskDetailView.vue`, `views/NoteDetailView.vue` — primary consumers of the three shared activity components (`ActivityLogButton`, `StreakDisplay`, `ActivityHistory`)
