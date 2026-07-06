#!/bin/bash
set -e

# Creates the non-owner application login role used for PostgreSQL RLS
# (Sprint 9.1, Model A). The app connects as this role so it is subject to the
# row-level security policies; migrations run as the owner ($POSTGRES_USER) and
# bypass RLS. The password comes from POSTGRES_APP_PASSWORD (ops-provided, same
# trust layer as POSTGRES_PASSWORD) and defaults to a dev value; grants and the
# RLS policies themselves are applied by migration 000028.
APP_PASSWORD="${POSTGRES_APP_PASSWORD:-chesed_app}"

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    DO \$\$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'chesed_app') THEN
            CREATE ROLE chesed_app LOGIN PASSWORD '${APP_PASSWORD}';
        ELSE
            ALTER ROLE chesed_app LOGIN PASSWORD '${APP_PASSWORD}';
        END IF;
    END
    \$\$;
EOSQL
