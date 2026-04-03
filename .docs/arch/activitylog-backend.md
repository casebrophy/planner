# Activity Log Backend System

The activity log system provides event tracking and streak computation for any domain entity. It records activities (e.g., task completions, context updates) with timestamps and optional values, supporting historical analysis and streak metrics (current streak, longest streak, total count). The system follows the layered architecture pattern: handler (activitylogapp) → business logic (activitylogbus) → store (activitylogdb).

## Core Types

### Business Models

```go
// Log represents a single activity log entry
type Log struct {
	ID          uuid.UUID   // Unique log entry ID
	SubjectType string      // Entity type being logged (e.g., "task", "context", "check")
	SubjectID   uuid.UUID   // ID of the entity instance
	Value       *string     // Optional: associated value (e.g., score, metadata)
	LoggedAt    time.Time   // Timestamp of activity
}

// NewLog is the input model for creating a new log entry
type NewLog struct {
	SubjectType string      // Required: entity type
	SubjectID   uuid.UUID   // Required: entity ID
	Value       *string     // Optional: associated value
}

// StreakInfo represents streak statistics for an entity
type StreakInfo struct {
	Current    int        // Days in current streak (0 if streak not active)
	Longest    int        // Longest streak ever recorded (days)
	TotalCount int        // Total log entries for this entity
	LastLogged *time.Time // Timestamp of last log entry (nil if no logs)
}

// QueryFilter allows filtering logs by optional criteria
type QueryFilter struct {
	SubjectType *string     // Filter logs by entity type (exact match)
	SubjectID   *uuid.UUID  // Filter logs by entity ID (exact match)
	StartDate   *time.Time  // Filter logs from this date forward (inclusive)
	EndDate     *time.Time  // Filter logs up to this date (inclusive)
}
```

### HTTP/App Models

```go
// ActivityLog is the JSON API representation returned to clients
type ActivityLog struct {
	ID          string  `json:"id"`           // UUID as string
	SubjectType string  `json:"subjectType"` // Entity type
	SubjectID   string  `json:"subjectId"`   // Entity ID as string
	Value       *string `json:"value,omitempty"` // Optional value
	LoggedAt    string  `json:"loggedAt"`    // RFC3339 timestamp
}

// NewActivityLog is the JSON request body for creating log entries
type NewActivityLog struct {
	SubjectType string  `json:"subjectType"`  // Required
	SubjectID   string  `json:"subjectId"`    // Required: UUID as string
	Value       *string `json:"value"`        // Optional
}

// Streaks is the JSON API representation of streak statistics
type Streaks struct {
	Current    int     `json:"current"`              // Current streak (days)
	Longest    int     `json:"longest"`              // Longest streak (days)
	TotalCount int     `json:"totalCount"`           // Total logs
	LastLogged *string `json:"lastLogged,omitempty"` // RFC3339 timestamp or nil
}
```

### Database Models

```go
// logDB is the internal database representation of a log entry
type logDB struct {
	ID          uuid.UUID `db:"log_id"`
	SubjectType string    `db:"subject_type"`
	SubjectID   uuid.UUID `db:"subject_id"`
	Value       *string   `db:"value"`
	LoggedAt    time.Time `db:"logged_at"`
}

// logDateRow is used internally by QueryStreaks to fetch distinct dates
type logDateRow struct {
	LogDate time.Time `db:"log_date"`
}
```

### Order Constants

```go
const (
	OrderByLoggedAt = "logged_at"  // Order logs by timestamp (internal field)
)

var DefaultOrderBy = order.NewBy(OrderByLoggedAt, order.DESC)
```

### Storer Interface

```go
// Storer defines all database operations for activity logs
type Storer interface {
	Create(ctx context.Context, log Log) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Log, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryStreaks(ctx context.Context, subjectType string, subjectID uuid.UUID) (StreakInfo, error)
}
```

## File Map

### Models / Types

- **`business/domain/activitylogbus/model.go`** — Core domain models: `Log`, `NewLog`, `StreakInfo`, `QueryFilter`
- **`business/domain/activitylogbus/order.go`** — Ordering constants: `OrderByLoggedAt`, `DefaultOrderBy`
- **`app/domain/activitylogapp/model.go`** — HTTP API models: `ActivityLog`, `NewActivityLog`, `Streaks`; conversion functions `toAppLog()`, `toAppLogs()`, `toAppStreaks()`

### App (Handlers)

