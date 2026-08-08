-- name: FleetHealthTotals :one
-- Ratio-of-sums coverage (sum(actual)/sum(expected) across all devices),
-- not an average of per-device percentages — avoids a brand-new device's
-- tiny expected-readings denominator distorting the fleet-wide figure.
-- Coverage percentage itself is computed in Go from these two sums to
-- avoid a division-by-zero edge case in SQL when there are no devices yet.
WITH device_stats AS (
    SELECT
        d.device_id,
        d.created_at,
        count(t.ts) AS actual_readings
    FROM devices d
    LEFT JOIN telemetry t
        ON t.device_id = d.device_id
       AND t.ts >= sqlc.arg('window_start')::timestamptz
       AND t.ts <= sqlc.arg('now')::timestamptz
    GROUP BY d.device_id, d.created_at
),
expected AS (
    SELECT
        actual_readings,
        -- greatest(window_start, created_at) avoids penalizing a device for
        -- the portion of the window before it existed.
        GREATEST(
            EXTRACT(EPOCH FROM (sqlc.arg('now')::timestamptz - GREATEST(sqlc.arg('window_start')::timestamptz, created_at)))
              / sqlc.arg('expected_interval_seconds')::float8,
            0
        ) AS expected_readings
    FROM device_stats
)
SELECT
    (SELECT count(*)::bigint FROM sites) AS total_sites,
    (SELECT count(*)::bigint FROM devices) AS total_devices,
    (SELECT count(*)::bigint FROM devices WHERE revoked_at IS NULL AND last_contact_at > sqlc.arg('online_cutoff')::timestamptz) AS online_devices,
    COALESCE(SUM(actual_readings), 0)::bigint AS total_actual_readings,
    COALESCE(SUM(expected_readings), 0)::float8 AS total_expected_readings
FROM expected;

-- name: ListSiteHealth :many
-- Per-site coverage breakdown, keyset-paginated on site_id (simple textual
-- keyset — site_id is already the sites PK, so no secondary tiebreaker is
-- needed the way created_at-sorted lists elsewhere need one).
WITH device_stats AS (
    SELECT
        d.device_id,
        d.site_id,
        d.created_at,
        d.last_seen_at,
        d.revoked_at,
        d.last_contact_at,
        count(t.ts) AS actual_readings
    FROM devices d
    LEFT JOIN telemetry t
        ON t.device_id = d.device_id
       AND t.ts >= sqlc.arg('window_start')::timestamptz
       AND t.ts <= sqlc.arg('now')::timestamptz
    GROUP BY d.device_id, d.site_id, d.created_at, d.last_seen_at, d.revoked_at, d.last_contact_at
),
per_device AS (
    SELECT
        device_id, site_id, last_seen_at, revoked_at, last_contact_at,
        actual_readings,
        GREATEST(
            EXTRACT(EPOCH FROM (sqlc.arg('now')::timestamptz - GREATEST(sqlc.arg('window_start')::timestamptz, created_at)))
              / sqlc.arg('expected_interval_seconds')::float8,
            0
        ) AS expected_readings
    FROM device_stats
)
SELECT
    s.site_id,
    s.name AS site_name,
    s.created_at AS site_created_at,
    count(pd.device_id)::bigint AS total_devices,
    count(pd.device_id) FILTER (
        WHERE pd.revoked_at IS NULL AND pd.last_contact_at > sqlc.arg('online_cutoff')::timestamptz
    )::bigint AS online_devices,
    COALESCE(SUM(pd.actual_readings), 0)::bigint AS actual_readings,
    COALESCE(SUM(pd.expected_readings), 0)::float8 AS expected_readings,
    MIN(pd.last_seen_at)::timestamptz AS worst_last_seen_at
FROM sites s
LEFT JOIN per_device pd ON pd.site_id = s.site_id
WHERE sqlc.narg('cursor_site_id')::text IS NULL OR s.site_id > sqlc.narg('cursor_site_id')
GROUP BY s.site_id, s.name, s.created_at
ORDER BY s.site_id
LIMIT sqlc.arg('page_limit');
