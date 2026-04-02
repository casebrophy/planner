# Task & Context Model Refinement — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Simplify task statuses (open/blocked/done/dismissed), add task dependencies with auto-blocking, and split contexts into projects vs. areas with cascade-dismiss on project close.

**Architecture:** Three-layer changes (store → business → app) following existing patterns. New `task_dependencies` junction table. Task status enum rewritten. Context gains a `kind` column. MCP tools updated for new fields and dependency management.

**Tech Stack:** Go, PostgreSQL, sqlx, Vue 3 + Pinia (frontend)

**Spec:** `.docs/specs/2026-04-02-task-context-refinement-design.md`

---

## File Map

### New files
- `business/domain/taskbus/stores/taskdb/dependency.go` — dependency SQL queries
- `business/domain/taskbus/dependency.go` — dependency business methods + DependencyStorer interface
- `business/types/contextkind/contextkind.go` — new enum type
- `app/domain/taskapp/dependency.go` — HTTP handlers for dependency endpoints

### Modified files
- `business/sdk/migrate/sql/migrate.sql` — v1.17 migration
- `business/types/taskstatus/taskstatus.go` — new status values
- `business/domain/taskbus/model.go` — add `BlockedReason` field
- `business/domain/taskbus/taskbus.go` — extend Storer, add dependency methods, auto-blocking logic
- `business/domain/taskbus/stores/taskdb/model.go` — add `BlockedReason` to DB struct
- `business/domain/taskbus/stores/taskdb/taskdb.go` — add `blocked_reason` to SQL columns
- `business/domain/taskbus/stores/taskdb/filter.go` — (no changes needed yet)
- `business/domain/contextbus/model.go` — add `Kind` field, add `contextkind` import
- `business/domain/contextbus/contextbus.go` — add Kind to Create, validate close/pause for areas, cascade dismiss
- `business/domain/contextbus/filter.go` — add Kind filter
- `business/domain/contextbus/stores/contextdb/model.go` — add `Kind` to DB struct
- `business/domain/contextbus/stores/contextdb/contextdb.go` — add `kind` to SQL columns
- `business/domain/contextbus/stores/contextdb/filter.go` — add Kind filter
- `app/domain/taskapp/model.go` — add `BlockedReason`, `Dependencies`, `Dependents` fields
- `app/domain/taskapp/route.go` — add dependency routes
- `app/domain/taskapp/filter.go` — (no changes needed yet)
- `app/domain/contextapp/model.go` — add `Kind` field
- `app/domain/contextapp/contextapp.go` — cascade dismiss on project close
- `app/domain/contextapp/filter.go` — add `kind` query param
- `app/domain/mcpapp/tools.go` — update tool definitions
- `app/domain/mcpapp/mcpapp.go` — update tool handlers, add dependency tools

---

## Task 1: Database Migration (v1.17)

**Files:**
- Modify: `business/sdk/migrate/sql/migrate.sql`

- [ ] **Step 1: Write the migration SQL**

Append to the end of `migrate.sql`:

```sql
-- Version: 1.17
-- Description: Task status simplification, task dependencies, context kind

-- 1. Add blocked_reason to tasks
ALTER TABLE tasks ADD COLUMN blocked_reason TEXT NOT NULL DEFAULT '';

-- 2. Add kind to contexts
ALTER TABLE contexts ADD COLUMN kind TEXT NOT NULL DEFAULT 'project'
    CHECK (kind IN ('project', 'area'));

-- 3. Create task_dependencies table
CREATE TABLE task_dependencies (
    task_id       UUID NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    depends_on_id UUID NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (task_id, depends_on_id),
    CHECK (task_id != depends_on_id)
);
CREATE INDEX idx_task_deps_depends_on ON task_dependencies(depends_on_id);

-- 4. Migrate task statuses: todo→open, in_progress→open, cancelled→dismissed
UPDATE tasks SET status = 'open' WHERE status IN ('todo', 'in_progress');
UPDATE tasks SET status = 'dismissed' WHERE status = 'cancelled';

-- 5. Update task status constraint
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_status_check;
ALTER TABLE tasks ADD CONSTRAINT tasks_status_check CHECK (status IN ('open', 'blocked', 'done', 'dismissed'));
```

- [ ] **Step 2: Run migration locally to verify**

```bash
make db-up && make migrate
```

Expected: migration applies without errors.

- [ ] **Step 3: Commit**

```bash
git add business/sdk/migrate/sql/migrate.sql
git commit -m "feat: v1.17 migration — task statuses, dependencies, context kind"
```

---

## Task 2: Task Status Enum

**Files:**
- Modify: `business/types/taskstatus/taskstatus.go`

- [ ] **Step 1: Update the status enum values**

Replace the existing var block and statuses map:

Old:
```go
var (
	Todo       = Status{"todo"}
	InProgress = Status{"in_progress"}
	Done       = Status{"done"}
	Cancelled  = Status{"cancelled"}
)

var statuses = map[string]Status{
	Todo.value:       Todo,
	InProgress.value: InProgress,
	Done.value:       Done,
	Cancelled.value:  Cancelled,
}
```

New:
```go
var (
	Open      = Status{"open"}
	Blocked   = Status{"blocked"}
	Done      = Status{"done"}
	Dismissed = Status{"dismissed"}
)

var statuses = map[string]Status{
	Open.value:      Open,
	Blocked.value:   Blocked,
	Done.value:      Done,
	Dismissed.value: Dismissed,
}
```

- [ ] **Step 2: Fix all compilation errors from removed symbols**

