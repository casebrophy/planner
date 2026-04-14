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

**Verify:** Write tests in `contextbus_test.go`:
- Create list context → succeeds
- Pause list context → fails (InvalidArgument)
- Close list context → succeeds
- Create list with area parent → succeeds
- Create list with project parent → fails (InvalidArgument)

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
Task 1 (enum + migration) → Task 2 (business rules) → Task 3 (TS types + store)
Task 3 → Task 4 (board view)
Task 3 → Task 5 (detail view)
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
