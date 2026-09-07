#!/usr/bin/env bash
# Block until the test PostgreSQL accepts connections.
#
# Referenced by Dockerfile.test (integration-test stage), which previously
# COPYd this path even though the file did not exist - so that stage could
# never build.

set -euo pipefail

HOST="${TEST_DB_HOST:-${DB_HOST:-postgres-test}}"
PORT="${TEST_DB_PORT:-${DB_PORT:-5432}}"
USER="${TEST_DB_USER:-${DB_USER:-test}}"
DB="${TEST_DB_NAME:-${DB_NAME:-kerp_test}}"
TIMEOUT="${DB_WAIT_TIMEOUT:-60}"

echo "Waiting for postgres at ${HOST}:${PORT} (db=${DB}, user=${USER}), timeout ${TIMEOUT}s"

deadline=$(( $(date +%s) + TIMEOUT ))
until pg_isready -h "$HOST" -p "$PORT" -U "$USER" -d "$DB" -q; do
    if [ "$(date +%s)" -ge "$deadline" ]; then
        echo "ERROR: postgres at ${HOST}:${PORT} did not become ready within ${TIMEOUT}s" >&2
        exit 1
    fi
    sleep 1
done

echo "postgres is ready"
