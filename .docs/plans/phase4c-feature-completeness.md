# Phase 4c — Feature Completeness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire generators, MCP tools, and trigger logic to existing data infrastructure that isn't connected — clarification card generators, observation query tool, thread extraction flag, and debrief triggers.

**Architecture:** Four independent work streams that converge on the clarification queue: (1) add `get_outcome_observations` MCP tool, (2) add `task_debrief` clarification kind + debrief trigger logic in a new `debriefbus` package, (3) add `overlapping_contexts` detection via keyword/tag matching in `inactivitybus`, (4) add optional AI extraction flag to `threadbus.AddEntry()`. Each stream is independently shippable.

**Tech Stack:** Go, PostgreSQL, JSON-RPC 2.0 (MCP), Claude API (thread extraction only)

**Key decision notes:**
- Thread AI extraction: optional `Extract bool` flag on `NewThreadEntry` — MCP calls skip it (Claude already classifies), system/pipeline calls opt in
- Voice reference cards: deferred to voice capture phase (no source exists)
- Overlapping contexts: keyword/tag matching only (pg_trgm not available, embeddings in Phase 6) — noted as partial
- Context debrief 24h delay: use `snoozed_until` field on creation, existing unsnooze logic handles surfacing

---

### Task 1: Add `task_debrief` clarification kind

**Files:**
- Modify: `business/types/clarificationkind/clarificationkind.go`
- Modify: `business/sdk/migrate/sql/migrate.sql`

This adds a new kind for task completion debriefs. The DB CHECK constraint needs a migration to allow the new value.

- [ ] **Step 1: Add `TaskDebrief` to the clarificationkind enum**

In `business/types/clarificationkind/clarificationkind.go`, add the new kind variable and register it:

```go
// Add after ContextDebrief = Kind{"context_debrief"}
TaskDebrief = Kind{"task_debrief"}
```

Add to `kinds` map:
```go
TaskDebrief.value: TaskDebrief,
```

Add to `KindWeights` map:
```go
TaskDebrief: 0.9,
```

- [ ] **Step 2: Add migration 1.13 to expand the CHECK constraint**

Append to `business/sdk/migrate/sql/migrate.sql`:

```sql
-- Version: 1.13
-- Description: Add task_debrief to clarification_items kind CHECK constraint
ALTER TABLE clarification_items DROP CONSTRAINT IF EXISTS clarification_items_kind_check;
ALTER TABLE clarification_items ADD CONSTRAINT clarification_items_kind_check CHECK (kind IN (
    'context_assignment', 'stale_task', 'ambiguous_deadline',
    'new_context', 'overlapping_contexts', 'ambiguous_action',
    'voice_reference', 'inactivity_prompt', 'context_debrief',
    'task_debrief'
));
```

- [ ] **Step 3: Verify compilation**

Run: `cd /Users/casebrophy/planner && go build ./...`
Expected: clean build

- [ ] **Step 4: Commit**

```bash
git add business/types/clarificationkind/clarificationkind.go business/sdk/migrate/sql/migrate.sql
git commit -m "feat: add task_debrief clarification kind and migration 1.13"
```

---

### Task 2: Add `get_outcome_observations` MCP tool

**Files:**
- Modify: `app/domain/mcpapp/tools.go` (add tool definition)
- Modify: `app/domain/mcpapp/mcpapp.go` (add dispatch case + handler method)

The `observationbus.QueryBySubject()` method already exists. This wires a new MCP tool to expose it.

- [ ] **Step 1: Add tool definition to `tools.go`**

Add after the `record_outcome` tool definition (line ~233) in `app/domain/mcpapp/tools.go`:

```go
{
    Name:        "get_outcome_observations",
    Description: "Query outcome observations for a task or context. Returns all recorded observations ordered by most recent first. Use to understand history, patterns, and lessons from past work.",
    InputSchema: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "subject_type": map[string]any{
                "type":        "string",
                "enum":        []string{"task", "context"},
                "description": "Type of subject to query observations for",
            },
            "subject_id": map[string]any{
                "type":        "string",
                "description": "UUID of the task or context",
            },
            "page": map[string]any{
                "type":        "integer",
                "description": "Page number (default 1)",
            },
            "rows_per_page": map[string]any{
                "type":        "integer",
                "description": "Results per page (default 20)",
            },
        },
        "required": []string{"subject_type", "subject_id"},
    },
},
```

