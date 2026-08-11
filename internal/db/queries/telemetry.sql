-- name: ListTelemetryForSite :many
-- Keyset pagination on (ts, device_id) DESC. cursor_ts NULL means first page.
-- from_ts/to_ts NULL means unbounded on that side.
SELECT ts, power_kw, energy_kwh_total, voltage_v, status, device_id, rssi
FROM telemetry
WHERE site_id = $1
  AND (sqlc.narg('from_ts')::timestamptz IS NULL OR ts >= sqlc.narg('from_ts'))
  AND (sqlc.narg('to_ts')::timestamptz IS NULL OR ts <= sqlc.narg('to_ts'))
  AND (
    sqlc.narg('cursor_ts')::timestamptz IS NULL
    OR (ts, device_id) < (sqlc.narg('cursor_ts')::timestamptz, sqlc.narg('cursor_device_id')::text)
  )
ORDER BY ts DESC, device_id DESC
LIMIT sqlc.arg('page_limit');
