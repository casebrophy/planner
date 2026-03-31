# Planner

Personal intelligence layer — conversation-first task and context management. Self-hosted, single-user.

---

## Developer Setup

### Prerequisites

- [Go 1.26+](https://go.dev/dl/)
- [Docker + Docker Compose](https://docs.docker.com/get-docker/)
- [Node.js](https://nodejs.org/) (for frontend)

### Option A — Local (API + DB in Docker)

1. **Create a `.env` file** at the repo root:

   ```env
   PLANNER_DB_HOST=localhost
   PLANNER_DB_PORT=5433
   PLANNER_DB_USER=planner
   PLANNER_DB_PASSWORD=planner
   PLANNER_DB_NAME=planner
   PLANNER_DB_DISABLE_TLS=true
   PLANNER_AUTH_API_KEY=devkey123
   ```

2. **Start the database:**

   ```bash
   make db-up
   ```

3. **Run migrations and seed data:**

   ```bash
   make migrate
   make seed
   ```

4. **Start the API:**

   ```bash
   make dev
   ```

   API is now at `http://localhost:8080`. Authenticate with `X-API-Key: devkey123`.

5. **Start the frontend** (separate terminal):

   ```bash
   cd web && npm install
   make frontend-dev
   ```

   Frontend is served at `http://localhost:3000`.

---

### Option B — Full Docker Stack

Runs backend, frontend, and database together.

```bash
make up
```

- Backend: `http://localhost:8081`
- Frontend: `http://localhost:3001`
- PostgreSQL: `localhost:5433`

Run migrations and seed inside the container:

```bash
make docker-migrate
make docker-seed
```

Tail logs:

```bash
make logs-all
```

Tear everything down:

```bash
make down
```

---

## Common Commands

| Command | Description |
|---|---|
| `make dev` | Run API locally |
| `make frontend-dev` | Build and serve frontend |
| `make db-up` | Start just the database |
| `make migrate` | Run DB migrations |
| `make seed` | Seed sample data |
| `make up` | Start full Docker stack |
| `make down` | Stop Docker stack |
| `make test` | Run Go tests |
| `make lint` | Run `go vet` |

---

## Architecture

Three-layer Go backend: **app → business → store**

```
app/domain/<name>app/        # HTTP handlers + DTOs
business/domain/<name>bus/   # Domain logic + Storer interface
  stores/<name>db/           # SQL store implementation
foundation/                  # Web framework, logger, sqldb helpers
```

See [CLAUDE.md](CLAUDE.md) for detailed architecture rules and cross-layer impact guidelines.

---

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PLANNER_DB_HOST` | `localhost` | PostgreSQL host |
| `PLANNER_DB_PORT` | `5433` | PostgreSQL port (Docker maps to 5433 locally) |
| `PLANNER_DB_USER` | `planner` | DB user |
| `PLANNER_DB_PASSWORD` | `planner` | DB password |
| `PLANNER_DB_NAME` | `planner` | DB name |
| `PLANNER_DB_DISABLE_TLS` | `true` | Disable TLS for local dev |
| `PLANNER_AUTH_API_KEY` | `devkey123` | API key for `X-API-Key` header |
| `PLANNER_SMTP_ENABLED` | `false` | Enable SMTP ingest listener |
| `PLANNER_SMTP_DOMAIN` | `localhost` | Domain for SMTP server |
| `PLANNER_ANTHROPIC_API_KEY` | — | Anthropic API key (for AI features) |
| `PLANNER_ANTHROPIC_MODEL` | `claude-sonnet-4-20250514` | Claude model to use |