- [ ] **Step 2: Add dispatch case in `callTool()` switch**

In `app/domain/mcpapp/mcpapp.go`, add a new case in the `callTool()` switch (before the `default` case):

```go
case "get_outcome_observations":
    return a.toolGetOutcomeObservations(ctx, params.Arguments)
```

- [ ] **Step 3: Implement `toolGetOutcomeObservations()`**

Add the handler method to `app/domain/mcpapp/mcpapp.go`:

```go
func (a *app) toolGetOutcomeObservations(ctx context.Context, args json.RawMessage) (toolResult, error) {
	var input struct {
		SubjectType string `json:"subject_type"`
		SubjectID   string `json:"subject_id"`
		Page        int    `json:"page"`
		RowsPerPage int    `json:"rows_per_page"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return toolResult{}, fmt.Errorf("invalid arguments: %w", err)
	}

	if input.SubjectType == "" {
		return toolResult{}, fmt.Errorf("subject_type is required")
	}

	subjectID, err := uuid.Parse(input.SubjectID)
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid subject_id: %w", err)
	}

	pageNum := input.Page
	if pageNum < 1 {
		pageNum = 1
	}
	rowsPerPage := input.RowsPerPage
	if rowsPerPage < 1 {
		rowsPerPage = 20
	}

	pg, err := page.Parse(strconv.Itoa(pageNum), strconv.Itoa(rowsPerPage))
	if err != nil {
		return toolResult{}, fmt.Errorf("invalid pagination: %w", err)
	}

	observations, err := a.observationBus.QueryBySubject(ctx, input.SubjectType, subjectID, pg)
	if err != nil {
		return toolResult{}, fmt.Errorf("query observations: %w", err)
	}

	total, err := a.observationBus.Count(ctx, observationbus.QueryFilter{
		SubjectType: &input.SubjectType,
		SubjectID:   &subjectID,
	})
	if err != nil {
		return toolResult{}, fmt.Errorf("count observations: %w", err)
	}

	return textResult(map[string]any{
		"observations": observations,
		"total":        total,
		"page":         pageNum,
		"rows_per_page": rowsPerPage,
	})
}
```

- [ ] **Step 4: Verify compilation**

Run: `cd /Users/casebrophy/planner && go build ./...`
Expected: clean build

- [ ] **Step 5: Commit**

```bash
git add app/domain/mcpapp/tools.go app/domain/mcpapp/mcpapp.go
git commit -m "feat: add get_outcome_observations MCP tool"
```

---

### Task 3: Create `debriefbus` — debrief trigger logic

**Files:**
- Create: `business/domain/debriefbus/debriefbus.go`
- Create: `business/domain/debriefbus/model.go`

A new business package that generates debrief clarification cards. It's called from task/context update flows.

- [ ] **Step 1: Create `debriefbus/model.go`**

```go
package debriefbus

import (
	"github.com/google/uuid"
)

// CompletedTask holds the data needed to evaluate debrief triggers for a task.
type CompletedTask struct {
	ID           uuid.UUID
	Title        string
	DurationMin  *int       // estimated duration
	CreatedAt    int64      // unix timestamp
	CompletedAt  int64      // unix timestamp
	HasBlockers  bool       // true if thread has blocker entries
}

// ClosedContext holds the data needed to generate context debrief cards.
type ClosedContext struct {
	ID    uuid.UUID
	Title string
}
```

- [ ] **Step 2: Create `debriefbus/debriefbus.go`**

```go
package debriefbus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/clarificationstatus"
	"github.com/casebrophy/planner/business/types/threadentrykind"
	"github.com/casebrophy/planner/foundation/logger"
)

// Business manages debrief card generation on task/context completion.
type Business struct {
	log              *logger.Logger
	clarificationBus *clarificationbus.Business
	threadBus        *threadbus.Business
}

// NewBusiness creates a new debrief business layer.
func NewBusiness(log *logger.Logger, clarificationBus *clarificationbus.Business, threadBus *threadbus.Business) *Business {
	return &Business{
		log:              log,
		clarificationBus: clarificationBus,
		threadBus:        threadBus,
	}
}

