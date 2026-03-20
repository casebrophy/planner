# Backend Phase 3b Activation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Activate the Phase 3b clarification queue so it fills naturally from ingestion, inactivity detection, and context closure — then expose it via MCP tools.

**Architecture:** Seven sequential backend changes: prerequisite enum additions, resolution dispatcher, ingestion wiring, inactivity job, debrief flow, unsnooze job, MCP tools. All follow the existing 3-layer pattern (app → bus → store). Background jobs use goroutines with ticker + context cancellation.

**Tech Stack:** Go, PostgreSQL, sqlx, `clarificationbus`, `ingestbus`, `mcpapp`

**Spec:** `docs/superpowers/specs/2026-03-20-phase3b-activation-and-clarification-ui-design.md`

**Verification:** `make lint && make test` after each task. Final check: `make lint && make test` on the full codebase.

---

## File Map

### Files to modify
| File | Change |
|------|--------|
| `business/types/observationkind/observationkind.go` | Add `Debrief` kind |
| `business/types/threadentrykind/threadentrykind.go` | Add `Note` kind |
| `business/domain/ingestbus/extractor/anthropic.go` | Add `ContextConfidence` field + `Interpretations` on `ActionItem` |
| `app/domain/clarificationapp/clarificationapp.go` | Add resolution dispatcher logic in `resolve` handler (line ~108) |
| `app/domain/clarificationapp/route.go` | Inject `taskBus`, `contextBus`, `emailBus`, `observationBus`, `rawinputBus` |
| `business/domain/ingestbus/ingestbus.go` | Add `clarificationBus` dependency, create clarification items in pipeline |
| `app/domain/contextapp/contextapp.go` | Add debrief trigger in `update` handler when status → closed |
| `app/domain/contextapp/route.go` | Inject `clarificationBus` dependency |
| `app/domain/mcpapp/mcpapp.go` | Add 6 new tool call handlers |
| `app/domain/mcpapp/tools.go` | Add 6 new tool definitions to `tools` slice |
| `app/domain/mcpapp/route.go` | Inject `clarificationBus`, `threadBus`, `observationBus` |
| `api/services/planner/main.go` | Wire clarificationBus (top-level, outside SMTP block), register `clarificationapp.Routes{}`, add goroutines |

### Files to create
| File | Purpose |
|------|---------|
| `business/domain/inactivitybus/inactivitybus.go` | Inactivity detection business logic |
| `business/domain/inactivitybus/model.go` | StaleItem model |
| `business/domain/inactivitybus/stores/inactivitydb/inactivitydb.go` | SQL queries for stale tasks/contexts |

---

## Task 0: Prerequisites — Enum Additions & Main.go Registration

**Files:**
- Modify: `business/types/observationkind/observationkind.go`
- Modify: `business/types/threadentrykind/threadentrykind.go`
- Modify: `business/domain/ingestbus/extractor/anthropic.go`
- Modify: `api/services/planner/main.go`

### Steps

- [ ] **Step 1: Add `Debrief` to observationkind**

In `business/types/observationkind/observationkind.go`, add `Debrief` to the kind constants and the `Parse()` switch. Follow the existing pattern for other kinds.

- [ ] **Step 2: Add `Note` to threadentrykind**

In `business/types/threadentrykind/threadentrykind.go`, add `Note` to the kind constants and the `Parse()` switch.

- [ ] **Step 3: Add `Interpretations` field to ActionItem**

In `business/domain/ingestbus/extractor/anthropic.go`, add to the `ActionItem` struct:

```go
Interpretations []string `json:"interpretations,omitempty"`
```

Update the Anthropic extraction prompt to request interpretations when action items are ambiguous.

- [ ] **Step 4: Ensure clarificationapp.Routes is registered in main.go**

Check `api/services/planner/main.go` — if `clarificationapp.Routes{}` is not in the `mux.WebAPI()` route adders list, add it. Also ensure `clarificationBus` is created at the **top level** (outside the `if cfg.SMTP.Enabled` block) so it's available to all consumers:

```go
// Top-level clarification wiring (NOT inside SMTP block)
clStore := clarificationdb.NewStore(log, db)
clBus := clarificationbus.NewBusiness(log, clStore)
```

