# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Environment

Create a `.env` file at repo root (Makefile auto-includes via `-include .env`):

| Variable | Value | Notes |
|----------|-------|-------|
| `PLANNER_DB_HOST` | `localhost` | |
| `PLANNER_DB_PORT` | `5433` | Docker maps Postgres to 5433, not 5432 |
| `PLANNER_DB_USER` | `planner` | |
| `PLANNER_DB_PASSWORD` | `planner` | |
| `PLANNER_DB_NAME` | `planner` | |
| `PLANNER_DB_DISABLE_TLS` | `true` | |
| `PLANNER_AUTH_API_KEY` | `devkey123` | Must match sidecar's key — see Sidecar section |

## Commands

```bash
# One-shot local dev (DB + migrate + API + Vite frontend)
make dev-up       # Start everything; Ctrl-C to stop
make dev-down     # Stop the dev database

# Git hooks (run once after cloning)
make install-hooks  # Installs pre-commit arch staleness check

# Backend only (requires DB running)
make dev

# Database
make db-up        # Start just the PostgreSQL container
make migrate      # Run migrations
make seed         # Seed with sample data

# Docker (full stack)
make up           # Start all containers
make down         # Stop all containers
make logs         # Tail backend logs
make logs-all     # Tail backend + frontend logs

# Testing and linting
make test         # go test ./... -count=1
make lint         # go vet ./...
go test ./business/domain/taskbus/... -run TestFuncName -count=1  # Single test

# Frontend
make frontend-dev     # Vite dev server (proxies /api to :8080)
make frontend-build   # Production build
make frontend-serve   # Build + serve via Go SPA server
make frontend-test    # Run frontend tests
make frontend-lint    # Lint frontend
make npm ARGS="..."   # Pass-through npm command

# Admin tooling
make admin ARGS=migrate
make admin ARGS=seed

# Secrets (SOPS + age encryption)
make secrets-edit     # Decrypt → edit in $EDITOR → re-encrypt
make secrets-show     # Print decrypted secrets to stdout
make secrets-add KEY=X VALUE=Y  # Add a secret

# Deployment
make deploy       # Run zarf/deploy/deploy.sh
make backup       # Run zarf/deploy/backup.sh (also runs on systemd timer)
```

## Architecture

Three-layer architecture: **app → business → store**. Each layer owns its own types; explicit conversion functions translate between layers.

```
api/services/planner/    # main.go — wire everything together (backend API on :8080)
api/services/frontend/   # Go SPA server — serves Vue dist + Cache-Control headers
api/tooling/admin/       # migration + seed CLI

app/domain/<name>app/    # HTTP handlers, request/response DTOs
  model.go               # App-layer structs + toApp*/toBus* converters
  <name>app.go           # Handler methods (create, update, delete, queryAll, queryByID)
  route.go               # Routes.Add() — registers endpoints, instantiates business + store
  filter.go              # parseFilter() — maps query params → QueryFilter
  order.go               # parseOrder() — maps request fields → business order constants

business/domain/<name>bus/   # Business logic, domain types, Storer interface
  model.go               # Business structs (NewX, UpdateX, X)
  <name>bus.go           # Business methods + Storer interface definition
  filter.go              # QueryFilter struct
  order.go               # OrderBy constants + DefaultOrderBy
  stores/<name>db/       # Store implementation
    model.go             # DB struct (db tags) + toDBX/toBusX converters
    <name>db.go          # Store methods (SQL queries)
    filter.go            # applyFilter() — builds WHERE clauses
    order.go             # orderByFields map + orderByClause()

business/types/          # Enum types (taskstatus, taskpriority, taskenergy, contextstatus,
                         #   contextkind, contextoutcome, debriefstatus, recurrence,
                         #   observationkind, clarificationkind, clarificationstatus,
                         #   rawinputsource, rawinputstatus, threadentrykind, threadsource)
business/sdk/            # Shared SDK: order, page, migrate, sanitize, sqldb
  unitest/               # Unit test helpers
  dbtest/                # DB integration test helpers (real Postgres)
foundation/web/          # HTTP framework: App, Handle(), HandlerFunc, Respond()
foundation/logger/       # Structured logger
foundation/claudecli/    # Claude Code CLI wrapper (used by sidecar)
foundation/docker/       # Docker helpers
foundation/otel/         # OpenTelemetry setup
app/sdk/errs/            # Error codes (InvalidArgument, NotFound, Internal, etc.) → HTTP status
app/sdk/mid/             # Middleware: auth (API key), logging, panics, errors
app/sdk/apitest/         # API integration test helpers
app/sdk/query/           # Query string parsing utilities

zarf/sidecar/            # Claude Code sidecar — see Sidecar section below
zarf/deploy/             # Deploy scripts, systemd units, VPS setup guide
zarf/compose/            # Docker Compose files
```

