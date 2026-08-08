-- +goose Up

-- Makes both audit tables genuinely tamper-evident (concept-note.md §11:
-- "verification-readiness"), not just append-only by application
-- convention. Each row gets prev_hash (the previous row's entry_hash)
-- and entry_hash (sha256 of prev_hash + this row's own fields), forming
-- a hash chain — rewriting or deleting any row breaks every hash after
-- it, which a verification pass (see VerifyIngestionAuditChain /
-- VerifyUserActionAuditChain queries) can detect and pinpoint.
--
-- Chaining is computed by a BEFORE INSERT trigger (not application code)
-- so it holds regardless of which process/language writes the row —
-- important since the ingestor writes ingestion_audit_log via raw SQL,
-- not through registry/sqlc.
--
-- pg_advisory_xact_lock serializes concurrent inserts on the same table
-- so the chain stays strictly linear even from multiple ingestor
-- instances (CLAUDE.md: "verify this holds under concurrent ingestor
-- instances"). Audit-log write volume is far below telemetry's, so this
-- serialization point is not a throughput concern.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE ingestion_audit_log ADD COLUMN prev_hash text;
ALTER TABLE ingestion_audit_log ADD COLUMN entry_hash text;

ALTER TABLE user_action_audit_log ADD COLUMN prev_hash text;
ALTER TABLE user_action_audit_log ADD COLUMN entry_hash text;

-- Backfill existing rows with a real chain BEFORE the immutability guard
-- triggers exist below — those triggers correctly forbid changing
-- prev_hash/entry_hash on UPDATE once the chain is live, which would
-- otherwise block this one-time backfill from ever setting them.
-- +goose StatementBegin
DO $$
DECLARE
    r record;
    prev text := '';
BEGIN
    FOR r IN SELECT id, device_id, raw_payload, received_at FROM ingestion_audit_log ORDER BY id ASC
    LOOP
        UPDATE ingestion_audit_log SET
            prev_hash = prev,
            entry_hash = encode(digest(prev || '|' || r.device_id || '|' || r.raw_payload::text || '|' || r.received_at::text, 'sha256'), 'hex')
        WHERE id = r.id;
        SELECT entry_hash INTO prev FROM ingestion_audit_log WHERE id = r.id;
    END LOOP;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
    r record;
    prev text := '';
BEGIN
    FOR r IN SELECT id, actor_user_id, action, target_type, target_id, metadata, created_at FROM user_action_audit_log ORDER BY id ASC
    LOOP
        UPDATE user_action_audit_log SET
            prev_hash = prev,
            entry_hash = encode(
                digest(
                    prev || '|' || coalesce(r.actor_user_id::text, '') || '|' || r.action || '|' ||
                    coalesce(r.target_type, '') || '|' || coalesce(r.target_id, '') || '|' ||
                    coalesce(r.metadata::text, '') || '|' || r.created_at::text,
                    'sha256'
                ),
                'hex'
            )
        WHERE id = r.id;
        SELECT entry_hash INTO prev FROM user_action_audit_log WHERE id = r.id;
    END LOOP;
END;
$$;
-- +goose StatementEnd

-- ingestion_audit_log's hash covers only the immutable "what/when did we
-- receive" fields (device_id, raw_payload, received_at) — processed/
-- error are set by the ingestor in a follow-up UPDATE once validation
-- finishes, so including them would make every legitimate status update
-- look like tampering. What's protected is exactly the fact that must
-- never be silently rewritten: what arrived, and when.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION ingestion_audit_log_chain() RETURNS trigger AS $$
DECLARE
    prev text;
BEGIN
    PERFORM pg_advisory_xact_lock(hashtext('ingestion_audit_log_chain'));
    SELECT entry_hash INTO prev FROM ingestion_audit_log ORDER BY id DESC LIMIT 1;
    NEW.prev_hash := coalesce(prev, '');
    NEW.entry_hash := encode(
        digest(coalesce(prev, '') || '|' || NEW.device_id || '|' || NEW.raw_payload::text || '|' || NEW.received_at::text, 'sha256'),
        'hex'
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_ingestion_audit_log_chain
    BEFORE INSERT ON ingestion_audit_log
    FOR EACH ROW EXECUTE FUNCTION ingestion_audit_log_chain();

-- user_action_audit_log rows have no update path anywhere in the
-- application, so the hash covers the full row.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION user_action_audit_log_chain() RETURNS trigger AS $$
DECLARE
    prev text;
BEGIN
    PERFORM pg_advisory_xact_lock(hashtext('user_action_audit_log_chain'));
    SELECT entry_hash INTO prev FROM user_action_audit_log ORDER BY id DESC LIMIT 1;
    NEW.prev_hash := coalesce(prev, '');
    NEW.entry_hash := encode(
        digest(
            coalesce(prev, '') || '|' || coalesce(NEW.actor_user_id::text, '') || '|' || NEW.action || '|' ||
            coalesce(NEW.target_type, '') || '|' || coalesce(NEW.target_id, '') || '|' ||
            coalesce(NEW.metadata::text, '') || '|' || NEW.created_at::text,
            'sha256'
        ),
        'hex'
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_user_action_audit_log_chain
    BEFORE INSERT ON user_action_audit_log
    FOR EACH ROW EXECUTE FUNCTION user_action_audit_log_chain();

-- Belt-and-suspenders DB-level guards, alongside the app layer never
-- exposing update/delete for either table: user_action_audit_log rejects
-- any UPDATE/DELETE outright; ingestion_audit_log allows the ingestor's
-- legitimate processed/error UPDATE but rejects any change to the
-- chained fields, and rejects DELETE outright.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION reject_audit_log_mutation() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'audit log rows are immutable';
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_user_action_audit_log_no_update
    BEFORE UPDATE ON user_action_audit_log
    FOR EACH ROW EXECUTE FUNCTION reject_audit_log_mutation();
CREATE TRIGGER trg_user_action_audit_log_no_delete
    BEFORE DELETE ON user_action_audit_log
    FOR EACH ROW EXECUTE FUNCTION reject_audit_log_mutation();
CREATE TRIGGER trg_ingestion_audit_log_no_delete
    BEFORE DELETE ON ingestion_audit_log
    FOR EACH ROW EXECUTE FUNCTION reject_audit_log_mutation();

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION reject_ingestion_audit_log_chain_mutation() RETURNS trigger AS $$
BEGIN
    IF NEW.device_id IS DISTINCT FROM OLD.device_id
        OR NEW.raw_payload IS DISTINCT FROM OLD.raw_payload
        OR NEW.received_at IS DISTINCT FROM OLD.received_at
        OR NEW.prev_hash IS DISTINCT FROM OLD.prev_hash
        OR NEW.entry_hash IS DISTINCT FROM OLD.entry_hash
    THEN
        RAISE EXCEPTION 'ingestion audit log chained fields are immutable';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_ingestion_audit_log_guard_update
    BEFORE UPDATE ON ingestion_audit_log
    FOR EACH ROW EXECUTE FUNCTION reject_ingestion_audit_log_chain_mutation();

-- +goose Down
DROP TRIGGER IF EXISTS trg_ingestion_audit_log_guard_update ON ingestion_audit_log;
DROP TRIGGER IF EXISTS trg_ingestion_audit_log_no_delete ON ingestion_audit_log;
DROP TRIGGER IF EXISTS trg_user_action_audit_log_no_delete ON user_action_audit_log;
DROP TRIGGER IF EXISTS trg_user_action_audit_log_no_update ON user_action_audit_log;
DROP TRIGGER IF EXISTS trg_user_action_audit_log_chain ON user_action_audit_log;
DROP TRIGGER IF EXISTS trg_ingestion_audit_log_chain ON ingestion_audit_log;
DROP FUNCTION IF EXISTS reject_ingestion_audit_log_chain_mutation();
DROP FUNCTION IF EXISTS reject_audit_log_mutation();
DROP FUNCTION IF EXISTS user_action_audit_log_chain();
DROP FUNCTION IF EXISTS ingestion_audit_log_chain();
ALTER TABLE user_action_audit_log DROP COLUMN IF EXISTS entry_hash;
ALTER TABLE user_action_audit_log DROP COLUMN IF EXISTS prev_hash;
ALTER TABLE ingestion_audit_log DROP COLUMN IF EXISTS entry_hash;
ALTER TABLE ingestion_audit_log DROP COLUMN IF EXISTS prev_hash;
