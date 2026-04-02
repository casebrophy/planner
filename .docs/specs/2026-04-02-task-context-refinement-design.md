# Task & Context Model Refinement

## Problem

The current task and context models don't match how the planner is actually used:

1. **Task statuses are overcomplicated.** Tasks go from "not started" to "done" — there's no meaningful `in_progress` window for things like "clean the car." The `cancelled` label is also misleading; tasks that aren't done are just dismissed.

2. **Contexts conflate two concepts.** "Bathroom Reno" (time-bounded project) and "Home Maintenance" (ongoing area) behave differently but share the same model with no distinction. Areas should never close. Projects should cascade-dismiss their tasks when closed.

3. **No task dependencies.** There's no way to say "buy paint before painting the bathroom." The daily plan can't reason about sequencing across days, and blocked tasks aren't surfaced.

## Design

### 1. Task Status Simplification

**Current:** `todo`, `in_progress`, `done`, `cancelled`
**New:** `open`, `blocked`, `done`, `dismissed`

| Status | Meaning |
|--------|---------|
| `open` | Ready to be worked on |
| `blocked` | Can't do this yet (manual reason or upstream dependency) |
| `done` | Completed |
| `dismissed` | Not doing it (no longer needed, context closed, etc.) |

Migration: `todo` → `open`, `in_progress` → `open`, `cancelled` → `dismissed`. No data loss.

The `blocked` status is set automatically when a task has unfinished upstream dependencies, or manually with a reason string.

### 2. Task Dependencies

New junction table `task_dependencies`:

```sql
CREATE TABLE task_dependencies (
    task_id       UUID NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    depends_on_id UUID NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (task_id, depends_on_id),
    CHECK (task_id != depends_on_id)
);
CREATE INDEX idx_task_deps_depends_on ON task_dependencies(depends_on_id);
```

Semantics: `task_id` depends on `depends_on_id`. Task A is **blocked** if any of its upstream dependencies are not `done`.

**Auto-blocking rules:**
- When a dependency is added and the upstream task isn't `done`, the downstream task becomes `blocked`.
- When an upstream task is marked `done`, re-evaluate all downstream tasks. If all upstreams are `done`, set status to `open`.
- When an upstream task is reopened (set back to `open`), downstream tasks become `blocked` again.
- Manual `blocked` status (with a reason, no dependency) is also supported — the `blocked_reason` field on the task captures this.

**New fields on tasks table:**
```sql
ALTER TABLE tasks ADD COLUMN blocked_reason TEXT DEFAULT '';
```

A task is considered blocked if: it has a non-empty `blocked_reason`, OR it has any upstream dependency that isn't `done`.

**Business layer:**
- `taskbus.AddDependency(ctx, taskID, dependsOnID)` — validates no cycles (direct only — A depends on B, B depends on A), adds row, auto-blocks if needed.
- `taskbus.RemoveDependency(ctx, taskID, dependsOnID)` — removes row, re-evaluates blocked status.
- `taskbus.QueryDependencies(ctx, taskID)` — returns upstream tasks.
- `taskbus.QueryDependents(ctx, taskID)` — returns downstream tasks.

**Cycle detection:** Check direct cycles only (A→B→A). Deep cycle detection is unnecessary for a personal task manager — the user won't create complex DAGs.

**MCP integration:** Claude suggests dependencies at capture time. When the user says "buy paint for the bathroom reno," Claude can check for existing tasks in the same context and suggest "this should be done before 'paint bathroom walls'."

### 3. Context Kind: Project vs. Area

**New field on contexts table:**
```sql
ALTER TABLE contexts ADD COLUMN kind TEXT NOT NULL DEFAULT 'project'
    CHECK (kind IN ('project', 'area'));
```

| Kind | Closes? | Debrief? | Task cascade on close? |
|------|---------|----------|----------------------|
| `project` | Yes | Yes | Remaining `open`/`blocked` tasks → `dismissed` |
| `area` | No | No | N/A |

