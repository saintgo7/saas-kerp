#!/usr/bin/env bash
# Apply db/migrations/*.up.sql to the test database, in numeric order.
#
# Why this exists: docker-compose.test.yml used to mount the whole
# db/migrations directory as /docker-entrypoint-initdb.d. PostgreSQL executes
# every .sql in that directory in C-collation order, so for each migration N it
# ran `N_x.down.sql` BEFORE `N_x.up.sql` - 20 rollback scripts interleaved with
# the 20 forward ones. With ON_ERROR_STOP the first rollback aborted database
# initialisation; without it the resulting schema was arbitrary.
#
# TODO: replace with `cmd/migrate up` once the Go migration binary exists
# (owned by the Go side). This script deliberately does not maintain a
# schema_migrations table - it is for throwaway test databases only.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIGRATIONS_DIR="${REPO_ROOT}/db/migrations"

export PGHOST="${TEST_DB_HOST:-${DB_HOST:-postgres-test}}"
export PGPORT="${TEST_DB_PORT:-${DB_PORT:-5432}}"
export PGUSER="${TEST_DB_USER:-${DB_USER:-test}}"
export PGPASSWORD="${TEST_DB_PASSWORD:-${DB_PASSWORD:-test}}"
export PGDATABASE="${TEST_DB_NAME:-${DB_NAME:-kerp_test}}"

if [ ! -d "$MIGRATIONS_DIR" ]; then
    echo "ERROR: $MIGRATIONS_DIR not found" >&2
    exit 1
fi

count=0
while IFS= read -r file; do
    echo "--> $(basename "$file")"
    psql --set ON_ERROR_STOP=1 --quiet --no-psqlrc -f "$file"
    count=$((count + 1))
done < <(find "$MIGRATIONS_DIR" -maxdepth 1 -name '*.up.sql' | LC_ALL=C sort)

if [ "$count" -eq 0 ]; then
    echo "ERROR: no *.up.sql migrations found in $MIGRATIONS_DIR" >&2
    exit 1
fi

echo "Applied ${count} migration(s)."
