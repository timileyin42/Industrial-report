-- +goose Up
-- Real household hybrid inverters (Chisage/Felicity/Extra Power — see
-- docs field-deployment guide) expose battery and solar-side telemetry
-- this platform never had a column for: PV (solar panel) output is
-- distinct from the inverter's AC output (the gap between the two is
-- exactly what reveals clipping or battery charging behavior), and
-- there's currently zero battery visibility at all. All nullable — an
-- existing device's payload (or any future non-hybrid, grid-tie-only
-- inverter) simply won't populate these, same "missing optional field
-- is valid, not an error" rule voltage_v already follows.
ALTER TABLE telemetry ADD COLUMN pv_power_kw       double precision;
ALTER TABLE telemetry ADD COLUMN battery_soc_pct   smallint;
ALTER TABLE telemetry ADD COLUMN battery_voltage_v double precision;
ALTER TABLE telemetry ADD COLUMN pv_voltage_v      double precision;
ALTER TABLE telemetry ADD COLUMN output_voltage_v  double precision;

-- inverter_brand/model live on devices (the actual registered telemetry
-- source), not sites — sites.inverter_make_model already exists as a
-- free-text site-level note, but a device-level brand is what the
-- dashboard's device-registry filtering and the register-profile
-- pattern actually key off. Nullable: not required at registration,
-- and irrelevant for any device that isn't a household hybrid inverter.
ALTER TABLE devices ADD COLUMN inverter_brand text;
ALTER TABLE devices ADD COLUMN inverter_model text;

-- +goose Down
ALTER TABLE devices DROP COLUMN IF EXISTS inverter_model;
ALTER TABLE devices DROP COLUMN IF EXISTS inverter_brand;

ALTER TABLE telemetry DROP COLUMN IF EXISTS output_voltage_v;
ALTER TABLE telemetry DROP COLUMN IF EXISTS pv_voltage_v;
ALTER TABLE telemetry DROP COLUMN IF EXISTS battery_voltage_v;
ALTER TABLE telemetry DROP COLUMN IF EXISTS battery_soc_pct;
ALTER TABLE telemetry DROP COLUMN IF EXISTS pv_power_kw;
