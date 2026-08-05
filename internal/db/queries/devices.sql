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