## Frontend

Vue 3 + TypeScript SPA in `api/services/frontend/web/src/`:

```
components/              # Reusable UI components
composables/             # Vue composables (useTaskNotes, etc.)
views/                   # Route-level views (TaskDetailView, ContextDetailView, etc.)
stores/                  # Pinia stores
services/                # API client layer
types/                   # TypeScript types
router/                  # Vue Router config
```

Frontend arch docs live in `.docs/arch/*-frontend.md`. Tests use Vitest (`make frontend-test`).

## Cross-layer Impact Rules

When modifying a domain, changes cascade across ALL layers. Always update together:

- **New field on a model**: update business model → DB struct + converters → SQL queries → app DTO + converters
- **New Storer method**: add to interface in `<name>bus.go` → implement in `stores/<name>db/<name>db.go`
- **New filter field**: `business/domain/<name>bus/filter.go` → `stores/<name>db/filter.go` (applyFilter) → `app/domain/<name>app/filter.go` (parseFilter)
- **New order field**: `business/domain/<name>bus/order.go` (constant) → `stores/<name>db/order.go` (SQL column) → `app/domain/<name>app/order.go` (request field name)
- **New enum value**: update `business/types/<enum>/` → database CHECK constraint in migration SQL

## Pre-reasoned Architecture Maps

`.docs/arch/` contains detailed dependency maps for every domain — backend (`<domain>-backend.md`) and frontend (`<domain>-frontend.md`). **Read the relevant arch file first** before modifying a domain — each file documents all types, file maps, impact callouts, routes, and cross-domain dependencies.

**IMPORTANT:** All docs live in `.docs/` (hidden, dot-prefixed). Do NOT create a `docs/` directory — it has been consolidated into `.docs/`.

## Key Patterns

**Error handling** — stores return `sqldb.ErrDBNotFound` (= `sql.ErrNoRows`) when a row isn't found. Handlers must check explicitly: `if errors.Is(err, sqldb.ErrDBNotFound) { return errs.New(errs.NotFound, err) }`. Unchecked, this surfaces as `errs.Internal` / 500.

**Handlers** implement `foundation/web.HandlerFunc` and return a `web.Encoder` (or `errs.Error`). Use `errs.New(errs.NotFound, err)` for not-found cases, `errs.New(errs.InvalidArgument, err)` for bad input.

**Enums** (taskstatus, taskpriority, etc.) are value types with `Parse()`, `MustParse()`, and text marshaling. Store layer converts to/from strings; business layer uses typed values.

**Pagination** uses `business/sdk/page.Page` (Number, RowsPerPage → Offset). **Ordering** uses `business/sdk/order.By` with field constant + direction.

**Auth** middleware (API key via `X-API-Key` header) is applied to all domain routes via `Routes.Add()`.

**Sanitize** — `business/sdk/sanitize` provides input sanitization for the ingestion pipeline. Used by `ingestbus` to clean extracted content.

## New Domain Checklist

New domains require files across all 3 layers. Follow the Cross-layer Impact Rules above, plus:
1. Migration SQL in `business/sdk/migrate/sql/`
2. Enum types if needed (`business/types/<enum>/`)
3. Business → Store → App layers (see architecture tree for file layout)
4. Wire in `api/services/planner/main.go`
5. Arch doc in `.docs/arch/<name>-backend.md`
6. Tests with `dbtest` (store) and `apitest` (API)

