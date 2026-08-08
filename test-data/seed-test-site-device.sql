-- Matches telemetry-sample.jsonl / telemetry-sample.csv
-- Run this before loading the sample telemetry, so foreign keys resolve.

INSERT INTO sites (site_id, address, gps_lat, gps_lng, inverter_make_model,
                    system_size_kw, install_date, timezone)
VALUES
    ('TEST-SITE-001', 'Simulated Rooftop, Lekki Phase 1, Lagos', 6.4432, 3.4726,
     'Growatt SPF 5000ES', 5.0, '2026-01-15', 'Africa/Lagos'),
    ('TEST-SITE-002', 'Simulated Commercial Roof, Ikeja, Lagos', 6.6018, 3.3515,
     'Huawei SUN2000-10KTL', 10.0, '2025-11-02', 'Africa/Lagos');

INSERT INTO devices (device_id, site_id, secret_hash)
VALUES
    ('TEST-ZG-0001', 'TEST-SITE-001', 'placeholder-set-real-hash-before-broker-auth-is-enforced'),
    ('TEST-ZG-0002', 'TEST-SITE-002', 'placeholder-set-real-hash-before-broker-auth-is-enforced');

-- CLEANUP (run before any real client sees the dashboard):
-- DELETE FROM ingestion_audit_log WHERE device_id IN ('TEST-ZG-0001','TEST-ZG-0002');
-- DELETE FROM telemetry WHERE device_id IN ('TEST-ZG-0001','TEST-ZG-0002');
-- DELETE FROM devices WHERE device_id IN ('TEST-ZG-0001','TEST-ZG-0002');
-- DELETE FROM sites WHERE site_id IN ('TEST-SITE-001','TEST-SITE-002');
