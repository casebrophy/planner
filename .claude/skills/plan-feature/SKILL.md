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

### Phase 0 Confirmation (interactive)

Before dispatching any agents, surface the bd output to the user concisely:

- Open issues that look related: `<list or "none">`
- Memory hits: `<list or "none">`
- In-progress overlap: `<list or "none">`

Ask: *"Any of these change scope? Should we continue from an existing issue rather than start fresh, or is the feature framing different from what's already tracked?"*

Wait for the user's response. Do not proceed to Phase 1 until they confirm. The cost of one round-trip here is much smaller than the cost of running the full pipeline against the wrong scope.

---

## Phase 1: Haiku Exploration

### Confirm direction and depth with user

Propose a direction and depth, but do NOT dispatch agents until the user confirms. Inferring silently is the single biggest source of plans that miss scope. Format the proposal like:

> Based on `<feature-name>`, I propose:
> - **Direction**: `<backend-only | frontend-only | full-stack>` because `<one-line reason>`
> - **Depth**: `<2 | 3 | 4>` because `<one-line reason>`
> - **Agents that will dispatch**: `<list>`
>
> Confirm or override?

**Direction options:**
- `backend-only` — skip frontend agent
- `frontend-only` — skip backend, migration agents
- `full-stack` — all agents

**Depth options:**
| Depth | When to use | Agents |
|-------|-------------|--------|
| 2 | Most features | Backend arch + patterns, Frontend arch + tests |
| 3 | New domain or cross-cutting feature | + Cross-cutting (auth, types, middleware, migrations) |
| 4 | Major feature touching multiple domains | + Related domains (integration point check) |

Wait for the user's response before dispatching.

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

## Phase 2.5: Interactive Brief Review

Before invoking Opus, surface the context brief from Phase 2 to the user verbatim — no paraphrasing. Then ask:

- *"Are any files missing from 'Files to Touch'?"*
- *"Are there cascade endpoints I haven't called out (e.g., a frontend store that consumes the new API field, a migration that needs a CHECK constraint)?"*
- *"Any cross-domain integration I'm assuming you'd want handled differently?"*

Wait for the user's response. Append any additions they supply to the brief before passing it to Phase 3.

The point of this gate: Phase 3 reads ONLY the brief, so any gap that survives this checkpoint becomes a gap in the plan — and a gap in the plan becomes incomplete beads.

---

## Compile-Gate Scoping Doctrine

The single biggest source of "things built but not hooked up" is beads that *can* close green with partial work. Scope each bead so incomplete implementation breaks the build or the test suite.

Apply the doctrine when generating the plan (Phase 3) and the per-bead wiring (Phase 4):

1. **Pair the producer and consumer.** A bead that adds a Storer method but no caller can close green even if the handler is never wired. Either include the caller in the same bead, OR include a test that exercises the new method end-to-end.

2. **Behavior tests over symbol existence.** A bead that adds `taskbus.UpdateTask.Notes` should include a test that PATCHes a task with `notes` and reads back the value via the API. If any layer in the chain — bus → store → app DTO → JSON — isn't wired, the test fails.

3. **Type-level gates.** Add a struct field that downstream conversion code MUST populate. If `toAppX` doesn't set it, the field is zero-valued and a behavior test catches it.

4. **Reviewer's question for every bead:** *"What's the smallest amount of partial work that closes this bead with a green build?"* If meaningful work could be omitted, the bead is too coarse — split it, or add a test that fails without the missing piece.

Phase 3 (Opus) and Phase 4 (wiring generation) MUST apply this doctrine. Each bead's wiring metadata records the compile gates so `execute-beads` verifies them mechanically.

---

## Phase 3: Opus Planning

Read ONLY the context brief from Phase 2 (with any Phase 2.5 additions). Do not read any raw files.

Invoke the `superpowers:writing-plans` skill to produce the implementation plan. The plan MUST apply the **Compile-Gate Scoping Doctrine** above: every bead either includes a behavior test exercising its full external surface, OR includes a consumer that requires the producer to exist (compile failure if the producer is missing). If a bead can be closed with partial work and a green build, split it.

The brief is the sole source of truth for:
- Which files to create/modify
- Which patterns to follow
- Which tests to write
- Which auth steps to include

The writing-plans skill will save the plan to `.docs/superpowers/plans/YYYY-MM-DD-<feature-name>.md`.

---

## Phase 4: Beads Issue Creation

After the implementation plan is written by `superpowers:writing-plans`, create beads issues with structured wiring metadata. The plan file path is `.docs/superpowers/plans/YYYY-MM-DD-<feature-name>.md`.

### 4.A — Generate per-bead wiring JSON

Dispatch a single haiku worker via the in-process **Agent** tool (NOT `claude -p`) with this prompt:

