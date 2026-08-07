# Zgnis Solar Monitoring — Phase 0 + Phase 1 + Phase 2 + Phase 3 + Phase 4

Phase 0 goal: one simulated device → MQTT → TimescaleDB → one API endpoint
returning a chartable series. Proves the whole pipe before any real hardware
is involved.

Phase 1 goal: turn that into a real, access-controlled platform — device
registry, device secret issuance, site mapping, per-site + fleet views, and
role-based access control (operator vs. restricted-to-one-site).

Phase 2 goal: validation rules (per-site plausibility bounds, energy-counter
reset detection, a coarse day/night sanity check), true-outage vs.
buffered-catch-up gap detection, explicit back-fill provenance tagging, and
a fleet-wide data-quality/health dashboard.

Phase 3 goal: analytics/KPIs (energy, specific yield, peak output, capacity
factor), CO2-avoided emissions (versioned, client-confirmed factor), site
and fleet comparisons/benchmarking, fleet-level trends and cohort rollups,
a naive trailing-baseline anomaly flag, CSV export, and the admin
audit-log browsing endpoint. Built on a new Timescale continuous
aggregate (`telemetry_daily`) rather than raw-scan queries.

Phase 4 goal: harden + scale — TLS wiring (dev-cert verified locally),
backup/restore drill scripts, a retention policy, a load-test tool,
a full OpenAPI spec, and documented rate-limiting/CORS decisions. See
`docs/tls.md`, `docs/retention.md`, `docs/openapi.yaml`,
`docs/load-test-results.md`, and `docs/decisions/`.

See `AGENTS.md` for architecture/conventions and `CLAUDE.md` for the
non-negotiable rules (phasing, security, design-token fidelity).

**Note:** every `curl -d '{...}'` example below needs
`-H "Content-Type: application/json"` — without it, Echo can't bind the
JSON body and requests will silently fail as if the fields were empty.

## 1. Generate dev TLS certs, then start infra

Since Phase 4, Mosquitto's TLS listener (8883) is active by default and
needs a certificate to start — generate the local dev-only cert **before**
bringing the stack up, or Mosquitto will fail to start:

```bash
./scripts/gen-dev-certs.sh
docker compose up -d
```

See `docs/tls.md` for what this does and doesn't cover.

## 2. Create MQTT credentials

```bash
# First entry needs -c to create the file
docker exec zgnis-mosquitto mosquitto_passwd -c -b /mosquitto/config/passwd ingestor-service supersecret
docker exec zgnis-mosquitto mosquitto_passwd -b /mosquitto/config/passwd ZG-0001 devicesecret
docker restart zgnis-mosquitto
```

## 3. Run migrations

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
goose -dir migrations postgres "postgres://zgnis:zgnis_dev_only@localhost:5432/zgnis_solar?sslmode=disable" up
```

This applies `0001_init.sql` (Phase 0 schema), `0002_registry_and_auth.sql`
(Phase 1: `users`, `user_action_audit_log`, device secret rotation
bookkeeping), `0003_data_quality.sql` (Phase 2: `devices.last_contact_at`,
`telemetry.quality_flags`), `0004_analytics_rollups.sql` (Phase 3: the
`telemetry_daily` continuous aggregate), `0005_emission_factor.sql`
(Phase 3: the versioned `grid_emission_factor` table), and
`0006_retention_policy.sql` (Phase 4: a 2-year retention policy on
`telemetry` — see `docs/retention.md`, this is a placeholder pending
client confirmation).

**Continuous aggregates need one extra step goose can't do for you**: the
view is created `WITH NO DATA`, so existing telemetry won't appear in it
until either its hourly refresh policy runs or you force it manually:

```bash
docker exec zgnis-timescaledb psql -U zgnis -d zgnis_solar -c \
  "CALL refresh_continuous_aggregate('telemetry_daily', NULL, NULL);"
