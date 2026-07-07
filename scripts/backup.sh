#!/usr/bin/env bash
# =============================================================================
# backup.sh — Logical backup + restore drill for the Chesed PostgreSQL database
# =============================================================================
# Chesed uses PostgreSQL 16. This script is the committable, self-contained piece
# of the backup & disaster-recovery story (Sprint 11, S12.3). It is triggered
# EXTERNALLY (cron / CI / a scheduled job) — there is no in-app scheduler, by
# design. See docs/runbooks/backup-and-dr.md for the operational procedure and
# docs/14-deployment-strategy.md ("Backup and Disaster Recovery") for RPO/RTO.
#
# Subcommands:
#   backup   (default)  pg_dump the database to a custom-format .dump + .sha256
#   drill               restore the latest dump into a THROWAWAY database and run
#                       a smoke query to PROVE the dump is restorable (PASS/FAIL)
#
# Usage:
#   scripts/backup.sh [backup]
#   scripts/backup.sh drill
#
# Connection (owner/admin — NOT the non-owner RLS role chesed_app):
#   Either provide a full URL ...
#     DATABASE_URL=postgres://chesed:chesed@localhost:5432/chesed?sslmode=disable
#   (ADMIN_DATABASE_URL is accepted as an alias and takes precedence if set) ...
#   ... or provide discrete libpq variables:
#     PGHOST PGPORT PGUSER PGDATABASE PGPASSWORD
#
# Environment (all optional, with defaults):
#   BACKUP_DIR              Directory for dumps            (default: ./backups)
#   BACKUP_RETENTION_DAYS   Prune dumps older than N days  (default: 30)
#   DRILL_DB_NAME           Throwaway DB name for `drill`  (default: chesed_drill_<pid>)
#
# Exit codes: 0 = success/PASS, non-zero = failure/FAIL.
# =============================================================================

set -euo pipefail

# -----------------------------------------------------------------------------
# Configuration
# -----------------------------------------------------------------------------
BACKUP_DIR="${BACKUP_DIR:-./backups}"
BACKUP_RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-30}"
DUMP_PREFIX="chesed-"
DUMP_EXT=".dump"

# Core tables the drill smoke-tests to prove the dump is restorable.
SMOKE_TABLES=("campus" "person" "triage" "audit_log")

log()  { printf '%s [backup.sh] %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*" >&2; }
fail() { log "ERROR: $*"; exit 1; }

# -----------------------------------------------------------------------------
# Connection resolution
# -----------------------------------------------------------------------------
# Resolve the admin/owner connection into a single connection string that the
# libpq tools (pg_dump, pg_restore, psql, createdb, dropdb) all understand.
#
# Precedence: ADMIN_DATABASE_URL > DATABASE_URL > discrete PG* variables.
# Backups MUST run as the owner/admin, never as the non-owner RLS role, so a
# custom-format dump captures every row regardless of row-level security.
resolve_conn() {
  if [ -n "${ADMIN_DATABASE_URL:-}" ]; then
    CONN_URL="$ADMIN_DATABASE_URL"
  elif [ -n "${DATABASE_URL:-}" ]; then
    CONN_URL="$DATABASE_URL"
  elif [ -n "${PGHOST:-}" ] && [ -n "${PGUSER:-}" ] && [ -n "${PGDATABASE:-}" ]; then
    # Discrete libpq variables are already exported; pg_* tools read them
    # directly. We leave CONN_URL empty and rely on the environment.
    CONN_URL=""
    : "${PGPORT:=5432}"
    export PGPORT
  else
    fail "no connection configured. Set DATABASE_URL (or ADMIN_DATABASE_URL), \
or PGHOST/PGPORT/PGUSER/PGDATABASE (+ PGPASSWORD)."
  fi
}

# psql/pg_dump/pg_restore invocation that works with either a URL or PG* vars.
# Usage: run_psql <extra args...>  |  the connection target is prepended.
psql_admin()       { if [ -n "$CONN_URL" ]; then psql "$CONN_URL" "$@"; else psql "$@"; fi; }
pg_dump_admin()    { if [ -n "$CONN_URL" ]; then pg_dump "$CONN_URL" "$@"; else pg_dump "$@"; fi; }

require_tool() { command -v "$1" >/dev/null 2>&1 || fail "required tool not found on PATH: $1"; }

# sha256 helper — prefer sha256sum (Linux), fall back to shasum (macOS/BSD).
sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  else
    fail "no sha256 tool found (need sha256sum or shasum)."
  fi
}

# Return the newest *.dump in BACKUP_DIR (by name — timestamps sort lexically).
latest_dump() {
  ls -1 "${BACKUP_DIR}/${DUMP_PREFIX}"*"${DUMP_EXT}" 2>/dev/null | sort | tail -n 1
}

