-- +goose Up
-- Cloud-to-cloud ingestion path: a genuinely vendor-agnostic alternative
-- to the MQTT pipeline, for a device whose inverter only reports into a
-- manufacturer's own cloud app rather than talking Modbus/MQTT directly.
-- This platform never integrates with any specific vendor's proprietary
-- API (every one is different, and most aren't public) — instead it
-- issues a per-device bearer token and accepts readings pushed to it in
-- one fixed JSON shape, from whatever external glue (a scraper, a
-- scheduled script, a Google Apps Script watching an export folder)
-- actually holds that vendor's own credentials. We never store or see
-- any third-party vendor password — only our own token, hashed exactly
-- like a device's MQTT secret (auth.HashSecret).
CREATE TABLE cloud_import_tokens (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    device_id    text NOT NULL REFERENCES devices(device_id),
    token_hash   text NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz,
    revoked_at   timestamptz
);

-- Partial index: only active tokens are ever looked up (by device_id,
-- during request auth) — a revoked token is never queried again except
-- for audit purposes, which don't need an index.
CREATE INDEX idx_cloud_import_tokens_device ON cloud_import_tokens(device_id) WHERE revoked_at IS NULL;

-- +goose Down
DROP TABLE cloud_import_tokens;