```

## 4. Seed a site + device (Phase 0 style, still useful for local testing)

```bash
docker exec zgnis-timescaledb psql -U zgnis -d zgnis_solar -c "
INSERT INTO sites (site_id, name, address, inverter_make_model, system_size_kw, timezone)
VALUES ('SITE-0001', 'Test Site', 'Test Site, Lagos', 'Growatt SPF 5000ES', 5.0, 'Africa/Lagos');
INSERT INTO devices (device_id, site_id, secret_hash)
VALUES ('ZG-0001', 'SITE-0001', 'placeholder-not-checked-in-phase0');
"
```

From Phase 1 onward, prefer registering sites/devices through the API (step
7 below) — this raw-SQL seed is kept only because the MQTT credential for
`ZG-0001` above was created by hand in step 2, for local testing without
going through the registration flow.

## 5. Fetch Go deps and run the services

```bash
go mod tidy

export DATABASE_URL="postgres://zgnis:zgnis_dev_only@localhost:5432/zgnis_solar?sslmode=disable"
export MQTT_BROKER_URL="tcp://localhost:1883"
export MQTT_USERNAME="ingestor-service"
export MQTT_PASSWORD="supersecret"
export JWT_SECRET="dev-only-change-me"   # required by cmd/api from Phase 1 onward

# Phase 2 — all optional, shown here at their defaults:
export ONLINE_THRESHOLD_MINUTES=10
export EXPECTED_READING_INTERVAL_MINUTES=5
export COVERAGE_WINDOW_HOURS=24

# Invites/password-reset emails — all optional. Unset RESEND_API_KEY or
# EMAIL_FROM_ADDRESS falls back to a logging no-op sender (see
# internal/email) so invite/reset endpoints still work in local dev
# without a Resend account; nothing is actually delivered until both are
# set and the sending domain is verified with Resend.
export RESEND_API_KEY=""
export EMAIL_FROM_ADDRESS=""   # e.g. "no-reply@cleanenergyanalytics.co.uk"
export APP_BASE_URL="http://localhost:5173"   # builds invite/reset links; defaults to the Vite dev server

# Async export jobs (Slice 3) — all optional. Unset any one of these and
# job creation still succeeds but every job fails fast with a clear
# "export storage isn't configured yet" error instead of hanging;
# the sync CSV endpoints (GET .../export/*.csv) need none of this.
export R2_ACCOUNT_ID=""
export R2_ACCESS_KEY_ID=""
export R2_SECRET_ACCESS_KEY=""
export R2_BUCKET=""

# Grid emission factor default country — optional, defaults to "NG" to
# match this platform's first deployment's already-seeded data. Every
# emissions endpoint also accepts an explicit ?country= override.
export GRID_COUNTRY="NG"

go run ./cmd/ingestor &
go run ./cmd/api &
```

**Frontend** (`web/.env`, see `web/.env.example`-equivalent below) also
takes one optional key for the site-location map on Site Detail:

```bash
VITE_API_BASE_URL=http://localhost:8080
VITE_GOOGLE_MAPS_API_KEY=   # optional — unset shows a graceful placeholder instead of a broken map embed; needs the Maps Embed API enabled on that key
```

## 6. Create the first operator account

There's no bootstrap HTTP endpoint for this on purpose — an unauthenticated
"create the first admin" route is itself a security hole. Run the one-off
seed command instead:

```bash
go run ./cmd/seed-operator -email admin@zgnis.test -password 'change-me-now'
```

## 7. Log in and drive the registry API

```bash
TOKEN=$(curl -s -X POST localhost:8080/v1/auth/login -H "Content-Type: application/json" \
  -d '{"email":"admin@zgnis.test","password":"change-me-now"}' | jq -r .token)

# Create a site
curl -s -X POST localhost:8080/v1/sites -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"site_id":"SITE-0002","name":"Test Site B","timezone":"Africa/Lagos","system_size_kw":5.0}'

