# Clarification Backend System

> Clarification items represent open questions that require user input to resolve ambiguities, make decisions, or handle edge cases during task/context ingestion and management. Items can be in pending, snoozed, resolved, or dismissed status. Resolution dispatches side-effects (context assignment, deadline confirmation, task creation, entity linking, etc.) based on the clarification kind and the user's answer. Priority scoring weights item age (40%) and kind importance (60%).

## Core Types

### App Layer

```go
type ClarificationItem struct {
	ID            string          `json:"id"`
	Kind          string          `json:"kind"`
	Status        string          `json:"status"`
	SubjectType   string          `json:"subjectType"`
	SubjectID     string          `json:"subjectId"`
	Question      string          `json:"question"`
	ClaudeGuess   json.RawMessage `json:"claudeGuess,omitempty"`
	Reasoning     *string         `json:"reasoning,omitempty"`
	AnswerOptions json.RawMessage `json:"answerOptions"`
	Answer        json.RawMessage `json:"answer,omitempty"`
	PriorityScore float32         `json:"priorityScore"`
	SnoozedUntil  *string         `json:"snoozedUntil,omitempty"`
	CreatedAt     string          `json:"createdAt"`
	ResolvedAt    *string         `json:"resolvedAt,omitempty"`
}

type ResolveInput struct {
	Answer json.RawMessage `json:"answer"`
}

type SnoozeInput struct {
	Hours int `json:"hours"`
}

type CountResponse struct {
	Count int `json:"count"`
}
```

### Business Layer

```go
type ClarificationItem struct {
	ID            uuid.UUID
	Kind          clarificationkind.Kind
	Status        clarificationstatus.Status
	SubjectType   string
	SubjectID     uuid.UUID
	Question      string
	ClaudeGuess   *json.RawMessage
	Reasoning     *string
	AnswerOptions json.RawMessage
	Answer        *json.RawMessage
	PriorityScore float32
	SnoozedUntil  *time.Time
	CreatedAt     time.Time
	ResolvedAt    *time.Time
}

type NewClarificationItem struct {
	Kind          clarificationkind.Kind
	SubjectType   string
	SubjectID     uuid.UUID
	Question      string
	ClaudeGuess   *json.RawMessage
	Reasoning     *string
	AnswerOptions json.RawMessage
	PriorityScore float32
	SnoozedUntil  *time.Time
}

type ResolveClarificationItem struct {
	Answer json.RawMessage
}

type QueryFilter struct {
	Status      *clarificationstatus.Status
	Kind        *clarificationkind.Kind
	SubjectType *string
	SubjectID   *uuid.UUID
}
```

### Store Layer

```go
type clarificationDB struct {
	ID            uuid.UUID        `db:"clarification_id"`
	Kind          string           `db:"kind"`
	Status        string           `db:"status"`
	SubjectType   string           `db:"subject_type"`
	SubjectID     uuid.UUID        `db:"subject_id"`
	Question      string           `db:"question"`
	ClaudeGuess   *json.RawMessage `db:"claude_guess"`
	Reasoning     *string          `db:"reasoning"`
	AnswerOptions json.RawMessage  `db:"answer_options"`
	Answer        *json.RawMessage `db:"answer"`
	PriorityScore float32          `db:"priority_score"`
	SnoozedUntil  *time.Time       `db:"snoozed_until"`
	CreatedAt     time.Time        `db:"created_at"`
	ResolvedAt    *time.Time       `db:"resolved_at"`
}
```

### Storer Interface

```go
type Storer interface {
	Create(ctx context.Context, item ClarificationItem) error
	Update(ctx context.Context, item ClarificationItem) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]ClarificationItem, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, id uuid.UUID) (ClarificationItem, error)
	UnsnoozeExpired(ctx context.Context, now time.Time) (int, error)
}
```

## File Map

### App Layer (app/domain/clarificationapp/)
- `clarificationapp.go` — **queryQueue()** list pending clarifications with filters and pagination; **queryByID()** fetch single clarification; **resolve()** resolve with answer and dispatch side-effects; **snooze()** snooze for N hours; **dismiss()** mark as dismissed; **countPending()** count pending; **dispatchResolution()** routes answers to appropriate side-effect handlers by kind and subject type
- `model.go` — **ClarificationItem** app DTO (IDs as strings); **toAppClarification()** business → DTO; **toAppClarifications()** batch converter
- `route.go` — **Routes.Add()** registers all six endpoints and wires dependencies
- `filter.go` — **parseFilter()** parses query params (status, kind, subject_type, subject_id) → QueryFilter
- `order.go` — **parseOrder()** → order.By; supports priority_score (DESC default) and created_at

