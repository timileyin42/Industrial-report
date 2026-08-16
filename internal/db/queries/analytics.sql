-- name: ListSiteDailyRollup :many
-- Per-device-per-day rows for one site, from the telemetry_daily continuous
-- aggregate. The registry layer sums across a site's devices per day (a
-- site can have more than one device) and falls back to
-- GetRawEnergyReadingsForDeviceDay for any device-day where has_reset is
-- true, rather than trusting energy_end_kwh - energy_start_kwh blindly
-- across a counter reset.
SELECT device_id, day, peak_power_kw, energy_start_kwh, energy_end_kwh, reading_count, backfilled_count, has_reset
FROM telemetry_daily
WHERE site_id = $1 AND day >= $2 AND day <= $3
ORDER BY day, device_id;

-- name: ListFleetDailyRollup :many
-- Same shape as ListSiteDailyRollup but fleet-wide, optionally filtered to
-- a cohort. Used by fleet energy/trends/benchmark/anomaly registry logic.
SELECT device_id, site_id, day, peak_power_kw, energy_start_kwh, energy_end_kwh, reading_count, backfilled_count, has_reset
FROM telemetry_daily
WHERE day >= $1 AND day <= $2
  AND (
    sqlc.narg('cohort_id')::text IS NULL
    OR site_id IN (SELECT site_id FROM sites WHERE cohort_id = sqlc.narg('cohort_id'))
  )
ORDER BY day, site_id, device_id;

-- name: GetFleetPowerCurve :many
-- Intraday "power right now, over time" curve — raw telemetry bucketed
-- to 5-minute intervals and averaged across every device, optionally
-- scoped to a cohort. Distinct from telemetry_daily (one row per
-- calendar day): this reads the raw hypertable directly since the
-- window is always short (a day or so), so pre-materializing it isn't
-- worth the extra continuous aggregate. Feeds the Dashboard's
-- "Generation Overview" Day view — a real sunrise-to-sunset power
-- curve, not the daily energy totals every other fleet chart plots.
SELECT time_bucket('5 minutes'::interval, ts)::timestamptz AS bucket, avg(power_kw)::double precision AS avg_power_kw
FROM telemetry
WHERE ts >= sqlc.arg('from')::timestamptz AND ts <= sqlc.arg('to')::timestamptz
  AND (
    sqlc.narg('cohort_id')::text IS NULL
    OR site_id IN (SELECT site_id FROM sites WHERE cohort_id = sqlc.narg('cohort_id'))
  )
GROUP BY bucket
ORDER BY bucket;

-- name: GetRawEnergyReadingsForDeviceDay :many
-- Reset-day fallback: ordered readings for one device on one day. The
-- registry computes true daily energy as sum(max(0, e[i] - e[i-1])) over
-- consecutive readings — this correctly captures both the pre-reset and
-- post-reset segments' real generation while ignoring the reset's negative
-- delta, without needing to know exactly when in the day the reset
-- occurred.
SELECT ts, energy_kwh_total FROM telemetry
WHERE device_id = $1 AND ts >= $2 AND ts < $3
ORDER BY ts;

-- name: GetPeakReadingTimeForDeviceDay :one
-- Continuous aggregates can't express argmax, so the peak reading's time
-- of day is resolved via a narrow indexed point lookup instead of a scan.
SELECT ts FROM telemetry
WHERE device_id = $1 AND ts >= $2 AND ts < $3 AND power_kw = $4
ORDER BY ts LIMIT 1;

-- name: ListSitesForAnalytics :many
-- Unpaginated site metadata used only as an internal join for
-- benchmarking/trends/comparisons (system size, inverter model, cohort).
-- Not itself an API list endpoint a client paginates — the user-facing
-- pagination is on the computed segments/periods, which stay small and
-- bounded regardless of fleet size at the scale this stack targets today.
SELECT site_id, name, timezone, system_size_kw, inverter_make_model, cohort_id, created_at
FROM sites
ORDER BY site_id;