# Register a device — capture the plaintext secret, shown exactly once
SECRET=$(curl -s -X POST localhost:8080/v1/devices -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"device_id":"ZG-0002","site_id":"SITE-0002"}' | jq -r .secret)

# Manual step (deliberately not automated — see AGENTS.md item 1): sync the
# secret into the Mosquitto password file so the device can actually connect.
docker exec zgnis-mosquitto mosquitto_passwd -b /mosquitto/config/passwd ZG-0002 "$SECRET"
docker restart zgnis-mosquitto

curl -s localhost:8080/v1/fleet/summary -H "Authorization: Bearer $TOKEN"
curl -s localhost:8080/v1/sites -H "Authorization: Bearer $TOKEN"
curl -s "localhost:8080/v1/sites/SITE-0002/telemetry" -H "Authorization: Bearer $TOKEN"
```

## 8. Simulate a device publishing a reading

No real datalogger yet, so simulate one with `mosquitto_pub`:

```bash
docker exec zgnis-mosquitto mosquitto_pub \
  -h localhost -u ZG-0001 -P devicesecret \
  -t devices/ZG-0001/telemetry \
  -m '{"device_id":"ZG-0001","ts":"2026-07-30T13:05:00Z","power_kw":3.42,"energy_kwh_total":1284.6,"voltage_v":231.4,"status":"ok"}'
```

## 9. Check it landed — now auth-gated

Every read now requires `Authorization: Bearer <token>` (Phase 0's endpoints
were open to anyone; that gap is closed in Phase 1):

```bash
curl -s http://localhost:8080/v1/sites/SITE-0001/telemetry -H "Authorization: Bearer $TOKEN"
curl -s http://localhost:8080/v1/devices/ZG-0001/status -H "Authorization: Bearer $TOKEN"
```

## 10. Prove role scoping

```bash
curl -s -X POST localhost:8080/v1/users -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"email":"restricted@zgnis.test","password":"change-me-too","role":"restricted","site_id":"SITE-0001"}'

RTOKEN=$(curl -s -X POST localhost:8080/v1/auth/login -H "Content-Type: application/json" \
  -d '{"email":"restricted@zgnis.test","password":"change-me-too"}' | jq -r .token)

curl -s localhost:8080/v1/sites/SITE-0001/telemetry -H "Authorization: Bearer $RTOKEN"                              # expect 200
curl -s -o /dev/null -w "%{http_code}\n" localhost:8080/v1/sites/SITE-0002/telemetry -H "Authorization: Bearer $RTOKEN"  # expect 403
```

If the restricted user gets their own site's data but a 403 (not a 404, not
an empty 200) on the other site, Phase 1's access control is proven.

## 11. Phase 2 — validation rules, gap detection, fleet health

**Per-site power ceiling rejects an oversized reading** (a 2kW site's
ceiling is `2 * 1.5 = 3kW`):

```bash
curl -s -X POST localhost:8080/v1/sites -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"site_id":"SITE-SMALL","name":"Small Site","timezone":"Africa/Lagos","system_size_kw":2.0}'
SECRET=$(curl -s -X POST localhost:8080/v1/devices -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"device_id":"ZG-SMALL","site_id":"SITE-SMALL"}' | jq -r .secret)
docker exec zgnis-mosquitto mosquitto_passwd -b /mosquitto/config/passwd ZG-SMALL "$SECRET"
docker restart zgnis-mosquitto

docker exec zgnis-mosquitto mosquitto_pub -h localhost -u ZG-SMALL -P "$SECRET" \
  -t devices/ZG-SMALL/telemetry \
  -m '{"device_id":"ZG-SMALL","ts":"2026-08-05T09:00:00Z","power_kw":10.0,"energy_kwh_total":5.0,"status":"ok"}'

docker exec zgnis-timescaledb psql -U zgnis -d zgnis_solar -c \
  "SELECT error FROM ingestion_audit_log WHERE device_id='ZG-SMALL' ORDER BY received_at DESC LIMIT 1;"
