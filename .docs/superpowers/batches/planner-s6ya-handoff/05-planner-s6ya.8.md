# Handoff: planner-s6ya.8 (Phase 5 of 8)

**Title**: Remove observations / debrief / weekly review surfaces and jobs
**Worker exit code**: 0

## Files changed (feature-relevant)
- DELETED: api/services/frontend/web/src/components/tasks/TaskDebriefDialog.vue
- DELETED: api/services/frontend/web/src/services/observationService.ts
- DELETED: __tests__ files for debrief/observation/weeklyreview
- M  api/services/frontend/web/src/views/TaskDetailView.vue (debrief affordance gone)
- M  api/services/frontend/web/src/views/ContextDetailView.vue (observation cards gone)
- M  api/services/frontend/web/src/components/tasks/TaskDetailPanel.vue (debrief refs removed)
- M  api/services/planner/jobs/weeklyreview.go (job disabled — runner returns early)
- M  api/services/planner/jobs/helpers_test.go (extracted shared helpers)
- D  api/services/planner/jobs/weeklyreview_test.go
- M  api/services/planner/main.go (job no longer registered)
- M  app/domain/taskapp/taskapp.go (no longer triggers outcome_observation generation)

## Side effects
- node_modules and package-lock.json caught up to current lockfile state (large diff, no functional impact)

## Public surface added
(none — pure removal/disablement)

## Tests
- Pruned: TaskDebriefDialog.test.ts, observationService.test.ts, weeklyreview_test.go, debrief/observation references in detail-view tests
- helpers_test.go extracts shared test setup for remaining job tests

## Deferred
(none)

## Notes for next phases
- TaskDetailView.vue and ContextDetailView.vue both touched — phase 7 (s6ya.6 contexts) will edit ContextDetailView again. Outcome/summary fields are independent of observation cards, no conflicts expected.
- TaskDetailPanel.vue touched again (phase 1 + phase 4 + now phase 5) — phase 6 (s6ya.3 task deps) will edit it once more. Sections are distinct (deps vs. notes vs. debrief).
- debriefbus and observationbus domain code intact but dormant.
- WeeklyReview job runner returns early; row generation stopped; tables intact.
