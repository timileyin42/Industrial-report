-- +goose Up

-- Placeholder window — MUST be reviewed and replaced with the
-- client-confirmed retention requirement before this runs against any
-- environment holding real site data. Same discipline as migration 0005's
-- emission factor: this is not a seeded guess to be treated as final.
-- Concept note §11 ties retention to verification-grade record-keeping
-- but gives no number; AGENTS.md explicitly defers this decision to
-- Phase 4 rather than inventing one earlier. 2 years is a reasonable
-- placeholder for annual reporting cycles pending that confirmation.
SELECT add_retention_policy('telemetry', INTERVAL '2 years');

-- Deliberately no retention policy on telemetry_daily: it's a rollup of
-- the same underlying data, and the whole point of keeping it is to
-- survive raw-telemetry pruning for historical trend/KPI queries. Pruning
-- it on the same schedule as the raw table would defeat that purpose.

-- +goose Down
SELECT remove_retention_policy('telemetry');