# expect: error mentions "exceeds plausible ceiling"; zero rows in telemetry for ZG-SMALL
```

**Energy-counter reset is flagged, not rejected** (publish a normal reading,
then a lower one simulating a reboot):

```bash
docker exec zgnis-mosquitto mosquitto_pub -h localhost -u ZG-SMALL -P "$SECRET" \
  -t devices/ZG-SMALL/telemetry -m '{"device_id":"ZG-SMALL","ts":"2026-08-05T09:05:00Z","power_kw":1.0,"energy_kwh_total":100.0,"status":"ok"}'
docker exec zgnis-mosquitto mosquitto_pub -h localhost -u ZG-SMALL -P "$SECRET" \
  -t devices/ZG-SMALL/telemetry -m '{"device_id":"ZG-SMALL","ts":"2026-08-05T09:10:00Z","power_kw":1.0,"energy_kwh_total":40.0,"status":"ok"}'

docker exec zgnis-timescaledb psql -U zgnis -d zgnis_solar -c \
  "SELECT ts, energy_kwh_total, quality_flags FROM telemetry WHERE device_id='ZG-SMALL' ORDER BY ts;"
# expect: both rows present; the second has quality_flags = {energy_reset}
```

**Backfilled provenance** (a message whose `ts` is well in the past gets
tagged `backfilled`, not `metered`):

```bash
PAST=$(date -u -v-30M +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d '-30 minutes' +%Y-%m-%dT%H:%M:%SZ)
docker exec zgnis-mosquitto mosquitto_pub -h localhost -u ZG-SMALL -P "$SECRET" \
  -t devices/ZG-SMALL/telemetry -m "{\"device_id\":\"ZG-SMALL\",\"ts\":\"$PAST\",\"power_kw\":1.2,\"energy_kwh_total\":101.0,\"status\":\"ok\"}"

docker exec zgnis-timescaledb psql -U zgnis -d zgnis_solar -c \
  "SELECT provenance FROM telemetry WHERE device_id='ZG-SMALL' ORDER BY ts DESC LIMIT 1;"
# expect: backfilled
```

**True outage vs. data gap** (the two-signal model):

```bash
curl -s localhost:8080/v1/devices/ZG-SMALL/status -H "Authorization: Bearer $TOKEN"

# simulate a true outage — device hasn't contacted the broker at all recently
docker exec zgnis-timescaledb psql -U zgnis -d zgnis_solar -c \
  "UPDATE devices SET last_contact_at = now() - interval '1 hour' WHERE device_id='ZG-SMALL';"
curl -s localhost:8080/v1/devices/ZG-SMALL/status -H "Authorization: Bearer $TOKEN"
# expect: online:false

# simulate "reconnected, still replaying an old backlog"
docker exec zgnis-timescaledb psql -U zgnis -d zgnis_solar -c \
  "UPDATE devices SET last_contact_at = now(), last_seen_at = now() - interval '1 hour' WHERE device_id='ZG-SMALL';"
