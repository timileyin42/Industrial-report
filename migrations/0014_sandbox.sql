-- +goose Up

-- Public, no-login "upload your own data and see it validated like a
-- real device sent it" sandbox — deliberately its own pair of tables,
-- never touching sites/devices/telemetry at all. That's not laziness:
-- it's the simplest way to guarantee this feature can never affect real
-- fleet dashboards/KPIs/RBAC, rather than threading an is_sandbox flag
-- through the dozen+ existing fleet-wide queries and risking missing one.
--
-- id is a long random token (registry layer generates it), not a serial
-- — that's the whole security model for a no-login "shareable link":
-- unguessable, not sequential/enumerable.
CREATE TABLE sandbox_runs (
    id             text PRIMARY KEY,
    system_size_kw numeric,
    row_count      integer NOT NULL DEFAULT 0,
    accepted_count integer NOT NULL DEFAULT 0,
    rejected_count integer NOT NULL DEFAULT 0,
    created_at     timestamptz NOT NULL DEFAULT now()
);

-- One row per CSV row uploaded, accepted or not — mirrors
-- ingestion_audit_log's own "log it either way" philosophy, so a
-- rejected row is visible with its reason, not silently dropped.
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
    is_reset         boolean NOT NULL DEFAULT false
);
CREATE INDEX idx_sandbox_readings_run ON sandbox_readings (run_id, row_number);

-- +goose Down
DROP TABLE IF EXISTS sandbox_readings;
DROP TABLE IF EXISTS sandbox_runs;
