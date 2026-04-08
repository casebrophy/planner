# Plan: Habit Tracking — Recurring Tasks Completion Grid
**Date:** 2026-04-08
**Branch:** `feat/habit-tracking`

## Overview

Surface recurring tasks as "habits" and show a GitHub-style completion grid: rows = habits, columns = past N days, cells = filled if completed that day. No new domain — habits ARE tasks with `recurrence_rule IS NOT NULL AND recurrence_parent_id IS NULL`. Completion data comes from the existing `activity_logs` table.

**Design Decision — Log Subject Binding:**
Recurring tasks fire a new child task UUID each day via `CreateNextRecurrence()`. To avoid joining children → parent for every grid query, log activity against **both** the child UUID AND the parent UUID on task completion. The grid queries by parent UUID. This is the simplest approach.

---

## Tasks

### Task 1: Backend — `HasRecurrence` task filter

**Files:**
- MODIFY `business/domain/taskbus/filter.go` — add `HasRecurrence *bool`
- MODIFY `business/domain/taskbus/stores/taskdb/filter.go` — add SQL clause `AND recurrence_rule IS NOT NULL` when `*HasRecurrence == true`; `AND recurrence_rule IS NULL` when false
- MODIFY `app/domain/taskapp/filter.go` — parse `hasRecurrence` query param → `*bool`

**Pattern:** Follow existing nullable filter fields (e.g., `Status *taskstatus.Status`) — nil = no filter, non-nil = apply clause.

**Test:** In `taskdb_test.go`, seed one task with recurrence_rule and one without. Query with `HasRecurrence = true`, assert only recurring task returned.

---

### Task 2: Backend — Log activity against parent task on completion

**File:**
- MODIFY `business/domain/taskbus/taskbus.go` — in the completion path (where `CreateNextRecurrence()` is called, ~line 128-132), also call `activityLogBus.Create()` with `SubjectType = "task"`, `SubjectID = task.RecurrenceParentID` (if set, else `task.ID`)

**Why:** The grid queries logs by parent UUID. Without this, grid cells would always be empty even after completing child instances.

**Note:** `taskbus` already has access to `activityLogBus` if it's wired in `main.go`. If not, wire it at initialization time.

**Test:** Complete a recurring task child instance; assert activity log created with parent's UUID as SubjectID.

---

### Task 3: Backend — Bulk activity log query

**Files:**
- MODIFY `business/domain/activitylogbus/model.go` — add struct:
  ```go
  type QueryBySubjectsFilter struct {
      SubjectType string
      SubjectIDs  []uuid.UUID
      From        time.Time
      To          time.Time
  }
  ```
- MODIFY `business/domain/activitylogbus/activitylogbus.go` — add method `QueryBySubjects(ctx, filter QueryBySubjectsFilter) ([]Log, error)` + add to `Storer` interface
- MODIFY `business/domain/activitylogbus/stores/activitylogdb/activitylogdb.go` — implement SQL:
  ```sql
  SELECT * FROM activity_logs
  WHERE subject_type = $1
    AND subject_id = ANY($2)
    AND logged_at >= $3
    AND logged_at <= $4
  ORDER BY logged_at ASC
  ```

**Pattern:** Follow `QueryStreaks()` at `activitylogdb.go:89-175` for store method shape. Use `pq.Array()` or plain `ANY` with a typed slice for `subject_id = ANY($2)`.

**Test:** Insert logs for 3 subject IDs; call `QueryBySubjects` with 2 of those IDs + date range; assert correct rows returned, out-of-range log excluded.

---

### Task 4: Backend — Bulk logs HTTP endpoint

**Files:**
- MODIFY `app/domain/activitylogapp/activitylogapp.go` — add handler `queryBulk`:
  - Query params: `subject_type` (string), `subject_ids` (comma-sep UUIDs), `from` (RFC3339), `to` (RFC3339)
  - Response: `map[string][]AppLog` keyed by subject_id string
- MODIFY `app/domain/activitylogapp/route.go` — register:
  ```go
  a.Handle(http.MethodGet, "/api/v1/activity-logs/bulk", hdl.queryBulk, authen)
  ```

**Pattern:** Follow existing handler methods in `activitylogapp.go` — use `r.URL.Query().Get()`, parse UUIDs with `uuid.Parse()`, call business layer, `web.Respond()`.

**Error handling:** If `subject_ids` is empty, return 400. If any UUID is malformed, return 400. Use `errs.New(errs.InvalidArgument, err)`.

**Test:** API test — seed recurring task + logs for two dates; call endpoint; assert 200 + correct map with date strings.

---

### Task 5: Backend — Migration index

**File:**
- MODIFY `business/sdk/migrate/sql/migrate.sql` — append:
  ```sql
  -- Version: 1.26
  -- Description: Add index on activity_logs for bulk habit grid queries
  CREATE INDEX IF NOT EXISTS idx_activity_logs_subject_date
      ON activity_logs(subject_type, subject_id, logged_at);
  ```

**Note:** Optimization only — feature works without it. Can be deferred.

---

### Task 6: Frontend — Task store `fetchHabits()` action

**File:**
- MODIFY `api/services/frontend/web/src/stores/taskStore.ts` — add action:
  ```ts
  async fetchHabits() {
    return this.fetchList({ hasRecurrence: true, excludeStatuses: ['done', 'dismissed'] })
    // or however the store's list action accepts filter params
  }
  ```

