# Outcome Feedback Cards Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make task debriefs fire on every completion (not just overrun/blockers), wire them into the REST API (currently only MCP), and add a weekly impact review card -- all using the existing clarification card infrastructure.

**Architecture:** Expand `debriefbus` to handle all outcome feedback. Fix the bug where `taskapp` never calls `debriefBus.OnTaskCompleted()`. Add `weekly_review` as a new clarification kind with a scheduled generator. Fix stale SQL CHECK constraints that are missing `task_debrief`, `entity_link`, and the new `weekly_review` kind.

**Tech Stack:** Go backend, PostgreSQL migrations, Vue 3 frontend (ClarificationCard.vue), TypeScript codegen

---

## File Structure

| Action | File | Responsibility |
|--------|------|----------------|
| MODIFY | `business/sdk/migrate/sql/migrate.sql` | v1.26: fix stale CHECK constraints, add `weekly_review` kind + `week` subject_type |
| MODIFY | `business/types/clarificationkind/clarificationkind.go` | Add `WeeklyReview` enum value |
| MODIFY | `business/domain/debriefbus/model.go` | Add `WeeklyReviewTask` struct |
| MODIFY | `business/domain/debriefbus/debriefbus.go` | Expand `OnTaskCompleted` to fire always; add `GenerateWeeklyReview` |
| MODIFY | `app/domain/taskapp/taskapp.go` | Wire `debriefBus.OnTaskCompleted()` into update handler |
| MODIFY | `app/domain/taskapp/route.go` | Add `debriefBus` dependency wiring |
| MODIFY | `app/domain/clarificationapp/clarificationapp.go` | Add `TaskDebrief` + `WeeklyReview` resolution side-effects |
| MODIFY | `api/services/planner/main.go` | Add weekly review scheduled job |
| MODIFY | `api/services/frontend/web/src/types/enums.ts` | Add `WeeklyReview` to labels + colors |
| MODIFY | `api/services/frontend/web/src/components/clarifications/ClarificationCard.vue` | Redesign TaskDebrief UI (buttons), add WeeklyReview UI (multi-select) |
| REGEN  | `api/services/frontend/web/src/types/generated/clarification-kind.ts` | Run `gen-ts-kinds` |

---

### Task 1: Migration -- fix stale CHECK constraints + add weekly_review

**Files:**
- Modify: `business/sdk/migrate/sql/migrate.sql` (append after line 429)

- [ ] **Step 1: Add migration v1.26**

Append to `business/sdk/migrate/sql/migrate.sql`:

```sql
-- Version: 1.26
-- Description: Fix stale clarification kind CHECK, add weekly_review kind and week subject_type
ALTER TABLE clarification_items DROP CONSTRAINT IF EXISTS clarification_items_kind_check;
ALTER TABLE clarification_items ADD CONSTRAINT clarification_items_kind_check CHECK (kind IN (
    'context_assignment', 'stale_task', 'ambiguous_deadline',
    'new_context', 'overlapping_contexts', 'ambiguous_action',
    'voice_reference', 'inactivity_prompt', 'context_debrief',
    'task_debrief', 'entity_link', 'weekly_review'
));

ALTER TABLE clarification_items DROP CONSTRAINT IF EXISTS clarification_items_subject_type_check;
ALTER TABLE clarification_items ADD CONSTRAINT clarification_items_subject_type_check CHECK (subject_type IN (
    'task', 'context', 'email', 'raw_input', 'week'
));
```

- [ ] **Step 2: Run migration to verify**

Run: `make migrate`
Expected: migration 1.26 applies without error.

- [ ] **Step 3: Commit**

```bash
git add business/sdk/migrate/sql/migrate.sql
git commit -m "fix: update stale clarification CHECK constraints, add weekly_review kind"
```

---

### Task 2: Add WeeklyReview enum + model types

**Files:**
- Modify: `business/types/clarificationkind/clarificationkind.go`
- Modify: `business/domain/debriefbus/model.go`

- [ ] **Step 1: Add WeeklyReview to clarificationkind**

