# Zgnis Test Telemetry Data

Static, reproducible sample data — no live MQTT connection needed. Safe
to hand to any AI tool, script, or teammate to run locally.

## Files

- **`telemetry-sample.jsonl`** — 576 readings (2 test devices × 1 full
  day × 5-minute intervals), one JSON object per line, in the exact
  payload shape the ingestor expects (`device_id`, `ts`, `power_kw`,
  `energy_kwh_total`, `voltage_v`, `status`).
- **`telemetry-sample.csv`** — same data, for spreadsheet/DB tools that
  prefer CSV.
- **`seed-test-site-device.sql`** — creates the two test sites/devices
  this data belongs to (`TEST-ZG-0001` on a 5kW system, `TEST-ZG-0002`
  on a 10kW system), so foreign keys resolve when you load telemetry.

## What's realistic about it (not random numbers)

- Zero output before ~6:30am / after ~6:30pm, smooth bell-curve peak
  at solar noon, capped at 85% of rated capacity
- Small instantaneous jitter + occasional cloud-cover dips
- Cumulative energy that actually integrates power over the interval
  (so daily-total and roll-up logic has something real to compute)
- Grid-tied voltage jitter (227-233V)
- Device 2 has a small chance of `fault`/`offline` readings mixed in,
  so validation/alerting logic has something to catch — device 1 is
  clean, for a "happy path" baseline
- Deterministic: regenerating with `generate.py` produces identical
  output (seeded), so test runs are reproducible

## How to use it locally

**Load straight into Postgres/Timescale** (skip MQTT entirely — good
for testing the API/dashboard without the ingestor in the loop):

```bash
psql "$DATABASE_URL" -f seed-test-site-device.sql

# then import the CSV directly:
psql "$DATABASE_URL" -c "\copy telemetry(device_id, ts, power_kw, energy_kwh_total, voltage_v, status) FROM 'telemetry-sample.csv' WITH (FORMAT csv, HEADER true)"
```

**Or replay it through MQTT** (tests the full ingestor path, dedup,
audit logging):

```bash
while IFS= read -r line; do
  device_id=$(echo "$line" | python3 -c "import json,sys; print(json.load(sys.stdin)['device_id'])")
  mosquitto_pub -h localhost -u "$device_id" -P devicesecret \
    -t "devices/$device_id/telemetry" -m "$line"
  sleep 0.05
done < telemetry-sample.jsonl
```

**Or hand `telemetry-sample.jsonl` directly to an AI coding tool** and
ask it to write a quick loader/import script in whatever language your
tooling prefers — the schema is self-describing and small enough to
paste inline if needed.

## Cleanup

Same rule as before: this is `TEST-` prefixed on purpose. Don't let it
land somewhere a real client could see it in a report or dashboard.
Delete via the same cleanup query pattern as `seed-test-site-device.sql`
before go-live.
