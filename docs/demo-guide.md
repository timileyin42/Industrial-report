# Zgnis Solar Platform — What We've Built (Demo & Walkthrough Guide)

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
avoided. **Phases 0 through 3 build that entire pipeline, minus the actual
visual dashboard** — everything today is proven through direct API calls
(what a future dashboard would call behind the scenes), not through
screens yet.

## 2. The pipeline, plain-English

```
Solar inverter → little box on-site → sends a message over the internet
   → our "mailroom" (Mosquitto) sorts messages by device
   → our "intake clerk" (the ingestor) checks each message makes sense
   → stores it in the database
   → an overnight "summarizer" pre-computes daily totals per device
   → our API hands out raw data AND summarized KPIs to whoever's allowed to see them
```

- **Mosquitto** is just message plumbing. It knows nothing about solar
  panels — it just makes sure device A's message can't be mistaken for
  device B's, and that only device A can send messages "as" device A.
- **The ingestor** is the one piece of code that actually understands what
  a solar reading means. It's the "quality control" step.
- **The database** (TimescaleDB, a version of Postgres built for
  time-stamped data) is where every reading lives permanently.
- **The summarizer** (new since Phase 3 — a Timescale "continuous
  aggregate") is a background job that keeps a running daily total per
  device, so KPI questions like "how much energy did this site make this
  month" don't have to re-scan every single reading ever taken. This is
  the difference between a query taking milliseconds vs. minutes once
  there's years of data.
- **The API** is the only door into that data — nobody, not even our own
  dashboard, is allowed to read the database directly. Everything goes
  through the API so access rules are enforced in one place.

## 3. What each phase actually delivered

### Phase 0 — "prove the pipe works at all"
One simulated device, one message, one chart-ready endpoint. No login, no
security, no rules — purely "can data get from a device to a chart." ✅ Done.

### Phase 1 — "make it a real, access-controlled system"
- **Device registry**: an admin can create a site and register a device
  through the API (not by hand-editing the database anymore). Registering
  a device generates it a secret password, shown to you **exactly once** —
  if you lose it, you rotate it for a new one, you never get the old one back.
- **Login & roles**: two kinds of accounts —
  - **Operator** — sees everything, every site, the whole fleet.
  - **Restricted** — locked to exactly one site. If a restricted user tries
    to look at a different site's data — even by guessing a URL — they get
    a flat **403 Forbidden**, not the data, not even a "not found." This is
    enforced on the server, so it can't be bypassed by a clever app user.
- **Admin audit trail**: every admin action (create a site, register a
  device, revoke a device, log in) is permanently logged with who did it
  and when — separate from the data-quality log below.

### Phase 2 — "catch bad data and know when a device goes dark"
Validation rules, gap/offline detection, and a fleet health view — see
section 4 for the specific numbers and section 5 for the "rejected vs.
flagged" and "offline vs. data gap" distinctions, both still accurate today.

### Phase 3 — "turn raw readings into the numbers people actually want"
This is the analytics layer — the part a client actually reads, rather
than the plumbing underneath it.

- **Energy generated** — daily, weekly, or monthly, per site or fleet-wide.
- **Specific yield (kWh per kWp)** — energy normalized by system size, so a
  tiny site and a huge site can be compared fairly.
- **Peak output** — the highest instantaneous power a site hit, and exactly
  when.
- **Capacity factor** — how hard a site is working relative to its
  theoretical maximum, assuming it ran flat-out 24/7. (See section 4 for
  why this is deliberately *not* called "Performance Ratio.")
- **CO₂ avoided** — energy generated × a configurable "how dirty is the
  grid" number, reported in kg and tonnes, with the exact factor used
  shown on every figure. **Nothing is computed until an operator explicitly
  sets that number** — see section 4.
- **Comparisons** — a site against its own last period, a site against the
  fleet average and percentile rank, and segmented views by system size or
  inverter brand.
- **Fleet trends** — total installed capacity and total energy over time,
  month-on-month.
- **Cohorts** — if sites are grouped into projects, a cohort view rolls up
  energy and capacity for just that group.
- **Anomaly flags** — a coarse "this site's output dropped off a cliff
  compared to its own recent normal" check (see section 4 — this is
  deliberately simple, not weather-aware).
- **CSV export** — raw telemetry or period summaries, downloadable.
- **Admin audit log, browsable** — Phase 1 started logging every admin
  action; Phase 3 is the first time you can actually query that log
  (who did what, when, filtered by action/user/date).

## 4. The specific limits — what they are and why (the "10kW" question, and friends)

