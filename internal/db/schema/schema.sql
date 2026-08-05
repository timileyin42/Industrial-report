-- sqlc schema snapshot: mirrors the combined "Up" state of
-- migrations/0001_init.sql through 0005_emission_factor.sql. This file is
-- not run against any database — goose migrations remain the only source
-- of truth for actual schema changes. Keep this in sync whenever a
-- migration adds/changes a table sqlc needs to know about.

CREATE TABLE sites (
    site_id             text PRIMARY KEY,
    cohort_id           text,
    address             text,
    name                text,
    gps_lat             double precision,
    gps_lng             double precision,
    inverter_make_model text,
    system_size_kw      numeric,
    install_date        date,
    timezone            text NOT NULL DEFAULT 'Africa/Lagos',
    created_at          timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE devices (
    device_id              text PRIMARY KEY,
    site_id                text REFERENCES sites(site_id),
    secret_hash            text NOT NULL,
    revoked_at             timestamptz,
    last_seen_at           timestamptz,
    created_at             timestamptz NOT NULL DEFAULT now(),
    secret_last_rotated_at timestamptz NOT NULL DEFAULT now(),
    install_notes          text,
    last_contact_at        timestamptz
);

CREATE TYPE provenance_type AS ENUM ('metered', 'estimated', 'backfilled');
CREATE TYPE reading_status AS ENUM ('ok', 'fault', 'offline');

CREATE TABLE telemetry (
    device_id         text NOT NULL REFERENCES devices(device_id),
    site_id           text NOT NULL REFERENCES sites(site_id),
    ts                timestamptz NOT NULL,
    power_kw          double precision NOT NULL,
    energy_kwh_total  double precision NOT NULL,
    voltage_v         double precision,
    status            reading_status NOT NULL DEFAULT 'ok',
    rssi              integer,
    provenance        provenance_type NOT NULL DEFAULT 'metered',
    received_at       timestamptz NOT NULL DEFAULT now(),
    quality_flags     text[] NOT NULL DEFAULT '{}',
    PRIMARY KEY (device_id, ts)
);

CREATE TABLE ingestion_audit_log (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    device_id    text NOT NULL,
    raw_payload  jsonb NOT NULL,
    received_at  timestamptz NOT NULL DEFAULT now(),
    processed    boolean NOT NULL DEFAULT false,
    error        text
);

CREATE TYPE user_role AS ENUM ('operator', 'restricted');

CREATE TABLE users (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email         text NOT NULL UNIQUE,
    password_hash text NOT NULL,
    role          user_role NOT NULL,
    site_id       text REFERENCES sites(site_id),
    created_at    timestamptz NOT NULL DEFAULT now(),
    disabled_at   timestamptz
);

CREATE TABLE user_action_audit_log (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor_user_id bigint REFERENCES users(id),
    action        text NOT NULL,
    target_type   text,
    target_id     text,
    metadata      jsonb,
    created_at    timestamptz NOT NULL DEFAULT now()
);

-- sqlc-only stand-in for the real migration's continuous aggregate
-- (`CREATE MATERIALIZED VIEW ... WITH (timescaledb.continuous)`), which
-- sqlc's plain PostgreSQL parser doesn't understand. Same output columns —
-- sqlc only needs the column set/types to generate query code against it.
CREATE TABLE telemetry_daily (
    device_id         text NOT NULL,
    site_id           text NOT NULL,
    day               timestamptz NOT NULL,
    peak_power_kw     double precision,
    energy_start_kwh  double precision,
    energy_end_kwh    double precision,
    reading_count     bigint NOT NULL,
    backfilled_count  bigint NOT NULL,
    has_reset         boolean NOT NULL
);

CREATE TABLE grid_emission_factor (
    id                 bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    kg_co2_per_kwh     numeric NOT NULL,
    country            text NOT NULL DEFAULT 'NG',
    source             text NOT NULL,
    effective_from     timestamptz NOT NULL,
    created_at         timestamptz NOT NULL DEFAULT now(),
    created_by_user_id bigint REFERENCES users(id)
);
