# Planning System Design

## Goal

A living planning system for the planner app that keeps `.docs/` in sync with the codebase, enables high-level feature conversations through Claude Code skills, and reduces token cost through structured formatting and on-demand loading.

## Problem

The planner app has 12 planning docs (~150KB, ~40-50k tokens) covering vision, architecture, data model, roadmap, and feature designs. These docs:
- Are too large to load into every session
- Drift from the codebase as features are built
- Are written in prose optimized for human reading, not LLM consumption
- Have no mechanism for evolving through conversation

## Design

### 1. CLAUDE.md Planning Index

A new section in CLAUDE.md (~40-50 lines) loaded into every session. Contains:
- App identity: one sentence (personal intelligence layer, single-user, conversation-first)
- Current roadmap phase + what's built vs. what's next
- Doc index: each `.docs/` file with one-line description
- Skill pointers: lists `/plan`, `/plan-feature`, `/plan-audit`, `/plan-status`

Manually maintained. `/plan-audit` flags when stale.

Example format:
```markdown
## Planner App Context

Personal intelligence layer — conversation-first task/context management, single-user, self-hosted.

**Current phase:** 2 (Contexts) — complete. Next: Phase 3 (Email ingestion) or Phase 4 (Frontend).

**Built:** tasks, contexts, context events, tags (CRUD + MCP), REST API, health checks, PostgreSQL, Docker Compose.
**Not built:** email ingestion, frontend, transactions, scheduling, semantic search, ML service, intent framework.

**Planning docs** (`.docs/`):
- `01-vision.md` — principles, success criteria, what this is/isn't
- `02-architecture.md` — system components, request flows, tech stack
- `03-data-model.md` — all table schemas, indexing, relationships
- `04-ingestion-pipeline.md` — source interface, sensitivity tiers, 9-stage pipeline
- `05-context-engine.md` — context lifecycle, scheduling philosophy
- `06-infrastructure.md` — server, Docker, nginx, DNS, deployment
- `07-roadmap.md` — phases 1-9 with done criteria
- `08-ai-model-layer.md` — Inferencer/Embedder interfaces, model router, RAG
- `09-frontend.md` — Vue shells, components, Pinia stores, Capacitor
- `10-clarification-patterns.md` — clarification queue, card types
- `11-feedback-loop.md` — thread system, debriefs, pattern recognition
- `12-intent-framework.md` — intent recognition, slot filling, adapters

**Architecture maps** (`.docs/arch/`): task-backend.md, context-backend.md, tag-backend.md, mcp-backend.md, check-backend.md

**Planning skills:** `/plan` (brainstorm), `/plan-feature <name>` (directed planning), `/plan-audit` (drift check), `/plan-status` (overview)
```

### 2. TOC.md Lookup Index

New file: `.docs/TOC.md`. The skill entry point for finding information without loading full docs.

Three lookup dimensions:
- **By domain** — task, context, email, transaction, etc. → file#section references
- **By concept** — sensitivity-tiers, scheduling, mcp-tools, etc. → file#section
- **By schema** — each table name → file#section where its DDL lives

Skills read TOC.md first, then load only the referenced sections. Skills resolve references by reading the target file and extracting content under the matching `##` heading through the next heading of equal or higher level. Maintained by `/plan-audit`.

Example format:
```markdown
# TOC

## By Domain
- task: `03-data-model.md#tasks`, `07-roadmap.md#phase-1`
- context: `03-data-model.md#contexts`, `05-context-engine.md#context-lifecycle`, `07-roadmap.md#phase-2`
- email: `04-ingestion-pipeline.md#smtp-receiver`, `03-data-model.md#emails`, `07-roadmap.md#phase-3`
- transaction: `03-data-model.md#transactions`, `04-ingestion-pipeline.md#csv-transaction-importer`

## By Concept
- sensitivity-tiers: `04-ingestion-pipeline.md#sensitivity-tiers`
- scheduling: `05-context-engine.md#scheduling`
- mcp-tools: `05-context-engine.md#mcp-tools`
- model-routing: `08-ai-model-layer.md#model-router`

## By Schema
- tasks: `03-data-model.md#tasks`
- contexts: `03-data-model.md#contexts`
- context_events: `03-data-model.md#context-events`
- raw_inputs: `03-data-model.md#raw-inputs`
- emails: `03-data-model.md#emails`
- embeddings: `08-ai-model-layer.md#vector-storage`
```

### 3. Doc Compression

Rewrite all 12 `.docs/` files to a structured, token-efficient format.

Rules:
- 1-2 sentence summary at top of each file
- Key-value pairs and tables over prose paragraphs
- Bullet lists over sentences, no filler words
- Code blocks only for actual schemas/interfaces, not illustrative examples
- No ASCII art — terse bullet descriptions of relationships
- No philosophy preambles — just what the system does and how
- Each `##` section is self-contained (readable without prior sections)
- Section anchors match TOC.md references

