#!/usr/bin/env bash
# =============================================================================
# restore.sh — Restore a Chesed PostgreSQL dump into a target database
# =============================================================================
# DESTRUCTIVE. This overwrites the schema/data of the target database with the
# contents of a custom-format dump produced by scripts/backup.sh. Because it is
# destructive it REFUSES to run without an explicit --confirm flag.
#
# Chesed uses PostgreSQL 16. The dump is restored with:
#   pg_restore --clean --if-exists --no-owner -d "$TARGET_URL" <dump>
#
# The dump's sha256 sidecar is verified BEFORE any change is made to the target.
# See docs/runbooks/backup-and-dr.md for the full recovery procedure and
# docs/14-deployment-strategy.md ("Backup and Disaster Recovery") for RPO/RTO.
#
# Usage:
#   scripts/restore.sh --confirm --dump <path-to-.dump> --target <postgres-url>
#   scripts/restore.sh --confirm --dump <path>          # target from env
#
# Target connection precedence (owner/admin — NOT the RLS role chesed_app):
#   --target <url>  >  RESTORE_TARGET_URL  >  ADMIN_DATABASE_URL  >  DATABASE_URL
#
# Flags:
#   --confirm            REQUIRED. Acknowledge that the target will be overwritten.
#   --dump <path>        REQUIRED. Path to the .dump file (its .sha256 must exist).
#   --target <url>       Optional. Target postgres:// URL (else from env, above).
#   --skip-checksum      Optional. Skip sha256 verification (NOT recommended).
#   -h | --help          Show this help.
#
# Exit codes: 0 = restore succeeded, non-zero = refused or failed.
# =============================================================================

set -euo pipefail

CONFIRM="no"
DUMP=""
TARGET_URL=""
SKIP_CHECKSUM="no"

log()  { printf '%s [restore.sh] %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*" >&2; }
fail() { log "ERROR: $*"; exit 1; }

require_tool() { command -v "$1" >/dev/null 2>&1 || fail "required tool not found on PATH: $1"; }

sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  else
    fail "no sha256 tool found (need sha256sum or shasum)."
  fi
}

usage() {
  grep -E '^# ' "$0" | sed 's/^# \{0,1\}//'
}

# -----------------------------------------------------------------------------
# Argument parsing
# -----------------------------------------------------------------------------
while [ "$#" -gt 0 ]; do
  case "$1" in
    --confirm)       CONFIRM="yes"; shift ;;
    --dump)          DUMP="${2:-}"; shift 2 || fail "--dump requires a path" ;;
    --dump=*)        DUMP="${1#*=}"; shift ;;
    --target)        TARGET_URL="${2:-}"; shift 2 || fail "--target requires a URL" ;;
    --target=*)      TARGET_URL="${1#*=}"; shift ;;
    --skip-checksum) SKIP_CHECKSUM="yes"; shift ;;
    -h|--help|help)  usage; exit 0 ;;
    *) fail "unknown argument: $1 (see --help)" ;;
  esac
done

# -----------------------------------------------------------------------------
# Guards
# -----------------------------------------------------------------------------
if [ "$CONFIRM" != "yes" ]; then
  log "REFUSING to run: restore is DESTRUCTIVE and overwrites the target database."
  log "Re-run with --confirm to proceed. See --help for usage."
  exit 2
fi

[ -n "$DUMP" ] || fail "--dump <path> is required."
[ -f "$DUMP" ] || fail "dump file not found: ${DUMP}"

# Resolve target connection.
if [ -z "$TARGET_URL" ]; then
  TARGET_URL="${RESTORE_TARGET_URL:-${ADMIN_DATABASE_URL:-${DATABASE_URL:-}}}"
fi
[ -n "$TARGET_URL" ] || fail "no target set. Pass --target <url> or set \
RESTORE_TARGET_URL / ADMIN_DATABASE_URL / DATABASE_URL."

require_tool pg_restore

# -----------------------------------------------------------------------------
# Checksum verification (before touching the target)
# -----------------------------------------------------------------------------
if [ "$SKIP_CHECKSUM" = "yes" ]; then
  log "WARNING: checksum verification skipped by --skip-checksum."
else
  side="${DUMP}.sha256"
  [ -f "$side" ] || fail "checksum sidecar missing: ${side} (use --skip-checksum to override)."
  expected="$(awk '{print $1}' "$side")"
  actual="$(sha256_of "$DUMP")"
  [ "$expected" = "$actual" ] || fail "checksum mismatch for ${DUMP} (expected ${expected}, got ${actual})."
  log "checksum verified for ${DUMP}"
fi

# -----------------------------------------------------------------------------
# Restore
# -----------------------------------------------------------------------------
# Redact any credentials in the URL for logging (strip user:pass@).
redacted_target="$(printf '%s' "$TARGET_URL" | sed -E 's#(://)[^@/]*@#\1***@#')"
log "restoring ${DUMP} into ${redacted_target}"
log "using: pg_restore --clean --if-exists --no-owner --no-privileges"

# --clean --if-exists : drop existing objects first, idempotent on re-run
# --no-owner          : do not restore ownership (role-agnostic)
# --no-privileges     : do not restore GRANTs (RLS grants applied by migrations)
pg_restore --clean --if-exists --no-owner --no-privileges \
  --dbname="$TARGET_URL" "$DUMP"

log "restore completed successfully."
log "REMINDER: run database migrations to re-apply RLS roles/grants, and \
re-import keycloak/realm-export.json if rebuilding Keycloak. See the runbook."
