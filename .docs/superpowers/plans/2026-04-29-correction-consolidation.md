# Correction Consolidation

**Goal:** collapse `correctionapp` (6-path conversion logic in an HTTP handler) and `reclassifybus` (task↔note conversion bus with preflight) into a single `correctionbus` that owns all 6 conversion paths, preflight gates, and audit-row recording. Reduce the API surface to one route and the frontend surface to one service method. Pick canonical semantics so users can't accidentally bypass safety by hitting the wrong endpoint.

**Why now:** the two paths drifted in safety. `POST /api/v1/corrections {item_type:"task", new_type:"note"}` silently converts recurring tasks; `POST /api/v1/tasks/{id}/convert-to-note` refuses them. Same intent, two doors, different rules. `correctionapp.go` is also ~320 lines of conversion logic in an HTTP handler — a direct violation of the project's three-layer rule.

**Out of scope:**
- New entity types (clarifications, transactions, etc.). The matrix stays at 6 paths for now.
- Event-source preflight gates (no-op by design — revisit if scheduled events convert and break daily plans).
- Refactor of `classificationcorrectionbus` — it's the audit-log persistence and stays untouched.

**Closes:** `planner-ow7c` (consolidation tracker), `planner-hurd` (RawInputID drop — already fixed by `planner-bztz` in commit `569940f8`; consolidation absorbs).

---

## Canonical semantics (the rules the new bus encodes)

| Concern | Decision |
|---|---|
| Content separator (task↔note, event↔note) | `"\n\n"` (reclassifybus's value, not correctionapp's `": "`) |
| `RawInputID` | preserved from source |
| `CreatedAt` | preserved from source |
| `UpdatedAt` | reset to `now` |
| `Unconfirmed` | forced `false` (user-initiated correction = confirmed) |
| Tags (task↔note) | copied via `INSERT ... SELECT ... ON CONFLICT DO NOTHING` |
| Tags (any path involving event) | not copied — no `event_tags` table |
| Synthetic times for `*→event` | `StartsAt = now+1h`, `EndsAt = now+2h` (matches current correctionapp) |
| Preflight (source = task) | refuse if recurring, has recurrence children, has dependents, scheduled in today's/future daily plan |
| Preflight (source = note) | none |
| Preflight (source = event) | none |
| Audit `Source` value | `"correction_applied"` for all 6 paths |
| New entity `Source` (notes) | `"correction"` (notes only — already enabled by Migration 1.46) |

---

## Architecture

### `business/domain/correctionbus/`

The new bus composes the four domain buses and owns the transaction. Internal structure: a flat dispatch map keyed by `(srcType, dstType)`, six small inline conversion functions. Not a registry/interface — that's overengineering for a 6-path matrix that won't grow.

```go
type Bus struct {
    log   *logger.Logger
    task  *taskbus.Business
    note  *notebus.Business
    event *eventbus.Business
    corr  *classificationcorrectionbus.Business
    db    *sqlx.DB
}

func NewBusiness(log, taskBus, noteBus, eventBus, corrBus, db) *Bus

// Single public entry point.
type Direction struct {
    From string  // "task" | "note" | "event"
    To   string  // "task" | "note" | "event"
}

type ConversionResult struct {
    ID   uuid.UUID
    Type string  // matches Direction.To
}

func (b *Bus) Convert(ctx context.Context, itemID uuid.UUID, dir Direction) (ConversionResult, error)
```

Internally `Convert`:
1. Validates `dir.From != dir.To` and both are in `{task, note, event}` (returns sentinel `ErrInvalidDirection`).
2. Fetches the source entity outside the tx; returns `ErrSourceNotFound` if missing (wraps `sqldb.ErrDBNotFound`).
3. If `dir.From == "task"` runs preflight — returns sentinel errors `ErrTaskRecurring`, `ErrTaskHasDependents`, `ErrTaskHasRecurrenceChildren`, `ErrTaskScheduled`.
4. `BeginTxx`, `defer tx.Rollback()`.
5. Looks up the conversion func from `map[Direction]conversionFunc` and runs it. Each func builds the target entity inline (preserving `RawInputID`/`CreatedAt`, resetting `UpdatedAt`, forcing `Unconfirmed=false`), calls `targetBus.CreateWithTx`, copies tags if applicable, calls `corrBus.RecordWithTx`, calls `sourceBus.DeleteWithTx`.
6. `tx.Commit()`.
7. Returns `ConversionResult{ID: newID, Type: dir.To}`.