curl -s localhost:8080/v1/devices/ZG-SMALL/status -H "Authorization: Bearer $TOKEN"
# expect: online:true, data_gap:true
```

**Fleet health dashboard** (operator-only, paginated):

```bash
curl -s localhost:8080/v1/fleet/health -H "Authorization: Bearer $TOKEN" | jq
curl -s -o /dev/null -w "%{http_code}\n" localhost:8080/v1/fleet/health -H "Authorization: Bearer $RTOKEN"
# expect: 403 for the restricted user — this is the fleet-wide operator view
```

If the oversized reading is rejected, the reset is flagged (not rejected),
the late message is tagged `backfilled`, the two outage signals behave as
described, and the restricted user is blocked from `/v1/fleet/health`,
Phase 2 is proven.

## 12. Phase 3 — analytics, emissions, comparisons, exports, audit log

**Energy, specific yield, peak, capacity factor** for a site (bare dates
like `2026-08-05` work for `from`/`to` — a bare `to` is treated as end of
day):

```bash
curl -s "localhost:8080/v1/sites/SITE-SMALL/analytics/energy?period=daily&from=2026-08-01&to=2026-08-05" -H "Authorization: Bearer $TOKEN" | jq
curl -s "localhost:8080/v1/sites/SITE-SMALL/analytics/specific-yield?period=daily&from=2026-08-01&to=2026-08-05" -H "Authorization: Bearer $TOKEN" | jq
curl -s "localhost:8080/v1/sites/SITE-SMALL/analytics/peak?from=2026-08-01&to=2026-08-05" -H "Authorization: Bearer $TOKEN" | jq
curl -s "localhost:8080/v1/sites/SITE-SMALL/analytics/capacity-factor?period=daily&from=2026-08-01&to=2026-08-05" -H "Authorization: Bearer $TOKEN" | jq
```

**Emissions — 409 until a factor is configured, then succeeds** (the
concept note says this figure must be client-confirmed, so nothing is
seeded):

```bash
curl -s -o /dev/null -w "%{http_code}\n" "localhost:8080/v1/sites/SITE-SMALL/analytics/emissions?period=daily" -H "Authorization: Bearer $TOKEN"   # expect 409

curl -s -X POST localhost:8080/v1/config/emission-factor -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"kg_co2_per_kwh": 0.4, "country": "NG", "source": "<client-confirmed citation>", "effective_from": "2026-01-01T00:00:00Z"}'

curl -s "localhost:8080/v1/sites/SITE-SMALL/analytics/emissions?period=daily&from=2026-08-01&to=2026-08-05" -H "Authorization: Bearer $TOKEN" | jq
```

**Comparisons, benchmarking, trends, cohorts** (all operator-only except
`compare/history`, which is site-scoped):

```bash
curl -s "localhost:8080/v1/sites/SITE-SMALL/analytics/compare/history?period=daily&as_of=2026-08-05" -H "Authorization: Bearer $TOKEN" | jq
curl -s "localhost:8080/v1/fleet/analytics/compare/fleet?site_id=SITE-SMALL&period=daily&as_of=2026-08-05" -H "Authorization: Bearer $TOKEN" | jq
curl -s "localhost:8080/v1/fleet/analytics/benchmark?segment_by=system_size_band&period=daily&as_of=2026-08-05" -H "Authorization: Bearer $TOKEN" | jq
curl -s "localhost:8080/v1/fleet/analytics/trends?period=daily&from=2026-08-01&to=2026-08-05" -H "Authorization: Bearer $TOKEN" | jq
curl -s "localhost:8080/v1/fleet/analytics/cohorts/COHORT-A?period=daily&from=2026-08-01&to=2026-08-05" -H "Authorization: Bearer $TOKEN" | jq
```

**Anomaly flags** (trailing-baseline only — see the `definition` field in
the response before reading too much into a flag):

```bash
curl -s "localhost:8080/v1/sites/SITE-SMALL/analytics/anomalies?as_of=2026-08-05&window_days=7" -H "Authorization: Bearer $TOKEN" | jq
curl -s "localhost:8080/v1/fleet/analytics/anomalies?as_of=2026-08-05&window_days=7" -H "Authorization: Bearer $TOKEN" | jq
```

**CSV export**:

```bash
curl -s "localhost:8080/v1/sites/SITE-SMALL/export/telemetry.csv?from=2026-08-01&to=2026-08-05" -H "Authorization: Bearer $TOKEN"
curl -s "localhost:8080/v1/sites/SITE-SMALL/export/summary.csv?period=daily&from=2026-08-01&to=2026-08-05" -H "Authorization: Bearer $TOKEN"
curl -s "localhost:8080/v1/fleet/export/summary.csv?period=daily&from=2026-08-01&to=2026-08-05" -H "Authorization: Bearer $TOKEN"
```

**Admin audit-log browsing** (the Phase 1/2 deferred catch-up — operator-only):

```bash
curl -s "localhost:8080/v1/audit/actions?action=emission_factor.set" -H "Authorization: Bearer $TOKEN" | jq
curl -s -o /dev/null -w "%{http_code}\n" "localhost:8080/v1/audit/actions" -H "Authorization: Bearer $RTOKEN"   # expect 403
```

**Restricted-role isolation** (same pattern as every prior phase):

```bash
curl -s -o /dev/null -w "%{http_code}\n" "localhost:8080/v1/sites/SITE-0001/analytics/energy?period=daily" -H "Authorization: Bearer $RTOKEN"   # expect 200 (own site)
curl -s -o /dev/null -w "%{http_code}\n" "localhost:8080/v1/sites/SITE-SMALL/analytics/energy?period=daily" -H "Authorization: Bearer $RTOKEN"  # expect 403 (other site)
curl -s -o /dev/null -w "%{http_code}\n" "localhost:8080/v1/fleet/analytics/trends?period=daily" -H "Authorization: Bearer $RTOKEN"             # expect 403 (fleet-wide)
```

Phase 3 is proven when: energy/yield/peak/capacity-factor numbers match a
hand-computed check against raw `telemetry` (including a day with a counter
reset, correctly split at the reset boundary); emissions return `409` then
succeed after configuration; comparisons/benchmarking/trends/cohorts return
correctly-shaped data; CSV exports include every row in range; the audit
log is browsable and operator-only; and role/site isolation holds for
every new endpoint.

## 13. Phase 4 — harden + scale

**TLS**, already verified in `docs/tls.md`'s own walkthrough — regenerate
dev certs, confirm Mosquitto's 8883 listener handshakes, confirm the
ingestor connects over `ssl://`, confirm the API's optional TLS listener
works when `API_TLS_CERT_FILE`/`API_TLS_KEY_FILE` are set.

