#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="/opt/planner"
BACKUP_DIR="/opt/planner/backups"

cd "$REPO_DIR"
mkdir -p "$BACKUP_DIR"

echo "=== Decrypting secrets ==="
SOPS_AGE_KEY_FILE="$REPO_DIR/zarf/keys/age.key" \
  sops --decrypt --input-type dotenv --output-type dotenv .secrets.env > .env.decrypted
chmod 600 .env.decrypted
cat .env .env.decrypted > .env.combined
chmod 600 .env.combined

COMPOSE="docker compose --env-file .env.combined -f zarf/compose/docker-compose.yml"
BACKUP_FILE="$BACKUP_DIR/planner_$(date +%Y%m%d_%H%M%S).sql.gz"

echo "=== Dumping database ==="
$COMPOSE exec -T db pg_dump -U planner planner | gzip > "$BACKUP_FILE"

if [ ! -s "$BACKUP_FILE" ]; then
    echo "ERROR: Backup file is empty"
    rm -f "$BACKUP_FILE" .env.decrypted .env.combined
    exit 1
fi

echo "=== Purging backups older than 7 days ==="
find "$BACKUP_DIR" -name "planner_*.sql.gz" -mtime +7 -delete

rm -f .env.decrypted .env.combined

echo "=== Backup complete: $BACKUP_FILE ($(du -h "$BACKUP_FILE" | cut -f1)) ==="
