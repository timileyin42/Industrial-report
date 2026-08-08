# Clean Energy Analytics — What We've Built (Demo & Walkthrough Guide)

This is a plain-English guide to what exists today, why it exists, and how
to talk through it live. It's written for the demo, not for developers —
see `README.md` for setup commands and `CLAUDE.md`/`AGENTS.md`/
`docs/concept-note.md` for the full technical spec and rules.

---

## 1. The big picture, in one paragraph

Solar sites out in the field have a small box (a "datalogger") that reads
the inverter and sends a message every few minutes saying "here's how much
power I'm making right now." Those messages travel over the internet to
our server, get checked for obvious nonsense, get stored, and then get
turned into the numbers an operator actually cares about — how much energy
a site made this month, how it compares to other sites, how much CO₂ it
avoided. **That whole pipeline exists and works today, and — unlike the
early phases — it's no longer API-only.** There's a real web dashboard an
operator or restricted user actually logs into, plus a public marketing
site, both built on top of the same APIs this guide's demo script exercises
directly.

## 2. The pipeline, plain-English

```
Solar inverter → little box on-site → sends a message over the internet
   → our "mailroom" (Mosquitto) sorts messages by device
   → our "intake clerk" (the ingestor) checks each message makes sense
   → stores it in the database
   → an overnight "summarizer" pre-computes daily totals per device
   → our API hands out raw data AND summarized KPIs to whoever's allowed to see them
   → the dashboard (or a direct API call) is how a person actually sees it
```

- **Mosquitto** is just message plumbing. It knows nothing about solar
  panels — it just makes sure device A's message can't be mistaken for
  device B's, and that only device A can send messages "as" device A.
- **The ingestor** is the one piece of code that actually understands what
  a solar reading means. It's the "quality control" step.
- **The database** (TimescaleDB, a version of Postgres built for
  time-stamped data) is where every reading lives permanently.
- **The summarizer** (a Timescale "continuous aggregate") is a background
  job that keeps a running daily total per device, so KPI questions like
  "how much energy did this site make this month" don't have to re-scan
  every single reading ever taken.
- **The API** is the only door into that data — nobody, not even our own
  dashboard, is allowed to read the database directly. Everything goes
  through the API so access rules are enforced in one place.
- **The dashboard** is a web app that calls that same API — it has no
  special back-door access, so anything demonstrated through the API in
  this guide is exactly what a real user sees on screen.

## 3. What's been built, phase by phase

### Phase 0 — "prove the pipe works at all"
One simulated device, one message, one chart-ready endpoint. ✅ Done.

### Phase 1 — "make it a real, access-controlled system"
- **Device registry**: create a site, register a device. Registering a
  device generates it a secret password, shown **exactly once** — lost it
  means rotating to a new one, never recovering the old one.
- **Login & roles** — **operator** (sees the whole fleet) vs. **restricted**
  (locked to exactly one site, enforced server-side on every request —
  guessing a URL to another site gets a flat 403, never data, never even
  "not found").
- **Admin audit trail** — every admin action is permanently logged with
  who did it and when, separate from the data-quality log.

### Phase 2 — "catch bad data and know when a device goes dark"
Validation rules, gap/offline detection, fleet health view — see section 5
for the "rejected vs. flagged" and "offline vs. data gap" distinctions.

### Phase 3 — "turn raw readings into the numbers people actually want"
Energy, specific yield, peak output, capacity factor, CO₂ avoided
(client-confirmed factor, never guessed), fleet comparisons/benchmarking,
cohorts, a trailing-baseline anomaly flag, CSV export, browsable admin
audit log. See section 4 for the specific thresholds behind these.

### Phase 4 — "harden and scale it"
- **TLS wiring** on every hop (device→broker, API→dashboard) — dev-cert
  verified locally; a real deployment swaps in a real certificate (see
  section 9).
- **Invites & password reset, by real email** — via Resend. There's no
  public sign-up (see section 6) — an operator invites a user, they get a
  real email with a time-limited link to set their own password.
