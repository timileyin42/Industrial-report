-- name: CreateVendorConnection :one
INSERT INTO vendor_connections (site_id, provider, encrypted_credential, created_by_user_id)
VALUES ($1, $2, $3, $4)
RETURNING id, site_id, provider, external_ref, status, last_synced_at, last_error, created_at;

-- name: ListVendorConnectionsForSite :many
SELECT id, site_id, provider, external_ref, status, last_synced_at, last_error, created_at
FROM vendor_connections
WHERE site_id = $1
ORDER BY created_at DESC;

-- name: ListActiveVendorConnections :many
-- Feeds cmd/vendor-sync's poll loop — every connection the sync binary
-- should still be trying, across every customer: 'pending' (never
-- synced yet — MarkVendorConnectionSynced is what promotes a row to
-- 'active', so pending has to be included here or a brand-new
-- connection could never reach 'active' in the first place), 'active',
-- and 'error' (retried every cycle rather than given up on — a vendor
-- outage or a momentarily wrong password shouldn't need a customer to
-- reconnect by hand). Only 'revoked' is excluded — the customer's own
-- explicit stop signal.
SELECT id, site_id, provider, encrypted_credential, external_ref, last_synced_at
FROM vendor_connections
WHERE status IN ('pending', 'active', 'error')
ORDER BY id;

-- name: GetVendorConnectionCredential :one
-- Decryption happens in Go (internal/registry/vendor_connections.go) —
-- this just returns the ciphertext, never anything already decrypted.
SELECT encrypted_credential FROM vendor_connections WHERE id = $1;

-- name: MarkVendorConnectionSynced :exec
UPDATE vendor_connections
SET status = 'active', last_synced_at = $2, last_error = NULL, external_ref = coalesce(sqlc.narg('external_ref'), external_ref)
WHERE id = $1;

-- name: MarkVendorConnectionError :exec
-- A provider-level failure marks only this one connection degraded —
-- never touches devices.last_contact_at (device-offline) or any other
-- customer's own connection row. See internal/syncengine's isolation
-- requirement.
UPDATE vendor_connections SET status = 'error', last_error = $2 WHERE id = $1;

-- name: RevokeVendorConnection :exec
UPDATE vendor_connections SET status = 'revoked' WHERE id = $1 AND site_id = $2;
