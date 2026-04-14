# List Context Kind

**Date:** 2026-04-14
**Status:** Draft
**Scope:** Backend + Frontend

## Summary

Add a `list` kind to contexts for simple checklist use cases (groceries, packing lists, wishlists). Lists reuse the existing task primitive — tasks within a list context render as checkbox items (title + done/open toggle). Lists can be sub-contexts of areas via the existing `parent_context_id` relationship.

## Design Decisions

1. **Reuse tasks, not new table** — aligns with composable primitives principle. Tasks already have Open → Done lifecycle. All metadata fields (priority, energy, due date) are optional and simply hidden in the list UI.
2. **List as context kind** — kind is already a behavioral differentiator (project vs area). List is a third behavioral variant.
3. **Lists can close** — unlike areas (always active), lists have a natural "done" state. No debrief flow on close.
4. **Lists cannot be paused** — pausing doesn't make sense for a checklist.
5. **Parent constraint: lists under areas only** — lists are lightweight sub-containers of areas. A list under a project is semantically confusing.
6. **Reusable lists via reset** — instead of closing and recreating a recurring list (groceries), a "reset" action sets all Done items back to Open. This preserves item identity across cycles, enabling future prediction ("you buy milk every ~10 days"). Task `completed_at` gets overwritten on reset, but the activity/thread log captures the historical completion timestamps for prediction queries later.

## Tasks

### Task 1: Backend — Add `list` enum value + migration

**Files:**
| Action | File |
|--------|------|
| MODIFY | `business/types/contextkind/contextkind.go` |
| MODIFY | `business/sdk/migrate/sql/migrate.sql` |

**Steps:**
1. Add `List ContextKind = "list"` constant in `contextkind.go`, add to parse map and set
2. Append a new migration step to `migrate.sql` that ALTERs the CHECK constraint:
   ```sql
   ALTER TABLE contexts DROP CONSTRAINT contexts_kind_check;
   ALTER TABLE contexts ADD CONSTRAINT contexts_kind_check CHECK (kind IN ('project', 'area', 'list'));
   ```

**Verify:** `make test` — existing context tests must still pass with the new enum value.

### Task 2: Backend — Business rules for list kind

**Files:**
| Action | File |
|--------|------|
| MODIFY | `business/domain/contextbus/contextbus.go` |

**Steps:**
1. Find the area status guard (~line 96) that prevents areas from being paused/closed
2. Extend the guard: lists also cannot be paused
3. Confirm lists CAN be closed (do NOT add list to the close-block guard)
4. Add parent-kind validation in `Create` and `Update`: if kind is `list` and `parent_context_id` is set, verify the parent's kind is `area`. Reject with `errs.InvalidArgument` if parent is a project or another list.
5. Skip debrief flow for list kind on close — audit any debrief-trigger logic and add a kind check
6. Add `ResetList(ctx, contextID)` method — bulk-updates all tasks in the list context from Done → Open, clears `completed_at`. This preserves item identity across cycles for future purchase prediction. The existing activity/thread log should capture status changes, providing historical completion timestamps.

**Verify:** Write tests in `contextbus_test.go`:
- Create list context → succeeds
- Pause list context → fails (InvalidArgument)
- Close list context → succeeds
- Create list with area parent → succeeds
- Create list with project parent → fails (InvalidArgument)
- Reset list → all Done tasks become Open, Open tasks unchanged

### Task 2b: Backend — Reset list endpoint

**Files:**
| Action | File |
|--------|------|
| MODIFY | `app/domain/contextapp/contextapp.go` |
| MODIFY | `app/domain/contextapp/route.go` |
| MODIFY | `business/domain/taskbus/taskbus.go` |
| MODIFY | `business/domain/taskbus/stores/taskdb/taskdb.go` |

**Steps:**
1. Add `ResetByContext(ctx, contextID)` to `taskbus` Storer interface — bulk UPDATE tasks SET status='open', completed_at=NULL WHERE context_id=X AND status='done'
2. Implement in `taskdb`
3. Add `resetList` handler in `contextapp` — validates context exists and is kind=list, then calls `taskbus.ResetByContext`
4. Register route: `POST /api/v1/contexts/{context_id}/reset`

**Verify:** API test: create list, add tasks, complete some, reset, verify all are Open again.

### Task 2c: Backend — Bulk delete tasks endpoint

**Files:**
| Action | File |
|--------|------|
| MODIFY | `app/domain/taskapp/taskapp.go` |
| MODIFY | `app/domain/taskapp/route.go` |
| MODIFY | `business/domain/taskbus/taskbus.go` |
| MODIFY | `business/domain/taskbus/stores/taskdb/taskdb.go` |