Search for `taskstatus.Todo`, `taskstatus.InProgress`, `taskstatus.Cancelled` across the codebase and replace:
- `taskstatus.Todo` → `taskstatus.Open`
- `taskstatus.InProgress` → `taskstatus.Open`
- `taskstatus.Cancelled` → `taskstatus.Dismissed`

Key files to check:
- `business/domain/taskbus/taskbus.go` (Create defaults to `taskstatus.Open`)
- `app/domain/taskapp/model.go` (`toBusNewTask` defaults to `taskstatus.Open`)
- `app/domain/mcpapp/mcpapp.go` (`toolCreateTask` defaults to `taskstatus.Open`)
- `app/domain/dailyplanapp/dailyplanapp.go` (may filter by status)
- `business/domain/ingestbus/ingestbus.go` (creates tasks)
- `business/domain/inactivitybus/inactivitybus.go` (checks task status)
- `business/domain/debriefbus/debriefbus.go` (checks task completion)

- [ ] **Step 3: Verify compilation**

```bash
make lint
```

Expected: no errors.

- [ ] **Step 4: Run tests**

```bash
make test
```

Expected: all pass (or known-unrelated failures).

- [ ] **Step 5: Commit**

```bash
git add business/types/taskstatus/ business/domain/ app/domain/
git commit -m "feat: simplify task statuses — open/blocked/done/dismissed"
```

---

## Task 3: Context Kind Enum

**Files:**
- Create: `business/types/contextkind/contextkind.go`

- [ ] **Step 1: Create the enum package**

```go
package contextkind

import "fmt"

type Kind struct {
	value string
}

var (
	Project = Kind{"project"}
	Area    = Kind{"area"}
)

var kinds = map[string]Kind{
	Project.value: Project,
	Area.value:    Area,
}

func Parse(s string) (Kind, error) {
	k, ok := kinds[s]
	if !ok {
		return Kind{}, fmt.Errorf("invalid context kind %q", s)
	}
	return k, nil
}

func MustParse(s string) Kind {
	k, ok := kinds[s]
	if !ok {
		panic(fmt.Sprintf("invalid context kind %q", s))
	}
	return k
}

func (k Kind) String() string {
	return k.value
}

func (k Kind) MarshalText() ([]byte, error) {
	return []byte(k.value), nil
}

func (k *Kind) UnmarshalText(data []byte) error {
	var err error
	*k, err = Parse(string(data))
	return err
}

func (k Kind) EqualString(v string) bool {
	return k.value == v
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./business/types/contextkind/...
```

- [ ] **Step 3: Commit**

```bash
git add business/types/contextkind/
git commit -m "feat: add contextkind enum type (project/area)"
```

---

## Task 4: Context Business Layer — Kind Field + Close Validation

**Files:**
- Modify: `business/domain/contextbus/model.go`
- Modify: `business/domain/contextbus/contextbus.go`
- Modify: `business/domain/contextbus/filter.go`
- Modify: `business/domain/contextbus/stores/contextdb/model.go`
- Modify: `business/domain/contextbus/stores/contextdb/contextdb.go`
- Modify: `business/domain/contextbus/stores/contextdb/filter.go`

- [ ] **Step 1: Add Kind to context business model**

In `business/domain/contextbus/model.go`, add to `Context` struct:

```go
Kind contextkind.Kind
```

Add to `NewContext` struct:

```go
Kind contextkind.Kind
```

Add to `UpdateContext` struct:

```go
Kind *contextkind.Kind
```

Import `"github.com/casebrophy/planner/business/types/contextkind"`.

- [ ] **Step 2: Update contextbus.Create to set Kind**

In `business/domain/contextbus/contextbus.go`, in the `Create` method, set `Kind: nt.Kind` on the Context struct being constructed. If `nt.Kind` is zero value, default to `contextkind.Project`.

Add to the Create method, before the `storer.Create` call:

```go
kind := nt.Kind
if kind == (contextkind.Kind{}) {
    kind = contextkind.Project
}
```

And set `Kind: kind` in the Context struct.

- [ ] **Step 3: Add close/pause validation for areas**

In `business/domain/contextbus/contextbus.go`, in the `Update` method, after applying the patch fields, add validation:

```go
if uc.Status != nil {
    if c.Kind == contextkind.Area && (*uc.Status == Closed || *uc.Status == Paused) {
        return Context{}, fmt.Errorf("cannot close or pause an area context")
    }
}
```

- [ ] **Step 4: Add Kind to UpdateContext field application**

In the `Update` method's field-patching section, add:

```go
if uc.Kind != nil {
    c.Kind = *uc.Kind
}
```

- [ ] **Step 5: Add Kind to context QueryFilter**

In `business/domain/contextbus/filter.go`, add to the `QueryFilter` struct:

```go
Kind *contextkind.Kind
```

- [ ] **Step 6: Update context DB model**

In `business/domain/contextbus/stores/contextdb/model.go`:

Add `Kind string \`db:"kind"\`` to the `contextDB` struct.

In `toDBContext`, add: `Kind: c.Kind.String()`.

In `toBusContext`, add: `Kind: contextkind.MustParse(c.Kind)`.

- [ ] **Step 7: Update context DB SQL**

In `business/domain/contextbus/stores/contextdb/contextdb.go`:

Add `kind` to the column list in all SQL queries (INSERT, UPDATE, SELECT).

- [ ] **Step 8: Update context DB filter**

In `business/domain/contextbus/stores/contextdb/filter.go`, in `applyFilter`, add:

```go
if filter.Kind != nil {
    buf.WriteString(" AND kind = :filter_kind")
    data["filter_kind"] = filter.Kind.String()
}
```

- [ ] **Step 9: Verify compilation**