In `business/types/clarificationkind/clarificationkind.go`, add `WeeklyReview` to the var block after `EntityLink`:

```go
WeeklyReview = Kind{"weekly_review"}
```

Add to `kinds` map:

```go
WeeklyReview.value: WeeklyReview,
```

Add to `AllKinds` slice:

```go
WeeklyReview,
```

Add to `KindWeights` map:

```go
WeeklyReview: 0.8,
```

- [ ] **Step 2: Add WeeklyReviewTask to debriefbus model**

In `business/domain/debriefbus/model.go`, add:

```go
// WeeklyReviewTask is a lightweight task reference for the weekly review card.
type WeeklyReviewTask struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
}
```

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 4: Commit**

```bash
git add business/types/clarificationkind/clarificationkind.go business/domain/debriefbus/model.go
git commit -m "feat: add WeeklyReview clarification kind and model types"
```

---

### Task 3: Expand OnTaskCompleted to fire on every completion

**Files:**
- Modify: `business/domain/debriefbus/debriefbus.go`

The current `OnTaskCompleted` (lines 39-102) only fires when there's a duration overrun >2x or blocker thread entries. We're removing that guard so it fires on every completion, with an adaptive question.

- [ ] **Step 1: Rewrite OnTaskCompleted**

Replace `OnTaskCompleted` in `business/domain/debriefbus/debriefbus.go` (lines 39-102) with:

```go
// OnTaskCompleted generates a task debrief card for every completed task.
// The question adapts based on whether the task had a duration estimate and
// whether it overran.
func (b *Business) OnTaskCompleted(ctx context.Context, ct CompletedTask) error {
	kind := clarificationkind.TaskDebrief
	pending := clarificationstatus.Pending
	snoozed := clarificationstatus.Snoozed
	subjectType := "task"

	existingPending, err := b.clarificationBus.Count(ctx, clarificationbus.QueryFilter{
		Kind:        &kind,
		Status:      &pending,
		SubjectType: &subjectType,
		SubjectID:   &ct.ID,
	})
	if err != nil {
		return fmt.Errorf("check existing debrief: %w", err)
	}

	existingSnoozed, err := b.clarificationBus.Count(ctx, clarificationbus.QueryFilter{
		Kind:        &kind,
		Status:      &snoozed,
		SubjectType: &subjectType,
		SubjectID:   &ct.ID,
	})
	if err != nil {
		return fmt.Errorf("check existing snoozed debrief: %w", err)
	}

	if existingPending > 0 || existingSnoozed > 0 {
		return nil
	}

	question := fmt.Sprintf("You completed '%s'. How important was this?", ct.Title)
	if ct.DurationMin != nil && *ct.DurationMin > 0 {
		actualMinutes := float64(ct.CompletedAt-ct.CreatedAt) / 60
		estimatedMinutes := float64(*ct.DurationMin)
		if actualMinutes > estimatedMinutes*2 {
			question = fmt.Sprintf("You completed '%s' — it took much longer than the %d min estimate. How important was this?", ct.Title, *ct.DurationMin)
		}
	}

	optionsJSON, err := json.Marshal([]map[string]string{
		{"label": "High impact", "value": "high"},
		{"label": "Medium impact", "value": "medium"},
		{"label": "Low impact", "value": "low"},
		{"label": "Not worth it", "value": "waste"},
		{"label": "Skip", "value": "skip"},
	})
	if err != nil {
		return fmt.Errorf("marshal options: %w", err)
	}

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
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: clean build. The `hasBlockerEntries` method becomes unused -- delete it (lines 205-219).

- [ ] **Step 3: Remove unused hasBlockerEntries method**

Delete `hasBlockerEntries` from `debriefbus.go` (lines 205-219). Also remove the `threadentrykind` import if no longer used.

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 5: Commit**

```bash
git add business/domain/debriefbus/debriefbus.go
git commit -m "feat: expand task debrief to fire on every completion with importance rating"
```

---

### Task 4: Wire debriefBus into taskapp (bug fix)

**Files:**
- Modify: `app/domain/taskapp/taskapp.go`
- Modify: `app/domain/taskapp/route.go`

This is the core bug fix: the REST API update handler never triggers debriefs.

- [ ] **Step 1: Add debriefBus to taskapp struct**

In `app/domain/taskapp/taskapp.go`, add `debriefBus` to the `app` struct:

```go
type app struct {
	taskBus   *taskbus.Business
	threadBus *threadbus.Business
	debriefBus *debriefbus.Business
}
```

Add the import:

```go
"github.com/casebrophy/planner/business/domain/debriefbus"
```

- [ ] **Step 2: Fire OnTaskCompleted in update handler**

In `app/domain/taskapp/taskapp.go`, inside the `update` method, after the existing thread entry goroutine (after line 110), add a second goroutine that fires the debrief when a task is completed:

```go
	// Fire debrief on task completion
	if a.debriefBus != nil && but.Status != nil && *but.Status == taskstatus.Done {
		go func() {
			ct := debriefbus.CompletedTask{
				ID:          updated.ID,
				Title:       updated.Title,
				DurationMin: updated.DurationMin,
				CreatedAt:   updated.CreatedAt.Unix(),
				CompletedAt: time.Now().Unix(),
			}
			if err := a.debriefBus.OnTaskCompleted(context.Background(), ct); err != nil {
				// Logged but doesn't fail the response
				_ = err
			}
		}()
	}