### Business Layer (business/domain/clarificationbus/)
- `clarificationbus.go` — **Create()** initial status (pending or snoozed), priority score (age_hours*0.4 + kind_weight*0.6); **Resolve()** → Resolved + ResolvedAt; **Snooze()** → Snoozed; **Dismiss()** → Dismissed + ResolvedAt; **Query/QueryByID/Count/UnsnoozeExpired** delegate to storer; **RecalculatePriority()** recalculates score
- `model.go` — ClarificationItem (typed Kind/Status), NewClarificationItem, ResolveClarificationItem
- `filter.go` — QueryFilter (Status, Kind, SubjectType, SubjectID)
- `order.go` — OrderByPriorityScore, OrderByCreatedAt; DefaultOrderBy = priority_score DESC

### Store Layer (business/domain/clarificationbus/stores/clarificationdb/)
- `clarificationdb.go` — **Create()** INSERT; **Update()** UPDATE all mutable fields; **Query()** SELECT with filter/ordering/pagination; **Count()** COUNT(*); **QueryByID()** single item; **UnsnoozeExpired()** UPDATE status=pending where snoozed_until <= now
- `model.go` — clarificationDB (string kind/status); **toDBClarification()** enums → strings; **toBusClarification()** strings → enums via MustParse
- `filter.go` — **applyFilter()** WHERE clauses for Status, Kind, SubjectType, SubjectID
- `order.go` — orderByFields map; **orderByClause()** builds ORDER BY fragment

## Impact Callouts

### ⚠ ClarificationItem (business/domain/clarificationbus/model.go)
Changing this struct requires:
- `clarificationapp/model.go` — update app DTO and toAppClarification converter
- `clarificationdb/model.go` — update clarificationDB struct and converters
- SQL migration — add/modify columns
- Business methods — Create, Resolve, Snooze, Dismiss may need to set new fields

### ⚠ Storer interface (business/domain/clarificationbus/clarificationbus.go)
Adding or changing a method affects:
- `clarificationdb/clarificationdb.go` — must implement the new method

### ⚠ QueryFilter (business/domain/clarificationbus/filter.go)
Adding a filter field requires:
- `clarificationapp/filter.go` — add parseFilter() case
- `clarificationdb/filter.go` — add applyFilter() WHERE clause

### ⚠ dispatchResolution (clarificationapp/clarificationapp.go)
Adding a new clarificationkind requires a new case branch with its JSON answer schema and calls to appropriate *Bus methods. No transactional guarantee — failures are logged but don't fail the resolve response.

### ⚠ Answer schemas per kind
- **ContextAssignment**: `{context_id: "uuid"}`
- **AmbiguousDeadline**: `{due_date: "2006-01-02" or RFC3339}`
- **AmbiguousAction**: `{is_task: bool, title: string, description: string, context_id?: "uuid"}`
- **NewContext**: `{action: "confirm"|"merge", title?: string, merge_target_id?: "uuid"}`
- **InactivityPrompt**: `{action: "completed"|other, note?: string}`
- **ContextDebrief**: `{response: string}`
- **StaleTask**: `{status: "open"|"done"|other}`
- **EntityLink**: `{confirmed: bool}`; AnswerOptions contains `{sourceId, targetId, sourceType, targetType, confidence}`
- **TaskDebrief**: `{value: "low"|"medium"|"high"|"skip"}`; records observation with importance + question when not "skip"; Weight=2.0
- **WeeklyReview**: `{selected_task_ids: ["uuid", ...]}`; records high-importance debrief observation per selected task; Weight=3.0

## Routes

| Method | Path | Handler |
|--------|------|---------|
| GET | /api/v1/clarifications | queryQueue — list pending/filtered clarifications |
| GET | /api/v1/clarifications/count | countPending — count pending |
| GET | /api/v1/clarifications/{id} | queryByID — fetch single clarification |
| POST | /api/v1/clarifications/{id}/resolve | resolve — resolve with answer, dispatch side-effects |
| POST | /api/v1/clarifications/{id}/snooze | snooze — snooze for N hours (default 24) |
| POST | /api/v1/clarifications/{id}/dismiss | dismiss — mark as dismissed |

All routes require `X-API-Key` header (mid.Auth middleware).

## Cross-Domain Dependencies

- **taskbus** — resolve may update task status/context or create new task
- **notebus** — resolve may update note's context assignment
- **eventbus** — resolve may update event's context assignment
- **contextbus** — resolve may update context or delete merged context
- **emailbus** — resolve may update email's context assignment
- **observationbus** — ContextDebrief kind records debrief observations
- **threadbus** — InactivityPrompt kind adds thread entry
- **entitylinkbus** — EntityLink kind creates entity links after confirmation
- **clarificationkind, clarificationstatus** — enums in business/types/ define Kind/Status values and KindWeights
