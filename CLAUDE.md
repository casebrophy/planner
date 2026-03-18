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

`.docs/arch/` contains detailed dependency maps for each domain (task, tag, context, check, mcp). **Read the relevant arch file first** before modifying a domain — each file documents all types, file maps, impact callouts, routes, and cross-domain dependencies.

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