What gets cut:
- "Why" explanations when "what" is self-evident
- Examples that duplicate schema definitions
- Information repeated across docs (single source of truth per fact)
- Future speculation not affecting current design

What stays:
- All table schemas (DDL)
- Interface definitions (Go interfaces)
- Hard constraints (tier rules, pipeline stage order)
- Cross-doc references

Target: ~150KB → ~60-70KB (40-60% reduction).

### 4. Skills

Four skills in `.claude/skills/`. The existing root `SKILL.md` (Claude MCP task-tracking skill) stays at the repo root — it serves a different purpose (runtime MCP tool guidance for Claude conversations, not Claude Code development skills).

#### `/plan` — Brainstorming

Purpose: Open-ended conversation about the app's direction.

Behavior:
1. Reads `TOC.md` + CLAUDE.md index
2. Loads relevant `.docs/` sections on demand as topics arise
3. Claude engages as thinking partner — pushes back, asks questions, proposes alternatives, references existing docs
4. Does NOT update docs unless explicitly asked ("ok write that down", "update the roadmap")
5. Can segue into `/plan-feature` when direction solidifies

Token budget: ~3,000-8,000 (TOC + 1-3 doc sections loaded incrementally).

#### `/plan-feature <name>` — Directed Feature Planning

Purpose: Concrete planning for a specific feature. Produces doc updates + implementation plan.

Behavior:
1. Reads `TOC.md` to find sections relevant to `<name>`
2. Loads only those sections + relevant `.docs/arch/` file
3. Guided conversation: requirements, constraints, trade-offs for this feature
4. Updates relevant `.docs/` files in place
5. Updates `TOC.md` if new sections added
6. Updates CLAUDE.md index if phase status changes
7. Invokes the `superpowers:writing-plans` skill (available via the superpowers plugin) to create a step-by-step implementation plan saved to `docs/superpowers/plans/`

Token budget: ~5,000-12,000.

#### `/plan-audit` — Drift Detection

Purpose: Cross-reference docs against codebase, surface mismatches.

Behavior:
1. Reads all `.docs/` files + scans codebase
2. Checks: tables in migration SQL vs. `03-data-model.md`, routes in code vs. arch files, roadmap items marked "not built" that exist, stale TOC entries, arch file freshness
3. Presents drift report
4. Asks before changing anything (docs represent intent — code might be wrong)
5. Can update docs, TOC, CLAUDE.md index, and arch files

Codebase scan is bounded to: `**/route.go` (routes), `business/sdk/migrate/sql/*.sql` (schemas), `business/domain/*/model.go` (business models), `app/domain/*/model.go` (app models). Does not read full handler or store implementations.

Token budget: ~35,000-50,000 (all compressed docs + bounded codebase scan).

#### `/plan-status` — Quick Orientation

Purpose: Read-only overview of where things stand.

Behavior:
1. Reads CLAUDE.md index
2. Fast codebase check (migration tables, route count, existing domains)
3. Reports: current phase, what's built, what's next, any known drift
4. No doc modifications

Token budget: ~1,000.

### 5. Skill Interaction Flow

```
Routine coding → CLAUDE.md index (always loaded, ~500 tokens)
    ↓ need orientation?
/plan-status (read-only, ~1k tokens)
    ↓ want to explore?
/plan (brainstorm, incremental loading, ~3-8k tokens)
    ↓ direction solidified?
/plan-feature <name> (updates docs + creates plan, ~5-12k tokens)
    ↓ code ships?
/plan-audit (resync docs ↔ code, ~35-50k tokens)
```

### 6. Token Budget Summary

| Mode | What's loaded | Estimated tokens |
|------|---------------|-----------------|
| Routine coding | CLAUDE.md index only | ~500 |
| `/plan-status` | Index + fast codebase scan | ~1,000 |
| `/plan` | TOC + 1-3 doc sections on demand | ~3,000-8,000 |
| `/plan-feature` | TOC + relevant sections + arch file | ~5,000-12,000 |
| `/plan-audit` | TOC + all docs + bounded codebase scan | ~35,000-50,000 |

### 7. Deliverables

1. Rewrite all 12 `.docs/` files to compressed format
2. Create `.docs/TOC.md` with domain/concept/schema indexes
3. Add planning context section to `CLAUDE.md`
4. Create `.claude/skills/plan/SKILL.md`
5. Create `.claude/skills/plan-feature/SKILL.md`
6. Create `.claude/skills/plan-audit/SKILL.md`
7. Create `.claude/skills/plan-status/SKILL.md`
8. Fix stale path in CLAUDE.md: `docs/arch/` → `.docs/arch/` (CLAUDE.md currently references the old location)
9. After compression, verify total `.docs/` size is under 75KB
