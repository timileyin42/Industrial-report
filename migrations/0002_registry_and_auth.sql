-- +goose Up

-- Site "name" is distinct from "address" per the design export (add_site
-- screen has both) — additive column, doesn't touch 0001.
ALTER TABLE sites ADD COLUMN name text;

-- Roles: operator sees everything, restricted is scoped to exactly one
-- site. See CLAUDE.md "Access control" — enforced server-side, not just UI.
CREATE TYPE user_role AS ENUM ('operator', 'restricted');

CREATE TABLE users (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email         text NOT NULL UNIQUE,
    password_hash text NOT NULL,       -- bcrypt, never plaintext
    role          user_role NOT NULL,
    site_id       text REFERENCES sites(site_id),  -- required for restricted, NULL for operator
    created_at    timestamptz NOT NULL DEFAULT now(),
    disabled_at   timestamptz,          -- soft-disable, mirrors devices.revoked_at

    CONSTRAINT restricted_requires_site
        CHECK (role = 'operator' OR site_id IS NOT NULL)
);

CREATE INDEX idx_users_site ON users (site_id);

-- Secret rotation bookkeeping — Phase 1 issues/rotates device secrets;
-- track when, without ever storing the plaintext value.
ALTER TABLE devices ADD COLUMN secret_last_rotated_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE devices ADD COLUMN install_notes text;

-- Admin-action audit log — separate from ingestion_audit_log (data-quality
-- concern) per CLAUDE.md: "don't conflate the two." Write-path only in
-- Phase 1; a browsing/reporting UI on this table is Phase 3.
CREATE TABLE user_action_audit_log (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor_user_id bigint REFERENCES users(id),
    action        text NOT NULL,   -- e.g. 'site.create', 'device.register', 'device.revoke', 'device.rotate_secret', 'auth.login'
    target_type   text,            -- 'site' | 'device' | 'user'
    target_id     text,
    metadata      jsonb,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_user_action_audit_actor ON user_action_audit_log (actor_user_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS user_action_audit_log;
ALTER TABLE devices DROP COLUMN IF EXISTS install_notes;
ALTER TABLE devices DROP COLUMN IF EXISTS secret_last_rotated_at;
DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS user_role;
ALTER TABLE sites DROP COLUMN IF EXISTS name;
