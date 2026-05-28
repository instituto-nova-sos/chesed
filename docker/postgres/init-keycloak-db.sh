#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE DATABASE chesed_keycloak;
    GRANT ALL PRIVILEGES ON DATABASE chesed_keycloak TO $POSTGRES_USER;
EOSQL