- **`app/domain/activitylogapp/activitylogapp.go`**
  - **`(*app) create()`** — POST /api/v1/activity-logs; creates a new log entry, validates subjectType and subjectId are required, parses subjectId UUID
  - **`(*app) queryAll()`** — GET /api/v1/activity-logs; lists log entries with pagination, filtering, and ordering
  - **`(*app) streaks()`** — GET /api/v1/activity-logs/streaks/{subject_type}/{subject_id}; retrieves streak statistics for a specific entity

- **`app/domain/activitylogapp/filter.go`**
  - **`parseFilter()`** — HTTP query parameter → `activitylogbus.QueryFilter`; extracts optional subject_type, subject_id (parses UUID), start_date (RFC3339), end_date (RFC3339) filters

- **`app/domain/activitylogapp/order.go`**
  - **`parseOrder()`** — HTTP query parameter → `order.By`; parses orderBy field (logged_at) with validation

- **`app/domain/activitylogapp/route.go`**
  - **`(Routes) Add()`** — Registers all activity log endpoints with router; creates Store and Business instances

### Business (Core)

- **`business/domain/activitylogbus/activitylogbus.go`**
  - **`NewBusiness()`** — Factory for Business; requires Logger and Storer
  - **`(*Business) Create()`** — Generates UUID, timestamp (current time), delegates to store; wraps errors
  - **`(*Business) Query()`** — Delegates to store with filter/order/pagination; wraps errors
  - **`(*Business) Count()`** — Returns matching log count; wraps errors
  - **`(*Business) QueryStreaks()`** — Queries streak statistics for an entity; wraps errors

### Store

- **`business/domain/activitylogbus/stores/activitylogdb/activitylogdb.go`**
  - **`NewStore()`** — Factory for Store; requires Logger and *sqlx.DB
  - **`(*Store) Create()`** — INSERT INTO activity_logs; named parameters `:log_id`, `:subject_type`, `:subject_id`, `:value`, `:logged_at`
  - **`(*Store) Query()`** — SELECT from activity_logs with WHERE 1=1 + optional filters, ORDER BY, LIMIT/OFFSET pagination
  - **`(*Store) Count()`** — SELECT COUNT(*) FROM activity_logs with optional filters
  - **`(*Store) QueryStreaks()`** — Two-phase query: (1) COUNT and MAX(logged_at) summary; (2) DISTINCT dates ordered DESC; computes current and longest streaks via `computeStreaks()` helper

- **`business/domain/activitylogbus/stores/activitylogdb/filter.go`**
  - **`applyFilter()`** — Appends SQL WHERE clauses for QueryFilter; supports subject_type (exact match), subject_id (exact match), start_date (>=), end_date (<=) date range filtering

- **`business/domain/activitylogbus/stores/activitylogdb/order.go`**
  - **`orderByClause()`** — Converts `order.By` field to SQL column name and direction; validates against allowed fields (logged_at)

- **`business/domain/activitylogbus/stores/activitylogdb/model.go`**
  - **`toDBLog()`** — Converts `activitylogbus.Log` → `logDB`
  - **`toBusLog()`** — Converts `logDB` → `activitylogbus.Log`
  - **`toBusLogs()`** — Bulk conversion `[]logDB` → `[]activitylogbus.Log`

- **`business/domain/activitylogbus/stores/activitylogdb/activitylogdb.go`** (helper)
  - **`computeStreaks()`** — Computes current and longest streaks from sorted distinct dates (DESC). Current streak = consecutive days ending today or yesterday; longest = longest run in history. Returns (current, longest) tuple.

## Impact Callouts

### ⚠ Log (`business/domain/activitylogbus/model.go`)
Changing the Log struct affects:
- `activitylogapp/model.go` — `toAppLog()` and `toAppLogs()` conversion functions must be updated
- `activitylogdb/model.go` — `toDBLog()` and `toBusLog()` conversion functions must be updated
- Database migration required if adding/removing fields
- API contract: ID and SubjectID must remain `uuid.UUID`, LoggedAt must remain `time.Time`

### ⚠ NewLog (`business/domain/activitylogbus/model.go`)
Changing the NewLog struct affects:
- `activitylogapp/model.go` — May need new conversion function if fields change
- `activitylogapp/activitylogapp.go` — `create()` handler validation logic must be updated (required field checks)
- HTTP POST request body schema changes (breaking API change)

### ⚠ StreakInfo (`business/domain/activitylogbus/model.go`)
Changing the StreakInfo struct affects:
- `activitylogapp/model.go` — `toAppStreaks()` conversion must be updated
- `activitylogapp/activitylogapp.go` — `streaks()` handler return value handling may need updates
- HTTP GET /api/v1/activity-logs/streaks response schema changes

