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
-- and 'error' (a transient/unconfirmed failure — vendor outage, network
-- blip — retried every cycle rather than given up on). Both 'revoked'
-- (the customer's own explicit stop) and 'invalid_credentials' (the
-- vendor's own response confirmed the password is actually wrong — see
-- MarkVendorConnectionInvalidCredentials) are excluded: retrying a
-- confirmed-wrong password forever isn't just wasteful, it risks
-- locking the customer out of their own vendor account (observed live
-- against Deye's login endpoint). A customer reconnects to retry it.
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

-- name: UpdateVendorConnectionCredential :exec
-- Provider-agnostic: re-encrypts and overwrites the whole credential
-- blob, used both by the OAuth callback (storing the very first token)
-- and by cmd/vendor-sync's refresh-before-poll (storing a rotated
-- access/refresh token pair) — the column has no idea which shape it
-- holds, decryption is what interprets it.
UPDATE vendor_connections SET encrypted_credential = $2 WHERE id = $1;

-- name: MarkVendorConnectionError :exec
-- A provider-level failure marks only this one connection degraded —
-- never touches devices.last_contact_at (device-offline) or any other
-- customer's own connection row. See internal/syncengine's isolation
-- requirement.
UPDATE vendor_connections SET status = 'error', last_error = $2 WHERE id = $1;

-- name: MarkVendorConnectionInvalidCredentials :exec
-- Distinct from MarkVendorConnectionError: this is only ever called
-- when the vendor's own response confirmed the account/password itself
-- is wrong (see syncengine.ErrInvalidCredentials), never for a
-- transient failure. 'invalid_credentials' rows are excluded from
-- ListActiveVendorConnections — the poll loop stops retrying this
-- connection until the customer reconnects with corrected credentials.
UPDATE vendor_connections SET status = 'invalid_credentials', last_error = $2 WHERE id = $1;

-- name: RevokeVendorConnection :exec
UPDATE vendor_connections SET status = 'revoked' WHERE id = $1 AND site_id = $2;
