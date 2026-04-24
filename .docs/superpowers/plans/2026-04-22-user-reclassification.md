# User Reclassification (Task ↔ Note)

**Goal:** let the user fix classification mistakes after ingestion by converting a task to a note or a note to a task — in-place from the detail view, preserving context, tags, and raw-input provenance. Log every conversion to `classificationcorrectionbus` so the ingest prompt-tuning loop can use corrections as training signal.

**Out of scope (deferred):** event↔task, event↔note, note↔event. Events carry a required timestamp that either has to be dropped (lossy) or solicited (needs a UI flow). We'll plan that separately once the task↔note path is working and we see whether the simple conversion covers most real misclassifications.

**Related:** `.docs/superpowers/plans/2026-04-22-ingestbus-classification-improvements.md` — that plan improves at-ingest accuracy; this plan handles post-hoc correction. Both write to `classificationcorrectionbus`; once this lands, Track 2 of that plan can finally use `correction_applied` rows as few-shot prompt signal.

---

## What exists already

- **`classificationcorrectionbus`** (`business/domain/classificationcorrectionbus/`) — already has `NewCorrection{ClauseText, PredictedType, Confidence, ActualType, Source}` with `Source` supporting `"correction_applied"`. Zero schema work needed. Just call `Record(ctx, nc)`.
- **`taskbus.Delete` / `notebus.Delete`** — assume standard CRUD exists; verify on implementation.
- **FK surface** (from `business/sdk/migrate/sql/migrate.sql`):
  - `task_tags`, `task_dependencies`, `daily_plan_entries`, `outcome_observations` all `ON DELETE CASCADE` from tasks
  - `notes.task_id` → `tasks(task_id)` `ON DELETE SET NULL`
  - `tasks.recurrence_parent_id` → `tasks(task_id)` **no ON DELETE** — deleting a recurrence parent with live children will error
  - `note_tags` `ON DELETE CASCADE` from notes

---

## Conversion semantics

### Task → Note

| Task field | Note field | Notes |
|------------|------------|-------|
| `Title` + `Description` | `Content` | `Content = Title` if Description is empty, else `Title + "\n\n" + Description` |
| `ContextID` | `ContextID` | Preserve |
| `RawInputID` | `RawInputID` | Preserve — this is the key provenance link |
| `ID` | (new UUID) | Note gets a new ID; task row is deleted |
| (none) | `Source` | Set to `"reclassified_from_task"` |
| (none) | `TaskID` | `null` — this note IS the task's successor, not an attachment |
| `Unconfirmed` | `Unconfirmed` | Preserve |

**Tag migration:** read `task_tags` for `task_id`, insert corresponding `note_tags` for the new `note_id` in the same tx, then cascade-delete the task.

