-- +goose Up
-- Shortened from 90 to 20 days ahead of scaling from 3 devices to
-- ~19 — this table's growth rate scales with device count, and this
-- log's forensic value (investigating a device's raw payload history)
-- doesn't need 90 days to be useful. Doesn't affect telemetry/
-- telemetry_daily (separate tables, separate 2-year retention) or
-- anything displayed on the dashboard — only the admin-only Ingestion
-- Log page's lookback window.
SELECT remove_retention_policy('ingestion_audit_log');
SELECT add_retention_policy('ingestion_audit_log', INTERVAL '20 days');

-- +goose Down
SELECT remove_retention_policy('ingestion_audit_log');
SELECT add_retention_policy('ingestion_audit_log', INTERVAL '90 days');
