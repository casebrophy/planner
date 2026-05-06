# Handoff: planner-s6ya.5 (Phase 3 of 8)

**Title**: Disable knowledge-gap clarifications (UI + generator)
**Worker exit code**: 0

## Files changed
- M  app/domain/clarificationapp/clarificationapp.go (kind filter excludes knowledge_gap)
- A  app/domain/clarificationapp/knowledge_gap_disabled_test.go
- M  business/domain/clarificationbus/filter.go (added a kind exclusion)
- M  business/domain/clarificationbus/stores/clarificationdb/filter.go (applies it in SQL)
- M  business/domain/knowledgegapbus/knowledgegapbus.go (generator short-circuited)

## Public surface added
(none — generator returns early; existing types untouched)

## Tests added
- knowledge_gap_disabled_test.go (verifies generator no-ops + queue filter excludes)

## Deferred
(none)

## Notes for next phases
- knowledgegap domain code is dormant but present (not deleted).
- Existing knowledge_gap clarification rows untouched in DB.
- Phase 4 (s6ya.2 thread entries) and phase 7 (s6ya.6 contexts) likely don't intersect this domain.