| What | Value today | Plain-English reason |
|---|---|---|
| **Power ceiling** | site's rated size × **1.5** | If a site is rated for 2kW, we reject any reading claiming more than 3kW as physically implausible — that's either a broken sensor, a wiring fault, or corrupted data. The 1.5x isn't arbitrary: real inverters are commonly built with some headroom above the panel's rated size, and output can legitimately spike briefly, so we don't want to reject genuine readings — but 1.5x still firmly catches "something is very wrong here" cases like a 2kW site reporting 10kW. |
| **"Online" cutoff** | **10 minutes** since the device last contacted us at all | If we haven't heard from a device in 10+ minutes, we consider it offline — a real outage. |
| **Expected reporting interval** | **5 minutes** | Devices are supposed to report every 1–5 minutes. We use the upper end (5 min) as the benchmark for "is this device behind" so we don't flag a perfectly healthy device that's just reporting on the slower end of normal. |
| **Backfill threshold** | **15 minutes** between when a reading says it happened and when it actually arrived | A device that's been offline buffers its readings and sends them all at once when it reconnects. If a message arrives more than 15 minutes "late," we tag it `backfilled` instead of `metered` — so nobody downstream mistakes old catch-up data for a live reading. |
| **Coverage window** | **24 hours** | The fleet health view looks at the last 24 hours to calculate "what % of expected readings did we actually get" per site and fleet-wide. |
| **Anomaly drop threshold** | **50%** below a site's own trailing 7-day average | Deliberately a big, obvious drop, not a subtle one — see below for why. |

**All of these are configuration, not hardcoded** — they can be tuned per
the client's real hardware behavior without changing code, once we see how
real devices behave in the field versus our current assumptions.

**Two Phase 3 concepts deserve their own explanation, because they're easy
to get wrong live:**

- **Capacity factor is NOT Performance Ratio (PR).** PR is the "correct"
  industry metric — it compares what a site actually produced against what
  it *should have* produced given the actual weather that day (cloud
  cover, sun angle, etc.). We don't have weather data anywhere in this
  system yet, so we can't compute real PR. What we compute instead —
  **capacity factor** — just compares actual output to a flat theoretical
  maximum (site size × 24 hours), with zero weather adjustment. It's a
  cruder, honest stand-in, and every single response says so explicitly
  so nobody mistakes it for PR in a report.
