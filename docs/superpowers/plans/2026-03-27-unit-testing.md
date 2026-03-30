# Unit Testing — All Backend Layers Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a complete, CI-ready test suite covering the business layer (all 10 domains) and app/HTTP layer (all 8 HTTP domains), following ardanlabs/service patterns with real Postgres via Docker.

**Architecture:** Three SDK packages (`dbtest`, `unitest`, `apitest`) provide the testing infrastructure. Each domain gets a `testutil.go` with seed helpers and a `_test.go` with table-driven tests. HTTP tests live under `app/domain/<name>app/tests/<name>api/`.

**Tech Stack:** `testcontainers-go` (Docker Postgres), `google/go-cmp` (diff), `net/http/httptest` (HTTP), `mux.WebAPI` (full handler for apitest).

**Reference:** ardanlabs/service is in Go module cache at `/Users/casebrophy/go/pkg/mod/github.com/ardanlabs/service@v0.0.0-20241014201603-3f8c756aba53/`

---

## File Map

### New SDK packages
| File | Responsibility |
|------|---------------|
| `business/sdk/dbtest/dbtest.go` | Spin up Docker Postgres, run migrations, return *Database, cleanup |
| `business/sdk/dbtest/business.go` | Wire all 10 *bus.Business instances into BusDomain |
| `business/sdk/dbtest/model.go` | Database and BusDomain struct types |
| `business/sdk/unitest/unittest.go` | Table-driven test runner: Run(t, []Table, name) |
| `business/sdk/unitest/model.go` | Table struct |
| `app/sdk/apitest/apitest.go` | HTTP test runner using mux.WebAPI + httptest |
| `app/sdk/apitest/model.go` | Table struct for HTTP tests |

### Per-domain business tests (10 domains)
| Domain | Files |
|--------|-------|
| taskbus | `business/domain/taskbus/testutil.go`, `taskbus_test.go` |
| contextbus | `business/domain/contextbus/testutil.go`, `contextbus_test.go` |
| tagbus | `business/domain/tagbus/testutil.go`, `tagbus_test.go` |
| clarificationbus | `business/domain/clarificationbus/testutil.go`, `clarificationbus_test.go` |
| emailbus | `business/domain/emailbus/testutil.go`, `emailbus_test.go` |
| rawinputbus | `business/domain/rawinputbus/testutil.go`, `rawinputbus_test.go` |
| threadbus | `business/domain/threadbus/testutil.go`, `threadbus_test.go` |
| observationbus | `business/domain/observationbus/testutil.go`, `observationbus_test.go` |
| ingestbus | `business/domain/ingestbus/testutil.go`, `ingestbus_test.go` (replace stub) |
| inactivitybus | `business/domain/inactivitybus/testutil.go`, `inactivitybus_test.go` |

### Per-domain HTTP tests (8 domains)
Each domain has `app/domain/<name>app/tests/<name>api/` with:
- `<name>_test.go` — entry point
- `seed_test.go` — insertSeedData
- `query_test.go`, `create_test.go`, `update_test.go`, `delete_test.go` (as applicable)

---

## Task 1: Add Dependencies

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add testcontainers-go and go-cmp**

```bash
cd /Users/casebrophy/planner
go get github.com/testcontainers/testcontainers-go
go get github.com/testcontainers/testcontainers-go/wait
go get github.com/google/go-cmp/cmp
go get github.com/google/go-cmp/cmp/cmpopts
```

- [ ] **Step 2: Verify build still compiles**

```bash
go build ./...
```
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add testcontainers-go and go-cmp test dependencies"
```

---

## Task 2: Create `business/sdk/dbtest` Package

**Files:**
- Create: `business/sdk/dbtest/model.go`
- Create: `business/sdk/dbtest/business.go`
- Create: `business/sdk/dbtest/dbtest.go`

- [ ] **Step 1: Create model.go**

```go
// business/sdk/dbtest/model.go
package dbtest

import (
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/emailbus"
	"github.com/casebrophy/planner/business/domain/inactivitybus"
	"github.com/casebrophy/planner/business/domain/ingestbus"
	"github.com/casebrophy/planner/business/domain/observationbus"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/tagbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/foundation/logger"
)

// Database owns state for running and shutting down tests.
type Database struct {
	DB        *sqlx.DB
	Log       *logger.Logger
	BusDomain BusDomain
}

// BusDomain holds all business domain instances for test use.
type BusDomain struct {
	Task          *taskbus.Business
	Context       *contextbus.Business
	Tag           *tagbus.Business
	Clarification *clarificationbus.Business
	Email         *emailbus.Business
	RawInput      *rawinputbus.Business
	Thread        *threadbus.Business
	Observation   *observationbus.Business
	Ingest        *ingestbus.Business
	Inactivity    *inactivitybus.Business
}
```

- [ ] **Step 2: Create business.go**

```go
// business/sdk/dbtest/business.go
package dbtest

import (
	clardb "github.com/casebrophy/planner/business/domain/clarificationbus/stores/clarificationdb"
	ctxdb "github.com/casebrophy/planner/business/domain/contextbus/stores/contextdb"
	emaildb "github.com/casebrophy/planner/business/domain/emailbus/stores/emaildb"
	inactdb "github.com/casebrophy/planner/business/domain/inactivitybus/stores/inactivitydb"
	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
	obsdb "github.com/casebrophy/planner/business/domain/observationbus/stores/observationdb"
	rawdb "github.com/casebrophy/planner/business/domain/rawinputbus/stores/rawinputdb"
	tagdb "github.com/casebrophy/planner/business/domain/tagbus/stores/tagdb"
	taskdb "github.com/casebrophy/planner/business/domain/taskbus/stores/taskdb"
	threaddb "github.com/casebrophy/planner/business/domain/threadbus/stores/threaddb"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/emailbus"
	"github.com/casebrophy/planner/business/domain/inactivitybus"
	"github.com/casebrophy/planner/business/domain/ingestbus"
	"github.com/casebrophy/planner/business/domain/observationbus"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/tagbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/jmoiron/sqlx"
)

func newBusDomains(log *logger.Logger, db *sqlx.DB) BusDomain {
	taskBus := taskbus.NewBusiness(log, taskdb.NewStore(log, db))
	contextBus := contextbus.NewBusiness(log, ctxdb.NewStore(log, db))
	tagBus := tagbus.NewBusiness(log, tagdb.New(log, db))
	clarBus := clarificationbus.NewBusiness(log, clardb.NewStore(log, db))
	emailBus := emailbus.NewBusiness(log, emaildb.NewStore(log, db))
	rawBus := rawinputbus.NewBusiness(log, rawdb.NewStore(log, db))
	threadBus := threadbus.NewBusiness(log, threaddb.NewStore(log, db))
	obsBus := observationbus.NewBusiness(log, obsdb.NewStore(log, db))
	ingestBus := ingestbus.NewBusiness(log, rawBus, emailBus, taskBus, contextBus, clarBus, &extractor.MockExtractor{})
	inactBus := inactivitybus.NewBusiness(log, inactdb.NewStore(log, db), clarBus)

	return BusDomain{
		Task:          taskBus,
		Context:       contextBus,
		Tag:           tagBus,
		Clarification: clarBus,
		Email:         emailBus,
		RawInput:      rawBus,
		Thread:        threadBus,
		Observation:   obsBus,
		Ingest:        ingestBus,
		Inactivity:    inactBus,
	}
}
```

> **Note:** `tagdb.New` (not `tagdb.NewStore`) — verify the exact constructor name by checking `business/domain/tagbus/stores/tagdb/tagdb.go`.

- [ ] **Step 3: Create dbtest.go**

```go
// business/sdk/dbtest/dbtest.go
package dbtest

import (
	"bytes"
	"context"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcwait "github.com/testcontainers/testcontainers-go/wait"

	"github.com/casebrophy/planner/business/sdk/migrate"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/business/sdk/sqldb"
)

const (
	dbUser     = "postgres"
	dbPassword = "postgres"
	dbName     = "postgres"
)

// New creates a fresh Docker Postgres, runs migrations, wires all business
// domains, and registers t.Cleanup to tear everything down.
func New(t *testing.T, testName string) *Database {
	t.Helper()
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:16",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     dbUser,
				"POSTGRES_PASSWORD": dbPassword,
				"POSTGRES_DB":       dbName,
			},
			WaitingFor: tcwait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("Starting postgres container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("Getting container host: %v", err)
	}

	mappedPort, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("Getting container port: %v", err)
	}

	port, _ := strconv.Atoi(mappedPort.Port())

	// Connect to the default postgres database to create a test-specific DB.
	dbM, err := sqldb.Open(sqldb.Config{
		Host:       host,
		Port:       port,
		User:       dbUser,
		Password:   dbPassword,
		Name:       dbName,
		DisableTLS: true,
	})
	if err != nil {
		t.Fatalf("Opening master DB connection: %v", err)
	}

	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := sqldb.StatusCheck(checkCtx, dbM); err != nil {
		t.Fatalf("DB status check failed: %v", err)
	}

	// Create a randomly-named database so parallel tests don't conflict.
	const letters = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, 6)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	testDBName := string(b)

	t.Logf("Creating test database: %s", testDBName)
	if _, err := dbM.ExecContext(ctx, "CREATE DATABASE "+testDBName); err != nil {
		t.Fatalf("Creating test database: %v", err)
	}

	db, err := sqldb.Open(sqldb.Config{
		Host:       host,
		Port:       port,
		User:       dbUser,
		Password:   dbPassword,
		Name:       testDBName,
		DisableTLS: true,
	})
	if err != nil {
		t.Fatalf("Opening test DB connection: %v", err)
	}

	t.Logf("Migrating test database: %s", testDBName)
	if err := migrate.Migrate(ctx, db); err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	var buf bytes.Buffer
	log := logger.New(&buf, logger.LevelInfo, "TEST")

	t.Cleanup(func() {
		t.Helper()
		t.Logf("Dropping test database: %s", testDBName)
		if _, err := dbM.ExecContext(ctx, "DROP DATABASE "+testDBName+" WITH (FORCE)"); err != nil {
			t.Logf("Warning dropping DB: %v", err)
		}
		db.Close()
		dbM.Close()
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Warning terminating container: %v", err)
		}
		t.Logf("=== TEST LOGS (%s) ===\n%s", testName, buf.String())
	})

	return &Database{
		DB:        db,
		Log:       log,
		BusDomain: newBusDomains(log, db),
	}
}
```

- [ ] **Step 4: Verify package compiles**

```bash
go build ./business/sdk/dbtest/...
```
Expected: no errors. Fix any import path issues (e.g., `tagdb.New` vs `tagdb.NewStore`).

- [ ] **Step 5: Commit**

```bash
git add business/sdk/dbtest/
git commit -m "test: add dbtest SDK package with Docker Postgres setup"
```

---

## Task 3: Create `business/sdk/unitest` Package

**Files:**
- Create: `business/sdk/unitest/model.go`
- Create: `business/sdk/unitest/unittest.go`

- [ ] **Step 1: Create model.go**

```go
// business/sdk/unitest/model.go
package unitest

