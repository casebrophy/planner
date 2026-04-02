# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Local dev environment — create a .env file at repo root:
# PLANNER_DB_HOST=localhost
# PLANNER_DB_PORT=5433        # Docker maps Postgres to 5433 locally (not 5432)
# PLANNER_DB_USER=planner
# PLANNER_DB_PASSWORD=planner
# PLANNER_DB_NAME=planner
# PLANNER_DB_DISABLE_TLS=true
# PLANNER_AUTH_API_KEY=devkey123
# Makefile auto-includes .env via -include .env

# Run the API locally (requires DB running)
make dev

# Database setup (local)
make db-up        # Start just the PostgreSQL container
make migrate      # Run migrations
make seed         # Seed with sample data

# Docker (full stack)
make up           # Start all containers
make down         # Stop all containers
make logs         # Tail backend logs

# Testing and linting
make test         # go test ./... -count=1
make lint         # go vet ./...

# Run a single test (when test files are added)
go test ./business/domain/taskbus/... -run TestFuncName -count=1

# Admin tooling
make admin ARGS=migrate
make admin ARGS=seed

# Frontend (local dev — build + serve)
make frontend-dev

# Tail all service logs
make logs-all
```

## Architecture

Three-layer architecture: **app → business → store**. Each layer owns its own types; explicit conversion functions translate between layers.

```
api/services/planner/    # main.go — wire everything together
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

business/types/          # Enum types (taskstatus, taskpriority, taskenergy, contextstatus)
business/sdk/            # Shared SDK: order, page, migrate
foundation/web/          # HTTP framework: App, Handle(), HandlerFunc, Respond()
foundation/logger/       # Structured logger
foundation/sqldb/        # sqlx helpers: NamedExecContext, NamedQuerySlice, NamedQueryStruct
app/sdk/errs/            # Error codes (InvalidArgument, NotFound, Internal, etc.) → HTTP status
app/sdk/mid/             # Middleware: auth (API key), logging, panics, errors
```

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

## MCP / Skill Integration

This repo also serves as a personal task manager via MCP. `SKILL.md` defines a Claude skill that calls the running API at `http://localhost:8080/mcp`. The MCP transport is Streamable HTTP (POST, JSON-RPC 2.0). See `app/domain/mcpapp/` for the MCP handler implementation.

## Planner App Context

Personal intelligence layer — conversation-first task/context management, single-user, self-hosted.

**Roadmap:** See `.docs/07-roadmap.md` for current phase and progress.

**Planning docs:** `.docs/01-*.md` through `.docs/12-*.md` cover vision, architecture, data model, ingestion, context engine, infrastructure, roadmap, AI layer, frontend, clarifications, feedback loop, and intent framework.

**Planning skills:** `/plan` (brainstorm), `/plan-feature <name>` (directed planning), `/plan-audit` (drift check), `/plan-status` (overview)


## Preferences

- **Parallel haiku agents** — dispatch parallel haiku agents for implementation work. Verify output compiles afterward. Check for pointer/value type mismatches.
- **Always include tests** — tests are part of the deliverable, not an afterthought. Plan and write them alongside features.
- **Auto-classify, manual override** — at capture time, Claude does the classification work (tagging, context assignment). User corrects if needed.

## Project Knowledge

When brainstorming, designing features, or making architecture decisions, run `bd memories <keyword>` to check for saved project context before proceeding. This contains decisions and rationale from past sessions (e.g., "composable primitives," "no iCal," "life dashboard vision").

<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:ca08a54f -->
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

## Session Completion

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd dolt push
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
<!-- END BEADS INTEGRATION -->
