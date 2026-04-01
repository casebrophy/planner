# Planning System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Compress .docs/ planning files for token efficiency, create a TOC lookup index, add a CLAUDE.md planning section, and build four Claude Code skills for brainstorming, feature planning, drift auditing, and status checking.

**Architecture:** All deliverables are markdown files — no code changes. Skills are prompt-based (SKILL.md files in `.claude/skills/`). Docs are rewritten in-place to a structured, token-efficient format. A TOC.md file acts as a lookup index so skills load only relevant sections.

**Tech Stack:** Markdown, Claude Code skills (YAML frontmatter + prompt body)

---

## File Structure

### Files to create:
- `.docs/TOC.md` — lookup index (by domain, concept, schema)
- `.claude/skills/plan/SKILL.md` — brainstorming skill
- `.claude/skills/plan-feature/SKILL.md` — directed feature planning skill
- `.claude/skills/plan-audit/SKILL.md` — drift detection skill
- `.claude/skills/plan-status/SKILL.md` — quick status overview skill

### Files to modify:
- `CLAUDE.md` — add planning context section, fix `.docs/arch/` path
- `.docs/01-vision.md` — compress
- `.docs/02-architecture.md` — compress
- `.docs/03-data-model.md` — compress
- `.docs/04-ingestion-pipeline.md` — compress
- `.docs/05-context-engine.md` — compress
- `.docs/06-infrastructure.md` — compress
- `.docs/07-roadmap.md` — compress
- `.docs/08-ai-model-layer.md` — compress
- `.docs/09-frontend.md` — compress
- `.docs/10-clarification-patterns.md` — compress
- `.docs/11-feedback-loop.md` — compress
- `.docs/12-intent-framework.md` — compress

---

### Task 1: Compress 01-vision.md

**Files:**
- Modify: `.docs/01-vision.md`

Current size: 71 lines. Target: ~30-35 lines.

- [ ] **Step 1: Read current file and compress**

