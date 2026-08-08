-- name: VerifyIngestionAuditChain :one
-- Recomputes each row's expected hash using the exact same expression as
-- the ingestion_audit_log_chain() trigger (migrations/0013) and compares
-- against what's actually stored — done entirely in SQL so the digest()
-- call here is byte-for-byte the same one that wrote the hash, avoiding
-- any cross-language JSON/text canonicalization mismatch a Go-side
-- recompute would risk. lag() over id order gives each row's expected
-- prev_hash without a second table scan.
WITH ordered AS (
    SELECT
        id, device_id, raw_payload, received_at, prev_hash, entry_hash,
        coalesce(lag(entry_hash) OVER (ORDER BY id), '') AS expected_prev
    FROM ingestion_audit_log
),
checked AS (
    SELECT
        id,
        (
            entry_hash IS DISTINCT FROM encode(digest(expected_prev || '|' || device_id || '|' || raw_payload::text || '|' || received_at::text, 'sha256'), 'hex')
            OR prev_hash IS DISTINCT FROM expected_prev
        ) AS bad
    FROM ordered
)
SELECT
    count(*) FILTER (WHERE bad)::bigint AS mismatch_count,
    min(id) FILTER (WHERE bad) AS first_bad_id
FROM checked;

-- name: VerifyUserActionAuditChain :one
-- Same approach as VerifyIngestionAuditChain, against
-- user_action_audit_log_chain()'s expression instead.
WITH ordered AS (
    SELECT
        id, actor_user_id, action, target_type, target_id, metadata, created_at, prev_hash, entry_hash,
        coalesce(lag(entry_hash) OVER (ORDER BY id), '') AS expected_prev
    FROM user_action_audit_log
),
checked AS (
    SELECT
        id,
        (
            entry_hash IS DISTINCT FROM encode(
                digest(
                    expected_prev || '|' || coalesce(actor_user_id::text, '') || '|' || action || '|' ||
                    coalesce(target_type, '') || '|' || coalesce(target_id, '') || '|' ||
                    coalesce(metadata::text, '') || '|' || created_at::text,
                    'sha256'
                ),
                'hex'
            )
            OR prev_hash IS DISTINCT FROM expected_prev
        ) AS bad
    FROM ordered
)
SELECT
    count(*) FILTER (WHERE bad)::bigint AS mismatch_count,
    min(id) FILTER (WHERE bad) AS first_bad_id
FROM checked;
