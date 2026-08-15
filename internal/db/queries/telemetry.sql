-- name: GetDeviceWithSiteContext :one
-- Same lookup cmd/ingestor/main.go runs as raw pgx on the MQTT hot path
-- (AGENTS.md's stack-split rationale doesn't apply here — this serves an
-- infrequent HTTP webhook, not a high-frequency broker subscription) —
-- used by the cloud-import path so both ingestion routes apply the exact
-- same revoked-device check and site-specific plausibility ceiling.
SELECT d.site_id, d.revoked_at, s.system_size_kw, s.timezone
FROM devices d JOIN sites s ON s.site_id = d.site_id
WHERE d.device_id = $1;

-- name: PreviousEnergyBeforeTS :one
-- Mirrors cmd/ingestor/main.go's previousEnergyByTS — chronological
-- lookup by ts, never by insertion order, so reset detection stays
-- correct under out-of-order/backfilled arrival (see domain.
-- DetectEnergyReset's own comment on why this matters).
SELECT energy_kwh_total FROM telemetry WHERE device_id = $1 AND ts < $2 ORDER BY ts DESC LIMIT 1;

-- name: InsertTelemetryReading :execrows
INSERT INTO telemetry (
    device_id, site_id, ts, power_kw, energy_kwh_total, voltage_v, status, provenance, quality_flags, rssi,
    pv_power_kw, battery_soc_pct, battery_voltage_v, pv_voltage_v, output_voltage_v
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
ON CONFLICT (device_id, ts) DO NOTHING;

-- name: ListTelemetryForSite :many
-- Keyset pagination on (ts, device_id) DESC. cursor_ts NULL means first page.
-- from_ts/to_ts NULL means unbounded on that side.
SELECT ts, power_kw, energy_kwh_total, voltage_v, status, device_id, rssi,
       pv_power_kw, battery_soc_pct, battery_voltage_v, pv_voltage_v, output_voltage_v
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
