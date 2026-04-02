# Phase 7b: Calendar View + Time Blocks Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Self-contained weekly calendar view with time-slotted task scheduling and merged event+block display.

**Architecture:** New `timeblockbus` domain (store → business → app, same three-layer pattern as events). Weekly calendar frontend view merges events (fixed commitments) with time blocks (scheduled tasks) into a unified week grid. MCP tools expose schedule query and block management for Claude.

**Tech Stack:** Go backend (sqlx, uuid, net/http), Vue 3 + TypeScript + Pinia frontend, PostgreSQL.

---

## File Structure

### Backend — New domain: `timeblockbus`

| File | Responsibility |
|------|---------------|
| `business/domain/timeblockbus/model.go` | `TimeBlock`, `NewTimeBlock`, `UpdateTimeBlock` structs |
| `business/domain/timeblockbus/timeblockbus.go` | `Business` struct, `Storer` interface, CRUD + query methods |
| `business/domain/timeblockbus/filter.go` | `QueryFilter` (TaskID, DateFrom, DateTo, Confirmed) |
| `business/domain/timeblockbus/order.go` | `OrderByStartsAt`, `OrderByCreatedAt`, `DefaultOrderBy` |
| `business/domain/timeblockbus/stores/timeblockdb/model.go` | `timeblockDB` struct, `toDBTimeBlock`/`toBusTimeBlock` converters |
| `business/domain/timeblockbus/stores/timeblockdb/timeblockdb.go` | `Store` with Create/Update/Delete/Query/Count/QueryByID |
| `business/domain/timeblockbus/stores/timeblockdb/filter.go` | `applyFilter` — WHERE clauses for task_id, date range, confirmed |
| `business/domain/timeblockbus/stores/timeblockdb/order.go` | `orderByClause` — map constants to SQL columns |
| `business/domain/timeblockbus/timeblockbus_test.go` | Business layer tests (create, query, update, delete) |
| `app/domain/timeblockapp/model.go` | JSON DTOs, `Encode()`, converters |
| `app/domain/timeblockapp/timeblockapp.go` | HTTP handlers (queryAll, queryByID, create, update, delete) |
| `app/domain/timeblockapp/filter.go` | `parseFilter` from query params |
| `app/domain/timeblockapp/order.go` | `parseOrder` from query params |
| `app/domain/timeblockapp/route.go` | `Routes.Add()` — wire store, bus, handlers, auth |

### Backend — Modified files

| File | Change |
|------|--------|
| `business/sdk/migrate/sql/migrate.sql` | Append Version 1.16 — `CREATE TABLE time_blocks` |
| `app/domain/mcpapp/mcpapp.go` | Add `timeBlockBus` field, 3 tool cases (`get_schedule`, `create_time_block`, `confirm_time_block`), tool definitions in `tools/list` |
| `app/domain/mcpapp/route.go` | Wire `timeblockdb.Store` + `timeblockbus.Business`, pass to `app` struct |
| `api/services/planner/main.go` | Add `timeblockapp.Routes{}` to `mux.WebAPI(...)` call |

### Frontend — New files

| File | Responsibility |
|------|---------------|
| `api/services/frontend/web/src/types/timeBlock.ts` | `TimeBlock`, `NewTimeBlock`, `UpdateTimeBlock`, `TimeBlockFilter` interfaces |
| `api/services/frontend/web/src/services/timeBlockService.ts` | CRUD service via `createCRUDService` |
| `api/services/frontend/web/src/stores/timeBlockStore.ts` | Pinia store via `createCRUDStore` |
| `api/services/frontend/web/src/composables/useSchedule.ts` | Merges events + time blocks for a week, provides week navigation |
| `api/services/frontend/web/src/views/ScheduleView.vue` | Weekly calendar grid with events and time blocks |
| `api/services/frontend/web/src/components/schedule/WeekGrid.vue` | 7-column day grid with hour rows |
| `api/services/frontend/web/src/components/schedule/ScheduleBlock.vue` | Positioned block (event or time block) within the grid |
| `api/services/frontend/web/src/components/schedule/TimeBlockForm.vue` | Create/edit time block (task select, start/end time) |

### Frontend — Modified files

| File | Change |
|------|--------|
| `api/services/frontend/web/src/router/index.ts` | Add `/schedule` route |
| `api/services/frontend/web/src/components/layout/AppSidebar.vue` | Add "Schedule" nav item |

---

## Task 1: Migration — `time_blocks` table

**Files:**
- Modify: `business/sdk/migrate/sql/migrate.sql` (append at end)

- [ ] **Step 1: Add migration SQL**

Append to the end of `business/sdk/migrate/sql/migrate.sql`:

```sql
-- Version: 1.16
-- Description: Add time_blocks table for calendar scheduling
CREATE TABLE time_blocks (
    block_id    UUID        NOT NULL DEFAULT gen_random_uuid(),
    task_id     UUID        NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    starts_at   TIMESTAMPTZ NOT NULL,
    ends_at     TIMESTAMPTZ NOT NULL,
    confirmed   BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (block_id)
);
CREATE INDEX idx_time_blocks_task    ON time_blocks(task_id);
CREATE INDEX idx_time_blocks_date    ON time_blocks(starts_at, ends_at);
```

- [ ] **Step 2: Run migration locally**

```bash
make db-up && make migrate
```

Expected: Migration applies cleanly, `time_blocks` table exists.

- [ ] **Step 3: Verify table exists**

```bash
docker exec -it $(docker ps -q -f name=postgres) psql -U planner -c "\d time_blocks"
```

Expected: Table with columns block_id, task_id, starts_at, ends_at, confirmed, created_at, updated_at.

- [ ] **Step 4: Commit**

```bash
git add business/sdk/migrate/sql/migrate.sql
git commit -m "feat: add time_blocks table migration (Phase 7b)"
```

---

## Task 2: Business layer — `timeblockbus`

**Files:**
- Create: `business/domain/timeblockbus/model.go`
- Create: `business/domain/timeblockbus/timeblockbus.go`
- Create: `business/domain/timeblockbus/filter.go`
- Create: `business/domain/timeblockbus/order.go`

- [ ] **Step 1: Create model.go**

```go
package timeblockbus

import (
	"time"

	"github.com/google/uuid"
)

type TimeBlock struct {
	ID        uuid.UUID
	TaskID    uuid.UUID
	StartsAt  time.Time
	EndsAt    time.Time
	Confirmed bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type NewTimeBlock struct {
	TaskID   uuid.UUID
	StartsAt time.Time
	EndsAt   time.Time
}

type UpdateTimeBlock struct {
	StartsAt  *time.Time
	EndsAt    *time.Time
	Confirmed *bool
}
```

- [ ] **Step 2: Create filter.go**

```go
package timeblockbus

import (
	"time"

	"github.com/google/uuid"
)

type QueryFilter struct {
	TaskID   *uuid.UUID
	DateFrom *time.Time
	DateTo   *time.Time
}
```

- [ ] **Step 3: Create order.go**

```go
package timeblockbus

import "github.com/casebrophy/planner/business/sdk/order"

const (
	OrderByStartsAt  = "starts_at"
	OrderByCreatedAt = "created_at"
)

var DefaultOrderBy = order.NewBy(OrderByStartsAt, order.ASC)
```

- [ ] **Step 4: Create timeblockbus.go**