- **Async CSV/report exports** — queued jobs, files land in Cloudflare R2,
  downloaded via a time-limited link rather than served straight through
  the API process.
- **Retention policy** — raw telemetry auto-expires after a configurable
  window (currently 2 years, a placeholder pending client confirmation);
  the daily summary rollup is kept indefinitely since it's what most
  historical queries actually need.
- **Rate limiting** on public-facing endpoints (login, invite, reset).
- **Load testing** — a dedicated `cmd/loadtest` tool, results in
  `docs/load-test-results.md`.
- **Full OpenAPI spec** — `docs/openapi.yaml`.

### Frontend build — the actual dashboard and marketing site
A full React/TypeScript app: login, an app shell (sidebar + top nav), Fleet
Dashboard, Sites, Devices, Fleet/Site Analytics, Fleet Health, Audit Log,
Ingestion Log, Invite User, Emission Factor settings, plus a 4-page public
marketing site (Home, Features, Solutions, Company) and the new Terms/
Privacy/Security pages (section 10). Redesigned this session into a light/
glass visual system (see section 8) — the current visual language, not a
first draft.

### Per-site country / multi-grid emissions
CO₂-offset reporting used to resolve one global "which country's grid"
setting for the whole platform. Sites now carry their own `country`,
resolved per-site (and per-country for a fleet spanning more than one grid)
instead — see section 6a.

## 4. The specific limits — what they are and why (the "10kW" question, and friends)

| What | Value today | Plain-English reason |
|---|---|---|
| **Power ceiling** | site's rated size × **1.5** | If a site is rated for 2kW, we reject any reading claiming more than 3kW as physically implausible. 1.5x still firmly catches "something is very wrong here" while not rejecting a genuine brief spike. |
| **"Online" cutoff** | **10 minutes** since the device last contacted us at all | If we haven't heard from a device in 10+ minutes, we consider it offline. |
| **Expected reporting interval** | **5 minutes** | Devices are supposed to report every 1–5 minutes; 5 min is the benchmark for "is this device behind." |
| **Backfill threshold** | **15 minutes** between when a reading says it happened and when it actually arrived | Buffered catch-up data gets tagged `backfilled` instead of `metered`. |
| **Coverage window** | **24 hours** | Fleet health's "% of expected readings actually received" looks at the last 24 hours. |
| **Anomaly drop threshold** | **50%** below a site's own trailing 7-day average | Deliberately a big, obvious drop, not a subtle one. |

**All of these are configuration, not hardcoded.**

**Two concepts deserve their own explanation, because they're easy to get
wrong live:**