```bash
make lint
```

- [ ] **Step 10: Commit**

```bash
git add business/domain/contextbus/ business/types/contextkind/
git commit -m "feat: add context kind (project/area) with close validation"
```

---

## Task 5: Task Business Layer — BlockedReason + Dependency Methods

**Files:**
- Modify: `business/domain/taskbus/model.go`
- Modify: `business/domain/taskbus/taskbus.go`
- Create: `business/domain/taskbus/dependency.go`
- Modify: `business/domain/taskbus/stores/taskdb/model.go`
- Modify: `business/domain/taskbus/stores/taskdb/taskdb.go`
- Create: `business/domain/taskbus/stores/taskdb/dependency.go`

- [ ] **Step 1: Add BlockedReason to task business model**

In `business/domain/taskbus/model.go`:

Add `BlockedReason string` to the `Task` struct.

Add `BlockedReason *string` to the `UpdateTask` struct.

- [ ] **Step 2: Update task DB model**

In `business/domain/taskbus/stores/taskdb/model.go`:

Add `BlockedReason string \`db:"blocked_reason"\`` to the `taskDB` struct.

In `toDBTask`: `BlockedReason: t.BlockedReason`.

In `toBusTask`: `BlockedReason: t.BlockedReason`.

- [ ] **Step 3: Update task DB SQL columns**

In `business/domain/taskbus/stores/taskdb/taskdb.go`:

Add `blocked_reason` to the column list in INSERT, UPDATE, and SELECT queries.

- [ ] **Step 4: Update taskbus.Update to handle BlockedReason**

In `business/domain/taskbus/taskbus.go`, in the `Update` method's field-patching section, add:

```go
if ut.BlockedReason != nil {
    tsk.BlockedReason = *ut.BlockedReason
}
```

- [ ] **Step 5: Create the DependencyStorer interface and business methods**

Create `business/domain/taskbus/dependency.go`:

```go
package taskbus

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Dependency struct {
	TaskID      uuid.UUID
	DependsOnID uuid.UUID
	CreatedAt   time.Time
}

type DependencyStorer interface {
	AddDependency(ctx context.Context, dep Dependency) error
	RemoveDependency(ctx context.Context, taskID, dependsOnID uuid.UUID) error
	QueryDependencies(ctx context.Context, taskID uuid.UUID) ([]Task, error)
	QueryDependents(ctx context.Context, taskID uuid.UUID) ([]Task, error)
	HasUnmetDependencies(ctx context.Context, taskID uuid.UUID) (bool, error)
}

func (b *Business) AddDependency(ctx context.Context, taskID, dependsOnID uuid.UUID) error {
	if taskID == dependsOnID {
		return fmt.Errorf("a task cannot depend on itself")
	}

	// Check for direct cycle: dependsOnID already depends on taskID.
	deps, err := b.depStorer.QueryDependencies(ctx, dependsOnID)
	if err != nil {
		return fmt.Errorf("checking for cycles: %w", err)
	}
	for _, d := range deps {
		if d.ID == taskID {
			return fmt.Errorf("adding this dependency would create a cycle")
		}
	}

	dep := Dependency{
		TaskID:      taskID,
		DependsOnID: dependsOnID,
		CreatedAt:   time.Now().UTC(),
	}

	if err := b.depStorer.AddDependency(ctx, dep); err != nil {
		return fmt.Errorf("adding dependency: %w", err)
	}

	// Auto-block the downstream task if upstream isn't done.
	upstream, err := b.storer.QueryByID(ctx, dependsOnID)
	if err != nil {
		return fmt.Errorf("querying upstream task: %w", err)
	}

	if upstream.Status != taskstatus.Done {
		downstream, err := b.storer.QueryByID(ctx, taskID)
		if err != nil {
			return fmt.Errorf("querying downstream task: %w", err)
		}
		if downstream.Status == taskstatus.Open {
			downstream.Status = taskstatus.Blocked
			downstream.UpdatedAt = time.Now().UTC()
			if err := b.storer.Update(ctx, downstream); err != nil {
				return fmt.Errorf("auto-blocking downstream task: %w", err)
			}
		}
	}

	return nil
}

func (b *Business) RemoveDependency(ctx context.Context, taskID, dependsOnID uuid.UUID) error {
	if err := b.depStorer.RemoveDependency(ctx, taskID, dependsOnID); err != nil {
		return fmt.Errorf("removing dependency: %w", err)
	}

	return b.reevaluateBlocked(ctx, taskID)
}

func (b *Business) QueryDependencies(ctx context.Context, taskID uuid.UUID) ([]Task, error) {
	return b.depStorer.QueryDependencies(ctx, taskID)
}

func (b *Business) QueryDependents(ctx context.Context, taskID uuid.UUID) ([]Task, error) {
	return b.depStorer.QueryDependents(ctx, taskID)
}

// reevaluateBlocked checks if a task should remain blocked based on its
// dependencies. If all upstream tasks are done and there's no manual
// blocked_reason, the task is set back to open.
func (b *Business) reevaluateBlocked(ctx context.Context, taskID uuid.UUID) error {
	task, err := b.storer.QueryByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("querying task: %w", err)
	}

	if task.Status != taskstatus.Blocked {
		return nil
	}

	if task.BlockedReason != "" {
		return nil
	}

	hasUnmet, err := b.depStorer.HasUnmetDependencies(ctx, taskID)
	if err != nil {
		return fmt.Errorf("checking unmet dependencies: %w", err)
	}

	if !hasUnmet {
		task.Status = taskstatus.Open
		task.UpdatedAt = time.Now().UTC()
		if err := b.storer.Update(ctx, task); err != nil {
			return fmt.Errorf("unblocking task: %w", err)
		}
	}

	return nil
}

// UnblockDependents is called when a task is marked done. It re-evaluates
// all downstream tasks that depend on this one.
func (b *Business) UnblockDependents(ctx context.Context, taskID uuid.UUID) error {
	dependents, err := b.depStorer.QueryDependents(ctx, taskID)
	if err != nil {
		return fmt.Errorf("querying dependents: %w", err)
	}

	for _, dep := range dependents {
		if err := b.reevaluateBlocked(ctx, dep.ID); err != nil {
			return fmt.Errorf("re-evaluating dependent %s: %w", dep.ID, err)
		}
	}

	return nil
}
```

