<p align="center">
  <img src="web/src/assets/brand/logo-full.svg" alt="Clean Energy Analytics" width="360">
</p>

<h3 align="center">Clean Energy Analytics</h3>
<p align="center">A verification-grade monitoring platform for distributed residential solar fleets.</p>

---

## What this is

Clean Energy Analytics ingests periodic telemetry from field-deployed solar
monitoring devices, stores it as time-series data, validates and cleans it,
and serves it through a role-based backend API and a real-time web
dashboard. Built for Zgnis Next Limited to monitor a growing fleet of
distributed residential solar installations — designed from the start to
scale to thousands of sites while keeping every reading clean, attributable,
and exportable enough to support future emissions reporting and independent
verification.

Devices report either directly (MQTT, for a datalogger that speaks the
platform's own protocol) or through a generic cloud-import webhook (for
inverters that only report into a manufacturer's own cloud app — a
production connector for the Chisage/PV Pro ecosystem ships in
`cmd/pvpro-sync` as a concrete example). Both paths run every reading
through the same validation, reset-detection, and provenance pipeline
before it ever reaches a chart.

## Architecture

```
Inverter --Modbus RTU--> Datalogger --MQTT/TLS--> Broker
  --> Ingestor (validate, dedup, audit log) --> TimescaleDB
  <-- Device Registry & Site Metadata
  --> Backend API --> Web Dashboard

Vendor cloud app (e.g. PV Pro) <--polls-- cmd/pvpro-sync --> Backend API
  (same validation/reset-detection/provenance pipeline as MQTT)
```

## Features

**Ingestion & devices**
- MQTT ingestion with QoS-1 dedup, out-of-order/backfilled arrival handling,
  and per-site plausibility validation
- Generic, vendor-agnostic cloud-import webhook for inverters that only
  report through a manufacturer's own app — a real PV Pro/E-linter CSP
  connector ships as a working example
- Device registry with per-device secrets, MQTT broker ACL sync
  (dynamic-security plugin, no manual `mosquitto_passwd`), rotation, and
  revocation enforced at both the broker and the application layer
- Cumulative energy-counter reset detection, including a single glitched
  sample vs. a genuine sustained reset

**Analytics & reporting**
- Energy, specific yield, peak output, capacity factor, and CO2-avoided
  emissions (versioned, client-confirmed grid emission factors)
- Site and fleet comparisons, benchmarking, trends, cohort rollups, and a
  trailing-baseline anomaly flag
- CSV/PDF export (sync and async, background job queue)
- Tamper-evident, hash-chained audit logs — ingestion (data-quality) and
  user-action (who changed what) tracked separately, each on its own
  retention policy

**Dashboard**
- React + TypeScript web app: fleet dashboard, per-site detail and
  analytics, live map view, device registry, alerts, and a live intraday
  power curve (not just daily totals)
- Role-based access end-to-end: an operator sees the whole fleet, a
  restricted account is scoped to exactly one site, enforced server-side
  on every request

**Hardening & scale**
- TLS on every hop (broker, API, DB volume), rate limiting, explicit CORS
- TimescaleDB continuous aggregates + retention policies (raw telemetry,
  ingestion audit log) instead of unbounded raw-table scans
- Backup/restore drill scripts, a load-test tool, and a full OpenAPI spec

## Tech stack

| | |
|---|---|
| **Backend** | Go, Echo v4, pgx/v5, sqlc, goose migrations |
| **Database** | PostgreSQL + TimescaleDB (continuous aggregates, retention policies) |
| **Frontend** | React 19, TypeScript, Vite, TanStack Query, Tailwind CSS, Zod |
| **Messaging** | Mosquitto MQTT (TLS, dynamic-security ACLs) |
| **Infra** | Docker Compose, Cloudflare R2 (export storage) |

## Project structure

```
cmd/            # api, ingestor, pvpro-sync (cloud-import connector), seed-operator, loadtest
internal/       # domain, registry (business logic), httpapi, db (sqlc), auth, mqttadmin
migrations/     # goose migrations — forward-only, the single source of schema truth
web/            # React dashboard (src/pages, src/components, src/api)
docs/           # concept note, OpenAPI spec, TLS/retention/decisions, verification guide
design/         # Stitch design export — token source of truth for the dashboard's UI
```

## Getting started (local dev)

```bash
# 1. Generate dev TLS certs, then start infra
./scripts/gen-dev-certs.sh
docker compose up -d

# 2. Run migrations
go install github.com/pressly/goose/v3/cmd/goose@latest
goose -dir migrations postgres "postgres://zgnis:zgnis_dev_only@localhost:5432/zgnis_solar?sslmode=disable" up

# 3. Create the first operator account (no bootstrap HTTP endpoint, by design)
go run ./cmd/seed-operator -email admin@zgnis.test -password 'change-me-now'

# 4. Run the backend
go mod tidy
export DATABASE_URL="postgres://zgnis:zgnis_dev_only@localhost:5432/zgnis_solar?sslmode=disable"
export JWT_SECRET="dev-only-change-me"
go run ./cmd/ingestor &
go run ./cmd/api &

# 5. Run the dashboard
cd web && npm install
echo "VITE_API_BASE_URL=http://localhost:8080" > .env
npm run dev
```

Full environment variable reference, MQTT credential setup, and a
step-by-step walkthrough proving every phase's behavior (validation
rejections, reset detection, role isolation, analytics, hardening) live in
**[docs/verification-guide.md](docs/verification-guide.md)**.

## Documentation

| Doc | Covers |
|---|---|
| [docs/concept-note.md](docs/concept-note.md) | The original client brief — scope, data model, security requirements |
| [docs/verification-guide.md](docs/verification-guide.md) | Full local setup + step-by-step behavior verification |
| [docs/openapi.yaml](docs/openapi.yaml) | Complete API spec, 1:1 with the registered routes |
| [docs/tls.md](docs/tls.md) | TLS setup and what it does/doesn't cover |
| [docs/retention.md](docs/retention.md) | Data retention policy and rationale |
| [docs/load-test-results.md](docs/load-test-results.md) | Load-test methodology and findings |
| [docs/decisions/](docs/decisions/) | Standing architectural decision records |
| [AGENTS.md](AGENTS.md) | Architecture and conventions for anyone (human or AI) working in this repo |
| [CLAUDE.md](CLAUDE.md) | Non-negotiable project rules — phasing, security, design-token fidelity |

## Delivery status

- **Phase 0–3 (done):** ingestion → registry/access-control → data-quality
  validation → analytics/reporting.
- **Phase 4 (in progress):** hardening + scale. TLS, retention, load
  testing, and OpenAPI spec are done; see
  [docs/verification-guide.md](docs/verification-guide.md#whats-deliberately-not-here-yet)
  for exactly what's open — durable export job queue, automated alert
  delivery, and multi-instance rate limiting, among others.

Sequencing is a deliberate project decision — see `CLAUDE.md` before
building ahead of the current phase.