// OnTaskCompleted evaluates whether a debrief card should be generated for a
// completed task. A card is generated when:
//   - Actual duration > 2x estimated duration, OR
//   - Thread contains blocker entries
//
// If neither condition is met, no card is created (simple completions don't
// need a debrief).
func (b *Business) OnTaskCompleted(ctx context.Context, ct CompletedTask) error {
	// Check for existing pending task_debrief for this task
	kind := clarificationkind.TaskDebrief
	pending := clarificationstatus.Pending
	subjectType := "task"
	existing, err := b.clarificationBus.Count(ctx, clarificationbus.QueryFilter{
		Kind:        &kind,
		Status:      &pending,
		SubjectType: &subjectType,
		SubjectID:   &ct.ID,
	})
	if err != nil {
		return fmt.Errorf("check existing debrief: %w", err)
	}
	if existing > 0 {
		return nil
	}

	// Check for blockers in thread
	hasBlockers, err := b.hasBlockerEntries(ctx, "task", ct.ID)
	if err != nil {
		return fmt.Errorf("check blockers: %w", err)
	}

	// Check for duration overrun (actual > 2x estimate)
	durationOverrun := false
	if ct.DurationMin != nil && *ct.DurationMin > 0 {
		actualMinutes := float64(ct.CompletedAt-ct.CreatedAt) / 60
		estimatedMinutes := float64(*ct.DurationMin)
		if actualMinutes > estimatedMinutes*2 {
			durationOverrun = true
		}
	}

	if !hasBlockers && !durationOverrun {
		return nil // no debrief needed for clean completions
	}

	// Build question based on trigger
	question := fmt.Sprintf("Task '%s' is done.", ct.Title)
	if durationOverrun {
		question += " It took significantly longer than estimated. What caused the overrun?"
	} else if hasBlockers {
		question += " It had blockers along the way. What finally unblocked it?"
	}

	optionsJSON, _ := json.Marshal([]map[string]string{
		{"label": "Add a lesson", "action": "lesson"},
		{"label": "Nothing notable", "action": "skip"},
		{"label": "Snooze", "action": "snooze"},
	})

	if _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:          clarificationkind.TaskDebrief,
		SubjectType:   "task",
		SubjectID:     ct.ID,
		Question:      question,
		AnswerOptions: json.RawMessage(optionsJSON),
		PriorityScore: 0.9,
	}); err != nil {
		return fmt.Errorf("create task debrief: %w", err)
	}

	return nil
}

