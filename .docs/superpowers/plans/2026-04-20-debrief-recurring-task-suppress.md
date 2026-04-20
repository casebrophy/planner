# Suppress TaskDebrief for recurring tasks + one-time cleanup

**Date:** 2026-04-20  
**Feature:** Recurring task debrief throttle + admin CLI dedupe

## Overview

Recurring tasks generate a `task_debrief` clarification card on every completion. Each instance is a new task with a new ID, so the existing per-task-ID dedup never triggers. The queue fills with duplicates (same context, same question, slightly different task IDs), and most cards lack actionable context. This feature adds a 30-day throttle at creation time: if a prior task_debrief for any sibling of the same recurrence parent was created within the window, skip the new card. A one-time admin CLI command (`debrief-dedupe`) dismisses all existing pending/snoozed duplicates for recurring tasks.

## Design decisions

1. **Throttle window is 30 days, configurable.** `TaskDebriefThrottle` field on `debriefbus.Business` defaults to 720 hours (30 days). Throttle is on *creation*, not completion — once a recurring task completes outside the window, the next debrief fires regardless of intermediate completions.
2. **Match on `RecurrenceParentID`, not subject_id.** Each instance of a recurring task has a different ID and is a separate subject. To find siblings, query the recurrence parent ID. `CompletedTask` carries both the task ID and its parent ID so `OnTaskCompleted` can look up the window without a taskbus dependency.
3. **Clarification query needs `CreatedSince` filter.** The store layer can build a WHERE clause on `created_at >= now - duration`. We add this filter if missing; check first to avoid duplication.
4. **Admin CLI is read-or-dismiss, not smart-merge.** The debrief-dedupe command finds recurring task debrief cards and dismisses ALL but the most recent one per unique task parent. This is safe: pending cards are never answered yet. If a card was already answered, it remains (only pending/snoozed are touched).
5. **No new migration.** No schema changes — just a filter condition + a business-layer guard.

## Files to Touch

| Action | File | Layer | Why |
|--------|------|-------|-----|
| MODIFY | `business/domain/debriefbus/model.go` | business | add `RecurrenceRule *string` and `RecurrenceParentID *uuid.UUID` to `CompletedTask` |
| MODIFY | `business/domain/debriefbus/debriefbus.go` | business | in `OnTaskCompleted`, check throttle window before creating; add `TaskDebriefThrottle` field to Business struct |
| MODIFY | `app/domain/taskapp/taskapp.go` | app | populate `CompletedTask.RecurrenceRule` and `RecurrenceParentID` when firing debrief |
| MODIFY | `app/domain/mcpapp/mcpapp.go` | app | populate both fields at both `OnTaskCompleted` call sites (lines ~531, ~592) |
| MODIFY | `business/domain/clarificationbus/filter.go` + `stores/clarificationdb/filter.go` | business+store | add `CreatedSince *time.Time` to QueryFilter if not present; audit first |
| CREATE | `api/tooling/admin/commands/debriefdedupe.go` | tool | new admin subcommand that dismisses duplicate recurring task debriefs |
| MODIFY | `api/tooling/admin/main.go` | tool | register `debrief-dedupe` command |
| CREATE | `business/domain/debriefbus/debriefbus_recurring_test.go` | test | table-driven: non-recurring creates; recurring within window skips; recurring outside window creates |
| CREATE | `api/tooling/admin/commands/debriefdedupe_test.go` | test | dry-run validation; real run dismisses; non-recurring untouched |

## Tasks

### Task 1: Extend CompletedTask with recurrence info

- [ ] Add `RecurrenceRule *string` and `RecurrenceParentID *uuid.UUID` to `CompletedTask` struct (debriefbus/model.go:8)
- [ ] Update caller at `app/domain/taskapp/taskapp.go:187` to populate both fields from the task payload
- [ ] Update both caller sites in `app/domain/mcpapp/mcpapp.go` (~531, ~592) to populate both fields
- [ ] Run `go build ./...` — should compile clean
- [ ] Run existing debriefbus tests (`make test business/domain/debriefbus`); must still pass

### Task 2: Add 30-day throttle for recurring tasks