# -----------------------------------------------------------------------------
# Subcommand: backup
# -----------------------------------------------------------------------------
cmd_backup() {
  require_tool pg_dump
  resolve_conn

  mkdir -p "$BACKUP_DIR"
  local ts dump sha
  ts="$(date -u +%Y%m%dT%H%M%SZ)"
  dump="${BACKUP_DIR}/${DUMP_PREFIX}${ts}${DUMP_EXT}"

  log "starting pg_dump -> ${dump}"
  # --format=custom  : compressed, selective restore, parallelizable
  # --no-owner       : restorable into any role (owner is re-created by migrations)
  # --no-privileges  : drop GRANTs (RLS role/grants are applied by migrations)
  pg_dump_admin --format=custom --no-owner --no-privileges --file="$dump"

  sha="$(sha256_of "$dump")"
  # Write the sidecar as "<hash>  <basename>" so `sha256sum -c` works from BACKUP_DIR.
  printf '%s  %s\n' "$sha" "$(basename "$dump")" > "${dump}.sha256"
  log "wrote checksum ${sha}  ${dump}.sha256"

  prune_old
  offsite_copy_hook "$dump"

  log "backup OK: ${dump}"
  printf '%s\n' "$dump"
}

# Prune dumps + sidecars older than BACKUP_RETENTION_DAYS. Never touches
# anything outside BACKUP_DIR and only matches our own naming pattern.
prune_old() {
  [ -d "$BACKUP_DIR" ] || return 0
  log "pruning dumps older than ${BACKUP_RETENTION_DAYS} day(s) in ${BACKUP_DIR}"
  # -mtime +N removes files strictly older than N*24h.
  find "$BACKUP_DIR" -maxdepth 1 -type f \
    -name "${DUMP_PREFIX}*${DUMP_EXT}" \
    -mtime +"${BACKUP_RETENTION_DAYS}" -print -delete || true
  find "$BACKUP_DIR" -maxdepth 1 -type f \
    -name "${DUMP_PREFIX}*${DUMP_EXT}.sha256" \
    -mtime +"${BACKUP_RETENTION_DAYS}" -print -delete || true
}

# -----------------------------------------------------------------------------
# Off-site copy HOOK (DISABLED by default — manual / env-gated)
# -----------------------------------------------------------------------------
# Copying dumps off-site is a deliberate, credentialed step and is NOT executed
# by default so the drill stays hermetic and free of cloud dependencies. To
# enable, set BACKUP_OFFSITE_URI (e.g. s3://my-bucket/chesed/ or a remote:path
# for rclone) and provide the corresponding credentials via the environment
# (never hardcode secrets). Uncomment the tool you use.
offsite_copy_hook() {
  local dump="$1"
  if [ -z "${BACKUP_OFFSITE_URI:-}" ]; then
    log "off-site copy skipped (BACKUP_OFFSITE_URI not set)"
    return 0
  fi
  log "off-site copy requested -> ${BACKUP_OFFSITE_URI}"
  # --- Choose ONE. Credentials come from the environment/instance role. ---
  # aws s3 cp "$dump"           "${BACKUP_OFFSITE_URI}"
  # aws s3 cp "${dump}.sha256"  "${BACKUP_OFFSITE_URI}"
  # rclone copy "$dump"          "${BACKUP_OFFSITE_URI}"
  # rclone copy "${dump}.sha256" "${BACKUP_OFFSITE_URI}"
  log "NOTE: off-site copy is a documented no-op in this script; enable a \
transfer command above intentionally. See docs/runbooks/backup-and-dr.md."
}

