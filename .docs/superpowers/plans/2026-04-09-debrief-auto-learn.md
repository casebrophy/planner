# Debrief Auto-Learn

> Always show the task debrief dialog on completion. Record skip data. Auto-skip future debriefs when the user consistently skips tasks with similar characteristics.

## Context

The debrief dialog (`TaskDebriefDialog.vue`) asks "How did it go?" when a task is marked done, but it's gated behind `task.trackOutcome` which is never set to true. The backend already fires `debriefBus.OnTaskCompleted()` unconditionally — the gap is frontend-only.

## Design Decisions

- **Frontend-only gate removal** — don't delete `TrackOutcome` from backend models (avoids cross-layer cascade)
- **Enrich observation payloads** — both submit and skip observations include task metadata (`contextId`, `priority`, `energy`) so the auto-skip heuristic can match patterns
- **Client-side heuristic** — query all `debrief` observations for `subjectType=task`, filter client-side by matching context/priority/energy. Observation volume is low (one per completed task), so no backend filter changes needed.
- **Heuristic threshold** — auto-skip if ≥5 observations exist for the matching profile AND skip rate >80%
- **No flash** — dialog uses a loading guard; auto-skip resolves before the dialog renders

## Tasks

### Task 1: Enrich observation payload + record skips (TaskDebriefDialog.vue)

**Files:** `api/services/frontend/web/src/components/tasks/TaskDebriefDialog.vue`

Changes:
1. Add `task` prop (full Task object) alongside existing `taskId` prop — needed for metadata
2. In `handleSubmit()`, enrich the observation data payload:
   ```ts
   data: {
     outcome: selected.value,
     note: note.value.trim() || undefined,
     contextId: props.task.contextId || null,
     priority: props.task.priority,
     energy: props.task.energy,
   }
   ```
3. In `handleSkip()`, record a skip observation before emitting close:
   ```ts
   async function handleSkip() {
     await observationService.record({
       subjectType: 'task',
       subjectId: props.task.id,
       kind: 'debrief',
       data: {
         outcome: 'skipped',
         contextId: props.task.contextId || null,
         priority: props.task.priority,
         energy: props.task.energy,
       },
     })
     emit('close')
   }
   ```

### Task 2: Remove trackOutcome gate + add auto-skip (TaskDetailView.vue)

**Files:** `api/services/frontend/web/src/views/TaskDetailView.vue`

Changes:
1. Remove the `trackOutcome` condition from `handleUpdate()`:
   ```ts
   // Before:
   if ((data as UpdateTask).status === 'done' && task.value?.trackOutcome) {
   // After:
   if ((data as UpdateTask).status === 'done') {
   ```
2. Pass full `task` object to `TaskDebriefDialog` (for metadata enrichment):
   ```vue
   <TaskDebriefDialog :open="showDebrief" :task="task" @close="showDebrief = false" />
   ```
3. Add auto-skip logic: before showing the dialog, check if this task's profile is auto-skippable:
   ```ts
   async function shouldAutoSkip(task: Task): Promise<boolean> {
     const observations = await observationService.queryByKind('task', 'debrief')
     const matching = observations.filter(o => {
       const d = o.data as { contextId?: string; priority?: string; energy?: string }
       return d.contextId === (task.contextId || null)
         && d.priority === task.priority
         && d.energy === task.energy
     })
     if (matching.length < 5) return false
     const skipped = matching.filter(o => (o.data as { outcome: string }).outcome === 'skipped')
     return skipped.length / matching.length > 0.8
   }
   ```
4. Wire it into `handleUpdate()`:
   ```ts
   if ((data as UpdateTask).status === 'done') {
     const skip = await shouldAutoSkip(task.value!)
     if (!skip) {
       showDebrief.value = true
     }
   }
   ```

### Task 3: Add queryByKind to observationService

**Files:** `api/services/frontend/web/src/services/observationService.ts`

The existing `queryBySubject()` filters by subjectId (single task). The heuristic needs all debrief observations across all tasks. Add:
```ts
async queryByKind(subjectType: string, kind: string): Promise<Observation[]> {
  // GET /api/v1/observations?subjectType=task&kind=debrief&rows_per_page=100
}
```

Check whether the backend observation query endpoint already supports `kind` as a query param (the backend `QueryFilter` has a `Kind` field — verify it's wired in `observationapp/filter.go`). If not, wire it.

### Task 4: Frontend tests

**Files:** `api/services/frontend/web/src/__tests__/components/tasks/TaskDebriefDialog.test.ts`

Tests:
1. `handleSkip()` calls `observationService.record` with `outcome: 'skipped'` + task metadata
2. `handleSubmit()` calls `observationService.record` with selected outcome + task metadata  
3. Auto-skip: when mocked observations show >80% skip rate with ≥5 samples, dialog does not open
4. No auto-skip: when <5 observations exist, dialog opens normally
5. No auto-skip: when skip rate is ≤80%, dialog opens normally

### Task 5: Verify backend observation filter supports `kind` param

**Files (if needed):**
- `app/domain/observationapp/filter.go` — add kind to parseFilter
- `business/domain/observationbus/filter.go` — already has Kind field
- `business/domain/observationbus/stores/observationdb/filter.go` — verify applyFilter handles Kind

This may already be wired. Check before modifying.

## Ordering

Task 5 (verify backend) → Task 3 (queryByKind service) → Task 1 (dialog changes) → Task 2 (view changes) → Task 4 (tests)

Tasks 1 and 2 can run in parallel after 3 and 5 are confirmed.