```go
package timeblockbus

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/foundation/logger"
)

type Storer interface {
	Create(ctx context.Context, block TimeBlock) error
	Update(ctx context.Context, block TimeBlock) error
	Delete(ctx context.Context, block TimeBlock) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]TimeBlock, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (TimeBlock, error)
}

type Business struct {
	log    *logger.Logger
	storer Storer
}

func NewBusiness(log *logger.Logger, storer Storer) *Business {
	return &Business{
		log:    log,
		storer: storer,
	}
}

func (b *Business) Create(ctx context.Context, nb NewTimeBlock) (TimeBlock, error) {
	now := time.Now()

	block := TimeBlock{
		ID:        uuid.New(),
		TaskID:    nb.TaskID,
		StartsAt:  nb.StartsAt,
		EndsAt:    nb.EndsAt,
		Confirmed: false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := b.storer.Create(ctx, block); err != nil {
		return TimeBlock{}, fmt.Errorf("create: %w", err)
	}

	return block, nil
}

func (b *Business) Update(ctx context.Context, block TimeBlock, ub UpdateTimeBlock) (TimeBlock, error) {
	if ub.StartsAt != nil {
		block.StartsAt = *ub.StartsAt
	}
	if ub.EndsAt != nil {
		block.EndsAt = *ub.EndsAt
	}
	if ub.Confirmed != nil {
		block.Confirmed = *ub.Confirmed
	}
	block.UpdatedAt = time.Now()

	if err := b.storer.Update(ctx, block); err != nil {
		return TimeBlock{}, fmt.Errorf("update: %w", err)
	}

	return block, nil
}

func (b *Business) Delete(ctx context.Context, block TimeBlock) error {
	if err := b.storer.Delete(ctx, block); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

func (b *Business) Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]TimeBlock, error) {
	blocks, err := b.storer.Query(ctx, filter, orderBy, page)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return blocks, nil
}

func (b *Business) Count(ctx context.Context, filter QueryFilter) (int, error) {
	n, err := b.storer.Count(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	return n, nil
}

func (b *Business) QueryByID(ctx context.Context, id uuid.UUID) (TimeBlock, error) {
	block, err := b.storer.QueryByID(ctx, id)
	if err != nil {
		return TimeBlock{}, fmt.Errorf("querybyid[%s]: %w", id, err)
	}
	return block, nil
}
```

- [ ] **Step 5: Verify it compiles**

```bash
cd /Users/casebrophy/personal/planner && go build ./business/domain/timeblockbus/...
```

Expected: No errors.

- [ ] **Step 6: Commit**

```bash
git add business/domain/timeblockbus/
git commit -m "feat: add timeblockbus business layer"
```

---

## Task 3: Store layer — `timeblockdb`

**Files:**
- Create: `business/domain/timeblockbus/stores/timeblockdb/model.go`
- Create: `business/domain/timeblockbus/stores/timeblockdb/timeblockdb.go`
- Create: `business/domain/timeblockbus/stores/timeblockdb/filter.go`
- Create: `business/domain/timeblockbus/stores/timeblockdb/order.go`

- [ ] **Step 1: Create model.go**

```go
package timeblockdb

import (
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/timeblockbus"
)

type timeblockDB struct {
	ID        uuid.UUID `db:"block_id"`
	TaskID    uuid.UUID `db:"task_id"`
	StartsAt  time.Time `db:"starts_at"`
	EndsAt    time.Time `db:"ends_at"`
	Confirmed bool      `db:"confirmed"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func toDBTimeBlock(bus timeblockbus.TimeBlock) timeblockDB {
	return timeblockDB{
		ID:        bus.ID,
		TaskID:    bus.TaskID,
		StartsAt:  bus.StartsAt,
		EndsAt:    bus.EndsAt,
		Confirmed: bus.Confirmed,
		CreatedAt: bus.CreatedAt,
		UpdatedAt: bus.UpdatedAt,
	}
}

func toBusTimeBlock(db timeblockDB) timeblockbus.TimeBlock {
	return timeblockbus.TimeBlock{
		ID:        db.ID,
		TaskID:    db.TaskID,
		StartsAt:  db.StartsAt,
		EndsAt:    db.EndsAt,
		Confirmed: db.Confirmed,
		CreatedAt: db.CreatedAt,
		UpdatedAt: db.UpdatedAt,
	}
}

func toBusTimeBlocks(dbs []timeblockDB) []timeblockbus.TimeBlock {
	blocks := make([]timeblockbus.TimeBlock, len(dbs))
	for i, db := range dbs {
		blocks[i] = toBusTimeBlock(db)
	}
	return blocks
}
```

- [ ] **Step 2: Create filter.go**

```go
package timeblockdb

import (
	"bytes"
	"fmt"

	"github.com/casebrophy/planner/business/domain/timeblockbus"
)

func applyFilter(filter timeblockbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	if filter.TaskID != nil {
		buf.WriteString(fmt.Sprintf(" AND task_id = :task_id"))
		data["task_id"] = *filter.TaskID
	}

	if filter.DateFrom != nil {
		buf.WriteString(fmt.Sprintf(" AND starts_at >= :date_from"))
		data["date_from"] = *filter.DateFrom
	}

	if filter.DateTo != nil {
		buf.WriteString(fmt.Sprintf(" AND ends_at <= :date_to"))
		data["date_to"] = *filter.DateTo
	}
}
```

- [ ] **Step 3: Create order.go**

```go
package timeblockdb

import (
	"fmt"

	"github.com/casebrophy/planner/business/domain/timeblockbus"
	"github.com/casebrophy/planner/business/sdk/order"
)

var orderByFields = map[string]string{
	timeblockbus.OrderByStartsAt:  "starts_at",
	timeblockbus.OrderByCreatedAt: "created_at",
}

func orderByClause(ob order.By) (string, error) {
	field, exists := orderByFields[ob.Field]
	if !exists {
		return "", fmt.Errorf("field %q does not exist", ob.Field)
	}
	return fmt.Sprintf("%s %s", field, ob.Direction), nil
}
```

- [ ] **Step 4: Create timeblockdb.go**

```go
package timeblockdb

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/business/domain/timeblockbus"
	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/foundation/logger"
)

type Store struct {
	log *logger.Logger
	db  sqlx.ExtContext
}

func NewStore(log *logger.Logger, db *sqlx.DB) *Store {
	return &Store{
		log: log,
		db:  db,
	}
}