Sentinel errors live in the bus (`var ErrTaskRecurring = errors.New("...")`) so the app layer maps them to HTTP codes via `errors.Is` rather than fragile string matching like `reclassifyapp.mapError` does today.

### `app/domain/correctionapp/` (slimmed)

```go
func (a *app) correct(ctx, r) (web.Encoder, error) {
    var body CorrectionRequest
    if err := web.Decode(r, &body); err != nil { return nil, errs.New(errs.InvalidArgument, err) }

    itemID, err := uuid.Parse(body.ItemID)
    if err != nil { return nil, errs.New(errs.InvalidArgument, err) }

    res, err := a.correctionBus.Convert(ctx, itemID, correctionbus.Direction{From: body.ItemType, To: body.NewType})
    if err != nil { return nil, mapError(err) }

    return CorrectionResult{ID: res.ID.String(), Type: res.Type}, nil
}
```

`mapError` uses `errors.Is`:
- `ErrInvalidDirection` → 400
- `ErrSourceNotFound` → 404
- `ErrTaskRecurring`, `ErrTaskHasDependents`, `ErrTaskHasRecurrenceChildren` → 400
- `ErrTaskScheduled` → 409 (matches existing reclassifyapp behavior)
- default → 500

The handler stays under ~80 lines including helpers. All 6 conversion arms and the inline `CreateWithTx`/`DeleteWithTx`/`RecordWithTx` calls move to the bus. Helper functions (`truncate`, `copyTaskTagsToNoteTags`, `copyNoteTagsToTaskTags`) move to the bus.

### Deletions

- `business/domain/reclassifybus/` (entire package)
- `app/domain/reclassifyapp/` (entire package)
- Routes `POST /api/v1/tasks/{id}/convert-to-note` and `POST /api/v1/notes/{id}/convert-to-task`
- `taskService.convertTaskToNote` (frontend)
- `noteService.convertNoteToTask` (frontend)
- `.docs/arch/reclassify-backend.md` and `.docs/arch/reclassify-frontend.md` (content folded into correction docs)

---

## Pre-flight verification

Before writing the new bus, verify `eventbus` exposes both `CreateWithTx(ctx, tx sqlx.ExtContext, event)` and `DeleteWithTx(ctx, tx sqlx.ExtContext, event)`. The current `correctionapp` uses these via `eventBus`. If they don't exist, the bus consolidation is blocked until they're added. **Action:** grep `business/domain/eventbus/eventbus.go` for both signatures before starting Phase 1.

---

## Implementation phases

The work is sequenced to keep the build green at every commit. Each phase compiles, tests pass, and the user can pause between phases without a broken tree.

### Phase 1 — Build `correctionbus` next to existing code

1. Create `business/domain/correctionbus/` with:
   - `correctionbus.go` — `Bus`, `NewBusiness`, `Convert`, the dispatch map, sentinel errors, six conversion funcs (`taskToNote`, `taskToEvent`, `noteToTask`, `noteToEvent`, `eventToTask`, `eventToNote`), `preflightTask` (recurring/children/dependents/daily-plan SQL ported verbatim from `reclassifybus.preflightTaskToNote`), tag-copy helpers (`copyTaskTagsToNoteTags`, `copyNoteTagsToTaskTags`).
   - `model.go` — `Direction`, `ConversionResult`. (Kept minimal; bus can grow these as needed.)
   - `correctionbus_test.go` — port `reclassifybus_test.go`'s 5 cases, expand to all 6 directions (12 tests: 6 happy + 4 task preflight refusals + invalid direction + source not found).

2. Add `Correction *correctionbus.Bus` to `business/sdk/dbtest.BusDomain` and wire it in the constructor (`business/sdk/dbtest/business.go` or wherever `newBusDomains` lives). **Keep `Reclassify` for now** so existing tests don't break.

3. Run `make test ./business/domain/correctionbus/...` — green.

**Phase 1 commit:** `feat(correctionbus): add unified bus for entity-type corrections`

### Phase 2 — Migrate `correctionapp` to use `correctionbus`