// OnContextClosed generates a 3-4 card closing review sequence for a closed
// context. Cards are created with snoozed_until = now + 24h so they surface
// after a cooling-off period.
func (b *Business) OnContextClosed(ctx context.Context, cc ClosedContext) error {
	// Check for existing pending context_debrief for this context
	kind := clarificationkind.ContextDebrief
	pending := clarificationstatus.Pending
	snoozed := clarificationstatus.Snoozed
	subjectType := "context"

	existingPending, err := b.clarificationBus.Count(ctx, clarificationbus.QueryFilter{
		Kind:        &kind,
		Status:      &pending,
		SubjectType: &subjectType,
		SubjectID:   &cc.ID,
	})
	if err != nil {
		return fmt.Errorf("check existing debrief: %w", err)
	}

	existingSnoozed, err := b.clarificationBus.Count(ctx, clarificationbus.QueryFilter{
		Kind:        &kind,
		Status:      &snoozed,
		SubjectType: &subjectType,
		SubjectID:   &cc.ID,
	})
	if err != nil {
		return fmt.Errorf("check existing snoozed debrief: %w", err)
	}

	if existingPending > 0 || existingSnoozed > 0 {
		return nil
	}

	snoozedUntil := time.Now().Add(24 * time.Hour)

	// Card 1: Outcome
	outcomeOptions, _ := json.Marshal([]map[string]string{
		{"label": "Went well", "action": "went_well"},
		{"label": "Mixed results", "action": "mixed"},
		{"label": "Difficult", "action": "difficult"},
		{"label": "Skip debrief", "action": "skip"},
	})
	if _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:          clarificationkind.ContextDebrief,
		SubjectType:   "context",
		SubjectID:     cc.ID,
		Question:      fmt.Sprintf("Context '%s' is closed. How did it go overall?", cc.Title),
		AnswerOptions: json.RawMessage(outcomeOptions),
		PriorityScore: 0.8,
		SnoozedUntil:  &snoozedUntil,
	}); err != nil {
		return fmt.Errorf("create outcome card: %w", err)
	}

	// Card 2: Biggest challenge
	challengeOptions, _ := json.Marshal([]map[string]string{
		{"label": "Timeline pressure", "action": "timeline"},
		{"label": "Unclear requirements", "action": "requirements"},
		{"label": "External dependencies", "action": "dependencies"},
		{"label": "No major challenges", "action": "none"},
	})
	if _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:          clarificationkind.ContextDebrief,
		SubjectType:   "context",
		SubjectID:     cc.ID,
		Question:      fmt.Sprintf("What was the biggest challenge with '%s'?", cc.Title),
		AnswerOptions: json.RawMessage(challengeOptions),
		PriorityScore: 0.7,
		SnoozedUntil:  &snoozedUntil,
	}); err != nil {
		return fmt.Errorf("create challenge card: %w", err)
	}

	// Card 3: Lesson (free text)
	lessonOptions, _ := json.Marshal([]map[string]string{
		{"label": "Add a lesson", "action": "lesson"},
		{"label": "Nothing to add", "action": "skip"},
	})
	if _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:          clarificationkind.ContextDebrief,
		SubjectType:   "context",
		SubjectID:     cc.ID,
		Question:      fmt.Sprintf("Any lessons or insights from '%s' worth remembering?", cc.Title),
		AnswerOptions: json.RawMessage(lessonOptions),
		PriorityScore: 0.6,
		SnoozedUntil:  &snoozedUntil,
	}); err != nil {
		return fmt.Errorf("create lesson card: %w", err)
	}

	return nil
}

