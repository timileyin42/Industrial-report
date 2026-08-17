-- +goose Up
-- telemetry_daily was materialized_only (real-time aggregation off),
-- which meant querying it for "today" always returned nothing until an
-- explicit refresh completed a whole calendar day — but a day-bucketed
-- continuous aggregate's scheduled refresh policy deliberately never
-- materializes the current, still-in-progress day (there's no complete
-- bucket to materialize until the day is over), so every dashboard
-- figure for "today" silently showed stale/empty data despite real
-- telemetry arriving continuously. Enabling real-time aggregation makes
-- queries against the view transparently blend the materialized
-- history with a live-computed aggregate over today's raw telemetry, so
-- "today" is always current without needing a manual refresh.
ALTER MATERIALIZED VIEW telemetry_daily SET (timescaledb.materialized_only = false);

-- +goose Down
ALTER MATERIALIZED VIEW telemetry_daily SET (timescaledb.materialized_only = true);
