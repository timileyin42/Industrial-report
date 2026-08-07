# Load test results

This is a diagnostic log, not a pass/fail gate (concept note §13: "load-test
with simulated devices before large roll-outs"). Each run below was
executed against the local docker-compose stack (single Mosquitto node,
single ingestor, single API instance, on a development laptop) using
`cmd/loadtest`. See that tool's own doc comment and `scripts/loadtest-provision.sh`
for how devices are provisioned before a run.

**What this does and doesn't prove**: this found the first bottleneck on
*this* hardware/topology. It does not prove a real production deployment
(different network, different infra sizing, possibly multiple API/ingestor
instances) behaves the same way — see README's "What's deliberately not
here yet" section.

## Run 1 — 92 devices, 20s, 3s interval

| Metric | Value |
|---|---|
| Devices connected | 92 / 92 |
| Connect failures | 0 |
| Publishes succeeded | 552 |
| Publishes failed | 0 |
| Effective rate | 27.6 publishes/sec |
| DB rows landed | 552 (verified via `SELECT count(*) FROM telemetry`) |

No issues. Baseline sanity check that the tool and pipeline work correctly
end to end.

## Run 2 — 400 devices, 45s, 5s interval, + concurrent API read load (20 rps)

| Metric | Value |
|---|---|
| Devices connected | 400 / 400 |
| Connect failures | 0 |
| Publishes succeeded | 3,209 |
| Publishes failed | 0 |
| Effective rate | 71.3 publishes/sec |
| Concurrent `GET /fleet/summary` calls | 983 ok, 0 failed |
| DB rows landed | 3,761 across 492 distinct devices (cumulative with run 1's devices) |
| Ingestion audit log errors | 0 |

**No degradation observed** at this scale on this hardware — every MQTT
connection succeeded, every publish succeeded, every concurrent API read
succeeded, and every row that was published landed correctly in
`telemetry` with zero validation/audit errors. Mosquitto, the ingestor,
pgx's connection pool, and the API's read path all held up cleanly at 400
concurrent device connections + read load.

## The actual first bottleneck found: device *registration*, not ingestion

Provisioning devices for these runs (via the real `POST /v1/devices` API,
not a bypass) hit `internal/httpapi/router.go`'s registration rate limiter
(`registerLimiter`, 2 requests/sec) almost immediately — the very first
attempt at bulk-provisioning 100 devices at full loop speed saw the
overwhelming majority rejected with `429 rate limit exceeded` after a
small initial burst (only 8 of the first 100 attempts succeeded).

This is **the rate limiter working exactly as designed** — CLAUDE.md
requires rate limiting on device registration specifically to prevent
brute-force/abuse, and it did its job. It just means:

- **Bulk-provisioning many devices through the real admin API is
  deliberately slow** (~2/sec) — fine for real field deployments, where
  devices are registered as they're physically installed, not in a batch
  script. `scripts/loadtest-provision.sh` paces itself under this limit
  (`REGISTER_DELAY=0.6s` between calls) to work *with* the limiter rather
  than around it — provisioning 500 devices this way took several
  minutes, which is the honest cost of testing "thousands of devices"
  through the real registration path rather than a shortcut that would
  give a false picture of the real system.
- **This is not an ingestion/MQTT bottleneck.** Once devices were
  provisioned, the actual telemetry pipeline (the thing "thousands of
  devices" is really asking about) showed no strain at all at 400
  concurrent connections.

## Where a real bottleneck would likely show up first at larger scale

Not measured directly in these runs, but worth stating plainly rather than
implying "thousands of devices, unconditionally proven fine":

- **Mosquitto single-node connection ceiling** — AGENTS.md already flags
  moving to EMQX as the answer "when Mosquitto's single-node throughput
  becomes the bottleneck (Phase 4 territory) — don't pre-optimize." These
  runs didn't get close to that ceiling; a much larger run (several
  thousand sustained connections) would be needed to find it.
- **pgx pool exhaustion / write contention on `telemetry`** — not observed
  at 400 devices' write volume; would need a much higher publish rate or
  device count to test for real.
- **The registration rate limiter itself**, if a real deployment ever
  needs to bulk-onboard hundreds of devices in a short window (e.g.
  migrating an existing fleet) rather than one at a time as they're
  installed — worth revisiting the limiter's burst allowance if that
  scenario becomes real, but not built speculatively here.

## Reproducing these runs

```bash
export OPERATOR_PASSWORD="change-me-now"
./scripts/loadtest-provision.sh 100    # paced under the rate limit; takes ~1-2 min

TOKEN=$(curl -s -X POST localhost:8080/v1/auth/login -H "Content-Type: application/json" \
  -d '{"email":"admin@zgnis.test","password":"change-me-now"}' | jq -r .token)

go run ./cmd/loadtest -devices-file loadtest-devices.csv -duration 30s -interval 5s \
  -api-token "$TOKEN" -api-rps 20
```
