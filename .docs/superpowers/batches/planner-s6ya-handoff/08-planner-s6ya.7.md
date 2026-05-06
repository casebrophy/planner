# Handoff: planner-s6ya.7 (Phase 8 of 8)

**Title**: Remove inactivity detection (cards + check job)
**Worker exit code**: 0

## Files changed
- DELETED: api/services/frontend/web/src/components/clarifications/ClarificationCard.vue (inactivity-related card)
- D  __tests__/components/clarifications/ClarificationCard.test.ts (orphaned)
- M  api/services/frontend/web/src/__tests__/helpers/testFactories.ts
- M  api/services/frontend/web/src/__tests__/types/enums.test.ts
- M  api/services/frontend/web/src/types/enums.ts
- M  api/services/planner/main.go (inactivity job no longer registered)

## Public surface added
(none)

## Tests
- Pruned: ClarificationCard inactivity test
- enums.test.ts updated

## Deferred
(none)

## Notes
- inactivity_checks table intact (data-keeping).
- inactivity domain code dormant, not deleted.
- Clarification queue now shows only type/correction kinds (knowledge-gap disabled in phase 3, inactivity removed here).
