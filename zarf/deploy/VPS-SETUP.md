# VPS Setup Guide

One-time setup for deploying the planner app to a fresh VPS.

**Prerequisites:** Ubuntu 22.04+, root/sudo access, a domain pointing to the VPS IP.

---

## 1. Install Docker

```bash
# Install Docker and Compose v2
sudo apt update
sudo apt install -y docker.io docker-compose-v2 curl git nginx certbot python3-certbot-nginx

# Start and enable Docker
sudo systemctl enable --now docker
```

## 2. Create deploy user

```bash
# Create user (no password login — SSH key only)
sudo adduser --disabled-password --gecos "" deployer

# Add to docker group so it can run docker compose
sudo usermod -aG docker deployer
```

## 3. Create GitHub Personal Access Token

1. Go to https://github.com/settings/personal-access-tokens/new
2. Name: `planner-vps-deploy`
3. Expiration: pick something reasonable (90 days, or no expiration for personal use)
4. Repository access: **Only select repositories** → `casebrophy/planner`
5. Permissions: **Contents** → Read-only (this is all that's needed for git fetch + CI status)
6. Generate and copy the token — you'll need it in steps 4 and 5

## 4. Clone the repo

```bash
# Switch to deploy user
sudo su - deployer

# Clone using the token (replace <TOKEN> with your actual token)
git clone https://<TOKEN>@github.com/casebrophy/planner.git /opt/planner

# Store credentials so git fetch works without prompting
cd /opt/planner
git config credential.helper store
# The token is already stored from the clone URL
```

## 5. Create production .env

```bash
cat > /opt/planner/.env << 'ENVEOF'
# Database (must match docker-compose db service)
PLANNER_DB_HOST=db
PLANNER_DB_PORT=5432
PLANNER_DB_USER=planner
PLANNER_DB_PASSWORD=CHANGE_ME_STRONG_PASSWORD
PLANNER_DB_NAME=planner
PLANNER_DB_DISABLE_TLS=true

# Auth — generate with: openssl rand -hex 32
PLANNER_API_KEY=CHANGE_ME_GENERATE_THIS

# SMTP (disable until DNS/MX is set up)
PLANNER_SMTP_ENABLED=false
PLANNER_SMTP_DOMAIN=yourdomain.com

# Anthropic (for email extraction — optional until email ingestion is enabled)
PLANNER_ANTHROPIC_API_KEY=sk-ant-CHANGE_ME
PLANNER_ANTHROPIC_MODEL=claude-sonnet-4-20250514

# GitHub token for CI status checks (the same PAT from step 3)
PLANNER_GITHUB_TOKEN=github_pat_CHANGE_ME
ENVEOF
```

Now edit the file and replace all `CHANGE_ME` values:

```bash
nano /opt/planner/.env
```

Generate a strong DB password and API key:
```bash
openssl rand -hex 32  # use for PLANNER_DB_PASSWORD
openssl rand -hex 32  # use for PLANNER_API_KEY
```

Lock down permissions:
```bash
chmod 600 /opt/planner/.env
```

## 6. Make scripts executable

```bash
chmod +x /opt/planner/zarf/deploy/deploy.sh
chmod +x /opt/planner/zarf/deploy/autopull.sh
```

## 7. Run initial deploy

```bash
cd /opt/planner
./zarf/deploy/deploy.sh
```

This will:
- Build Docker images (takes a few minutes on first run)
- Start PostgreSQL and wait for it to be healthy
- Run database migrations
- Start the backend and frontend
- Health check both services

Verify it worked:
```bash
curl http://127.0.0.1:8080/api/v1/readiness
# Should return: {"status":"ok"} or similar

curl -s http://127.0.0.1:3000/ | head -5
# Should return HTML of the Vue app
```

## 8. Install systemd timer (auto-deploy)

```bash
# Exit back to root/sudo user
exit

# Copy systemd units
sudo cp /opt/planner/zarf/deploy/planner-deploy.service /etc/systemd/system/
sudo cp /opt/planner/zarf/deploy/planner-deploy.timer /etc/systemd/system/

# Enable and start the timer
sudo systemctl daemon-reload
sudo systemctl enable --now planner-deploy.timer

# Verify timer is active
systemctl list-timers | grep planner
```

Test it works:
```bash
# Manually trigger the autopull
sudo systemctl start planner-deploy.service

# Check logs
journalctl -u planner-deploy --no-pager -n 20
```

## 9. Set up nginx reverse proxy

Create the API config:

```bash
sudo tee /etc/nginx/sites-available/planner-api << 'NGINX'
server {
    listen 80;
    server_name api.yourdomain.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # MCP endpoint rate limiting
    location /mcp {
        limit_req zone=mcp burst=10 nodelay;
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
NGINX
```

Create the frontend config:

```bash
sudo tee /etc/nginx/sites-available/planner-app << 'NGINX'
server {
    listen 80;
    server_name app.yourdomain.com;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
NGINX
```

Add the MCP rate limit zone (add to the `http` block in `/etc/nginx/nginx.conf`):

```bash
# Add this line inside the http { } block, before any server blocks
# sudo nano /etc/nginx/nginx.conf
limit_req_zone $binary_remote_addr zone=mcp:10m rate=30r/m;
```

Enable the sites:

```bash
sudo ln -s /etc/nginx/sites-available/planner-api /etc/nginx/sites-enabled/
sudo ln -s /etc/nginx/sites-available/planner-app /etc/nginx/sites-enabled/

# Test config
sudo nginx -t

# Reload
sudo systemctl reload nginx
```

## 10. Set up TLS with certbot

```bash
# Get certificates (certbot will modify the nginx configs automatically)
sudo certbot --nginx -d api.yourdomain.com -d app.yourdomain.com

# Certbot installs a systemd timer for auto-renewal
# Verify:
systemctl list-timers | grep certbot
```

## 11. DNS records

Set these at your domain registrar:

| Record | Type | Value |
|--------|------|-------|
| `api.yourdomain.com` | A | `<VPS IP>` |
| `app.yourdomain.com` | A | `<VPS IP>` |

If you plan to enable SMTP later, also add:

| Record | Type | Value |
|--------|------|-------|
| `mail.yourdomain.com` | A | `<VPS IP>` |
| `yourdomain.com` | MX 10 | `mail.yourdomain.com` |
| `yourdomain.com` | TXT | `v=spf1 a mx ~all` |

---

## Verification checklist

After completing all steps:

- [ ] `curl https://api.yourdomain.com/api/v1/readiness` returns 200
- [ ] `https://app.yourdomain.com` loads the Vue frontend
- [ ] `journalctl -u planner-deploy -f` shows the timer running every 60s
- [ ] Push a commit to main → wait 60s → VPS deploys automatically
- [ ] `systemctl list-timers | grep planner` shows the timer active

## Ongoing maintenance

**View deploy logs:**
```bash
journalctl -u planner-deploy --no-pager -n 50
```

**View app logs:**
```bash
cd /opt/planner && docker compose -f zarf/compose/docker-compose.yml logs -f backend
```

**Manual deploy (skip waiting for timer):**
```bash
sudo systemctl start planner-deploy.service
```

**Force redeploy (bypass CI check):**
```bash
sudo su - deployer
cd /opt/planner && ./zarf/deploy/deploy.sh
```

**Database backup:**
```bash
cd /opt/planner
docker compose -f zarf/compose/docker-compose.yml exec db pg_dump -U planner planner > backup_$(date +%Y%m%d).sql
```
