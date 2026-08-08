-- name: CreateDevice :one
INSERT INTO devices (device_id, site_id, secret_hash, install_notes)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetDevice :one
SELECT * FROM devices WHERE device_id = $1;

-- name: ListDevices :many
-- site_filter NULL means "all devices" (operator, no ?site_id= filter).
SELECT * FROM devices
WHERE (sqlc.narg('site_filter')::text IS NULL OR site_id = sqlc.narg('site_filter'))
  AND (
    sqlc.narg('cursor_created_at')::timestamptz IS NULL
    OR (created_at, device_id) < (sqlc.narg('cursor_created_at')::timestamptz, sqlc.narg('cursor_device_id')::text)
  )
ORDER BY created_at DESC, device_id DESC
LIMIT sqlc.arg('page_limit');

-- name: RevokeDevice :one
UPDATE devices SET revoked_at = now() WHERE device_id = $1 RETURNING *;

-- name: RotateDeviceSecret :one
UPDATE devices SET secret_hash = $2, secret_last_rotated_at = now() WHERE device_id = $1 RETURNING *;

-- name: CountOnlineDevices :one
-- cutoff is computed in Go from ONLINE_THRESHOLD_MINUTES, never a SQL
-- literal, so the threshold is configurable without a query change.
-- Filters on last_contact_at (reachability), not last_seen_at (reading
-- freshness) — see internal/registry's two-signal online/data_gap model.
SELECT count(*)::bigint FROM devices
WHERE revoked_at IS NULL AND last_contact_at > sqlc.arg('cutoff')::timestamptz;

-- name: CountDevices :one
SELECT count(*)::bigint FROM devices;

-- name: CurrentFleetGeneration :one
-- Sum of the most recent power_kw reading per online (not revoked,
-- contacted within the online threshold) device — a live "how much
-- power right now" figure, distinct from any cumulative/historical
-- energy total elsewhere on this platform. A LATERAL join per device,
-- not a rollup — deliberately live, not pre-aggregated.
SELECT coalesce(sum(latest.power_kw), 0)::double precision AS current_power_kw
FROM devices d
JOIN LATERAL (
    SELECT power_kw FROM telemetry t WHERE t.device_id = d.device_id ORDER BY t.ts DESC LIMIT 1
) latest ON true
WHERE d.revoked_at IS NULL AND d.last_contact_at > sqlc.arg('online_cutoff')::timestamptz;

-- name: ListRecentlyRevokedDevices :many
-- Feeds the Alerts page — a revocation is a real, timestamped event
-- worth surfacing there, same as an offline/fault condition.
SELECT * FROM devices
WHERE revoked_at IS NOT NULL AND revoked_at > sqlc.arg('since')::timestamptz
ORDER BY revoked_at DESC;

-- name: ListRecentFaultReadings :many
-- Latest reading per device that reported status='fault' within the
-- window, one row per device (DISTINCT ON), for the Alerts page. A raw
-- scan over recent telemetry, not a rollup — telemetry_daily doesn't
-- track status, and this is bounded to a short recent window, not the
-- full history CLAUDE.md's "hit roll-ups" rule is about.
SELECT DISTINCT ON (device_id) device_id, site_id, ts, status
FROM telemetry
WHERE status = 'fault' AND ts > sqlc.arg('since')::timestamptz
ORDER BY device_id, ts DESC;
