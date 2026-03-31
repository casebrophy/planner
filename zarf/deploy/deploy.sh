#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="/opt/planner"
COMPOSE="docker compose --env-file .env -f zarf/compose/docker-compose.yml"

cd "$REPO_DIR"

echo "=== Pulling latest code ==="
git fetch origin main
git reset --hard origin/main

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
for i in $(seq 1 15); do
    if curl -sf http://127.0.0.1:3001/ > /dev/null 2>&1; then
        echo "Frontend is healthy."
        break
    fi
    if [ "$i" -eq 15 ]; then
        echo "WARNING: Frontend health check failed after 30s."
        $COMPOSE logs --tail=50 frontend
        exit 1

    fi
    sleep 2
done

echo "=== Deploy complete ==="