import "context"

// Table represents one test case in a table-driven test.
type Table struct {
	Name    string
	ExpResp any
	ExcFunc func(ctx context.Context) any
	CmpFunc func(got any, exp any) string
}
```

- [ ] **Step 2: Create unittest.go**

```go
// business/sdk/unitest/unittest.go
package unitest

import (
	"context"
	"testing"
)

// Run executes each Table entry as a sub-test under testName.
func Run(t *testing.T, table []Table, testName string) {
	t.Helper()

	for _, tt := range table {
		tt := tt
		t.Run(testName+"-"+tt.Name, func(t *testing.T) {
			gotResp := tt.ExcFunc(context.Background())

			diff := tt.CmpFunc(gotResp, tt.ExpResp)
			if diff != "" {
				t.Log("DIFF")
				t.Logf("%s", diff)
				t.Log("GOT")
				t.Logf("%#v", gotResp)
				t.Log("EXP")
				t.Logf("%#v", tt.ExpResp)
				t.Fatalf("Should get the expected response")
			}
		})
	}
}
```

- [ ] **Step 3: Verify and commit**

```bash
go build ./business/sdk/unitest/...
git add business/sdk/unitest/
git commit -m "test: add unitest SDK package (table-driven test runner)"
```

---

## Task 4: Create `app/sdk/apitest` Package

**Files:**
- Create: `app/sdk/apitest/model.go`
- Create: `app/sdk/apitest/apitest.go`

- [ ] **Step 1: Create model.go**

```go
// app/sdk/apitest/model.go
package apitest

// Table represents one HTTP test case.
type Table struct {
	Name       string
	URL        string
	Method     string
	APIKey     string // set to TestAPIKey for authenticated requests, "" for 401 tests
	StatusCode int
	Input      any    // marshalled to JSON body; nil for GET/DELETE
	GotResp    any    // pointer to decode response into; nil if not checking body
	ExpResp    any    // expected response value
	CmpFunc    func(got any, exp any) string
}

// TestAPIKey is the API key used by all authenticated test requests.
const TestAPIKey = "test-api-key-123"
```

- [ ] **Step 2: Create apitest.go**

```go
// app/sdk/apitest/apitest.go
package apitest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/casebrophy/planner/app/domain/clarificationapp"
	"github.com/casebrophy/planner/app/domain/contextapp"
	"github.com/casebrophy/planner/app/domain/emailapp"
	"github.com/casebrophy/planner/app/domain/observationapp"
	"github.com/casebrophy/planner/app/domain/rawinputapp"
	"github.com/casebrophy/planner/app/domain/tagapp"
	"github.com/casebrophy/planner/app/domain/taskapp"
	"github.com/casebrophy/planner/app/domain/threadapp"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/sdk/dbtest"
)

// Test holds state for HTTP integration tests.
type Test struct {
	DB     *dbtest.Database
	APIKey string
	mux    http.Handler
}

// New creates a fresh Docker Postgres database, wires all HTTP routes,
// and returns a Test ready to handle requests via httptest.
func New(t *testing.T, testName string) *Test {
	t.Helper()

	db := dbtest.New(t, testName)

	cfg := mux.Config{
		Log:    db.Log,
		DB:     db.DB,
		APIKey: TestAPIKey,
		// AnthropicAPIKey and AnthropicModel left empty; reprocess endpoint
		// is not tested via HTTP (covered at ingestbus level).
	}

	handler := mux.WebAPI(cfg,
		taskapp.Routes{},
		contextapp.Routes{},
		tagapp.Routes{},
		clarificationapp.Routes{},
		emailapp.Routes{},
		rawinputapp.Routes{},
		threadapp.Routes{},
		observationapp.Routes{},
	)

	return &Test{
		DB:     db,
		APIKey: TestAPIKey,
		mux:    handler,
	}
}

// Run executes each Table entry as an HTTP sub-test under testName.
func (at *Test) Run(t *testing.T, table []Table, testName string) {
	t.Helper()

	for _, tt := range table {
		tt := tt
		t.Run(testName+"-"+tt.Name, func(t *testing.T) {
			var r *http.Request
			if tt.Input != nil {
				d, err := json.Marshal(tt.Input)
				if err != nil {
					t.Fatalf("Marshal input: %v", err)
				}
				r = httptest.NewRequest(tt.Method, tt.URL, bytes.NewBuffer(d))
				r.Header.Set("Content-Type", "application/json")
			} else {
				r = httptest.NewRequest(tt.Method, tt.URL, nil)
			}

			if tt.APIKey != "" {
				r.Header.Set("X-API-Key", tt.APIKey)
			}

			w := httptest.NewRecorder()
			at.mux.ServeHTTP(w, r)

			if w.Code != tt.StatusCode {
				t.Fatalf("%s: expected status %d, got %d\nbody: %s",
					tt.Name, tt.StatusCode, w.Code, w.Body.String())
			}

			if tt.GotResp == nil || tt.StatusCode == http.StatusNoContent {
				return
			}

			if err := json.Unmarshal(w.Body.Bytes(), tt.GotResp); err != nil {
				t.Fatalf("Unmarshal response: %v\nbody: %s", err, w.Body.String())
			}

			diff := tt.CmpFunc(tt.GotResp, tt.ExpResp)
			if diff != "" {
				t.Log("DIFF")
				t.Logf("%s", diff)
				t.Log("GOT")
				t.Logf("%#v", tt.GotResp)
				t.Log("EXP")
				t.Logf("%#v", tt.ExpResp)
				t.Fatalf("Response mismatch")
			}
		})
	}
}
```

- [ ] **Step 3: Verify and commit**

```bash
go build ./app/sdk/apitest/...
git add app/sdk/apitest/
git commit -m "test: add apitest SDK package (HTTP test runner with API key auth)"
```

---

## Task 5: Task Domain — Business Layer Tests

**Files:**
- Create: `business/domain/taskbus/testutil.go`
- Modify: `business/domain/taskbus/taskbus_test.go` (replace empty stub)

- [ ] **Step 1: Create testutil.go**

```go
// business/domain/taskbus/testutil.go
package taskbus

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/casebrophy/planner/business/types/taskenergy"
	"github.com/casebrophy/planner/business/types/taskpriority"
	"github.com/casebrophy/planner/business/types/taskstatus"
)

// TestGenerateNewTasks returns n unsaved NewTask structs with unique titles.
func TestGenerateNewTasks(n int) []NewTask {
	tasks := make([]NewTask, n)
	idx := rand.Intn(10000)
	for i := range tasks {
		idx++
		tasks[i] = NewTask{
			Title:       fmt.Sprintf("Task%d", idx),
			Description: fmt.Sprintf("Description for task %d", idx),
			Status:      taskstatus.Todo,
			Priority:    taskpriority.Medium,
			Energy:      taskenergy.Medium,
		}
	}
	return tasks
}

// TestSeedTasks creates n tasks via the Business layer and returns them.
func TestSeedTasks(ctx context.Context, n int, api *Business) ([]Task, error) {
	newTasks := TestGenerateNewTasks(n)
	tasks := make([]Task, len(newTasks))
	for i, nt := range newTasks {
		task, err := api.Create(ctx, nt)
		if err != nil {
			return nil, fmt.Errorf("seeding task idx %d: %w", i, err)
		}
		tasks[i] = task
	}
	return tasks, nil
}
```

- [ ] **Step 2: Write taskbus_test.go** (replaces the empty stub file)

```go
// business/domain/taskbus/taskbus_test.go
package taskbus_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/unitest"
	"github.com/casebrophy/planner/business/types/taskpriority"
	"github.com/casebrophy/planner/business/types/taskstatus"
)

type seedData struct {
	tasks []taskbus.Task
}

func insertSeedData(bd dbtest.BusDomain) (seedData, error) {
	ctx := context.Background()
	tasks, err := taskbus.TestSeedTasks(ctx, 3, bd.Task)
	if err != nil {
		return seedData{}, err
	}
	return seedData{tasks: tasks}, nil
}

func Test_Task(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Task")

	sd, err := insertSeedData(db.BusDomain)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	unitest.Run(t, query(db.BusDomain, sd), "query")
	unitest.Run(t, create(db.BusDomain, sd), "create")
	unitest.Run(t, update(db.BusDomain, sd), "update")
	unitest.Run(t, delete(db.BusDomain, sd), "delete")
}

var cmpOpts = []cmp.Option{cmpopts.EquateApproxTime(time.Second)}

func query(bd dbtest.BusDomain, sd seedData) []unitest.Table {
	pg := page.MustParse("1", "10")

	return []unitest.Table{
		{
			Name:    "all",
			ExpResp: sd.tasks,
			ExcFunc: func(ctx context.Context) any {
				got, err := bd.Task.Query(ctx, taskbus.QueryFilter{}, taskbus.DefaultOrderBy, pg)
				if err != nil {
					return err
				}
				return got
			},
			CmpFunc: func(got any, exp any) string {
				gotTasks, ok := got.([]taskbus.Task)
				if !ok {
					return "got is not []taskbus.Task"
				}
				// Sort both by ID for stable comparison.
				expTasks := exp.([]taskbus.Task)
				return cmp.Diff(expTasks, gotTasks,
					cmpopts.SortSlices(func(a, b taskbus.Task) bool { return a.ID.String() < b.ID.String() }),
					cmp.Options(cmpOpts))
			},
		},
		{
			Name:    "by-status",
			ExpResp: sd.tasks,
			ExcFunc: func(ctx context.Context) any {
				status := taskstatus.Todo
				got, err := bd.Task.Query(ctx, taskbus.QueryFilter{Status: &status}, taskbus.DefaultOrderBy, pg)
				if err != nil {
					return err
				}
				return got
			},
			CmpFunc: func(got any, exp any) string {
				gotTasks, ok := got.([]taskbus.Task)
				if !ok {
					return "got is not []taskbus.Task"
				}
				expTasks := exp.([]taskbus.Task)
				return cmp.Diff(expTasks, gotTasks,
					cmpopts.SortSlices(func(a, b taskbus.Task) bool { return a.ID.String() < b.ID.String() }),
					cmp.Options(cmpOpts))
			},
		},
	}
}

func create(bd dbtest.BusDomain, sd seedData) []unitest.Table {
	return []unitest.Table{
		{
			Name: "basic",
			ExpResp: taskbus.Task{
				Title:       "NewTask-create",
				Description: "Created in test",
				Status:      taskstatus.Todo,
				Priority:    taskpriority.High,
				Energy:      taskbus.TestGenerateNewTasks(1)[0].Energy,
			},
			ExcFunc: func(ctx context.Context) any {
				nt := taskbus.NewTask{
					Title:       "NewTask-create",
					Description: "Created in test",
					Status:      taskstatus.Todo,
					Priority:    taskpriority.High,
					Energy:      taskbus.TestGenerateNewTasks(1)[0].Energy,
				}
				task, err := bd.Task.Create(ctx, nt)
				if err != nil {
					return err
				}
				return task
			},
			CmpFunc: func(got any, exp any) string {
				gotTask, ok := got.(taskbus.Task)
				if !ok {
					return "got is not taskbus.Task"
				}
				expTask := exp.(taskbus.Task)
				expTask.ID = gotTask.ID
				expTask.Energy = gotTask.Energy
				expTask.DebriefStatus = gotTask.DebriefStatus
				expTask.CreatedAt = gotTask.CreatedAt
				expTask.UpdatedAt = gotTask.UpdatedAt
				return cmp.Diff(expTask, gotTask, cmp.Options(cmpOpts))
			},
		},
	}
}

