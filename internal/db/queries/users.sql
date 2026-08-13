-- name: CreateUser :one
INSERT INTO users (email, password_hash, role, site_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetEarliestOperator :one
-- The first operator account ever created in this environment — i.e.
-- the one created via cmd/seed-operator, not necessarily still the only
-- operator. Used to route marketing-site notifications (demo requests)
-- to a real inbox without a dedicated "company notification address"
-- concept in the schema.
SELECT * FROM users WHERE role = 'operator' ORDER BY created_at ASC, id ASC LIMIT 1;

-- name: ListUsers :many
-- Keyset pagination, same convention as ListSites/ListDevices.
SELECT * FROM users
WHERE (
    sqlc.narg('cursor_created_at')::timestamptz IS NULL
    OR (created_at, id) < (sqlc.narg('cursor_created_at')::timestamptz, sqlc.narg('cursor_id')::bigint)
)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('page_limit');

-- name: SetUserDisabled :one
-- disabled_at itself, not a boolean flag — NULL means active, a real
-- timestamp means disabled since that moment (audit-friendly, matches
-- devices.revoked_at's existing convention in this codebase).
UPDATE users SET disabled_at = $2 WHERE id = $1
RETURNING *;

-- name: CreateUserActionAuditLog :exec
INSERT INTO user_action_audit_log (actor_user_id, action, target_type, target_id, metadata)
VALUES ($1, $2, $3, $4, $5);
