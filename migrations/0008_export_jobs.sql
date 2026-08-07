-- +goose Up

-- Async counterpart to the synchronous CSV endpoints (GET .../export/*.csv,
-- unchanged) — same underlying data, but the request returns a job
-- immediately and the caller polls/downloads once it's done. Useful once
-- an export's date range/row count makes a synchronous request risk a
-- client timeout; the sync endpoints remain the simpler default for
-- everyday small exports.
CREATE TYPE export_job_status AS ENUM ('pending', 'running', 'completed', 'failed');
CREATE TYPE export_job_type AS ENUM ('site_telemetry_csv', 'site_summary_csv', 'fleet_summary_csv');

CREATE TABLE export_jobs (
    id                    bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    requested_by_user_id  bigint NOT NULL REFERENCES users(id),
    job_type              export_job_type NOT NULL,
    site_id               text REFERENCES sites(site_id), -- null for fleet-wide jobs
    status                export_job_status NOT NULL DEFAULT 'pending',
    result_key            text,  -- R2 object key once completed
    error                 text,
    created_at            timestamptz NOT NULL DEFAULT now(),
    completed_at          timestamptz
);

CREATE INDEX idx_export_jobs_requester ON export_jobs (requested_by_user_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS export_jobs;
DROP TYPE IF EXISTS export_job_type;
DROP TYPE IF EXISTS export_job_status;