**Note:** Filter for `recurrence_parent_id IS NULL` needs backend support too (Task 1 only adds `HasRecurrence`). If the backend doesn't yet filter out child instances, add `RecurrenceParentOnly *bool` filter as part of Task 1 or add it here as a separate sub-task.

---

### Task 7: Frontend — Activity log service + store for habit grid

**Files:**
- MODIFY `api/services/frontend/web/src/services/activityLogService.ts` — add:
  ```ts
  getBulkLogs(subjectType: string, subjectIds: string[], from: Date, to: Date): Promise<Record<string, ActivityLog[]>>
  ```
  Calls `GET /api/v1/activity-logs/bulk?subject_type=...&subject_ids=...&from=...&to=...`

- MODIFY `api/services/frontend/web/src/stores/activityLogStore.ts` — add:
  - State: `habitGrid: Record<string, string[]>` (habit ID → array of completed date strings `YYYY-MM-DD`)
  - Action: `fetchHabitGrid(habitIds: string[], days = 30)` — calls `getBulkLogs`, groups logs by `subjectId` → `loggedAt` date strings

**Type:** Add `BulkLogsResponse = Record<string, ActivityLog[]>` to `types/activityLog.ts`.

**Test:** Mock `getBulkLogs` returning two logs for habit A, one for habit B; assert `habitGrid` state correct.

---

### Task 8: Frontend — `HabitRow.vue` component

**File:** CREATE `api/services/frontend/web/src/components/habits/HabitRow.vue`

**Props:**
```ts
{
  habit: Task           // the parent recurring task
  completedDates: string[]  // YYYY-MM-DD strings
  days: Date[]          // array of N dates (column headers)
}
```

**Renders:**
- Habit title (left, fixed width)
- Streak badge (reuse `StreakDisplay` or compute from `completedDates`)
- N day cells — filled (purple/indigo) if date in `completedDates`, empty (gray) if not
- Clicking an empty cell logs activity for that date (calls `activityLogStore.logActivity`)

**Test:** Mount with mock props; assert filled cell for completed date, empty for skipped; assert click on empty cell calls logActivity.

---

### Task 9: Frontend — `HabitGrid.vue` component

**File:** CREATE `api/services/frontend/web/src/components/habits/HabitGrid.vue`

**Props:**
```ts
{
  habits: Task[]
  habitGrid: Record<string, string[]>
  days: Date[]
}
```

**Renders:**
- Header row: date labels (Mon Apr 7, etc.)
- One `HabitRow` per habit
- CSS Grid layout: `grid-template-columns: 180px repeat(N, 1fr)`
- Empty state if no habits

**Behavior:** Passes through `days` and `completedDates` (from `habitGrid[habit.id]`) to each row.

**Test:** Mount with 2 habits, 7 days; assert 2 rows rendered; assert header dates match props.

---

### Task 10: Frontend — `HabitsView.vue` route view

**File:** CREATE `api/services/frontend/web/src/views/HabitsView.vue`

**Behavior:**
1. On mount: `taskStore.fetchHabits()` → populate habit list
2. On mount: `activityLogStore.fetchHabitGrid(habitIds, 30)` → populate grid
3. Renders:
   - Page header "Habits"
   - Range selector (30 days / 90 days toggle)
   - `HabitGrid` with fetched data
   - Loading skeleton while fetching
4. Re-fetches grid when range changes

---

### Task 11: Frontend — Router + sidebar wiring

**Files:**
- MODIFY `api/services/frontend/web/src/router/index.ts` — add:
  ```ts
  { path: '/habits', name: 'habits', component: () => import('@/views/HabitsView.vue') }
  ```
- MODIFY `api/services/frontend/web/src/components/layout/AppSidebar.vue` — add to `primaryNavItems`:
  ```ts
  { name: 'Habits', path: '/habits', icon: 'repeat' }
  ```
  Add the repeat/loop SVG icon in the `v-else-if` chain in the template.

---

## Implementation Order (with dependencies)

```
Task 1 (HasRecurrence filter)
  └── Task 6 (frontend fetchHabits — needs hasRecurrence param)

Task 2 (log against parent on completion) — independent

Task 3 (QueryBySubjects store method)
  └── Task 4 (bulk HTTP endpoint)
       └── Task 7 (frontend service + store)
            └── Task 8 (HabitRow)
                 └── Task 9 (HabitGrid)
                      └── Task 10 (HabitsView)
                           └── Task 11 (routing)

Task 5 (migration index) — independent, can be last
```

---

## Open Questions

1. **Parent-only filter in backend:** Should `HasRecurrence` also imply `recurrence_parent_id IS NULL`? Probably yes for the habits list. Decide at implementation time — could be a separate `IsRecurrenceRoot *bool` filter or fold into `HasRecurrence`.

2. **Logging past dates:** On the grid, clicking a past date should log activity with that specific date as `logged_at`, not `time.Now()`. The `POST /api/v1/activity-logs` endpoint accepts `logged_at` — ensure the frontend sends the correct date when logging from the grid.

3. **Marking done from grid vs. completing the task:** Clicking a grid cell adds an activity log (records "I did this"). Should it also mark the current day's child task instance as `done`? Probably not — they're separate signals. Keep the grid as a "tracking" surface separate from task management.