// hasBlockerEntries checks if a subject's thread contains any blocker entries.
func (b *Business) hasBlockerEntries(ctx context.Context, subjectType string, subjectID uuid.UUID) (bool, error) {
	blockerKind := threadentrykind.Blocker
	filter := threadbus.QueryFilter{
		SubjectType: &subjectType,
		SubjectID:   &subjectID,
		Kind:        &blockerKind,
	}

	count, err := b.threadBus.Count(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("count blockers: %w", err)
	}

	return count > 0, nil
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd /Users/casebrophy/planner && go build ./...`
Expected: clean build. Check that `clarificationstatus.Snoozed` exists — if not, it's the status value for snoozed items.

- [ ] **Step 4: Commit**

```bash
git add business/domain/debriefbus/
git commit -m "feat: add debriefbus for task/context debrief card generation"
```

---

### Task 4: Wire debrief triggers into task and context update flows

**Files:**
- Modify: `app/domain/mcpapp/mcpapp.go` (add debriefBus field, wire into complete/update handlers)
- Modify: `app/domain/mcpapp/route.go` (construct and inject debriefBus)

The debrief triggers fire from the MCP layer (not the business layer) to keep taskbus/contextbus lean and dependency-free. When `complete_task` or `update_task` transitions status to `done`, call `debriefBus.OnTaskCompleted()`. When `update_context` transitions status to `closed`, call `debriefBus.OnContextClosed()`.

- [ ] **Step 1: Add `debriefBus` to `app` struct**

In `app/domain/mcpapp/mcpapp.go`, add to the `app` struct:

```go
type app struct {
	taskBus          *taskbus.Business
	contextBus       *contextbus.Business
	emailBus         *emailbus.Business
	clarificationBus *clarificationbus.Business
	threadBus        *threadbus.Business
	observationBus   *observationbus.Business
	debriefBus       *debriefbus.Business
}
```

Add the import: `"github.com/casebrophy/planner/business/domain/debriefbus"`

- [ ] **Step 2: Construct debriefBus in route.go**

In `app/domain/mcpapp/route.go`, after creating `thBus` and `obBus`, add:

```go
dbBus := debriefbus.NewBusiness(cfg.Log, clBus, thBus)
```

And add to the struct literal:

```go
hdl := &app{
    taskBus:          taskBus,
    contextBus:       ctxBus,
    emailBus:         emBus,
    clarificationBus: clBus,
    threadBus:        thBus,
    observationBus:   obBus,
    debriefBus:       dbBus,
}
```

Add the import: `"github.com/casebrophy/planner/business/domain/debriefbus"`

- [ ] **Step 3: Wire task debrief into `toolCompleteTask()`**

Find `toolCompleteTask()` in `mcpapp.go`. After the task is successfully updated to `done` status, add:

```go
// Fire debrief trigger (best-effort, don't fail the completion)
go func() {
    ct := debriefbus.CompletedTask{
        ID:          task.ID,
        Title:       task.Title,
        DurationMin: task.DurationMin,
        CreatedAt:   task.CreatedAt.Unix(),
        CompletedAt: task.CompletedAt.Unix(),
    }
    if err := a.debriefBus.OnTaskCompleted(context.Background(), ct); err != nil {
        // Log but don't fail the completion
    }
}()
```

- [ ] **Step 4: Wire task debrief into `toolUpdateTask()`**

Find `toolUpdateTask()` in `mcpapp.go`. After the update succeeds, check if status was changed to `done`:

```go
// Check if task was just completed via status update
if input.Status != nil && *input.Status == "done" && task.CompletedAt != nil {
    go func() {
        ct := debriefbus.CompletedTask{
            ID:          task.ID,
            Title:       task.Title,
            DurationMin: task.DurationMin,
            CreatedAt:   task.CreatedAt.Unix(),
            CompletedAt: task.CompletedAt.Unix(),
        }
        if err := a.debriefBus.OnTaskCompleted(context.Background(), ct); err != nil {
            // Log but don't fail the update
        }
    }()
}
```

- [ ] **Step 5: Wire context debrief into `toolUpdateContext()`**

Find `toolUpdateContext()` in `mcpapp.go`. After the update succeeds, check if status was changed to `closed`:

```go
// Check if context was just closed
if input.Status != nil && *input.Status == "closed" {
    go func() {
        cc := debriefbus.ClosedContext{
            ID:    updatedCtx.ID,
            Title: updatedCtx.Title,
        }
        if err := a.debriefBus.OnContextClosed(context.Background(), cc); err != nil {
            // Log but don't fail the update
        }
    }()
}
```

- [ ] **Step 6: Verify compilation**

Run: `cd /Users/casebrophy/planner && go build ./...`
Expected: clean build

- [ ] **Step 7: Commit**

```bash
git add app/domain/mcpapp/mcpapp.go app/domain/mcpapp/route.go
git commit -m "feat: wire debrief triggers into task completion and context close"
```

---

### Task 5: Add `overlapping_contexts` detection to inactivitybus

**Files:**
- Modify: `business/domain/inactivitybus/inactivitybus.go`
- Modify: `business/domain/inactivitybus/model.go` (if StaleItem needs extension or new type)
- Modify: Storer interface in `inactivitybus.go`
- Modify: `business/domain/inactivitybus/stores/inactivitydb/inactivitydb.go`

Add keyword/tag-based overlap detection. This is a **partial** implementation — true overlap detection requires embeddings (Phase 6). This catches obvious cases: contexts sharing 2+ tags or very similar titles.

- [ ] **Step 1: Check existing model.go for StaleItem**

Read `business/domain/inactivitybus/model.go` for the `StaleItem` type and add a new `OverlapPair` type.

Add to model.go:
```go
// OverlapPair represents two contexts that may overlap.
type OverlapPair struct {
	ContextID1 uuid.UUID
	Title1     string
	ContextID2 uuid.UUID
	Title2     string
	SharedTags int
}
```

- [ ] **Step 2: Add `QueryOverlappingContexts()` to Storer interface**

In `business/domain/inactivitybus/inactivitybus.go`, add to the Storer interface:

```go
// QueryOverlappingContexts returns pairs of active contexts that share 2+
// tags. This is a keyword-based heuristic — true similarity requires embeddings.
QueryOverlappingContexts(ctx context.Context) ([]OverlapPair, error)
```

- [ ] **Step 3: Implement the SQL query in `inactivitydb`**

In `business/domain/inactivitybus/stores/inactivitydb/inactivitydb.go`, add:

```go
func (s *Store) QueryOverlappingContexts(ctx context.Context) ([]OverlapPair, error) {
	const q = `
		SELECT
			ct1.context_id AS context_id_1,
			c1.title AS title_1,
			ct2.context_id AS context_id_2,
			c2.title AS title_2,
			COUNT(*) AS shared_tags
		FROM context_tags ct1
		JOIN context_tags ct2 ON ct1.tag_id = ct2.tag_id AND ct1.context_id < ct2.context_id
		JOIN contexts c1 ON c1.context_id = ct1.context_id AND c1.status = 'active'
		JOIN contexts c2 ON c2.context_id = ct2.context_id AND c2.status = 'active'
		GROUP BY ct1.context_id, c1.title, ct2.context_id, c2.title
		HAVING COUNT(*) >= 2
		ORDER BY shared_tags DESC
		LIMIT 10`

	var results []struct {
		ContextID1 uuid.UUID `db:"context_id_1"`
		Title1     string    `db:"title_1"`
		ContextID2 uuid.UUID `db:"context_id_2"`
		Title2     string    `db:"title_2"`
		SharedTags int       `db:"shared_tags"`
	}

	if err := sqldb.NamedQuerySlice(ctx, s.db, q, struct{}{}, &results); err != nil {
		return nil, fmt.Errorf("query overlapping contexts: %w", err)
	}

	pairs := make([]inactivitybus.OverlapPair, len(results))
	for i, r := range results {
		pairs[i] = inactivitybus.OverlapPair{
			ContextID1: r.ContextID1,
			Title1:     r.Title1,
			ContextID2: r.ContextID2,
			Title2:     r.Title2,
			SharedTags: r.SharedTags,
		}
	}

	return pairs, nil
}
```

- [ ] **Step 4: Add `CheckOverlaps()` to inactivitybus**

In `business/domain/inactivitybus/inactivitybus.go`, add a new method:

```go
// CheckOverlaps scans for active contexts that share 2+ tags and creates
// overlapping_contexts clarification cards. This is keyword-based only —
// embedding-based similarity detection is deferred to Phase 6.
func (b *Business) CheckOverlaps(ctx context.Context) error {
	pairs, err := b.storer.QueryOverlappingContexts(ctx)
	if err != nil {
		return fmt.Errorf("query overlapping contexts: %w", err)
	}

	for _, pair := range pairs {
		if err := b.createOverlapPrompt(ctx, pair); err != nil {
			b.log.Error(ctx, "inactivity", "msg", "failed to create overlap prompt", "error", err,
				"context_1", pair.ContextID1, "context_2", pair.ContextID2)
		}
	}

	b.log.Info(ctx, "inactivity", "msg", "overlap check complete", "pairs_found", len(pairs))
	return nil
}

func (b *Business) createOverlapPrompt(ctx context.Context, pair OverlapPair) error {
	// Check for existing pending overlap card for either context in the pair
	kind := clarificationkind.OverlappingContexts
	pending := clarificationstatus.Pending
	subjectType := "context"

	existing1, err := b.clarificationBus.Count(ctx, clarificationbus.QueryFilter{
		Kind:        &kind,
		Status:      &pending,
		SubjectType: &subjectType,
		SubjectID:   &pair.ContextID1,
	})
	if err != nil {
		return fmt.Errorf("check existing: %w", err)
	}
	if existing1 > 0 {
		return nil
	}

	optionsJSON, _ := json.Marshal([]map[string]string{
		{"label": "Keep separate", "action": "keep"},
		{"label": "Merge them", "action": "merge"},
		{"label": "Dismiss", "action": "dismiss"},
	})

	question := fmt.Sprintf("'%s' and '%s' share %d tags — are these overlapping? Should they be merged?",
		pair.Title1, pair.Title2, pair.SharedTags)

	if _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:          clarificationkind.OverlappingContexts,
		SubjectType:   "context",
		SubjectID:     pair.ContextID1,
		Question:      question,
		AnswerOptions: json.RawMessage(optionsJSON),
	}); err != nil {
		return fmt.Errorf("create clarification: %w", err)
	}

	return nil
}
```

- [ ] **Step 5: Wire `CheckOverlaps()` into `CheckAll()`**

At the end of `CheckAll()`, before the final log line, add:

```go
if err := b.CheckOverlaps(ctx); err != nil {
    b.log.Error(ctx, "inactivity", "msg", "overlap check failed", "error", err)
}
```

- [ ] **Step 6: Verify compilation**

Run: `cd /Users/casebrophy/planner && go build ./...`
Expected: clean build

- [ ] **Step 7: Commit**

```bash
git add business/domain/inactivitybus/ business/domain/inactivitybus/stores/inactivitydb/
git commit -m "feat: add overlapping_contexts detection via shared tags (keyword-based)"
```

---

### Task 6: Add optional AI extraction to thread entries

**Files:**
- Modify: `business/domain/threadbus/model.go` (add `Extract` field to `NewThreadEntry`)
- Modify: `business/domain/threadbus/threadbus.go` (add extractor dependency, conditional extraction)
- Create: `business/domain/threadbus/extractor.go` (Extractor interface + extraction logic)

- [ ] **Step 1: Create `threadbus/extractor.go` with the Extractor interface**

```go
package threadbus