- [ ] **Step 5: Run `make lint`**

Run: `make lint`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add business/types/ business/domain/ingestbus/extractor/ api/services/planner/main.go
git commit -m "chore: add prerequisite enum values and wiring for Phase 3b"
```

---

## Task 1: Resolution Dispatcher

**Files:**
- Modify: `app/domain/clarificationapp/route.go`
- Modify: `app/domain/clarificationapp/clarificationapp.go:108`

### Steps

- [ ] **Step 1: Expand the app struct and route wiring**

In `app/domain/clarificationapp/route.go`, add bus dependencies to the app struct and wire them in `Routes.Add()`:

```go
// In route.go — update the app instantiation to include additional buses.
// Follow the same pattern as mcpapp/route.go: create store → create bus → inject.
// Add these imports and instantiations:
//   taskdb.NewStore → taskbus.NewBusiness
//   contextdb.NewStore → contextbus.NewBusiness
//   emaildb.NewStore → emailbus.NewBusiness
//   observationdb.NewStore → observationbus.NewBusiness
//   rawinputdb.NewStore → rawinputbus.NewBusiness
```

Update the `app` struct in `clarificationapp.go`:

```go
type app struct {
	clarificationBus *clarificationbus.Business
	taskBus          *taskbus.Business
	contextBus       *contextbus.Business
	emailBus         *emailbus.Business
	observationBus   *observationbus.Business
	rawinputBus      *rawinputbus.Business
}
```

- [ ] **Step 2: Run `make lint` to verify compilation**

Run: `make lint`
Expected: PASS (struct change + wiring should compile)

- [ ] **Step 3: Implement the resolution dispatcher**

In `clarificationapp.go`, replace the TODO at line ~108 with dispatcher logic:

```go
// After the existing: item, err = a.clarificationBus.Resolve(ctx, item, rc)
// Add dispatch based on item.Kind:

switch item.Kind {
case clarificationkind.ContextAssignment:
    // answer is JSON: {"context_id": "uuid"}
    // Parse answer, call a.emailBus.Update to set context_id

case clarificationkind.InactivityPrompt:
    // answer is JSON: {"action": "close|extend|note", "note": "..."}
    // Based on action, call a.taskBus.Update or a.contextBus.Update

case clarificationkind.AmbiguousAction:
    // answer is JSON: {"title": "...", "description": "...", "context_id": "..."}
    // Call a.taskBus.Create with the clarified action

case clarificationkind.NewContext:
    // answer is JSON: {"action": "confirm|merge", "merge_target_id": "..."}
    // If confirm: no-op (context already exists)
    // If merge: call a.contextBus.Delete on subject, move tasks to merge_target

case clarificationkind.StaleTask:
    // answer is JSON: {"action": "close|deprioritize|note", "note": "..."}
    // Call a.taskBus.Update accordingly

case clarificationkind.AmbiguousDeadline:
    // answer is JSON: {"due_date": "2026-03-25T00:00:00Z"}
    // Call a.taskBus.Update with the correct due date

case clarificationkind.VoiceReference:
    // answer is JSON: {"resolved_text": "...", "task_id": "..."}
    // Update the referenced task or raw_input

case clarificationkind.ContextDebrief:
    // answer is JSON: {"response": "free text answer"}
    // Call a.observationBus.Record with kind=debrief, data=answer
}
```

Each case: unmarshal `item.Answer` into a case-specific struct, validate, call the appropriate bus method. Log errors but don't fail the resolve (the clarification is already resolved; side-effect failure should be logged, not returned to user).

- [ ] **Step 4: Run `make lint`**

Run: `make lint`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add app/domain/clarificationapp/
git commit -m "feat: add resolution dispatcher for clarification side-effects"
```

---

## Task 2: Clarification Generation in Ingestion

**Files:**
- Modify: `business/domain/ingestbus/ingestbus.go`
- Modify: `business/domain/ingestbus/extractor/anthropic.go`
- Modify: `api/services/planner/main.go`

### Steps

- [ ] **Step 1: Add ContextConfidence to EmailExtraction**

In `business/domain/ingestbus/extractor/anthropic.go`, add to the `EmailExtraction` struct:

```go
type EmailExtraction struct {
    // ... existing fields ...
    ContextConfidence float64 `json:"context_confidence"` // 0.0-1.0
}
```

Update the Anthropic extraction prompt (in the `ExtractEmail` method) to request a `context_confidence` score in its JSON response.

- [ ] **Step 2: Add clarificationBus to ingestbus**

In `business/domain/ingestbus/ingestbus.go`:

```go
type Business struct {
    log             *logger.Logger
    rawInputBus     *rawinputbus.Business
    emailBus        *emailbus.Business
    taskBus         *taskbus.Business
    contextBus      *contextbus.Business
    extractor       extractor.Extractor
    clarificationBus *clarificationbus.Business // NEW
}

func NewBusiness(
    log *logger.Logger,
    rawInputBus *rawinputbus.Business,
    emailBus *emailbus.Business,
    taskBus *taskbus.Business,
    contextBus *contextbus.Business,
    ext extractor.Extractor,
    clarificationBus *clarificationbus.Business, // NEW
) *Business {
    return &Business{
        log:              log,
        rawInputBus:      rawInputBus,
        emailBus:         emailBus,
        taskBus:          taskBus,
        contextBus:       contextBus,
        extractor:        ext,
        clarificationBus: clarificationBus,
    }
}
```

- [ ] **Step 3: Wire clarificationBus in main.go**

In `api/services/planner/main.go`, after the existing clarificationBus creation (or create it if not yet instantiated), pass it to `ingestbus.NewBusiness`:

```go
// Add clarification store + bus creation before ingestbus wiring
clStore := clarificationdb.NewStore(log, db)
clBus := clarificationbus.NewBusiness(log, clStore)

// Update ingestbus.NewBusiness call to include clBus
ingestBus := ingestbus.NewBusiness(log, riBus, emBus, tBus, cBus, ext, clBus)
```

- [ ] **Step 4: Run `make lint` to verify wiring compiles**

Run: `make lint`
Expected: PASS

- [ ] **Step 5: Add clarification generation to pipeline**

In `ingestbus.go`, after the context matching step in `processRawInput`, add:

```go
// After extraction and context matching:
if extraction.ContextConfidence < 0.7 && extraction.SuggestedContextID != nil {
    // Low confidence context match — create clarification
    candidatesJSON, _ := json.Marshal(map[string]any{
        "suggested_context_id": *extraction.SuggestedContextID,
        "confidence":           extraction.ContextConfidence,
        "alternatives":         extraction.SuggestedContextKeywords,
    })
    options := json.RawMessage(candidatesJSON)
    b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
        Kind:          clarificationkind.ContextAssignment,
        SubjectType:   "email",
        SubjectID:     email.ID,
        Question:      fmt.Sprintf("Which context does this email from %s belong to? (%.0f%% confidence: suggested %s)", extraction.SenderName, extraction.ContextConfidence*100, *extraction.SuggestedContextID),
        AnswerOptions: options,
    })
}

// After task creation, if action items were ambiguous:
for _, action := range extraction.ActionItems {
    if len(action.Interpretations) > 1 { // if Interpretations field exists
        optionsJSON, _ := json.Marshal(action.Interpretations)
        b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
            Kind:          clarificationkind.AmbiguousAction,
            SubjectType:   "email",
            SubjectID:     email.ID,
            Question:      fmt.Sprintf("Ambiguous action item: '%s' — which interpretation is correct?", action.Title),
            AnswerOptions: json.RawMessage(optionsJSON),
        })
    }
}
```

- [ ] **Step 6: Run `make lint && make test`**

Run: `make lint && make test`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add business/domain/ingestbus/ api/services/planner/main.go
git commit -m "feat: generate clarification items from ingestion pipeline"
```

---

## Task 3: Inactivity Detection Job

**Files:**
- Create: `business/domain/inactivitybus/inactivitybus.go`
- Create: `business/domain/inactivitybus/model.go`
- Create: `business/domain/inactivitybus/stores/inactivitydb/inactivitydb.go`
- Modify: `api/services/planner/main.go`

### Steps

- [ ] **Step 1: Create the StaleItem model**

Create `business/domain/inactivitybus/model.go`:

```go
package inactivitybus

import (
    "time"

    "github.com/google/uuid"
    "github.com/casebrophy/planner/business/types/taskpriority"
)

