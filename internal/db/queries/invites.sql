-- name: CreateInvite :one
INSERT INTO invites (user_id, token_hash, invited_by_user_id, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListActiveInvites :many
-- Small, TTL-bounded result set — see migrations/0007's comment on why
-- token_hash isn't looked up by equality.
SELECT * FROM invites WHERE accepted_at IS NULL AND expires_at > now();

-- name: MarkInviteAccepted :exec
UPDATE invites SET accepted_at = now() WHERE id = $1;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = $2 WHERE id = $1;
