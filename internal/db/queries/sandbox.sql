-- name: CreateSandboxRun :one
INSERT INTO sandbox_runs (id, system_size_kw, row_count, accepted_count, rejected_count)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetSandboxRun :one
SELECT * FROM sandbox_runs WHERE id = $1;

-- name: CreateSandboxReading :exec
INSERT INTO sandbox_readings
    (run_id, row_number, ts, power_kw, energy_kwh_total, voltage_v, status, accepted, rejection_reason, provenance, is_reset, rssi)
VALUES
    ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);

-- name: ListSandboxReadings :many
-- Unpaginated by design — a run is capped at a small row count
-- (registry.MaxSandboxRows) when uploaded, so this always returns a
-- bounded result, same reasoning as ListSitesForAnalytics.
SELECT * FROM sandbox_readings WHERE run_id = $1 ORDER BY row_number;

-- name: DeleteOldSandboxRuns :exec
-- Lazy self-cleanup instead of a cron job: run once whenever a new
-- upload happens (see registry.Sandbox.Upload). This is public,
-- unauthenticated, and accumulates over time — nothing else ever purges
-- it otherwise.
DELETE FROM sandbox_runs WHERE created_at < $1;
