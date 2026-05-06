# Handoff: planner-s6ya.4 (Phase 2 of 8)

**Title**: Remove daily plan auto-assembly
**Worker exit code**: 0

## Files changed (summary)
- DELETED: api/services/frontend/web/src/components/dailyplan/* (DismissModal, EventCard, PlanGroupHeader, PlanItemCard)
- DELETED: api/services/frontend/web/src/composables/useDailyPlan.ts
- DELETED: api/services/frontend/web/src/services/dailyPlanService.ts
- DELETED: api/services/frontend/web/src/stores/dailyPlanStore.ts
- DELETED: api/services/frontend/web/src/types/dailyPlan.ts
- DELETED: api/services/frontend/web/src/views/DailyPlanView.vue
- M  api/services/frontend/web/src/views/TodayView.vue (massively trimmed)
- M  api/services/frontend/web/src/types/index.ts (dropped dailyPlan re-export)
- M  api/services/planner/main.go (dailyplan wiring trimmed)
- M  app/domain/dailyplanapp/dailyplanapp.go (327 → ~minimal lines; auto-assembly stripped)
- M  app/domain/dailyplanapp/route.go (routes trimmed)
- M  app/domain/mcpapp/mcpapp.go (generate_daily_plan tool removed)

## Public surface added
(none — pure removal)

## Tests deleted
- __tests__/services/dailyPlanService.test.ts
- __tests__/stores/dailyPlanStore.test.ts
- __tests__/components/shared/EnergyIndicator.test.ts (orphaned by phase 1)

## Deferred
(none)

## Notes for next phases
- DailyPlan UI is gone. Don't reference dailyPlanService / dailyPlanStore / DailyPlanView.
- TodayView.vue was rewritten to a minimal version; phase 8 (s6ya.7 inactivity removal) may touch it.
- mcpapp.go was edited; phase 5 (s6ya.8 observations/debrief) and phase 7 (s6ya.6 contexts) may also touch it — be careful with conflicts.
- dailyplanapp still exists with read-only API endpoints (cut posture: keep API shape).
