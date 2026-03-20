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

| Service | Internal port | External bind | Volume |
|---------|--------------|---------------|--------|
| `backend` | 8080 | `127.0.0.1:8080` | `taskdata:/data`, `./data/imports:/data/imports` |
| `smtp` | 25, 587 | `0.0.0.0:25`, `0.0.0.0:587` | — |
| `frontend` | 80 | `127.0.0.1:3000` | — |
| `ml` | 8090 | `127.0.0.1:8090` | — |

SMTP port 25 must be externally reachable for email receipt. All others are localhost-only behind nginx. iOS app (Capacitor) builds separately and talks directly to `api.yourdomain.com`.

---

## Environment variables

Stored in `.env` at project root.

| Variable | Required | Default | Notes |
|----------|----------|---------|-------|
| `API_KEY` | yes | — | `openssl rand -hex 32` |
| `PORT` | no | `8080` | — |
| `DB_PATH` | no | `/data/tasks.db` | — |
| `SMTP_DOMAIN` | no | — | `mail.yourdomain.com` |
| `LOG_LEVEL` | no | `info` | — |

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