func update(bd dbtest.BusDomain, sd seedData) []unitest.Table {
	newTitle := "Updated Title"
	newPriority := taskpriority.High

	return []unitest.Table{
		{
			Name:    "basic",
			ExpResp: newTitle,
			ExcFunc: func(ctx context.Context) any {
				task := sd.tasks[0]
				updated, err := bd.Task.Update(ctx, task, taskbus.UpdateTask{
					Title:    &newTitle,
					Priority: &newPriority,
				})
				if err != nil {
					return err
				}
				return updated.Title
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(exp, got)
			},
		},
	}
}

func delete(bd dbtest.BusDomain, sd seedData) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "basic",
			ExpResp: nil,
			ExcFunc: func(ctx context.Context) any {
				task := sd.tasks[len(sd.tasks)-1]
				if err := bd.Task.Delete(ctx, task); err != nil {
					return err
				}
				return nil
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(exp, got)
			},
		},
	}
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./business/domain/taskbus/... -v -count=1 -timeout 120s
```
Expected: `PASS` for `Test_Task/query-all`, `query-by-status`, `create-basic`, `update-basic`, `delete-basic`.

- [ ] **Step 4: Commit**

```bash
git add business/domain/taskbus/testutil.go business/domain/taskbus/taskbus_test.go
git commit -m "test(taskbus): add business layer tests with dbtest + unitest"
```

---

## Task 6: Task Domain — HTTP Tests

**Files:**
- Create: `app/domain/taskapp/tests/taskapi/task_test.go`
- Create: `app/domain/taskapp/tests/taskapi/seed_test.go`
- Create: `app/domain/taskapp/tests/taskapi/query_test.go`
- Create: `app/domain/taskapp/tests/taskapi/create_test.go`
- Create: `app/domain/taskapp/tests/taskapi/update_test.go`
- Create: `app/domain/taskapp/tests/taskapi/delete_test.go`

- [ ] **Step 1: Create task_test.go**

```go
// app/domain/taskapp/tests/taskapi/task_test.go
package taskapi_test

import (
	"testing"

	"github.com/casebrophy/planner/app/sdk/apitest"
)

func Test_Task(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "Test_Task")

	sd, err := insertSeedData(test.DB)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	test.Run(t, query200(sd), "query-200")
	test.Run(t, queryByID200(sd), "querybyid-200")
	test.Run(t, create200(sd), "create-200")
	test.Run(t, create400(sd), "create-400")
	test.Run(t, create401(sd), "create-401")
	test.Run(t, update200(sd), "update-200")
	test.Run(t, update401(sd), "update-401")
	test.Run(t, delete200(sd), "delete-200")
	test.Run(t, delete401(sd), "delete-401")
}
```

- [ ] **Step 2: Create seed_test.go**

```go
// app/domain/taskapp/tests/taskapi/seed_test.go
package taskapi_test

import (
	"context"

	"github.com/casebrophy/planner/app/domain/taskapp"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
)

type seedData struct {
	tasks []taskbus.Task
}

func insertSeedData(db *dbtest.Database) (seedData, error) {
	ctx := context.Background()
	tasks, err := taskbus.TestSeedTasks(ctx, 2, db.BusDomain.Task)
	if err != nil {
		return seedData{}, err
	}
	return seedData{tasks: tasks}, nil
}

// toAppTask converts a business task to the app layer Task DTO for comparison.
// Directly reimplements the converter so tests don't depend on unexported funcs.
func toAppTaskID(t taskbus.Task) string {
	return t.ID.String()
}

// appNewTask builds a valid NewTask request body.
func appNewTask(title string) taskapp.NewTask {
	return taskapp.NewTask{
		Title:       title,
		Description: "Test task",
		Priority:    "medium",
		Energy:      "medium",
	}
}
```

- [ ] **Step 3: Create query_test.go**

```go
// app/domain/taskapp/tests/taskapi/query_test.go
package taskapi_test

import (
	"net/http"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/casebrophy/planner/app/domain/taskapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
)

func query200(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "all",
			URL:        "/api/v1/tasks",
			Method:     http.MethodGet,
			APIKey:     apitest.TestAPIKey,
			StatusCode: http.StatusOK,
			GotResp:    &[]taskapp.Task{},
			ExpResp:    &[]taskapp.Task{},
			CmpFunc: func(got any, exp any) string {
				gotTasks := got.(*[]taskapp.Task)
				if len(*gotTasks) < 2 {
					return "expected at least 2 tasks in response"
				}
				return ""
			},
		},
	}
}

func queryByID200(sd seedData) []apitest.Table {
	task := sd.tasks[0]
	return []apitest.Table{
		{
			Name:       "basic",
			URL:        "/api/v1/tasks/" + task.ID.String(),
			Method:     http.MethodGet,
			APIKey:     apitest.TestAPIKey,
			StatusCode: http.StatusOK,
			GotResp:    &taskapp.Task{},
			ExpResp:    &taskapp.Task{ID: task.ID.String(), Title: task.Title},
			CmpFunc: func(got any, exp any) string {
				gotTask := got.(*taskapp.Task)
				expTask := exp.(*taskapp.Task)
				return cmp.Diff(expTask.ID, gotTask.ID,
					cmpopts.EquateEmpty())
			},
		},
	}
}
```

- [ ] **Step 4: Create create_test.go**

```go
// app/domain/taskapp/tests/taskapi/create_test.go
package taskapi_test

import (
	"net/http"

	"github.com/google/go-cmp/cmp"

	"github.com/casebrophy/planner/app/domain/taskapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
)

func create200(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "basic",
			URL:        "/api/v1/tasks",
			Method:     http.MethodPost,
			APIKey:     apitest.TestAPIKey,
			StatusCode: http.StatusOK,
			Input:      appNewTask("HTTP Create Test"),
			GotResp:    &taskapp.Task{},
			ExpResp:    &taskapp.Task{Title: "HTTP Create Test", Status: "todo", Priority: "medium", Energy: "medium"},
			CmpFunc: func(got any, exp any) string {
				gotTask := got.(*taskapp.Task)
				expTask := exp.(*taskapp.Task)
				expTask.ID = gotTask.ID
				expTask.DebriefStatus = gotTask.DebriefStatus
				expTask.CreatedAt = gotTask.CreatedAt
				expTask.UpdatedAt = gotTask.UpdatedAt
				return cmp.Diff(expTask, gotTask)
			},
		},
	}
}

func create400(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "missing-title",
			URL:        "/api/v1/tasks",
			Method:     http.MethodPost,
			APIKey:     apitest.TestAPIKey,
			StatusCode: http.StatusBadRequest,
			Input:      taskapp.NewTask{Priority: "invalid-priority"},
			GotResp:    nil,
		},
	}
}

func create401(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "no-key",
			URL:        "/api/v1/tasks",
			Method:     http.MethodPost,
			APIKey:     "", // no X-API-Key header
			StatusCode: http.StatusUnauthorized,
			Input:      appNewTask("Should Fail"),
			GotResp:    nil,
		},
	}
}
```

- [ ] **Step 5: Create update_test.go**

```go
// app/domain/taskapp/tests/taskapi/update_test.go
package taskapi_test

import (
	"net/http"

	"github.com/casebrophy/planner/app/domain/taskapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
)

func update200(sd seedData) []apitest.Table {
	task := sd.tasks[0]
	newTitle := "Updated via HTTP"
	return []apitest.Table{
		{
			Name:       "title",
			URL:        "/api/v1/tasks/" + task.ID.String(),
			Method:     http.MethodPut,
			APIKey:     apitest.TestAPIKey,
			StatusCode: http.StatusOK,
			Input:      taskapp.UpdateTask{Title: &newTitle},
			GotResp:    &taskapp.Task{},
			ExpResp:    nil,
			CmpFunc: func(got any, exp any) string {
				gotTask := got.(*taskapp.Task)
				if gotTask.Title != newTitle {
					return "expected title to be updated to " + newTitle
				}
				return ""
			},
		},
	}
}

func update401(sd seedData) []apitest.Table {
	task := sd.tasks[0]
	newTitle := "Should Fail"
	return []apitest.Table{
		{
			Name:       "no-key",
			URL:        "/api/v1/tasks/" + task.ID.String(),
			Method:     http.MethodPut,
			APIKey:     "",
			StatusCode: http.StatusUnauthorized,
			Input:      taskapp.UpdateTask{Title: &newTitle},
			GotResp:    nil,
		},
	}
}
```

- [ ] **Step 6: Create delete_test.go**

```go
// app/domain/taskapp/tests/taskapi/delete_test.go
package taskapi_test

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/apitest"
)

func delete200(sd seedData) []apitest.Table {
	task := sd.tasks[1]
	return []apitest.Table{
		{
			Name:       "basic",
			URL:        "/api/v1/tasks/" + task.ID.String(),
			Method:     http.MethodDelete,
			APIKey:     apitest.TestAPIKey,
			StatusCode: http.StatusNoContent,
			GotResp:    nil,
		},
	}
}

func delete401(sd seedData) []apitest.Table {
	task := sd.tasks[0]
	return []apitest.Table{
		{
			Name:       "no-key",
			URL:        "/api/v1/tasks/" + task.ID.String(),
			Method:     http.MethodDelete,
			APIKey:     "",
			StatusCode: http.StatusUnauthorized,
			GotResp:    nil,
		},
	}
}
```

- [ ] **Step 7: Run HTTP tests**

```bash
go test ./app/domain/taskapp/tests/... -v -count=1 -timeout 120s
```
Expected: all sub-tests PASS.

- [ ] **Step 8: Commit**

```bash
git add app/domain/taskapp/tests/
git commit -m "test(taskapp): add HTTP layer tests for task domain"
```

---

## Task 7: Context Domain — Business + HTTP Tests

**Files:**
- Create: `business/domain/contextbus/testutil.go`
- Create: `business/domain/contextbus/contextbus_test.go`
- Create: `app/domain/contextapp/tests/contextapi/` (5 files, same structure as Task 6)

- [ ] **Step 1: Create contextbus/testutil.go**

```go
// business/domain/contextbus/testutil.go
package contextbus

import (
	"context"
	"fmt"
	"math/rand"
)

// TestGenerateNewContexts returns n unsaved NewContext structs.
func TestGenerateNewContexts(n int) []NewContext {
	contexts := make([]NewContext, n)
	idx := rand.Intn(10000)
	for i := range contexts {
		idx++
		contexts[i] = NewContext{
			Title:       fmt.Sprintf("Context%d", idx),
			Description: fmt.Sprintf("Description for context %d", idx),
		}
	}
	return contexts
}

