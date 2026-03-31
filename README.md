# Planner

A personal intelligence layer for conversation-first task and context management. Self-hosted, single-user, and built to capture, organize, and surface what matters through a REST API, a Vue PWA, and an MCP interface for Claude.

---

## What it does

- **Task management** — create, update, filter, and order tasks with status, priority, energy level, and due dates
- **Context engine** — group tasks into contexts with lifecycle states; track context events and inactivity
- **Tags** — apply and query tags across tasks
- **Capture and ingestion** — raw inputs queued for processing; optional SMTP listener ingests emails
- **Clarification queue** — ambiguous items surface for review and resolution
- **Threads and observations** — append notes to tasks; record observations against any subject
- **Transactions** — financial transaction tracking with CSV import
- **MCP interface** — exposes tools (`create_task`, `list_tasks`, `update_task`, `complete_task`, `add_note`, `search_tasks`, and clarification tools) over Streamable HTTP so Claude can act as a task assistant
- **PWA frontend** — responsive Vue shell with mobile tab bar, service worker, and Capacitor support for iOS packaging

---

## Stack

| Layer | Technology |
|---|---|
| Language (backend) | Go 1.26 |
| HTTP framework | Custom (`foundation/web`) |
| Database | PostgreSQL 17 |
| DB driver | `pgx/v5` + `sqlx` |
| Migrations | `ardanlabs/darwin` |
| AI SDK | `anthropics/anthropic-sdk-go` |
| Frontend framework | Vue 3 + Vite |
| State management | Pinia |
| Routing | Vue Router |
| Mobile packaging | Capacitor (iOS) |
| Container runtime | Docker Compose |
| Reverse proxy | nginx (production) |
| TLS | Let's Encrypt / certbot |

---

## Running locally

### Prerequisites

- Go 1.26+
- Docker and Docker Compose v2
- Node.js 20+ (for frontend)

### 1. Start the database

```bash
make db-up
```

This starts only the PostgreSQL container, bound to `localhost:5433`.

### 2. Create a `.env` file at the repo root

```bash
PLANNER_DB_HOST=localhost
PLANNER_DB_PORT=5433
PLANNER_DB_USER=planner
PLANNER_DB_PASSWORD=planner
PLANNER_DB_NAME=planner
PLANNER_DB_DISABLE_TLS=true
PLANNER_AUTH_API_KEY=devkey123
```

Optional variables:

```bash
PLANNER_SMTP_ENABLED=false              # set true to enable email ingestion
PLANNER_SMTP_DOMAIN=localhost
PLANNER_ANTHROPIC_API_KEY=sk-ant-...    # required for AI-powered clarification generation
PLANNER_ANTHROPIC_MODEL=claude-sonnet-4-20250514
```

The Makefile auto-includes `.env` via `-include .env`.

### 3. Run migrations and seed

```bash
make migrate
make seed
```

### 4. Start the API

```bash
make dev
```

The API listens on `http://localhost:8080`.

### 5. Start the frontend (optional)

```bash
make frontend-dev
```

This builds the Vue app and serves it from `http://localhost:3000`.

---

## Authentication

All API routes require an `X-API-Key` header matching `PLANNER_AUTH_API_KEY`.

```bash
curl -H "X-API-Key: devkey123" http://localhost:8080/api/v1/tasks
```

Generate a strong key for production:

```bash
openssl rand -hex 32
```

---

## MCP setup

The planner exposes an MCP server at `/mcp` using Streamable HTTP (POST, JSON-RPC 2.0). This lets Claude act as a conversational interface to your task data.

### Configure Claude to use it

Copy `SKILL.md` into your Claude project or global instructions. Update the server URL and API key:

```
MCP Server URL: http://localhost:8080/mcp   (or your deployed URL)
Auth header:    X-API-Key: <your-api-key>
```

### Available MCP tools

| Tool | What it does |
|---|---|
| `create_task` | Create a task with title, priority, due date, tags |
| `list_tasks` | Query tasks with optional filters |
| `update_task` | Update fields on an existing task |
| `complete_task` | Mark a task done |
| `add_note` | Append a thread entry to a task |
| `search_tasks` | Full-text search across tasks |
| `get_clarification_queue` | List pending clarifications |
| `resolve_clarification` | Resolve a clarification with a chosen action |
| `snooze_clarification` | Defer a clarification |

