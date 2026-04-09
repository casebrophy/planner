---
name: plan-feature
description: Directed feature planning for the planner app. Use when the user has decided what to build and wants to make it concrete — runs a tiered Haiku→Sonnet→Opus pipeline to minimize token usage and maximize plan accuracy. Argument is the feature name (e.g., "email-ingestion", "frontend", "scheduling").
---

# Feature Planning — Tiered Pipeline

Plan a specific feature using a cost-efficient multi-agent pipeline. Haiku agents explore the codebase cheaply. Sonnet synthesizes findings into a context brief. Opus reads only the brief and produces the implementation plan.

---

## Phase 0: Beads Pre-Check

Run these before dispatching any agents. They are cheap CLI calls.

```bash
bd search <feature-keyword>
bd memories <feature-keyword>
bd list --status=in_progress
```

Collect the output. It will be passed to Sonnet as the "Beads Context" section of the brief.

If `bd list --status=in_progress` returns an issue that exactly matches this feature, surface it to the user and ask whether to proceed or continue from the existing issue.

---

## Phase 1: Haiku Exploration

### Determine direction and depth

**Direction** (infer from feature description, default to `full-stack`):
- `backend-only` — skip frontend agent
- `frontend-only` — skip backend, migration agents
- `full-stack` — all agents

**Depth** (infer from feature scope):
| Depth | When to use | Agents |
|-------|-------------|--------|
| 2 | Most features | Backend arch + patterns, Frontend arch + tests |
| 3 | New domain or cross-cutting feature | + Cross-cutting (auth, types, middleware, migrations) |
| 4 | Major feature touching multiple domains | + Related domains (integration point check) |

### Dispatch agents

Launch ALL applicable agents in a SINGLE message using the Agent tool. Every agent MUST use `model: "haiku"`.

**Backend agent prompt (include if direction is `backend-only` or `full-stack`):**

> You are a codebase exploration agent. Search the planner app backend for patterns relevant to implementing `<feature-name>`.
>
> 1. Check `.docs/arch/` for any file matching `<feature-keyword>-backend.md`. If found, read it and return its contents verbatim — stop here.
> 2. If no arch file, search these directories:
>    - `app/domain/` — existing app-layer handlers and DTOs
>    - `business/domain/` — existing business logic and Storer interfaces
>    - `business/types/` — enum types
>    - `business/sdk/migrate/sql/` — migration SQL patterns
> 3. Find the closest existing domain to `<feature-name>` (the most structurally similar).
> 4. For that domain, report:
>    - All file paths with one-line descriptions
>    - Key type definitions (structs, interfaces)
>    - Migration SQL table structure
>    - Any cross-layer cascade rules evident from the code
>
> Return a compact findings report. File path + line number + one-line summary for each match.

**Frontend agent prompt (include if direction is `frontend-only` or `full-stack`):**

> You are a codebase exploration agent. Search the planner app frontend for patterns relevant to implementing `<feature-name>`.
>
> 1. Check `.docs/arch/` for any file matching `<feature-keyword>.md` (no `-backend` suffix). If found, read it and return its contents verbatim — stop here.
> 2. If no arch file, search these directories:
>    - `web/src/views/` — existing view components
>    - `web/src/stores/` — Pinia stores (CRUD factory pattern)
>    - `web/src/services/` — API service files (CRUD factory pattern)
>    - `web/src/components/` — reusable components
>    - `web/src/**/__tests__/` — existing frontend test files
> 3. Find the closest existing frontend feature to `<feature-name>`.
> 4. For that feature, report:
>    - All file paths with one-line descriptions
>    - Store structure (state shape, actions)
>    - Service pattern (`createCRUDService` usage)
>    - Existing test file paths and what they test
>
> Return a compact findings report. File path + one-line summary for each match.

**Cross-cutting agent prompt (include at depth 3+):**

> You are a codebase exploration agent. Find all auth, type, and middleware patterns in the planner app relevant to implementing `<feature-name>`.
>
> Search:
> 1. `app/sdk/mid/` — auth middleware implementation
> 2. `app/domain/<closest-domain>app/route.go` — how routes register auth middleware
> 3. `business/types/` — all enum type files (list names and `Parse()` signatures)
> 4. `business/sdk/migrate/sql/` — most recent migration file (report its filename and structure)
>
> Report:
> - Exact auth middleware usage pattern from an existing `route.go`
> - `X-API-Key` header usage if visible in any client code
> - List of all enum types with their valid values
> - Migration file naming convention
>
> Return a compact findings report.

**Related-domains agent prompt (include at depth 4):**

> You are a codebase exploration agent. Find integration points between `<feature-name>` and existing domains.
>
> Search `business/domain/` for domains that are likely to reference or be referenced by `<feature-name>`. For each candidate:
> - Report the Storer interface methods
> - Report any foreign key relationships visible in migration SQL
> - Report any cross-domain calls in the business layer
>
> Return a compact findings report.

---

## Phase 2: Sonnet Synthesis

### Option A (default): 1 Sonnet agent

After all Haiku agents return, dispatch a single Sonnet agent (`model: "sonnet"`) with this prompt:

> You are a context synthesis agent. You have received codebase findings from parallel Haiku exploration agents. Your job is to produce a structured "context brief" that will be read by an Opus planning agent — Opus will NOT read any raw files, only your brief.
>
> ## Haiku Findings
>
> `<paste all Haiku agent output here>`
>
> ## Beads Context
>
> `<paste Phase 0 bd output here>`
>
> ## Instructions
>
> Produce a context brief in exactly this format. Be specific — include file paths and line numbers. Do not omit sections.
>
> ---
>
> # Context Brief: `<feature-name>`
>
> ## Beads Context
> - Related open issues: `<list or "none">`
> - Memory hits: `<list or "none">`
> - Active in-progress work that overlaps: `<list or "none">`
>
> ## Files to Touch
> | Action | File | Layer | Why |
> |--------|------|-------|-----|
> (Fill every row. Action = CREATE or MODIFY. Layer = business/store/app/frontend/wire/migration)
>
> ## Pattern to Follow
> Closest existing feature: **`<domain>`** (`<path>`)
> - `<pattern name>`: `<file>:<line>` — `<one-line description>`
> (List 4–8 specific pattern references)
>
> ## Cascade Rules
> - New field on model → update DB struct + converters + SQL + app DTO (converters in `stores/<name>db/model.go` and `app/domain/<name>app/model.go`)
> - New filter field → `bus/filter.go` + `stores/<name>db/filter.go` + `app/domain/<name>app/filter.go`
> - New enum value → `business/types/<enum>/` + migration CHECK constraint
> - New Storer method → interface in `<name>bus.go` + implementation in `stores/<name>db/<name>db.go`
> (Add any feature-specific cascade rules you observed)
>
> ## Auth Checklist
> - [ ] Routes registered with auth middleware via `Routes.Add()`
> - [ ] `X-API-Key` header pattern followed (not written from scratch — copy from `<reference file>`)
> - [ ] Sidecar auth alignment verified if feature touches backend→sidecar calls
> - [ ] No unauthenticated endpoints introduced unintentionally
> - Auth middleware reference: `<exact file:line from Haiku findings>`
>
> ## Test Approach
> - Store tests: `business/sdk/dbtest` — closest fixture: `<file path from Haiku>`
> - API tests: `app/sdk/apitest` — closest fixture: `<file path from Haiku>`
> - Frontend unit tests: `web/src/**/__tests__/` (Vitest) — closest fixture: `<file path from Haiku>`
> - Frontend component tests: `<file path from Haiku>`
>
> ## Gotchas
> - `sqldb.ErrDBNotFound` must be checked explicitly (not caught by default → surfaces as 500)
> - DB port is 5433 in local dev (not 5432)
> - Frontend CRUD factories: `createCRUDService`, `createCRUDStore` — copy the pattern exactly
> (Add any feature-specific gotchas you found in Haiku findings)
>
> ---
>
> Return only the context brief. No preamble, no explanation.

### Option B (upgrade path): 3 Sonnet agents

If Option A's brief is missing domain-specific detail, switch to 3 Sonnet agents dispatched in parallel:
- **Backend Sonnet:** fills "Files to Touch (backend rows)", "Pattern to Follow", "Cascade Rules", backend "Gotchas"
- **Frontend Sonnet:** fills "Files to Touch (frontend rows)", frontend "Test Approach", frontend "Gotchas"
- **Cross-cutting Sonnet:** fills "Auth Checklist", all "Test Approach" sections, migration notes

Merge their outputs into the same brief format before Phase 3. No changes to Phase 1 or Phase 3 required.

---

## Phase 3: Opus Planning

Read ONLY the context brief from Phase 2. Do not read any raw files.

Invoke the `superpowers:writing-plans` skill to produce the implementation plan, using the brief as the sole source of truth for:
- Which files to create/modify
- Which patterns to follow
- Which tests to write
- Which auth steps to include

The writing-plans skill will save the plan to `.docs/superpowers/plans/YYYY-MM-DD-<feature-name>.md`.

---

## Phase 4: Beads Issue Creation

After the implementation plan is written, create beads issues linked back to the plan.

The plan file path is: `.docs/superpowers/plans/YYYY-MM-DD-<feature-name>.md`

```bash
# 1. Create a parent feature issue
bd create --title="<feature-name>" --description="<one-line feature summary>" --type=feature --priority=2 --context="Plan: .docs/superpowers/plans/YYYY-MM-DD-<feature-name>.md"

# 2. Create child task issues under the parent, each linking the plan
bd create --title="<task title>" --description="<task description>" --type=task --priority=2 --parent=<parent-issue-id> --context="Plan: .docs/superpowers/plans/YYYY-MM-DD-<feature-name>.md — see section: <relevant heading>"

# 3. Wire sequential dependencies
bd dep add <later-issue-id> <earlier-issue-id>
```

Rules:
- One parent feature issue for the whole plan, child task issues underneath
- Each child issue's `--context` links the plan file AND the specific section relevant to that task
- One beads issue per plan task (not per step)
- Set dependencies to match the plan's ordering constraints
- Use `bd remember` to save any key design decisions that emerged during planning