// TestSeedContexts creates n contexts via the Business layer and returns them.
func TestSeedContexts(ctx context.Context, n int, api *Business) ([]Context, error) {
	newContexts := TestGenerateNewContexts(n)
	contexts := make([]Context, len(newContexts))
	for i, nc := range newContexts {
		c, err := api.Create(ctx, nc)
		if err != nil {
			return nil, fmt.Errorf("seeding context idx %d: %w", i, err)
		}
		contexts[i] = c
	}
	return contexts, nil
}
```

- [ ] **Step 2: Create contextbus/contextbus_test.go**

```go
// business/domain/contextbus/contextbus_test.go
package contextbus_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/unitest"
)

type seedData struct {
	contexts []contextbus.Context
}

func insertSeedData(bd dbtest.BusDomain) (seedData, error) {
	ctx := context.Background()
	contexts, err := contextbus.TestSeedContexts(ctx, 3, bd.Context)
	if err != nil {
		return seedData{}, err
	}
	return seedData{contexts: contexts}, nil
}

func Test_Context(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Context")

	sd, err := insertSeedData(db.BusDomain)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	unitest.Run(t, query(db.BusDomain, sd), "query")
	unitest.Run(t, create(db.BusDomain, sd), "create")
	unitest.Run(t, update(db.BusDomain, sd), "update")
	unitest.Run(t, delete(db.BusDomain, sd), "delete")
}

var cmpOpts = []cmp.Option{cmpopts.EquateApproxTime(time.Second)}

func query(bd dbtest.BusDomain, sd seedData) []unitest.Table {
	pg := page.MustParse("1", "10")

	return []unitest.Table{
		{
			Name:    "all",
			ExpResp: sd.contexts,
			ExcFunc: func(ctx context.Context) any {
				got, err := bd.Context.Query(ctx, contextbus.QueryFilter{}, contextbus.DefaultOrderBy, pg)
				if err != nil {
					return err
				}
				return got
			},
			CmpFunc: func(got any, exp any) string {
				gotContexts, ok := got.([]contextbus.Context)
				if !ok {
					return "got is not []contextbus.Context"
				}
				expContexts := exp.([]contextbus.Context)
				return cmp.Diff(expContexts, gotContexts,
					cmpopts.SortSlices(func(a, b contextbus.Context) bool { return a.ID.String() < b.ID.String() }),
					cmp.Options(cmpOpts))
			},
		},
	}
}

func create(bd dbtest.BusDomain, sd seedData) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "basic",
			ExpResp: "Context-created",
			ExcFunc: func(ctx context.Context) any {
				c, err := bd.Context.Create(ctx, contextbus.NewContext{
					Title:       "Context-created",
					Description: "Test context",
				})
				if err != nil {
					return err
				}
				return c.Title
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(exp, got)
			},
		},
	}
}

func update(bd dbtest.BusDomain, sd seedData) []unitest.Table {
	newTitle := "Updated Context Title"
	return []unitest.Table{
		{
			Name:    "title",
			ExpResp: newTitle,
			ExcFunc: func(ctx context.Context) any {
				c := sd.contexts[0]
				updated, err := bd.Context.Update(ctx, c, contextbus.UpdateContext{Title: &newTitle})
				if err != nil {
					return err
				}
				return updated.Title
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(exp, got)
			},
		},
	}
}

func delete(bd dbtest.BusDomain, sd seedData) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "basic",
			ExpResp: nil,
			ExcFunc: func(ctx context.Context) any {
				c := sd.contexts[len(sd.contexts)-1]
				if err := bd.Context.Delete(ctx, c); err != nil {
					return err
				}
				return nil
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(exp, got)
			},
		},
	}
}
```

> **Note:** Check `contextbus.go` for the `Delete`, `Update`, and `Query` method signatures — `contextbus.Context` does not have a `Status` field as a typed enum package like taskbus; it uses `contextbus.Status` defined inline.

- [ ] **Step 3: Create context HTTP tests**

Create `app/domain/contextapp/tests/contextapi/context_test.go`:

```go
package contextapi_test

import (
	"testing"

	"github.com/casebrophy/planner/app/sdk/apitest"
)

func Test_Context(t *testing.T) {
	t.Parallel()

	test := apitest.New(t, "Test_Context")

	sd, err := insertSeedData(test.DB)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	test.Run(t, query200(sd), "query-200")
	test.Run(t, queryByID200(sd), "querybyid-200")
	test.Run(t, create200(sd), "create-200")
	test.Run(t, create401(sd), "create-401")
	test.Run(t, update200(sd), "update-200")
	test.Run(t, update401(sd), "update-401")
	test.Run(t, delete200(sd), "delete-200")
	test.Run(t, delete401(sd), "delete-401")
}
```

Create `app/domain/contextapp/tests/contextapi/seed_test.go`:

```go
package contextapi_test

import (
	"context"

	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
)

type seedData struct {
	contexts []contextbus.Context
}

func insertSeedData(db *dbtest.Database) (seedData, error) {
	ctx := context.Background()
	contexts, err := contextbus.TestSeedContexts(ctx, 2, db.BusDomain.Context)
	if err != nil {
		return seedData{}, err
	}
	return seedData{contexts: contexts}, nil
}
```

Create `app/domain/contextapp/tests/contextapi/create_test.go`:

```go
package contextapi_test

import (
	"net/http"

	"github.com/casebrophy/planner/app/domain/contextapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
)

func create200(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "basic",
			URL:        "/api/v1/contexts",
			Method:     http.MethodPost,
			APIKey:     apitest.TestAPIKey,
			StatusCode: http.StatusOK,
			Input:      contextapp.NewContext{Title: "HTTP Context", Description: "test"},
			GotResp:    &contextapp.Context{},
			ExpResp:    &contextapp.Context{Title: "HTTP Context", Status: "active"},
			CmpFunc: func(got any, exp any) string {
				g := got.(*contextapp.Context)
				e := exp.(*contextapp.Context)
				if g.Title != e.Title || g.Status != e.Status {
					return "title or status mismatch"
				}
				return ""
			},
		},
	}
}

func create401(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "no-key",
			URL:        "/api/v1/contexts",
			Method:     http.MethodPost,
			APIKey:     "",
			StatusCode: http.StatusUnauthorized,
			Input:      contextapp.NewContext{Title: "Fail"},
			GotResp:    nil,
		},
	}
}

func query200(sd seedData) []apitest.Table {
	return []apitest.Table{
		{
			Name:       "all",
			URL:        "/api/v1/contexts",
			Method:     http.MethodGet,
			APIKey:     apitest.TestAPIKey,
			StatusCode: http.StatusOK,
			GotResp:    &[]contextapp.Context{},
			CmpFunc: func(got any, exp any) string {
				g := got.(*[]contextapp.Context)
				if len(*g) < 2 {
					return "expected at least 2 contexts"
				}
				return ""
			},
		},
	}
}

func queryByID200(sd seedData) []apitest.Table {
	c := sd.contexts[0]
	return []apitest.Table{
		{
			Name:       "basic",
			URL:        "/api/v1/contexts/" + c.ID.String(),
			Method:     http.MethodGet,
			APIKey:     apitest.TestAPIKey,
			StatusCode: http.StatusOK,
			GotResp:    &contextapp.Context{},
			CmpFunc: func(got any, exp any) string {
				g := got.(*contextapp.Context)
				if g.ID != c.ID.String() {
					return "ID mismatch"
				}
				return ""
			},
		},
	}
}

func update200(sd seedData) []apitest.Table {
	c := sd.contexts[0]
	newTitle := "Updated Context"
	return []apitest.Table{
		{
			Name:       "title",
			URL:        "/api/v1/contexts/" + c.ID.String(),
			Method:     http.MethodPut,
			APIKey:     apitest.TestAPIKey,
			StatusCode: http.StatusOK,
			Input:      contextapp.UpdateContext{Title: &newTitle},
			GotResp:    &contextapp.Context{},
			CmpFunc: func(got any, exp any) string {
				g := got.(*contextapp.Context)
				if g.Title != newTitle {
					return "title not updated"
				}
				return ""
			},
		},
	}
}

func update401(sd seedData) []apitest.Table {
	c := sd.contexts[0]
	newTitle := "Fail"
	return []apitest.Table{
		{
			Name:       "no-key",
			URL:        "/api/v1/contexts/" + c.ID.String(),
			Method:     http.MethodPut,
			APIKey:     "",
			StatusCode: http.StatusUnauthorized,
			Input:      contextapp.UpdateContext{Title: &newTitle},
			GotResp:    nil,
		},
	}
}

func delete200(sd seedData) []apitest.Table {
	c := sd.contexts[1]
	return []apitest.Table{
		{
			Name:       "basic",
			URL:        "/api/v1/contexts/" + c.ID.String(),
			Method:     http.MethodDelete,
			APIKey:     apitest.TestAPIKey,
			StatusCode: http.StatusNoContent,
			GotResp:    nil,
		},
	}
}

func delete401(sd seedData) []apitest.Table {
	c := sd.contexts[0]
	return []apitest.Table{
		{
			Name:       "no-key",
			URL:        "/api/v1/contexts/" + c.ID.String(),
			Method:     http.MethodDelete,
			APIKey:     "",
			StatusCode: http.StatusUnauthorized,
			GotResp:    nil,
		},
	}
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./business/domain/contextbus/... ./app/domain/contextapp/tests/... -v -count=1 -timeout 120s
```
Expected: all sub-tests PASS.

- [ ] **Step 5: Commit**

```bash
git add business/domain/contextbus/ app/domain/contextapp/tests/
git commit -m "test(contextbus/contextapp): add business and HTTP layer tests"
```

---

## Task 8: Tag Domain — Business + HTTP Tests

**Files:**
- Create: `business/domain/tagbus/testutil.go`
- Create: `business/domain/tagbus/tagbus_test.go`
- Create: `app/domain/tagapp/tests/tagapi/` (3 files — query, create, delete only; no update endpoint)

- [ ] **Step 1: Create tagbus/testutil.go**

```go
// business/domain/tagbus/testutil.go
package tagbus

import (
	"context"
	"fmt"
	"math/rand"
)

// TestGenerateNewTags returns n unsaved NewTag structs with unique names.
func TestGenerateNewTags(n int) []NewTag {
	tags := make([]NewTag, n)
	idx := rand.Intn(10000)
	for i := range tags {
		idx++
		tags[i] = NewTag{Name: fmt.Sprintf("tag-%d", idx)}
	}
	return tags
}

// TestSeedTags creates n tags via the Business layer and returns them.
func TestSeedTags(ctx context.Context, n int, api *Business) ([]Tag, error) {
	newTags := TestGenerateNewTags(n)
	tags := make([]Tag, len(newTags))
	for i, nt := range newTags {
		tag, err := api.Create(ctx, nt)
		if err != nil {
			return nil, fmt.Errorf("seeding tag idx %d: %w", i, err)
		}
		tags[i] = tag
	}
	return tags, nil
}
```

- [ ] **Step 2: Create tagbus_test.go**

```go
// business/domain/tagbus/tagbus_test.go
package tagbus_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/casebrophy/planner/business/domain/tagbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/unitest"
)

