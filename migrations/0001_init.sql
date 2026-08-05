-- +goose Up

CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE sites (
    site_id             text PRIMARY KEY,
    cohort_id           text,
    address             text,
    gps_lat             double precision,
    gps_lng             double precision,
    inverter_make_model text,
    system_size_kw      numeric,
    install_date        date,
    timezone            text NOT NULL DEFAULT 'Africa/Lagos',
    created_at          timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE devices (
    device_id     text PRIMARY KEY,
    site_id       text REFERENCES sites(site_id),
    secret_hash   text NOT NULL,       -- bcrypt hash of the MQTT credential, never store plaintext
    revoked_at    timestamptz,         -- non-null = revoked, ingestor must reject
    last_seen_at  timestamptz,         -- updated on every accepted reading; basis for Phase 2 gap detection
    created_at    timestamptz NOT NULL DEFAULT now()
);

-- Provenance: never blend metered and backfilled/estimated data silently.
-- See concept note section 11 and AGENTS.md item 4.
CREATE TYPE provenance_type AS ENUM ('metered', 'estimated', 'backfilled');
CREATE TYPE reading_status AS ENUM ('ok', 'fault', 'offline');

CREATE TABLE telemetry (
    device_id         text NOT NULL REFERENCES devices(device_id),
    site_id           text NOT NULL REFERENCES sites(site_id),
    ts                timestamptz NOT NULL,
    power_kw          double precision NOT NULL,
    energy_kwh_total  double precision NOT NULL,  -- cumulative, monotonic per device
    voltage_v         double precision,            -- optional per concept note section 6
    status            reading_status NOT NULL DEFAULT 'ok',
    rssi              integer,
    provenance        provenance_type NOT NULL DEFAULT 'metered',
    received_at       timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (device_id, ts)  -- dedup: identical (device_id, ts) is the same reading, ON CONFLICT DO NOTHING on insert
);

SELECT create_hypertable('telemetry', 'ts');

CREATE INDEX idx_telemetry_site_ts ON telemetry (site_id, ts DESC);

-- Append-only, never updated in place except to mark processed/error —
-- this is the audit trail: every message received, before validation,
-- so nothing is ever silently dropped. See AGENTS.md item 4.
CREATE TABLE ingestion_audit_log (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    device_id    text NOT NULL,
    raw_payload  jsonb NOT NULL,
    received_at  timestamptz NOT NULL DEFAULT now(),
    processed    boolean NOT NULL DEFAULT false,
    error        text
);

CREATE INDEX idx_audit_device_received ON ingestion_audit_log (device_id, received_at DESC);

-- +goose Down
DROP TABLE IF EXISTS ingestion_audit_log;
DROP TABLE IF EXISTS telemetry;
DROP TYPE IF EXISTS reading_status;
DROP TYPE IF EXISTS provenance_type;
DROP TABLE IF EXISTS devices;
DROP TABLE IF EXISTS sites;