import "context"

// Extractor classifies raw thread entry text into structured fields.
// Implemented by the Claude client for AI extraction.
type Extractor interface {
	ExtractThreadEntry(ctx context.Context, content string, subjectType string) (ExtractionResult, error)
}

// ExtractionResult contains the structured fields extracted from a thread entry.
type ExtractionResult struct {
	Kind              string  `json:"kind"`
	Sentiment         *string `json:"sentiment"`
	BlockingParty     *string `json:"blocking_party"`
	TimelineDeltaDays *int    `json:"timeline_delta_days"`
	RequiresAction    bool    `json:"requires_action"`
	Confidence        float64 `json:"confidence"`
}
```

- [ ] **Step 2: Add `Extract` field to `NewThreadEntry`**

In `business/domain/threadbus/model.go`, add to `NewThreadEntry`:

```go
type NewThreadEntry struct {
	SubjectType    string
	SubjectID      uuid.UUID
	Kind           threadentrykind.Kind
	Content        string
	Metadata       *json.RawMessage
	Source         threadsource.Source
	SourceID       *uuid.UUID
	Sentiment      *string
	RequiresAction bool
	Extract        bool // When true, run AI extraction to classify kind/sentiment/etc.
}
```

- [ ] **Step 3: Update Business struct to accept optional Extractor**

In `business/domain/threadbus/threadbus.go`, update:

```go
type Business struct {
	log       *logger.Logger
	storer    Storer
	extractor Extractor // nil = no extraction
}

