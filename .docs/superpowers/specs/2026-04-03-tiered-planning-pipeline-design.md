# Design: Tiered Planning Pipeline

**Date:** 2026-04-03
**Status:** Approved
**Replaces:** `/plan`, `/plan-feature` skills

## Goals

1. Minimize token usage — Opus never reads raw files during planning
2. Reduce errors and inconsistent code — Opus always has full pattern context before deciding
3. Catch auth, test, and cascade issues systematically, not by memory

## Pipeline Overview

```
/plan <feature>  or  /plan-feature <feature>
        │
        ▼
[0] BEADS PRE-CHECK (Opus, cheap CLI calls)
    bd search <feature>
    bd memories <feature>
    bd list --status=in_progress
    → surfaces conflicts, past decisions, active work

        │
        ▼
[1] HAIKU PHASE (parallel, cheap file reads)
    2–4 agents depending on direction + depth
    → each returns a raw findings report

        │
        ▼
[2] SONNET SYNTHESIS (1 agent, upgradeable to 3 for Option B)
    reads all Haiku findings
    → produces structured context brief

        │ brief only — Opus never reads raw files
        ▼
[3] OPUS PLANNING (main session)
    reads ONLY the context brief
    invokes writing-plans skill
    → produces implementation plan

        │
        ▼
[4] BEADS ISSUE CREATION (Opus, post-plan)
    auto-creates beads issues from plan steps
    wires up dependencies
```

## Phase 0: Beads Pre-Check

Run before dispatching any agents. Cheap — just CLI calls.

```bash
bd search <feature-keyword>       # related existing issues
bd memories <feature-keyword>     # saved decisions from past sessions
bd list --status=in_progress      # active work that might conflict
```

Output is passed directly into the Sonnet brief's "Beads Context" section.

## Phase 1: Haiku Agent Scoping

Scoping is determined by two axes:

### Direction
| Value | Haiku agents skip |
|-------|-------------------|
| `backend-only` | frontend agent |
| `frontend-only` | backend, migration agents |
| `full-stack` (default) | nothing |

Sonnet infers direction from the feature description. If ambiguous, defaults to `full-stack`.

### Depth (number of agents)
| Depth | When | Agents |
|-------|------|--------|
| 2 | Most features | Backend arch + patterns, Frontend arch + test patterns |
| 3 | New domain or cross-cutting | + Cross-cutting (auth, types, middleware, migrations) |
| 4 | Major feature touching multiple domains | + Related domains (integration point check) |

### Agent Prompts

Each Haiku agent receives:
- A focused directory scope (one layer)
- Specific patterns to find (types, interfaces, converters, test fixtures)
- Instruction to return: file path, line number, one-line summary
- `model: "haiku"` — never Sonnet or Opus

## Phase 2: Sonnet Synthesis

### Option A (default): 1 Sonnet agent
Reads all Haiku findings, produces the full context brief.

### Option B (upgrade): 3 Sonnet agents
- Agent 1: Backend sections of brief
- Agent 2: Frontend sections of brief
- Agent 3: Cross-cutting sections (auth, cascade rules, gotchas)

Outputs are merged into the same brief format before Opus sees them. Upgrade path requires no changes to Phase 1 (Haiku) or Phase 3 (Opus).

## Context Brief Format

```markdown
# Context Brief: <feature name>

## Beads Context
- Related open issues: <list or "none">
- Memory hits: <list or "none">
- Active in-progress work that overlaps: <list or "none">

## Files to Touch
| Action | File | Layer | Why |
|--------|------|-------|-----|
| CREATE | ... | business | ... |
| MODIFY | ... | app | ... |

## Pattern to Follow
Closest existing feature: **<domain>** (<path>)
- Key pattern references (file:line)

## Cascade Rules
- New field on model → DB struct + converters + SQL + app DTO
- New filter field → bus/filter.go + store/filter.go + app/filter.go
- New enum value → business/types/<enum>/ + migration CHECK constraint

## Auth Checklist
- [ ] Routes registered with auth middleware via Routes.Add()
- [ ] X-API-Key header pattern followed (not written from scratch)
- [ ] Sidecar auth alignment verified if feature touches backend→sidecar calls
- [ ] No unauthenticated endpoints introduced unintentionally
- Existing auth reference: app/sdk/mid/auth.go

## Test Approach
- Store tests: business/sdk/dbtest (real Postgres, no mocks)
- API tests: app/sdk/apitest
- Frontend unit tests: web/src/**/__tests__/ (Vitest)
- Frontend component tests: web/src/views/__tests__/ or web/src/components/__tests__/
- Closest existing fixtures: <file paths from Haiku>

## Gotchas
- <codebase-specific traps found by Haiku>
```

## Phase 3: Opus Planning

Opus reads ONLY the context brief — no raw file reads during planning.
Invokes `writing-plans` skill to produce the implementation plan.
The brief is the source of truth; Opus reasons and decides, does not explore.

## Phase 4: Beads Issue Creation

After `writing-plans` completes, auto-create beads issues from the plan steps:

```bash
bd create --title="<step title>" --description="<step description>" --type=task
bd dep add <issue-b> <issue-a>   # wire sequential dependencies
```

One issue per implementation step. Dependencies reflect the plan's ordering.

## Option B Upgrade Path

To switch from Option A to Option B:
1. Change Phase 2 from 1 Sonnet agent to 3 domain-specialist agents
2. Each agent fills its sections of the brief
3. Merge outputs before passing to Opus
4. No changes to Phase 0, 1, or 3

## Skill Implementation Notes

- This skill replaces the existing `plan` and `plan-feature` skills
- Implemented as `~/.claude/skills/plan/SKILL.md` — since `/plan` already exists, update it in place rather than creating a new skill
- Sonnet determines direction + depth before dispatching Haiku — Opus is not involved in that decision
- If beads pre-check finds an in-progress issue that exactly matches the feature, surface it to the user before proceeding
