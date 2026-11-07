#!/usr/bin/env bash
set -euo pipefail

database="${1:?Usage: bash scripts/migrate.sh DATABASE_NAME}"
project_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
sql_file=$(mktemp)
trap 'rm -f "$sql_file"' EXIT

cat > "$sql_file" <<'SQL'
SET LOCAL lock_timeout = '15s';
SELECT pg_advisory_xact_lock(817204631);
CREATE TABLE IF NOT EXISTS schema_migrations (
    version text PRIMARY KEY,
    applied_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);
SQL

for migration in "$project_dir"/backend/migrations/*.sql; do
    version=$(basename "$migration")

    if [[ ! "$version" =~ ^[0-9]{3}_[a-z0-9_]+\.sql$ ]]; then
        echo "Invalid migration filename: $version" >&2
        exit 1
    fi

    cat >> "$sql_file" <<SQL
SELECT EXISTS (
    SELECT 1 FROM schema_migrations WHERE version = '$version'
) AS already_applied \gset
\if :already_applied
\echo Skipping $version
\else
\echo Applying $version
SQL

    cat "$migration" >> "$sql_file"
    printf '\n' >> "$sql_file"

    cat >> "$sql_file" <<SQL
INSERT INTO schema_migrations (version) VALUES ('$version');
\endif
SQL
done

docker compose exec -T db     psql -X -w -v ON_ERROR_STOP=1 --single-transaction     -U todo -d "$database" < "$sql_file"
