# Solarman integration — status and activation steps

`cmd/solarman-sync` is a cloud-import connector for Solarman's Open API
(`doc.solarmanpv.com`) — a real, officially documented API, unlike
`cmd/pvpro-sync`'s reverse-engineered PV Pro connector. Solarman is a
white-label monitoring backend used by roughly 200 inverter brands (Deye,
Growatt, Solis, GoodWe, SMA, Sungrow, Sofar, SolaX, Huawei, and more), so
this one connector covers any customer whose inverter reports through it
— the same "one connector, many brands" leverage `pvpro-sync` already
gets from PV Pro/Sunsynk/Powerview sharing the E-linter CSP backend, at
far larger scale and without reverse-engineering.

## Current status

**Code is written and builds cleanly, but has not been run against a
real Solarman account** — there are no API credentials yet. It's wired
into `docker-compose.yml` behind the `solarman` profile specifically so
it does **not** start with a plain `docker compose up -d` (it would
crash-loop without `SOLARMAN_APP_ID`/`SOLARMAN_APP_SECRET`/etc. set).

## What's confirmed vs. best-effort

Everything in `cmd/solarman-sync/solarman_client.go`'s auth flow,
station/device discovery shape, and the `currentData` key mapping
(`T_AC_OP`, `PVTP`, `Et_ge0`, `B_left_cap1`, `E_Puse_t1`, `PG_Pt1`) is
confirmed against two independent, real, tested Solarman Open API client
implementations (a Home Assistant integration's source code) — not
guessed.

**Not yet confirmed against a live account:**
- The exact field name for a station's GPS coordinates. `stationCoords()`
  tries several plausible candidates and falls back to "no coordinates"
  (never a fabricated `0,0`) — a station registered without a location
  needs one manual `PATCH /v1/sites/:id/location` until this is verified
  and fixed against real data.
- `battery_voltage_v` — pack voltage only appears on Solarman's separate
  `BATTERY` device-type response, which this connector doesn't currently
  fetch (only `INVERTER`-type devices are registered/synced — a
  station's own battery/collector entries are finer-grained diagnostics
  for the same physical system, not a second thing to monitor
  separately). A real, flagged gap, not an oversight.
- Whether `deviceListItems`/`stationList` carry a model/brand name field
  useful for `devices.inverter_model` — currently left blank.

## To activate

1. **Request API access.** Have a Solarman account (Smart or Business
   tier), then email `customerservice@solarmanpv.com` including:
   - The exact sentence: *"I have understood all the contents of the
     Developer Agreement and agree to be bound by it"*
   - The account email
   - Your role (individual / installer / investor / distributor)
   - Reason for API access (include a website URL if you have one)
   - Plan type: Free tier is capped at **200,000 API calls/year and 3
     plants** — see the polling-interval note below before assuming Free
     is enough for a real fleet.

   Approval isn't instant (no fixed SLA is published). You'll receive an
   **App ID and App Secret** by email once approved.

2. **Set the real credentials** in `.env` (never commit them):
   ```
   SOLARMAN_APP_ID=<from Solarman>
   SOLARMAN_APP_SECRET=<from Solarman>
   SOLARMAN_EMAIL=<the Solarman account's own login email>
   SOLARMAN_PASSWORD=<the Solarman account's own login password>
   ```

3. **Mind the free-tier call budget before picking a poll interval.**
   Each sync cycle costs roughly `1 + stations + devices` API calls
   (`list_stations`, `list_devices` per station, `currentData` per
   device). A modest 10-device/3-station fleet is ~14 calls/cycle —
   polling every 30 seconds (matching `pvpro-sync`'s cadence) would burn
   the entire annual free-tier budget in about 5 days. The default,
   `SOLARMAN_POLL_INTERVAL_SECONDS=1200` (20 minutes), keeps a 10-device
   fleet under roughly a third of the free-tier budget. Recompute for
   your actual fleet size before changing it, and consider the paid tier
   once real device counts are known.

4. **Start it:**
   ```bash
   docker compose --profile solarman build solarman-sync
   docker compose --profile solarman up -d solarman-sync
   ```

5. **Verify against real data before trusting it** — same discipline
   used for every other real data source in this project. Check:
   ```bash
   docker logs -f zgnis-solarman-sync
   ```
   for `solarman: synced device ...` lines with plausible power/energy
   values, then cross-check a device's real numbers directly against the
   Solarman mobile app/web portal for that same account, the same way
   `pvpro-sync` was verified against the PV Pro app during its own
   rollout. If a station registers without GPS coordinates or a reading
   looks wrong, that's the "not yet confirmed" list above — fix the
   specific field mapping in `solarman_client.go`/`main.go`'s
   `buildReading()` once the real response shape is visible in the logs,
   the same way PV Pro's own field names were refined iteratively
   against real responses.

## Why not build one connector per vendor instead?

Researched as part of choosing this approach (see conversation history)
— there's no single API that works for literally every inverter vendor
without any vendor-specific code, because cloud APIs aren't standardized
across manufacturers the way, say, Modbus is at the local-network level.
Solarman was chosen as the first (and highest-leverage) connector because
it's a real, officially documented API that already covers ~200 brands
in one integration — the same leverage already proven with PV Pro/
E-linter, but broader and without reverse-engineering risk.

Two follow-ups flagged but not built yet, pending a decision on
priority:
- **Direct connectors for major brands Solarman doesn't cover** — e.g.
  GivEnergy (UK-based, has its own official API at
  `givenergy.cloud/docs/api`) or SolarEdge (`developer.solaredge.com`).
- **Enode** (`enode.com`) — a commercial aggregator API covering 20+
  solar inverter/battery brands behind one unified integration, free for
  up to 5 devices then paid. Worth evaluating as a fallback for brands
  outside Solarman's coverage once real customer demand shows which
  brands actually need it, rather than integrating it speculatively now.

A genuinely separate, larger piece of work — not yet started — is
**per-user credential storage**: everything above still uses one shared
account (like `pvpro-sync`'s own PV Pro login), not a self-service
"connect your own inverter account" flow for arbitrary end users. That
needs encrypted per-user vendor-credential storage and a real UI, and is
a deliberate follow-up, not something quietly folded into this change.
