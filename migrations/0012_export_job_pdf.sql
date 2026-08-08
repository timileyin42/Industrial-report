-- +goose Up

-- Adds PDF as a report format alongside the existing CSV job types.
-- Only the two period-summary reports get a PDF variant — raw telemetry
-- (site_telemetry_csv) stays CSV-only, since a row-per-reading dump is a
-- CSV shape, not a report shape; concept-note.md's "PDF or CSV" language
-- is about human-readable summary reports, not raw data exports.
ALTER TYPE export_job_type ADD VALUE 'site_summary_pdf';
ALTER TYPE export_job_type ADD VALUE 'fleet_summary_pdf';

-- +goose Down
-- Postgres has no DROP VALUE for enums; rebuilding the type to remove a
-- value would require rewriting every dependent column, which isn't
-- worth it for a down-migration on a dev-only rollback path. Any job
-- rows already using the new values would need manual cleanup first.
SELECT 1;
