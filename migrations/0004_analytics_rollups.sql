-- +goose Up
-- +goose NO TRANSACTION

-- Daily, per-device rollup — the single physical aggregate every Phase 3
-- weekly/monthly/site/fleet analytics view is computed from in Go, rather
-- than a second physical rollup per window (concept note §10's own design
-- note: "keep each metric's definition consistent... single source of
-- truth"). Built now rather than deferred again — AGENTS.md has flagged
-- since Phase 1 that roll-ups were meant to exist, and every Phase 3 KPI
-- is fundamentally "aggregate telemetry by day."
CREATE MATERIALIZED VIEW telemetry_daily
WITH (timescaledb.continuous) AS
SELECT
    device_id,
    site_id,
    time_bucket('1 day', ts) AS day,
    max(power_kw)            AS peak_power_kw,
    min(energy_kwh_total)    AS energy_start_kwh,
    max(energy_kwh_total)    AS energy_end_kwh,
    count(*)                 AS reading_count,
    count(*) FILTER (WHERE provenance = 'backfilled') AS backfilled_count,
    bool_or('energy_reset' = ANY(quality_flags))      AS has_reset
FROM telemetry
GROUP BY device_id, site_id, day
WITH NO DATA;

-- start_offset absorbs backfilled/out-of-order arrivals (concept note §11);
-- end_offset is acceptable staleness for daily-grain analytics — raw
-- telemetry still serves the existing live Phase 0/2 endpoints unchanged.
SELECT add_continuous_aggregate_policy('telemetry_daily',
    start_offset => INTERVAL '3 days',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour');

-- +goose Down
SELECT remove_continuous_aggregate_policy('telemetry_daily', if_exists => true);
DROP MATERIALIZED VIEW IF EXISTS telemetry_daily;