1. Modify `app/domain/correctionapp/route.go`:
   - Build `correctionbus` (which itself composes task/note/event/corr buses + db).
   - Replace `app{db, taskBus, noteBus, eventBus, correctionBus}` with `app{correctionBus *correctionbus.Bus}`.

2. Modify `app/domain/correctionapp/correctionapp.go`:
   - Strip the 6-arm switch, helper funcs, all direct CreateWithTx/DeleteWithTx/RecordWithTx calls.
   - New `correct()` is ~25 lines: decode body, parse UUID, call `correctionBus.Convert()`, map errors, return result.
   - Add `mapError(err)` using `errors.Is` against bus sentinel errors.

3. Modify `app/domain/correctionapp/correctionapp_test.go`:
   - Update `newHandler()` to construct `correctionbus.NewBusiness(...)` and inject just the bus.
   - All 9 existing tests stay; assertions unchanged. Watch for the content-separator change (`": "` → `"\n\n"`) — any test asserting on note `Content` strings needs updating.

4. Run `make test ./app/domain/correctionapp/... ./business/domain/correctionbus/...` — green.

**Phase 2 commit:** `refactor(correctionapp): delegate conversions to correctionbus`

### Phase 3 — Delete reclassifybus and reclassifyapp

1. Delete `business/domain/reclassifybus/` (entire directory).
2. Delete `app/domain/reclassifyapp/` (entire directory).
3. Remove `Reclassify` field and its wiring from `business/sdk/dbtest.BusDomain`.
4. Modify `api/services/planner/main.go`:
   - Remove `reclassifyapp` import (line 31).
   - Remove `reclassifyapp.Routes{}` from the routes batch (line 301).
5. Run `make test` — full suite must pass.

**Phase 3 commit:** `chore(reclassify): remove deprecated reclassifybus and reclassifyapp`

### Phase 4 — Frontend cleanup

1. Modify `api/services/frontend/web/src/views/TaskDetailView.vue`:
   - Replace `handleConvertToNote()` body (lines 133-151) to call `correctionService.correct(task.value.id, 'task', 'note')` instead of `taskService.convertTaskToNote(...)`. Keep the rest of the function (taskStore.remove, navigation, query preservation) identical — `correctionService.correct` returns `{id, type}` which is enough for the navigation step.
   - Remove `import { taskService }` if no longer used (verify).

2. Modify `api/services/frontend/web/src/views/NoteDetailView.vue`:
   - Replace `handleConvertToTask()` body (lines 114-132) to call `correctionService.correct(note.value.id, 'note', 'task')`.
   - **Address handlePromote bug** (lines 86-95): `newType==='event'` currently routes to `'tasks'`. Either fix to route to `'tasks'` for task and... we don't have an `'events'` route in the same shape — verify the existing event-detail route name. Lowest-friction fix: `router.push({ name: newType === 'task' ? 'tasks' : 'events' })` if `events` is a valid route, otherwise route to `tasks` for both with a comment explaining why. **Decision needed at planning time** — surface in beads issue.

3. Delete `convertTaskToNote` from `api/services/frontend/web/src/services/taskService.ts` (lines 22-23).
4. Delete `convertNoteToTask` from `api/services/frontend/web/src/services/noteService.ts` (lines 17-18).

5. Modify `api/services/frontend/web/src/__tests__/views/TaskDetailView.reclassify.test.ts`:
   - Remove the `taskService.convertTaskToNote` mock (lines 55-63).
   - All 5 test cases now assert `correctionService.correct` was called with `('task', 'note')`.
6. Mirror for `api/services/frontend/web/src/__tests__/views/NoteDetailView.reclassify.test.ts`.