// StaleItem represents a task or context that hasn't been updated within its threshold.
type StaleItem struct {
    SubjectType  string                // "task" or "context"
    SubjectID    uuid.UUID
    Priority     taskpriority.Priority // for tasks; contexts use ExpectedUpdateDays
    LastThreadAt *time.Time
    DaysSince    int
}
```

- [ ] **Step 2: Create the store**

Create `business/domain/inactivitybus/stores/inactivitydb/inactivitydb.go`:

```go
package inactivitydb

import (
    "context"

    "github.com/casebrophy/planner/business/domain/inactivitybus"
    "github.com/casebrophy/planner/foundation/logger"
    "github.com/jmoiron/sqlx"
)

type Store struct {
    log *logger.Logger
    db  *sqlx.DB
}

func NewStore(log *logger.Logger, db *sqlx.DB) *Store {
    return &Store{log: log, db: db}
}

// QueryStaleTasks returns tasks where time since last thread entry exceeds the priority-based threshold.
func (s *Store) QueryStaleTasks(ctx context.Context) ([]inactivitybus.StaleItem, error) {
    const q = `
        SELECT
            'task' as subject_type,
            t.task_id as subject_id,
            t.priority,
            t.last_thread_at,
            EXTRACT(EPOCH FROM (NOW() - COALESCE(t.last_thread_at, t.created_at))) / 86400 as days_since
        FROM tasks t
        WHERE t.status NOT IN ('done', 'cancelled')
        AND (
            (t.priority = 'urgent' AND COALESCE(t.last_thread_at, t.created_at) < NOW() - INTERVAL '1 day')
            OR (t.priority = 'high' AND COALESCE(t.last_thread_at, t.created_at) < NOW() - INTERVAL '2 days')
            OR (t.priority = 'medium' AND COALESCE(t.last_thread_at, t.created_at) < NOW() - INTERVAL '5 days')
            OR (t.priority = 'low' AND COALESCE(t.last_thread_at, t.created_at) < NOW() - INTERVAL '14 days')
        )
        AND NOT EXISTS (
            SELECT 1 FROM clarification_items ci
            WHERE ci.subject_type = 'task'
            AND ci.subject_id = t.task_id
            AND ci.kind = 'inactivity_prompt'
            AND ci.status = 'pending'
        )`

    var items []StaleItem
    if err := sqlx.SelectContext(ctx, s.db, &items, q); err != nil {
        return nil, err
    }
    return toBusItems(items), nil
}

// QueryStaleContexts returns contexts where time since last thread entry exceeds expected_update_days.
func (s *Store) QueryStaleContexts(ctx context.Context) ([]inactivitybus.StaleItem, error) {
    const q = `
        SELECT
            'context' as subject_type,
            c.context_id as subject_id,
            '' as priority,
            c.last_thread_at,
            EXTRACT(EPOCH FROM (NOW() - COALESCE(c.last_thread_at, c.created_at))) / 86400 as days_since
        FROM contexts c
        WHERE c.status = 'active'
        AND COALESCE(c.last_thread_at, c.created_at) < NOW() - (COALESCE(c.expected_update_days, 7) || ' days')::INTERVAL
        AND NOT EXISTS (
            SELECT 1 FROM clarification_items ci
            WHERE ci.subject_type = 'context'
            AND ci.subject_id = c.context_id
            AND ci.kind = 'inactivity_prompt'
            AND ci.status = 'pending'
        )`

    var items []StaleItem
    if err := sqlx.SelectContext(ctx, s.db, &items, q); err != nil {
        return nil, err
    }
    return toBusItems(items), nil
}
```

Add the DB model struct and converter:

```go
type StaleItem struct {
    SubjectType string  `db:"subject_type"`
    SubjectID   string  `db:"subject_id"`
    Priority    string  `db:"priority"`
    LastThreadAt *time.Time `db:"last_thread_at"`
    DaysSince   float64 `db:"days_since"`
}