# -----------------------------------------------------------------------------
# Subcommand: drill
# -----------------------------------------------------------------------------
# Restore the latest dump into a THROWAWAY database and prove it is queryable.
# This is what `make backup-drill` runs against the compose Postgres. It never
# touches the live database and always cleans up the throwaway DB, even on error.
cmd_drill() {
  require_tool pg_restore
  require_tool psql
  require_tool createdb
  require_tool dropdb
  resolve_conn

  local dump
  dump="$(latest_dump)" || true
  [ -n "${dump:-}" ] && [ -f "$dump" ] || fail "no dump found in ${BACKUP_DIR} to drill. Run 'backup.sh backup' first."

  verify_checksum "$dump"

  local drill_db admin_maint
  drill_db="${DRILL_DB_NAME:-chesed_drill_$$}"
  admin_maint="$(admin_conn_for_db postgres)"

  log "restore drill: dump=${dump} throwaway-db=${drill_db}"

  # Ensure a clean slate, then create the throwaway DB. Always drop it on exit.
  cleanup_drill() {
    log "cleanup: dropping throwaway database ${drill_db}"
    psql "$admin_maint" -v ON_ERROR_STOP=0 -qtAc \
      "DROP DATABASE IF EXISTS \"${drill_db}\" WITH (FORCE);" >/dev/null 2>&1 || \
    psql "$admin_maint" -v ON_ERROR_STOP=0 -qtAc \
      "DROP DATABASE IF EXISTS \"${drill_db}\";" >/dev/null 2>&1 || true
  }
  trap cleanup_drill EXIT

  psql "$admin_maint" -v ON_ERROR_STOP=1 -qtAc \
    "DROP DATABASE IF EXISTS \"${drill_db}\";" >/dev/null
  psql "$admin_maint" -v ON_ERROR_STOP=1 -qtAc \
    "CREATE DATABASE \"${drill_db}\";" >/dev/null
  log "created throwaway database ${drill_db}"

  local target
  target="$(admin_conn_for_db "$drill_db")"

  # Restore into the throwaway DB. --no-owner keeps it role-agnostic. We do not
  # use --clean here because the DB is brand new. Exit status is captured so we
  # can emit an explicit FAIL rather than aborting on a partial-restore warning.
  if ! pg_restore --no-owner --no-privileges --exit-on-error \
        --dbname="$target" "$dump"; then
    log "DRILL: FAIL (pg_restore reported errors)"
    exit 1
  fi
  log "restore into ${drill_db} completed; running smoke queries"

  local t count
  for t in "${SMOKE_TABLES[@]}"; do
    if ! count="$(psql "$target" -v ON_ERROR_STOP=1 -qtAc "SELECT count(*) FROM ${t};" 2>/dev/null)"; then
      log "DRILL: FAIL (smoke query failed for table '${t}')"
      exit 1
    fi
    log "smoke: SELECT count(*) FROM ${t} => ${count}"
  done

  log "DRILL: PASS (dump ${dump} is restorable and core tables are queryable)"
  printf 'DRILL: PASS %s\n' "$dump"
}

# Verify the .sha256 sidecar matches the dump before trusting it.
verify_checksum() {
  local dump="$1" side expected actual
  side="${dump}.sha256"
  [ -f "$side" ] || fail "checksum sidecar missing: ${side}"
  expected="$(awk '{print $1}' "$side")"
  actual="$(sha256_of "$dump")"
  [ "$expected" = "$actual" ] || fail "checksum mismatch for ${dump} (expected ${expected}, got ${actual})"
  log "checksum verified for ${dump}"
}

# Build an admin connection string pointed at a specific database (dbname),
# preserving host/port/user/credentials from CONN_URL or the PG* environment.
# When CONN_URL is a URL we swap only the path (database) component.
admin_conn_for_db() {
  local db="$1"
  if [ -n "$CONN_URL" ]; then
    swap_db_in_url "$CONN_URL" "$db"
  else
    # Discrete PG* vars: psql accepts a keyword/value connstring; PGPASSWORD is
    # read from the environment automatically and is never echoed.
    printf 'host=%s port=%s user=%s dbname=%s' \
      "${PGHOST}" "${PGPORT:-5432}" "${PGUSER}" "${db}"
  fi
}

# Replace the database (path) segment of a postgres:// URL while keeping the
# query string (e.g. ?sslmode=disable) intact. Pure shell parameter expansion,
# no external processes, so it is safe with credentials in the URL.
swap_db_in_url() {
  local url="$1" newdb="$2" scheme rest query hostpart
  case "$url" in
    postgres://*)   scheme="postgres://";   rest="${url#postgres://}" ;;
    postgresql://*) scheme="postgresql://"; rest="${url#postgresql://}" ;;
    *) fail "unsupported connection URL scheme (expected postgres:// or postgresql://)" ;;
  esac
  # Split off the query string, if any.
  query=""
  case "$rest" in *\?*) query="?${rest#*\?}"; rest="${rest%%\?*}" ;; esac
  # rest is now user:pass@host:port/dbname ; strip the /dbname tail.
  hostpart="${rest%%/*}"
  printf '%s%s/%s%s' "$scheme" "$hostpart" "$newdb" "$query"
}

# -----------------------------------------------------------------------------
# Dispatch
# -----------------------------------------------------------------------------
main() {
  local sub="${1:-backup}"
  case "$sub" in
    backup) shift || true; cmd_backup "$@" ;;
    drill)  shift || true; cmd_drill "$@" ;;
    -h|--help|help)
      grep -E '^# ' "$0" | sed 's/^# \{0,1\}//'
      ;;
    *) fail "unknown subcommand: ${sub} (expected: backup | drill)" ;;
  esac
}

main "$@"