- [ ] **Step 6: Update Business struct to hold DependencyStorer**

In `business/domain/taskbus/taskbus.go`:

Add `depStorer DependencyStorer` to the `Business` struct.

Update `NewBusiness`:

```go
func NewBusiness(log *logger.Logger, storer Storer, depStorer DependencyStorer) *Business {
	return &Business{
		log:       log,
		storer:    storer,
		depStorer: depStorer,
	}
}
```

- [ ] **Step 7: Wire UnblockDependents into Update**

In `business/domain/taskbus/taskbus.go`, in the `Update` method, after the `storer.Update` call, add:

```go
if tsk.Status == taskstatus.Done {
	if err := b.UnblockDependents(ctx, tsk.ID); err != nil {
		b.log.Info(ctx, "unblock_dependents_failed", "task_id", tsk.ID, "error", err)
	}
}
```

- [ ] **Step 8: Create the dependency store implementation**

Create `business/domain/taskbus/stores/taskdb/dependency.go`:

```go
package taskdb

import (
	"bytes"
	"context"
	"fmt"

	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/foundation/sqldb"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DependencyStore struct {
	log *logger.Logger
	db  sqlx.ExtContext
}

func NewDependencyStore(log *logger.Logger, db *sqlx.DB) *DependencyStore {
	return &DependencyStore{
		log: log,
		db:  db,
	}
}

func (s *DependencyStore) AddDependency(ctx context.Context, dep taskbus.Dependency) error {
	const q = `
	INSERT INTO task_dependencies (task_id, depends_on_id, created_at)
	VALUES (:task_id, :depends_on_id, :created_at)`

	data := struct {
		TaskID      uuid.UUID `db:"task_id"`
		DependsOnID uuid.UUID `db:"depends_on_id"`
		CreatedAt   string    `db:"created_at"`
	}{
		TaskID:      dep.TaskID,
		DependsOnID: dep.DependsOnID,
		CreatedAt:   dep.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("inserting dependency: %w", err)
	}

	return nil
}

func (s *DependencyStore) RemoveDependency(ctx context.Context, taskID, dependsOnID uuid.UUID) error {
	const q = `
	DELETE FROM task_dependencies
	WHERE task_id = :task_id AND depends_on_id = :depends_on_id`

	data := struct {
		TaskID      uuid.UUID `db:"task_id"`
		DependsOnID uuid.UUID `db:"depends_on_id"`
	}{
		TaskID:      taskID,
		DependsOnID: dependsOnID,
	}

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("deleting dependency: %w", err)
	}

	return nil
}

func (s *DependencyStore) QueryDependencies(ctx context.Context, taskID uuid.UUID) ([]taskbus.Task, error) {
	const q = `
	SELECT t.task_id, t.context_id, t.title, t.description, t.status, t.priority,
	       t.energy, t.duration_min, t.due_date, t.scheduled_at,
	       t.expected_update_days, t.last_thread_at, t.debrief_status,
	       t.blocked_reason, t.created_at, t.updated_at, t.completed_at
	FROM tasks t
	INNER JOIN task_dependencies td ON t.task_id = td.depends_on_id
	WHERE td.task_id = :task_id
	ORDER BY t.created_at`

	data := struct {
		TaskID uuid.UUID `db:"task_id"`
	}{
		TaskID: taskID,
	}

	var rows []taskDB
	if err := sqldb.NamedQuerySlice(ctx, s.log, s.db, q, data, &rows); err != nil {
		return nil, fmt.Errorf("querying dependencies: %w", err)
	}

	return toBusTasks(rows), nil
}

func (s *DependencyStore) QueryDependents(ctx context.Context, taskID uuid.UUID) ([]taskbus.Task, error) {
	const q = `
	SELECT t.task_id, t.context_id, t.title, t.description, t.status, t.priority,
	       t.energy, t.duration_min, t.due_date, t.scheduled_at,
	       t.expected_update_days, t.last_thread_at, t.debrief_status,
	       t.blocked_reason, t.created_at, t.updated_at, t.completed_at
	FROM tasks t
	INNER JOIN task_dependencies td ON t.task_id = td.task_id
	WHERE td.depends_on_id = :depends_on_id
	ORDER BY t.created_at`

	data := struct {
		DependsOnID uuid.UUID `db:"depends_on_id"`
	}{
		DependsOnID: taskID,
	}

	var rows []taskDB
	if err := sqldb.NamedQuerySlice(ctx, s.log, s.db, q, data, &rows); err != nil {
		return nil, fmt.Errorf("querying dependents: %w", err)
	}

	return toBusTasks(rows), nil
}

func (s *DependencyStore) HasUnmetDependencies(ctx context.Context, taskID uuid.UUID) (bool, error) {
	const q = `
	SELECT COUNT(*) AS count
	FROM task_dependencies td
	INNER JOIN tasks t ON t.task_id = td.depends_on_id
	WHERE td.task_id = :task_id AND t.status != 'done'`

	data := struct {
		TaskID uuid.UUID `db:"task_id"`
	}{
		TaskID: taskID,
	}

	var result struct {
		Count int `db:"count"`
	}

	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &result); err != nil {
		return false, fmt.Errorf("checking unmet dependencies: %w", err)
	}

	return result.Count > 0, nil
}
```

