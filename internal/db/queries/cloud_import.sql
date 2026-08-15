-- name: CreateCloudImportToken :one
INSERT INTO cloud_import_tokens (device_id, token_hash) VALUES ($1, $2) RETURNING *;

-- name: RevokeCloudImportTokensForDevice :exec
-- Issuing a new token revokes any previous one for the same device — one
-- active credential at a time, same model as device secret rotation.
UPDATE cloud_import_tokens SET revoked_at = now() WHERE device_id = $1 AND revoked_at IS NULL;

-- name: ListActiveCloudImportTokensForDevice :many
-- Bounded to one device's own tokens (there's realistically at most one
-- active, occasionally two mid-rotation) — verified by hash comparison
-- in Go, same pattern as invites/password-reset tokens, since a token
-- isn't looked up by its plaintext value.
SELECT * FROM cloud_import_tokens WHERE device_id = $1 AND revoked_at IS NULL;

-- name: MarkCloudImportTokenUsed :exec
UPDATE cloud_import_tokens SET last_used_at = now() WHERE id = $1;