type seedData struct {
	tags []tagbus.Tag
}

func insertSeedData(bd dbtest.BusDomain) (seedData, error) {
	ctx := context.Background()
	tags, err := tagbus.TestSeedTags(ctx, 3, bd.Tag)
	if err != nil {
		return seedData{}, err
	}
	return seedData{tags: tags}, nil
}

func Test_Tag(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Tag")

	sd, err := insertSeedData(db.BusDomain)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	unitest.Run(t, query(db.BusDomain, sd), "query")
	unitest.Run(t, create(db.BusDomain, sd), "create")
	unitest.Run(t, delete(db.BusDomain, sd), "delete")
}

func query(bd dbtest.BusDomain, sd seedData) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "all",
			ExpResp: len(sd.tags),
			ExcFunc: func(ctx context.Context) any {
				// tagbus.Query signature may vary — check tagbus.go.
				// This example assumes Query returns ([]Tag, error).
				tags, err := bd.Tag.Query(ctx, tagbus.QueryFilter{})
				if err != nil {
					return err
				}
				return len(tags)
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(exp, got, cmpopts.EquateEmpty())
			},
		},
	}
}

func create(bd dbtest.BusDomain, sd seedData) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "basic",
			ExpResp: "tag-created",
			ExcFunc: func(ctx context.Context) any {
				tag, err := bd.Tag.Create(ctx, tagbus.NewTag{Name: "tag-created"})
				if err != nil {
					return err
				}
				return tag.Name
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(exp, got)
			},
		},
	}
}

func delete(bd dbtest.BusDomain, sd seedData) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "basic",
			ExpResp: nil,
			ExcFunc: func(ctx context.Context) any {
				tag := sd.tags[len(sd.tags)-1]
				if err := bd.Tag.Delete(ctx, tag); err != nil {
					return err
				}
				return nil
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(exp, got)
			},
		},
	}
}
```

> **Note:** Check `tagbus.go` for exact `Query` signature — tagbus has no pagination if `Query(ctx, QueryFilter{})` is the signature. Adjust accordingly.

- [ ] **Step 3: Create tag HTTP tests**

Create `app/domain/tagapp/tests/tagapi/tag_test.go`:

```go
package tagapi_test

import (
	"testing"
	"github.com/casebrophy/planner/app/sdk/apitest"
)

func Test_Tag(t *testing.T) {
	t.Parallel()
	test := apitest.New(t, "Test_Tag")
	sd, err := insertSeedData(test.DB)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}
	test.Run(t, query200(sd), "query-200")
	test.Run(t, create200(sd), "create-200")
	test.Run(t, create401(sd), "create-401")
	test.Run(t, delete200(sd), "delete-200")
	test.Run(t, delete401(sd), "delete-401")
}
```

Create `app/domain/tagapp/tests/tagapi/seed_test.go`:

```go
package tagapi_test

import (
	"context"
	"github.com/casebrophy/planner/business/domain/tagbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
)

type seedData struct{ tags []tagbus.Tag }

func insertSeedData(db *dbtest.Database) (seedData, error) {
	ctx := context.Background()
	tags, err := tagbus.TestSeedTags(ctx, 2, db.BusDomain.Tag)
	if err != nil {
		return seedData{}, err
	}
	return seedData{tags: tags}, nil
}
```

Create `app/domain/tagapp/tests/tagapi/crud_test.go`:

```go
package tagapi_test

import (
	"net/http"
	"github.com/casebrophy/planner/app/domain/tagapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
)

func query200(sd seedData) []apitest.Table {
	return []apitest.Table{{
		Name: "all", URL: "/api/v1/tags", Method: http.MethodGet,
		APIKey: apitest.TestAPIKey, StatusCode: http.StatusOK,
		GotResp: &[]tagapp.Tag{},
		CmpFunc: func(got, exp any) string {
			g := got.(*[]tagapp.Tag)
			if len(*g) < 2 { return "expected ≥2 tags" }
			return ""
		},
	}}
}

func create200(sd seedData) []apitest.Table {
	return []apitest.Table{{
		Name: "basic", URL: "/api/v1/tags", Method: http.MethodPost,
		APIKey: apitest.TestAPIKey, StatusCode: http.StatusOK,
		Input: tagapp.NewTag{Name: "http-tag"},
		GotResp: &tagapp.Tag{},
		ExpResp: &tagapp.Tag{Name: "http-tag"},
		CmpFunc: func(got, exp any) string {
			g, e := got.(*tagapp.Tag), exp.(*tagapp.Tag)
			if g.Name != e.Name { return "name mismatch" }
			return ""
		},
	}}
}

func create401(sd seedData) []apitest.Table {
	return []apitest.Table{{
		Name: "no-key", URL: "/api/v1/tags", Method: http.MethodPost,
		APIKey: "", StatusCode: http.StatusUnauthorized,
		Input: tagapp.NewTag{Name: "fail"},
	}}
}

func delete200(sd seedData) []apitest.Table {
	return []apitest.Table{{
		Name: "basic", URL: "/api/v1/tags/" + sd.tags[1].ID.String(),
		Method: http.MethodDelete, APIKey: apitest.TestAPIKey,
		StatusCode: http.StatusNoContent,
	}}
}

func delete401(sd seedData) []apitest.Table {
	return []apitest.Table{{
		Name: "no-key", URL: "/api/v1/tags/" + sd.tags[0].ID.String(),
		Method: http.MethodDelete, APIKey: "",
		StatusCode: http.StatusUnauthorized,
	}}
}
```

- [ ] **Step 4: Run and commit**

```bash
go test ./business/domain/tagbus/... ./app/domain/tagapp/tests/... -v -count=1 -timeout 120s
git add business/domain/tagbus/ app/domain/tagapp/tests/
git commit -m "test(tagbus/tagapp): add business and HTTP layer tests"
```

---

## Task 9: Clarification Domain — Business + HTTP Tests

**Files:**
- Create: `business/domain/clarificationbus/testutil.go`
- Create: `business/domain/clarificationbus/clarificationbus_test.go`
- Create: `app/domain/clarificationapp/tests/clarificationapi/` (replace stub + add HTTP tests)

- [ ] **Step 1: Create clarificationbus/testutil.go**

```go
// business/domain/clarificationbus/testutil.go
package clarificationbus

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/types/clarificationkind"
)

// TestGenerateNewClarifications returns n unsaved NewClarificationItem structs.
func TestGenerateNewClarifications(n int) []NewClarificationItem {
	items := make([]NewClarificationItem, n)
	idx := rand.Intn(10000)
	emptyOpts, _ := json.Marshal([]string{"yes", "no"})
	for i := range items {
		idx++
		items[i] = NewClarificationItem{
			Kind:          clarificationkind.StaleTask,
			SubjectType:   "task",
			SubjectID:     uuid.New(),
			Question:      fmt.Sprintf("Is task %d still relevant?", idx),
			AnswerOptions: json.RawMessage(emptyOpts),
			PriorityScore: 0.5,
		}
	}
	return items
}

// TestSeedClarifications creates n clarification items and returns them.
func TestSeedClarifications(ctx context.Context, n int, api *Business) ([]ClarificationItem, error) {
	newItems := TestGenerateNewClarifications(n)
	items := make([]ClarificationItem, len(newItems))
	for i, ni := range newItems {
		item, err := api.Create(ctx, ni)
		if err != nil {
			return nil, fmt.Errorf("seeding clarification idx %d: %w", i, err)
		}
		items[i] = item
	}
	return items, nil
}
```

- [ ] **Step 2: Create clarificationbus_test.go** (replaces empty stub in app layer, business test here)

```go
// business/domain/clarificationbus/clarificationbus_test.go
package clarificationbus_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/unitest"
)

type seedData struct {
	items []clarificationbus.ClarificationItem
}

func insertSeedData(bd dbtest.BusDomain) (seedData, error) {
	ctx := context.Background()
	items, err := clarificationbus.TestSeedClarifications(ctx, 3, bd.Clarification)
	if err != nil {
		return seedData{}, err
	}
	return seedData{items: items}, nil
}

func Test_Clarification(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Clarification")

	sd, err := insertSeedData(db.BusDomain)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	unitest.Run(t, query(db.BusDomain, sd), "query")
	unitest.Run(t, create(db.BusDomain, sd), "create")
}

var cmpOpts = []cmp.Option{cmpopts.EquateApproxTime(time.Second)}

func query(bd dbtest.BusDomain, sd seedData) []unitest.Table {
	pg := page.MustParse("1", "10")

	return []unitest.Table{
		{
			Name:    "all",
			ExpResp: len(sd.items),
			ExcFunc: func(ctx context.Context) any {
				n, err := bd.Clarification.Count(ctx, clarificationbus.QueryFilter{})
				if err != nil {
					return err
				}
				return n
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(exp, got)
			},
		},
		{
			Name:    "list",
			ExpResp: sd.items,
			ExcFunc: func(ctx context.Context) any {
				got, err := bd.Clarification.Query(ctx, clarificationbus.QueryFilter{}, clarificationbus.DefaultOrderBy, pg)
				if err != nil {
					return err
				}
				return got
			},
			CmpFunc: func(got any, exp any) string {
				gotItems, ok := got.([]clarificationbus.ClarificationItem)
				if !ok {
					return "got is not []ClarificationItem"
				}
				expItems := exp.([]clarificationbus.ClarificationItem)
				return cmp.Diff(expItems, gotItems,
					cmpopts.SortSlices(func(a, b clarificationbus.ClarificationItem) bool { return a.ID.String() < b.ID.String() }),
					cmp.Options(cmpOpts))
			},
		},
	}
}

func create(bd dbtest.BusDomain, sd seedData) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "basic",
			ExpResp: "task",
			ExcFunc: func(ctx context.Context) any {
				items := clarificationbus.TestGenerateNewClarifications(1)
				item, err := bd.Clarification.Create(ctx, items[0])
				if err != nil {
					return err
				}
				return item.SubjectType
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(exp, got)
			},
		},
	}
}
```

- [ ] **Step 3: Create clarification HTTP tests**

Create `app/domain/clarificationapp/tests/clarificationapi/clarification_test.go`:

```go
package clarificationapi_test

import (
	"testing"
	"github.com/casebrophy/planner/app/sdk/apitest"
)

func Test_Clarification(t *testing.T) {
	t.Parallel()
	test := apitest.New(t, "Test_Clarification")
	sd, err := insertSeedData(test.DB)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}
	test.Run(t, queryQueue200(sd), "queue-200")
	test.Run(t, queryByID200(sd), "querybyid-200")
	test.Run(t, queryQueue401(sd), "queue-401")
}
```

Create `app/domain/clarificationapp/tests/clarificationapi/seed_test.go`:

```go
package clarificationapi_test

import (
	"context"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
)

type seedData struct{ items []clarificationbus.ClarificationItem }

