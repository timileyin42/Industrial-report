# Data retention

## Current policy

`migrations/0006_retention_policy.sql` sets a **2-year** retention window
on the `telemetry` hypertable via Timescale's `add_retention_policy()`.
Rows older than 2 years are automatically dropped by a background job.

**This is a placeholder, not a confirmed figure.** Concept note §11 ties
retention to verification-grade record-keeping (the generation record may
need to support independent verification/emissions reporting later) but
gives no specific number, and `AGENTS.md` explicitly defers this decision
rather than inventing one earlier. Before this policy runs against any
environment holding real site data, confirm the actual required window
with the client — it may need to be much longer (e.g. matching whatever
verification/audit period their emissions reporting requires) or, in
principle, shorter.

`telemetry_daily` (the Phase 3 continuous aggregate) deliberately has **no**
retention policy. It's a rollup of the same underlying data, and its whole
purpose is to let historical trend/KPI queries keep working after raw
`telemetry` rows have been pruned. Applying the same retention window to
both would defeat that.

## How to change the window later

Never change retention with a live `ALTER`/manual `psql` edit — that
breaks the goose-migrations-are-forward-only discipline this repo follows
everywhere else. Instead:

```sql
-- +goose Up
SELECT remove_retention_policy('telemetry');
SELECT add_retention_policy('telemetry', INTERVAL '<new window>');

-- +goose Down
SELECT remove_retention_policy('telemetry');
SELECT add_retention_policy('telemetry', INTERVAL '2 years');
```

Add this as a new migration file (e.g. `0007_update_retention_window.sql`)
— never edit `0006` in place, same as every other migration in this repo.

## Verifying the policy is active

```sql
SELECT * FROM timescaledb_information.jobs WHERE proc_name = 'policy_retention';
```

Should show one job scoped to the `telemetry` hypertable with the
configured interval.
