-- +goose Up
-- Sandbox now mirrors the full real payload shape, including rssi (see
-- domain.TelemetryPayload's own rssi field, added alongside this — a
-- real gap where the concept note's data model and the telemetry table
-- both had it, but nothing between them ever actually read it).
ALTER TABLE sandbox_readings ADD COLUMN rssi integer;

-- +goose Down
ALTER TABLE sandbox_readings DROP COLUMN IF EXISTS rssi;
