-- +goose Up

-- An invited user's row is created immediately (so role/site_id are set
-- up front by the operator) with an unusable password_hash — a random
-- secret nobody knows, hashed the same way a real password would be. The
-- user can't log in until they accept the invite and set a real password.
CREATE TABLE invites (
    id                 bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id            bigint NOT NULL REFERENCES users(id),
    token_hash         text NOT NULL,
    invited_by_user_id bigint REFERENCES users(id),
    expires_at         timestamptz NOT NULL,
    accepted_at        timestamptz,
    created_at         timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_invites_user ON invites (user_id);

-- token_hash can't be looked up by equality (bcrypt salts per-hash) — the
-- accept/reset endpoints list active (non-expired, unused) rows and
-- bcrypt-compare each. Outstanding invites/resets are always a tiny set,
-- so this is a non-issue at any real scale this platform operates at.
CREATE TABLE password_reset_tokens (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    bigint NOT NULL REFERENCES users(id),
    token_hash text NOT NULL,
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_password_reset_user ON password_reset_tokens (user_id);

-- +goose Down
DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS invites;
