-- +goose Up
-- Foundation for the self-service "Connect Your Inverter" flow: a
-- customer's own manufacturer-cloud login (PV Pro/E-linter CSP first,
-- more vendors later), stored so a background sync process can poll it
-- independently of any other customer's connection. One row per
-- customer-connected account, distinct from cmd/pvpro-sync's/
-- cmd/solarman-sync's own single shared operator account — those are
-- untouched by this table.
CREATE TABLE vendor_connections (
    id                   bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    site_id              text NOT NULL REFERENCES sites(site_id),
    provider             text NOT NULL,
    external_ref         text,
    -- AES-256-GCM ciphertext of the vendor credential (JSON-encoded
    -- {email,password} for a password-form provider). Never plaintext,
    -- never logged — see internal/registry/vendor_connections.go.
    encrypted_credential bytea NOT NULL,
    status               text NOT NULL DEFAULT 'pending',
    last_synced_at       timestamptz,
    last_error           text,
    created_by_user_id   bigint REFERENCES users(id),
    created_at           timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_vendor_connections_site ON vendor_connections(site_id);

-- Partial index: the sync loop only ever queries active connections,
-- same "index only what's actually queried" rule cloud_import_tokens
-- already follows (migrations/0017_cloud_import.sql).
CREATE INDEX idx_vendor_connections_provider_active
  ON vendor_connections(provider) WHERE status = 'active';

-- +goose Down
DROP TABLE IF EXISTS vendor_connections;