- [ ] Add `TaskDebriefThrottle time.Duration` field to `debriefbus.Business` struct (debriefbus.go)
- [ ] In `NewBusiness`, default it to `720 * time.Hour` (30 days)
- [ ] In `OnTaskCompleted`, after checking per-task pending/snoozed count (line ~42-64):
  - If `ct.RecurrenceRule == nil`, keep existing behavior
  - If recurring, query clarificationBus for task_debrief cards created in the last `TaskDebriefThrottle` matching ANY subject_id from the recurrence parent (see step 3)
  - Use new `CreatedSince` filter (see Task 3) to find recent cards
  - If any found, log info "skip task_debrief: recurring task within throttle window" and return
- [ ] Compile + test (`make test business/domain/debriefbus`)

### Task 3: Add CreatedSince filter to clarification queries

- [ ] Read `business/domain/clarificationbus/filter.go` and `business/domain/clarificationbus/stores/clarificationdb/filter.go` to check if `CreatedSince *time.Time` already exists
- [ ] If missing, add to `QueryFilter` struct in both files
- [ ] In the store's `applyFilter`, add a WHERE clause: `AND created_at >= $N` if `CreatedSince != nil`
- [ ] Compile + test

### Task 4: Admin CLI `debrief-dedupe`

- [ ] Create `api/tooling/admin/commands/debriefdedupe.go` following the structure of `gapbackfill.go`:
  - Flags: `--dry-run` (print, don't modify), `--limit` (default 1000, hard cap 5000)
  - Logic: SELECT clarification_items with kind='task_debrief', status IN ('pending','snoozed'), JOIN tasks WHERE recurrence_rule IS NOT NULL
  - For each group (by recurrence_parent_id or task title), keep the most recent, dismiss others
  - Dry-run prints summary by parent ID; real run counts dismissed + kept
- [ ] Register in `api/tooling/admin/main.go` (add to the commands map or switch)
- [ ] Test with `make test api/tooling/admin/commands` and manual `make admin ARGS="debrief-dedupe --dry-run"` 

### Task 5: Tests

- [ ] Create `business/domain/debriefbus/debriefbus_recurring_test.go` with table-driven tests:
  - Non-recurring task always creates a debrief (baseline)
  - Recurring task within window skips debrief
  - Recurring task outside window (after throttle expires) creates debrief
  - Throttle window edge cases (exactly at boundary)
- [ ] Create `api/tooling/admin/commands/debriefdedupe_test.go`:
  - Dry-run prints count but does not modify (verify via SELECT after)
  - Real run dismisses all but most-recent per parent
  - Non-recurring debrief cards are untouched (remain pending)
  - Empty result (no recurring) is handled gracefully

## Cascade / Verification Checklist

- [ ] `make test` passes (all suites)
- [ ] `make lint` passes
- [ ] Update `.docs/arch/debrief-backend.md` if it exists (add "Throttle for recurring" section under Trigger logic); or create minimal arch stub
- [ ] Manual smoke test: create recurring task, complete twice within 30 days, verify only first completion fires debrief
- [ ] Manual CLI test: `make admin ARGS="debrief-dedupe --dry-run"` shows count; `make admin ARGS="debrief-dedupe"` dismisses

## Gotchas

- `CompletedTask` is also used by `debriefbus.GenerateWeeklyReview` path (see previous feature). New fields are optional pointers (`*string`, `*uuid.UUID`), so WeeklyReview is not broken — it just passes `nil`.
- Admin command must wire debriefBus + taskBus + clarificationBus correctly; follow `gapbackfill.go` dependency injection pattern exactly.
- Throttle query must handle the case where `RecurrenceParentID` is nil (non-recurring tasks). Check `ct.RecurrenceParentID != nil` before querying.
- CreatedSince filter defaults to `nil` (no filter) — backward compatible with existing queries.

## Non-goals

- Not changing the 30-day window to be per-user-configurable (hardcoded default is fine; make it a Business field if context later demands it).
- Not adding a UI affordance to hide or batch-manage duplicates (admin CLI is sufficient).
- Not resolving duplicate debriefs for *completed* recurring tasks (only throttles new ones).
- Not touching `ContextDebrief`, `WeeklyReview`, or any other clarification kind.
