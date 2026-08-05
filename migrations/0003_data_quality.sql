-- +goose Up

-- Wall-clock "have we heard from this device at all" signal, distinct from
-- last_seen_at (the *reading timestamp* of the latest accepted reading).
-- Needed to tell a true outage apart from a device that's reconnected and
-- is actively replaying a buffered backlog of old readings.
ALTER TABLE devices ADD COLUMN last_contact_at timestamptz;

CREATE INDEX idx_devices_last_contact_at ON devices (last_contact_at)
    WHERE revoked_at IS NULL;

-- Review-worthy conditions on an otherwise-accepted, real reading (e.g. a
-- detected energy-counter reset, a night-time non-zero-output flag) — an
-- array so future flag types don't need another migration. Deliberately
-- separate from `provenance` (which is about *how* a reading was obtained
-- and must never be blended) — conflating the two would violate CLAUDE.md.
ALTER TABLE telemetry ADD COLUMN quality_flags text[] NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE telemetry DROP COLUMN IF EXISTS quality_flags;
DROP INDEX IF EXISTS idx_devices_last_contact_at;
ALTER TABLE devices DROP COLUMN IF EXISTS last_contact_at;