- [ ] **Step 9: Verify compilation**

```bash
make lint
```

Note: this will likely fail because callers of `NewBusiness` need to pass the new `depStorer` parameter. That's fixed in Task 7 (route wiring). For now, just verify the business/store packages compile:

```bash
go build ./business/domain/taskbus/... ./business/domain/taskbus/stores/taskdb/...
```

- [ ] **Step 10: Commit**

```bash
git add business/domain/taskbus/ business/types/taskstatus/
git commit -m "feat: task dependencies and blocked_reason in business/store layers"
```

---

## Task 6: Context Cascade — Dismiss Tasks on Project Close

**Files:**
- Modify: `business/domain/contextbus/contextbus.go`
- Modify: `app/domain/contextapp/contextapp.go`

This task wires the cascade: when a project context is closed, all its open/blocked tasks are dismissed.

- [ ] **Step 1: Add a TaskDismisser interface to contextbus**

In `business/domain/contextbus/contextbus.go`, add an optional collaborator:

```go
type TaskDismisser interface {
	DismissTasksByContext(ctx context.Context, contextID uuid.UUID) (int, error)
}
```

Add to the `Business` struct:

```go
taskDismisser TaskDismisser
```

Add a setter method (keeps existing constructor unchanged):

```go
func (b *Business) WithTaskDismisser(td TaskDismisser) {
	b.taskDismisser = td
}
```

- [ ] **Step 2: Add DismissTasksByContext to the task store**

In `business/domain/taskbus/stores/taskdb/taskdb.go`, add:

```go
func (s *Store) DismissTasksByContext(ctx context.Context, contextID uuid.UUID) (int, error) {
	const q = `
	UPDATE tasks
	SET status = 'dismissed', updated_at = :updated_at
	WHERE context_id = :context_id AND status IN ('open', 'blocked')`

	data := struct {
		ContextID uuid.UUID `db:"context_id"`
		UpdatedAt time.Time `db:"updated_at"`
	}{
		ContextID: contextID,
		UpdatedAt: time.Now().UTC(),
	}

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return 0, fmt.Errorf("dismissing tasks by context: %w", err)
	}

	// NamedExecContext doesn't return row count, so we count after.
	return 0, nil
}
```

Also add a wrapper in `business/domain/taskbus/taskbus.go`:

```go
func (b *Business) DismissTasksByContext(ctx context.Context, contextID uuid.UUID) (int, error) {
	return b.storer.DismissTasksByContext(ctx, contextID)
}
```

And add `DismissTasksByContext` to the `Storer` interface:

```go
DismissTasksByContext(ctx context.Context, contextID uuid.UUID) (int, error)
```

- [ ] **Step 3: Wire cascade into context close**

In `app/domain/contextapp/contextapp.go`, in the `update` handler, after the existing debrief trigger logic for closed contexts, add the cascade dismiss:

```go
if updated.Kind == contextkind.Project && updated.Status == contextbus.Closed {
    if a.taskBus != nil {
        if _, err := a.taskBus.DismissTasksByContext(ctx, updated.ID); err != nil {
            a.log.Info(ctx, "cascade_dismiss_failed", "context_id", updated.ID, "error", err)
        }
    }
}
```

The `app` struct in `contextapp` needs a `taskBus` field. Add it and wire it in `route.go`.

- [ ] **Step 4: Update contextapp route.go to wire taskBus**

In `app/domain/contextapp/route.go`, in `Routes.Add`:

```go
taskStore := taskdb.NewStore(cfg.Log, cfg.DB)
depStore := taskdb.NewDependencyStore(cfg.Log, cfg.DB)
taskBus := taskbus.NewBusiness(cfg.Log, taskStore, depStore)

hdl := &app{contextBus: bus, clarificationBus: clarBus, taskBus: taskBus}
```

Add `taskBus *taskbus.Business` to the `app` struct in `contextapp.go`.

- [ ] **Step 5: Verify compilation**

```bash
make lint
```

- [ ] **Step 6: Commit**

```bash
git add business/domain/contextbus/ business/domain/taskbus/ app/domain/contextapp/
git commit -m "feat: cascade dismiss open tasks when project context closes"
```

---

## Task 7: Route Wiring — Fix All NewBusiness Callers

**Files:**
- Modify: `app/domain/taskapp/route.go`
- Modify: `app/domain/mcpapp/route.go`
- Modify: `app/domain/dailyplanapp/route.go`
- Modify: any other file calling `taskbus.NewBusiness`

The `NewBusiness` signature changed to include `DependencyStorer`. Every call site needs updating.

- [ ] **Step 1: Find all callers**

```bash
grep -rn "taskbus.NewBusiness" app/ business/
```

- [ ] **Step 2: Update each caller**

For each call site, add the dependency store:

```go
taskStore := taskdb.NewStore(cfg.Log, cfg.DB)
depStore := taskdb.NewDependencyStore(cfg.Log, cfg.DB)
taskBus := taskbus.NewBusiness(cfg.Log, taskStore, depStore)
```

- [ ] **Step 3: Verify full compilation**

```bash
make lint
```

Expected: clean.

- [ ] **Step 4: Run tests**

```bash
make test
```

- [ ] **Step 5: Commit**

```bash
git add app/domain/
git commit -m "feat: wire DependencyStore into all taskbus.NewBusiness callers"
```