**Steps:**
1. Add `DeleteBatch(ctx, ids []uuid.UUID)` to `taskbus` Storer interface — `DELETE FROM tasks WHERE task_id = ANY($1)`
2. Implement in `taskdb`
3. Add `deleteBatch` handler in `taskapp` — accepts `{"ids": ["uuid1", "uuid2", ...]}` body
4. Register route: `DELETE /api/v1/tasks/batch`
5. Validate all IDs exist, return 404 if any don't (or 204 on success)

**Verify:** API test: create 5 tasks, batch delete 3, verify 2 remain.

### Task 3: Frontend — TypeScript types + store

**Files:**
| Action | File |
|--------|------|
| MODIFY | `api/services/frontend/web/src/types/` (ContextKind type) |
| MODIFY | `api/services/frontend/web/src/stores/contextStore.ts` |

**Steps:**
1. Add `'list'` to the `ContextKind` TypeScript union type
2. In `contextStore.ts`, add `list: []` bucket to `contextsByKind` computed property
3. Audit any `switch`/`if` on `context.kind` — add list handling where needed

### Task 4: Frontend — Board view (Lists section)

**Files:**
| Action | File |
|--------|------|
| MODIFY | `api/services/frontend/web/src/views/ContextBoardView.vue` |

**Steps:**
1. Add a "Lists" column/section to the Kanban board, after Projects and Areas
2. Render list contexts using ContextCard (same as projects/areas)
3. Show parent area name as a subtitle if `parentContextId` is set

### Task 5: Frontend — Detail view (Checklist UI)

**Files:**
| Action | File |
|--------|------|
| MODIFY | `api/services/frontend/web/src/views/ContextDetailView.vue` |
| MODIFY or CREATE | Checklist-specific components if needed |

**Steps:**
1. Add `v-else-if="context.kind === 'list'"` branch in ContextDetailView
2. Render tasks as simple checkbox rows: `[ ] Title` / `[x] Title`
3. Toggle = PATCH task status between `open` and `done`
4. Inline "add item" input at the bottom (like a simple text input + enter to add)
5. Hide: priority, energy, due date, duration, scheduling, debrief UI, timeline view
6. Show: title, checkbox, optional drag-to-reorder (stretch goal)
7. Allow closing the list context (button or auto-close when all items done — product decision)
8. Add "Reset list" button — calls `ResetList` endpoint, refreshes task list. Useful for recurring lists (weekly groceries). Button shown when at least one task is Done.
9. **Bulk edit mode** — toggle into a selection mode for mass operations:
   - Each row gets a selection checkbox (separate from the done/open toggle)
   - "Select all" / "Select done" / "Select open" shortcuts
   - Toolbar actions: **Delete selected**, Dismiss selected
   - Swipe-to-delete on individual items (mobile-friendly)
   - This is critical for reusable lists — after resetting a grocery list, you want to quickly prune items you no longer buy

### Task 6: Frontend — Context creation (kind=list option)

**Files:**
| Action | File |
|--------|------|
| MODIFY | Context creation form/modal |

**Steps:**
1. Add "List" as a kind option when creating a new context
2. When kind=list, show a parent area selector (pre-filtered to area-kind contexts only)
3. Ensure `kind: 'list'` is explicitly sent in the POST body (not relying on default)

### Task 7: Tests

**Files:**
| Action | File |
|--------|------|
| MODIFY | Backend test files (contextbus, contextdb, apitest) |
| MODIFY | Frontend test files (contextStore, ContextDetailView) |

Covered inline in each task above. Summary:
- Backend: enum parse, business rule guards, parent-kind validation, CRUD lifecycle
- Frontend: store grouping, detail view branch rendering, task toggle behavior

### Task 8: Arch docs update

**Files:**
| Action | File |
|--------|------|
| MODIFY | `.docs/arch/context-backend.md` |
| MODIFY | `.docs/arch/context-frontend.md` |

Update kind enum tables, business rule notes, and kind-aware rendering documentation.

## Ordering Constraints

```
Task 1 (enum + migration) → Task 2 (business rules) → Task 2b (reset endpoint) + Task 2c (bulk delete) → Task 3 (TS types + store)
Task 3 → Task 4 (board view)
Task 3 → Task 5 (detail view, includes reset button + bulk edit mode)
Task 3 → Task 6 (creation form)
Tasks 4,5,6 → Task 7 (tests, though each task includes its own tests)
Task 7 → Task 8 (arch docs — update last)
```

## Gotchas

- Migration must be a NEW step appended to `migrate.sql`, not an edit to the existing CHECK constraint step
- Search for every `contextkind.Area` and `contextkind.Project` comparison in Go — missing a branch is a silent bug
- Frontend TS union type must be updated BEFORE any component code or TypeScript will reject `'list'`
- `toBusNewContext` in `contextapp/model.go` defaults empty kind to Project — frontend must always send `kind=list` explicitly
- Debrief trigger on context close must check kind and skip for lists
