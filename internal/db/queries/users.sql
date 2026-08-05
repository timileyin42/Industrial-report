-- name: CreateUser :one
INSERT INTO users (email, password_hash, role, site_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: CreateUserActionAuditLog :exec
INSERT INTO user_action_audit_log (actor_user_id, action, target_type, target_id, metadata)
VALUES ($1, $2, $3, $4, $5);