func insertSeedData(db *dbtest.Database) (seedData, error) {
	ctx := context.Background()
	items, err := clarificationbus.TestSeedClarifications(ctx, 2, db.BusDomain.Clarification)
	if err != nil {
		return seedData{}, err
	}
	return seedData{items: items}, nil
}
```

Create `app/domain/clarificationapp/tests/clarificationapi/query_test.go`:

```go
package clarificationapi_test

import (
	"net/http"
	"github.com/casebrophy/planner/app/sdk/apitest"
)

// Note: check clarificationapp.go for the exact response type used by queryQueue.
// It may return a []ClarificationItem app DTO — adjust GotResp accordingly.

func queryQueue200(sd seedData) []apitest.Table {
	return []apitest.Table{{
		Name: "queue", URL: "/api/v1/clarifications", Method: http.MethodGet,
		APIKey: apitest.TestAPIKey, StatusCode: http.StatusOK,
		GotResp: &[]map[string]any{},
		CmpFunc: func(got, exp any) string {
			g := got.(*[]map[string]any)
			if len(*g) < 2 { return "expected ≥2 items" }
			return ""
		},
	}}
}

func queryByID200(sd seedData) []apitest.Table {
	item := sd.items[0]
	return []apitest.Table{{
		Name: "basic", URL: "/api/v1/clarifications/" + item.ID.String(),
		Method: http.MethodGet, APIKey: apitest.TestAPIKey, StatusCode: http.StatusOK,
		GotResp: &map[string]any{},
		CmpFunc: func(got, exp any) string {
			g := got.(*map[string]any)
			if (*g)["id"] != item.ID.String() { return "ID mismatch" }
			return ""
		},
	}}
}

func queryQueue401(sd seedData) []apitest.Table {
	return []apitest.Table{{
		Name: "no-key", URL: "/api/v1/clarifications", Method: http.MethodGet,
		APIKey: "", StatusCode: http.StatusUnauthorized,
	}}
}
```

> **Note:** Check `app/domain/clarificationapp/model.go` for the exact response DTO types and replace `map[string]any` with the concrete struct. The resolve/snooze/dismiss endpoints require request bodies — add tests if time allows.

- [ ] **Step 4: Run and commit**

```bash
go test ./business/domain/clarificationbus/... ./app/domain/clarificationapp/tests/... -v -count=1 -timeout 120s
git add business/domain/clarificationbus/ app/domain/clarificationapp/tests/
git commit -m "test(clarificationbus/clarificationapp): add business and HTTP layer tests"
```

---

## Task 10: Email Domain — Business + HTTP Tests (Read-Only)

**Files:**
- Create: `business/domain/emailbus/testutil.go`
- Create: `business/domain/emailbus/emailbus_test.go`
- Create: `app/domain/emailapp/tests/emailapi/` (2 files — query only)

- [ ] **Step 1: Create emailbus/testutil.go**

```go
// business/domain/emailbus/testutil.go
package emailbus

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TestGenerateNewEmails returns n unsaved NewEmail structs.
func TestGenerateNewEmails(n int) []NewEmail {
	emails := make([]NewEmail, n)
	for i := range emails {
		rawID := uuid.New()
		emails[i] = NewEmail{
			RawInputID:  rawID,
			FromAddress: fmt.Sprintf("sender%d@example.com", i),
			ToAddress:   "inbox@planner.test",
			Subject:     fmt.Sprintf("Test Email %d", i),
			BodyText:    fmt.Sprintf("Body of email %d", i),
			ReceivedAt:  time.Now().UTC(),
		}
	}
	return emails
}

// TestSeedEmails creates n emails via the Business layer and returns them.
func TestSeedEmails(ctx context.Context, n int, api *Business) ([]Email, error) {
	newEmails := TestGenerateNewEmails(n)
	emails := make([]Email, len(newEmails))
	for i, ne := range newEmails {
		email, err := api.Create(ctx, ne)
		if err != nil {
			return nil, fmt.Errorf("seeding email idx %d: %w", i, err)
		}
		emails[i] = email
	}
	return emails, nil
}
```

> **Note:** Check `emailbus.go` for whether `Create` is exposed. If email creation goes through ingestbus only, use `ingestbus.ProcessEmail` or seed via raw SQL. If `Create` does not exist on `emailbus.Business`, seed via `rawinputbus` + `ingestbus` instead.

- [ ] **Step 2: Create emailbus_test.go**

```go
// business/domain/emailbus/emailbus_test.go
package emailbus_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/casebrophy/planner/business/domain/emailbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/unitest"
)

type seedData struct {
	emails []emailbus.Email
}

func insertSeedData(bd dbtest.BusDomain) (seedData, error) {
	ctx := context.Background()
	emails, err := emailbus.TestSeedEmails(ctx, 3, bd.Email)
	if err != nil {
		return seedData{}, err
	}
	return seedData{emails: emails}, nil
}

func Test_Email(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Email")

	sd, err := insertSeedData(db.BusDomain)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	unitest.Run(t, query(db.BusDomain, sd), "query")
}

func query(bd dbtest.BusDomain, sd seedData) []unitest.Table {
	pg := page.MustParse("1", "10")

	return []unitest.Table{
		{
			Name:    "all",
			ExpResp: len(sd.emails),
			ExcFunc: func(ctx context.Context) any {
				n, err := bd.Email.Count(ctx, emailbus.QueryFilter{})
				if err != nil {
					return err
				}
				return n
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(exp, got)
			},
		},
		{
			Name:    "by-id",
			ExpResp: sd.emails[0].ID.String(),
			ExcFunc: func(ctx context.Context) any {
				email, err := bd.Email.QueryByID(ctx, sd.emails[0].ID)
				if err != nil {
					return err
				}
				return email.ID.String()
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(exp, got)
			},
		},
		{
			Name:    "query-list",
			ExpResp: 3,
			ExcFunc: func(ctx context.Context) any {
				emails, err := bd.Email.Query(ctx, emailbus.QueryFilter{}, emailbus.DefaultOrderBy, pg)
				if err != nil {
					return err
				}
				return len(emails)
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(exp, got)
			},
		},
	}
}
```

- [ ] **Step 3: Create email HTTP tests** (query only — no create/update/delete endpoints)

Create `app/domain/emailapp/tests/emailapi/email_test.go`:

```go
package emailapi_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/business/domain/emailbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
)

type seedData struct{ emails []emailbus.Email }

func insertSeedData(db *dbtest.Database) (seedData, error) {
	ctx := context.Background()
	emails, err := emailbus.TestSeedEmails(ctx, 2, db.BusDomain.Email)
	if err != nil {
		return seedData{}, err
	}
	return seedData{emails: emails}, nil
}

func Test_Email(t *testing.T) {
	t.Parallel()
	test := apitest.New(t, "Test_Email")
	sd, err := insertSeedData(test.DB)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}
	test.Run(t, []apitest.Table{
		{
			Name: "query-all", URL: "/api/v1/emails", Method: http.MethodGet,
			APIKey: apitest.TestAPIKey, StatusCode: http.StatusOK,
			GotResp: &[]map[string]any{},
			CmpFunc: func(got, exp any) string {
				g := got.(*[]map[string]any)
				if len(*g) < 2 { return "expected ≥2 emails" }
				return ""
			},
		},
		{
			Name: "query-by-id", URL: "/api/v1/emails/" + sd.emails[0].ID.String(),
			Method: http.MethodGet, APIKey: apitest.TestAPIKey, StatusCode: http.StatusOK,
			GotResp: &map[string]any{},
			CmpFunc: func(got, exp any) string { return "" },
		},
		{
			Name: "query-401", URL: "/api/v1/emails", Method: http.MethodGet,
			APIKey: "", StatusCode: http.StatusUnauthorized,
		},
	}, "email")
}
```

- [ ] **Step 4: Run and commit**

```bash
go test ./business/domain/emailbus/... ./app/domain/emailapp/tests/... -v -count=1 -timeout 120s
git add business/domain/emailbus/ app/domain/emailapp/tests/
git commit -m "test(emailbus/emailapp): add business and HTTP layer tests"
```

---

## Task 11: RawInput Domain — Business + HTTP Tests

**Files:**
- Create: `business/domain/rawinputbus/testutil.go`
- Create: `business/domain/rawinputbus/rawinputbus_test.go`
- Create: `app/domain/rawinputapp/tests/rawinputapi/` (query only — reprocess has Anthropic dependency)

- [ ] **Step 1: Create rawinputbus/testutil.go**

```go
// business/domain/rawinputbus/testutil.go
package rawinputbus

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/casebrophy/planner/business/types/rawinputsource"
)

// TestGenerateNewRawInputs returns n unsaved NewRawInput structs.
func TestGenerateNewRawInputs(n int) []NewRawInput {
	inputs := make([]NewRawInput, n)
	idx := rand.Intn(10000)
	for i := range inputs {
		idx++
		inputs[i] = NewRawInput{
			SourceType: rawinputsource.Email,
			RawContent: fmt.Sprintf(`{"subject":"Test %d","body":"Content %d"}`, idx, idx),
		}
	}
	return inputs
}

// TestSeedRawInputs creates n raw inputs via the Business layer and returns them.
func TestSeedRawInputs(ctx context.Context, n int, api *Business) ([]RawInput, error) {
	newInputs := TestGenerateNewRawInputs(n)
	inputs := make([]RawInput, len(newInputs))
	for i, ni := range newInputs {
		input, err := api.Create(ctx, ni)
		if err != nil {
			return nil, fmt.Errorf("seeding rawinput idx %d: %w", i, err)
		}
		inputs[i] = input
	}
	return inputs, nil
}
```

- [ ] **Step 2: Create rawinputbus_test.go**

```go
// business/domain/rawinputbus/rawinputbus_test.go
package rawinputbus_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/unitest"
)

type seedData struct{ inputs []rawinputbus.RawInput }

func insertSeedData(bd dbtest.BusDomain) (seedData, error) {
	ctx := context.Background()
	inputs, err := rawinputbus.TestSeedRawInputs(ctx, 3, bd.RawInput)
	if err != nil {
		return seedData{}, err
	}
	return seedData{inputs: inputs}, nil
}

func Test_RawInput(t *testing.T) {
	t.Parallel()
	db := dbtest.New(t, "Test_RawInput")
	sd, err := insertSeedData(db.BusDomain)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}
	unitest.Run(t, query(db.BusDomain, sd), "query")
	unitest.Run(t, create(db.BusDomain, sd), "create")
}

func query(bd dbtest.BusDomain, sd seedData) []unitest.Table {
	pg := page.MustParse("1", "10")
	return []unitest.Table{
		{
			Name:    "all",
			ExpResp: len(sd.inputs),
			ExcFunc: func(ctx context.Context) any {
				n, err := bd.RawInput.Count(ctx, rawinputbus.QueryFilter{})
				if err != nil { return err }
				return n
			},
			CmpFunc: func(got, exp any) string { return cmp.Diff(exp, got) },
		},
		{
			Name:    "list",
			ExpResp: 3,
			ExcFunc: func(ctx context.Context) any {
				inputs, err := bd.RawInput.Query(ctx, rawinputbus.QueryFilter{}, rawinputbus.DefaultOrderBy, pg)
				if err != nil { return err }
				return len(inputs)
			},
			CmpFunc: func(got, exp any) string { return cmp.Diff(exp, got) },
		},
	}
}

