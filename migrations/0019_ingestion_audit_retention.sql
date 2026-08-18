-- +goose Up
-- ingestion_audit_log had no retention at all, and (separately, fixed in
-- application code alongside this migration) was logging the full raw
-- payload of every single successful ingestion, not just the rejected/
-- anomalous cases it exists for — 23,617 rows / ~14MB after under a
-- week with only 3 devices polling every 30s, nearly all routine
-- duplicates of data already safely in telemetry. Converting to a
-- hypertable (migrating existing rows into chunks) lets a retention
-- policy prune it automatically, same mechanism already used for
-- telemetry in migration 0006.
--
-- The existing primary key is on id alone, which doesn't include the
-- partitioning column — TimescaleDB requires any unique constraint on a
-- hypertable to include it (global uniqueness can't be enforced across
-- chunks otherwise), so it must be dropped first. id stays as a plain
-- identity column; nothing references it via foreign key.
ALTER TABLE ingestion_audit_log DROP CONSTRAINT ingestion_audit_log_pkey;

SELECT create_hypertable('ingestion_audit_log', 'received_at', migrate_data => true, if_not_exists => true);

-- 90 days — a diagnostic/forensic log of ingestion problems doesn't need
-- years of history the way telemetry does; 90 days covers investigating
-- a device that started misbehaving weeks ago without keeping this
-- table growing forever.
SELECT add_retention_policy('ingestion_audit_log', INTERVAL '90 days');

-- +goose Down
SELECT remove_retention_policy('ingestion_audit_log');
-- Hypertable conversion itself isn't reverted — same as migration 0006,
-- going back to a plain table would require copying all chunk data back
-- out, which isn't worth supporting for a Down path.
