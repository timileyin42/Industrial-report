-- sqlc schema snapshot: mirrors the combined "Up" state of
-- migrations/0001_init.sql through 0007_invites_and_password_resets.sql.
-- This file is
-- not run against any database — goose migrations remain the only source
-- of truth for actual schema changes. Keep this in sync whenever a
-- migration adds/changes a table sqlc needs to know about.

-- see migrations/0013_audit_log_tamper_evidence.sql — needed for digest()
-- used by the audit-log verification queries below.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

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
    timezone            text NOT NULL DEFAULT 'UTC', -- see migrations/0009_delocalize_defaults.sql
    country             text NOT NULL, -- see migrations/0010_site_country.sql — no default, must be set explicitly
    is_primary          boolean NOT NULL DEFAULT false, -- see migrations/0011_primary_site.sql
    created_at          timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_sites_one_primary ON sites (is_primary) WHERE is_primary = true;

CREATE TABLE devices (
    device_id              text PRIMARY KEY,
    site_id                text REFERENCES sites(site_id),
    secret_hash            text NOT NULL,
    revoked_at             timestamptz,
    last_seen_at           timestamptz,
    created_at             timestamptz NOT NULL DEFAULT now(),
    secret_last_rotated_at timestamptz NOT NULL DEFAULT now(),
    install_notes          text,
    last_contact_at        timestamptz,
    inverter_brand         text,
    inverter_model         text
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
    pv_power_kw       double precision,
    battery_soc_pct   smallint,
    battery_voltage_v double precision,
    pv_voltage_v      double precision,
    output_voltage_v  double precision,
    load_power_kw     double precision,
    grid_power_kw     double precision,
    PRIMARY KEY (device_id, ts)
);

CREATE TABLE cloud_import_tokens (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    device_id    text NOT NULL REFERENCES devices(device_id),
    token_hash   text NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz,
    revoked_at   timestamptz
);

CREATE TABLE ingestion_audit_log (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    device_id    text NOT NULL,
    raw_payload  jsonb NOT NULL,
    received_at  timestamptz NOT NULL DEFAULT now(),
    processed    boolean NOT NULL DEFAULT false,
    error        text,
    -- prev_hash/entry_hash: see migrations/0013_audit_log_tamper_evidence.sql
    prev_hash    text,
    entry_hash   text
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
    created_at    timestamptz NOT NULL DEFAULT now(),
    -- prev_hash/entry_hash: see migrations/0013_audit_log_tamper_evidence.sql
    prev_hash     text,
    entry_hash    text
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

-- mirrors migrations/0007_invites_and_password_resets.sql
CREATE TABLE invites (
    id                 bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id            bigint NOT NULL REFERENCES users(id),
    token_hash         text NOT NULL,
    invited_by_user_id bigint REFERENCES users(id),
    expires_at         timestamptz NOT NULL,
    accepted_at        timestamptz,
    created_at         timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE password_reset_tokens (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    bigint NOT NULL REFERENCES users(id),
    token_hash text NOT NULL,
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- mirrors migrations/0008_export_jobs.sql
CREATE TYPE export_job_status AS ENUM ('pending', 'running', 'completed', 'failed');
-- site_summary_pdf/fleet_summary_pdf added by migrations/0012_export_job_pdf.sql
CREATE TYPE export_job_type AS ENUM ('site_telemetry_csv', 'site_summary_csv', 'fleet_summary_csv', 'site_summary_pdf', 'fleet_summary_pdf');

CREATE TABLE export_jobs (
    id                    bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    requested_by_user_id  bigint NOT NULL REFERENCES users(id),
    job_type              export_job_type NOT NULL,
    site_id               text REFERENCES sites(site_id),
    status                export_job_status NOT NULL DEFAULT 'pending',
    result_key            text,
    error                 text,
    created_at            timestamptz NOT NULL DEFAULT now(),
    completed_at          timestamptz
);

-- mirrors migrations/0014_sandbox.sql
CREATE TABLE sandbox_runs (
    id             text PRIMARY KEY,
    system_size_kw numeric,
    row_count      integer NOT NULL DEFAULT 0,
    accepted_count integer NOT NULL DEFAULT 0,
    rejected_count integer NOT NULL DEFAULT 0,
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sandbox_readings (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    run_id           text NOT NULL REFERENCES sandbox_runs(id) ON DELETE CASCADE,
    row_number       integer NOT NULL,
    ts               timestamptz,
    power_kw         double precision,
    energy_kwh_total double precision,
    voltage_v        double precision,
    status           text,
    accepted         boolean NOT NULL,
    rejection_reason text,
    provenance       text,
    is_reset         boolean NOT NULL DEFAULT false,
    rssi             integer
);