**Retention** — confirm the policy is registered:

```bash
docker exec zgnis-timescaledb psql -U zgnis -d zgnis_solar -c \
  "SELECT job_id, hypertable_name, config FROM timescaledb_information.jobs WHERE proc_name = 'policy_retention';"
```

**Backups + restore drill**:

```bash
./scripts/backup.sh
./scripts/restore-drill.sh
# expect: RESTORE DRILL PASSED, with non-zero row counts and telemetry recognized as a hypertable
```

**Load testing** — provision synthetic devices through the real API, then
run the load tool:

```bash
export OPERATOR_PASSWORD="change-me-now"
./scripts/loadtest-provision.sh 100        # takes a while — paced under the registration rate limit, see docs/load-test-results.md
go run ./cmd/loadtest -devices-file loadtest-devices.csv -duration 30s -interval 5s
```

See `docs/load-test-results.md` for recorded runs and the first bottleneck
found.

**Documented exports** — every `/v1` endpoint is specified in
`docs/openapi.yaml`; it validates as OpenAPI 3.0 and its path list matches
`internal/httpapi/router.go`'s registered routes 1:1.

**Rate limiting / CORS** — see `docs/decisions/rate-limiting.md` for why
the current single-instance limiter is accepted as-is, and why CORS has
nothing left to build until a real dashboard origin exists.

## What's deliberately not here yet

- Ingestor-side device secret verification — the ingestor still trusts the
  Mosquitto broker's own auth; `secret_hash` is used by the registry for
  issuance/rotation only (adding payload-level secret verification would be
  a protocol change, out of scope so far)
- Automated broker credential provisioning — syncing a newly issued device
  secret into Mosquitto's password file is still a manual `mosquitto_passwd`
  step (AGENTS.md item 1: not automated yet, by design)