```

Add the `"time"` import if not already present.

- [ ] **Step 3: Wire debriefBus in route.go**

In `app/domain/taskapp/route.go`, inside `Routes.Add()`, instantiate and inject debriefBus. Add after the `threadBus` line:

```go
	clarStore := clarificationdb.NewStore(cfg.Log, cfg.DB)
	clarBus := clarificationbus.NewBusiness(cfg.Log, clarStore)
	debriefBus := debriefbus.NewBusiness(cfg.Log, clarBus, threadBus)
```

Update the handler instantiation:

```go
	hdl := &app{taskBus: taskBus, threadBus: threadBus, debriefBus: debriefBus}
```

Add imports:

```go
"github.com/casebrophy/planner/business/domain/clarificationbus"
"github.com/casebrophy/planner/business/domain/clarificationbus/stores/clarificationdb"
"github.com/casebrophy/planner/business/domain/debriefbus"
```

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 5: Test manually**

Run: `make dev-up`
Complete a task via the frontend. Check that a `task_debrief` clarification card appears in the queue.

- [ ] **Step 6: Commit**

```bash
git add app/domain/taskapp/taskapp.go app/domain/taskapp/route.go
git commit -m "fix: wire debriefBus into taskapp so REST completions trigger debrief cards"
```

---

### Task 5: Add GenerateWeeklyReview to debriefbus

**Files:**
- Modify: `business/domain/debriefbus/debriefbus.go`

- [ ] **Step 1: Add GenerateWeeklyReview method**

Add to `business/domain/debriefbus/debriefbus.go`:

```go
// GenerateWeeklyReview creates a weekly impact review card from the given
// completed tasks. The caller (scheduler in main.go) queries and provides
// the task list. weekID is an ISO week string like "2026-W15" used for dedup.
func (b *Business) GenerateWeeklyReview(ctx context.Context, weekID string, tasks []CompletedTask) error {
	if len(tasks) == 0 {
		return nil
	}

	// Deterministic UUID from week string for dedup
	subjectID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("planner:weekly-review:"+weekID))

	kind := clarificationkind.WeeklyReview
	pending := clarificationstatus.Pending
	snoozed := clarificationstatus.Snoozed
	subjectType := "week"

	existingPending, err := b.clarificationBus.Count(ctx, clarificationbus.QueryFilter{
		Kind:        &kind,
		Status:      &pending,
		SubjectType: &subjectType,
		SubjectID:   &subjectID,
	})
	if err != nil {
		return fmt.Errorf("check existing weekly review: %w", err)
	}

	existingSnoozed, err := b.clarificationBus.Count(ctx, clarificationbus.QueryFilter{
		Kind:        &kind,
		Status:      &snoozed,
		SubjectType: &subjectType,
		SubjectID:   &subjectID,
	})
	if err != nil {
		return fmt.Errorf("check existing snoozed weekly review: %w", err)
	}

	if existingPending > 0 || existingSnoozed > 0 {
		return nil
	}

	// Build task list for answer options
	reviewTasks := make([]WeeklyReviewTask, len(tasks))
	for i, t := range tasks {
		reviewTasks[i] = WeeklyReviewTask{
			ID:    t.ID,
			Title: t.Title,
		}
	}

	optionsJSON, err := json.Marshal(map[string]any{
		"tasks": reviewTasks,
	})
	if err != nil {
		return fmt.Errorf("marshal weekly review options: %w", err)
	}

	question := fmt.Sprintf("You completed %d tasks this week. Which had the most impact?", len(tasks))

	if _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:          clarificationkind.WeeklyReview,
		SubjectType:   "week",
		SubjectID:     subjectID,
		Question:      question,
		AnswerOptions: json.RawMessage(optionsJSON),
		PriorityScore: 0.8,
	}); err != nil {
		return fmt.Errorf("create weekly review: %w", err)
	}

	return nil
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 3: Commit**