**Context status changes:**
- **Current:** `active`, `paused`, `closed`
- **New:** Same statuses, but `closed` and `paused` are only valid for `project` kind. Areas are always `active`.
- Attempting to close or pause an area returns an error.

**Cascade behavior on project close:**
- All tasks with `status = 'open'` or `status = 'blocked'` AND `context_id = <project>` get set to `dismissed`.
- Tasks already `done` are left alone.
- A clarification card is generated listing the dismissed tasks, in case the user wants to reassign any of them to a different context.

### 4. Daily Plan Impact

No schema changes needed. The daily plan generation prompt is updated to reason about:
- **Dependencies:** "Do X today because it unblocks Y for tomorrow."
- **Blocked tasks:** Excluded from the daily plan unless the blocking task is also in the plan (in which case, order them correctly).
- **Context kind:** Projects with due dates get priority. Areas contribute ongoing/recurring tasks.

### 5. Filter & Query Changes

**Task filters — add:**
- `HasDependencies *bool` — filter to tasks that block or are blocked by others
- `BlockedBy *uuid.UUID` — filter to tasks blocked by a specific task

**Context filters — add:**
- `Kind *string` — filter by `project` or `area`

### 6. REST API Changes

**New endpoints:**
- `POST /api/v1/tasks/{task_id}/dependencies/{depends_on_id}` — add dependency
- `DELETE /api/v1/tasks/{task_id}/dependencies/{depends_on_id}` — remove dependency
- `GET /api/v1/tasks/{task_id}/dependencies` — list upstream dependencies
- `GET /api/v1/tasks/{task_id}/dependents` — list downstream dependents

**Modified endpoints:**
- `POST /api/v1/tasks` — accepts optional `depends_on` array of task IDs
- `PUT /api/v1/tasks/{task_id}` — accepts optional `blocked_reason`
- `POST /api/v1/contexts` — accepts `kind` field (default: `project`)
- `PUT /api/v1/contexts/{context_id}` — `kind` can be set but not changed from `area` to `project` if tasks exist (to avoid confusion)

### 7. MCP Tool Changes

**New tools:**
- `add_task_dependency` — params: `task_id`, `depends_on_id`
- `remove_task_dependency` — params: `task_id`, `depends_on_id`
- `get_task_dependencies` — params: `task_id` — returns both upstream and downstream

**Modified tools:**
- `create_task` — add optional `depends_on` param (array of task IDs), optional `blocked_reason`
- `update_task` — add optional `blocked_reason`
- `create_context` — add `kind` param (default: `project`)
- `get_inference_context` — include dependency graph info for tasks in the daily plan window

### 8. Frontend Changes

**Task detail view:**
- Show dependencies section: "Blocked by" (upstream) and "Blocks" (downstream) with task links
- Show `blocked_reason` if set manually
- Visual indicator for blocked status (distinct from open)

**Task board:**
- Blocked tasks visually distinct (grayed out or separate column)
- Filter by blocked/open/done/dismissed

**Context board:**
- Show kind badge (project/area)
- Area contexts don't show close button
- Project close confirmation shows count of tasks that will be dismissed

**Context create/edit:**
- Kind selector (project/area), defaults to project

## Migration Strategy

Single migration file with:
1. Add `kind` column to contexts (default `project` — existing contexts are projects)
2. Add `blocked_reason` column to tasks
3. Create `task_dependencies` table
4. Update task status CHECK constraint: `('open', 'blocked', 'done', 'dismissed')`
5. Migrate existing data: `todo` → `open`, `in_progress` → `open`, `cancelled` → `dismissed`
6. Update context status CHECK to enforce `closed`/`paused` only for projects (application-level, not DB constraint — simpler)

## What This Does NOT Include

- Recurring tasks (Phase 7c)
- Notes/knowledge capture (Phase 7c)
- Trackable logs (Phase 7c)
- Deep cycle detection (unnecessary for personal use)
- Auto-suggested dependencies from AI (the MCP tool interface supports it, but the AI prompt tuning is separate work)