func toBusItems(dbItems []StaleItem) []inactivitybus.StaleItem {
    // Convert DB structs → bus structs
}
```

- [ ] **Step 3: Create the business logic**

Create `business/domain/inactivitybus/inactivitybus.go`:

```go
package inactivitybus

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/casebrophy/planner/business/domain/clarificationbus"
    "github.com/casebrophy/planner/business/types/clarificationkind"
    "github.com/casebrophy/planner/foundation/logger"
)

type Storer interface {
    QueryStaleTasks(ctx context.Context) ([]StaleItem, error)
    QueryStaleContexts(ctx context.Context) ([]StaleItem, error)
}

type Business struct {
    log              *logger.Logger
    storer           Storer
    clarificationBus *clarificationbus.Business
}

func NewBusiness(log *logger.Logger, storer Storer, clarificationBus *clarificationbus.Business) *Business {
    return &Business{log: log, storer: storer, clarificationBus: clarificationBus}
}

// CheckAll finds stale tasks and contexts and creates clarification items for each.
func (b *Business) CheckAll(ctx context.Context) error {
    staleTasks, err := b.storer.QueryStaleTasks(ctx)
    if err != nil {
        return fmt.Errorf("query stale tasks: %w", err)
    }

    staleContexts, err := b.storer.QueryStaleContexts(ctx)
    if err != nil {
        return fmt.Errorf("query stale contexts: %w", err)
    }

    for _, item := range append(staleTasks, staleContexts...) {
        options, _ := json.Marshal(map[string]any{
            "actions": []string{"close", "extend", "note"},
        })
        _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
            Kind:        clarificationkind.InactivityPrompt,
            SubjectType: item.SubjectType,
            SubjectID:   item.SubjectID,
            Question:    fmt.Sprintf("No updates on this %s for %d days. What would you like to do?", item.SubjectType, item.DaysSince),
            AnswerOptions: json.RawMessage(options),
        })
        if err != nil {
            b.log.Error(ctx, "inactivity", "msg", "failed to create clarification", "subject", item.SubjectID, "error", err)
        }
    }

    b.log.Info(ctx, "inactivity", "msg", "check complete", "stale_tasks", len(staleTasks), "stale_contexts", len(staleContexts))
    return nil
}
```

- [ ] **Step 4: Run `make lint`**

Run: `make lint`
Expected: PASS

- [ ] **Step 5: Wire the inactivity goroutine in main.go**

In `api/services/planner/main.go`, after existing bus creation:

```go
// Inactivity detection
inactivityStore := inactivitydb.NewStore(log, db)
inactivityBus := inactivitybus.NewBusiness(log, inactivityStore, clBus)

// Background jobs
jobCtx, jobCancel := context.WithCancel(ctx)
defer jobCancel()

// Inactivity check — every 15 minutes
go func() {
    ticker := time.NewTicker(15 * time.Minute)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            if err := inactivityBus.CheckAll(jobCtx); err != nil {
                log.Error(jobCtx, "inactivity", "msg", "check failed", "error", err)
            }
        case <-jobCtx.Done():
            return
        }
    }
}()
```

- [ ] **Step 6: Run `make lint && make test`**

Run: `make lint && make test`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add business/domain/inactivitybus/ api/services/planner/main.go
git commit -m "feat: add inactivity detection job for stale tasks and contexts"
```

---

## Task 4: Context Debrief Flow

**Files:**
- Modify: `app/domain/contextapp/contextapp.go`
- Modify: `app/domain/contextapp/route.go`

### Steps

- [ ] **Step 1: Inject clarificationBus into contextapp**

In `app/domain/contextapp/route.go`, add clarification store/bus creation and pass to app struct. In `contextapp.go`, expand the struct:

```go
type app struct {
    contextBus       *contextbus.Business
    clarificationBus *clarificationbus.Business
}
```

- [ ] **Step 2: Add debrief trigger in the update handler**

In `contextapp.go`, in the `update` method, after the successful `a.contextBus.Update(ctx, c, buc)` call:

