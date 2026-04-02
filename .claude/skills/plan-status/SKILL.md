---
name: plan-status
description: Quick read-only overview of the planner app's current state. Use to orient yourself — what's built, what's next, any obvious drift. Does not modify any files.
---

# Planning Status

Quick orientation for the planner app — what's built, what's planned, what's next.

## Process

1. Read the "Planner App Context" section of `CLAUDE.md`
2. Run a fast codebase check:
   - `ls app/domain/` — which domain packages exist
   - `grep "CREATE TABLE" business/sdk/migrate/sql/migrate.sql` — which tables exist
   - Count routes: `grep -r "a.Handle\|a.HandleNoMiddleware" app/domain/*/route.go`
3. Cross-reference with the "Built" / "Not built" lists in CLAUDE.md
4. Read `07-roadmap.md` to identify current phase and next phase
5. Run beads commands for work status:
   - `bd stats` — project health overview
   - `bd ready` — what's available to work on next
   - `bd list --status=in_progress` — what's currently being worked on
   - `bd blocked` — what's stuck

## Output format

```
Current phase: N (Name) — status
Built: [list of working features]
Not built: [list of planned but unimplemented features]
Next up: [1-2 most logical next steps]
Drift: [any obvious mismatches between CLAUDE.md and codebase, or "none detected"]

Beads:
  Open: N | In progress: N | Blocked: N
  Ready to work: [list of ready issues with IDs]
  Currently active: [list of in-progress issues with IDs]
```

## Rules

- **Read-only** — do not modify any files
- Keep output concise — this is a quick check, not a full audit
- If significant drift is found, suggest running `/plan-audit` for a thorough review