### ⚠ QueryFilter (`business/domain/activitylogbus/model.go`)
Changing the QueryFilter struct affects:
- `activitylogapp/filter.go` — `parseFilter()` must be updated to parse new query parameter fields
- `activitylogdb/filter.go` — `applyFilter()` must generate SQL WHERE clauses for new fields
- Query capabilities and HTTP query parameter schema

### ⚠ Storer Interface (`business/domain/activitylogbus/activitylogbus.go`)
Adding/changing a Storer method affects:
- `activitylogbus/activitylogbus.go` — Business methods must call the store method
- `activitylogdb/activitylogdb.go` — Store struct must implement the method with matching signature
- `activitylogapp/activitylogapp.go` — May need new handlers if new query method is added
- Contract breaking change if existing method signatures are modified

### ⚠ activitylogdb.Store (`business/domain/activitylogbus/stores/activitylogdb/activitylogdb.go`)
Changing Store methods affects:
- All Storer interface methods must maintain same signature as declared in `business/domain/activitylogbus/activitylogbus.go`
- SQL queries must handle all filter combinations and order-by fields
- `QueryStreaks()` helper function `computeStreaks()` depends on sorted dates DESC — changing sort order breaks streak logic
- Database schema (activity_logs table) must match query structure and column names

### ⚠ computeStreaks() Algorithm (`business/domain/activitylogbus/stores/activitylogdb/activitylogdb.go`)
Streak calculation logic:
- Operates on distinct dates sorted DESC (most recent first)
- Current streak requires most recent log to be today or yesterday (gap <= 1 day)
- Consecutive days are detected via 1-day gaps; missing days break streaks
- Timezone: all dates are truncated to UTC midnight for comparison
- If algorithm changes: streak values will retroactively change for all entities

## Routes

| Method | Path | Handler | Notes |
|--------|------|---------|-------|
| POST | `/api/v1/activity-logs` | `create()` | Create log entry; JSON body: `{"subjectType":"...", "subjectId":"...", "value":"..."}` |
| GET | `/api/v1/activity-logs` | `queryAll()` | List log entries; supports `subject_type` (filter), `subject_id` (filter), `start_date` (RFC3339), `end_date` (RFC3339), `orderBy` (logged_at), `page`, `rows` |
| GET | `/api/v1/activity-logs/streaks/{subject_type}/{subject_id}` | `streaks()` | Retrieve streak statistics for an entity; returns `{"current":N, "longest":N, "totalCount":N, "lastLogged":"..."}` |

All routes require authentication via API key middleware (`mid.Auth(cfg.APIKey)`).

## Database Schema

### activity_logs table
```sql
CREATE TABLE activity_logs (
    log_id UUID NOT NULL DEFAULT gen_random_uuid(),
    subject_type TEXT NOT NULL,
    subject_id UUID NOT NULL,
    value TEXT,
    logged_at TIMESTAMP NOT NULL,
    PRIMARY KEY (log_id)
);

CREATE INDEX idx_activity_logs_subject ON activity_logs(subject_type, subject_id, logged_at DESC);
CREATE INDEX idx_activity_logs_date ON activity_logs(logged_at DESC);
```
- `log_id`: Primary key, auto-generated UUID
- `subject_type`: Entity type being logged (e.g., "task", "context", "check"); text field, no foreign key constraint
- `subject_id`: ID of entity being logged; UUID; no foreign key to allow logging for any domain
- `value`: Optional associated value (e.g., score, metadata, status)
- `logged_at`: Timestamp of activity; indexed DESC for efficient streak and recent queries
- Composite index on (subject_type, subject_id, logged_at DESC) optimizes QueryStreaks and Query with subject filters

## Cross-Domain Dependencies

### Universal Subject Logging
- Activity logs can track any domain entity via (subject_type, subject_id) pair
- No foreign keys enforced — orphaned logs OK (entity can be deleted independently)
- Domain handlers may call activity log business methods to record events
- Common subject types: "task", "context", "check" (habit tracking)

### Streak Tracking Use Cases
- Task completion streaks: track daily task completions
- Context engagement streaks: track days context was active
- Check-in streaks: track daily habit or routine completion
- QueryStreaks() algorithm: consecutive days ending today/yesterday count as active streak

### SDK Dependencies
- `business/sdk/order` — Order.By type for sorting logs
- `business/sdk/page` — Page type for pagination
- `business/sdk/sqldb` — Named SQL query execution helpers
- `foundation/logger` — Logger for Store operations
- `foundation/web` — HTTP encoding/decoding framework