```bash
git add business/domain/debriefbus/debriefbus.go
git commit -m "feat: add GenerateWeeklyReview to debriefbus"
```

---

### Task 6: Wire weekly review scheduler in main.go

**Files:**
- Modify: `api/services/planner/main.go`

- [ ] **Step 1: Add weekly review scheduled job**

In `api/services/planner/main.go`, after the daily plan generation goroutine block (after the closing `}()` around line 449), add:

```go
	// Weekly impact review: runs once per week on Sunday at 18:00
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		var lastGenWeek string

		for {
			select {
			case <-jobCtx.Done():
				return
			case <-ticker.C:
				now := time.Now()
				year, week := now.ISOWeek()
				weekID := fmt.Sprintf("%d-W%02d", year, week)

				// Fire on Sunday at 18:00
				if now.Weekday() != time.Sunday || now.Format("15:04") != "18:00" || lastGenWeek == weekID {
					continue
				}
				lastGenWeek = weekID
				log.Info(jobCtx, "weekly-review", "msg", "generating weekly impact review", "week", weekID)

				// Query tasks completed in the past 7 days
				done := taskstatus.Done
				completedTasks, err := taskBus.Query(jobCtx, taskbus.QueryFilter{Status: &done}, taskbus.DefaultOrderBy, page.New(1, 200))
				if err != nil {
					log.Error(jobCtx, "weekly-review", "msg", "failed to fetch completed tasks", "error", err)
					continue
				}

				// Filter to tasks completed in the last 7 days
				cutoff := now.AddDate(0, 0, -7)
				var recentTasks []debriefbus.CompletedTask
				for _, t := range completedTasks {
					if t.CompletedAt != nil && t.CompletedAt.After(cutoff) {
						ct := debriefbus.CompletedTask{
							ID:    t.ID,
							Title: t.Title,
						}
						recentTasks = append(recentTasks, ct)
					}
				}

				if err := debriefBus.GenerateWeeklyReview(jobCtx, weekID, recentTasks); err != nil {
					log.Error(jobCtx, "weekly-review", "msg", "failed to generate weekly review", "error", err)
				}
			}
		}
	}()
```

Ensure `debriefBus` is instantiated earlier in main.go (it should already be wired for mcpapp). If not, add it near the other bus instantiations:

