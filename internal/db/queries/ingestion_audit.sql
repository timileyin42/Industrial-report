-- name: ListIngestionAuditLog :many
-- The ingestor's own write-only audit trail (every message received,
-- before validation — see migrations/0001_init.sql), finally given a read
-- path. Kept deliberately separate from user_action_audit_log — this is
-- data-quality/verification evidence, not an admin-action trail (see
-- CLAUDE.md: "don't conflate the two"). Joins devices to resolve site_id
-- for site-scoping (restricted users) and display, since the log itself
-- only stores device_id. A device that's since been deleted still shows
-- its audit rows with site_id NULL rather than disappearing.
SELECT
    l.id, l.device_id, d.site_id, l.raw_payload, l.received_at, l.processed, l.error
FROM ingestion_audit_log l
LEFT JOIN devices d ON d.device_id = l.device_id
WHERE
    (sqlc.narg('site_id')::text IS NULL OR d.site_id = sqlc.narg('site_id'))
    AND (sqlc.narg('device_id')::text IS NULL OR l.device_id = sqlc.narg('device_id'))
    AND (sqlc.narg('errors_only')::boolean IS NULL OR (l.error IS NOT NULL) = sqlc.narg('errors_only'))
    AND (sqlc.narg('from_ts')::timestamptz IS NULL OR l.received_at >= sqlc.narg('from_ts'))
    AND (sqlc.narg('to_ts')::timestamptz IS NULL OR l.received_at <= sqlc.narg('to_ts'))
    AND (
        sqlc.narg('cursor_received_at')::timestamptz IS NULL
        OR (l.received_at, l.id) < (sqlc.narg('cursor_received_at')::timestamptz, sqlc.narg('cursor_id')::bigint)
    )
ORDER BY l.received_at DESC, l.id DESC
LIMIT sqlc.arg('page_limit');

-- name: LastIngestionReceivedAt :one
-- Most recent message the ingestor has seen, fleet-wide, regardless of
-- whether it passed validation — the Dashboard's ingestion-pipeline
-- status widget uses "how long ago was that" as its health signal, not
-- a synthetic uptime percentage this platform has no way to compute.
SELECT max(received_at)::timestamptz AS last_received_at FROM ingestion_audit_log;