---

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `PLANNER_DB_HOST` | — | PostgreSQL host |
| `PLANNER_DB_PORT` | `5432` | PostgreSQL port (Docker maps to `5433` locally) |
| `PLANNER_DB_USER` | — | Database user |
| `PLANNER_DB_PASSWORD` | — | Database password |
| `PLANNER_DB_NAME` | — | Database name |
| `PLANNER_DB_DISABLE_TLS` | `false` | Set `true` for local/Docker where TLS is not configured |
| `PLANNER_AUTH_API_KEY` | — | API key for all REST and MCP endpoints |
| `PLANNER_SMTP_ENABLED` | `false` | Enable the SMTP listener for email ingestion |
| `PLANNER_SMTP_ADDR` | `:2525` | Address the SMTP listener binds to inside the container |
| `PLANNER_SMTP_DOMAIN` | — | Domain accepted by the SMTP listener |
| `PLANNER_ANTHROPIC_API_KEY` | — | Anthropic API key for AI-powered features |
| `PLANNER_ANTHROPIC_MODEL` | `claude-sonnet-4-20250514` | Model used for clarification generation |

---

## Production deployment

The repo includes a full VPS deployment setup in `zarf/deploy/`. The full guide is at [`zarf/deploy/VPS-SETUP.md`](zarf/deploy/VPS-SETUP.md). Summary:

### First-time setup

1. Provision an Ubuntu 22.04+ VPS. Point your domain's A records at it.
2. Install Docker, nginx, and certbot on the VPS.
3. Clone the repo to `/opt/planner`.
4. Create `/opt/planner/.env` with production values (see the env table above).
5. Run the initial deploy:

```bash
./zarf/deploy/deploy.sh
```

This builds Docker images, runs migrations, and starts all services.

### Nginx configuration

The deploy guide includes nginx configs that proxy:

- `api.yourdomain.com` → `127.0.0.1:8080` (backend API + MCP)
- `app.yourdomain.com` → `127.0.0.1:3000` (Vue frontend)

TLS is handled by certbot:

```bash
sudo certbot --nginx -d api.yourdomain.com -d app.yourdomain.com
```

### Auto-deploy (CD)

A systemd timer (`planner-deploy.timer`) runs `autopull.sh` every 60 seconds. The script checks for new commits on `main`, runs `deploy.sh` if there are changes, and logs the result to journald.

```bash
sudo cp zarf/deploy/planner-deploy.service /etc/systemd/system/
sudo cp zarf/deploy/planner-deploy.timer   /etc/systemd/system/
sudo systemctl enable --now planner-deploy.timer
```

### Docker services

| Service | Internal port | External binding |
|---|---|---|
| `db` | 5432 | `127.0.0.1:5433` |
| `backend` | 8080 | `127.0.0.1:8080` |
| `backend` (SMTP) | 2525 | `0.0.0.0:25` |
| `frontend` | 3000 | `127.0.0.1:3000` |

---

## Make targets

| Target | Description |
|---|---|
| `make dev` | Run the API locally (requires `.env` and DB running) |
| `make migrate` | Run database migrations against the local DB |
| `make seed` | Seed the database with sample data |
| `make db-up` | Start the PostgreSQL container only |
| `make db-down` | Stop the PostgreSQL container |
| `make frontend-dev` | Build the Vue app and serve it locally |
| `make build` | Build Docker images |
| `make up` | Start all Docker containers (DB + backend + frontend) |
| `make down` | Stop all Docker containers |
| `make restart` | Restart the backend container |
| `make logs` | Tail backend container logs |
| `make logs-all` | Tail backend and frontend container logs |
| `make docker-migrate` | Run migrations inside the running backend container |
| `make docker-seed` | Seed inside the running backend container |
| `make test` | Run all Go tests (`go test ./... -count=1`) |
| `make lint` | Run `go vet ./...` |
| `make tidy` | Run `go mod tidy` |
| `make admin ARGS=<cmd>` | Run the admin CLI with arbitrary arguments |
| `make deploy` | Run the deploy script (`zarf/deploy/deploy.sh`) |
| `make cap-build` | Build the Vue app and sync to the Capacitor iOS project |
| `make cap-open` | Open the iOS project in Xcode |

---

## Project structure

```
api/services/planner/    # main.go — wires all domains together
api/services/frontend/   # Go SPA server (serves the built Vue app)
api/tooling/admin/       # Migration and seed CLI

app/domain/<name>app/    # HTTP handlers and request/response DTOs
business/domain/<name>bus/  # Business logic, domain types, Storer interfaces
  stores/<name>db/       # SQL store implementations
business/types/          # Enum types (taskstatus, taskpriority, etc.)
business/sdk/            # Shared SDK: ordering, pagination, migrations
foundation/web/          # HTTP framework
foundation/logger/       # Structured logger
foundation/sqldb/        # sqlx helpers and ErrDBNotFound

web/                     # Vue 3 frontend (Vite, Pinia, Vue Router, Capacitor)
zarf/compose/            # Docker Compose configuration
zarf/docker/             # Dockerfiles
zarf/deploy/             # VPS deploy scripts and systemd units
```