- **Capacity factor is NOT Performance Ratio (PR).** PR compares actual
  output against what it *should have* produced given the real weather
  that day. We don't have irradiance data, so we compute **capacity
  factor** instead — actual output vs. a flat theoretical maximum, zero
  weather adjustment. Every response says so explicitly. (The dashboard
  does show real *current* weather via Open-Meteo now — see section 8 —
  but that's a live conditions widget, not an input to this calculation.)
- **The anomaly check is a blunt instrument, on purpose.** Without
  irradiance data, we can't tell "cloudy day" apart from "something's
  actually broken," so it only flags a genuinely severe drop (50%+).

## 5. The two different "something's wrong" signals — don't mix these up in the demo

- **A rejected reading** — physically impossible. **Never stored**, but
  logged permanently in a "raw intake" audit trail so nothing is silently
  dropped without a trace.
- **A flagged reading** — plausible but noteworthy (e.g. an energy counter
  reset from an inverter reboot). **Is stored**, exactly as received, just
  tagged. Energy/yield/CO₂ figures correctly work around a flagged reset
  day instead of producing a nonsense negative number.
- **"Offline" vs. "data gap"**:
  - **Offline** = hasn't contacted us at all — a true outage.
  - **Data gap** = *is* talking to us, but its latest *accepted* reading is
    stale — e.g. just reconnected, still working through a backlog.

## 6. How anyone actually gets an account (no public sign-up, by design)

This platform is invite-only, not self-serve — worth stating explicitly
since it's a common point of confusion:

1. **The very first operator account** for a fresh environment is created
   via a backend command (`cmd/seed-operator`), not through any web page —
   there's deliberately no unauthenticated "create the first admin" HTTP
   route, since that would itself be a security hole.
2. **Every account after that** comes from an operator using the Invite
   User page (or `POST /v1/users/invite`) — they enter an email, role, and
   (for a restricted account) a site; the invitee gets a real email with a
   link to set their own password.

There is no "Sign Up" button on the marketing site, and that's not a gap —
there's genuinely nowhere for one to go.

## 6a. Per-site country and multi-grid emissions

Every site now has its own country (a 2-letter code, e.g. `NG`, `GB`),
required at creation and correctable afterward from Site Detail. This
matters because CO₂-offset math depends on which grid a site's power
displaces:

- **A single site's emissions** always resolve through *that site's own*
  country's emission factor.
- **A fleet spanning more than one country** sums CO₂ per country using
  each one's own factor — never blending two different grids' factors into
  one number. If a country in the mix has no factor configured yet, its
  sites' generation is excluded from the total (never guessed) and
  reported separately as "unconfigured," rather than either failing the
  whole view or silently under-reporting.

## 7. The dashboard: what's real, what's illustrative

Everything on the dashboard is backed by a real API call — no invented
numbers, per the same discipline the backend already enforces:

- **Fleet Dashboard** — real KPIs (sites, capacity, 30-day energy, CO₂
  offset), a real weather widget (Open-Meteo, keyless, based on the first
  site with a saved location), an environmental-impact panel using real
  EPA GHG-equivalency constants applied to real measured CO₂ data.
- Metrics with no real data source yet (battery storage, grid import on
  the dashboard's own "Energy Flow" panel — this platform has no
  battery/grid telemetry concept) render "Not tracked," never a fabricated
  figure.
- The **landing page's** Energy Flow illustration uses example numbers —
  normal marketing-site practice (like any product screenshot with demo
  data), not a claim about a specific customer's fleet, and it's called
  out as such in the code.

## 8. The visual design system

Redesigned this session from an earlier dark-industrial look to a light/
glass system: soft white/glass cards, sky-blue primary + amber secondary,
Plus Jakarta Sans throughout, large rounded geometry, soft diffused
shadows. The landing page's hero and closing CTA use real (rights-checked)
stock video backgrounds; the "From Chaos to Control" cards on the landing
page have a springy hover/nudge animation. Adding a site uses an actual
Google Map (search, click, or drag a pin) that auto-fills address,
country, and timezone from the picked location — not three separate
manual fields anymore.

## 9. Deployment status

- **Local/self-hosted infrastructure**: TimescaleDB and Mosquitto run as
  plain Docker containers — not a managed cloud database/broker. A
  `docker-compose.yml` now also builds and runs the API, ingestor, and web
  frontend as containers alongside them, plus one-shot `migrate` and
  `seed-operator` services.
- **Target server**: a self-managed VPS, reached over SSH — not yet fully
  cut over; DNS for the production domain and its `api.` subdomain needs
  to point at that server before the containers there are live to the
  public internet.
- **TLS in production**: terminated at Cloudflare's edge (the domain is
  proxied through Cloudflare), rather than the API container presenting
  its own certificate — `API_TLS_CERT_FILE`/`KEY_FILE` are deliberately
  left unset for this setup.
- **Object storage**: Cloudflare R2 for exported files, kept private,
  served only via time-limited presigned links.

## 10. Legal pages

