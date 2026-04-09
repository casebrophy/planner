# ActivityLog Backend System

> The activity log domain tracks user activities (task completions, context interactions, habit tracking) with flexible subjectType/subjectID binding and streak calculation. Supports filtering by subject and date range, with configurable ordering and pagination.

## Core Types

### App Layer (HTTP DTOs)

```go
type ActivityLog struct {
	ID          string  `json:"id"`
	SubjectType string  `json:"subjectType"`
	SubjectID   string  `json:"subjectId"`
	Value       *string `json:"value,omitempty"`
	LoggedAt    string  `json:"loggedAt"`
}

type NewActivityLog struct {
	SubjectType string  `json:"subjectType"`
	SubjectID   string  `json:"subjectId"`
	Value       *string `json:"value"`
}

type Streaks struct {
	Current    int     `json:"current"`
	Longest    int     `json:"longest"`
	TotalCount int     `json:"totalCount"`
	LastLogged *string `json:"lastLogged,omitempty"`
}

type BulkLogsResponse struct {
	Items map[string][]ActivityLog `json:"items"`
}
```

### Business Layer (Domain Types)

```go
type Log struct {
	ID          uuid.UUID
	SubjectType string
	SubjectID   uuid.UUID
	Value       *string
	LoggedAt    time.Time
}

type NewLog struct {
	SubjectType string
	SubjectID   uuid.UUID
	Value       *string
}

type StreakInfo struct {
	Current    int
	Longest    int
	TotalCount int
	LastLogged *time.Time
}

type QueryFilter struct {
	SubjectType *string
	SubjectID   *uuid.UUID
	StartDate   *time.Time
	EndDate     *time.Time
}

type QueryBySubjectsFilter struct {
	SubjectType string
	SubjectIDs  []uuid.UUID
	From        time.Time
	To          time.Time
}
```

### Store Layer (Database Model)

```go
type logDB struct {
	ID          uuid.UUID `db:"log_id"`
	SubjectType string    `db:"subject_type"`
	SubjectID   uuid.UUID `db:"subject_id"`
	Value       *string   `db:"value"`
	LoggedAt    time.Time `db:"logged_at"`
}
```

### Storer Interface

```go
type Storer interface {
	Create(ctx context.Context, log Log) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Log, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryStreaks(ctx context.Context, subjectType string, subjectID uuid.UUID) (StreakInfo, error)
	QueryBySubjects(ctx context.Context, filter QueryBySubjectsFilter) ([]Log, error)
}
```

### Order Constants

```go
const (
	OrderByLoggedAt = "logged_at"
)

var DefaultOrderBy = order.NewBy(OrderByLoggedAt, order.DESC)
```

## File Map

### App Layer (`app/domain/activitylogapp/`)
- `activitylogapp.go` — **create()** creates a new activity log; **queryAll()** retrieves filtered/ordered/paginated logs; **queryBulk()** returns logs for multiple subjects grouped by subject ID; **streaks()** fetches streak data for a subject
- `model.go` — **toAppLog()**, **toAppLogs()**, **toAppStreaks()**, **BulkLogsResponse** convert business types to HTTP DTOs
- `route.go` — **Routes.Add()** wires store → business → handlers, registers four endpoints with auth middleware
- `filter.go` — **parseFilter()** maps query params (subject_type, subject_id, start_date, end_date) to QueryFilter
- `order.go` — **parseOrder()** maps request orderBy field to activitylogbus.OrderByLoggedAt constant

### Business Layer (`business/domain/activitylogbus/`)
- `activitylogbus.go` — **Business.Create()** generates UUID + timestamp; **Business.Query()** passes filter/order/page to storer; **Business.Count()** counts matching records; **Business.QueryStreaks()** delegates streak calculation to storer; **Business.QueryBySubjects()** bulk-queries logs for multiple subject IDs within a date range
- `model.go` — Log, NewLog, StreakInfo, QueryBySubjectsFilter domain types
- `filter.go` — QueryFilter struct (SubjectType, SubjectID, StartDate, EndDate pointers for optional filtering)
- `order.go` — OrderByLoggedAt constant and DefaultOrderBy (DESC)

### Store Layer (`business/domain/activitylogbus/stores/activitylogdb/`)
- `activitylogdb.go` — **Create()** inserts into activity_logs; **Query()** SELECT with dynamic WHERE/ORDER/LIMIT; **Count()** COUNT(*) with filter; **QueryStreaks()** runs summary + date queries then calls **computeStreaks()** for current/longest streak; **QueryBySubjects()** bulk-queries logs for multiple subject IDs with dynamic IN clause
- `model.go` — logDB struct with db tags, **toDBLog()**, **toBusLog()**, **toBusLogs()** converters
- `filter.go` — **applyFilter()** builds WHERE clauses for SubjectType, SubjectID, StartDate, EndDate
- `order.go` — orderByFields map (logged_at → "logged_at"), **orderByClause()** builds SQL ORDER BY fragment

## Impact Callouts

### ⚠ Log / NewLog (business/domain/activitylogbus/model.go)
Changing fields requires updates across:
- `activitylogapp/model.go` — update toAppLog/toAppLogs converters
- `activitylogdb/model.go` — update logDB struct tags and toDBLog/toBusLog converters
- `activitylogdb/activitylogdb.go` — update INSERT/SELECT column lists

### ⚠ QueryFilter (business/domain/activitylogbus/filter.go)
Adding a new filter field requires:
- `activitylogapp/filter.go` — parse the query param and add to QueryFilter
- `activitylogdb/filter.go` — add WHERE clause in applyFilter()

### ⚠ Order Constants (business/domain/activitylogbus/order.go)
Adding a new sortable field requires:
- `activitylogdb/order.go` — add to orderByFields map
- `activitylogapp/order.go` — add to orderByFields map for request parsing

### ⚠ Storer Interface (business/domain/activitylogbus/activitylogbus.go)
Adding a new method requires:
- `activitylogdb/activitylogdb.go` — implement the method

### ⚠ computeStreaks() (activitylogdb/activitylogdb.go)
Streak rules (gap tolerance, today/yesterday logic) affect QueryStreaks() results. Dates must be pre-sorted DESC for algorithm correctness.

## Routes

| Method | Path | Handler |
|--------|------|---------|
| POST | /api/v1/activity-logs | create — creates new log; requires SubjectType, SubjectID, optional Value |
| GET | /api/v1/activity-logs | queryAll — lists logs with optional filters and ordering |
| GET | /api/v1/activity-logs/bulk | queryBulk — returns logs for multiple subjects grouped by subject ID; requires subject_type, subject_ids, from, to query params |
| GET | /api/v1/activity-logs/streaks/{subject_type}/{subject_id} | streaks — returns current/longest streak and total count |

All routes require `X-API-Key` header (auth middleware).

## Cross-Domain Dependencies

- **taskbus** — logs track task completion via SubjectType="task"
- **contextbus** — logs track context interactions via SubjectType="context"
- **business/sdk/page** — pagination via page.Page
- **business/sdk/order** — ordering via order.By; only OrderByLoggedAt currently supported
- **business/sdk/sqldb** — NamedExecContext, NamedQuerySlice, NamedQueryStruct helpers
- **app/sdk/query** — queryAll() returns query.NewResult for consistent pagination response shape
