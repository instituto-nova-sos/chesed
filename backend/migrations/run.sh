#!/bin/sh
set -e

MAX_RETRIES=30
RETRY_INTERVAL=2

echo "==> Waiting for PostgreSQL to be ready..."
# `migrate version` returns a non-zero exit with "no migration" on a fresh,
# reachable database — that still means Postgres is UP, just not migrated yet.
# So we treat connectivity errors (not "no migration") as "not ready" instead of
# relying on the version exit code alone.
i=0
until ! migrate -path ./migrations -database "$DATABASE_URL" version 2>&1 \
  | grep -qiE "connection refused|could not connect|no such host|dial tcp|connect: |timeout"; do
  i=$((i + 1))
  if [ "$i" -ge "$MAX_RETRIES" ]; then
    echo "FATAL: PostgreSQL not reachable after ${MAX_RETRIES} attempts"
    exit 1
  fi
  echo "    Attempt $i/$MAX_RETRIES — retrying in ${RETRY_INTERVAL}s..."
  sleep "$RETRY_INTERVAL"
done

echo "==> PostgreSQL is ready."
echo "==> Current migration version:"
migrate -path ./migrations -database "$DATABASE_URL" version 2>&1 || true

echo "==> Running migrations..."
if migrate -path ./migrations -database "$DATABASE_URL" up 2>&1; then
  echo "==> Migrations completed successfully."
  migrate -path ./migrations -database "$DATABASE_URL" version 2>&1
  exit 0
fi

echo "==> Migration failed. Checking for dirty state..."
VERSION_OUTPUT=$(migrate -path ./migrations -database "$DATABASE_URL" version 2>&1 || true)

if echo "$VERSION_OUTPUT" | grep -q "dirty"; then
  DIRTY_VERSION=$(echo "$VERSION_OUTPUT" | grep -oE '[0-9]+')
  echo "==> Database is dirty at version $DIRTY_VERSION. Forcing version to clear dirty flag..."
  migrate -path ./migrations -database "$DATABASE_URL" force "$DIRTY_VERSION"
  echo "==> Retrying migrations..."
  migrate -path ./migrations -database "$DATABASE_URL" up 2>&1
  echo "==> Migrations completed successfully after dirty state recovery."
  migrate -path ./migrations -database "$DATABASE_URL" version 2>&1
else
  echo "FATAL: Migration failed (not a dirty-state issue). Check migration files and database state."
  exit 1
fi