---

## Task 8: App Layer — Task Dependency Endpoints

**Files:**
- Create: `app/domain/taskapp/dependency.go`
- Modify: `app/domain/taskapp/route.go`
- Modify: `app/domain/taskapp/model.go`

- [ ] **Step 1: Add dependency fields to app-layer Task model**

In `app/domain/taskapp/model.go`, add to the `Task` struct:

```go
BlockedReason string   `json:"blockedReason,omitempty"`
```

Update `toAppTask` to include:

```go
BlockedReason: t.BlockedReason,
```

Update `toBusUpdateTask` to handle `BlockedReason`:

```go
if at.BlockedReason != nil {
    ut.BlockedReason = at.BlockedReason
}
```

Add `BlockedReason *string \`json:"blockedReason"\`` to the `UpdateTask` struct.

- [ ] **Step 2: Create dependency HTTP handlers**

Create `app/domain/taskapp/dependency.go`:

```go
package taskapp

import (
	"context"
	"net/http"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/foundation/web"
	"github.com/google/uuid"
)

func (a *app) addDependency(ctx context.Context, r *http.Request) web.Encoder {
	taskID, err := uuid.Parse(web.Param(r, "task_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	dependsOnID, err := uuid.Parse(web.Param(r, "depends_on_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if err := a.taskBus.AddDependency(ctx, taskID, dependsOnID); err != nil {
		return errs.New(errs.Internal, err)
	}

	return web.NewNoContent()
}

func (a *app) removeDependency(ctx context.Context, r *http.Request) web.Encoder {
	taskID, err := uuid.Parse(web.Param(r, "task_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	dependsOnID, err := uuid.Parse(web.Param(r, "depends_on_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if err := a.taskBus.RemoveDependency(ctx, taskID, dependsOnID); err != nil {
		return errs.New(errs.Internal, err)
	}

	return web.NewNoContent()
}

func (a *app) queryDependencies(ctx context.Context, r *http.Request) web.Encoder {
	taskID, err := uuid.Parse(web.Param(r, "task_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	tasks, err := a.taskBus.QueryDependencies(ctx, taskID)
	if err != nil {
		return errs.New(errs.Internal, err)
	}

	return toAppTasks(tasks)
}

func (a *app) queryDependents(ctx context.Context, r *http.Request) web.Encoder {
	taskID, err := uuid.Parse(web.Param(r, "task_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	tasks, err := a.taskBus.QueryDependents(ctx, taskID)
	if err != nil {
		return errs.New(errs.Internal, err)
	}

	return toAppTasks(tasks)
}
```

- [ ] **Step 3: Add dependency routes**

In `app/domain/taskapp/route.go`, add after the existing routes:

```go
a.Handle(http.MethodPost, "/api/v1/tasks/{task_id}/dependencies/{depends_on_id}", hdl.addDependency, authen)
a.Handle(http.MethodDelete, "/api/v1/tasks/{task_id}/dependencies/{depends_on_id}", hdl.removeDependency, authen)
a.Handle(http.MethodGet, "/api/v1/tasks/{task_id}/dependencies", hdl.queryDependencies, authen)
a.Handle(http.MethodGet, "/api/v1/tasks/{task_id}/dependents", hdl.queryDependents, authen)
```

- [ ] **Step 4: Verify compilation**

```bash
make lint
```

- [ ] **Step 5: Commit**

```bash
git add app/domain/taskapp/
git commit -m "feat: REST endpoints for task dependencies"
```

---

## Task 9: App Layer — Context Kind in API

**Files:**
- Modify: `app/domain/contextapp/model.go`
- Modify: `app/domain/contextapp/filter.go`

- [ ] **Step 1: Add Kind to app-layer context models**

In `app/domain/contextapp/model.go`:

Add `Kind string \`json:"kind"\`` to the `Context` struct.

Add `Kind string \`json:"kind"\`` to the `NewContext` struct.

Add `Kind *string \`json:"kind"\`` to the `UpdateContext` struct.

Update `toAppContext`: `Kind: c.Kind.String()`.

Update `toBusNewContext` to parse Kind:

```go
kind := contextkind.Project
if nc.Kind != "" {
    var err error
    kind, err = contextkind.Parse(nc.Kind)
    if err != nil {
        return contextbus.NewContext{}, fmt.Errorf("parsing kind: %w", err)
    }
}
bus.Kind = kind
```

Update `toBusUpdateContext` to parse Kind if provided.

- [ ] **Step 2: Add kind to context filter**

In `app/domain/contextapp/filter.go`, add in `parseFilter`:

```go
if v := r.URL.Query().Get("kind"); v != "" {
    kind, err := contextkind.Parse(v)
    if err != nil {
        return contextbus.QueryFilter{}, fmt.Errorf("invalid kind: %w", err)
    }
    filter.Kind = &kind
}
```

- [ ] **Step 3: Verify compilation**

```bash
make lint
```

- [ ] **Step 4: Commit**

```bash
git add app/domain/contextapp/
git commit -m "feat: context kind (project/area) in REST API"
```

---

## Task 10: MCP Tool Updates

**Files:**
- Modify: `app/domain/mcpapp/tools.go`
- Modify: `app/domain/mcpapp/mcpapp.go`

- [ ] **Step 1: Add dependency MCP tool definitions**

In `app/domain/mcpapp/tools.go`, add three new tool definitions:

```go
{
    Name:        "add_task_dependency",
    Description: "Add a dependency between tasks. The first task depends on (is blocked by) the second task.",
    InputSchema: inputSchema{
        Type:     "object",
        Required: []string{"task_id", "depends_on_id"},
        Properties: map[string]property{
            "task_id":       {Type: "string", Description: "ID of the task that depends on another"},
            "depends_on_id": {Type: "string", Description: "ID of the task that must be completed first"},
        },
    },
},
{
    Name:        "remove_task_dependency",
    Description: "Remove a dependency between tasks.",
    InputSchema: inputSchema{
        Type:     "object",
        Required: []string{"task_id", "depends_on_id"},
        Properties: map[string]property{
            "task_id":       {Type: "string", Description: "ID of the downstream task"},
            "depends_on_id": {Type: "string", Description: "ID of the upstream task to unlink"},
        },
    },
},
{
    Name:        "get_task_dependencies",
    Description: "Get both upstream dependencies (what blocks this task) and downstream dependents (what this task blocks).",
    InputSchema: inputSchema{
        Type:     "object",
        Required: []string{"task_id"},
        Properties: map[string]property{
            "task_id": {Type: "string", Description: "The task ID to query dependencies for"},
        },
    },
},
```

- [ ] **Step 2: Update create_task tool definition**

Add `blocked_reason` and `depends_on` to the `create_task` properties:

```go
"blocked_reason": {Type: "string", Description: "Why this task is blocked (manual block reason)"},
"depends_on":     {Type: "array", Description: "Array of task IDs this task depends on", Items: &property{Type: "string"}},
```

Check if the `property` struct supports `Items` — if not, skip the `depends_on` array on create and have Claude call `add_task_dependency` separately.

- [ ] **Step 3: Update create_context and update_context tool definitions**

Add `kind` property to `create_context`:

```go
"kind": {Type: "string", Description: "Context kind: 'project' (time-bounded, closeable) or 'area' (ongoing, never closes). Default: project"},
```

- [ ] **Step 4: Add tool handler cases to callTool switch**

In `app/domain/mcpapp/mcpapp.go`, add cases in the `callTool` switch:

```go
case "add_task_dependency":
    return a.toolAddTaskDependency(ctx, args)
case "remove_task_dependency":
    return a.toolRemoveTaskDependency(ctx, args)
case "get_task_dependencies":
    return a.toolGetTaskDependencies(ctx, args)
```

- [ ] **Step 5: Implement dependency tool handlers**

In `app/domain/mcpapp/mcpapp.go`, add:

```go
func (a *app) toolAddTaskDependency(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		TaskID      string `json:"task_id"`
		DependsOnID string `json:"depends_on_id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, fmt.Errorf("parsing arguments: %w", err)
	}

	taskID, err := uuid.Parse(input.TaskID)
	if err != nil {
		return textResult(map[string]string{"error": "invalid task_id"}), nil
	}

	dependsOnID, err := uuid.Parse(input.DependsOnID)
	if err != nil {
		return textResult(map[string]string{"error": "invalid depends_on_id"}), nil
	}

	if err := a.taskBus.AddDependency(ctx, taskID, dependsOnID); err != nil {
		return textResult(map[string]string{"error": err.Error()}), nil
	}

	return textResult(map[string]string{
		"message":       "Dependency added",
		"task_id":       input.TaskID,
		"depends_on_id": input.DependsOnID,
	}), nil
}