Rewrite `.docs/01-vision.md` following these rules:
- 1-2 sentence summary at top
- Bullet lists, no prose paragraphs
- Cut "design philosophy" preambles
- Cut "what not to build" (that's roadmap territory)
- Keep: core principles, success criteria, single-user constraint

The compressed file should look like:

```markdown
# Vision

Personal intelligence layer over the real data of your life. Conversation-first — you talk, the system captures structure. Not another task manager.

## Problem
- **Capture failure** — things fall through cracks, no frictionless recording
- **Context failure** — tasks recorded without surrounding information that makes them actionable

## What this is NOT
- Not a team tool (single-user, no sharing)
- Not a calendar replacement (connects to calendar, doesn't replace it)
- Not a note-taking app (notes serve tasks/contexts only)
- Not an automation platform (connects to data sources, not other productivity tools)

## Core Principles
- **Privacy first** — self-hosted, no third-party raw data access, only Anthropic API gets structured prompts
- **Capture over maintenance** — near-zero maintenance, add by speaking, no form fields
- **Context over tasks** — tasks emerge from contexts, not the reverse
- **Ground truth over intention** — privilege financial data, calendar events, received email over intention-based records
- **Extensibility** — new data sources via adapter interface without core restructuring

## Success Criteria
1. Say it out loud → tracked with context, no other action
2. "What do I need to do about X?" → relevant tasks + context + schedule
3. Forward email with deadline → task auto-created, linked, scheduled
4. Relevant transaction appears → auto-associated to context
5. "Plan my week" → realistic schedule from calendar + tasks + priorities
```

- [ ] **Step 2: Verify the file is under 40 lines and all key facts preserved**

Check: principles list complete, success criteria complete, constraints stated.

- [ ] **Step 3: Commit**

```bash
git add .docs/01-vision.md
git commit -m "docs: compress 01-vision.md for token efficiency"
```

---

### Task 2: Compress 02-architecture.md

**Files:**
- Modify: `.docs/02-architecture.md`

Current size: 223 lines. Target: ~90-100 lines.

- [ ] **Step 1: Read current file and compress**

Key cuts:
- Remove ASCII diagram — replace with terse bullet description of component relationships
- Remove full nginx config examples (that's `06-infrastructure.md`)
- Remove detailed request flow prose — convert to numbered steps
- Keep: component list, tech choices table, subdomain routing, security model

Structure:
```markdown
# Architecture

Single Go binary + Vue 3 frontend + SQLite, self-hosted via Docker Compose with nginx reverse proxy.

## Components
- **MCP server** (`POST /mcp`) — Claude's interface, JSON-RPC 2.0
- **REST API** (`/api/v1/...`) — frontend interface, standard JSON
- **Ingestion pipeline** — internal, processes raw data from source adapters
- **Intent framework** — agentic layer, recognises goals, fills slots, executes via adapters (see `12-intent-framework.md`)
- **AI model layer** — routes inference/embedding by sensitivity tier (see `08-ai-model-layer.md`)

## Source Adapters
Implement `Source` interface → feed ingestion pipeline via `RawInput`.

Built: (none yet — voice is indirect via Claude MCP)
Planned: SMTP receiver (email), CSV importer (transactions), receipt capture (photo)
Future: calendar (read-only), Apple Health

## Frontend
- Vue 3 + Vite, single codebase, two shells (web sidebar + mobile tab bar)
- Capacitor for native iOS
- See `09-frontend.md` for full component architecture

## Subdomain Routing
| Subdomain | Target |
|-----------|--------|
| app.domain.com | Vue frontend (port 5173/static) |
| api.domain.com | Go backend (port 8080) |
| mail.domain.com | SMTP receiver (port 25/587) |

## Request Flows

**Voice → task:** Siri shortcut → Claude API → reads SKILL.md → calls create_task via POST /mcp → Go validates + writes DB → Claude confirms

**Email → context update:** Forward email → SMTP stores raw → pipeline extracts → Claude identifies context + tasks + deadlines → writes entities → optional notification

**Transaction → context:** Upload CSV → parser normalises → pipeline processes → Claude matches to contexts → unmatched flagged for review

**Planning:** User asks Claude → list_tasks + list_contexts via MCP → reads calendar (when available) → reasons across data → proposes prioritised plan

## Security
- Single static API key (`X-API-Key` header or `Authorization: Bearer`)
- TLS via nginx, internal Docker traffic plain HTTP
- Rate limiting on `/mcp`
- No multi-tenancy, no user accounts
- Claude receives structured/summarised data, not raw PII

## Tech Choices

| Layer | Choice | Reason |
|-------|--------|--------|
| Backend | Go | Fast, single binary, low memory |
| Database | SQLite (WAL mode) | No service to manage, single-user sufficient |
| Vector search | sqlite-vec | Adds vectors to existing DB |
| MCP transport | Streamable HTTP | Claude connector standard |
| External inference | Anthropic API | Tier 1 + promoted Tier 2 |
| Local inference | Ollama | Tier 2 sanitization + Tier 3 |
| Local embeddings | nomic-embed-text via Ollama | 768-dim, fast, private |
| Frontend | Vue 3 + Vite | Fast dev, component-based |
| Mobile | Capacitor | Wraps Vue as native iOS |
| State | Pinia | Simple, Vue 3 integrated |
| Reverse proxy | nginx | TLS, rate limiting |
| Container | Docker Compose | Right complexity for personal app |
| ML service | Python + FastAPI | ML ecosystem, isolated |
```

- [ ] **Step 2: Verify under 110 lines, tech table complete, all components listed**

- [ ] **Step 3: Commit**

```bash
git add .docs/02-architecture.md
git commit -m "docs: compress 02-architecture.md for token efficiency"
```

---

### Task 3: Compress 03-data-model.md

**Files:**
- Modify: `.docs/03-data-model.md`

Current size: 315 lines. Target: ~160-180 lines.

- [ ] **Step 1: Read current file and compress**

Key cuts:
- Remove prose explanations of fields — the DDL + comments are sufficient
- Remove "design philosophy" section — keep entity relationship as terse bullets
- Remove "what Claude sees" table (move to `08-ai-model-layer.md` if not there)
- Remove "evolution notes" section — nullable columns are self-documenting
- Keep: ALL DDL exactly as-is, entity relationships, index strategy, thread/feedback additions

Structure:
```markdown
# Data Model

Three top-level concepts: **contexts** (ongoing situations), **tasks** (discrete actions), **sources** (external data).

## Entity Relationships
- contexts → tasks (one-to-many, optional), context_events (timeline), raw_inputs, tags (many-to-many)
- tasks → context (optional parent), thread_entries (log), time_blocks, tags (many-to-many)
- raw_inputs → emails, transactions (future source types)

## Tables

### contexts
{DDL as-is}

### context_events
{DDL as-is, keep kind enum comment}

### tasks
{DDL as-is, keep field comments for energy, duration_min, scheduled_at}

### thread_entries
{DDL as-is}

### time_blocks
{DDL as-is}

### raw_inputs
{DDL as-is}

### emails
{DDL as-is}

### transactions
{DDL as-is, keep amount=cents note}

### tags / task_tags / context_tags
{DDL as-is}

## Indexes
{All CREATE INDEX statements}

## Thread & Feedback Additions
{ALTER TABLE statements and new tables: context_outcomes}
```

- [ ] **Step 2: Verify all DDL preserved exactly, under 180 lines**

- [ ] **Step 3: Commit**

```bash
git add .docs/03-data-model.md
git commit -m "docs: compress 03-data-model.md for token efficiency"
```

---

### Task 4: Compress 04-ingestion-pipeline.md

**Files:**
- Modify: `.docs/04-ingestion-pipeline.md`

Current size: 267 lines. Target: ~120-140 lines.

- [ ] **Step 1: Read current file and compress**

Key cuts:
- Remove prose explanations of pipeline stages — convert to numbered list with one-line descriptions
- Remove example regex patterns for PII detection (implementation detail, not design)
- Condense source adapter descriptions
- Keep: Source interface (Go code), RawInput struct, sensitivity tier rules, pipeline stages, extraction schemas, error handling rules

Structure:
```markdown
# Ingestion Pipeline

Every data source → `RawInput` → 9-stage pipeline. Tier classification determines which model handles each stage.

## Source Interface
{Go interface code block — Source, EmitFunc, RawInput, SensitivityTier}

## Sensitivity Tiers
| Tier | Rule | Examples |
|------|------|----------|
| 1 — API permitted | No PII in raw form | Receipts, task content, voice captures |
| 2 — Local then API | Contains PII, sanitize locally first | Bank CSV, emails, credit card exports |
| 3 — Fully local | Never leaves server | Health data, user-flagged |

Default tier by source table.
Classifier can promote (never demote). Tier 2/3 permanent once assigned.

## Pipeline Stages
1. **Store raw input** — write to raw_inputs with status=pending (non-negotiable first step)
2. **Classify tier** — local regex scan for PII patterns, promote Tier 1→2 if found
3. **Extract** — model by tier: Tier 1→API, Tier 2→local, Tier 3→local
4. **Sanitize & promote** (Tier 2 only) — log to sanitization_log, block if PII re-detected
5. **Route to context** — match against active contexts; high confidence=auto, low=flag for review
6. **Write entities** — context_events, tasks, emails/transactions records
7. **Embed chunks** — tier-appropriate embeddings model
8. **Update context summary** — rewrite contexts.summary via tier-appropriate inferencer
9. **Mark processed** — status→processed

## Extraction Schemas
{Email extraction JSON schema}
{Transaction extraction JSON schema}

## Source Adapters (v1)
- **SMTP receiver** — `emersion/go-smtp`, default Tier 2
- **CSV importer** — bank/CC exports, always Tier 2, dedup by (source, date, description, amount)
- **Receipt importer** — photo/text, default Tier 1, promoted if card numbers found

## Error Handling
- Stage 1 failure → input dropped, logged (only unrecoverable failure)
- Stage 2-8 failure → record stays pending/failed, exponential backoff retry (cap 1hr)
- Local model unavailable → queue and wait (never fallback to API for Tier 2/3)
- API unavailable → Tier 1 queues and retries
- PII re-detected at promotion → flagged for manual review (never silently promoted)
```

- [ ] **Step 2: Verify tier rules complete, pipeline stages all 9 present, Go interface preserved**

- [ ] **Step 3: Commit**

```bash
git add .docs/04-ingestion-pipeline.md
git commit -m "docs: compress 04-ingestion-pipeline.md for token efficiency"
```

---

### Task 5: Compress 05-context-engine.md

**Files:**
- Modify: `.docs/05-context-engine.md`

Current size: 156 lines. Target: ~80-90 lines.

- [ ] **Step 1: Read current file and compress**

Key cuts:
- Remove prose about "emergent behaviour" — just state what the operations are
- Condense scheduling section — keep algorithm, cut conversational examples
- Keep: context operations, summary rewrite rules, lifecycle states, scheduling inputs/algorithm, MCP tool list

- [ ] **Step 2: Verify context lifecycle, scheduling algorithm, MCP tools all present, under 90 lines**

- [ ] **Step 3: Commit**

```bash
git add .docs/05-context-engine.md
git commit -m "docs: compress 05-context-engine.md for token efficiency"
```

---

### Task 6: Compress 06-infrastructure.md

**Files:**
- Modify: `.docs/06-infrastructure.md`

Current size: 270 lines. Target: ~100-120 lines.

- [ ] **Step 1: Read current file and compress**

Key cuts:
- Remove full nginx config blocks — keep only the routing summary table
- Remove certbot commands (operational, not design)
- Condense Docker Compose — keep service list with ports/volumes, cut full YAML
- Keep: DNS config, Docker services table, env vars, deployment workflow, backup strategy, MCP connector registration

- [ ] **Step 2: Verify DNS records, Docker services, env vars all present, under 120 lines**

- [ ] **Step 3: Commit**

```bash
git add .docs/06-infrastructure.md
git commit -m "docs: compress 06-infrastructure.md for token efficiency"
```

---

### Task 7: Compress 07-roadmap.md

**Files:**
- Modify: `.docs/07-roadmap.md`

Current size: 387 lines. Target: ~150-170 lines.

- [ ] **Step 1: Read current file and compress**

Key cuts:
- Each phase: goal (1 sentence), key deliverables (bullet list), done-when (1 sentence)
- Remove "what you can do" narrative sections — they duplicate deliverables
- Remove detailed component lists that are covered in dedicated docs
- Keep: phase ordering, dependencies between phases, deferred items list

Structure per phase:
```markdown
## Phase N — Name
**Goal:** One sentence.
**Deliverables:**
- item 1
- item 2
**Done when:** One sentence.
```

- [ ] **Step 2: Verify all 9 phases + 9b present, deferred list present, under 170 lines**

- [ ] **Step 3: Commit**

```bash
git add .docs/07-roadmap.md
git commit -m "docs: compress 07-roadmap.md for token efficiency"
```

---

### Task 8: Compress 08-ai-model-layer.md

**Files:**
- Modify: `.docs/08-ai-model-layer.md`

Current size: 382 lines. Target: ~150-170 lines.

- [ ] **Step 1: Read current file and compress**

Key cuts:
- Remove prose explanations of router logic — the Go code is self-documenting
- Remove Ollama Docker Compose snippet (that's `06-infrastructure.md`)
- Remove SKILL.md guidance section (that's SKILL.md territory)
- Condense RAG section — keep what gets embedded table, retrieval pipeline steps, re-ranking formula
- Keep: ALL Go interfaces (Inferencer, Embedder, ModelRouter, InferRequest/Response), concrete implementation configs, router logic, vector storage DDL, MCP tool schema

- [ ] **Step 2: Verify all Go interfaces preserved, router rules complete, embedding DDL present, under 170 lines**

- [ ] **Step 3: Commit**

```bash
git add .docs/08-ai-model-layer.md
git commit -m "docs: compress 08-ai-model-layer.md for token efficiency"
```

---

### Task 9: Compress 09-frontend.md

**Files:**
- Modify: `.docs/09-frontend.md`

Current size: 258 lines. Target: ~110-130 lines.

- [ ] **Step 1: Read current file and compress**

Key cuts:
- Remove prose descriptions of shells — convert to terse bullet lists
- Condense component descriptions — name + one-line purpose
- Remove Capacitor plugin table (implementation detail)
- Keep: two-shell architecture, navigation structures, shared component list, Pinia store list, route table, api.ts structure, build commands

- [ ] **Step 2: Verify component list complete, route table complete, store list complete, under 130 lines**

- [ ] **Step 3: Commit**

```bash
git add .docs/09-frontend.md
git commit -m "docs: compress 09-frontend.md for token efficiency"
```

---

### Task 10: Compress 10-clarification-patterns.md

**Files:**
- Modify: `.docs/10-clarification-patterns.md`

Current size: 291 lines. Target: ~120-140 lines.

- [ ] **Step 1: Read current file and compress**

Key cuts:
- Remove ASCII card mockups — describe card anatomy as a field list
- Remove prose about "session model" — convert to bullets
- Remove pattern recognition layer prose — keep the three-layer summary table
- Keep: card generation triggers table, answer options table, card anatomy fields, session/ordering rules, "by the way" section description, pattern layers, surfacing locations, DDL for clarification_items and pattern_observations

- [ ] **Step 2: Verify all DDL preserved, trigger list complete, answer options table complete, under 140 lines**

- [ ] **Step 3: Commit**

```bash
git add .docs/10-clarification-patterns.md
git commit -m "docs: compress 10-clarification-patterns.md for token efficiency"
```

---

### Task 11: Compress 11-feedback-loop.md

**Files:**
- Modify: `.docs/11-feedback-loop.md`

Current size: 442 lines. Target: ~170-200 lines.

- [ ] **Step 1: Read current file and compress**

Key cuts:
- Remove prose about "turning time into knowledge" — cut preamble
- Remove ASCII card mockups for inactivity and debrief — describe as field lists
- Remove example JSON for inferred observations — keep the kind list only
- Condense debrief card descriptions — keep the adaptive framing rules as a table
- Keep: thread entry kinds list, thread extraction schema, inactivity threshold table, inactivity rules (what resets/doesn't reset clock), debrief trigger rules, ALL DDL (thread_entries, inactivity_checks, outcome_observations, ALTER TABLE statements), MCP tools

- [ ] **Step 2: Verify all DDL preserved, thread kinds complete, MCP tools listed, under 200 lines**

- [ ] **Step 3: Commit**

```bash
git add .docs/11-feedback-loop.md
git commit -m "docs: compress 11-feedback-loop.md for token efficiency"
```

---

### Task 12: Compress 12-intent-framework.md

**Files:**
- Modify: `.docs/12-intent-framework.md`

Current size: 348 lines. Target: ~140-160 lines.

- [ ] **Step 1: Read current file and compress**

Key cuts:
- Remove example intents/non-intents list — keep the distinction as a 1-sentence rule
- Remove full 6-step creation flow prose — convert to numbered steps with 1-line descriptions
- Remove plan generation prose — keep the 4 requirements as bullets
- Remove slot source emoji indicators (UI detail)
- Keep: Go Slot/FillRule structs, three-tier adapter descriptions, intent lifecycle steps, fill strategies list, crystallisation rules, ALL DDL (intent_adapters, intent_executions, workflow_creation_sessions), MCP tools, confirmation gate principle

- [ ] **Step 2: Verify Go structs preserved, all DDL preserved, MCP tools listed, under 160 lines**

- [ ] **Step 3: Commit**

```bash
git add .docs/12-intent-framework.md
git commit -m "docs: compress 12-intent-framework.md for token efficiency"
```

---

### Task 13: Create TOC.md

**Files:**
- Create: `.docs/TOC.md`

- [ ] **Step 1: Create the TOC file**

Read all compressed `.docs/` files. For each `##` heading, create an entry in the appropriate TOC section. Group by domain, concept, and schema.

```markdown
# TOC

Lookup index for `.docs/` planning files. Skills resolve `file#section` references by reading the target file and extracting content under the matching `##` heading through the next `##` of equal or higher level.

## By Domain
- task: `03-data-model.md#tasks`, `07-roadmap.md#phase-1`
- context: `03-data-model.md#contexts`, `03-data-model.md#context-events`, `05-context-engine.md#context-operations`, `05-context-engine.md#context-lifecycle`, `07-roadmap.md#phase-2`
- email: `03-data-model.md#emails`, `04-ingestion-pipeline.md#smtp-receiver`, `07-roadmap.md#phase-3`
- transaction: `03-data-model.md#transactions`, `04-ingestion-pipeline.md#csv-importer`, `07-roadmap.md#phase-5`
- tag: `03-data-model.md#tags`
- thread: `03-data-model.md#thread-entries`, `11-feedback-loop.md#task-threads`
- clarification: `10-clarification-patterns.md#clarification-queue`, `03-data-model.md#clarification-items`
- pattern: `10-clarification-patterns.md#pattern-recognition`, `11-feedback-loop.md#feedback-loop`
- intent: `12-intent-framework.md#intent-lifecycle`, `12-intent-framework.md#adapter-registry`
- scheduling: `05-context-engine.md#scheduling`, `03-data-model.md#time-blocks`
- frontend: `09-frontend.md#web-shell`, `09-frontend.md#mobile-shell`, `09-frontend.md#shared-components`

## By Concept
- sensitivity-tiers: `04-ingestion-pipeline.md#sensitivity-tiers`
- model-routing: `08-ai-model-layer.md#model-router`
- rag-search: `08-ai-model-layer.md#rag-semantic-search`
- context-summary: `05-context-engine.md#context-operations`
- crystallisation: `12-intent-framework.md#crystallisation`
- inactivity-detection: `11-feedback-loop.md#inactivity-detection`
- debrief: `11-feedback-loop.md#feedback-loop`
- source-adapters: `04-ingestion-pipeline.md#source-adapters`
- pipeline-stages: `04-ingestion-pipeline.md#pipeline-stages`
- auth: `02-architecture.md#security`
- privacy: `01-vision.md#core-principles`

## By Schema
- contexts: `03-data-model.md#contexts`
- context_events: `03-data-model.md#context-events`
- tasks: `03-data-model.md#tasks`
- thread_entries: `03-data-model.md#thread-entries`
- time_blocks: `03-data-model.md#time-blocks`
- raw_inputs: `03-data-model.md#raw-inputs`
- emails: `03-data-model.md#emails`
- transactions: `03-data-model.md#transactions`
- tags: `03-data-model.md#tags`
- clarification_items: `10-clarification-patterns.md#clarification-items`
- pattern_observations: `10-clarification-patterns.md#pattern-observations`
- outcome_observations: `11-feedback-loop.md#outcome-observations`
- inactivity_checks: `11-feedback-loop.md#inactivity-checks`
- intent_adapters: `12-intent-framework.md#adapter-registry`
- intent_executions: `12-intent-framework.md#adapter-registry`
- workflow_creation_sessions: `12-intent-framework.md#adapter-registry`
- embeddings: `08-ai-model-layer.md#vector-storage`
```

- [ ] **Step 2: Verify every `##` section in every `.docs/` file has a TOC entry in at least one dimension**

Cross-check by reading each compressed doc and confirming its sections appear in the TOC.

- [ ] **Step 3: Commit**

```bash
git add .docs/TOC.md
git commit -m "docs: create TOC.md lookup index for planning skills"
```

---

### Task 14: Verify compression target

- [ ] **Step 1: Check total size of .docs/ files**

```bash
cat .docs/*.md | wc -c
```

Expected: under 75,000 bytes (75KB).

If over 75KB, identify the largest files and apply further compression. The primary targets for additional cuts:
- `03-data-model.md` — can DDL comments be trimmed?
- `11-feedback-loop.md` — can example JSON be cut further?
- `08-ai-model-layer.md` — can Go interface docs be trimmed?

- [ ] **Step 2: If under target, move on. If over, apply targeted cuts and re-verify.**

---

### Task 15: Update CLAUDE.md

**Files:**
- Modify: `CLAUDE.md`

- [ ] **Step 1: Fix stale arch path**

In CLAUDE.md, find `docs/arch/` and replace with `.docs/arch/`.

- [ ] **Step 2: Add planning context section**

Add the following section at the end of CLAUDE.md:

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

- [ ] **Step 3: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: add planning context section to CLAUDE.md, fix arch path"
```

---

### Task 16: Create /plan skill

**Files:**
- Create: `.claude/skills/plan/SKILL.md`

- [ ] **Step 1: Create the skill file**

```markdown
---
name: plan
description: Open-ended brainstorming about the planner app's direction. Use when the user wants to discuss features, explore ideas, or think about what to build next. Loads planning context on demand — does NOT update docs unless explicitly asked.
---

# Planner Brainstorming

You are a thinking partner for evolving a personal task management app. The user wants to explore ideas, discuss trade-offs, and shape the app's direction through conversation.

## Setup

1. Read `.docs/TOC.md` to understand what planning docs exist
2. Read the "Planner App Context" section of `CLAUDE.md` for current state

## Behavior

- Engage as a thinking partner — push back on ideas, ask probing questions, propose alternatives
- When a topic comes up, use TOC.md to find the relevant `.docs/` section and read it
- Reference what the planning docs say vs. what the user is proposing
- Flag conflicts between the proposal and existing design decisions
- Suggest trade-offs and alternatives — don't just agree

## Rules

- **Do NOT update any docs** unless the user explicitly says "write that down", "update the roadmap", "save this to the docs", etc.
- **Do NOT create implementation plans** — that's `/plan-feature`
- **Load docs incrementally** — read TOC.md first, then only the sections relevant to the current topic
- Keep conversation flowing — don't dump large doc excerpts, summarize and reference

## When direction solidifies

If the conversation reaches a clear feature decision, suggest: "Want me to run `/plan-feature <name>` to make this concrete and create an implementation plan?"
```

- [ ] **Step 2: Commit**

```bash
git add .claude/skills/plan/SKILL.md
git commit -m "feat: create /plan brainstorming skill"
```

---

### Task 17: Create /plan-feature skill

**Files:**
- Create: `.claude/skills/plan-feature/SKILL.md`

- [ ] **Step 1: Create the skill file**

```markdown
---
name: plan-feature
description: Directed feature planning for the planner app. Use when the user has decided what to build and wants to make it concrete — updates planning docs and creates an implementation plan. Argument is the feature name (e.g., "email-ingestion", "frontend", "scheduling").
---

# Feature Planning

Plan a specific feature for the planner app. Updates the relevant `.docs/` planning files and produces an implementation plan.

## Setup

1. Read `.docs/TOC.md`
2. Search TOC for all entries matching the feature name argument
3. Read only the matched `.docs/` sections
4. Read the relevant `.docs/arch/` file if the domain already exists
5. Read the "Planner App Context" section of `CLAUDE.md` for current state

## Process

1. **Summarize what the docs say** about this feature — what's already designed, what's unspecified
2. **Guided conversation** — walk through requirements, constraints, and trade-offs with the user
3. **When aligned, update docs:**
   - Modify the relevant `.docs/` file sections in place (follow the compressed format — bullets, tables, no prose)
   - Add new `##` sections if needed
   - Update `.docs/TOC.md` if new sections were added
   - Update "Planner App Context" in `CLAUDE.md` if phase status changes
4. **Create implementation plan** — invoke the `superpowers:writing-plans` skill

## Doc update rules

- Follow the existing compressed format in each file (see any `.docs/` file for reference)
- Every new `##` section must be added to `TOC.md` in the appropriate dimension(s)
- Keep DDL exact and complete — no pseudo-SQL
- Keep Go interfaces exact and complete — no pseudocode
- One source of truth per fact — don't duplicate information across docs
```

- [ ] **Step 2: Commit**

```bash
git add .claude/skills/plan-feature/SKILL.md
git commit -m "feat: create /plan-feature directed planning skill"
```

---

### Task 18: Create /plan-audit skill

**Files:**
- Create: `.claude/skills/plan-audit/SKILL.md`

- [ ] **Step 1: Create the skill file**

```markdown
---
name: plan-audit
description: Cross-reference planning docs against the codebase to detect drift. Use after building features, or periodically, to keep docs in sync with reality. Surfaces mismatches and asks before changing anything.
---

# Planning Docs Audit

Cross-reference `.docs/` planning files against the actual codebase to find drift.

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

## Output

Present a drift report organized by check type. For each finding:
- What the doc says
- What the code says
- Suggested fix (doc update or code change)

## Rules

- **Ask before changing anything** — docs represent intent, not just reality. The code might be wrong.
- After user approves changes, update the relevant docs, TOC.md, and CLAUDE.md index
- If arch files need regeneration, suggest running `/go-arch` for the affected domains
```

- [ ] **Step 2: Commit**

```bash
git add .claude/skills/plan-audit/SKILL.md
git commit -m "feat: create /plan-audit drift detection skill"
```

---

### Task 19: Create /plan-status skill

**Files:**
- Create: `.claude/skills/plan-status/SKILL.md`

- [ ] **Step 1: Create the skill file**

```markdown
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

## Output format

```
Current phase: N (Name) — status
Built: [list of working features]
Not built: [list of planned but unimplemented features]
Next up: [1-2 most logical next steps]
Drift: [any obvious mismatches between CLAUDE.md and codebase, or "none detected"]
```

## Rules

- **Read-only** — do not modify any files
- Keep output concise — this is a quick check, not a full audit
- If significant drift is found, suggest running `/plan-audit` for a thorough review
```

- [ ] **Step 2: Commit**

```bash
git add .claude/skills/plan-status/SKILL.md
git commit -m "feat: create /plan-status overview skill"
```

---

### Task 20: Final verification

- [ ] **Step 1: Verify all files exist**

```bash
ls -la .docs/TOC.md
ls -la .claude/skills/plan/SKILL.md
ls -la .claude/skills/plan-feature/SKILL.md
ls -la .claude/skills/plan-audit/SKILL.md
ls -la .claude/skills/plan-status/SKILL.md
```

- [ ] **Step 2: Verify .docs/ size under 75KB**

```bash
cat .docs/*.md | wc -c
```

Expected: under 75,000 bytes.

- [ ] **Step 3: Verify CLAUDE.md has planning section and correct arch path**

```bash
grep "Planner App Context" CLAUDE.md
grep ".docs/arch/" CLAUDE.md
```

Both should return matches.

- [ ] **Step 4: Verify TOC.md references resolve**

Spot-check 3 TOC references by reading the target file and confirming the `##` heading exists:
- `03-data-model.md#tasks` — should have `## tasks` or `### tasks` heading
- `04-ingestion-pipeline.md#sensitivity-tiers` — should have matching heading
- `12-intent-framework.md#crystallisation` — should have matching heading

- [ ] **Step 5: Final commit if any fixes were needed**

```bash
git add -A
git commit -m "docs: planning system implementation complete"
```
