# Handoff: planner-s6ya.2 (Phase 4 of 8)

**Title**: Remove thread entries UI surfaces
**Worker exit code**: 0

## Files changed
- M  api/services/frontend/web/src/components/tasks/TaskDetailPanel.vue (notes panel removed)
- M  api/services/frontend/web/src/views/TaskDetailView.vue (notes section removed)
- M  api/services/frontend/web/src/views/ContextDetailView.vue (timeline section removed)
- D  __tests__/views/TaskDetailView.reclassify.test.ts (orphaned)
- M  __tests__/views/TaskDetailView.test.ts (notes assertions removed)
- M  __tests__/views/ContextDetailView.test.ts

## Public surface added
(none — pure removal)

## Tests
- Pruned: notes/timeline-related tests
- Existing test suite still passes

## Deferred
(none)

## Notes for next phases
- TaskDetailView.vue and TaskDetailPanel.vue both touched → phase 6 (s6ya.3 task deps UI) will edit these again. No conflicts expected since deps UI is a different section.
- ContextDetailView.vue touched → phase 7 (s6ya.6 context outcome/summary) will edit this again. Different fields, no conflicts expected.
- thread_entries table + bus + API endpoints kept (data-keeping).
- useTaskNotes composable referenced in arch waiver — likely deleted; re-grep if needed.
