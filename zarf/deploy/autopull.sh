#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="/opt/planner"
GITHUB_REPO="casebrophy/planner"

cd "$REPO_DIR"

# Load .env for GITHUB_TOKEN if available
if [ -f .env ]; then
    set -a
    # shellcheck disable=SC1091
    source .env
    set +a
fi

# Fetch latest from origin
git fetch origin main --quiet

LOCAL=$(git rev-parse HEAD)
REMOTE=$(git rev-parse origin/main)

if [ "$LOCAL" = "$REMOTE" ]; then
    exit 0
fi

echo "New commits detected: $LOCAL -> $REMOTE"

# Gate on CI status (skip deploy if CI hasn't passed)
if [ -n "${PLANNER_GITHUB_TOKEN:-}" ]; then
    STATUS=$(curl -sf \
        -H "Authorization: token $PLANNER_GITHUB_TOKEN" \
        -H "Accept: application/vnd.github+json" \
        "https://api.github.com/repos/$GITHUB_REPO/commits/$REMOTE/status" \
        | grep -o '"state":"[^"]*"' | head -1 | cut -d'"' -f4)

    if [ "$STATUS" != "success" ]; then
        echo "CI status is '$STATUS' (not 'success'). Skipping deploy."
        exit 0
    fi
    echo "CI passed. Deploying."
else
    echo "No PLANNER_GITHUB_TOKEN set. Skipping CI check."
fi

# Deploy
exec "$REPO_DIR/zarf/deploy/deploy.sh"