func create(bd dbtest.BusDomain, sd seedData) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "basic",
			ExpResp: "email",
			ExcFunc: func(ctx context.Context) any {
				inputs := rawinputbus.TestGenerateNewRawInputs(1)
				input, err := bd.RawInput.Create(ctx, inputs[0])
				if err != nil { return err }
				return input.SourceType.String()
			},
			CmpFunc: func(got, exp any) string { return cmp.Diff(exp, got) },
		},
	}
}
```

- [ ] **Step 3: Create rawinput HTTP tests**

Create `app/domain/rawinputapp/tests/rawinputapi/rawinput_test.go`:

```go
package rawinputapi_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
)

type seedData struct{ inputs []rawinputbus.RawInput }

func insertSeedData(db *dbtest.Database) (seedData, error) {
	ctx := context.Background()
	inputs, err := rawinputbus.TestSeedRawInputs(ctx, 2, db.BusDomain.RawInput)
	if err != nil { return seedData{}, err }
	return seedData{inputs: inputs}, nil
}

func Test_RawInput(t *testing.T) {
	t.Parallel()
	test := apitest.New(t, "Test_RawInput")
	sd, err := insertSeedData(test.DB)
	if err != nil { t.Fatalf("Seeding error: %s", err) }

	// GET endpoints only — reprocess requires Anthropic, tested at ingestbus level.
	test.Run(t, []apitest.Table{
		{
			Name: "query-all", URL: "/api/v1/raw-inputs", Method: http.MethodGet,
			APIKey: apitest.TestAPIKey, StatusCode: http.StatusOK,
			GotResp: &[]map[string]any{},
			CmpFunc: func(got, exp any) string {
				g := got.(*[]map[string]any)
				if len(*g) < 2 { return "expected ≥2 raw inputs" }
				return ""
			},
		},
		{
			Name: "query-by-id", URL: "/api/v1/raw-inputs/" + sd.inputs[0].ID.String(),
			Method: http.MethodGet, APIKey: apitest.TestAPIKey, StatusCode: http.StatusOK,
			GotResp: &map[string]any{},
			CmpFunc: func(got, exp any) string { return "" },
		},
		{
			Name: "query-401", URL: "/api/v1/raw-inputs", Method: http.MethodGet,
			APIKey: "", StatusCode: http.StatusUnauthorized,
		},
	}, "rawinput")
}
```

- [ ] **Step 4: Run and commit**

```bash
go test ./business/domain/rawinputbus/... ./app/domain/rawinputapp/tests/... -v -count=1 -timeout 120s
git add business/domain/rawinputbus/ app/domain/rawinputapp/tests/
git commit -m "test(rawinputbus/rawinputapp): add business and HTTP layer tests"
```

---

## Task 12: Thread Domain — Business + HTTP Tests

**Files:**
- Create: `business/domain/threadbus/testutil.go`
- Create: `business/domain/threadbus/threadbus_test.go`
- Create: `app/domain/threadapp/tests/threadapi/thread_test.go`

- [ ] **Step 1: Create threadbus/testutil.go**

```go
// business/domain/threadbus/testutil.go
package threadbus

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/types/threadentrykind"
	"github.com/casebrophy/planner/business/types/threadsource"
)

// TestGenerateNewThreadEntries returns n unsaved NewThreadEntry structs for the given subject.
func TestGenerateNewThreadEntries(n int, subjectType string, subjectID uuid.UUID) []NewThreadEntry {
	entries := make([]NewThreadEntry, n)
	idx := rand.Intn(10000)
	for i := range entries {
		idx++
		entries[i] = NewThreadEntry{
			SubjectType: subjectType,
			SubjectID:   subjectID,
			Kind:        threadentrykind.Update,
			Content:     fmt.Sprintf("Thread update %d", idx),
			Source:      threadsource.User,
		}
	}
	return entries
}

// TestSeedThreadEntries creates n thread entries and returns them.
func TestSeedThreadEntries(ctx context.Context, n int, api *Business, subjectType string, subjectID uuid.UUID) ([]ThreadEntry, error) {
	newEntries := TestGenerateNewThreadEntries(n, subjectType, subjectID)
	entries := make([]ThreadEntry, len(newEntries))
	for i, ne := range newEntries {
		entry, err := api.AddEntry(ctx, ne)
		if err != nil {
			return nil, fmt.Errorf("seeding thread entry idx %d: %w", i, err)
		}
		entries[i] = entry
	}
	return entries, nil
}
```

> **Note:** Check `threadbus.go` for the method name — it may be `AddEntry`, `Create`, or `Add`. Adjust accordingly.

- [ ] **Step 2: Create threadbus_test.go**

```go
// business/domain/threadbus/threadbus_test.go
package threadbus_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/unitest"
)

type seedData struct {
	subjectID uuid.UUID
	entries   []threadbus.ThreadEntry
}

func insertSeedData(bd dbtest.BusDomain) (seedData, error) {
	ctx := context.Background()
	subjectID := uuid.New()
	entries, err := threadbus.TestSeedThreadEntries(ctx, 3, bd.Thread, "task", subjectID)
	if err != nil {
		return seedData{}, err
	}
	return seedData{subjectID: subjectID, entries: entries}, nil
}

func Test_Thread(t *testing.T) {
	t.Parallel()
	db := dbtest.New(t, "Test_Thread")
	sd, err := insertSeedData(db.BusDomain)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}
	unitest.Run(t, query(db.BusDomain, sd), "query")
	unitest.Run(t, create(db.BusDomain, sd), "create")
}

func query(bd dbtest.BusDomain, sd seedData) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "by-subject",
			ExpResp: len(sd.entries),
			ExcFunc: func(ctx context.Context) any {
				// Check threadbus.go for the QueryBySubject/Query method signature.
				entries, err := bd.Thread.QueryBySubject(ctx, "task", sd.subjectID)
				if err != nil { return err }
				return len(entries)
			},
			CmpFunc: func(got, exp any) string { return cmp.Diff(exp, got) },
		},
	}
}

func create(bd dbtest.BusDomain, sd seedData) []unitest.Table {
	subjectID := uuid.New()
	return []unitest.Table{
		{
			Name:    "basic",
			ExpResp: "update",
			ExcFunc: func(ctx context.Context) any {
				entries := threadbus.TestGenerateNewThreadEntries(1, "task", subjectID)
				entry, err := bd.Thread.AddEntry(ctx, entries[0])
				if err != nil { return err }
				return entry.Kind.String()
			},
			CmpFunc: func(got, exp any) string { return cmp.Diff(exp, got) },
		},
	}
}
```

- [ ] **Step 3: Create thread HTTP tests**

Create `app/domain/threadapp/tests/threadapi/thread_test.go`:

```go
package threadapi_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
)

type seedData struct {
	subjectID uuid.UUID
	entries   []threadbus.ThreadEntry
}

func insertSeedData(db *dbtest.Database) (seedData, error) {
	ctx := context.Background()
	subjectID := uuid.New()
	entries, err := threadbus.TestSeedThreadEntries(ctx, 2, db.BusDomain.Thread, "task", subjectID)
	if err != nil { return seedData{}, err }
	return seedData{subjectID: subjectID, entries: entries}, nil
}

func Test_Thread(t *testing.T) {
	t.Parallel()
	test := apitest.New(t, "Test_Thread")
	sd, err := insertSeedData(test.DB)
	if err != nil { t.Fatalf("Seeding error: %s", err) }

	// Check threadapp.go for the NewThreadEntry request body structure.
	test.Run(t, []apitest.Table{
		{
			Name: "query-by-subject",
			URL:  "/api/v1/threads/task/" + sd.subjectID.String(),
			Method: http.MethodGet, APIKey: apitest.TestAPIKey, StatusCode: http.StatusOK,
			GotResp: &[]map[string]any{},
			CmpFunc: func(got, exp any) string {
				g := got.(*[]map[string]any)
				if len(*g) < 2 { return "expected ≥2 thread entries" }
				return ""
			},
		},
		{
			Name: "query-401",
			URL:  "/api/v1/threads/task/" + sd.subjectID.String(),
			Method: http.MethodGet, APIKey: "", StatusCode: http.StatusUnauthorized,
		},
	}, "thread")
}
```

- [ ] **Step 4: Run and commit**

```bash
go test ./business/domain/threadbus/... ./app/domain/threadapp/tests/... -v -count=1 -timeout 120s
git add business/domain/threadbus/ app/domain/threadapp/tests/
git commit -m "test(threadbus/threadapp): add business and HTTP layer tests"
```

---

## Task 13: Observation Domain — Business + HTTP Tests

**Files:**
- Create: `business/domain/observationbus/testutil.go`
- Create: `business/domain/observationbus/observationbus_test.go`
- Create: `app/domain/observationapp/tests/observationapi/observation_test.go`

- [ ] **Step 1: Create observationbus/testutil.go**

```go
// business/domain/observationbus/testutil.go
package observationbus

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/types/observationkind"
)

// TestGenerateNewObservations returns n unsaved NewObservation structs.
func TestGenerateNewObservations(n int, subjectType string, subjectID uuid.UUID) []NewObservation {
	obs := make([]NewObservation, n)
	idx := rand.Intn(10000)
	data, _ := json.Marshal(map[string]any{"value": idx})
	for i := range obs {
		idx++
		obs[i] = NewObservation{
			SubjectType: subjectType,
			SubjectID:   subjectID,
			Kind:        observationkind.CompletionPattern,
			Data:        json.RawMessage(data),
			Source:      fmt.Sprintf("test-source-%d", idx),
			Confidence:  0.8,
			Weight:      1.0,
		}
	}
	return obs
}

// TestSeedObservations creates n observations via the Business layer.
func TestSeedObservations(ctx context.Context, n int, api *Business, subjectType string, subjectID uuid.UUID) ([]Observation, error) {
	newObs := TestGenerateNewObservations(n, subjectType, subjectID)
	obs := make([]Observation, len(newObs))
	for i, no := range newObs {
		o, err := api.Record(ctx, no)
		if err != nil {
			return nil, fmt.Errorf("seeding observation idx %d: %w", i, err)
		}
		obs[i] = o
	}
	return obs, nil
}
```

> **Note:** Check `observationbus.go` for the method name — may be `Record`, `Create`, or `Add`.

- [ ] **Step 2: Create observationbus_test.go**

```go
// business/domain/observationbus/observationbus_test.go
package observationbus_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/observationbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/unitest"
)

type seedData struct {
	subjectID uuid.UUID
	obs       []observationbus.Observation
}

func insertSeedData(bd dbtest.BusDomain) (seedData, error) {
	ctx := context.Background()
	subjectID := uuid.New()
	obs, err := observationbus.TestSeedObservations(ctx, 3, bd.Observation, "task", subjectID)
	if err != nil {
		return seedData{}, err
	}
	return seedData{subjectID: subjectID, obs: obs}, nil
}