## Testing

Tests use real Postgres via `business/sdk/dbtest` (not mocks). Key packages:

- `business/sdk/unitest` — unit test helpers, test fixtures
- `business/sdk/dbtest` — spins up a test database, runs migrations, provides cleanup
- `app/sdk/apitest` — HTTP integration test helpers, builds a test server

```bash
make test                                                    # All tests
go test ./business/domain/taskbus/... -run TestCreate -count=1  # Single test
```

## Sidecar

`zarf/sidecar/` is a Claude Code sidecar process that runs alongside the backend on the VPS. It receives orchestration requests from the backend and dispatches them to Claude Code subagents.

**Double-envelope gotcha:** The sidecar uses `--output-format json`, so its response wraps the subagent output in a CLI envelope `{type:result, result:...}`. The backend's `runHTTP` must unwrap TWO envelopes: (1) sidecar's `{result:...}` and (2) the nested CLI envelope. Without both, `json.Unmarshal` silently produces zero-value structs.

**Auth alignment:** The sidecar systemd unit reads `PLANNER_AUTH_API_KEY` from `.env` directly, but the backend container gets `PLANNER_API_KEY` from decrypted secrets mapped to `PLANNER_AUTH_API_KEY` in `docker-compose.yml`. If these diverge, backend→sidecar proxy requests get 401.

## Deployment & Infrastructure

Deploy/backup/secrets commands are in the Commands section above. For VPS setup and systemd units, see `zarf/deploy/VPS-SETUP.md`.

## MCP / Skill Integration

This repo also serves as a personal task manager via MCP. `SKILL.md` defines a Claude skill that calls the running API at `http://localhost:8080/mcp`. The MCP transport is Streamable HTTP (POST, JSON-RPC 2.0). See `app/domain/mcpapp/` for the MCP handler implementation.

## Planner App Context

Personal intelligence layer — conversation-first task/context management, single-user, self-hosted.

**Roadmap:** See `.docs/07-roadmap.md` for current phase and progress.

**Planning docs:** `.docs/01-*.md` through `.docs/12-*.md` cover vision, architecture, data model, ingestion, context engine, infrastructure, roadmap, AI layer, frontend, clarifications, feedback loop, and intent framework.

**Planning skills:** `/plan` (brainstorm), `/plan-feature <name>` (directed planning), `/plan-audit` (drift check), `/plan-status` (overview)
**Session skills:** `/learn` (session review — unfinished work, efficiency, lessons)


## Preferences

- **Parallel haiku agents** — dispatch parallel haiku agents for implementation work. Verify output compiles afterward. Check for pointer/value type mismatches.
- **Always include tests** — tests are part of the deliverable, not an afterthought. Plan and write them alongside features.
- **Auto-classify, manual override** — at capture time, Claude does the classification work (tagging, context assignment). User corrects if needed.

## Token Optimization Rules

- **Parallel agents only for independent tasks** — never parallel-modify the same domain; use `bd dep add` for sequencing
- **Batch code reviews** — review once at feature end, not after every commit
- **Update arch docs once** — regenerate `.docs/arch/` at feature end, not during development
- **No auto-dispatch for coordinated work** — use sequential beads issues instead of `/full-stack` or `/phase`

## Project Knowledge

When brainstorming, designing features, or making architecture decisions, run `bd memories <keyword>` to check for saved project context before proceeding. This contains decisions and rationale from past sessions (e.g., "composable primitives," "no iCal," "life dashboard vision").

<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:ca08a54f -->
## Beads Issue Tracker

Use `bd` for ALL task tracking (not TodoWrite/TaskCreate/markdown). Run `bd prime` for full command reference. Key commands: `bd ready`, `bd show <id>`, `bd update <id> --claim`, `bd close <id>`. Use `bd remember` for persistent knowledge, `/learn` at session end.

Session completion workflow is injected by the session-start hook — see `bd prime` for details. Work is NOT complete until `git push` succeeds.
<!-- END BEADS INTEGRATION -->