7. Run `make frontend-test` — green.
8. Run `make frontend-build` — green (per project's never-skip-build rule).

**Phase 4 commit:** `refactor(frontend): unify conversion calls through correctionService`

### Phase 5 — Arch docs

1. Modify `.docs/arch/correction-backend.md`:
   - Update file map to reflect new `business/domain/correctionbus/` location.
   - Document the `Convert(direction)` API and sentinel errors.
   - Merge in reclassify's preflight rules.
   - Update Routes section (only `/corrections` survives).
   - Update Cross-Domain Dependencies (now: taskbus, notebus, eventbus, classificationcorrectionbus — direct).
   - Append an Updates entry for 2026-04-29.

2. Modify `.docs/arch/correction-frontend.md`:
   - Note that `correctionService.correct` is the single conversion entry point across all detail views.
   - Update Cross-Domain Dependencies — remove `taskService.convertTaskToNote` and `noteService.convertNoteToTask`.

3. Delete `.docs/arch/reclassify-backend.md`.
4. Delete `.docs/arch/reclassify-frontend.md`.

5. Update `.docs/TOC.md` if any index entries point at the deleted reclassify docs.

**Phase 5 commit:** `docs(arch): consolidate correction docs; remove reclassify`

### Phase 6 — Close beads issues

1. `bd close planner-ow7c` (consolidation done).
2. `bd close planner-hurd` (subsumed; lineage already preserved as of `569940f8`).

---

## Test plan

### Backend
- `business/domain/correctionbus/correctionbus_test.go` — 12 tests:
  - Happy path × 6 directions (assert: target created, source deleted, audit row written, lineage preserved, separator is `\n\n`, `Unconfirmed=false`)
  - `TestPreflight_Recurring` (task source)
  - `TestPreflight_HasDependents` (task source)
  - `TestPreflight_HasRecurrenceChildren` (task source)
  - `TestPreflight_ScheduledInPlan` (task source) — expect 409 mapping behavior at app layer
  - `TestConvert_InvalidDirection` (e.g., task→task)
  - `TestConvert_SourceNotFound`
- `app/domain/correctionapp/correctionapp_test.go` — 9 existing tests rewired to use `correctionbus`. Assertions on response shape stay; assertions on note content strings need separator update.

### Frontend
- `__tests__/views/TaskDetailView.reclassify.test.ts` — 5 tests, mocks updated.
- `__tests__/views/NoteDetailView.reclassify.test.ts` — 5 tests, mocks updated.

### Manual smoke (after Phase 4)
- Convert a task to a note via the convert button — confirm new note has both old tags, same `RawInputID`, `unconfirmed=false`, content separated by `\n\n`.
- Convert a recurring task — confirm 400 error with sentinel-derived message (was previously allowed via `/corrections`).

---

## Cascade rules surfaced

- `correctionbus.NewBusiness` adds `eventBus` to the composition; `route.go` must construct `eventbus.NewBusiness` (already does for correctionapp today).
- `dbtest.BusDomain.Reclassify` removed; any test referencing `db.BusDomain.Reclassify` outside `reclassifybus_test.go` (deleted) must be cleaned. `grep -rn "BusDomain.Reclassify"` before Phase 3.
- `main.go` must have `reclassifyapp` import and route entry both removed (Go fails compile on unused imports).
- Migration 1.46 already added `'correction'` to `notes.source` CHECK; no new migration needed.

---

## Risks and mitigations

- **Content separator change (`": "` → `"\n\n"`) is a user-visible behavior change.** Any existing notes converted via `/corrections` retain their `": "` content; new conversions use `\n\n`. Acceptable for a personal app, but mention in commit message. Mitigation: none needed — single user, one-time change.
- **`eventbus.CreateWithTx`/`DeleteWithTx` may be missing.** Verify before Phase 1; if missing, add as a pre-step (small Storer addition + db implementation).
- **Frontend `handlePromote` event-routing bug** is pre-existing; fixing in Phase 4 widens scope. Decision: fix it because we're already touching the function and leaving a known bug is worse than a slightly wider PR. Surface in the beads issue.
- **Test rewiring breaks dbtest for unrelated suites** if any test reaches into `BusDomain.Reclassify`. Phase 3 grep is the gate.

---

## Sequencing summary

```
Phase 1 (new bus + tests)           → green build
Phase 2 (handler delegates to bus)  → green build (correctionapp tests pass with new wiring)
Phase 3 (delete reclassify)         → green build (full make test)
Phase 4 (frontend cleanup)          → green build + frontend tests + frontend build
Phase 5 (arch docs)                 → no build impact
Phase 6 (close beads)               → bookkeeping
```

Each phase commits independently. Phases 1-3 can be one PR or three; Phase 4 should land in the same PR as Phases 1-3 to avoid an interim state where backend has changed but frontend still calls deleted endpoints — though since we're not deleting the routes until Phase 3, the frontend keeps working until Phase 3 lands.

**Recommendation: single PR covering all six phases, six commits.** Single user, no in-flight clients, no need for staged rollout.