**Refuse conversion if:**
- Task has `RecurrenceRule != nil` or has live recurrence children (`SELECT 1 FROM tasks WHERE recurrence_parent_id = $1 LIMIT 1`) — recurring tasks aren't notes.
- Task has dependents (`task_dependencies.depends_on_id = $1`) — would silently break a dependency graph. Preflight-check and return `errs.InvalidArgument` with a message naming the dependents. (Don't silently cascade-delete the rows.)
- Task is in `daily_plan_entries` for today or future — return 409 conflict, force the user to remove from plan first.

**Correction record:** `ClauseText` = task title (fall back to description if title is empty), `PredictedType = "task"`, `Confidence = 0` (we don't have the original extraction confidence post-hoc), `ActualType = "note"`, `Source = "correction_applied"`.

### Note → Task

| Note field | Task field | Notes |
|------------|------------|-------|
| `Content` (first line) | `Title` | Split on `\n`; trim |
| `Content` (remainder) | `Description` | Empty if Content is single-line |
| `ContextID` | `ContextID` | Preserve |
| `RawInputID` | `RawInputID` | Preserve |
| (none) | `Status` | `taskstatus.Pending` (verify constant name) |
| (none) | `Priority` | `taskpriority.Medium` |
| (none) | `Energy` | `taskenergy.Medium` (or default) |
| `Unconfirmed` | `Unconfirmed` | Force `true` — user should review the generated title/description |

**Orphan check:** note may be referenced by other notes via `notes.task_id`? No — `task_id` points **to** a task, and there's no reverse FK on notes. Only `note_tags` needs migration. Clean.

**Correction record:** mirror of above with `PredictedType="note"`, `ActualType="task"`.

---

## Architecture decision: where do the methods live?

Two options:

1. **Add `ConvertToNote` on taskbus, `ConvertToTask` on notebus.** Each bus owns its own conversion. Cross-bus call to create the target entity. Pro: locality — you read `taskbus.go` to see every task-lifecycle op. Con: taskbus now depends on notebus + classificationcorrectionbus, and vice versa — risks an import cycle and bloats both buses.

2. **New `reclassifybus` under `business/domain/reclassifybus/`** that orchestrates the transaction across taskbus, notebus, and classificationcorrectionbus. Pro: no cycle, single place for the conversion logic + future task↔event support. Con: one more domain; the individual buses need enough exported surface (tag reads, raw CRUD) for the orchestrator to work.

**Recommendation: option 2.** The conversion is already a cross-domain transaction with correction logging — that's domain logic in its own right, not a lifecycle concern of either taskbus or notebus. The extra domain is cheap; the import cycle risk of option 1 isn't.

`reclassifybus` exposes:

```go
type Bus struct {
    log   *logger.Logger
    task  *taskbus.Business
    note  *notebus.Business
    corr  *classificationcorrectionbus.Business
    sqldb *sqlx.DB  // for the transaction boundary + tag copy
}

func (b *Bus) TaskToNote(ctx context.Context, taskID uuid.UUID) (notebus.Note, error)
func (b *Bus) NoteToTask(ctx context.Context, noteID uuid.UUID) (taskbus.Task, error)
```

Both methods run in a single DB transaction (need `sqldb.Beginx` + pass the tx into the sub-bus calls — verify taskbus/notebus store methods accept a tx, or add tx-variants if not). Correction record is written **in the same tx** so we don't drift.

No Storer interface inside reclassifybus — it's a pure orchestrator, no SQL of its own except the tag copy and the preflight checks.

---

## App layer

Two endpoints in a new `app/domain/reclassifyapp/`:

```
POST /api/v1/tasks/{id}/convert-to-note   → returns the new Note
POST /api/v1/notes/{id}/convert-to-task   → returns the new Task
```

No request body. Auth middleware as usual via `Routes.Add()`.

Response is the newly-created entity (full DTO) so the frontend can replace the view state without a separate fetch.

Error mapping:
- `errs.NotFound` — source entity missing
- `errs.InvalidArgument` — recurrence / dependent blocking
- `errs.FailedPrecondition` (or equivalent) — scheduled in daily plan

---

## Frontend

`web/src/views/TaskDetailView.vue`:

- Add a menu item / button: **"Convert to note"** (overflow menu or right sidebar, wherever destructive/lifecycle actions live today — follow existing pattern).
- On click: confirm dialog ("This will convert the task to a note. Tags and context will be preserved. Continue?"). On confirm, call service, route to the new note's detail view.

`web/src/views/NoteDetailView.vue`:

- **"Convert to task"** — same UX. Confirmation explains that status/priority will default and the note content will be split into title + description.
- Route to the new task's detail view.

`web/src/services/`:
- Add `convertTaskToNote(taskId): Promise<Note>` to the task service.
- Add `convertNoteToTask(noteId): Promise<Task>` to the note service.

`web/src/stores/`:
- Task store: on successful conversion, remove the task from cache. Bump a "reclassified" counter or emit an event if other views need to react (optional — skip unless there's an obvious subscriber).
- Note store: same, inverse.

Error toasts for each error class (not-found / recurring / has-dependents / in-plan).

Preserve the context filter — if the user was viewing tasks within context X, they should land on the note in context X.

---

## Files to touch

### Backend — new

| Action | File | Why |
|--------|------|-----|
| CREATE | `business/domain/reclassifybus/reclassifybus.go` | Bus struct, `TaskToNote`, `NoteToTask`, preflight checks |
| CREATE | `business/domain/reclassifybus/model.go` | Result types if needed (may be empty; just returns taskbus/notebus types) |
| CREATE | `business/domain/reclassifybus/reclassifybus_test.go` | dbtest: both conversions happy-path, refuse-recurring, refuse-with-dependents, tag migration |
| CREATE | `app/domain/reclassifyapp/reclassifyapp.go` | Two handlers |
| CREATE | `app/domain/reclassifyapp/route.go` | Register routes with auth middleware |
| CREATE | `app/domain/reclassifyapp/reclassifyapp_test.go` | apitest: 200 paths, 400/404/409 error paths |
| MODIFY | `api/services/planner/main.go` | Wire `reclassifybus.NewBusiness(...)` and register `reclassifyapp.Routes.Add()` |
| CREATE | `.docs/arch/reclassify-backend.md` | Follow existing arch-doc template |

### Backend — modify (only if needed)

| Action | File | Why |
|--------|------|-----|
| POTENTIALLY MODIFY | `business/domain/taskbus/taskbus.go` | If current Delete signature doesn't accept a tx, add a tx-variant OR expose the storer directly to reclassifybus |
| POTENTIALLY MODIFY | `business/domain/notebus/notebus.go` | Same |

This is the one "discover on implementation" item — check how existing cross-bus transactional code handles it and copy that pattern. (Grep for other places where a bus calls `sqldb.Beginx` and coordinates writes across domains.)

### Frontend

| Action | File | Why |
|--------|------|-----|
| MODIFY | `web/src/services/taskService.ts` (or equivalent) | Add `convertTaskToNote` |
| MODIFY | `web/src/services/noteService.ts` | Add `convertNoteToTask` |
| MODIFY | `web/src/views/TaskDetailView.vue` | Menu item + confirm dialog + route transition |
| MODIFY | `web/src/views/NoteDetailView.vue` | Same, inverse |
| MODIFY | `web/src/stores/taskStore.ts` | Remove-from-cache action |
| MODIFY | `web/src/stores/noteStore.ts` | Same |
| CREATE | `web/src/views/__tests__/TaskDetailView.reclassify.test.ts` | Confirm + service-call + navigation |
| CREATE | `web/src/views/__tests__/NoteDetailView.reclassify.test.ts` | Same |
| CREATE | `.docs/arch/reclassify-frontend.md` | Arch doc |

---

## Test plan

**Store (dbtest):**
- `TestTaskToNote_HappyPath` — task with context + tags + raw_input → note with same context, tags copied, raw_input preserved, task row gone, `note_tags` row exists, correction logged.
- `TestTaskToNote_RefusesRecurring` — task with `recurrence_rule` set → `InvalidArgument`, task still exists, no correction logged.
- `TestTaskToNote_RefusesRecurrenceParent` — seed a recurrence parent + child → conversion of parent fails.
- `TestTaskToNote_RefusesWithDependents` — task_dependencies row pointing at this task → `InvalidArgument` with dependent IDs in the error.
- `TestTaskToNote_RefusesWhenInTodaysPlan` — daily_plan_entries row → conflict.
- `TestNoteToTask_HappyPath` — note → task with title=first line, description=rest, tags migrated, correction logged.
- `TestNoteToTask_SingleLineContent` — content has no newline → title=content, description="".
- `TestConversion_TxRollback` — induce failure in correction-record write → both original entity and new entity absent (tx rolled back cleanly).

**API (apitest):**
- 200 on both conversions, response body shape matches Task / Note DTO.
- 404 on unknown id.
- 400 on recurring / dependent task.
- 401 without API key.

**Frontend (Vitest):**
- Click action → confirm dialog appears.
- Confirm → service called with correct id → navigation fires on resolve.
- Cancel → no service call.
- Service error → toast, no navigation.

---

## Gotchas

- **Transaction scope.** Correction record **must** be written in the same tx as the entity swap — otherwise a DB hiccup between the two writes leaves us with either an orphaned correction or (worse) a completed conversion with no record of why. Check how other cross-bus transactions are handled in this repo before writing new tx-handling.
- **Recurrence parent FK has no `ON DELETE`.** Preflight must explicitly check for children before calling `Delete`, or the tx will fail with a FK violation late in the process. Fail fast.
- **`task_dependencies` cascades.** Don't rely on the cascade — detect dependents in preflight and refuse. Silent cascade would hide the user's intent ("did you mean to wipe the blocker graph?").
- **`notes.task_id` is `ON DELETE SET NULL`.** If we delete a task that has notes attached (e.g. journal entries on that task), those notes will have their `task_id` nulled. That's probably fine — they become orphaned notes — but surface this in the confirm dialog copy ("3 notes are currently linked to this task; they will remain as standalone notes").
- **Confidence field is `float64`, not `*float64`.** Post-hoc correction has no ingest confidence; use `0`. Don't pretend otherwise. (Consider making it nullable in a future migration, but not in this plan's scope.)
- **Don't update `classificationcorrectionbus.Source` enum** — `"correction_applied"` already exists per the existing test (`Test_ClassificationCorrection`). Reuse it.
- **Frontend route transition.** On conversion, the source view must navigate away **before** the store removes the old entity from cache, otherwise the view unmount triggers a "not found" flash. Common Vue bug — follow existing delete-and-navigate patterns in the codebase.
- **DB port 5433 in local dev.**

---

## Sequencing / beads

Proposed issues:

1. **Parent (feature):** "User reclassification (task ↔ note)"
2. **Child:** "reclassifybus: backend conversion logic + tests" — the store + bus + dbtest layer, no app/frontend yet
3. **Child:** "reclassifyapp: HTTP endpoints + apitest" — depends on #2
4. **Child:** "Frontend: convert actions on TaskDetailView / NoteDetailView" — depends on #3
5. **Child:** "Arch docs: reclassify-backend.md + reclassify-frontend.md" — depends on #4 (write once at end, project convention)

Each child's `--context` links this plan file + its relevant heading.

---

## Feedback loop back to ingest prompt tuning

Once this lands, the ingest plan's Track 2 can pull the last N `classificationcorrectionbus` rows with `Source = "correction_applied"` as rotating few-shot examples in `buildGenericTextExtractionPrompt`. Out of scope for this plan — but the whole point of logging corrections is to feed this loop, so leave a TODO comment in `reclassifybus.go` pointing at that plan.

---

## Open questions

- **Should the old entity be soft-deleted instead of hard-deleted?** Leaves a breadcrumb for "where did that task go", costs schema complexity. Decision: hard delete for now — the correction record is the breadcrumb. Revisit if users report losing work.
- **Should we preserve `created_at` on the new entity?** Probably yes — this is a reclassification, not a new capture. Add a `PreserveCreatedAt` path to the store Create if not already available.
- **UI for bulk reclassification** (select N tasks → convert all to notes) — out of scope. One at a time.
