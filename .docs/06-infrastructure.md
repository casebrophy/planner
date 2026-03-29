# Infrastructure & deployment

Single VPS running Docker Compose services, fronted by host-level nginx for TLS termination. All secrets in `.env`; never committed.

---

## Server requirements

Ubuntu 22.04+, Docker + Compose v2, nginx (host), certbot, 1GB RAM min, 20GB disk.

---

## DNS configuration

| Record | Type | Value |
|--------|------|-------|
| `yourdomain.com` | A | `<server-ip>` |
| `app.yourdomain.com` | A | `<server-ip>` |
| `api.yourdomain.com` | A | `<server-ip>` |
| `mail.yourdomain.com` | A | `<server-ip>` |
| `yourdomain.com` | MX 10 | `mail.yourdomain.com` |
| `yourdomain.com` | TXT | `v=spf1 a mx ~all` |

---

## TLS

Wildcard cert via certbot (`certonly --dns-<registrar> -d yourdomain.com -d '*.yourdomain.com'`). Certbot installs a systemd timer for auto-renewal.

---

## nginx routing

One config file per subdomain in `/etc/nginx/sites-available/`. All HTTP redirects to HTTPS.

| Subdomain | Upstream | Notes |
|-----------|----------|-------|
| `api.yourdomain.com` | `127.0.0.1:8080` | `/mcp` rate-limited: 30r/m, burst 10 |
| `app.yourdomain.com` | `127.0.0.1:3000` | Vue frontend static files |

---

## Docker services

| Service | Internal port | External bind | Volume | Notes |
|---------|--------------|---------------|--------|-------|
| `db` | 5432 | `127.0.0.1:5433` | `pgdata` | PostgreSQL 17 |
| `backend` | 8080, 2525 | `127.0.0.1:8080`, `0.0.0.0:25:2525` | — | REST API + embedded SMTP (when enabled) |
| `frontend` | 3000 | `127.0.0.1:3000` | — | Go static file server serving pre-built Vue app |
| `ml` | 8090 | `127.0.0.1:8090` | — | Future — Phase 8 |

SMTP is embedded in the backend binary via `smtpbus`, not a separate container. It listens on `:2525` internally and is mapped to host port 25 for external MTA delivery. Enabled via `PLANNER_SMTP_ENABLED=true`.

---

## Environment variables

Stored in `.env` at project root.

| Variable | Required | Default | Notes |
|----------|----------|---------|-------|
| `PLANNER_AUTH_API_KEY` | yes | — | `openssl rand -hex 32` |
| `PLANNER_DB_HOST` | no | `localhost` | `db` in Docker |
| `PLANNER_DB_PORT` | no | `5432` | — |
| `PLANNER_DB_USER` | no | `planner` | — |
| `PLANNER_DB_PASSWORD` | yes | — | — |
| `PLANNER_DB_NAME` | no | `planner` | — |
| `PLANNER_DB_DISABLE_TLS` | no | `true` | — |
| `PLANNER_SMTP_ENABLED` | no | `false` | Set `true` to start SMTP listener |
| `PLANNER_SMTP_ADDR` | no | `:2525` | Internal listen address |
| `PLANNER_SMTP_DOMAIN` | no | `localhost` | Domain for RCPT TO validation |
| `PLANNER_ANTHROPIC_API_KEY` | no | — | Required when SMTP is enabled |
| `PLANNER_ANTHROPIC_MODEL` | no | `claude-sonnet-4-20250514` | — |
| `PLANNER_FRONTEND_DIR` | no | `/service/web` | Path to pre-built frontend assets |
| `PLANNER_WEB_CORS_ORIGINS` | no | `*` | CORS allowed origins |

---

## Deployment workflow

**First deploy:**
1. `git clone <repo> /opt/taskapp && cd /opt/taskapp`
2. `cp .env.example .env` — set `API_KEY`
3. `make init` — verify .env, create data directories
4. `make build` — docker compose build
5. `make up` — docker compose up -d
6. `make smoke` — verify endpoints

**Subsequent deploys:** `git pull && make build && make up`

---

## Backup

Daily cron at 3am: `pg_dump` the planner database, compress and copy to `/opt/taskapp/backups/planner-YYYYMMDD.sql.gz`. Purge files older than 30 days. Off-site: rsync backups to another machine or Backblaze B2.

---

## MCP connector registration

1. Claude.ai → Settings → Connectors → Add custom connector
2. **Name**: Task App | **URL**: `https://api.yourdomain.com/mcp` | **Auth**: Custom header `X-API-Key: <your-api-key>`
3. Test connection — Claude calls `tools/list` to confirm available tools
4. Upload `skill/SKILL.md` to Claude skills directory

---

## Monitoring

- **Uptime**: UptimeRobot pings `/health` every 5 min, alerts on down
- **Disk**: cron alert if data volume >80%
- **Logs**: `docker compose logs -f`; enable Docker log rotation (`max-size=10m`, `max-file=3`)
- **nginx**: grep `/var/log/nginx/error.log` for 5xx periodically