Terms of Service, Privacy Policy, and Security pages now exist (linked
from the marketing site's footer, which previously pointed nowhere).
Grounded in what the platform actually stores and does — not generic
boilerplate — and drafted with reference to Nigeria's NDPA 2023 and the UK
GDPR/DPA 2018, since the business operates in both. **These still need a
lawyer's review before going live** — the governing-law jurisdiction and
contact addresses are explicit placeholders, not finished facts.

## 11. What's deliberately not built yet (so you're not caught off guard)

- **No real Performance Ratio, no weather-aware anomaly detection** — both
  need an irradiance data source beyond the current-conditions weather
  widget, which shows live weather but isn't fed into either calculation.
- **No PDF reports** — CSV/async export works; PDF would be a new
  dependency, a deliberate go/no-go decision not yet made.
- **No push notifications / alerting to a person** — anomalies and
  low-coverage warnings are visible in-dashboard; nobody gets pinged yet.
- **No general "edit a site" form** — beyond the new country-correction
  control on Site Detail, a site's other fields (name, address, system
  size, etc.) can't be edited after creation yet.
- **The day/night check is deliberately coarse** — assumes night is
  8pm–5am local time rather than real sunrise/sunset; only ever flags,
  never rejects.
- **Mutual TLS on the MQTT broker isn't enabled** — devices authenticate
  with username/password today, not a client certificate.
- **DNS/production cutover isn't finished** — see section 9.

## 12. A suggested live demo script

**API-level (unchanged from earlier phases, still accurate):**
1. Log in as an operator via `POST /v1/auth/login` — show the token comes
   back with a role.
2. Register a site and device — point out the device secret shown once.
3. Publish a normal reading, then an oversized one (rejected, but
   audited), then a lower-energy one simulating a reboot (flagged, not
   rejected, and the day's energy total is still correct).
4. Pull fleet health as operator, then the same call as a restricted user
   — show the 403.
5. Try the CO₂ endpoint before configuring a factor (409, explained), set
   one, show it succeed with the exact factor cited.
6. Revoke a device, show its next message still gets audited but rejected.

**Now also walk through the actual dashboard, since it exists:**
7. Open the login page — explain there's no sign-up button and why
   (section 6), log in as the seeded operator.
8. Fleet Dashboard — point out every number is real, including the
   weather widget and environmental-impact panel; explain what a "Not
   tracked" field means and why it's not filled with a fake number.
9. Add a site — search an address on the map, show country and timezone
   auto-fill from the pick, save it.
10. Site Detail — correct a site's country inline, explain why (the
    country column was backfilled to a guess for pre-existing sites).
11. Fleet Analytics with a fleet spanning more than one country — show the
    per-country emissions breakdown, and what "unconfigured" looks like
    for a country with no factor set yet.
12. Invite a user from the Invite User page — this is the only way any
    account after the first gets created.
13. Visit the footer's Privacy Policy / Terms / Security links — show
    they're real pages now, not dead links.

## 13. Quick glossary

- **Telemetry** — the actual reading data (power, energy, timestamp, status).
- **Mosquitto / MQTT** — the message-delivery system devices use.
- **Ingestor** — receives, checks, and stores telemetry.
- **Provenance** — how a reading was obtained: `metered` (live),
  `backfilled` (arrived late from a buffered outage).
- **Quality flag** — a note on an accepted reading worth a second look.
- **JWT** — the login token proving who a user is and their role.
- **Operator / restricted** — the two account types.
- **Continuous aggregate / rollup** — pre-computed daily totals, kept
  automatically up to date.
- **Specific yield** — energy generated ÷ a site's rated size (kWh/kWp).
- **Capacity factor** — actual output vs. a flat theoretical maximum; not
  Performance Ratio.
- **Cohort** — a named group of sites reported on together.
- **Emission factor** — kg CO₂ per kWh from a given country's grid; now
  resolved per-site, never one global assumption (section 6a).
- **R2** — Cloudflare's S3-compatible object storage, used for export
  files.
- **NDPA / UK GDPR** — Nigeria's Data Protection Act 2023 and the UK's
  data protection regime; both apply since the business operates in both
  countries (section 10).
