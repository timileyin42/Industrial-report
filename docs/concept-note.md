**CLEAN ENERGY DATA PLATFORM**

Concept Note *--- Zgnis Next Limited*

# **1. Purpose & Scope**

This note briefs the developer responsible for building and running a
platform that collects, stores, and displays electricity-generation data
from a large number of distributed residential solar systems. Each
system reports through a monitoring device installed on site. The
platform's job is to receive that data reliably, keep it clean and
secure, and present it through dashboards.

The scope of this document is strictly technical: architecture, data
pipeline, data model, stack, and the developer's ongoing
responsibilities. It should be designed from day one to scale to
**thousands of sites** and to keep clean, exportable, auditable records
so that additional reporting or data-export requirements can be added
later without re-engineering the core.

# **2. What the Platform Does**

Five core functions:

-   **Ingest ---** receive periodic readings pushed from monitoring
    devices in the field.

-   **Store ---** persist every reading as time-stamped, per-site
    time-series data.

-   **Visualise ---** show generation over time per site and across the
    whole fleet.

-   **Manage devices ---** register devices, map each to its site,
    authenticate them, and track which are online.

-   **Analyse ---** turn raw readings into performance and environmental
    metrics, comparisons, and reports that make the data easy to
    interpret.

# **3. System Architecture**

Data flows one direction --- from the inverter at the edge to the
dashboard --- with a device registry providing the site context:

\[ Inverter \]\
\| Modbus (RS485)\
\[ Datalogger + 4G/SIM \]\
\| MQTT over TLS (or HTTPS)\
\[ Ingestion Broker / Endpoint \]\
\|\
\[ Validation & Processing \]\
\|\
\[ Time-Series Database \] \<\-\-- \[ Device Registry & Site Metadata
\]\
\|\
\[ Backend API \]\
\|\
\[ Web Dashboard \]

# **4. Edge Layer (Field Devices)**

-   Each site has a datalogger that reads the inverter over **Modbus RTU
    (RS485)** and sends readings out over its own **4G/SIM** connection.

-   Inverter brands differ in how they expose data. Each brand/model
    needs a **register-map profile** (which Modbus register holds power,
    energy, voltage, etc.). Treat these profiles as configuration, not
    code, so new brands can be added quickly.

-   Devices must **buffer readings locally** during power or network
    outages and back-fill when the connection returns, so outages create
    no permanent gaps.

-   Reading interval: target **1--5 minutes** per device. Design for
    jitter and out-of-order arrival.

# **5. Data Ingestion**

Preferred transport is **MQTT over TLS** --- lightweight and well suited
to many small intermittent devices. An HTTPS endpoint is an acceptable
simpler alternative for early testing. Every device authenticates with
its own credential (device ID + secret / client certificate); no device
is trusted without it. A typical message payload:

{\
\"device_id\": \"ZG-0001\",\
\"ts\": \"2026-07-30T13:05:00Z\", // UTC, ISO 8601\
\"power_kw\": 3.42,\
\"energy_kwh_total\": 1284.6, // cumulative meter reading\
\"voltage_v\": 231.4,\
\"status\": \"ok\"\
}

# **6. Core Data Model**

Telemetry (one row per device per reading):

  -------------------------------------------------------------------------
  **Field**          **Type**     **Description**
  ------------------ ------------ -----------------------------------------
  device_id          string       Unique ID of the reporting device

  site_id            string       Site the device is mapped to

  ts                 timestamp    Reading time, stored in UTC

  power_kw           float        Instantaneous output

  energy_kwh_total   float        Cumulative generation (monotonic)

  voltage_v          float        Optional

  status             string       ok / fault / offline code

  rssi               int          Optional signal strength for diagnostics
  -------------------------------------------------------------------------

Site metadata is stored separately (one row per site): site_id,
cohort_id, address, gps, inverter_make_model, system_size_kw,
install_date, timezone. Derive periodic energy (daily/monthly) from
differences in the cumulative energy_kwh_total rather than summing
instantaneous power. The grid emission factor is held as versioned
configuration (not per reading), so avoided-emissions figures can be
recomputed if it changes.

# **7. Storage**

-   Use a **time-series database** --- e.g. TimescaleDB (PostgreSQL
    extension) or InfluxDB --- built for high-volume time-stamped writes
    and range queries.

-   Keep raw readings; also maintain **roll-ups** (hourly/daily
    aggregates per site) so dashboards stay fast at scale.

-   Plan retention and archival early; the dataset grows continuously.

# **8. Backend & API**

-   A backend service exposes read APIs to the dashboard (per-site
    series, fleet summaries, device status) and admin APIs (register a
    device, map to a site, revoke a device).

-   The dashboard **only ever reads from the platform's own database**
    --- never directly from a device or a manufacturer cloud.

-   Provide clean, documented, versioned endpoints so export/reporting
    features can be layered on later.

# **9. Frontend / Dashboard**

-   **Operator view ---** one login showing every site: live status,
    generation charts, offline devices, fleet totals, and search/filter
    by site.

-   **Per-site view ---** a single system's generation history and
    health.

-   **Multi-tenant / roles ---** build access control so an operator
    sees everything while a restricted account can be limited to a
    single site's data. Design this in from the start; retrofitting it
    is painful.

# **10. Analytics & Insights (Decision Support)**

Raw readings alone are hard to act on. This layer turns the stored
time-series into clear metrics, comparisons, and reports so the data can
be understood at a glance and used to make informed decisions. It is
computed on top of the roll-ups, exposed through the API and dashboard,
and exportable.

**Core performance metrics (per site and fleet-wide)**

-   **Energy generated ---** daily, weekly, monthly, and cumulative.

-   **Specific yield (kWh per kWp) ---** output normalised by system
    size, so systems of different sizes compare fairly.

-   **Peak output ---** value and time of day it occurs.

-   **Availability / capacity factor ---** how consistently a system
    generates versus its potential.

-   **Performance ratio (PR) ---** actual output against what the system
    should produce for its rating and conditions --- the standard
    measure of how well a system performs.

**Environmental & emissions metrics**

Generation converts directly into avoided grid emissions, so these are
computed as first-class metrics, not add-ons:

-   **CO₂ emissions avoided ---** energy generated (kWh) × the grid
    emission factor (kg CO₂ per kWh), reported per site and fleet-wide,
    both per period and as a cumulative lifetime total.

-   **Grid emission factor ---** a configurable, versioned parameter set
    from the official national figure (for Nigeria, on the order of
    \~0.4 kg CO₂/kWh --- confirm and set the current official value).
    Versioning lets figures be recalculated if the official factor
    changes.

-   **Cumulative lifetime totals ---** maintain running lifetime energy
    and running lifetime avoided emissions per site, per cohort, and
    fleet-wide.

-   **Consistent units ---** report in kg and tonnes CO₂, stating the
    factor and period used alongside every figure.

**Comparisons & benchmarking**

-   A site against its own history (this month vs last).

-   A site against the fleet average, or its percentile rank within the
    fleet.

-   Actual vs expected for the system's size --- to surface
    under-performers.

-   Segmented views: by region/state, by system-size band, and by
    inverter brand.

**Fleet-level trends & aggregates**

-   Total installed capacity (sum of kWp) and how it grows as sites are
    added.

-   Total and cumulative energy generated across all sites.

-   Growth and month-on-month movement over time.

-   Grouping of sites into cohorts/projects, so energy and avoided
    emissions can be aggregated and reported per cohort as well as
    fleet-wide.

**Performance anomaly detection**

-   Flag sites generating materially below expectation for their size or
    the season, sudden drops against baseline, or sustained low output.
    This complements the connectivity checks in the next section, but
    focuses on generation performance rather than merely whether a
    device is online.

**Reporting & export**

-   Scheduled and on-demand summary reports (per site and fleet-wide) as
    PDF or CSV.

-   A documented data API and clean, well-labelled exports --- explicit
    units and timestamps --- so figures can feed spreadsheets or other
    tools downstream.

**Optional / later**

-   Weather or irradiance normalisation for fairer comparison; simple
    generation forecasting; and configurable KPI thresholds with
    alerting.

**Design notes:** compute analytics from the roll-ups (not raw scans)
for speed; keep each metric's definition consistent and documented as a
single source of truth; and make units and time zones explicit
everywhere.

# **11. Data Quality, Integrity & Verification Readiness**

The platform's value is trustworthy data, and the generation record may
later need to support emissions reporting and independent verification.
Integrity and traceability are therefore first-class requirements, not
afterthoughts:

-   **Sanity checks ---** flag readings that are impossible (negative
    energy, output far above system size) and expect the daylight/night
    pattern (non-zero by day, \~zero at night).

-   **Gap & offline detection ---** detect when a device stops reporting
    and raise an alert; distinguish a true outage from delayed buffered
    data.

-   **Deduplication & back-fill ---** handle repeated or late messages
    idempotently (e.g. unique on device_id + ts).

-   **Fleet health dashboard ---** percentage of devices reporting,
    data-uptime per site, last-seen times.

-   **Data completeness / coverage ---** track the percentage of
    expected readings actually received, per site and fleet; this
    coverage figure is itself a key quality metric and matters directly
    for verification-grade data.

-   **Provenance flags ---** tag each reading as directly metered versus
    estimated or back-filled, and never silently mix them; downstream
    reporting must be able to tell them apart.

-   **Audit trail & traceability ---** keep an append-only,
    tamper-evident record of readings, with every reading traceable
    through device → site → owner → cohort, so the dataset can support
    independent verification.

-   **Retention for verification ---** retain raw metered records for
    the full period any downstream reporting or verification may
    require.

# **12. Security & Data Protection (technical)**

-   Encrypt data in transit (TLS) and at rest.

-   Per-device credentials; ability to revoke a compromised device.

-   Role-based access control on the dashboard and APIs; audit logs of
    who accessed what.

-   Site metadata includes personal data (owner address/contact), so
    apply least-privilege access, encryption, retention limits, and
    breach detection. Follow the data-protection requirements the
    company specifies.

-   Regular automated backups, with restores tested.

# **13. Scalability**

Design for growth from a handful of pilot devices to thousands. Favour a
horizontally scalable broker, batched/asynchronous writes, indexed
time-series storage, and pre-computed roll-ups. Load-test with simulated
devices before large roll-outs.

# **14. Suggested Stack (options, not mandates)**

  -----------------------------------------------------------------------
  **Layer**      **Options**                 **Notes**
  -------------- --------------------------- ----------------------------
  Edge logger    Commercial 4G Modbus        Reads Modbus, pushes MQTT
                 logger, or Pi/ESP32         

  Ingestion      EMQX / Mosquitto (MQTT), or TLS + per-device auth
                 HTTPS                       

  Processing     Node.js or Python service   Validation, dedup, roll-ups

  Storage        TimescaleDB or InfluxDB     Time-series optimised

  Backend API    NestJS (Node) or FastAPI    Documented, versioned
                 (Python)                    

  Frontend       React + charting lib        Grafana is a fast internal
                                             option

  Hosting/Ops    A managed cloud + Grafana   Backups, monitoring
                 alerts                      
  -----------------------------------------------------------------------

# **15. Delivery Phases**

-   **Phase 0 --- One device end-to-end:** a single logger → broker →
    database → one live chart. Proves the whole chain.

-   **Phase 1 --- Many devices + registry:** device auth, site mapping,
    per-site and fleet views.

-   **Phase 2 --- Quality + alerting:** validation rules, offline/gap
    alerts, back-fill handling, health dashboard.

-   **Phase 3 --- Analytics, access & roles:** KPIs, comparisons and
    reports; operator and restricted views; audit logging.

-   **Phase 4 --- Harden & scale:** TLS everywhere, backups, retention,
    load-tested to thousands of devices, documented export endpoints.

# **16. Ongoing Responsibilities**

-   Keep ingestion, database, and dashboards healthy; watch uptime and
    alerts.

-   Add a register-map profile for each new inverter brand/model
    encountered in the field.

-   Monitor the data-quality dashboard and coordinate with the field
    team on offline or faulty devices.

-   Manage backups, retention, credentials, and access control.

-   Maintain clear documentation --- architecture, register maps, API,
    and runbooks for common failures.

*Technical concept note for platform development and operation. Stack
items are recommendations; the developer may propose equivalents that
meet the same reliability, security, and scalability goals.*