> You are a wiring-spec generator. Read:
> 1. The implementation plan at `.docs/superpowers/plans/YYYY-MM-DD-<feature-name>.md`
> 2. The "Cross-layer Impact Rules" and "New Domain Checklist" sections of `/Users/casebrophy/personal/planner/CLAUDE.md`
>
> For each child task identified in the plan (each row of the implementation breakdown that will become a separate bead), emit one JSON file at `.docs/superpowers/plans/YYYY-MM-DD-<feature-name>/wiring/<task-slug>.json` matching this schema:
>
> ```json
> {
>   "version": 2,
>   "issue_title": "<bead title>",
>   "domain": "<closest existing domain or 'new'>",
>   "kind": "new_field | new_filter | new_order | new_storer_method | new_enum | new_domain | modify",
>   "files": [
>     {
>       "path": "<repo-relative path>",
>       "action": "create | modify",
>       "reason": "<one-line why this file is in scope>",
>       "expected_symbols": ["<source-text fragment>", "..."],
>       "external_surface": false
>     }
>   ],
>   "compile_gates": [
>     {
>       "consumer_file": "<file that uses the new symbol>",
>       "consumer_symbol": "<source fragment in consumer file>",
>       "requires_producer_symbol": "<producer symbol the consumer references>"
>     }
>   ],
>   "behavior_test": {
>     "path": "<test file path>",
>     "test_name": "<test function name>"
>   },
>   "test_packages": ["./<go package glob>", "..."],
>   "frontend_changed": false
> }
> ```
>
> Rules:
> - The `kind` field is the deterministic anchor — fill `files` from the matching cross-layer cascade rule, then specialize the names (e.g., `new_filter` → `business/domain/<X>bus/filter.go` + `business/domain/<X>bus/stores/<X>db/filter.go` + `app/domain/<X>app/filter.go`).
> - `expected_symbols` MUST be literal source-text fragments suitable for `git grep -F`, not regex patterns.
> - Mark `external_surface: true` only for symbols downstream phases will reference (function signatures, exported types, route paths).
> - **`compile_gates` is the "won't compile if unhooked" guarantee.** For every `external_surface: true` symbol added by this bead, identify a consumer file in scope that references it and record the pair. Catches the "added a Storer method but no handler calls it" failure mode mechanically.
> - **`behavior_test` is the alternative gate** when no in-scope consumer exists (e.g., backend-only bead with no caller in this bead). The named test must exercise the full external surface added — if any layer is unhooked, it fails.
> - **At least one of `compile_gates` (non-empty) or `behavior_test` MUST be present.** A bead with neither is too coarse and can close green with partial work — flag it in the report so the user can split or add a test.
> - Set `frontend_changed: true` if any file path is under `api/services/frontend/web/src/`.
> - Create the wiring directory if it does not exist.
>
> Report under 100 words: number of wiring files written, beads where neither `compile_gates` nor `behavior_test` could be set (these need user review), any tasks where the cascade was unclear.

Wait for the worker to complete and verify the wiring/ directory contains one JSON file per child task.

### 4.B — Interactive bead review

Before creating any beads, surface the proposed bead set to the user. Show one block per bead with this shape:

```
[<task-slug>] <issue title>
  Depends on:    <list or "none">
  Compile gates: <count> — <one-line description, e.g., "handler at app/domain/taskapp/taskapp.go references taskbus.UpdateTask.Notes">
  Behavior test: <path::testname or "none">
  Files (<count>): <comma-separated short list>
```

Then ask:

- *"Any bead too big?"* (a bead where partial work could still close green)
- *"Any cascade endpoint without a compile gate?"* (e.g., a frontend store that consumes a new API field but isn't covered)
- *"Want to merge or split any?"*

Apply requested merges/splits/additions to the wiring JSON files before proceeding to 4.C.

This gate is the last opportunity to catch decomposition problems before beads are created — once they're in beads, fixing them is more expensive (close + re-create + re-wire dependencies).

### 4.C — Create beads with metadata

For the parent feature issue:

```bash
bd create --title="<feature-name>" \
  --description="<one-line feature summary>" \
  --type=feature --priority=2 \
  --context="Plan: .docs/superpowers/plans/YYYY-MM-DD-<feature-name>.md"
```

For each child task issue, attach the wiring JSON via `--metadata`:

```bash
bd create --title="<task title>" \
  --description="<task description>" \
  --type=task --priority=2 \
  --parent=<parent-issue-id> \
  --context="Plan: .docs/superpowers/plans/YYYY-MM-DD-<feature-name>.md — section: <relevant heading>" \
  --metadata @.docs/superpowers/plans/YYYY-MM-DD-<feature-name>/wiring/<task-slug>.json
```

Wire dependencies:

```bash
bd dep add <later-issue-id> <earlier-issue-id>
```

Rules:
- One parent feature issue, child task issues underneath
- Each child bead's `--metadata` points at its wiring JSON file
- `execute-beads` reads the wiring back via `bd show <id> --json | jq .metadata` and uses it to enforce the build/test/wiring hard gate before closing
- v1 wiring (no `compile_gates` / `behavior_test`) is honored as legacy by `execute-beads` — those gates are skipped for v1 beads. Use v2 for any new beads.
- Use `bd remember` to save any key design decisions that emerged during planning
