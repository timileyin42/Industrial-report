package main

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/timileyin42/zgnis-solar/internal/domain"
)

// testPool connects to DATABASE_URL, same convention as
// internal/registry's test helpers — skips cleanly when unset so
// `go test ./...` stays safe without a database available.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set — skipping integration test that needs a real database")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func uniqueID(prefix string) string {
	return prefix + strconv.FormatInt(time.Now().UnixNano(), 36)
}

// seedTestDevice creates a throwaway site + device (optionally revoked)
// for handleMessage to look up, mirroring exactly what the real
// registration/revocation flows leave behind.
func seedTestDevice(t *testing.T, ctx context.Context, pool *pgxpool.Pool, revoked bool) (siteID, deviceID string) {
	t.Helper()
	siteID = uniqueID("site-ingest-")
	if _, err := pool.Exec(ctx,
		`INSERT INTO sites (site_id, name, timezone, country, system_size_kw) VALUES ($1, 'Ingestor Test Site', 'UTC', 'NG', 5)`,
		siteID,
	); err != nil {
		t.Fatalf("seed site: %v", err)
	}
	deviceID = uniqueID("dev-ingest-")
	var revokedAt any
	if revoked {
		revokedAt = time.Now().UTC()
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO devices (device_id, site_id, secret_hash, revoked_at) VALUES ($1, $2, 'test-hash', $3)`,
		deviceID, siteID, revokedAt,
	); err != nil {
		t.Fatalf("seed device: %v", err)
	}
	// This runs against the same shared local dev database a person may
	// be actively looking at in the dashboard — never leave test sites/
	// devices/telemetry/audit rows behind for their Sites/Devices lists
	// to accumulate.
	t.Cleanup(func() {
		cctx := context.Background()
		_, _ = pool.Exec(cctx, `DELETE FROM ingestion_audit_log WHERE device_id = $1`, deviceID)
		_, _ = pool.Exec(cctx, `DELETE FROM telemetry WHERE device_id = $1`, deviceID)
		_, _ = pool.Exec(cctx, `DELETE FROM devices WHERE device_id = $1`, deviceID)
		_, _ = pool.Exec(cctx, `DELETE FROM sites WHERE site_id = $1`, siteID)
	})
	return siteID, deviceID
}

func telemetryRowCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, deviceID string) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM telemetry WHERE device_id = $1`, deviceID).Scan(&count); err != nil {
		t.Fatalf("count telemetry rows: %v", err)
	}
	return count
}

func lastAuditError(t *testing.T, ctx context.Context, pool *pgxpool.Pool, deviceID string) (errText string, processed bool) {
	t.Helper()
	var err *string
	if scanErr := pool.QueryRow(ctx,
		`SELECT error, processed FROM ingestion_audit_log WHERE device_id = $1 ORDER BY id DESC LIMIT 1`,
		deviceID,
	).Scan(&err, &processed); scanErr != nil {
		t.Fatalf("query audit log: %v", scanErr)
	}
	if err != nil {
		errText = *err
	}
	return errText, processed
}

func validPayload(deviceID string) []byte {
	body, _ := json.Marshal(domain.TelemetryPayload{
		DeviceID:       deviceID,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		PowerKW:        2.5,
		EnergyKWhTotal: 100,
		Status:         domain.StatusOK,
	})
	return body
}