- **The anomaly check is a blunt instrument, on purpose.** Without weather
  data, we can't tell "this site made less energy today because it was
  cloudy" apart from "this site made less energy today because something
  is actually broken." So instead of trying to be clever, the check only
  flags a genuinely severe drop (50%+ below the site's own recent normal)
  — a coarse safety net, not a diagnostic tool. Every anomaly response
  says this explicitly too.

## 5. The two different "something's wrong" signals — don't mix these up in the demo

This is the single most important distinction to get right when explaining
Phase 2 (still true in Phase 3):

- **A rejected reading** — physically impossible (negative energy, power
  wildly over the site's size). This is **never stored** — it's logged in
  a permanent "raw intake" audit trail (so nothing is ever silently
  dropped without a trace) but it never touches the real data.
- **A flagged reading** — plausible but noteworthy (e.g., the device's
  cumulative energy counter went backwards, which really happens when an
  inverter reboots or gets replaced). This **is stored**, exactly as
  received, just tagged so nobody accidentally treats it as a clean data
  point later. We keep it because for solar generation records that may
  later be used for verification/emissions reporting, throwing away real
  readings is worse than flagging them. (Phase 3's energy/yield/CO₂
  numbers correctly work around a flagged reset day instead of producing
  a nonsense negative number — verified live.)
- **"Offline" vs. "data gap"** — two different problems that look similar
  but aren't:
  - **Offline** = the device hasn't contacted us at all — a true outage.
  - **Data gap** = the device *is* talking to us right now, but its latest
    *accepted* reading is stale — e.g., it just reconnected and is still
    working through a backlog of buffered readings. Not broken, just behind.

## 6. What's deliberately not built yet (so you're not caught off guard)

- **No dashboard/screens yet** — everything above is proven via direct API
  calls. The visual design system exists (`design/` folder, Stitch export)
  but no frontend has been wired up to these APIs yet.
- **No email** (verification, password reset) — genuinely not scheduled to
  any phase yet; needs a decision on an email provider first.
- **No push notifications / alerting to a person** — today, "alerts"/
  anomalies mean "queryable through the API," not "somebody gets pinged."
  There's no notification channel built yet.
- **No real Performance Ratio, no weather-aware anomaly detection** — both
  need a weather/irradiance data source that doesn't exist in this stack
  yet (see section 4). Capacity factor and the trailing-baseline anomaly
  check are the honest, weather-free stand-ins.
- **No PDF reports** — CSV export works today; PDF would mean adding a new
  code dependency, which is a deliberate go/no-go decision we haven't made,
  not an oversight.
- **No "region" field on a site** — the benchmarking view can group sites
  by system size or inverter brand for real; a "region" grouping uses
  "cohort" (project grouping) as a stand-in, since there's no dedicated
  region/state field on a site yet.
- **The emissions (CO₂) numbers don't work until someone sets the
  official grid emission factor** — this is deliberate, not a bug: that
  number has to be a real, client-confirmed figure (the concept note says
  so explicitly), so the system refuses to guess one. An operator has to
  explicitly configure it once before any CO₂ figure will compute.
- **The day/night check is deliberately coarse** — it assumes night is
  8pm–5am local time rather than calculating actual sunrise/sunset, so it
  can misjudge by up to an hour around dawn/dusk. It only ever flags,
  never rejects, so this is a safe simplification, not a real gap.
- **The fleet health view (Phase 2) still scans recent raw data** rather
  than using the new daily-summary shortcut Phase 3 built for KPIs — a
  known follow-up, not a regression.

## 7. A suggested live demo script

This mirrors exactly what was tested and verified — nothing below is
hypothetical, all of it has been run successfully:

1. **Log in as an operator.** Show the token comes back with a role.
2. **Register a new site and device.** Point out the device secret is
   shown once — this is what would get flashed on-screen in a real "add
   device" flow, then never shown again.
3. **Publish a normal reading** (via a simulated device) and show it
   landing in the site's telemetry feed within seconds.
4. **Publish an oversized reading** (e.g. 10kW on a 2kW site) — show it
   gets rejected, and show the audit trail proving it wasn't silently lost,
   just refused.
5. **Publish a reading with a lower energy count than before** (simulating
   an inverter reboot) — show it's accepted, not rejected, but flagged —
   and then show the site's energy total for that day is still correct
   despite the reset.
6. **Check a device's status** — show `online` / `data_gap` and explain
   the difference live using the "offline vs. behind" framing above.
7. **Pull up fleet health** as the operator — show fleet-wide + per-site
   numbers. Then try the same call as a restricted-role user and show the
   403 — this is the access-control story in one screen.
8. **Pull up a site's energy, specific yield, and peak output** — this is
   the "what does the client actually see" moment.
9. **Try the CO₂ endpoint before configuring a factor** — show the 409,
   explain why (no invented numbers), then set a factor and show it
   succeed with the exact factor cited in the response.
10. **Compare a site to the fleet average** and pull up the benchmark view
    segmented by system size — the "how do I stack up" story.
11. **Export a CSV** — the "give me a spreadsheet" moment.
12. **Revoke a device** and show its next message gets audited but
    rejected — proving revocation actually stops data, not just hides it
    in the UI.

## 8. Quick glossary

- **Telemetry** — the actual reading data (power, energy, timestamp, status).
- **Mosquitto / MQTT** — the message-delivery system devices use to send
  telemetry; not solar-specific, just a reliable postal service for small
  devices.
- **Ingestor** — our service that receives, checks, and stores telemetry.
- **Provenance** — how a reading was obtained: `metered` (live), `backfilled`
  (arrived late from a buffered outage), `estimated` (not used yet — reserved
  for a future gap-filling feature).
- **Quality flag** — a note on an accepted reading that something about it
  is worth a second look (e.g. an energy counter reset).
- **JWT** — the login token a user carries after signing in; proves who
  they are and what role they have on every request.
- **Operator / restricted** — the two account types; restricted is locked
  to one site, enforced on the server every time, not just hidden in a UI.
- **Continuous aggregate / rollup** — the "summarizer" from section 2: a
  pre-computed daily total per device, kept automatically up to date, so
  KPI queries stay fast as the amount of data grows.
- **Specific yield** — energy generated divided by a site's rated size
  (kWh per kWp) — lets you compare a small and large site fairly.
- **Capacity factor** — actual output vs. a flat theoretical maximum; not
  the same as Performance Ratio (see section 4).
- **Cohort** — a named group of sites (e.g. a project or region grouping)
  that can be reported on together.
- **Emission factor** — the "how many kg of CO₂ does one kWh from the grid
  represent" number, used to compute avoided emissions; must be explicitly
  set by an operator, never assumed.