- Automated alert emails (e.g. "device offline") — email delivery now
  exists (invites, password reset), but there's no scheduler/cron and no
  dedup logic to avoid re-notifying on every check; still queryable state
  only (anomalies, fleet health), same as before
- A durable, restart-surviving export job queue — `internal/registry/exports.go`'s
  async jobs run in-process (a job's params live in a goroutine closure,
  never persisted); a process restart mid-job is self-healed by marking
  it failed after 10 minutes, not resumed. Real queue infra (e.g. a
  separate worker process reading persisted job params) is a follow-up
  if export volume ever needs it — the sync CSV endpoints remain the
  simpler default either way
- Per-site country/grid tracking — `GRID_COUNTRY` (env var) and each
  emission-factor's own `country` field support multiple values, but
  there's still one global "current" factor per country, not a per-site
  assignment. A fleet genuinely spanning grids needs a `sites.country`
  column and a bigger change to `internal/registry/emissions.go`
- A real sunrise/sunset or solar-position day/night check — Phase 2 ships a
  coarse, dependency-free civil-time heuristic (`domain.IsCoarseNight`,
  20:00–05:00 local) instead; it's advisory-only (flags, never rejects) and
  will misjudge the ~30-60 minute window around actual dawn/dusk
- A backfilled reading landing *between* two already-stored readings only
  checks the reset condition forward, not retroactively against the row
  after it — a known, flagged limitation, not fixed yet
- The fleet-health coverage query (Phase 2) still scans raw `telemetry`
  over a bounded window — Phase 3's own `telemetry_daily` continuous
  aggregate doesn't retrofit it; that's a follow-up, not done here
- Persisted alerts with an ack/resolve lifecycle — there's no notification
  channel yet, so "alerts"/anomalies mean queryable state, not anything
  sent to anyone
- **Performance Ratio (PR)** — needs weather/irradiance data to know what a
  site *should* produce; no such data source exists in this stack. What
  ships instead is `capacity-factor` (energy vs. a weather-free theoretical
  max) — explicitly labeled as not PR in its own response
- **Weather/season-aware anomaly detection** ("materially below
  expectation for season") — same missing dependency as PR. What ships is
  a narrower "sudden drop vs. own trailing baseline" check, explicitly
  labeled as such (see the `definition` field on every anomaly response)
- **PDF report generation** — CSV (stdlib, zero new deps) covers export;
  PDF needs a new dependency, flagged as a go/no-go call, not added
- **"Region" segmentation** — `sites` has no dedicated region/state column,
  so `segment_by=region` uses `cohort_id` as the closest available
  grouping (documented in the API response itself)
- Weather/irradiance normalization, generation forecasting, configurable
  KPI alert thresholds — the concept note's own "Optional/later" bucket
- **A real CA-issued TLS certificate** — Phase 4 activates TLS everywhere
  in this stack (broker, DB, API) but only with a self-signed dev cert
  (`scripts/gen-dev-certs.sh`); a real cert for a real hostname is a
  deployment-time decision this repo can't make for itself. See
  `docs/tls.md`'s checklist.
- **Off-site/cloud backup storage and a recurring schedule** —
  `scripts/backup.sh`/`restore-drill.sh` prove backup+restore works
  locally; where backups actually live long-term and what triggers them
  on a cadence is a deployment decision, not faked with a placeholder
  credential.
- **The real retention window** — `0006_retention_policy.sql` sets a
  2-year placeholder; the actual client-confirmed figure isn't decided
  here (`docs/retention.md`).
- **Multi-instance rate limiting** — the in-memory limiter is correct for
  a single-instance deployment (what this project runs); a shared-store
  fix is deferred as a documented, permanent decision record
  (`docs/decisions/rate-limiting.md`), not built speculatively.
- **Proof this holds at real production scale** — `cmd/loadtest` found a
  real bottleneck locally (see `docs/load-test-results.md`), but a laptop
  running docker-compose isn't the target deployment's infra; this is a
  diagnostic tool, not a production capacity guarantee.
