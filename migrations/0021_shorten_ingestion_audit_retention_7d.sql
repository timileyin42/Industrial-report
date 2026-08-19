-- +goose Up
-- Further shortened from 20 to 7 days — same reasoning as migration
-- 0020, taken further: this log's forensic value (investigating a
-- device's raw payload history after something went wrong) is most
-- useful in the first few days after an incident, and 7 days keeps the
-- steady-state size proportionally smaller as device count grows.
-- Doesn't affect telemetry/telemetry_daily (separate tables, separate
-- 2-year retention) or anything displayed on the dashboard — only the
-- admin-only Ingestion Log page's lookback window.
SELECT remove_retention_policy('ingestion_audit_log');
SELECT add_retention_policy('ingestion_audit_log', INTERVAL '7 days');

-- +goose Down
SELECT remove_retention_policy('ingestion_audit_log');
SELECT add_retention_policy('ingestion_audit_log', INTERVAL '20 days');