func Test_Observation(t *testing.T) {
	t.Parallel()
	db := dbtest.New(t, "Test_Observation")
	sd, err := insertSeedData(db.BusDomain)
	if err != nil {
		t.Fatalf("Seeding error: %s", err)
	}
	unitest.Run(t, query(db.BusDomain, sd), "query")
	unitest.Run(t, record(db.BusDomain, sd), "record")
}

func query(bd dbtest.BusDomain, sd seedData) []unitest.Table {
	return []unitest.Table{
		{
			Name:    "by-subject",
			ExpResp: len(sd.obs),
			ExcFunc: func(ctx context.Context) any {
				// Check observationbus.go for QueryBySubject method signature.
				obs, err := bd.Observation.QueryBySubject(ctx, "task", sd.subjectID)
				if err != nil { return err }
				return len(obs)
			},
			CmpFunc: func(got, exp any) string { return cmp.Diff(exp, got) },
		},
	}
}

func record(bd dbtest.BusDomain, sd seedData) []unitest.Table {
	subjectID := uuid.New()
	return []unitest.Table{
		{
			Name:    "basic",
			ExpResp: float32(0.8),
			ExcFunc: func(ctx context.Context) any {
				items := observationbus.TestGenerateNewObservations(1, "task", subjectID)
				o, err := bd.Observation.Record(ctx, items[0])
				if err != nil { return err }
				return o.Confidence
			},
			CmpFunc: func(got, exp any) string { return cmp.Diff(exp, got) },
		},
	}
}
```

- [ ] **Step 3: Create observation HTTP tests**

Create `app/domain/observationapp/tests/observationapi/observation_test.go`:

```go
package observationapi_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/business/domain/observationbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
)

type seedData struct {
	subjectID uuid.UUID
	obs       []observationbus.Observation
}

func insertSeedData(db *dbtest.Database) (seedData, error) {
	ctx := context.Background()
	subjectID := uuid.New()
	obs, err := observationbus.TestSeedObservations(ctx, 2, db.BusDomain.Observation, "task", subjectID)
	if err != nil { return seedData{}, err }
	return seedData{subjectID: subjectID, obs: obs}, nil
}

func Test_Observation(t *testing.T) {
	t.Parallel()
	test := apitest.New(t, "Test_Observation")
	sd, err := insertSeedData(test.DB)
	if err != nil { t.Fatalf("Seeding error: %s", err) }

	// Check observationapp.go for the NewObservation request body structure.
	test.Run(t, []apitest.Table{
		{
			Name: "query-by-subject",
			URL:  "/api/v1/observations/task/" + sd.subjectID.String(),
			Method: http.MethodGet, APIKey: apitest.TestAPIKey, StatusCode: http.StatusOK,
			GotResp: &[]map[string]any{},
			CmpFunc: func(got, exp any) string {
				g := got.(*[]map[string]any)
				if len(*g) < 2 { return "expected ≥2 observations" }
				return ""
			},
		},
		{
			Name: "query-401",
			URL:  "/api/v1/observations/task/" + sd.subjectID.String(),
			Method: http.MethodGet, APIKey: "", StatusCode: http.StatusUnauthorized,
		},
	}, "observation")
}
```

- [ ] **Step 4: Run and commit**

```bash
go test ./business/domain/observationbus/... ./app/domain/observationapp/tests/... -v -count=1 -timeout 120s
git add business/domain/observationbus/ app/domain/observationapp/tests/
git commit -m "test(observationbus/observationapp): add business and HTTP layer tests"
```

---

## Task 14: Ingest Domain — Business Tests Only

The ingest pipeline orchestrates multiple buses. Tests validate the end-to-end flow: raw input → email record → task/clarification creation. No HTTP layer (ingestbus is not exposed via REST directly).

**Files:**
- Create: `business/domain/ingestbus/testutil.go`
- Modify: `business/domain/ingestbus/ingestbus_test.go` (replace stub)

- [ ] **Step 1: Create ingestbus/testutil.go**

```go
// business/domain/ingestbus/testutil.go
package ingestbus

import (
	"fmt"
	"math/rand"
)

// TestRawEmailContent returns a raw email JSON payload suitable for ingestion.
func TestRawEmailContent(idx int) string {
	return fmt.Sprintf(`{
		"from": "sender%d@example.com",
		"to": "inbox@planner.test",
		"subject": "Test email %d",
		"body": "Please create a task to review the Q%d report by next week."
	}`, idx, idx, idx)
}

// TestGenerateRawEmailPayloads returns n raw email content strings.
func TestGenerateRawEmailPayloads(n int) []string {
	payloads := make([]string, n)
	idx := rand.Intn(10000)
	for i := range payloads {
		idx++
		payloads[i] = TestRawEmailContent(idx)
	}
	return payloads
}
```

- [ ] **Step 2: Replace ingestbus_test.go stub**

```go
// business/domain/ingestbus/ingestbus_test.go
package ingestbus_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/casebrophy/planner/business/domain/ingestbus"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/unitest"
	"github.com/casebrophy/planner/business/types/rawinputsource"
)

func Test_Ingest(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Ingest")

	unitest.Run(t, ingest(db.BusDomain), "ingest")
}

func ingest(bd dbtest.BusDomain) []unitest.Table {
	return []unitest.Table{
		{
			Name: "creates-raw-input",
			// The mock extractor returns empty extraction (no task created),
			// but the raw input record should be created successfully.
			ExpResp: rawinputsource.Email.String(),
			ExcFunc: func(ctx context.Context) any {
				payload := ingestbus.TestGenerateRawEmailPayloads(1)[0]

				// Create a raw input first (simulating what the SMTP receiver does).
				rawInput, err := bd.RawInput.Create(ctx, rawinputbus.NewRawInput{
					SourceType: rawinputsource.Email,
					RawContent: payload,
				})
				if err != nil {
					return fmt.Errorf("creating raw input: %w", err)
				}

				// Process via ingest pipeline (MockExtractor returns empty result).
				if err := bd.Ingest.ProcessEmail(ctx, rawInput.ID); err != nil {
					return fmt.Errorf("processing email: %w", err)
				}

				return rawInput.SourceType.String()
			},
			CmpFunc: func(got, exp any) string { return cmp.Diff(exp, got) },
		},
	}
}
```

> **Note:** Check `ingestbus.go` for the method name to process an email — it may be `ProcessEmail`, `IngestEmail`, or `Process`. Adjust accordingly. Also check whether it takes a `rawInputID` or the full raw content.

- [ ] **Step 3: Add missing import in test file**

The `fmt` package is used in the ExcFunc closure — ensure it's imported:

```go
import (
    "context"
    "fmt"
    "testing"
    // ...
)
```

- [ ] **Step 4: Run and commit**

```bash
go test ./business/domain/ingestbus/... -v -count=1 -timeout 120s
git add business/domain/ingestbus/
git commit -m "test(ingestbus): add pipeline integration tests"
```

---

## Task 15: Inactivity Domain — Business Tests Only

**Files:**
- Create: `business/domain/inactivitybus/testutil.go`
- Create: `business/domain/inactivitybus/inactivitybus_test.go`

The inactivity bus queries stale tasks/contexts and creates clarifications. Tests need seeded tasks/contexts with known update times.

- [ ] **Step 1: Create inactivitybus/testutil.go**

```go
// business/domain/inactivitybus/testutil.go
package inactivitybus

// No seed helpers needed — inactivitybus reads from existing tasks/contexts.
// Tests seed data via taskbus.TestSeedTasks and contextbus.TestSeedContexts,
// then call CheckInactivity to validate clarification creation.
```

- [ ] **Step 2: Create inactivitybus_test.go**

```go
// business/domain/inactivitybus/inactivitybus_test.go
package inactivitybus_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/unitest"
)

func Test_Inactivity(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Inactivity")

	unitest.Run(t, checkStale(db.BusDomain), "check-stale")
}

func checkStale(bd dbtest.BusDomain) []unitest.Table {
	return []unitest.Table{
		{
			Name: "no-stale-tasks",
			// Freshly seeded tasks are not stale (just created).
			ExpResp: 0,
			ExcFunc: func(ctx context.Context) any {
				// Seed a fresh task (not yet stale).
				_, err := taskbus.TestSeedTasks(ctx, 1, bd.Task)
				if err != nil {
					return err
				}

				// Check inactivity — fresh tasks should not trigger clarifications.
				// Check inactivitybus.go for the method name: may be Check(), Run(), or CheckInactivity().
				stale, err := bd.Inactivity.QueryStaleTasks(ctx)
				if err != nil {
					return err
				}
				return len(stale)
			},
			CmpFunc: func(got, exp any) string { return cmp.Diff(exp, got) },
		},
	}
}
```

> **Note:** Check `inactivitybus.go` for exposed methods. The `Storer.QueryStaleTasks` method is called internally — the `Business` layer method may be `Check(ctx)` or `RunInactivityCheck(ctx)`. If the Business layer doesn't directly expose stale queries, test via `bd.Inactivity.Check(ctx)` and verify clarification count increases.

- [ ] **Step 3: Run and commit**

```bash
go test ./business/domain/inactivitybus/... -v -count=1 -timeout 120s
git add business/domain/inactivitybus/
git commit -m "test(inactivitybus): add inactivity detection tests"
```

---

## Final Verification

- [ ] **Run the full test suite**

```bash
go test ./... -count=1 -timeout 300s
```
Expected: all test functions PASS. Docker container spin-up takes ~5s per test function — 17 test functions ≈ ~3 minutes total.

- [ ] **Run linter**

```bash
make lint
```
Expected: `go vet ./...` passes with no issues.

- [ ] **Verify build**

```bash
go build ./...
```
Expected: no compilation errors.

---

## Notes for Implementers

1. **tagdb.New vs NewStore**: The tagapp `route.go` uses `tagdb.New(...)` — verify this is the correct constructor in `business/domain/tagbus/stores/tagdb/tagdb.go` before writing `business.go`.

2. **emailbus.Create**: Check whether `emailbus.Business` exposes a `Create` method. If email records are only created via the ingest pipeline, seed email data by creating raw inputs and running them through `ingestbus.ProcessEmail` with the `MockExtractor`.

3. **threadbus method names**: Check `threadbus.go` for `AddEntry` vs `Create`. The `QueryBySubject` method may be named differently.

4. **observationbus method names**: Check `observationbus.go` for `Record` vs `Create`.

5. **ingestbus ProcessEmail**: Check the exact method signature in `ingestbus.go` — may take `rawInputID uuid.UUID` or the full email content.

6. **clarificationapp response types**: Replace `map[string]any` in clarification HTTP tests with the concrete `ClarificationItem` app DTO from `app/domain/clarificationapp/model.go`.

7. **Context status "active"**: The `contextbus.Active` constant is defined in `contextbus/model.go` (not a separate types package). When seeding contexts, the initial status is `Active`.

8. **Parallel test safety**: Each `Test_*` function creates its own Docker container and random database — they are fully isolated and safe to run in parallel. The `t.Parallel()` call at the top of each test function enables this.
