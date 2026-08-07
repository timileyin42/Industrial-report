#!/usr/bin/env bash
# Dumps the local TimescaleDB container to a timestamped file under
# backups/ (gitignored) and prunes anything older than $RETAIN_DAYS.
#
# This backs up the LOCAL docker-compose database only. It does not upload
# anywhere — an off-site/cloud destination is a deployment-time decision
# this script deliberately doesn't invent (see docs/decisions and
# README's Phase 4 section). Restores are verified by scripts/restore-drill.sh
# — which is not optional ceremony: a plain pg_dump/pg_restore of a
# TimescaleDB hypertable does NOT restore cleanly without
# timescaledb_pre_restore()/post_restore() on the restore side. A pg_dump
# warning here about "circular foreign-key constraints on continuous_agg"
# is Timescale's own internal catalog table, not this project's schema —
# benign, and expected on every run.
set -euo pipefail

CONTAINER="${TIMESCALE_CONTAINER:-zgnis-timescaledb}"
DB_USER="${DB_USER:-zgnis}"
DB_NAME="${DB_NAME:-zgnis_solar}"
RETAIN_DAYS="${RETAIN_DAYS:-14}"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKUP_DIR="$REPO_ROOT/backups"
mkdir -p "$BACKUP_DIR"

TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
OUT_FILE="$BACKUP_DIR/zgnis_solar-$TIMESTAMP.dump"

echo "Backing up $DB_NAME from container $CONTAINER -> $OUT_FILE"
docker exec "$CONTAINER" pg_dump -U "$DB_USER" -d "$DB_NAME" -Fc > "$OUT_FILE"
echo "Backup complete: $(du -h "$OUT_FILE" | cut -f1)"

echo "Pruning backups older than $RETAIN_DAYS days..."
find "$BACKUP_DIR" -name "zgnis_solar-*.dump" -mtime "+$RETAIN_DAYS" -print -delete