func (a *app) toolRemoveTaskDependency(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		TaskID      string `json:"task_id"`
		DependsOnID string `json:"depends_on_id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, fmt.Errorf("parsing arguments: %w", err)
	}

	taskID, err := uuid.Parse(input.TaskID)
	if err != nil {
		return textResult(map[string]string{"error": "invalid task_id"}), nil
	}

	dependsOnID, err := uuid.Parse(input.DependsOnID)
	if err != nil {
		return textResult(map[string]string{"error": "invalid depends_on_id"}), nil
	}

	if err := a.taskBus.RemoveDependency(ctx, taskID, dependsOnID); err != nil {
		return textResult(map[string]string{"error": err.Error()}), nil
	}

	return textResult(map[string]string{
		"message":       "Dependency removed",
		"task_id":       input.TaskID,
		"depends_on_id": input.DependsOnID,
	}), nil
}

func (a *app) toolGetTaskDependencies(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		TaskID string `json:"task_id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, fmt.Errorf("parsing arguments: %w", err)
	}

	taskID, err := uuid.Parse(input.TaskID)
	if err != nil {
		return textResult(map[string]string{"error": "invalid task_id"}), nil
	}

	deps, err := a.taskBus.QueryDependencies(ctx, taskID)
	if err != nil {
		return textResult(map[string]string{"error": err.Error()}), nil
	}

	dependents, err := a.taskBus.QueryDependents(ctx, taskID)
	if err != nil {
		return textResult(map[string]string{"error": err.Error()}), nil
	}

	type taskSummary struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Status string `json:"status"`
	}

	blockedBy := make([]taskSummary, len(deps))
	for i, d := range deps {
		blockedBy[i] = taskSummary{ID: d.ID.String(), Title: d.Title, Status: d.Status.String()}
	}

	blocks := make([]taskSummary, len(dependents))
	for i, d := range dependents {
		blocks[i] = taskSummary{ID: d.ID.String(), Title: d.Title, Status: d.Status.String()}
	}

	result := map[string]any{
		"task_id":    input.TaskID,
		"blocked_by": blockedBy,
		"blocks":     blocks,
	}

	return textResult(result), nil
}
```

- [ ] **Step 6: Update toolCreateTask for blocked_reason**

In `toolCreateTask`, add parsing for `blocked_reason`:

```go
if v, ok := input["blocked_reason"].(string); ok && v != "" {
    // After task creation, set blocked_reason via update
    brStr := v
    ut := taskbus.UpdateTask{
        BlockedReason: &brStr,
        Status:        &taskstatus.Blocked,
    }
    task, _ = a.taskBus.Update(ctx, task, ut)
}
```

- [ ] **Step 7: Update toolCreateContext for kind**

In `toolCreateContext`, add parsing for `kind`:

```go
kind := contextkind.Project
if v, ok := input["kind"].(string); ok && v != "" {
    var err error
    kind, err = contextkind.Parse(v)
    if err != nil {
        return textResult(map[string]string{"error": fmt.Sprintf("invalid kind: %s", v)}), nil
    }
}
nt := contextbus.NewContext{
    Title:       title,
    Description: desc,
    Kind:        kind,
}
```

- [ ] **Step 8: Verify compilation**

```bash
make lint
```

- [ ] **Step 9: Run tests**

```bash
make test
```

- [ ] **Step 10: Commit**

```bash
git add app/domain/mcpapp/
git commit -m "feat: MCP tools for task dependencies and context kind"
```

---

## Task 11: Frontend — Task Status + Blocked State

**Files:**
- Modify: frontend task-related stores, components, and views

This task updates the frontend to reflect the new task statuses and show blocked state. Since frontend patterns vary, the implementer should:

- [ ] **Step 1: Update task status constants**

Find the frontend file that defines task status options (likely in a composable, store, or constants file). Replace:
- `todo` → `open`
- `in_progress` → remove
- `cancelled` → `dismissed`
- Add `blocked`

- [ ] **Step 2: Update task board/list UI**

- Blocked tasks should be visually distinct (e.g., grayed out, different icon, or separate section)
- Remove any "In Progress" column or filter option
- Add "Blocked" as a visible state

- [ ] **Step 3: Update task detail view**

- Show `blockedReason` if present
- Show dependencies section (blocked by / blocks) — can be a simple list of linked task titles for now

- [ ] **Step 4: Add dependency management UI**

- In task detail, add ability to link/unlink dependencies via the REST API
- Simple search + select for "blocked by" tasks

- [ ] **Step 5: Verify frontend builds**

```bash
make frontend-dev
```

- [ ] **Step 6: Commit**

```bash
git add frontend/
git commit -m "feat: frontend task status simplification and dependency display"
```

---

## Task 12: Frontend — Context Kind

**Files:**
- Modify: frontend context-related stores, components, and views

- [ ] **Step 1: Update context creation**

- Add kind selector (project/area) to context create form, default to project

- [ ] **Step 2: Update context board**

- Show kind badge on context cards
- Allow filtering by kind
- Hide close button for area contexts

- [ ] **Step 3: Update context detail**

- Show kind indicator
- For projects: show close button with confirmation ("X open tasks will be dismissed")
- For areas: no close option

- [ ] **Step 4: Verify frontend builds**

```bash
make frontend-dev
```

- [ ] **Step 5: Commit**

```bash
git add frontend/
git commit -m "feat: frontend context kind (project/area) support"
```

---

## Task 13: Update Architecture Docs

**Files:**
- Modify: `.docs/arch/task-backend.md`
- Modify: `.docs/arch/context-backend.md`
- Modify: `.docs/07-roadmap.md`

- [ ] **Step 1: Regenerate task arch doc**

Run `/go-arch` for the task domain to update the dependency map with new fields, dependency methods, and status values.

- [ ] **Step 2: Regenerate context arch doc**

Run `/go-arch` for the context domain to update with Kind field and cascade behavior.

- [ ] **Step 3: Update roadmap**

Add a new section or update Phase 7c to reflect that task simplification and context kind are complete. Note task dependencies as a new capability.

- [ ] **Step 4: Commit**

```bash
git add .docs/
git commit -m "docs: update arch docs and roadmap for task/context refinement"
```

---

## Execution Order

Tasks 1–3 are independent foundations (migration, task status enum, context kind enum).

Tasks 4–6 build on 1–3 (business layer changes).

Task 7 fixes wiring after Task 5 changes the constructor signature.

Tasks 8–10 are app-layer changes that depend on 4–7.

Tasks 11–12 are frontend, depend on 8–10.

Task 13 is docs, done last.

```
[1: Migration] ─────────────────────┐
[2: Task Status Enum] ──────────────┤
[3: Context Kind Enum] ─────────────┤
                                     ├──▶ [4: Context Bus Kind] ──┐
                                     ├──▶ [5: Task Bus Deps] ─────┤
                                     │                             ├──▶ [6: Cascade Dismiss] ──┐
                                     │                             │                            │
                                     │                             ├──▶ [7: Fix Wiring] ────────┤
                                     │                                                          │
                                     │    ┌─────────────────────────────────────────────────────┘
                                     │    │
                                     │    ├──▶ [8: Task App Layer] ──┐
                                     │    ├──▶ [9: Context App Layer]┤
                                     │    │                          ├──▶ [10: MCP Tools] ──┐
                                     │    │                          │                       │
                                     │    │                          │    ┌──────────────────┘
                                     │    │                          │    │
                                     │    │                          │    ├──▶ [11: FE Tasks]──┐
                                     │    │                          │    ├──▶ [12: FE Contexts]┤
                                     │    │                          │    │                      │
                                     │    │                          │    │    ┌─────────────────┘
                                     │    │                          │    │    │
                                     │    │                          │    │    └──▶ [13: Docs]
```

**Parallelizable groups:**
- Tasks 1, 2, 3 (independent foundations)
- Tasks 4, 5 (after 1–3, independent of each other)
- Tasks 8, 9 (after 7, independent of each other)
- Tasks 11, 12 (after 10, independent of each other)
