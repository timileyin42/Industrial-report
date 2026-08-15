// Command api serves the dashboard/admin API backed by TimescaleDB.
// Per concept note section 8: the dashboard only ever reads from the
// platform's own database — never from a device or manufacturer cloud.
//
// This file is wiring only — handlers live in internal/httpapi, business
// logic in internal/registry, queries in internal/db (sqlc-generated).
package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/timileyin42/zgnis-solar/internal/auth"
	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/email"
	"github.com/timileyin42/zgnis-solar/internal/httpapi"
	"github.com/timileyin42/zgnis-solar/internal/mqttadmin"
	"github.com/timileyin42/zgnis-solar/internal/registry"
	"github.com/timileyin42/zgnis-solar/internal/storage"
)

func main() {
	dbURL := withSSLMode(mustEnv("DATABASE_URL"))
	jwtSecret := mustEnv("JWT_SECRET")

	onlineThreshold := envMinutes("ONLINE_THRESHOLD_MINUTES", 10)
	expectedInterval := envMinutes("EXPECTED_READING_INTERVAL_MINUTES", 5)
	coverageWindow := envHours("COVERAGE_WINDOW_HOURS", 24)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)

	// mqttAdmin is nil (not fatal) when MQTT_ADMIN_USERNAME/PASSWORD
	// aren't set — see internal/mqttadmin's own doc comment. Its
	// bootstrap step also provisions the ingestor's own broker
	// credential (from MQTT_USERNAME/MQTT_PASSWORD) if missing, so a
	// fresh deployment needs zero manual `mosquitto_passwd`/ACL setup.
	mqttAdmin, err := mqttadmin.NewClientFromEnv(ctx)
	if err != nil {
		log.Fatalf("mqtt admin (dynsec) connect: %v", err)
	}

	sites := registry.NewSites(queries)
	devices := registry.NewDevices(queries, onlineThreshold, expectedInterval, mqttAdmin)
	users := registry.NewUsers(queries)
	fleet := registry.NewFleet(sites, devices, onlineThreshold, expectedInterval, coverageWindow)
	telemetry := registry.NewTelemetry(queries)
	analytics := registry.NewAnalytics(queries)
	emissions := registry.NewEmissions(analytics, sites, queries, os.Getenv("GRID_COUNTRY"))
	benchmark := registry.NewBenchmark(analytics)
	anomaly := registry.NewAnomaly(analytics)
	auditLog := registry.NewAuditLog(queries)
	ingestionAudit := registry.NewIngestionAudit(queries)
	alerts := registry.NewAlerts(fleet, anomaly, queries)

	// APP_BASE_URL builds invite/reset links (e.g. https://app.cleanenergyanalytics.co.uk)
	// — defaults to the Vite dev server origin so local dev works without
	// setting it. RESEND_API_KEY/EMAIL_FROM_ADDRESS are optional: unset
	// falls back to a logging no-op sender (see internal/email).
	appBaseURL := envOr("APP_BASE_URL", "http://localhost:5173")
	sender := email.NewSenderFromEnv()
	invites := registry.NewInvites(queries, sender, appBaseURL)
	passwordReset := registry.NewPasswordReset(queries, sender, appBaseURL)

	// R2 storage is optional infra — NewFromEnv returns a nil client (not
	// an error) when unconfigured, so the API still starts; export jobs
	// just fail fast with a clear message until R2_* env vars are set.
	storageClient, err := storage.NewFromEnv(ctx)
	if err != nil {
		log.Fatalf("r2 storage init: %v", err)
	}
	exports := registry.NewExports(queries, telemetry, analytics, storageClient)
	sandbox := registry.NewSandbox(queries)
	// COMPANY_CONTACT_EMAIL is separate from the seeded operator's login
	// email — the demo-request form notifies both, per CLAUDE.md's
	// principle of not conflating distinct concerns (here: "who can log
	// in" vs. "who handles sales inquiries"). Unset just skips that half
	// of the notification (see DemoRequests.Submit), same no-op-if-unset
	// pattern as RESEND_API_KEY.
	demoRequests := registry.NewDemoRequests(queries, sender, os.Getenv("COMPANY_CONTACT_EMAIL"))
	cloudImport := registry.NewCloudImport(queries)

	e := httpapi.NewRouter(httpapi.Deps{
		Sites:          sites,
		Devices:        devices,
		Users:          users,
		Fleet:          fleet,
		Telemetry:      telemetry,
		Analytics:      analytics,
		Emissions:      emissions,
		Benchmark:      benchmark,
		Anomaly:        anomaly,
		AuditLog:       auditLog,
		IngestionAudit: ingestionAudit,
		Invites:        invites,
		PasswordReset:  passwordReset,
		Exports:        exports,
		Alerts:         alerts,
		Sandbox:        sandbox,
		DemoRequests:   demoRequests,
		CloudImport:    cloudImport,
		Issuer:         auth.NewTokenIssuer(jwtSecret),
	})

	// Phase 4: optional TLS listener. Only takes effect when both
	// API_TLS_CERT_FILE and API_TLS_KEY_FILE are set — unset behavior is
	// unchanged from Phase 0-3 (plain e.Start), so this is additive.
	certFile := os.Getenv("API_TLS_CERT_FILE")
	keyFile := os.Getenv("API_TLS_KEY_FILE")
	if certFile != "" && keyFile != "" {
		log.Println("api listening on :8080 (TLS)")
		e.Logger.Fatal(e.StartTLS(":8080", certFile, keyFile))
		return
	}

	log.Println("api listening on :8080")
	e.Logger.Fatal(e.Start(":8080"))
}

// withSSLMode appends sslmode to a Postgres connection string when it
// doesn't already specify one, using DATABASE_SSLMODE (default "disable"
// — matches Phase 0-3's local-dev default; set to "require"/"verify-full"
// once a real deployment has real database TLS in place. See docs/tls.md.
func withSSLMode(dbURL string) string {
	if strings.Contains(dbURL, "sslmode=") {
		return dbURL
	}
	mode := os.Getenv("DATABASE_SSLMODE")
	if mode == "" {
		mode = "disable"
	}
	sep := "?"
	if strings.Contains(dbURL, "?") {
		sep = "&"
	}
	return dbURL + sep + "sslmode=" + mode
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s not set", key)
	}
	return v
}

func envMinutes(key string, def int) time.Duration {
	return time.Duration(envInt(key, def)) * time.Minute
}

func envHours(key string, def int) time.Duration {
	return time.Duration(envInt(key, def)) * time.Hour
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Fatalf("invalid %s %q: %v", key, v, err)
	}
	return n
}
