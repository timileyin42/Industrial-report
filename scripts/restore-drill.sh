#!/usr/bin/env bash
# Restores the most recent backup (from scripts/backup.sh) into a scratch
# TimescaleDB container, runs sanity checks, then tears the scratch
# container down. This is the actual "restore actually tested" drill —
# CLAUDE.md is explicit that an untested backup is an assumption, not a
# backup.
set -euo pipefail

DB_USER="${DB_USER:-zgnis}"
DB_PASSWORD="${DB_PASSWORD:-zgnis_dev_only}"
DB_NAME="${DB_NAME:-zgnis_solar}"
SCRATCH_CONTAINER="zgnis-restore-drill"
SCRATCH_PORT="${SCRATCH_PORT:-5433}"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKUP_DIR="$REPO_ROOT/backups"

LATEST="$(ls -t "$BACKUP_DIR"/zgnis_solar-*.dump 2>/dev/null | head -1 || true)"
if [ -z "$LATEST" ]; then
  echo "No backup found in $BACKUP_DIR — run scripts/backup.sh first." >&2
  exit 1
fi
echo "Restoring from: $LATEST"

cleanup() {
  echo "Tearing down scratch container $SCRATCH_CONTAINER..."
  docker rm -f "$SCRATCH_CONTAINER" >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "Starting scratch TimescaleDB container on port $SCRATCH_PORT..."
docker run -d --name "$SCRATCH_CONTAINER" \
  -e POSTGRES_USER="$DB_USER" -e POSTGRES_PASSWORD="$DB_PASSWORD" -e POSTGRES_DB="$DB_NAME" \
  -p "$SCRATCH_PORT:5432" \
  timescale/timescaledb:latest-pg16 >/dev/null

echo "Waiting for scratch container to accept connections..."
for _ in $(seq 1 30); do
  if docker exec "$SCRATCH_CONTAINER" pg_isready -U "$DB_USER" -d "$DB_NAME" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

docker exec "$SCRATCH_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -c "CREATE EXTENSION IF NOT EXISTS timescaledb;" >/dev/null

# A plain pg_restore against a hypertable dump fails outright — Timescale's
# own catalog needs to be put into a restore-aware state first. This is
# NOT optional ceremony: the first version of this script skipped these
# and the restore genuinely failed (COPY errors on hypertable chunks,
# "ONLY option not supported on hypertable operations"). Discovering that
# IS the point of a real restore drill, not something to paper over.
echo "Preparing Timescale catalog for restore..."
docker exec "$SCRATCH_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -c "SELECT timescaledb_pre_restore();" >/dev/null

echo "Restoring dump into scratch container..."
docker cp "$LATEST" "$SCRATCH_CONTAINER:/tmp/restore.dump"
docker exec "$SCRATCH_CONTAINER" pg_restore -U "$DB_USER" -d "$DB_NAME" --no-owner /tmp/restore.dump

echo "Finalizing Timescale catalog after restore..."
docker exec "$SCRATCH_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -c "SELECT timescaledb_post_restore();" >/dev/null

echo
echo "--- Sanity checks ---"
FAIL=0

check_nonzero() {
  local label="$1" query="$2"
  local count
  count="$(docker exec "$SCRATCH_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -tAc "$query")"
  echo "$label: $count"
  if [ "$count" -eq 0 ]; then
    echo "  FAIL: expected non-zero rows"
    FAIL=1
  fi
}

check_nonzero "sites"   "SELECT count(*) FROM sites;"
check_nonzero "devices" "SELECT count(*) FROM devices;"
check_nonzero "telemetry" "SELECT count(*) FROM telemetry;"

HYPERTABLE="$(docker exec "$SCRATCH_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -tAc \
  "SELECT count(*) FROM timescaledb_information.hypertables WHERE hypertable_name = 'telemetry';")"
echo "telemetry recognized as hypertable: $HYPERTABLE"
if [ "$HYPERTABLE" -eq 0 ]; then
  echo "  FAIL: telemetry is not a recognized hypertable after restore"
  FAIL=1
fi

echo
if [ "$FAIL" -eq 0 ]; then
  echo "RESTORE DRILL PASSED"
else
  echo "RESTORE DRILL FAILED — see failures above"
  exit 1
fi
