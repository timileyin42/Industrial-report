// Command ingestor subscribes to devices/+/telemetry on the MQTT broker and
// persists readings into TimescaleDB.
//
// Order of operations per message, deliberately in this order (AGENTS.md item 4):
//  1. Write the raw payload to ingestion_audit_log FIRST, unconditionally.
//     A message that later fails validation is still recorded — never
//     silently dropped.
//  2. Update devices.last_contact_at unconditionally — this is the "have we
//     heard from this device at all" signal (Phase 2), independent of
//     whether the payload turns out to be valid.
//  3. Look up device + its site (for site-specific validation bounds).
//  4. Validate.
//  5. Classify provenance / detect an energy-counter reset / run the
//     day-night quality heuristic (internal/domain — pure functions).
//  6. Upsert into telemetry with ON CONFLICT DO NOTHING on (device_id, ts)
//     — dedup for retried/duplicate MQTT deliveries.
//  7. Advance devices.last_seen_at, but only forward — never let an
//     out-of-order/backfilled message walk it backward.
//  8. Mark the audit log row processed (or record the validation error).
//
// Deliberately still raw pgx here, not sqlc/internal/registry — AGENTS.md's
// stack split keeps the ingestor on the hot path with no ORM overhead.
// Judgment calls (ceiling math, reset detection, provenance, day/night) live
// in internal/domain as pure functions; this file only does I/O + wiring.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/timileyin42/zgnis-solar/internal/domain"
)

func main() {
	dbURL := mustEnv("DATABASE_URL")
	brokerURL := mustEnv("MQTT_BROKER_URL") // e.g. tcp://localhost:1883 or ssl://localhost:8883
	mqttUser := mustEnv("MQTT_USERNAME")    // "ingestor-service"
	mqttPass := mustEnv("MQTT_PASSWORD")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	opts := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID("zgnis-ingestor").
		SetUsername(mqttUser).
		SetPassword(mqttPass).
		SetAutoReconnect(true)

	// Phase 4: optional TLS. Only takes effect when MQTT_TLS_CA_CERT is set
	// (paired with an ssl:// broker URL) — unset behavior is unchanged from
	// Phase 0-3, so this is additive, not a breaking default.
	if caCertPath := os.Getenv("MQTT_TLS_CA_CERT"); caCertPath != "" {
		tlsConfig, err := buildTLSConfig(caCertPath)
		if err != nil {
			log.Fatalf("mqtt tls config: %v", err)
		}
		opts.SetTLSConfig(tlsConfig)
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("mqtt connect: %v", token.Error())
	}
	defer client.Disconnect(250)

	handler := func(_ mqtt.Client, msg mqtt.Message) {
		if err := handleMessage(ctx, pool, msg.Topic(), msg.Payload()); err != nil {
			log.Printf("handle message from %s: %v", msg.Topic(), err)
		}
	}

	// devices/{device_id}/telemetry — one topic per device (AGENTS.md item 1)
	if token := client.Subscribe("devices/+/telemetry", 1, handler); token.Wait() && token.Error() != nil {
		log.Fatalf("subscribe: %v", token.Error())
	}

	log.Println("ingestor running, subscribed to devices/+/telemetry")
	<-ctx.Done()
	log.Println("shutting down")
}

