#!/usr/bin/env bash
set -euo pipefail

# Trigger bulk reingest for tasks, notes, and events.
#
# Usage:
#   zarf/deploy/reingest-all.sh                          # all three entity types
#   zarf/deploy/reingest-all.sh task                     # just one
#   zarf/deploy/reingest-all.sh task note                # subset
#   CONTEXT_ID=<uuid> zarf/deploy/reingest-all.sh task   # scoped to a context
#
# Required env:
#   PLANNER_AUTH_API_KEY    API key (export before running)
#
# Optional env:
#   PLANNER_API_URL   default http://127.0.0.1:8080
#                     on the VPS use http://127.0.0.1:8081
#   CONTEXT_ID        UUID to scope the reingest to a single context

API_URL="${PLANNER_API_URL:-http://127.0.0.1:8080}"
KEY="${PLANNER_AUTH_API_KEY:-${PLANNER_API_KEY:-}}"

if [[ -z "$KEY" ]]; then
  echo "error: export PLANNER_AUTH_API_KEY before running" >&2
  exit 1
fi

types=("$@")
if [[ ${#types[@]} -eq 0 ]]; then
  types=(task note event)
fi

for t in "${types[@]}"; do
  case "$t" in
    task|note|event) ;;
    *) echo "error: unknown entity type '$t' (want task|note|event)" >&2; exit 1 ;;
  esac

  body="{\"entityType\":\"$t\""
  if [[ -n "${CONTEXT_ID:-}" ]]; then
    body+=",\"contextId\":\"$CONTEXT_ID\""
  fi
  body+="}"

  echo "-> reingest $t"
  curl -fsS -X POST "$API_URL/api/v1/reingest/bulk" \
    -H "X-API-Key: $KEY" \
    -H "Content-Type: application/json" \
    -d "$body"
  echo
done
