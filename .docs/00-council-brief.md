# Planner — Design Brief

## What it is
A personal intelligence layer over the real data of one person's life — a **single surface** for tasks, calendar, knowledge, habits, and (eventually) finances. Conversation-first: you talk, the system captures structure. Self-hosted, single-user, no sharing.

Think "life dashboard," not "task manager." Tasks, events, notes, transactions, and recurring logs all compose through **contexts** (ongoing situations) and **tags** rather than per-domain tables.

## Core principles
- **Privacy first** — self-hosted; sensitive data never leaves the box. Claude CLI runs locally via Claude Max. Local Ollama for sanitization + low-tier inference.
- **Capture over maintenance** — speak it, done. No forms.
- **Context over tasks** — tasks emerge from contexts, not the reverse.
- **Ground truth over intention** — privilege calendar events, transactions, received email over self-reported intent.
- **Auto-classify, manual override** — Claude tags and classifies at capture; user corrects rarely.
- **Composable primitives, not domain tables** — no "health" or "people" tables; notes + tags + recurring tasks + activity logs + contexts cover any domain.

## What it is NOT
- Not a team tool, not multi-tenant, no auth beyond a static API key.
- Not a calendar replacement (it eventually *is* the calendar, but read-only of external).
- Not a note-taking app (notes capture facts/preferences/ideas for recall, not authoring).
- Not an automation platform (data sources only, not Zapier-style productivity glue).

## Architecture
- **Backend:** single Go binary, three layers (`app → business → store`), each layer owns its types with explicit converters. Domain trees: `app/domain/<x>app`, `business/domain/<x>bus`, `business/domain/<x>bus/stores/<x>db`.
- **DB:** PostgreSQL (Docker, port 5433 local). pgvector for embeddings.
- **Frontend:** Vue 3 + Vite + Pinia, single codebase, web sidebar shell + mobile tab-bar shell via `useShell()`. PWA installed; Capacitor deferred.
- **Two API surfaces:**
  - `POST /mcp` — JSON-RPC 2.0 for Claude (capture, query, planning).
  - `/api/v1/...` — REST for the Vue frontend.
- **Ingestion pipeline (`ingestbus`)** — raw inputs → extraction (`ClaudeCodeExtractor`) → sanitization → entity creation → embedding → clarification fanout.
- **Sidecar** — separate Claude Code process on the VPS for orchestration, double-envelope JSON unwrapping.
- **AI model layer** — `Inferencer`/`Embedder`/`ModelRouter` interfaces with sensitivity-tier routing (e.g., transactions → local Ollama only).

## Data model (top-level)
- **contexts** — `project` (closeable) or `area` (ongoing). Have summary, debrief, outcome.
- **tasks** — `open | blocked | done | dismissed`. Optional context, recurrence, dependencies (auto-blocking), duration, due/scheduled times.
- **events** — fixed commitments; constrain daily plan.
- **notes** — knowledge capture, tagged, optionally context-linked.
- **daily_plans / daily_plan_items** — AI-generated grouped task list with override tracking.
- **time_blocks** — manual or AI-scheduled task slots in the weekly calendar.
- **transactions** — bank CSV import, sanitized via local Ollama.
- **thread_entries** — unified updates across any subject.
- **clarification_items** — swipeable review deck of unresolved questions (no push notifications).
- **outcome_observations / activity_logs / entity_links** — pattern data, habit tracking, cross-entity relationships.

## Current state (Phase status)
- ✅ **Phases 1–4, 4c, 7a (mostly), 7b, 7b.1, 7c, 7d** complete: tasks, contexts, web + PWA shell, clarifications, daily plan, weekly calendar + time blocks, recurring tasks, notes, activity logs, entity links.
- ⚠️ **Phase 3 (email)** — code built; SMTP disabled by default, needs MX/DNS + extraction prompt tuning.
- ✅ **Phase 5 (transactions)** — CSV import + tier-routed enrichment shipped.
- ⚠️ **Phase 6 (semantic search)** — embeddings + `search_semantic` MCP tool live; SKILL.md guidance pending.
- 🔜 **Phase 5b** (pattern recognition), **Phase 8** (Python ML service), **Phase 9** (intent framework — agentic adapter system) not yet started.

## Design rules of thumb (when advising)
- Three-layer separation is sacred — never let store types leak to app, never let app types leak to business.
- New field = update business model + DB struct/converters + SQL + app DTO/converters together.
- Composition over new tables — before proposing a new entity, ask whether tags + notes + a context can carry it.
- Clarification queue, not push notifications — the system accumulates questions; the user pulls.
- Tier routing is a first-class concern — anything touching transactions, raw email bodies, or PII must respect sensitivity tiers.
- Self-hosted constraints matter — no external SaaS dependencies, no multi-user assumptions, no shared infra patterns.
