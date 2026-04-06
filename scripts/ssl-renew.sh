#!/usr/bin/env bash
# =============================================================================
# ssl-renew.sh — Reload nginx after Certbot auto-renewal
# =============================================================================
# The certbot container automatically attempts renewal every 12 hours.
# However, nginx needs to be reloaded to pick up renewed certificates.
#
# Add this script to the host's crontab to reload nginx after renewal:
#
#   # Reload nginx to pick up renewed TLS certificates (twice daily)
#   0 0,12 * * * /path/to/chesed/scripts/ssl-renew.sh >> /var/log/chesed-ssl-renew.log 2>&1
#
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

ENV_FILE="${ENV_FILE:-$PROJECT_DIR/.env.prod}"
COMPOSE_FILE="$PROJECT_DIR/docker-compose.prod.yml"

echo "[$(date -u '+%Y-%m-%dT%H:%M:%SZ')] Starting TLS renewal check..."

docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" exec -T certbot certbot renew --quiet
docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" exec -T frontend nginx -s reload

echo "[$(date -u '+%Y-%m-%dT%H:%M:%SZ')] TLS renewal check complete."
