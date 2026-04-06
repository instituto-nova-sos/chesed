#!/usr/bin/env bash
# =============================================================================
# deploy.sh — Production deployment script for Chesed
# =============================================================================
# Runs on the production server (called via SSH from GitHub Actions CD workflow).
# Handles both first deploys (TLS bootstrap) and subsequent updates.
#
# Usage:
#   bash scripts/deploy.sh <commit-sha>
#   bash scripts/deploy.sh              # defaults to latest origin/main
#
# Prerequisites:
#   - .env.prod configured on the server
#   - Docker and Docker Compose installed
#   - Git repository cloned to the working directory
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
ENV_FILE="$PROJECT_DIR/.env.prod"
COMPOSE_FILE="$PROJECT_DIR/docker-compose.prod.yml"
COMPOSE_CMD="docker compose -f $COMPOSE_FILE --env-file $ENV_FILE"
TARGET_SHA="${1:-}"

cd "$PROJECT_DIR"

log() {
    echo "[$(date -u '+%Y-%m-%dT%H:%M:%SZ')] $*"
}

# =============================================================================
# Pre-flight checks
# =============================================================================

log "==> Starting Chesed deployment"

if [ ! -f "$ENV_FILE" ]; then
    log "FATAL: $ENV_FILE not found. Copy .env.prod.example to .env.prod and fill in values."
    exit 1
fi

# shellcheck source=/dev/null
source "$ENV_FILE"

if [ -z "${DOMAIN_NAME:-}" ]; then
    log "FATAL: DOMAIN_NAME is not set in $ENV_FILE"
    exit 1
fi

# =============================================================================
# Pull latest code
# =============================================================================

log "==> Fetching latest code from origin..."
git fetch origin main

if [ -n "$TARGET_SHA" ]; then
    log "==> Checking out commit: $TARGET_SHA"
    git checkout "$TARGET_SHA"
else
    log "==> Checking out latest origin/main"
    git checkout origin/main
fi

log "==> Current commit: $(git rev-parse HEAD)"

# =============================================================================
# First deploy: TLS certificate bootstrap
# =============================================================================

log "==> Checking for existing TLS certificates..."
CERT_EXISTS=$($COMPOSE_CMD run --rm --entrypoint "" certbot \
    sh -c 'test -f /etc/letsencrypt/live/chesed/fullchain.pem && echo "yes" || echo "no"' 2>/dev/null || echo "no")

if [ "$CERT_EXISTS" = "no" ]; then
    log "==> First deploy detected: no TLS certificates found"
    log "==> Running init-letsencrypt.sh to bootstrap Let's Encrypt..."
    bash "$PROJECT_DIR/scripts/init-letsencrypt.sh"
    log "==> TLS bootstrap complete"
else
    log "==> TLS certificates found, skipping bootstrap"
fi

# =============================================================================
# Build and deploy
# =============================================================================

log "==> Building Docker images..."
$COMPOSE_CMD build

log "==> Starting services..."
$COMPOSE_CMD up -d

# =============================================================================
# Health checks
# =============================================================================

log "==> Waiting for services to become healthy..."

# Check nginx (port 80, no TLS needed)
MAX_RETRIES=20
RETRY_INTERVAL=10

for i in $(seq 1 "$MAX_RETRIES"); do
    if curl -sf http://localhost/nginx-health > /dev/null 2>&1; then
        log "==> Frontend (nginx) is healthy"
        break
    fi
    if [ "$i" -eq "$MAX_RETRIES" ]; then
        log "FATAL: Frontend health check failed after $((MAX_RETRIES * RETRY_INTERVAL))s"
        log "==> Recent frontend logs:"
        $COMPOSE_CMD logs --tail=50 frontend
        exit 1
    fi
    log "    Attempt $i/$MAX_RETRIES — retrying in ${RETRY_INTERVAL}s..."
    sleep "$RETRY_INTERVAL"
done

# Check API through nginx (HTTPS, self-signed cert OK for localhost)
for i in $(seq 1 "$MAX_RETRIES"); do
    if curl -sfk https://localhost/api/v1/health > /dev/null 2>&1; then
        log "==> API is healthy"
        break
    fi
    if [ "$i" -eq "$MAX_RETRIES" ]; then
        log "FATAL: API health check failed after $((MAX_RETRIES * RETRY_INTERVAL))s"
        log "==> Recent API logs:"
        $COMPOSE_CMD logs --tail=50 api
        log "==> Recent migration logs:"
        $COMPOSE_CMD logs --tail=30 migrate
        exit 1
    fi
    log "    Attempt $i/$MAX_RETRIES — retrying in ${RETRY_INTERVAL}s..."
    sleep "$RETRY_INTERVAL"
done

# =============================================================================
# Cleanup
# =============================================================================

log "==> Pruning unused Docker images..."
docker image prune -f

log "==> Deployment complete"
log "    Commit: $(git rev-parse HEAD)"
log "    Domain: https://$DOMAIN_NAME"