func handleMessage(ctx context.Context, pool *pgxpool.Pool, topic string, raw []byte) error {
	now := time.Now().UTC()
	deviceID := deviceIDFromTopic(topic)

	// Step 1: audit log first, unconditionally, before any validation.
	var auditID int64
	err := pool.QueryRow(ctx,
		`INSERT INTO ingestion_audit_log (device_id, raw_payload) VALUES ($1, $2) RETURNING id`,
		deviceID, raw,
	).Scan(&auditID)
	if err != nil {
		// If we can't even write the audit row, log loudly and stop —
		// do not proceed to "process" a message we can't account for.
		return err
	}

	// Step 2: unconditional reachability signal — the device contacted the
	// broker, regardless of whether this payload turns out to be valid.
	if _, err := pool.Exec(ctx, `UPDATE devices SET last_contact_at = $1 WHERE device_id = $2`, now, deviceID); err != nil {
		log.Printf("update last_contact_at for %s: %v", deviceID, err)
	}

	var payload domain.TelemetryPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		markAuditError(ctx, pool, auditID, "unmarshal: "+err.Error())
		return err
	}

	// Step 3: look up device joined with its site — need site_id, revoked_at
	// for the existing checks, plus system_size_kw and timezone for Phase 2's
	// site-specific ceiling and day/night heuristic.
	var siteID string
	var revokedAt *time.Time
	var systemSizeKW *float64
	var siteTimezone string
	err = pool.QueryRow(ctx,
		`SELECT d.site_id, d.revoked_at, s.system_size_kw, s.timezone
		 FROM devices d JOIN sites s ON s.site_id = d.site_id
		 WHERE d.device_id = $1`,
		deviceID,
	).Scan(&siteID, &revokedAt, &systemSizeKW, &siteTimezone)
	if err != nil {
		markAuditError(ctx, pool, auditID, "unknown device: "+err.Error())
		return err
	}
	if revokedAt != nil {
		markAuditError(ctx, pool, auditID, "device revoked")
		return nil
	}

	// Step 4: validate against this site's plausibility ceiling.
	ceiling := domain.PowerCeilingKW(systemSizeKW)
	ts, err := payload.Validate(ceiling)
	if err != nil {
		markAuditError(ctx, pool, auditID, "validation: "+err.Error())
		return err
	}

	// Step 5: provenance + quality flags — pure judgment calls, delegated to
	// internal/domain.
	provenance := domain.ClassifyProvenance(ts, now)

	// pgx sends a nil slice as SQL NULL, not as an empty array — must start
	// non-nil since the column is NOT NULL DEFAULT '{}' and this INSERT
	// always supplies an explicit value rather than relying on the default.
	qualityFlags := []string{}
	previousEnergy, err := previousEnergyByTS(ctx, pool, deviceID, ts)
	if err != nil {
		log.Printf("lookup previous energy for %s: %v", deviceID, err)
	} else if domain.DetectEnergyReset(previousEnergy, payload.EnergyKWhTotal) {
		qualityFlags = append(qualityFlags, domain.QualityFlagEnergyReset)
	}

	if loc, err := time.LoadLocation(siteTimezone); err != nil {
		log.Printf("load timezone %q for site %s: %v", siteTimezone, siteID, err)
	} else if domain.IsCoarseNight(ts.In(loc)) && payload.PowerKW > 0 {
		qualityFlags = append(qualityFlags, domain.QualityFlagNightNonzeroOutput)
	}

	// Step 6: upsert with dedup on (device_id, ts).
	_, err = pool.Exec(ctx,
		`INSERT INTO telemetry (device_id, site_id, ts, power_kw, energy_kwh_total, voltage_v, status, provenance, quality_flags)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (device_id, ts) DO NOTHING`,
		deviceID, siteID, ts, payload.PowerKW, payload.EnergyKWhTotal, payload.VoltageV, coalesceStatus(payload.Status),
		string(provenance), qualityFlags,
	)
	if err != nil {
		markAuditError(ctx, pool, auditID, "insert: "+err.Error())
		return err
	}

	// Step 7: advance last_seen_at, but only forward — an out-of-order or
	// backfilled message must never walk this backward (Phase 2 bug fix).
	_, err = pool.Exec(ctx,
		`UPDATE devices SET last_seen_at = $1 WHERE device_id = $2 AND (last_seen_at IS NULL OR last_seen_at < $1)`,
		ts, deviceID,
	)
	if err != nil {
		log.Printf("update last_seen for %s: %v", deviceID, err)
	}

	_, err = pool.Exec(ctx, `UPDATE ingestion_audit_log SET processed = true WHERE id = $1`, auditID)
	return err
}

// previousEnergyByTS returns the energy_kwh_total of the reading
// immediately preceding ts for this device, chronologically — never the
// most-recently-inserted row. This is what makes reset detection correct
// under out-of-order/backfilled arrival: a legitimately older reading is
// compared against an even-older baseline, never against a later one that
// happened to be inserted first.
func previousEnergyByTS(ctx context.Context, pool *pgxpool.Pool, deviceID string, ts time.Time) (*float64, error) {
	var energy float64
	err := pool.QueryRow(ctx,
		`SELECT energy_kwh_total FROM telemetry WHERE device_id = $1 AND ts < $2 ORDER BY ts DESC LIMIT 1`,
		deviceID, ts,
	).Scan(&energy)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &energy, nil
}

func markAuditError(ctx context.Context, pool *pgxpool.Pool, auditID int64, errMsg string) {
	if _, err := pool.Exec(ctx, `UPDATE ingestion_audit_log SET error = $1 WHERE id = $2`, errMsg, auditID); err != nil {
		log.Printf("failed to record audit error: %v", err)
	}
}

func deviceIDFromTopic(topic string) string {
	// devices/{device_id}/telemetry
	parts := strings.Split(topic, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

func coalesceStatus(s string) string {
	if s == "" {
		return domain.StatusOK
	}
	return s
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s not set", key)
	}
	return v
}

// buildTLSConfig loads a CA cert so the ingestor can verify the broker's
// certificate over ssl://. See docs/tls.md — the cert this points at
// locally is dev-only (scripts/gen-dev-certs.sh); a real deployment needs
// a real CA-issued cert for a real hostname.
func buildTLSConfig(caCertPath string) (*tls.Config, error) {
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA cert at %s", caCertPath)
	}
	return &tls.Config{RootCAs: pool}, nil
}
