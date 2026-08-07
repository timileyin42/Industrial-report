-- name: CreatePasswordResetToken :one
INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListActivePasswordResetTokens :many
-- Same small, TTL-bounded scan-and-bcrypt-compare pattern as
-- ListActiveInvites — see migrations/0007's comment.
SELECT * FROM password_reset_tokens WHERE used_at IS NULL AND expires_at > now();

-- name: MarkPasswordResetTokenUsed :exec
UPDATE password_reset_tokens SET used_at = now() WHERE id = $1;
