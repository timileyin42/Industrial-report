-- +goose Up
-- Same pattern as migration 0016 (pv_power_kw, battery_soc_pct, etc.) —
-- hybrid inverters with a grid meter or load-monitoring CT expose
-- household consumption and grid import/export, which this platform
-- never had columns for despite already receiving them in PV Pro's raw
-- flow response (loadOrEpsPower, gridOrMeterPower). Nullable: a device
-- without a grid meter or load CT simply won't populate these, same
-- "missing optional field is valid, not an error" rule every other
-- hybrid-inverter field already follows.
ALTER TABLE telemetry ADD COLUMN load_power_kw double precision;
ALTER TABLE telemetry ADD COLUMN grid_power_kw double precision;

-- +goose Down
ALTER TABLE telemetry DROP COLUMN IF EXISTS grid_power_kw;
ALTER TABLE telemetry DROP COLUMN IF EXISTS load_power_kw;
