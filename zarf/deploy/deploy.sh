#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="/opt/planner"

cd "$REPO_DIR"

echo "=== Pulling latest code ==="
git fetch origin main
git reset --hard origin/main

echo "=== Decrypting secrets ==="
SOPS_AGE_KEY_FILE="$REPO_DIR/zarf/keys/age.key" \
  sops --decrypt --input-type dotenv --output-type dotenv .secrets.env > .env.decrypted
chmod 600 .env.decrypted
cat .env .env.decrypted > .env.combined
chmod 600 .env.combined

COMPOSE="docker compose --env-file .env.combined -f zarf/compose/docker-compose.yml"

echo "=== Building Docker images ==="
$COMPOSE build

echo "=== Ensuring database is running ==="
$COMPOSE up -d db
for i in $(seq 1 15); do
    if $COMPOSE exec db pg_isready -U planner > /dev/null 2>&1; then
        break
    fi
    if [ "$i" -eq 15 ]; then
        echo "ERROR: Database not ready after 15 attempts."
        exit 1
    fi
    sleep 2
done

echo "=== Running migrations ==="
$COMPOSE run --rm backend /service/admin migrate

echo "=== Restarting services ==="
$COMPOSE up -d

echo "=== Health check: backend ==="
for i in $(seq 1 30); do
    if curl -sf http://127.0.0.1:8081/api/v1/readiness > /dev/null 2>&1; then
        echo "Backend is healthy."
        break
    fi
    if [ "$i" -eq 30 ]; then
        echo "ERROR: Backend health check failed after 60s."
        $COMPOSE logs --tail=50 backend
        exit 1
    fi
    sleep 2
done

echo "=== Health check: frontend ==="
for i in $(seq 1 30); do
    if curl -so /dev/null -w '' http://127.0.0.1:3001/ 2>/dev/null; then
        echo "Frontend is healthy."
        break
    fi
    if [ "$i" -eq 30 ]; then
        echo "WARNING: Frontend health check failed after 60s."
        $COMPOSE logs --tail=50 frontend
        exit 1
    fi
    sleep 2
done

echo "=== Rebuilding sidecar ==="
cd "$REPO_DIR/zarf/sidecar"
GOOS=linux GOARCH=amd64 go build -o sidecar .
cd "$REPO_DIR"

echo "=== Restarting sidecar ==="
sudo systemctl restart planner-sidecar
if systemctl is-active --quiet planner-sidecar; then
    echo "Sidecar is healthy."
else
    echo "WARNING: Sidecar failed to start."
    journalctl -u planner-sidecar --no-pager -n 20
fi

rm -f .env.decrypted .env.combined

echo "=== Deploy complete ==="
