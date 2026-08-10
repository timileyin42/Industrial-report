package registry

import (
	"context"
	"testing"
)

// These tests never delete rows from ingestion_audit_log or
// user_action_audit_log, unlike every other test in this package. That's
// deliberate, not an oversight: both tables are hash-chained
// (migrations/0013_audit_log_tamper_evidence.sql), where each row's stored
// hash depends on the row immediately before it by id. Deleting a row —
// even via the disable-trigger bypass used elsewhere for genuine
// cleanup — permanently orphans its neighbor's prev_hash, since nothing
// recomputes the downstream chain. That's not a bug; it's the same
// property that makes the chain tamper-evident in the first place. It
// does mean these two tables accumulate rows forever, same as real
// production audit rows would — consistent with their own immutability,
// not an exception to the "never leave test rows behind" rule elsewhere.
//
// Because this is a shared dev database that other tests/manual runs also
// write to, these tests never assert "the whole chain is valid" — a
// mismatch anywhere earlier in the table's history (this session's own,
// or someone else's) would make that assertion permanently, unrelatedly
// false. Instead they snapshot VerifyChain's result before their own
// inserts and compare the delta, which holds regardless of what else has
// ever touched the table.

// TestIngestionAuditVerifyChain_CleanChainIsValid confirms VerifyChain
// doesn't manufacture new mismatches for rows that were never tampered
// with — the control case the tampering test below depends on to mean
// anything.
func TestIngestionAuditVerifyChain_CleanChainIsValid(t *testing.T) {
	q := testQueries(t)
	pool := testRawPool(t)
	ctx := context.Background()
	deviceID := uniqueID("dev-chain-clean-")

	audit := NewIngestionAudit(q)
	before, err := audit.VerifyChain(ctx)
	if err != nil {
		t.Fatalf("VerifyChain before insert: %v", err)
	}

	for i := 0; i < 3; i++ {
		if _, err := pool.Exec(ctx,
			`INSERT INTO ingestion_audit_log (device_id, raw_payload) VALUES ($1, '{"x":1}'::jsonb)`,
			deviceID,
		); err != nil {
			t.Fatalf("insert audit row %d: %v", i, err)
		}
	}

	after, err := audit.VerifyChain(ctx)
	if err != nil {
		t.Fatalf("VerifyChain after insert: %v", err)
	}
	if after.MismatchCount != before.MismatchCount {
		t.Fatalf("expected 3 cleanly-chained inserts to add zero new mismatches, count went from %d to %d",
			before.MismatchCount, after.MismatchCount)
	}
}

// TestIngestionAuditVerifyChain_DetectsTampering is the actual regression
// test for the tamper-evidence feature: even a superuser-level bypass —
// disabling the guard trigger, editing a row's content, re-enabling it —
// must be detectable by VerifyChain afterward. This is the same attack
// this feature was manually verified against once during development;
// this test makes that verification permanent instead of a one-off
// manual check.
func TestIngestionAuditVerifyChain_DetectsTampering(t *testing.T) {
	q := testQueries(t)
	pool := testRawPool(t)
	ctx := context.Background()
	deviceID := uniqueID("dev-chain-tamper-")

	audit := NewIngestionAudit(q)
	before, err := audit.VerifyChain(ctx)
	if err != nil {
		t.Fatalf("VerifyChain before tampering: %v", err)
	}

	var targetID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO ingestion_audit_log (device_id, raw_payload) VALUES ($1, '{"x":1}'::jsonb) RETURNING id`,
		deviceID,
	).Scan(&targetID); err != nil {
		t.Fatalf("insert first audit row: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO ingestion_audit_log (device_id, raw_payload) VALUES ($1, '{"x":2}'::jsonb)`,
		deviceID,
	); err != nil {
		t.Fatalf("insert second audit row: %v", err)
	}

	// Simulate a bypass: disable the guard trigger, rewrite the row's
	// content, re-enable it — mirroring exactly how a real attacker with
	// superuser access (or an operator error) would have to go around the
	// normal UPDATE-guard trigger, since a normal UPDATE is already
	// rejected outright.
	if _, err := pool.Exec(ctx, `ALTER TABLE ingestion_audit_log DISABLE TRIGGER trg_ingestion_audit_log_guard_update`); err != nil {
		t.Fatalf("disable update guard: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE ingestion_audit_log SET raw_payload = '{"x":999}'::jsonb WHERE id = $1`, targetID); err != nil {
		t.Fatalf("simulate tampering: %v", err)
	}
	if _, err := pool.Exec(ctx, `ALTER TABLE ingestion_audit_log ENABLE TRIGGER trg_ingestion_audit_log_guard_update`); err != nil {
		t.Fatalf("re-enable update guard: %v", err)
	}

	after, err := audit.VerifyChain(ctx)
	if err != nil {
		t.Fatalf("VerifyChain after tampering: %v", err)
	}
	if after.MismatchCount <= before.MismatchCount {
		t.Fatalf("expected tampering to strictly increase the mismatch count, went from %d to %d",
			before.MismatchCount, after.MismatchCount)
	}
	if after.Valid {
		t.Fatal("expected VerifyChain to report the chain as invalid after tampering")
	}
}

// TestAuditLogVerifyChain_DetectsTampering is the same regression, for the
// admin-action trail (user_action_audit_log) — a separate table with its
// own trigger, deliberately not sharing state with ingestion_audit_log per
// CLAUDE.md ("don't conflate the two").
func TestAuditLogVerifyChain_DetectsTampering(t *testing.T) {
	q := testQueries(t)
	pool := testRawPool(t)
	ctx := context.Background()
	targetID := uniqueID("target-chain-")

	auditLog := NewAuditLog(q)
	before, err := auditLog.VerifyChain(ctx)
	if err != nil {
		t.Fatalf("VerifyChain before tampering: %v", err)
	}

	recordAction(ctx, q, 1, "test.action", "test", targetID, map[string]any{"n": 1})

	var rowID int64
	if err := pool.QueryRow(ctx,
		`SELECT id FROM user_action_audit_log WHERE target_id = $1 ORDER BY id DESC LIMIT 1`, targetID,
	).Scan(&rowID); err != nil {
		t.Fatalf("find inserted audit row: %v", err)
	}

	if _, err := pool.Exec(ctx, `ALTER TABLE user_action_audit_log DISABLE TRIGGER trg_user_action_audit_log_no_update`); err != nil {
		t.Fatalf("disable update guard: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE user_action_audit_log SET action = 'tampered.action' WHERE id = $1`, rowID); err != nil {
		t.Fatalf("simulate tampering: %v", err)
	}
	if _, err := pool.Exec(ctx, `ALTER TABLE user_action_audit_log ENABLE TRIGGER trg_user_action_audit_log_no_update`); err != nil {
		t.Fatalf("re-enable update guard: %v", err)
	}

	after, err := auditLog.VerifyChain(ctx)
	if err != nil {
		t.Fatalf("VerifyChain after tampering: %v", err)
	}
	if after.MismatchCount <= before.MismatchCount {
		t.Fatalf("expected tampering to strictly increase the mismatch count, went from %d to %d",
			before.MismatchCount, after.MismatchCount)
	}
	if after.Valid {
		t.Fatal("expected VerifyChain to report the chain as invalid after tampering")
	}
}
