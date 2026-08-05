-- name: ListUserActionAuditLog :many
-- The Phase 3 catch-up on migration 0002's own deferred TODO ("a
-- browsing/reporting UI on this table is Phase 3"). Keyset-paginated on
-- (created_at, id); every filter is optional (NULL = don't filter on it).
SELECT
    l.id, l.actor_user_id, u.email AS actor_email, l.action, l.target_type, l.target_id, l.metadata, l.created_at
FROM user_action_audit_log l
LEFT JOIN users u ON u.id = l.actor_user_id
WHERE
    (sqlc.narg('actor_user_id')::bigint IS NULL OR l.actor_user_id = sqlc.narg('actor_user_id'))
    AND (sqlc.narg('action')::text IS NULL OR l.action = sqlc.narg('action'))
    AND (sqlc.narg('target_type')::text IS NULL OR l.target_type = sqlc.narg('target_type'))
    AND (sqlc.narg('target_id')::text IS NULL OR l.target_id = sqlc.narg('target_id'))
    AND (sqlc.narg('from_ts')::timestamptz IS NULL OR l.created_at >= sqlc.narg('from_ts'))
    AND (sqlc.narg('to_ts')::timestamptz IS NULL OR l.created_at <= sqlc.narg('to_ts'))
    AND (
        sqlc.narg('cursor_created_at')::timestamptz IS NULL
        OR (l.created_at, l.id) < (sqlc.narg('cursor_created_at')::timestamptz, sqlc.narg('cursor_id')::bigint)
    )
ORDER BY l.created_at DESC, l.id DESC
LIMIT sqlc.arg('page_limit');