```go
// Check if status transitioned to closed
// IMPORTANT: Context.Status type is defined in contextbus package or contextstatus package.
// Verify the actual type by reading contextbus/model.go before using.
// Use the correct closed constant (e.g. contextstatus.Closed or the string "closed").
updated, err := a.contextBus.Update(ctx, c, buc)
if err != nil {
    // existing error handling
}

// Debrief trigger: if status changed to closed
oldClosed := c.Status.String() == "closed"   // adapt to actual type
newClosed := updated.Status.String() == "closed"
if !oldClosed && newClosed {
    // Set debrief_status = pending on the context
    pendingStatus := debriefstatus.Pending
    a.contextBus.Update(ctx, updated, contextbus.UpdateContext{
        DebriefStatus: &pendingStatus,
    })

    snoozeUntil := time.Now().Add(24 * time.Hour)
    debriefQuestions := []string{
        "What was the outcome of this context?",
        "What was the biggest challenge?",
        "What would you do differently next time?",
    }
    for _, q := range debriefQuestions {
        metadata, _ := json.Marshal(map[string]any{
            "context_id":      updated.ID.String(),
            "debrief_question": q,
        })
        _, err := a.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
            Kind:          clarificationkind.ContextDebrief,
            SubjectType:   "context",
            SubjectID:     updated.ID,
            Question:      q,
            AnswerOptions: json.RawMessage(metadata),
            SnoozedUntil:  &snoozeUntil,
        })
        if err != nil {
            // Log but don't fail the update
            log.Error(ctx, "debrief", "msg", "failed to create debrief card", "error", err)
        }
    }
}
```

Note: The `NewClarificationItem` struct has a `SnoozedUntil` field. When set, `clarificationbus.Create` should set status to `snoozed`. Check the existing `Create` method — if it doesn't handle this, the status must be set manually or the Create method must be updated.

- [ ] **Step 3: Run `make lint && make test`**

Run: `make lint && make test`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add app/domain/contextapp/
git commit -m "feat: trigger debrief clarification cards on context closure"
```

---

## Task 5: Unsnooze Expired Job

**Files:**
- Modify: `api/services/planner/main.go`

### Steps

- [ ] **Step 1: Add unsnooze goroutine**

In `main.go`, alongside the inactivity goroutine (using the same `jobCtx`):

```go
// Unsnooze expired — every 5 minutes
go func() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            count, err := clBus.UnsnoozeExpired(jobCtx)
            if err != nil {
                log.Error(jobCtx, "unsnooze", "msg", "failed", "error", err)
            } else if count > 0 {
                log.Info(jobCtx, "unsnooze", "msg", "unsnoozed items", "count", count)
            }
        case <-jobCtx.Done():
            return
        }
    }
}()
```

- [ ] **Step 2: Run `make lint && make test`**

Run: `make lint && make test`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add api/services/planner/main.go
git commit -m "feat: add background job to unsnooze expired clarification items"
```

---

## Task 6: MCP Tools

**Files:**
- Modify: `app/domain/mcpapp/mcpapp.go`
- Modify: `app/domain/mcpapp/route.go`

### Steps

- [ ] **Step 1: Expand mcpapp struct and route wiring**

In `mcpapp/route.go`, add store/bus creation for clarification, thread, and observation. In `mcpapp.go`, expand the struct:

```go
type app struct {
    taskBus          *taskbus.Business
    contextBus       *contextbus.Business
    emailBus         *emailbus.Business
    clarificationBus *clarificationbus.Business
    threadBus        *threadbus.Business
    observationBus   *observationbus.Business
}
```

- [ ] **Step 2: Add tool definitions to tools.go**

In `app/domain/mcpapp/tools.go`, add 6 new entries to the `tools` slice (the `tools/list` handler returns this variable):

