#!/usr/bin/env bash
# =============================================================================
# init-letsencrypt.sh — Bootstrap Let's Encrypt certificates for Chesed
# =============================================================================
# This script solves the chicken-and-egg problem: nginx needs TLS certificates
# to start on port 443, but Certbot needs nginx running on port 80 to complete
# the ACME HTTP-01 challenge.
#
# Usage:
#   1. Copy .env.prod.example to .env.prod and fill in DOMAIN_NAME + CERTBOT_EMAIL
#   2. Run: sudo bash scripts/init-letsencrypt.sh
#   3. Then start the full stack: docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
#
# What it does:
#   1. Creates a dummy self-signed certificate so nginx can start
#   2. Starts nginx (frontend) container
#   3. Requests a real certificate from Let's Encrypt via Certbot
#   4. Replaces the dummy certificate with the real one
#   5. Reloads nginx to pick up the real certificate
# =============================================================================

set -euo pipefail

# Load environment variables
ENV_FILE="${ENV_FILE:-.env.prod}"
if [ ! -f "$ENV_FILE" ]; then
    echo "Error: $ENV_FILE not found. Copy .env.prod.example to .env.prod and fill in values."
    exit 1
fi

# shellcheck source=/dev/null
source "$ENV_FILE"

# Validate required variables
if [ -z "${DOMAIN_NAME:-}" ]; then
    echo "Error: DOMAIN_NAME is not set in $ENV_FILE"
    exit 1
fi
if [ -z "${CERTBOT_EMAIL:-}" ]; then
    echo "Error: CERTBOT_EMAIL is not set in $ENV_FILE"
    exit 1
fi

COMPOSE_FILE="docker-compose.prod.yml"
COMPOSE_CMD="docker compose -f $COMPOSE_FILE --env-file $ENV_FILE"

# Certificate path inside the certbot-conf volume
CERT_DIR="/etc/letsencrypt/live/chesed"

# Use staging for testing (set LETSENCRYPT_STAGING=1 to use staging server)
STAGING_ARG=""
if [ "${LETSENCRYPT_STAGING:-0}" = "1" ]; then
    STAGING_ARG="--staging"
    echo "Using Let's Encrypt STAGING server (certificates will NOT be trusted)"
fi

echo "=== Chesed TLS Bootstrap ==="
echo "Domain:  $DOMAIN_NAME"
echo "Email:   $CERTBOT_EMAIL"
echo "Staging: ${LETSENCRYPT_STAGING:-0}"
echo ""

# Step 1: Create dummy certificate so nginx can start
echo "Step 1/5: Creating dummy certificate for nginx startup..."
$COMPOSE_CMD run --rm --entrypoint "" certbot sh -c "
    mkdir -p $CERT_DIR &&
    openssl req -x509 -nodes -newkey rsa:2048 -days 1 \
        -keyout $CERT_DIR/privkey.pem \
        -out $CERT_DIR/fullchain.pem \
        -subj '/CN=localhost'
"
echo "  Dummy certificate created."

# Step 2: Start nginx
echo "Step 2/5: Starting nginx..."
$COMPOSE_CMD up -d frontend
echo "  Waiting for nginx to become healthy..."
sleep 5

# Step 3: Delete dummy certificate
echo "Step 3/5: Removing dummy certificate..."
$COMPOSE_CMD run --rm --entrypoint "" certbot sh -c "
    rm -rf /etc/letsencrypt/live/chesed &&
    rm -rf /etc/letsencrypt/archive/chesed &&
    rm -rf /etc/letsencrypt/renewal/chesed.conf
"
echo "  Dummy certificate removed."

# Step 4: Request real certificate from Let's Encrypt
echo "Step 4/5: Requesting certificate from Let's Encrypt..."
$COMPOSE_CMD run --rm certbot certonly \
    --webroot \
    --webroot-path=/var/www/certbot \
    --email "$CERTBOT_EMAIL" \
    --agree-tos \
    --no-eff-email \
    --cert-name chesed \
    -d "$DOMAIN_NAME" \
    $STAGING_ARG
echo "  Certificate obtained successfully."

# Step 5: Reload nginx with real certificate
echo "Step 5/5: Reloading nginx with real certificate..."
$COMPOSE_CMD exec frontend nginx -s reload
echo "  Nginx reloaded."

echo ""
echo "=== TLS Bootstrap Complete ==="
echo "Your site is now available at https://$DOMAIN_NAME"
echo ""
echo "Next steps:"
echo "  1. Start the full stack: $COMPOSE_CMD up -d"
echo "  2. The certbot container will auto-renew certificates every 12 hours"
echo "  3. Add a cron job for nginx reload (see scripts/ssl-renew.sh)"