// TestHandleMessageRejectsUnknownDevice confirms a reading for a
// device_id that was never registered is rejected AND recorded in the
// ingestion audit log with a clear reason — never silently dropped, and
// never accepted just because it parsed as valid JSON.
func TestHandleMessageRejectsUnknownDevice(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	deviceID := uniqueID("dev-unknown-")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM ingestion_audit_log WHERE device_id = $1`, deviceID)
	})

	err := handleMessage(ctx, pool, "devices/"+deviceID+"/telemetry", validPayload(deviceID))
	if err == nil {
		t.Fatal("expected an error for an unregistered device_id, got nil")
	}
	if telemetryRowCount(t, ctx, pool, deviceID) != 0 {
		t.Fatal("a reading from an unregistered device must never be stored")
	}
	errText, _ := lastAuditError(t, ctx, pool, deviceID)
	if errText == "" {
		t.Fatal("expected the ingestion audit log to record why this was rejected, got no error text")
	}
}

// TestHandleMessageRejectsRevokedDevice is the core regression test for
// issue #1's revocation requirement: a revoked device's data must be
// rejected at the ingestion path itself (not just hidden in the UI), and
// still recorded in the audit log for forensic value — never silently
// dropped, never stored as live telemetry either.
func TestHandleMessageRejectsRevokedDevice(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	_, deviceID := seedTestDevice(t, ctx, pool, true)

	// handleMessage deliberately returns nil for a revoked device (see
	// cmd/ingestor/main.go step 3) — a policy rejection, not a transient
	// failure worth an MQTT redelivery retry storm. The real assertions
	// are what actually happened in the database, not the return value.
	if err := handleMessage(ctx, pool, "devices/"+deviceID+"/telemetry", validPayload(deviceID)); err != nil {
		t.Fatalf("handleMessage should return nil (not retry) for a revoked device, got %v", err)
	}
	if telemetryRowCount(t, ctx, pool, deviceID) != 0 {
		t.Fatal("a revoked device's reading must never be stored, even though handleMessage returns nil")
	}
	errText, _ := lastAuditError(t, ctx, pool, deviceID)
	if errText != "device revoked" {
		t.Fatalf(`expected audit log error "device revoked", got %q`, errText)
	}
}

// TestHandleMessageAcceptsValidReading is the control case — confirms
// the rejection tests above are actually testing rejection, not a
// broken table/query that would make everything fail regardless.
func TestHandleMessageAcceptsValidReading(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	_, deviceID := seedTestDevice(t, ctx, pool, false)

	if err := handleMessage(ctx, pool, "devices/"+deviceID+"/telemetry", validPayload(deviceID)); err != nil {
		t.Fatalf("expected a valid reading from a registered, non-revoked device to succeed, got %v", err)
	}
	if telemetryRowCount(t, ctx, pool, deviceID) != 1 {
		t.Fatal("expected exactly one telemetry row for a valid reading")
	}
	errText, processed := lastAuditError(t, ctx, pool, deviceID)
	if errText != "" {
		t.Fatalf("expected no audit error for a valid reading, got %q", errText)
	}
	if !processed {
		t.Fatal("expected the audit log row to be marked processed for a successfully ingested reading")
	}
}

// TestHandleMessageDeduplicatesConcurrentDelivery is CLAUDE.md's explicit
// concurrency requirement: "Duplicate MQTT delivery of the same reading
// (QoS 1 can redeliver) — handled by the (device_id, ts) unique
// constraint; verify this holds under concurrent ingestor instances, not
// just single-process testing." Every goroutine here uses its own pool
// connection (mirroring separate ingestor processes hitting the same DB
// concurrently) and the exact same payload — the ON CONFLICT DO NOTHING
// upsert in cmd/ingestor/main.go must leave exactly one row regardless of
// how many redeliveries race each other.
func TestHandleMessageDeduplicatesConcurrentDelivery(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	_, deviceID := seedTestDevice(t, ctx, pool, false)
	body := validPayload(deviceID)
	topic := "devices/" + deviceID + "/telemetry"

	const concurrency = 20
	var wg sync.WaitGroup
	errs := make([]error, concurrency)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = handleMessage(ctx, pool, topic, body)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: handleMessage returned an error for a redelivered duplicate: %v", i, err)
		}
	}
	if got := telemetryRowCount(t, ctx, pool, deviceID); got != 1 {
		t.Fatalf("expected exactly 1 telemetry row after %d concurrent identical deliveries, got %d", concurrency, got)
	}
}

// TestHandleMessageRejectsImplausiblePower confirms a reading far above
// a site's plausibility ceiling is rejected (never stored) but still
// audited with the validation failure reason — the "rejected vs
// flagged" distinction this platform's data-quality story depends on.
func TestHandleMessageRejectsImplausiblePower(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	_, deviceID := seedTestDevice(t, ctx, pool, false) // site has system_size_kw=5

	body, _ := json.Marshal(domain.TelemetryPayload{
		DeviceID:       deviceID,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		PowerKW:        500, // wildly over a 5kW site's plausibility ceiling
		EnergyKWhTotal: 100,
		Status:         domain.StatusOK,
	})

	if err := handleMessage(ctx, pool, "devices/"+deviceID+"/telemetry", body); err == nil {
		t.Fatal("expected an error for a reading far above the site's plausibility ceiling")
	}
	if telemetryRowCount(t, ctx, pool, deviceID) != 0 {
		t.Fatal("an implausible reading must never be stored")
	}
	errText, _ := lastAuditError(t, ctx, pool, deviceID)
	if errText == "" {
		t.Fatal("expected the audit log to record the validation failure reason")
	}
}
