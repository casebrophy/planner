# Handoff: planner-s6ya.1 (Phase 1 of 8)

**Title**: Remove energy field from task UI
**Merge commit**: HEAD after merge
**Worker exit code**: 0

## Files changed
M  api/services/frontend/web/src/__tests__/components/tasks/TaskDebriefDialog.test.ts
M  api/services/frontend/web/src/__tests__/helpers/testFactories.ts
M  api/services/frontend/web/src/__tests__/types/enums.test.ts
D  api/services/frontend/web/src/components/shared/EnergyIndicator.vue
M  api/services/frontend/web/src/components/tasks/TaskCard.vue
M  api/services/frontend/web/src/components/tasks/TaskDebriefDialog.vue
M  api/services/frontend/web/src/components/tasks/TaskDetailPanel.vue
M  api/services/frontend/web/src/components/tasks/TaskForm.vue
M  api/services/frontend/web/src/composables/useToday.ts
M  api/services/frontend/web/src/types/enums.ts
M  api/services/frontend/web/src/types/task.ts

## Public surface added
(none)

## Tests added
(updated existing: TaskDebriefDialog.test.ts, enums.test.ts)

## Deferred
(none)

## Notes for next phases
- `EnergyIndicator.vue` is gone. Don't reference it.
- `Energy` enum / type may still exist in TS types but no UI components consume it.
- `useToday.ts` was touched (energy-based sort/filter removed). Phase 4 (s6ya.2) likely doesn't touch this composable.
