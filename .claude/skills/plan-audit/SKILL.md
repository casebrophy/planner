---
name: plan-audit
description: Cross-reference planning docs against the codebase to detect drift. Use after building features, or periodically, to keep docs in sync with reality. Surfaces mismatches and asks before changing anything.
---

# Planning Docs Audit

Cross-reference `.docs/` planning files against the actual codebase to find drift.

## Setup

1. Run `bd list --status=open` to see current tracked work
2. Run `bd stale` to find issues with no recent activity
3. Run `bd orphans` to find issues with broken dependencies

## Codebase scan (bounded)

Read ONLY these files:
- `business/sdk/migrate/sql/*.sql` — actual DB schema
- `**/route.go` — actual routes
- `business/domain/*/model.go` — actual business models
- `app/domain/*/model.go` — actual app models
- `.docs/arch/*.md` — architecture maps

Do NOT read full handler or store implementations.

## Checks

1. **Schema drift** — tables/columns in migration SQL vs. `03-data-model.md`
2. **Route drift** — routes registered in code vs. `.docs/arch/` route tables
3. **Model drift** — business model fields vs. `.docs/arch/` type definitions
4. **Roadmap drift** — items marked "not built" in `07-roadmap.md` that now exist in code
5. **TOC staleness** — `.docs/TOC.md` entries pointing to sections that no longer exist
6. **Arch file freshness** — `.docs/arch/` files vs. actual code (check if models/routes have changed)
7. **CLAUDE.md index** — "Built" / "Not built" lists vs. actual codebase state
8. **Beads hygiene** — stale issues, orphaned dependencies, open issues for already-completed work

## Output

Present a drift report organized by check type. For each finding:
- What the doc says
- What the code says
- Suggested fix (doc update or code change)

For beads hygiene, list:
- Stale issues that should be closed or updated
- Orphaned dependencies that need fixing
- Open issues whose work is already done in code

## Rules

- **Ask before changing anything** — docs represent intent, not just reality. The code might be wrong.
- After user approves changes, update the relevant docs, TOC.md, and CLAUDE.md index
- If arch files need regeneration, suggest running `/go-arch` for the affected domains
- Close beads issues that correspond to completed work (with user approval)
- Create new beads issues for drift findings that require code or doc changes