```go
debriefBus := debriefbus.NewBusiness(log, clarBus, threadBus)
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: clean build. Check that `debriefBus` variable is accessible in the goroutine scope (it should be, since it's declared in the outer function before the goroutine).

- [ ] **Step 3: Commit**

```bash
git add api/services/planner/main.go
git commit -m "feat: wire weekly impact review scheduler in main.go (Sunday 18:00)"
```

---

### Task 7: Add resolution side-effects for TaskDebrief and WeeklyReview

**Files:**
- Modify: `app/domain/clarificationapp/clarificationapp.go`

- [ ] **Step 1: Add TaskDebrief resolution**

In `app/domain/clarificationapp/clarificationapp.go`, find the `dispatchResolution` switch statement. There is currently no case for `clarificationkind.TaskDebrief`. Add one before the closing `}` of the switch (before the `case clarificationkind.EntityLink` block or after it):

```go
	case clarificationkind.TaskDebrief:
		var answer struct {
			Value string `json:"value"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil || answer.Value == "" || answer.Value == "skip" {
			return
		}
		obsData, _ := json.Marshal(map[string]string{
			"importance": answer.Value,
			"question":   item.Question,
		})
		if _, err := a.observationBus.Record(ctx, observationbus.NewObservation{
			SubjectType: item.SubjectType,
			SubjectID:   item.SubjectID,
			Kind:        observationkind.Debrief,
			Data:        json.RawMessage(obsData),
			Source:      "user",
			Confidence:  1.0,
			Weight:      2.0,
		}); err != nil {
			return
		}
```

- [ ] **Step 2: Add WeeklyReview resolution**

Add another case in the same switch:

```go
	case clarificationkind.WeeklyReview:
		var answer struct {
			SelectedTaskIDs []string `json:"selected_task_ids"`
		}
		if err := json.Unmarshal(*item.Answer, &answer); err != nil || len(answer.SelectedTaskIDs) == 0 {
			return
		}
		for _, taskIDStr := range answer.SelectedTaskIDs {
			taskID, err := uuid.Parse(taskIDStr)
			if err != nil {
				continue
			}
			obsData, _ := json.Marshal(map[string]string{
				"importance": "high",
				"source":     "weekly_review",
				"week":       item.SubjectID.String(),
			})
			if _, err := a.observationBus.Record(ctx, observationbus.NewObservation{
				SubjectType: "task",
				SubjectID:   taskID,
				Kind:        observationkind.Debrief,
				Data:        json.RawMessage(obsData),
				Source:      "user",
				Confidence:  1.0,
				Weight:      3.0,
			}); err != nil {
				continue
			}
		}
```

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 4: Commit**

```bash
git add app/domain/clarificationapp/clarificationapp.go
git commit -m "feat: add TaskDebrief and WeeklyReview resolution side-effects"
```

---

### Task 8: Frontend -- update enums and regenerate TypeScript types

**Files:**
- Modify: `api/services/frontend/web/src/types/enums.ts`
- Regenerate: `api/services/frontend/web/src/types/generated/clarification-kind.ts`

- [ ] **Step 1: Regenerate TypeScript clarification kind types**

Run: `go run api/tooling/gen-ts-kinds/main.go`
Expected: `api/services/frontend/web/src/types/generated/clarification-kind.ts` now includes `"weekly_review"`.

- [ ] **Step 2: Add WeeklyReview to enums.ts labels and colors**

In `api/services/frontend/web/src/types/enums.ts`, add to `ClarificationKindLabels`:

```typescript
[ClarificationKind.WeeklyReview]: 'Weekly Review',
```

Add to `ClarificationKindColors`:

```typescript
[ClarificationKind.WeeklyReview]: '#f59e0b',
```

- [ ] **Step 3: Verify frontend build**

Run: `make frontend-build`
Expected: clean build. TypeScript will error if the exhaustive Record types are missing the new kind.

- [ ] **Step 4: Commit**

```bash
git add api/services/frontend/web/src/types/enums.ts api/services/frontend/web/src/types/generated/clarification-kind.ts
git commit -m "feat: add WeeklyReview to frontend enums and regenerate types"
```

---

### Task 9: Frontend -- redesign TaskDebrief and add WeeklyReview card UI

**Files:**
- Modify: `api/services/frontend/web/src/components/clarifications/ClarificationCard.vue`

- [ ] **Step 1: Redesign TaskDebrief from textarea to button options**

In `ClarificationCard.vue`, replace the Task Debrief section (lines 322-340) with:

```vue
      <!-- Task Debrief (importance rating) -->
      <div
        v-else-if="item.kind === ClarificationKind.TaskDebrief"
        class="flex flex-col gap-2"
      >
        <div class="grid grid-cols-2 gap-2">
          <button
            v-for="opt in (Array.isArray(options) ? options as Array<{label: string, value: string}> : [])"
            :key="opt.value"
            :class="[
              'px-4 py-2.5 text-sm font-medium text-white rounded-lg transition-colors',
              opt.value === 'high' ? 'bg-emerald-600 hover:bg-emerald-500' :
              opt.value === 'medium' ? 'bg-blue-600 hover:bg-blue-500' :
              opt.value === 'low' ? 'bg-amber-600 hover:bg-amber-500' :
              opt.value === 'waste' ? 'bg-red-600 hover:bg-red-500' :
              'bg-gray-600 hover:bg-gray-500 col-span-2'
            ]"
            @click="resolveWithValue({ value: opt.value })"
          >
            {{ opt.label }}
          </button>
        </div>
      </div>
```

- [ ] **Step 2: Add WeeklyReview card UI**

Add a new section before the fallback `v-else` block:

```vue
      <!-- Weekly Review (multi-select tasks) -->
      <div
        v-else-if="item.kind === ClarificationKind.WeeklyReview"
        class="flex flex-col gap-2"
      >
        <p class="text-sm text-gray-400 mb-1">
          Select the tasks that had the most impact:
        </p>
        <button
          v-for="task in ((options as {tasks?: Array<{id: string, title: string}>})?.tasks ?? [])"
          :key="task.id"
          :class="[
            'w-full px-4 py-2.5 text-sm font-medium text-left rounded-lg transition-colors border',
            selectedWeeklyTasks.has(task.id)
              ? 'bg-emerald-600/20 border-emerald-500 text-emerald-300'
              : 'bg-gray-700 border-gray-600 text-gray-100 hover:border-gray-500'
          ]"
          @click="toggleWeeklyTask(task.id)"
        >
          {{ task.title }}
        </button>
        <button
          :disabled="selectedWeeklyTasks.size === 0"
          class="w-full px-4 py-2.5 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-500 rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed mt-1"
          @click="resolveWithValue({ selected_task_ids: [...selectedWeeklyTasks] })"
        >
          Submit ({{ selectedWeeklyTasks.size }} selected)
        </button>
      </div>
```

- [ ] **Step 3: Add reactive state and toggle function**

In the `<script setup>` section, add after the existing refs:

```typescript
const selectedWeeklyTasks = ref(new Set<string>())

function toggleWeeklyTask(id: string) {
  const next = new Set(selectedWeeklyTasks.value)
  if (next.has(id)) {
    next.delete(id)
  } else {
    next.add(id)
  }
  selectedWeeklyTasks.value = next
}
```

- [ ] **Step 4: Verify frontend build**

Run: `make frontend-build`
Expected: clean build.

- [ ] **Step 5: Commit**

```bash
git add api/services/frontend/web/src/components/clarifications/ClarificationCard.vue
git commit -m "feat: redesign TaskDebrief card as importance buttons, add WeeklyReview multi-select card"
```

---

### Task 10: Integration test -- verify end-to-end

- [ ] **Step 1: Run full test suite**

Run: `make test`
Expected: all existing tests pass. No new test files needed -- the debrief changes are behavioral (trigger always vs. conditionally) and the weekly review uses existing clarification infrastructure.

- [ ] **Step 2: Manual smoke test**

Run: `make dev-up`

1. Complete a task from the frontend daily planner
2. Navigate to clarification queue -- should see a "How important was this?" card
3. Tap "High impact" -- card resolves
4. Check that an outcome observation was recorded (via API: `GET /api/v1/observations?subject_type=task&subject_id=<id>`)

- [ ] **Step 3: Verify frontend build clean**

Run: `make frontend-build`
Expected: clean build.

- [ ] **Step 4: Final commit if any cleanup needed**

```bash
git add -A
git commit -m "chore: integration test cleanup"
```