func NewBusiness(log *logger.Logger, storer Storer) *Business {
	return &Business{
		log:    log,
		storer: storer,
	}
}

// WithExtractor sets an optional AI extractor for thread entry classification.
func (b *Business) WithExtractor(ext Extractor) {
	b.extractor = ext
}
```

- [ ] **Step 4: Add extraction logic to `AddEntry()`**

Update `AddEntry()` to conditionally run extraction:

```go
func (b *Business) AddEntry(ctx context.Context, ne NewThreadEntry) (ThreadEntry, error) {
	now := time.Now()

	kind := ne.Kind
	sentiment := ne.Sentiment
	requiresAction := ne.RequiresAction
	metadata := ne.Metadata

	// Run AI extraction if requested and extractor is available
	if ne.Extract && b.extractor != nil {
		result, err := b.extractor.ExtractThreadEntry(ctx, ne.Content, ne.SubjectType)
		if err != nil {
			b.log.Error(ctx, "threadbus", "msg", "extraction failed, using defaults", "error", err)
		} else if result.Confidence >= 0.6 {
			if parsed, err := threadentrykind.Parse(result.Kind); err == nil {
				kind = parsed
			}
			if result.Sentiment != nil {
				sentiment = result.Sentiment
			}
			requiresAction = result.RequiresAction

			// Store extraction metadata
			metaJSON, _ := json.Marshal(result)
			raw := json.RawMessage(metaJSON)
			metadata = &raw
		}
	}

	entry := ThreadEntry{
		ID:             uuid.New(),
		SubjectType:    ne.SubjectType,
		SubjectID:      ne.SubjectID,
		Kind:           kind,
		Content:        ne.Content,
		Metadata:       metadata,
		Source:         ne.Source,
		SourceID:       ne.SourceID,
		Sentiment:      sentiment,
		RequiresAction: requiresAction,
		CreatedAt:      now,
	}

	if err := b.storer.Create(ctx, entry); err != nil {
		return ThreadEntry{}, fmt.Errorf("create: %w", err)
	}

	return entry, nil
}
```

Add the `"encoding/json"` import to the file.

- [ ] **Step 5: Verify compilation**

Run: `cd /Users/casebrophy/planner && go build ./...`
Expected: clean build. The `Extract` field defaults to `false` (zero value), so all existing callers are unaffected.

- [ ] **Step 6: Commit**

```bash
git add business/domain/threadbus/
git commit -m "feat: add optional AI extraction flag to thread entry creation"
```

---

### Task 7: Update priority scoring in clarificationbus

**Files:**
- Modify: `business/domain/clarificationbus/clarificationbus.go` (update Create to compute priority_score using kind weights)

Currently `PriorityScore` is set by the caller. The design spec says: `priority_score = age_hours × 0.4 + kind_weight × 0.6`. At creation time, age is 0, so initial score = `kind_weight × 0.6`. This ensures newly created cards of high-weight kinds (like `TaskDebrief` at 0.9) rank higher.

- [ ] **Step 1: Read `clarificationbus.go` Create method**

Check the current `Create()` implementation to see where `PriorityScore` is set.

- [ ] **Step 2: Update `Create()` to compute initial priority score**

In `business/domain/clarificationbus/clarificationbus.go`, update the `Create` method to use kind weights when the caller doesn't set a score:

```go
// In Create(), after building the ClarificationItem:
if item.PriorityScore == 0 {
    if weight, ok := clarificationkind.KindWeights[item.Kind]; ok {
        item.PriorityScore = weight * 0.6
    }
}
```

This uses the existing `KindWeights` map from `clarificationkind` and applies the `× 0.6` factor from the design spec (since age component is 0 at creation).

- [ ] **Step 3: Verify compilation**

Run: `cd /Users/casebrophy/planner && go build ./...`
Expected: clean build

- [ ] **Step 4: Commit**

```bash
git add business/domain/clarificationbus/clarificationbus.go
git commit -m "feat: compute initial priority score from kind weights on clarification create"
```

---

### Task 8: Update TOC.md and docs

**Files:**
- Modify: `.docs/TOC.md` (add debriefbus references)
- Modify: `.docs/07-roadmap.md` (mark Phase 4c items as done)

- [ ] **Step 1: Add debriefbus to TOC.md**

Add under `## By Domain`, in the appropriate alphabetical position:
```markdown
- debrief: `07-roadmap.md#phase-4c--feature-completeness`
```

- [ ] **Step 2: Update TOC implementation plans section**

Add:
```markdown
- phase-4c-feature-completeness: `plans/phase4c-feature-completeness.md`
```

- [ ] **Step 3: Verify all files compile**

Run: `cd /Users/casebrophy/planner && go build ./...`
Expected: clean build

- [ ] **Step 4: Commit**

```bash
git add .docs/TOC.md
git commit -m "docs: update TOC with Phase 4c references"
```

---

## Verification

After all tasks are complete:

1. **Compile check:** `go build ./...` — must succeed
2. **Migration check:** Run `make migrate` against a local DB to verify migration 1.13 applies cleanly
3. **MCP tool check:** Start the server (`make dev`), send a `tools/list` JSON-RPC request, verify `get_outcome_observations` appears in the response
4. **Debrief trigger check:** Complete a task via MCP `complete_task` tool, check if a `task_debrief` clarification item appears in the DB (only if the task had blockers or duration overrun)
5. **Overlap check:** Create two contexts with 2+ shared tags, run the inactivity check job, verify an `overlapping_contexts` card appears
6. **Thread extraction:** Verify `AddEntry()` still works with `Extract: false` (existing behavior) — all current callers should be unaffected

## Known limitations (documented, not addressed)

- **Overlapping contexts detection** is keyword/tag-based only — true semantic similarity requires Phase 6 embeddings
- **Voice reference cards** are deferred until voice capture source exists
- **Thread AI extraction** has the interface but no implementation yet — wire a concrete extractor when needed
- **"By the way" section** after clearing 5+ cards is a frontend concern, not addressed here