func (s *Store) Create(ctx context.Context, block timeblockbus.TimeBlock) error {
	const q = `
	INSERT INTO time_blocks
		(block_id, task_id, starts_at, ends_at, confirmed, created_at, updated_at)
	VALUES
		(:block_id, :task_id, :starts_at, :ends_at, :confirmed, :created_at, :updated_at)`

	if err := sqldb.NamedExecContext(ctx, s.db, q, toDBTimeBlock(block)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Update(ctx context.Context, block timeblockbus.TimeBlock) error {
	const q = `
	UPDATE
		time_blocks
	SET
		starts_at = :starts_at,
		ends_at = :ends_at,
		confirmed = :confirmed,
		updated_at = :updated_at
	WHERE
		block_id = :block_id`

	if err := sqldb.NamedExecContext(ctx, s.db, q, toDBTimeBlock(block)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Delete(ctx context.Context, block timeblockbus.TimeBlock) error {
	data := struct {
		ID uuid.UUID `db:"block_id"`
	}{
		ID: block.ID,
	}

	const q = `
	DELETE FROM
		time_blocks
	WHERE
		block_id = :block_id`

	if err := sqldb.NamedExecContext(ctx, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Query(ctx context.Context, filter timeblockbus.QueryFilter, orderBy order.By, page page.Page) ([]timeblockbus.TimeBlock, error) {
	data := map[string]any{
		"offset":        page.Offset(),
		"rows_per_page": page.RowsPerPage,
	}

	const q = `
	SELECT
		block_id, task_id, starts_at, ends_at, confirmed, created_at, updated_at
	FROM
		time_blocks`

	buf := bytes.NewBufferString(q)
	buf.WriteString(" WHERE 1=1")
	applyFilter(filter, data, buf)

	orderClause, err := orderByClause(orderBy)
	if err != nil {
		return nil, err
	}

	buf.WriteString(fmt.Sprintf(" ORDER BY %s", orderClause))
	buf.WriteString(" OFFSET :offset FETCH NEXT :rows_per_page ROWS ONLY")

	var dbBlocks []timeblockDB
	if err := sqldb.NamedQuerySlice(ctx, s.db, buf.String(), data, &dbBlocks); err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return toBusTimeBlocks(dbBlocks), nil
}

func (s *Store) Count(ctx context.Context, filter timeblockbus.QueryFilter) (int, error) {
	data := map[string]any{}

	const q = `
	SELECT
		COUNT(*) AS count
	FROM
		time_blocks`

	buf := bytes.NewBufferString(q)
	buf.WriteString(" WHERE 1=1")
	applyFilter(filter, data, buf)

	var count struct {
		Count int `db:"count"`
	}
	if err := sqldb.NamedQueryStruct(ctx, s.db, buf.String(), data, &count); err != nil {
		return 0, fmt.Errorf("namedquerystruct: %w", err)
	}

	return count.Count, nil
}

func (s *Store) QueryByID(ctx context.Context, id uuid.UUID) (timeblockbus.TimeBlock, error) {
	data := struct {
		ID uuid.UUID `db:"block_id"`
	}{
		ID: id,
	}

	const q = `
	SELECT
		block_id, task_id, starts_at, ends_at, confirmed, created_at, updated_at
	FROM
		time_blocks
	WHERE
		block_id = :block_id`

	var dbBlock timeblockDB
	if err := sqldb.NamedQueryStruct(ctx, s.db, q, data, &dbBlock); err != nil {
		return timeblockbus.TimeBlock{}, fmt.Errorf("namedquerystruct[%s]: %w", id, err)
	}

	return toBusTimeBlock(dbBlock), nil
}
```

- [ ] **Step 5: Verify it compiles**

```bash
go build ./business/domain/timeblockbus/...
```

Expected: No errors.

- [ ] **Step 6: Commit**

```bash
git add business/domain/timeblockbus/stores/
git commit -m "feat: add timeblockdb store layer"
```

---

## Task 4: Business layer tests

**Files:**
- Create: `business/domain/timeblockbus/timeblockbus_test.go`

- [ ] **Step 1: Write tests**

```go
package timeblockbus_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/taskbus/stores/taskdb"
	"github.com/casebrophy/planner/business/domain/timeblockbus"
	"github.com/casebrophy/planner/business/domain/timeblockbus/stores/timeblockdb"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/unitest"
	"github.com/casebrophy/planner/business/types/taskpriority"
	"github.com/casebrophy/planner/business/types/taskstatus"
)

func Test_TimeBlock(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_TimeBlock")

	store := timeblockdb.NewStore(db.Log, db.DB)
	bus := timeblockbus.NewBusiness(db.Log, store)

	// Create a task to reference (time blocks require a task FK).
	taskStore := taskdb.NewStore(db.Log, db.DB)
	taskBus := taskbus.NewBusiness(db.Log, taskStore)

	unitest.Run(t, createAndQuery(bus, taskBus), "create-and-query")
	unitest.Run(t, updateTimeBlock(bus, taskBus), "update")
	unitest.Run(t, deleteTimeBlock(bus, taskBus), "delete")
}

func createTask(ctx context.Context, taskBus *taskbus.Business) (taskbus.Task, error) {
	return taskBus.Create(ctx, taskbus.NewTask{
		Title:    "Test Task for Time Block",
		Status:   taskstatus.MustParse("todo"),
		Priority: taskpriority.MustParse("medium"),
	})
}

func createAndQuery(bus *timeblockbus.Business, taskBus *taskbus.Business) []unitest.Table {
	return []unitest.Table{
		{
			Name: "basic-create",
			ExpResp: timeblockbus.TimeBlock{
				Confirmed: false,
			},
			ExcFunc: func(ctx context.Context) any {
				task, err := createTask(ctx, taskBus)
				if err != nil {
					return err
				}

				nb := timeblockbus.NewTimeBlock{
					TaskID:   task.ID,
					StartsAt: time.Now().Add(24 * time.Hour).Truncate(time.Second),
					EndsAt:   time.Now().Add(25 * time.Hour).Truncate(time.Second),
				}
				resp, err := bus.Create(ctx, nb)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(timeblockbus.TimeBlock)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(timeblockbus.TimeBlock)
				return cmp.Diff(gotResp.Confirmed, expResp.Confirmed)
			},
		},
		{
			Name: "query-all",
			ExpResp: []timeblockbus.TimeBlock{
				{Confirmed: false},
			},
			ExcFunc: func(ctx context.Context) any {
				task, err := createTask(ctx, taskBus)
				if err != nil {
					return err
				}

				nb := timeblockbus.NewTimeBlock{
					TaskID:   task.ID,
					StartsAt: time.Now().Add(24 * time.Hour),
					EndsAt:   time.Now().Add(25 * time.Hour),
				}
				if _, err := bus.Create(ctx, nb); err != nil {
					return err
				}

				resp, err := bus.Query(ctx, timeblockbus.QueryFilter{}, timeblockbus.DefaultOrderBy, page.MustParse("1", "10"))
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.([]timeblockbus.TimeBlock)
				if !exists {
					return "error occurred"
				}
				expResp := exp.([]timeblockbus.TimeBlock)
				if len(gotResp) < len(expResp) {
					return cmp.Diff(len(gotResp), len(expResp))
				}
				return ""
			},
		},
		{
			Name: "query-by-id",
			ExpResp: timeblockbus.TimeBlock{
				Confirmed: false,
			},
			ExcFunc: func(ctx context.Context) any {
				task, err := createTask(ctx, taskBus)
				if err != nil {
					return err
				}

				nb := timeblockbus.NewTimeBlock{
					TaskID:   task.ID,
					StartsAt: time.Now().Add(24 * time.Hour),
					EndsAt:   time.Now().Add(25 * time.Hour),
				}
				created, err := bus.Create(ctx, nb)
				if err != nil {
					return err
				}

				resp, err := bus.QueryByID(ctx, created.ID)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(timeblockbus.TimeBlock)
				if !exists {
					return "error occurred"
				}
				if gotResp.ID.String() == "" {
					return "expected non-empty ID"
				}
				return ""
			},
		},
	}
}

func updateTimeBlock(bus *timeblockbus.Business, taskBus *taskbus.Business) []unitest.Table {
	return []unitest.Table{
		{
			Name: "confirm-block",
			ExpResp: timeblockbus.TimeBlock{
				Confirmed: true,
			},
			ExcFunc: func(ctx context.Context) any {
				task, err := createTask(ctx, taskBus)
				if err != nil {
					return err
				}

				nb := timeblockbus.NewTimeBlock{
					TaskID:   task.ID,
					StartsAt: time.Now().Add(24 * time.Hour),
					EndsAt:   time.Now().Add(25 * time.Hour),
				}
				created, err := bus.Create(ctx, nb)
				if err != nil {
					return err
				}

				confirmed := true
				ub := timeblockbus.UpdateTimeBlock{
					Confirmed: &confirmed,
				}
				resp, err := bus.Update(ctx, created, ub)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(timeblockbus.TimeBlock)
				if !exists {
					return "error occurred"
				}
				expResp := exp.(timeblockbus.TimeBlock)
				return cmp.Diff(gotResp.Confirmed, expResp.Confirmed)
			},
		},
		{
			Name: "reschedule-block",
			ExpResp: timeblockbus.TimeBlock{},
			ExcFunc: func(ctx context.Context) any {
				task, err := createTask(ctx, taskBus)
				if err != nil {
					return err
				}

				nb := timeblockbus.NewTimeBlock{
					TaskID:   task.ID,
					StartsAt: time.Now().Add(24 * time.Hour),
					EndsAt:   time.Now().Add(25 * time.Hour),
				}
				created, err := bus.Create(ctx, nb)
				if err != nil {
					return err
				}

				newStart := time.Now().Add(48 * time.Hour)
				newEnd := time.Now().Add(49 * time.Hour)
				ub := timeblockbus.UpdateTimeBlock{
					StartsAt: &newStart,
					EndsAt:   &newEnd,
				}
				resp, err := bus.Update(ctx, created, ub)
				if err != nil {
					return err
				}
				return resp
			},
			CmpFunc: func(got any, exp any) string {
				gotResp, exists := got.(timeblockbus.TimeBlock)
				if !exists {
					return "error occurred"
				}
				if gotResp.StartsAt.Before(time.Now().Add(47 * time.Hour)) {
					return "expected rescheduled start time"
				}
				return ""
			},
		},
	}
}

func deleteTimeBlock(bus *timeblockbus.Business, taskBus *taskbus.Business) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "basic-delete",
			ExpResp: nil,
			ExcFunc: func(ctx context.Context) any {
				task, err := createTask(ctx, taskBus)
				if err != nil {
					return err
				}

				nb := timeblockbus.NewTimeBlock{
					TaskID:   task.ID,
					StartsAt: time.Now().Add(24 * time.Hour),
					EndsAt:   time.Now().Add(25 * time.Hour),
				}
				created, err := bus.Create(ctx, nb)
				if err != nil {
					return err
				}

				if err := bus.Delete(ctx, created); err != nil {
					return err
				}

				_, err = bus.QueryByID(ctx, created.ID)
				if err == nil {
					return "expected error for deleted block, got nil"
				}

				return nil
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}
}
```

- [ ] **Step 2: Run tests**

```bash
make db-up && go test ./business/domain/timeblockbus/... -count=1 -v
```

Expected: All tests pass.

- [ ] **Step 3: Commit**

```bash
git add business/domain/timeblockbus/timeblockbus_test.go
git commit -m "test: add timeblockbus business layer tests"
```

---

## Task 5: App layer — `timeblockapp`

**Files:**
- Create: `app/domain/timeblockapp/model.go`
- Create: `app/domain/timeblockapp/timeblockapp.go`
- Create: `app/domain/timeblockapp/filter.go`
- Create: `app/domain/timeblockapp/order.go`
- Create: `app/domain/timeblockapp/route.go`

- [ ] **Step 1: Create model.go**

```go
package timeblockapp

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/timeblockbus"
)

type TimeBlock struct {
	ID        string `json:"id"`
	TaskID    string `json:"taskId"`
	StartsAt  string `json:"startsAt"`
	EndsAt    string `json:"endsAt"`
	Confirmed bool   `json:"confirmed"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func (tb TimeBlock) Encode() ([]byte, string, error) {
	data, err := json.Marshal(tb)
	return data, "application/json", err
}

type NewTimeBlock struct {
	TaskID   string `json:"taskId"`
	StartsAt string `json:"startsAt"`
	EndsAt   string `json:"endsAt"`
}

type UpdateTimeBlock struct {
	StartsAt  *string `json:"startsAt"`
	EndsAt    *string `json:"endsAt"`
	Confirmed *bool   `json:"confirmed"`
}

func toAppTimeBlock(bus timeblockbus.TimeBlock) TimeBlock {
	return TimeBlock{
		ID:        bus.ID.String(),
		TaskID:    bus.TaskID.String(),
		StartsAt:  bus.StartsAt.Format(time.RFC3339),
		EndsAt:    bus.EndsAt.Format(time.RFC3339),
		Confirmed: bus.Confirmed,
		CreatedAt: bus.CreatedAt.Format(time.RFC3339),
		UpdatedAt: bus.UpdatedAt.Format(time.RFC3339),
	}
}

func toAppTimeBlocks(blocks []timeblockbus.TimeBlock) []TimeBlock {
	items := make([]TimeBlock, len(blocks))
	for i, b := range blocks {
		items[i] = toAppTimeBlock(b)
	}
	return items
}

func toBusNewTimeBlock(app NewTimeBlock) (timeblockbus.NewTimeBlock, error) {
	taskID, err := uuid.Parse(app.TaskID)
	if err != nil {
		return timeblockbus.NewTimeBlock{}, fmt.Errorf("parsing task id: %w", err)
	}

	startsAt, err := time.Parse(time.RFC3339, app.StartsAt)
	if err != nil {
		return timeblockbus.NewTimeBlock{}, fmt.Errorf("parsing starts_at: %w", err)
	}

	endsAt, err := time.Parse(time.RFC3339, app.EndsAt)
	if err != nil {
		return timeblockbus.NewTimeBlock{}, fmt.Errorf("parsing ends_at: %w", err)
	}

	return timeblockbus.NewTimeBlock{
		TaskID:   taskID,
		StartsAt: startsAt,
		EndsAt:   endsAt,
	}, nil
}

func toBusUpdateTimeBlock(app UpdateTimeBlock) (timeblockbus.UpdateTimeBlock, error) {
	var ub timeblockbus.UpdateTimeBlock

	if app.StartsAt != nil {
		t, err := time.Parse(time.RFC3339, *app.StartsAt)
		if err != nil {
			return timeblockbus.UpdateTimeBlock{}, fmt.Errorf("parsing starts_at: %w", err)
		}
		ub.StartsAt = &t
	}

	if app.EndsAt != nil {
		t, err := time.Parse(time.RFC3339, *app.EndsAt)
		if err != nil {
			return timeblockbus.UpdateTimeBlock{}, fmt.Errorf("parsing ends_at: %w", err)
		}
		ub.EndsAt = &t
	}

	if app.Confirmed != nil {
		ub.Confirmed = app.Confirmed
	}

	return ub, nil
}
```

- [ ] **Step 2: Create filter.go**

```go
package timeblockapp

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/timeblockbus"
)

func parseFilter(r *http.Request) (timeblockbus.QueryFilter, error) {
	var filter timeblockbus.QueryFilter

	if v := r.URL.Query().Get("task_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return timeblockbus.QueryFilter{}, err
		}
		filter.TaskID = &id
	}

	if v := r.URL.Query().Get("date_from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return timeblockbus.QueryFilter{}, err
		}
		filter.DateFrom = &t
	}

	if v := r.URL.Query().Get("date_to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return timeblockbus.QueryFilter{}, err
		}
		filter.DateTo = &t
	}

	return filter, nil
}
```

- [ ] **Step 3: Create order.go**

```go
package timeblockapp

import (
	"net/http"

	"github.com/casebrophy/planner/business/domain/timeblockbus"
	"github.com/casebrophy/planner/business/sdk/order"
)

var orderByFields = map[string]string{
	"starts_at":  timeblockbus.OrderByStartsAt,
	"created_at": timeblockbus.OrderByCreatedAt,
}

func parseOrder(r *http.Request) (order.By, error) {
	return order.Parse(orderByFields, r.URL.Query().Get("orderBy"), timeblockbus.DefaultOrderBy)
}
```

- [ ] **Step 4: Create timeblockapp.go**

```go
package timeblockapp

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/timeblockbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	timeBlockBus *timeblockbus.Business
}

func (a *app) queryAll(ctx context.Context, r *http.Request) web.Encoder {
	qp := r.URL.Query()

	pg, err := page.Parse(qp.Get("page"), qp.Get("rows"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	filter, err := parseFilter(r)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	orderBy, err := parseOrder(r)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	blocks, err := a.timeBlockBus.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return errs.New(errs.Internal, err)
	}

	total, err := a.timeBlockBus.Count(ctx, filter)
	if err != nil {
		return errs.New(errs.Internal, err)
	}

	return query.NewResult(toAppTimeBlocks(blocks), total, pg.Number, pg.RowsPerPage)
}

func (a *app) queryByID(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "block_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	block, err := a.timeBlockBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.New(errs.Internal, err)
	}

	return toAppTimeBlock(block)
}

func (a *app) create(ctx context.Context, r *http.Request) web.Encoder {
	var anb NewTimeBlock
	if err := web.Decode(r, &anb); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if anb.TaskID == "" || anb.StartsAt == "" || anb.EndsAt == "" {
		return errs.New(errs.InvalidArgument, errors.New("taskId, startsAt, and endsAt are required"))
	}

	bnb, err := toBusNewTimeBlock(anb)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	block, err := a.timeBlockBus.Create(ctx, bnb)
	if err != nil {
		return errs.New(errs.Internal, err)
	}

	return toAppTimeBlock(block)
}

func (a *app) update(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "block_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	block, err := a.timeBlockBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.New(errs.Internal, err)
	}

	var aub UpdateTimeBlock
	if err := web.Decode(r, &aub); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	bub, err := toBusUpdateTimeBlock(aub)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	updated, err := a.timeBlockBus.Update(ctx, block, bub)
	if err != nil {
		return errs.New(errs.Internal, err)
	}

	return toAppTimeBlock(updated)
}

func (a *app) delete(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "block_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	block, err := a.timeBlockBus.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return errs.New(errs.NotFound, err)
		}
		return errs.New(errs.Internal, err)
	}

	if err := a.timeBlockBus.Delete(ctx, block); err != nil {
		return errs.New(errs.Internal, err)
	}

	return web.NoResponse{}
}
```

- [ ] **Step 5: Create route.go**

```go
package timeblockapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/timeblockbus"
	"github.com/casebrophy/planner/business/domain/timeblockbus/stores/timeblockdb"
	"github.com/casebrophy/planner/foundation/web"
)

type Routes struct{}

func (Routes) Add(a *web.App, cfg mux.Config) {
	store := timeblockdb.NewStore(cfg.Log, cfg.DB)
	bus := timeblockbus.NewBusiness(cfg.Log, store)

	hdl := &app{timeBlockBus: bus}

	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/time-blocks", hdl.queryAll, authen)
	a.Handle(http.MethodGet, "/api/v1/time-blocks/{block_id}", hdl.queryByID, authen)
	a.Handle(http.MethodPost, "/api/v1/time-blocks", hdl.create, authen)
	a.Handle(http.MethodPut, "/api/v1/time-blocks/{block_id}", hdl.update, authen)
	a.Handle(http.MethodDelete, "/api/v1/time-blocks/{block_id}", hdl.delete, authen)
}
```

- [ ] **Step 6: Wire into main.go**

Add `timeblockapp.Routes{}` to the `mux.WebAPI(...)` call in `api/services/planner/main.go` and add the import `"github.com/casebrophy/planner/app/domain/timeblockapp"`.

- [ ] **Step 7: Verify it compiles**

```bash
go build ./...
```

Expected: No errors.

- [ ] **Step 8: Commit**

```bash
git add app/domain/timeblockapp/ api/services/planner/main.go
git commit -m "feat: add timeblockapp REST API (CRUD + routes)"
```

---

## Task 6: MCP tools — `get_schedule`, `create_time_block`, `confirm_time_block`

**Files:**
- Modify: `app/domain/mcpapp/mcpapp.go` (add `timeBlockBus` field, 3 tool cases, 3 tool methods, 3 tool definitions)
- Modify: `app/domain/mcpapp/route.go` (wire `timeblockdb.Store` + `timeblockbus.Business`)

- [ ] **Step 1: Add `timeBlockBus` to MCP app struct**

In `app/domain/mcpapp/mcpapp.go`, add field to the `app` struct:

```go
timeBlockBus *timeblockbus.Business
```

Add import: `"github.com/casebrophy/planner/business/domain/timeblockbus"`

- [ ] **Step 2: Add tool definitions to `tools/list` response**

Find the `listTools` method (or the `tools/list` case). Add these three tool definitions to the tools array:

```go
{
    Name:        "get_schedule",
    Description: "Get events and time blocks for a date range. Returns both fixed events and scheduled task blocks merged by time.",
    InputSchema: json.RawMessage(`{"type":"object","properties":{"date_from":{"type":"string","description":"Start date (RFC3339)"},"date_to":{"type":"string","description":"End date (RFC3339)"}},"required":["date_from","date_to"]}`),
},
{
    Name:        "create_time_block",
    Description: "Schedule a task into a specific time slot. Creates a proposed (unconfirmed) time block.",
    InputSchema: json.RawMessage(`{"type":"object","properties":{"task_id":{"type":"string","description":"UUID of the task to schedule"},"starts_at":{"type":"string","description":"Block start time (RFC3339)"},"ends_at":{"type":"string","description":"Block end time (RFC3339)"}},"required":["task_id","starts_at","ends_at"]}`),
},
{
    Name:        "confirm_time_block",
    Description: "Mark a proposed time block as confirmed.",
    InputSchema: json.RawMessage(`{"type":"object","properties":{"block_id":{"type":"string","description":"UUID of the time block to confirm"}},"required":["block_id"]}`),
},
```

- [ ] **Step 3: Add cases to `callTool` switch**

```go
case "get_schedule":
    return a.toolGetSchedule(ctx, params.Arguments)
case "create_time_block":
    return a.toolCreateTimeBlock(ctx, params.Arguments)
case "confirm_time_block":
    return a.toolConfirmTimeBlock(ctx, params.Arguments)
```

- [ ] **Step 4: Implement tool methods**

```go
func (a *app) toolGetSchedule(ctx context.Context, args json.RawMessage) web.Encoder {
	var input struct {
		DateFrom string `json:"date_from"`
		DateTo   string `json:"date_to"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolError("invalid arguments: " + err.Error())
	}

	dateFrom, err := time.Parse(time.RFC3339, input.DateFrom)
	if err != nil {
		return toolError("invalid date_from: " + err.Error())
	}
	dateTo, err := time.Parse(time.RFC3339, input.DateTo)
	if err != nil {
		return toolError("invalid date_to: " + err.Error())
	}

	// Fetch events in range.
	eventFilter := eventbus.QueryFilter{
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
	}
	events, err := a.eventBus.Query(ctx, eventFilter, eventbus.DefaultOrderBy, page.MustParse("1", "1000"))
	if err != nil {
		return toolError("querying events: " + err.Error())
	}

	// Fetch time blocks in range.
	blockFilter := timeblockbus.QueryFilter{
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
	}
	blocks, err := a.timeBlockBus.Query(ctx, blockFilter, timeblockbus.DefaultOrderBy, page.MustParse("1", "1000"))
	if err != nil {
		return toolError("querying time blocks: " + err.Error())
	}

	result := struct {
		Events     []eventbus.Event         `json:"events"`
		TimeBlocks []timeblockbus.TimeBlock `json:"time_blocks"`
	}{
		Events:     events,
		TimeBlocks: blocks,
	}

	return textResult(result)
}

func (a *app) toolCreateTimeBlock(ctx context.Context, args json.RawMessage) web.Encoder {
	var input struct {
		TaskID   string `json:"task_id"`
		StartsAt string `json:"starts_at"`
		EndsAt   string `json:"ends_at"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolError("invalid arguments: " + err.Error())
	}

	taskID, err := uuid.Parse(input.TaskID)
	if err != nil {
		return toolError("invalid task_id: " + err.Error())
	}

	startsAt, err := time.Parse(time.RFC3339, input.StartsAt)
	if err != nil {
		return toolError("invalid starts_at: " + err.Error())
	}

	endsAt, err := time.Parse(time.RFC3339, input.EndsAt)
	if err != nil {
		return toolError("invalid ends_at: " + err.Error())
	}

	block, err := a.timeBlockBus.Create(ctx, timeblockbus.NewTimeBlock{
		TaskID:   taskID,
		StartsAt: startsAt,
		EndsAt:   endsAt,
	})
	if err != nil {
		return toolError("creating time block: " + err.Error())
	}

	return textResult(block)
}

func (a *app) toolConfirmTimeBlock(ctx context.Context, args json.RawMessage) web.Encoder {
	var input struct {
		BlockID string `json:"block_id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolError("invalid arguments: " + err.Error())
	}

	blockID, err := uuid.Parse(input.BlockID)
	if err != nil {
		return toolError("invalid block_id: " + err.Error())
	}

	block, err := a.timeBlockBus.QueryByID(ctx, blockID)
	if err != nil {
		return toolError("block not found: " + err.Error())
	}

	confirmed := true
	updated, err := a.timeBlockBus.Update(ctx, block, timeblockbus.UpdateTimeBlock{
		Confirmed: &confirmed,
	})
	if err != nil {
		return toolError("confirming block: " + err.Error())
	}

	return textResult(updated)
}
```

- [ ] **Step 5: Wire in route.go**

In `app/domain/mcpapp/route.go`, add:

```go
import "github.com/casebrophy/planner/business/domain/timeblockbus"
import "github.com/casebrophy/planner/business/domain/timeblockbus/stores/timeblockdb"
```

In the `Add` function body, add:

```go
tbStore := timeblockdb.NewStore(cfg.Log, cfg.DB)
tbBus := timeblockbus.NewBusiness(cfg.Log, tbStore)
```

And add `timeBlockBus: tbBus,` to the `&app{...}` struct literal.

- [ ] **Step 6: Verify it compiles**

```bash
go build ./...
```

Expected: No errors.

- [ ] **Step 7: Commit**

```bash
git add app/domain/mcpapp/
git commit -m "feat: add MCP tools for schedule (get_schedule, create_time_block, confirm_time_block)"
```

---

## Task 7: Frontend — types, service, store

**Files:**
- Create: `api/services/frontend/web/src/types/timeBlock.ts`
- Create: `api/services/frontend/web/src/services/timeBlockService.ts`
- Create: `api/services/frontend/web/src/stores/timeBlockStore.ts`

- [ ] **Step 1: Create types/timeBlock.ts**

```typescript
export interface TimeBlock {
  id: string
  taskId: string
  startsAt: string
  endsAt: string
  confirmed: boolean
  createdAt: string
  updatedAt: string
}

export interface NewTimeBlock {
  taskId: string
  startsAt: string
  endsAt: string
}

export interface UpdateTimeBlock {
  startsAt?: string
  endsAt?: string
  confirmed?: boolean
}

export interface TimeBlockFilter {
  taskId?: string
  dateFrom?: string
  dateTo?: string
}
```

- [ ] **Step 2: Create services/timeBlockService.ts**

```typescript
import { createCRUDService } from './createCRUDService'
import type { TimeBlock, NewTimeBlock, UpdateTimeBlock, TimeBlockFilter } from '@/types/timeBlock'

export const timeBlockService = createCRUDService<TimeBlock, NewTimeBlock, UpdateTimeBlock, TimeBlockFilter>({
  basePath: '/api/v1/time-blocks',
  mapFilter: (f) => ({
    task_id: f.taskId,
    date_from: f.dateFrom,
    date_to: f.dateTo,
  }),
})
```

- [ ] **Step 3: Create stores/timeBlockStore.ts**

```typescript
import { defineStore } from 'pinia'
import { createCRUDStore } from './createCRUDStore'
import { timeBlockService } from '@/services/timeBlockService'
import type { TimeBlock, NewTimeBlock, UpdateTimeBlock, TimeBlockFilter } from '@/types/timeBlock'

export const useTimeBlockStore = defineStore('timeBlock', () => {
  const crud = createCRUDStore<TimeBlock, NewTimeBlock, UpdateTimeBlock, TimeBlockFilter>({
    name: 'timeBlock',
    service: timeBlockService,
    defaultOrderBy: 'starts_at',
    defaultRowsPerPage: 100,
  })

  return { ...crud }
})
```

- [ ] **Step 4: Commit**

```bash
git add api/services/frontend/web/src/types/timeBlock.ts api/services/frontend/web/src/services/timeBlockService.ts api/services/frontend/web/src/stores/timeBlockStore.ts
git commit -m "feat: add time block types, service, and store"
```

---

## Task 8: Frontend — schedule composable

**Files:**
- Create: `api/services/frontend/web/src/composables/useSchedule.ts`

- [ ] **Step 1: Create useSchedule.ts**

This composable merges events and time blocks for the current week, provides week navigation, and exposes the data the calendar view needs.

```typescript
import { ref, computed, onMounted } from 'vue'
import { useCalendarEventStore } from '@/stores/calendarEventStore'
import { useTimeBlockStore } from '@/stores/timeBlockStore'
import { useTaskStore } from '@/stores/taskStore'
import { usePolling } from './usePolling'
import type { CalendarEvent } from '@/types/calendarEvent'
import type { TimeBlock } from '@/types/timeBlock'
import type { Task } from '@/types/task'

export interface ScheduleItem {
  type: 'event' | 'timeBlock'
  id: string
  title: string
  startsAt: Date
  endsAt: Date
  allDay?: boolean
  confirmed?: boolean
  taskId?: string
  location?: string
  event?: CalendarEvent
  timeBlock?: TimeBlock
  task?: Task
}

export function useSchedule() {
  const eventStore = useCalendarEventStore()
  const timeBlockStore = useTimeBlockStore()
  const taskStore = useTaskStore()

  const weekOffset = ref(0)

  const weekStart = computed(() => {
    const now = new Date()
    const day = now.getDay()
    const diff = now.getDate() - day + (day === 0 ? -6 : 1) // Monday start
    const start = new Date(now)
    start.setDate(diff + weekOffset.value * 7)
    start.setHours(0, 0, 0, 0)
    return start
  })

  const weekEnd = computed(() => {
    const end = new Date(weekStart.value)
    end.setDate(end.getDate() + 7)
    end.setHours(0, 0, 0, 0)
    return end
  })

  const weekDays = computed(() => {
    const days: Date[] = []
    for (let i = 0; i < 7; i++) {
      const d = new Date(weekStart.value)
      d.setDate(d.getDate() + i)
      days.push(d)
    }
    return days
  })

  const weekLabel = computed(() => {
    const start = weekStart.value
    const end = new Date(start)
    end.setDate(end.getDate() + 6)
    const opts: Intl.DateTimeFormatOptions = { month: 'short', day: 'numeric' }
    return `${start.toLocaleDateString(undefined, opts)} - ${end.toLocaleDateString(undefined, opts)}`
  })

  const isCurrentWeek = computed(() => weekOffset.value === 0)

  function prevWeek() { weekOffset.value-- }
  function nextWeek() { weekOffset.value++ }
  function goToToday() { weekOffset.value = 0 }

  const scheduleItems = computed<ScheduleItem[]>(() => {
    const items: ScheduleItem[] = []

    for (const ev of eventStore.items) {
      items.push({
        type: 'event',
        id: ev.id,
        title: ev.title,
        startsAt: new Date(ev.startsAt),
        endsAt: new Date(ev.endsAt),
        allDay: ev.allDay,
        location: ev.location,
        event: ev,
      })
    }

    const tasks = taskStore.items
    const taskMap = new Map<string, Task>()
    for (const t of tasks) {
      taskMap.set(t.id, t)
    }

    for (const tb of timeBlockStore.items) {
      const task = taskMap.get(tb.taskId)
      items.push({
        type: 'timeBlock',
        id: tb.id,
        title: task?.title ?? 'Untitled task',
        startsAt: new Date(tb.startsAt),
        endsAt: new Date(tb.endsAt),
        confirmed: tb.confirmed,
        taskId: tb.taskId,
        timeBlock: tb,
        task,
      })
    }

    items.sort((a, b) => a.startsAt.getTime() - b.startsAt.getTime())
    return items
  })

  function itemsForDay(day: Date): ScheduleItem[] {
    const dayStart = new Date(day)
    dayStart.setHours(0, 0, 0, 0)
    const dayEnd = new Date(day)
    dayEnd.setHours(23, 59, 59, 999)

    return scheduleItems.value.filter(item => {
      return item.startsAt <= dayEnd && item.endsAt >= dayStart
    })
  }

  async function load() {
    const from = weekStart.value.toISOString()
    const to = weekEnd.value.toISOString()

    eventStore.setFilter({ dateFrom: from, dateTo: to })
    timeBlockStore.setFilter({ dateFrom: from, dateTo: to })

    await Promise.all([
      eventStore.fetchList(true),
      timeBlockStore.fetchList(true),
      taskStore.fetchList(),
    ])
  }

  onMounted(load)
  usePolling(load)

  return {
    weekOffset,
    weekStart,
    weekEnd,
    weekDays,
    weekLabel,
    isCurrentWeek,
    prevWeek,
    nextWeek,
    goToToday,
    scheduleItems,
    itemsForDay,
    load,
    loading: computed(() => eventStore.loading || timeBlockStore.loading),
  }
}
```

- [ ] **Step 2: Commit**

```bash
git add api/services/frontend/web/src/composables/useSchedule.ts
git commit -m "feat: add useSchedule composable (merges events + time blocks)"
```

---

## Task 9: Frontend — ScheduleBlock component

**Files:**
- Create: `api/services/frontend/web/src/components/schedule/ScheduleBlock.vue`

- [ ] **Step 1: Create ScheduleBlock.vue**

A positioned block that renders inside the day column. Handles both events (blue) and time blocks (green/gray).

```vue
<script setup lang="ts">
import type { ScheduleItem } from '@/composables/useSchedule'
import { computed } from 'vue'

const props = defineProps<{
  item: ScheduleItem
  dayStart: Date
}>()

const emit = defineEmits<{
  click: [item: ScheduleItem]
  delete: [item: ScheduleItem]
}>()

const startHour = 7
const endHour = 22
const totalMinutes = (endHour - startHour) * 60

const topPercent = computed(() => {
  const itemStart = props.item.startsAt
  const minutesFromStart = (itemStart.getHours() - startHour) * 60 + itemStart.getMinutes()
  return Math.max(0, (minutesFromStart / totalMinutes) * 100)
})

const heightPercent = computed(() => {
  const durationMs = props.item.endsAt.getTime() - props.item.startsAt.getTime()
  const durationMin = durationMs / 60000
  return Math.max(2, (durationMin / totalMinutes) * 100)
})

const timeLabel = computed(() => {
  const fmt = (d: Date) => d.toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' })
  return `${fmt(props.item.startsAt)} - ${fmt(props.item.endsAt)}`
})

const bgClass = computed(() => {
  if (props.item.type === 'event') return 'bg-blue-600/80 border-blue-500'
  if (props.item.confirmed) return 'bg-emerald-600/80 border-emerald-500'
  return 'bg-gray-700/80 border-gray-600 border-dashed'
})
</script>

<template>
  <div
    class="absolute left-0.5 right-0.5 rounded px-1.5 py-0.5 text-xs border cursor-pointer overflow-hidden hover:brightness-110 transition-all"
    :class="bgClass"
    :style="{ top: topPercent + '%', height: heightPercent + '%', minHeight: '20px' }"
    @click="emit('click', item)"
  >
    <div class="font-medium text-gray-100 truncate">{{ item.title }}</div>
    <div class="text-gray-300 truncate text-[10px]">{{ timeLabel }}</div>
    <div v-if="item.location" class="text-gray-400 truncate text-[10px]">{{ item.location }}</div>
  </div>
</template>
```

- [ ] **Step 2: Commit**

```bash
git add api/services/frontend/web/src/components/schedule/ScheduleBlock.vue
git commit -m "feat: add ScheduleBlock component"
```

---

## Task 10: Frontend — TimeBlockForm component

**Files:**
- Create: `api/services/frontend/web/src/components/schedule/TimeBlockForm.vue`

- [ ] **Step 1: Create TimeBlockForm.vue**

Form to create or edit a time block. Task is selected from a dropdown of open tasks.

```vue
<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useTaskStore } from '@/stores/taskStore'
import type { TimeBlock } from '@/types/timeBlock'

const props = defineProps<{
  block?: TimeBlock | null
  initialDate?: string
}>()

const emit = defineEmits<{
  save: [data: { taskId: string; startsAt: string; endsAt: string; confirmed?: boolean }]
  cancel: []
}>()

const taskStore = useTaskStore()

const taskId = ref(props.block?.taskId ?? '')
const startsAt = ref('')
const endsAt = ref('')
const confirmed = ref(props.block?.confirmed ?? false)

onMounted(async () => {
  await taskStore.fetchList()

  if (props.block) {
    startsAt.value = toDatetimeLocal(props.block.startsAt)
    endsAt.value = toDatetimeLocal(props.block.endsAt)
  } else if (props.initialDate) {
    const d = new Date(props.initialDate)
    d.setHours(9, 0, 0, 0)
    startsAt.value = toDatetimeLocal(d.toISOString())
    const end = new Date(d)
    end.setHours(10, 0, 0, 0)
    endsAt.value = toDatetimeLocal(end.toISOString())
  }
})

function toDatetimeLocal(iso: string): string {
  const d = new Date(iso)
  const pad = (n: number) => n.toString().padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const openTasks = computed(() =>
  taskStore.items.filter(t => t.status === 'todo' || t.status === 'in_progress')
)

const isValid = computed(() => taskId.value && startsAt.value && endsAt.value)

function handleSubmit() {
  if (!isValid.value) return
  emit('save', {
    taskId: taskId.value,
    startsAt: new Date(startsAt.value).toISOString(),
    endsAt: new Date(endsAt.value).toISOString(),
    confirmed: confirmed.value,
  })
}
</script>

<template>
  <form @submit.prevent="handleSubmit" class="space-y-4">
    <div>
      <label class="block text-sm text-gray-400 mb-1">Task</label>
      <select
        v-model="taskId"
        class="w-full bg-gray-800 border border-gray-700 rounded px-3 py-2 text-gray-100"
        :disabled="!!block"
      >
        <option value="" disabled>Select a task...</option>
        <option v-for="task in openTasks" :key="task.id" :value="task.id">
          {{ task.title }}
        </option>
      </select>
    </div>

    <div class="grid grid-cols-2 gap-3">
      <div>
        <label class="block text-sm text-gray-400 mb-1">Start</label>
        <input
          v-model="startsAt"
          type="datetime-local"
          class="w-full bg-gray-800 border border-gray-700 rounded px-3 py-2 text-gray-100"
        />
      </div>
      <div>
        <label class="block text-sm text-gray-400 mb-1">End</label>
        <input
          v-model="endsAt"
          type="datetime-local"
          class="w-full bg-gray-800 border border-gray-700 rounded px-3 py-2 text-gray-100"
        />
      </div>
    </div>

    <div class="flex items-center gap-2">
      <input
        v-model="confirmed"
        type="checkbox"
        id="confirmed"
        class="rounded bg-gray-800 border-gray-700"
      />
      <label for="confirmed" class="text-sm text-gray-300">Confirmed</label>
    </div>

    <div class="flex justify-end gap-2 pt-2">
      <button
        type="button"
        @click="emit('cancel')"
        class="px-4 py-2 text-sm text-gray-400 hover:text-gray-200"
      >
        Cancel
      </button>
      <button
        type="submit"
        :disabled="!isValid"
        class="px-4 py-2 text-sm bg-blue-600 text-white rounded hover:bg-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {{ block ? 'Update' : 'Schedule' }}
      </button>
    </div>
  </form>
</template>
```

- [ ] **Step 2: Commit**

```bash
git add api/services/frontend/web/src/components/schedule/TimeBlockForm.vue
git commit -m "feat: add TimeBlockForm component"
```

---

## Task 11: Frontend — ScheduleView (weekly calendar)

**Files:**
- Create: `api/services/frontend/web/src/views/ScheduleView.vue`
- Modify: `api/services/frontend/web/src/router/index.ts`
- Modify: `api/services/frontend/web/src/components/layout/AppSidebar.vue`

- [ ] **Step 1: Create ScheduleView.vue**

Weekly calendar view: 7-column grid with hour markers, events and time blocks positioned absolutely.

```vue
<script setup lang="ts">
import { ref, watch } from 'vue'
import { useSchedule, type ScheduleItem } from '@/composables/useSchedule'
import { useTimeBlockStore } from '@/stores/timeBlockStore'
import { useToastStore } from '@/stores/toastStore'
import PageHeader from '@/components/layout/PageHeader.vue'
import DrawerPanel from '@/components/shared/DrawerPanel.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import ScheduleBlock from '@/components/schedule/ScheduleBlock.vue'
import TimeBlockForm from '@/components/schedule/TimeBlockForm.vue'
import type { TimeBlock } from '@/types/timeBlock'

const {
  weekDays,
  weekLabel,
  isCurrentWeek,
  prevWeek,
  nextWeek,
  goToToday,
  itemsForDay,
  loading,
  load,
} = useSchedule()

const timeBlockStore = useTimeBlockStore()
const toastStore = useToastStore()

const showForm = ref(false)
const editingBlock = ref<TimeBlock | null>(null)
const initialDate = ref<string | undefined>()

const startHour = 7
const endHour = 22
const hours = Array.from({ length: endHour - startHour }, (_, i) => startHour + i)

function openCreateForm(day?: Date) {
  editingBlock.value = null
  initialDate.value = day?.toISOString()
  showForm.value = true
}

function handleBlockClick(item: ScheduleItem) {
  if (item.type === 'timeBlock' && item.timeBlock) {
    editingBlock.value = item.timeBlock
    showForm.value = true
  }
}

async function handleSave(data: { taskId: string; startsAt: string; endsAt: string; confirmed?: boolean }) {
  try {
    if (editingBlock.value) {
      await timeBlockStore.update(editingBlock.value.id, {
        startsAt: data.startsAt,
        endsAt: data.endsAt,
        confirmed: data.confirmed,
      })
      toastStore.success('Time block updated')
    } else {
      await timeBlockStore.create({
        taskId: data.taskId,
        startsAt: data.startsAt,
        endsAt: data.endsAt,
      })
      toastStore.success('Time block created')
    }
    showForm.value = false
    editingBlock.value = null
    await load()
  } catch (e) {
    toastStore.error('Failed to save time block')
  }
}

function closeForm() {
  showForm.value = false
  editingBlock.value = null
}

const dayNames = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']

function isToday(day: Date): boolean {
  const now = new Date()
  return day.getDate() === now.getDate() &&
    day.getMonth() === now.getMonth() &&
    day.getFullYear() === now.getFullYear()
}

// Reload when week changes.
watch(() => weekDays.value[0]?.toISOString(), () => {
  load()
})
</script>

<template>
  <div class="h-full flex flex-col">
    <PageHeader title="Schedule" :subtitle="weekLabel">
      <template #actions>
        <div class="flex items-center gap-2">
          <button
            @click="prevWeek"
            class="p-2 text-gray-400 hover:text-gray-200 rounded hover:bg-gray-800"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/></svg>
          </button>
          <button
            v-if="!isCurrentWeek"
            @click="goToToday"
            class="px-3 py-1.5 text-sm text-gray-300 hover:text-gray-100 rounded hover:bg-gray-800"
          >
            Today
          </button>
          <button
            @click="nextWeek"
            class="p-2 text-gray-400 hover:text-gray-200 rounded hover:bg-gray-800"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/></svg>
          </button>
          <button
            @click="openCreateForm()"
            class="ml-2 px-3 py-1.5 text-sm bg-blue-600 text-white rounded hover:bg-blue-500"
          >
            + Schedule
          </button>
        </div>
      </template>
    </PageHeader>

    <LoadingSpinner v-if="loading" />

    <div v-else class="flex-1 overflow-auto">
      <!-- Day headers -->
      <div class="grid grid-cols-[60px_repeat(7,1fr)] border-b border-gray-800 sticky top-0 bg-gray-950 z-10">
        <div class="p-2"></div>
        <div
          v-for="(day, i) in weekDays"
          :key="i"
          class="p-2 text-center border-l border-gray-800"
          :class="isToday(day) ? 'text-blue-400' : 'text-gray-400'"
        >
          <div class="text-xs font-medium">{{ dayNames[i] }}</div>
          <div class="text-lg" :class="isToday(day) ? 'font-bold' : ''">{{ day.getDate() }}</div>
        </div>
      </div>

      <!-- Time grid -->
      <div class="grid grid-cols-[60px_repeat(7,1fr)]">
        <!-- Hour labels + day columns -->
        <template v-for="hour in hours" :key="hour">
          <!-- Hour label -->
          <div class="h-16 border-b border-gray-800/50 pr-2 text-right">
            <span class="text-[10px] text-gray-500 -mt-2 block">
              {{ hour === 12 ? '12 PM' : hour > 12 ? `${hour - 12} PM` : `${hour} AM` }}
            </span>
          </div>
          <!-- Day cells for this hour -->
          <div
            v-for="(day, di) in weekDays"
            :key="`${hour}-${di}`"
            class="h-16 border-l border-b border-gray-800/50 relative"
            @dblclick="openCreateForm(day)"
          ></div>
        </template>

        <!-- Overlay: positioned schedule blocks -->
        <!-- These are placed in each day column using absolute positioning -->
      </div>

      <!-- Positioned blocks overlay -->
      <div class="grid grid-cols-[60px_repeat(7,1fr)] absolute inset-0" style="top: calc(2.5rem + 1px); pointer-events: none;">
        <div></div>
        <div
          v-for="(day, di) in weekDays"
          :key="`blocks-${di}`"
          class="relative border-l border-gray-800/0"
          :style="{ height: `${hours.length * 4}rem` }"
        >
          <ScheduleBlock
            v-for="item in itemsForDay(day)"
            :key="item.id"
            :item="item"
            :day-start="day"
            style="pointer-events: auto;"
            @click="handleBlockClick"
          />
        </div>
      </div>
    </div>

    <DrawerPanel :open="showForm" title="Schedule Task" @close="closeForm">
      <TimeBlockForm
        :block="editingBlock"
        :initial-date="initialDate"
        @save="handleSave"
        @cancel="closeForm"
      />
    </DrawerPanel>
  </div>
</template>
```

- [ ] **Step 2: Add route to router/index.ts**

Add to the routes array (after the `/events` route):

```typescript
{
  path: '/schedule',
  name: 'schedule',
  component: () => import('@/views/ScheduleView.vue'),
},
```

- [ ] **Step 3: Add nav item to AppSidebar.vue**

Add to the `navItems` array (after the Events entry):

```typescript
{ name: 'Schedule', path: '/schedule', icon: 'schedule' },
```

Add a calendar/schedule SVG icon in the icon rendering section. Use:

```html
<svg v-else-if="item.icon === 'schedule'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/>
</svg>
```

- [ ] **Step 4: Verify frontend builds**

```bash
cd api/services/frontend/web && npm run build
```

Expected: Build succeeds with no TypeScript errors.

- [ ] **Step 5: Commit**

```bash
git add api/services/frontend/web/src/views/ScheduleView.vue api/services/frontend/web/src/router/index.ts api/services/frontend/web/src/components/layout/AppSidebar.vue
git commit -m "feat: add weekly ScheduleView with calendar grid"
```

---

## Task 12: Arch doc + TOC update

**Files:**
- Create: `.docs/arch/timeblock-backend.md`
- Modify: `.docs/TOC.md`

- [ ] **Step 1: Generate arch doc**

Use the `go-arch` skill (or write manually) to create `.docs/arch/timeblock-backend.md` documenting the new `timeblockbus` domain: core types, file map, impact callouts, routes, cross-domain dependencies.

- [ ] **Step 2: Update TOC.md**

Add to the "By Domain" section:

```
- timeblock: `03-data-model.md#time-blocks`, `05-context-engine.md#scheduling`, `07-roadmap.md#phase-7b--calendar-view--time-blocks`, `arch/timeblock-backend.md`
```

Update the "By Schema" section — change the `time_blocks` line from `(future — Phase 7b)` to just the reference:

```
- time_blocks: `03-data-model.md#time-blocks`, `arch/timeblock-backend.md`
```

Add to "Implementation Plans":

```
- phase-7b-calendar-time-blocks: `plans/phase7b-calendar-time-blocks.md`
```

- [ ] **Step 3: Update 07-roadmap.md**

Change Phase 7b status to reflect implementation:

```
## Phase 7b — Calendar View + Time Blocks  ⚠️ Partial
```

Add a `**Remaining:**` note at the bottom of the section.

- [ ] **Step 4: Commit**

```bash
git add .docs/
git commit -m "docs: add timeblock arch doc, update TOC and roadmap for Phase 7b"
```

---

## Task 13: Integration test — full stack smoke test

**Files:**
- Run manually (no new test files — uses existing `make test` and `make dev` flows)

- [ ] **Step 1: Run all backend tests**

```bash
make test
```

Expected: All existing tests pass, new timeblockbus tests pass.

- [ ] **Step 2: Run lint**

```bash
make lint
```

Expected: No vet errors.

- [ ] **Step 3: Start the server and test REST API**

```bash
make dev &
sleep 3

# Create a task first
TASK=$(curl -s -X POST http://localhost:8080/api/v1/tasks \
  -H "X-API-Key: devkey123" \
  -H "Content-Type: application/json" \
  -d '{"title":"Test task","status":"todo","priority":"medium"}')
TASK_ID=$(echo $TASK | jq -r '.id')

# Create a time block
curl -s -X POST http://localhost:8080/api/v1/time-blocks \
  -H "X-API-Key: devkey123" \
  -H "Content-Type: application/json" \
  -d "{\"taskId\":\"$TASK_ID\",\"startsAt\":\"2026-04-02T09:00:00Z\",\"endsAt\":\"2026-04-02T10:00:00Z\"}"

# Query time blocks
curl -s http://localhost:8080/api/v1/time-blocks?page=1\&rows=10 \
  -H "X-API-Key: devkey123"
```

Expected: Time block creates and queries successfully.

- [ ] **Step 4: Test MCP tools**

```bash
# get_schedule
curl -s -X POST http://localhost:8080/mcp \
  -H "X-API-Key: devkey123" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_schedule","arguments":{"date_from":"2026-04-01T00:00:00Z","date_to":"2026-04-07T23:59:59Z"}}}'
```

Expected: Returns events and time_blocks arrays.

- [ ] **Step 5: Verify frontend**

```bash
cd api/services/frontend/web && npm run build && npm run dev &
```

Navigate to `http://localhost:5173/schedule` — weekly grid should render.

- [ ] **Step 6: Stop dev server, commit any fixes**

```bash
kill %1 %2 2>/dev/null
```