```go
{Name: "get_clarification_queue", Description: "Get pending clarification items that need human input", InputSchema: map[string]any{
    "type": "object",
    "properties": map[string]any{
        "status":        map[string]any{"type": "string", "description": "Filter by status (default: pending)", "enum": []string{"pending", "snoozed", "resolved", "dismissed"}},
        "page":          map[string]any{"type": "integer", "description": "Page number (default: 1)"},
        "rows_per_page": map[string]any{"type": "integer", "description": "Items per page (default: 20)"},
    },
}},
{Name: "resolve_clarification", Description: "Resolve a clarification item with an answer", InputSchema: map[string]any{
    "type": "object",
    "properties": map[string]any{
        "id":     map[string]any{"type": "string", "description": "Clarification item ID"},
        "answer": map[string]any{"type": "object", "description": "Answer payload (structure depends on kind)"},
    },
    "required": []string{"id", "answer"},
}},
{Name: "snooze_clarification", Description: "Snooze a clarification item for later", InputSchema: map[string]any{
    "type": "object",
    "properties": map[string]any{
        "id":             map[string]any{"type": "string", "description": "Clarification item ID"},
        "duration_hours": map[string]any{"type": "integer", "description": "Hours to snooze (default: 24)"},
    },
    "required": []string{"id"},
}},
{Name: "dismiss_clarification", Description: "Dismiss a clarification item", InputSchema: map[string]any{
    "type": "object",
    "properties": map[string]any{
        "id": map[string]any{"type": "string", "description": "Clarification item ID"},
    },
    "required": []string{"id"},
}},
{Name: "add_thread_entry", Description: "Add an entry to a task or context thread", InputSchema: map[string]any{
    "type": "object",
    "properties": map[string]any{
        "subject_type": map[string]any{"type": "string", "enum": []string{"task", "context"}},
        "subject_id":   map[string]any{"type": "string", "description": "Task or context ID"},
        "kind":         map[string]any{"type": "string", "description": "Entry kind (note, status_change, update, decision)"},
        "content":      map[string]any{"type": "string", "description": "Entry content"},
        "source":       map[string]any{"type": "string", "description": "Source (user, claude, system)"},
    },
    "required": []string{"subject_type", "subject_id", "kind", "content", "source"},
}},
{Name: "record_outcome", Description: "Record an outcome observation for a task or context", InputSchema: map[string]any{
    "type": "object",
    "properties": map[string]any{
        "subject_type": map[string]any{"type": "string", "enum": []string{"task", "context"}},
        "subject_id":   map[string]any{"type": "string", "description": "Task or context ID"},
        "kind":         map[string]any{"type": "string", "description": "Observation kind"},
        "content":      map[string]any{"type": "string", "description": "Observation content"},
    },
    "required": []string{"subject_type", "subject_id", "kind", "content"},
}},
```

- [ ] **Step 3: Run `make lint`**

Run: `make lint`
Expected: PASS

- [ ] **Step 4: Add tool call handlers**

In the `tools/call` switch, add cases for each tool. Follow the existing pattern (unmarshal args → validate → call bus → textResult):

```go
case "get_clarification_queue":
    // Parse status (default "pending"), page, rows_per_page
    // Build QueryFilter{Status: status}
    // Call a.clarificationBus.Query(ctx, filter, order.By{}, pg)
    // Return textResult(items)

case "resolve_clarification":
    // Parse id, answer
    // Fetch item: a.clarificationBus.QueryByID(ctx, id)
    // Call a.clarificationBus.Resolve(ctx, item, ResolveClarificationItem{Answer: answer})
    // Return textResult(resolved)

case "snooze_clarification":
    // Parse id, duration_hours (default 24)
    // Fetch item, call a.clarificationBus.Snooze(ctx, item, time.Now().Add(hours))
    // Return textResult(snoozed)

case "dismiss_clarification":
    // Parse id
    // Fetch item, call a.clarificationBus.Dismiss(ctx, item)
    // Return textResult(dismissed)

case "add_thread_entry":
    // Parse subject_type, subject_id, kind, content, source as strings
    // IMPORTANT: Parse enums with threadentrykind.Parse(kindStr) and threadsource.Parse(sourceStr)
    // Return errs.New(errs.InvalidArgument, err) if parse fails
    // Call a.threadBus.AddEntry(ctx, threadbus.NewThreadEntry{...})
    // Return textResult(entry)

case "record_outcome":
    // Parse subject_type, subject_id, kind, content as strings
    // IMPORTANT: Parse enum with observationkind.Parse(kindStr)
    // Return errs.New(errs.InvalidArgument, err) if parse fails
    // Call a.observationBus.Record(ctx, observationbus.NewObservation{...})
    // Return textResult(observation)
```

- [ ] **Step 5: Run `make lint && make test`**

Run: `make lint && make test`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add app/domain/mcpapp/
git commit -m "feat: add MCP tools for clarifications, threads, and observations"
```

---

## Final Verification

- [ ] **Run full lint and test suite**

```bash
make lint && make test
```

Expected: All pass. The backend activation is complete.
